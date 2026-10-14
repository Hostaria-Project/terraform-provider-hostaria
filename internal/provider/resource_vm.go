package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hostaria/terraform-provider-hostaria/internal/client"
)

var (
	_ resource.Resource                = &vmResource{}
	_ resource.ResourceWithConfigure   = &vmResource{}
	_ resource.ResourceWithImportState = &vmResource{}
)

func NewVMResource() resource.Resource { return &vmResource{} }

type vmResource struct {
	client *client.Client
}

type bootstrapModel struct {
	Username       types.String `tfsdk:"username"`
	AuthMethod     types.String `tfsdk:"auth_method"`
	SSHPublicKeys  types.List   `tfsdk:"ssh_public_keys"`
	SSHKeyIDs      types.List   `tfsdk:"ssh_key_ids"`
	Password       types.String `tfsdk:"password"`
	UserData       types.String `tfsdk:"user_data"`
	UserDataFormat types.String `tfsdk:"user_data_format"`
}

type vmResourceModel struct {
	ID        types.Int64     `tfsdk:"id"`
	UUID      types.String    `tfsdk:"uuid"`
	OfferID   types.Int64     `tfsdk:"offer_id"`
	OsID      types.Int64     `tfsdk:"os_id"`
	Name      types.String    `tfsdk:"name"`
	Hostname  types.String    `tfsdk:"hostname"`
	Status    types.String    `tfsdk:"status"`
	IpAddress types.String    `tfsdk:"ip_address"`
	Until     types.String    `tfsdk:"until"`
	CpuCores  types.Int64     `tfsdk:"cpu_cores"`
	RamMB     types.Int64     `tfsdk:"ram_mb"`
	StorageGB types.Int64     `tfsdk:"storage_gb"`
	Bootstrap *bootstrapModel `tfsdk:"bootstrap"`
}

func (r *vmResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vps"
}

func (r *vmResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VPS Hostaria (`POST/GET/PUT/DELETE /v1/vps`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "ID public du VPS.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID d'infrastructure (temporaire puis réel).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"offer_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Offre catalogue. Changer d'offre resize in-place (pas de recreate). Disque non réductible.",
			},
			"os_id": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Nom demandé. L'API y ajoute un suffixe d'unicité ; le name-label réel est `hostname`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Nom réel (XCP), ex. tf-smoke-01-2808753805.",
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"ip_address": schema.StringAttribute{
				Computed: true,
			},
			"until": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "RFC3339 : destruction automatique (TTL). Facturation à l'usage jusqu'à cette date.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cpu_cores": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "vCPU désirés (plancher = offre). Update in-place, VM haltée le temps du resize.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"ram_mb": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "RAM en Mo (pas de 512). Update in-place.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"storage_gb": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Disque Go, agrandissement uniquement.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"bootstrap": schema.SingleNestedBlock{
				MarkdownDescription: "Premier boot : utilisateur, clés SSH, fragment cloud-config. Si aucune clé n'est fournie ici, l'API injecte les clés SSH du compte (Compte → IaC). `user_data` ne s'applique qu'à la création : le modifier recrée le VPS.",
				Attributes: map[string]schema.Attribute{
					"username":        schema.StringAttribute{Optional: true},
					"auth_method":     schema.StringAttribute{Optional: true},
					"ssh_public_keys": schema.ListAttribute{Optional: true, ElementType: types.StringType},
					"ssh_key_ids":     schema.ListAttribute{Optional: true, ElementType: types.StringType},
					"password":        schema.StringAttribute{Optional: true, Sensitive: true},
					"user_data": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						MarkdownDescription: "Fragment #cloud-config fusionné au socle Hostaria. Autorisé : hostname, fqdn, packages, timezone, locale. Interdit : runcmd, write_files, réseau (IP, netplan, DNS), utilisateur, clés SSH. Premier boot uniquement.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"user_data_format": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "cloud-config (seul format v1). Défaut si vide.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
		},
	}
}

