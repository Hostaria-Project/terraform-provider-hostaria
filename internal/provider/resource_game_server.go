package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hostaria/terraform-provider-hostaria/internal/client"
)

var (
	_ resource.Resource                = &gameServerResource{}
	_ resource.ResourceWithConfigure   = &gameServerResource{}
	_ resource.ResourceWithImportState = &gameServerResource{}
)

func NewGameServerResource() resource.Resource { return &gameServerResource{} }

type gameServerResource struct {
	client *client.Client
}

type gameServerModel struct {
	ID      types.String `tfsdk:"id"`
	Game    types.String `tfsdk:"game"`
	Name    types.String `tfsdk:"name"`
	OfferID types.Int64  `tfsdk:"offer_id"`
	Status  types.String `tfsdk:"status"`
}

func (r *gameServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_game_server"
}

func (r *gameServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Serveur de jeu (`/v1/game-servers`, game = ark | satisfactory).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"game": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringOneOf("ark", "satisfactory"),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"offer_id": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *gameServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *gameServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gameServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	gs, err := r.client.CreateGameServer(ctx, client.GameServer{
		Game:    plan.Game.ValueString(),
		Name:    plan.Name.ValueString(),
		OfferID: plan.OfferID.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Création game server", err.Error())
		return
	}
	setGameServer(&plan, gs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gameServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gameServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	gs, err := r.client.GetGameServer(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Lecture game server", err.Error())
		return
	}
	setGameServer(&state, gs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gameServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gameServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	gs, err := r.client.UpdateGameServer(ctx, plan.ID.ValueString(), client.GameServer{
		Game:    plan.Game.ValueString(),
		Name:    plan.Name.ValueString(),
		OfferID: plan.OfferID.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Mise à jour game server", err.Error())
		return
	}
	setGameServer(&plan, gs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gameServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gameServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteGameServer(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Suppression game server", err.Error())
	}
}

func (r *gameServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func setGameServer(m *gameServerModel, gs *client.GameServer) {
	m.ID = types.StringValue(gs.ID)
	m.Game = types.StringValue(gs.Game)
	m.Name = types.StringValue(gs.Name)
	m.OfferID = types.Int64Value(gs.OfferID)
	m.Status = types.StringValue(gs.Status)
}
