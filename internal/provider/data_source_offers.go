package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hostaria/terraform-provider-hostaria/internal/client"
)

var (
	_ datasource.DataSource              = &offersDataSource{}
	_ datasource.DataSourceWithConfigure = &offersDataSource{}
)

func NewOffersDataSource() datasource.DataSource { return &offersDataSource{} }

type offersDataSource struct {
	client *client.Client
}

type offersDataSourceModel struct {
	Offers []offerItemModel `tfsdk:"offers"`
}

type offerItemModel struct {
	ID              types.Int64   `tfsdk:"id"`
	Name            types.String  `tfsdk:"name"`
	Price           types.Float64 `tfsdk:"price"`
	Ram             types.Int64   `tfsdk:"ram"`
	CpuCores        types.Int64   `tfsdk:"cpu_cores"`
	Storage         types.Int64   `tfsdk:"storage"`
	BillingInterval types.String  `tfsdk:"billing_interval"`
}

func (d *offersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_offers"
}

func (d *offersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Catalogue des offres publiques (`GET /v1/offers`).",
		Attributes: map[string]schema.Attribute{
			"offers": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: offerAttrs(),
				},
			},
		},
	}
}

func offerAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":               schema.Int64Attribute{Computed: true},
		"name":             schema.StringAttribute{Computed: true},
		"price":            schema.Float64Attribute{Computed: true},
		"ram":              schema.Int64Attribute{Computed: true},
		"cpu_cores":        schema.Int64Attribute{Computed: true},
		"storage":          schema.Int64Attribute{Computed: true},
		"billing_interval": schema.StringAttribute{Computed: true},
	}
}

func (d *offersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *offersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state offersDataSourceModel
	offers, err := d.client.ListOffers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Lecture offres", err.Error())
		return
	}
	state.Offers = make([]offerItemModel, 0, len(offers))
	for _, o := range offers {
		state.Offers = append(state.Offers, offerToItem(o))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func offerToItem(o client.Offer) offerItemModel {
	return offerItemModel{
		ID:              types.Int64Value(o.ID),
		Name:            types.StringValue(o.Name),
		Price:           types.Float64Value(o.Price),
		Ram:             types.Int64Value(o.Ram),
		CpuCores:        types.Int64Value(o.CpuCores),
		Storage:         types.Int64Value(o.Storage),
		BillingInterval: types.StringValue(o.BillingInterval),
	}
}