func (r *vmResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *vmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cr := client.CreateVMRequest{
		OfferID: plan.OfferID.ValueInt64(),
		OsID:    plan.OsID.ValueInt64(),
		Name:    plan.Name.ValueString(),
		Until:   plan.Until.ValueString(),
	}
	if !plan.CpuCores.IsNull() && !plan.CpuCores.IsUnknown() {
		cr.CpuCores = plan.CpuCores.ValueInt64()
	}
	if !plan.RamMB.IsNull() && !plan.RamMB.IsUnknown() {
		cr.RamMB = plan.RamMB.ValueInt64()
	}
	if !plan.StorageGB.IsNull() && !plan.StorageGB.IsUnknown() {
		cr.StorageGB = plan.StorageGB.ValueInt64()
	}
	if plan.Bootstrap != nil {
		boot := &client.Bootstrap{
			Username:       plan.Bootstrap.Username.ValueString(),
			AuthMethod:     plan.Bootstrap.AuthMethod.ValueString(),
			Password:       plan.Bootstrap.Password.ValueString(),
			UserData:       plan.Bootstrap.UserData.ValueString(),
			UserDataFormat: plan.Bootstrap.UserDataFormat.ValueString(),
		}
		if !plan.Bootstrap.SSHPublicKeys.IsNull() && !plan.Bootstrap.SSHPublicKeys.IsUnknown() {
			resp.Diagnostics.Append(plan.Bootstrap.SSHPublicKeys.ElementsAs(ctx, &boot.SSHPublicKeys, false)...)
		}
		if !plan.Bootstrap.SSHKeyIDs.IsNull() && !plan.Bootstrap.SSHKeyIDs.IsUnknown() {
			resp.Diagnostics.Append(plan.Bootstrap.SSHKeyIDs.ElementsAs(ctx, &boot.SSHKeyIDs, false)...)
		}
		if resp.Diagnostics.HasError() {
			return
		}
		cr.Bootstrap = boot
	}

	vm, err := r.client.CreateVM(ctx, cr)
	if err != nil {
		msg := err.Error()
		if ae, ok := err.(*client.APIError); ok {
			switch ae.Code {
			case "payment_method_required":
				msg = "Carte requise : ajoutez un moyen de paiement dans Compte → Facturation, puis relancez apply."
			case "ssh_key_required":
				msg = "Clé SSH requise : hostaria_ssh_key ou bootstrap.ssh_public_keys, ou une clé sur le profil."
			case "credit_limit_exceeded":
				msg = "Plafond d'usage atteint. Détruisez des VPS ou attendez le prélèvement."
			case "disk_down":
				msg = "Le disque ne peut qu'être agrandi (XCP ne réduit pas un VDI)."
			}
		}
		resp.Diagnostics.AddError("Création VPS", msg)
		return
	}
	setVMModel(&plan, vm)
	warnSuggestedOffer(&resp.Diagnostics, vm)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lookup := state.UUID.ValueString()
	if lookup == "" && !state.ID.IsNull() {
		lookup = strconv.FormatInt(state.ID.ValueInt64(), 10)
	}
	vm, err := r.client.GetVM(ctx, lookup)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Lecture VPS", err.Error())
		return
	}
	boot := state.Bootstrap
	setVMModel(&state, vm)
	state.Bootstrap = boot
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *vmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state vmResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lookup := plan.UUID.ValueString()
	if lookup == "" {
		lookup = strconv.FormatInt(plan.ID.ValueInt64(), 10)
	}
	body := map[string]any{}
	if !plan.OfferID.Equal(state.OfferID) {
		body["offer_id"] = plan.OfferID.ValueInt64()
	}
	if !plan.CpuCores.IsNull() && !plan.CpuCores.IsUnknown() && !plan.CpuCores.Equal(state.CpuCores) {
		body["cpu_cores"] = plan.CpuCores.ValueInt64()
	}
	if !plan.RamMB.IsNull() && !plan.RamMB.IsUnknown() && !plan.RamMB.Equal(state.RamMB) {
		body["ram_mb"] = plan.RamMB.ValueInt64()
	}
	if !plan.StorageGB.IsNull() && !plan.StorageGB.IsUnknown() && !plan.StorageGB.Equal(state.StorageGB) {
		body["storage_gb"] = plan.StorageGB.ValueInt64()
	}
	var (
		vm  *client.VM
		err error
	)
	if len(body) == 0 {
		vm, err = r.client.GetVM(ctx, lookup)
	} else {
		vm, err = r.client.UpdateVM(ctx, lookup, body)
	}
	if err != nil {
		msg := err.Error()
		if ae, ok := err.(*client.APIError); ok && ae.Code == "disk_down" {
			msg = "Le disque ne peut qu'être agrandi (XCP ne réduit pas un VDI)."
		}
		resp.Diagnostics.AddError("Mise à jour VPS", msg)
		return
	}
	boot := plan.Bootstrap
	setVMModel(&plan, vm)
	plan.Bootstrap = boot
	warnSuggestedOffer(&resp.Diagnostics, vm)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *vmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vmResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lookup := state.UUID.ValueString()
	if lookup == "" {
		lookup = strconv.FormatInt(state.ID.ValueInt64(), 10)
	}
	if err := r.client.DeleteVM(ctx, lookup); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Suppression VPS", err.Error())
	}
}

func (r *vmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if n, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), n)...)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
}

func setVMModel(m *vmResourceModel, vm *client.VM) {
	requested := m.Name
	m.ID = types.Int64Value(vm.ID)
	m.UUID = types.StringValue(vm.IDVm)
	m.OfferID = types.Int64Value(vm.OfferID)
	m.OsID = types.Int64Value(vm.OsID)
	m.Hostname = types.StringValue(vm.Name)
	if !requested.IsNull() && !requested.IsUnknown() && strings.TrimSpace(requested.ValueString()) != "" {
		m.Name = requested
	} else {
		m.Name = types.StringValue(vm.Name)
	}
	m.Status = types.StringValue(vm.Status)
	if vm.CpuCores > 0 {
		m.CpuCores = types.Int64Value(vm.CpuCores)
	}
	if vm.RamMB > 0 {
		m.RamMB = types.Int64Value(vm.RamMB)
	}
	if vm.StorageGB > 0 {
		m.StorageGB = types.Int64Value(vm.StorageGB)
	}
	if vm.IpAddress != nil {
		m.IpAddress = types.StringValue(*vm.IpAddress)
	} else {
		m.IpAddress = types.StringNull()
	}
}

func warnSuggestedOffer(diags interface{ AddWarning(string, string) }, vm *client.VM) {
	if vm == nil || vm.SuggestedOffer == nil || vm.SuggestedOffer.ID == 0 {
		return
	}
	s := vm.SuggestedOffer
	msg := s.Name
	if s.SavingsCents > 0 {
		msg += " économiserait environ " + strconv.FormatInt(s.SavingsCents, 10) + " ct/mois"
	}
	diags.AddWarning(
		"Offre catalogue plus rentable",
		msg+". Passe offer_id à "+strconv.FormatInt(s.ID, 10)+" dans le HCL si tu veux la bascule. Terraform ne change pas l'offre tout seul.",
	)
}
