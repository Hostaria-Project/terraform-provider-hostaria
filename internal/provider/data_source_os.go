package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hostaria/terraform-provider-hostaria/internal/client"
)

var (
	_ datasource.DataSource              = &osDataSource{}
	_ datasource.DataSourceWithConfigure = &osDataSource{}
)

func NewOSDataSource() datasource.DataSource { return &osDataSource{} }

type osDataSource struct {
	client *client.Client
}

type osDataSourceModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Slug     types.String `tfsdk:"slug"`
	OS       types.String `tfsdk:"os"`
	Version  types.String `tfsdk:"version"`
	Family   types.String `tfsdk:"family"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

func (d *osDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_os"
}

func (d *osDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Un OS du catalogue, par `id` ou `slug` (`GET /v1/os`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"name":      schema.StringAttribute{Computed: true},
			"slug":      schema.StringAttribute{Optional: true, Computed: true},
			"os":        schema.StringAttribute{Computed: true},
			"version":   schema.StringAttribute{Computed: true},
			"family":    schema.StringAttribute{Computed: true},
			"is_active": schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *osDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *osDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg osDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}
	if !cfg.ID.IsNull() && !cfg.ID.IsUnknown() && cfg.ID.ValueInt64() > 0 {
		osRow, err := d.client.GetOS(ctx, cfg.ID.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Lecture OS", err.Error())
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, osToModel(*osRow))...)
		return
	}
	slug := strings.TrimSpace(cfg.Slug.ValueString())
	if slug == "" {
		resp.Diagnostics.AddError("OS", "fournir id ou slug")
		return
	}
	list, err := d.client.ListOS(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Lecture OS", err.Error())
		return
	}
	for _, o := range list {
		if strings.EqualFold(o.Slug, slug) {
			resp.Diagnostics.Append(resp.State.Set(ctx, osToModel(o))...)
			return
		}
	}
	resp.Diagnostics.AddError("OS introuvable", fmt.Sprintf("aucun OS slug %q", slug))
}

func osToModel(o client.OS) osDataSourceModel {
	return osDataSourceModel{
		ID:       types.Int64Value(o.ID),
		Name:     types.StringValue(o.Name),
		Slug:     types.StringValue(o.Slug),
		OS:       types.StringValue(o.OS),
		Version:  types.StringValue(o.Version),
		Family:   types.StringValue(o.Family),
		IsActive: types.BoolValue(o.IsActive),
	}
}
