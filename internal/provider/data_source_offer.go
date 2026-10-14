package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hostaria/terraform-provider-hostaria/internal/client"
)

var (
	_ datasource.DataSource              = &offerDataSource{}
	_ datasource.DataSourceWithConfigure = &offerDataSource{}
)

func NewOfferDataSource() datasource.DataSource { return &offerDataSource{} }

type offerDataSource struct {
	client *client.Client
}

func (d *offerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_offer"
}

func (d *offerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Une offre, par `id` ou `name` (`GET /v1/offers`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"price":            schema.Float64Attribute{Computed: true},
			"ram":              schema.Int64Attribute{Computed: true},
			"cpu_cores":        schema.Int64Attribute{Computed: true},
			"storage":          schema.Int64Attribute{Computed: true},
			"billing_interval": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *offerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *offerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg offerItemModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}
	if !cfg.ID.IsNull() && !cfg.ID.IsUnknown() && cfg.ID.ValueInt64() > 0 {
		off, err := d.client.GetOffer(ctx, cfg.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Lecture offre", err.Error())
			return
		}
		item := offerToItem(*off)
		resp.Diagnostics.Append(resp.State.Set(ctx, &item)...)
		return
	}
	name := strings.TrimSpace(cfg.Name.ValueString())
	if name == "" {
		resp.Diagnostics.AddError("Offre", "fournir id ou name")
		return
	}
	list, err := d.client.ListOffers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Lecture offres", err.Error())
		return
	}
	for _, o := range list {
		if strings.EqualFold(o.Name, name) {
			item := offerToItem(o)
			resp.Diagnostics.Append(resp.State.Set(ctx, &item)...)
			return
		}
	}
	resp.Diagnostics.AddError("Offre introuvable", fmt.Sprintf("aucune offre nommée %q", name))
}
