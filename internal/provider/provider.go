package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hostaria/terraform-provider-hostaria/internal/client"
)

var _ provider.Provider = &hostariaProvider{}

type hostariaProvider struct {
	version string
}

type hostariaProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &hostariaProvider{version: version}
	}
}

func (p *hostariaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hostaria"
	resp.Version = p.version
}

func (p *hostariaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider Hostaria (Terraform / OpenTofu). Appelle `https://api.hostaria.cloud/v1/...` — jamais `/api/v1` ni `/v1/api/v1`.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Hôte API, défaut `https://api.hostaria.cloud`. Peut déjà finir par `/v1`. Surcharge : `HOSTARIA_ENDPOINT`.",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Bearer (`host_live_…` ou JWT de transition). Surcharge : `HOSTARIA_API_KEY`.",
			},
		},
	}
}

func (p *hostariaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg hostariaProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("HOSTARIA_ENDPOINT")
	if v := strings.TrimSpace(cfg.Endpoint.ValueString()); !cfg.Endpoint.IsNull() && !cfg.Endpoint.IsUnknown() && v != "" {
		endpoint = v
	}
	apiKey := os.Getenv("HOSTARIA_API_KEY")
	if v := strings.TrimSpace(cfg.APIKey.ValueString()); !cfg.APIKey.IsNull() && !cfg.APIKey.IsUnknown() && v != "" {
		apiKey = v
	}
	c := client.New(endpoint, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *hostariaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVMResource,
		NewSSHKeyResource,
		NewGameServerResource,
	}
}

func (p *hostariaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOffersDataSource,
		NewOfferDataSource,
		NewOSDataSource,
	}
}

func configureClient(providerData any, diags interface{ AddError(string, string) }) *client.Client {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError("Configure inattendu", "Le provider n'a pas transmis le client Hostaria.")
		return nil
	}
	return c
}
