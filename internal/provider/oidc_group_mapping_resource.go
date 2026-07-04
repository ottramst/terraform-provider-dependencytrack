package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &OIDCGroupMappingResource{}
var _ resource.ResourceWithImportState = &OIDCGroupMappingResource{}

func NewOIDCGroupMappingResource() resource.Resource {
	return &OIDCGroupMappingResource{}
}

// OIDCGroupMappingResource defines the resource implementation.
type OIDCGroupMappingResource struct {
	data *Data
}

// OIDCGroupMappingResourceModel describes the resource data model.
type OIDCGroupMappingResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Group types.String `tfsdk:"group"`
	Team  types.String `tfsdk:"team"`
}

func (r *OIDCGroupMappingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_group_mapping"
}

func (r *OIDCGroupMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Maps an OpenID Connect (OIDC) group to a team in Dependency-Track. Users authenticated via OIDC inherit the permissions of every team their groups map to. OIDC groups and mappings are plain database entities and can be managed without an identity provider configured.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the mapping in the format `group_uuid/team_uuid`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the OIDC group. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"team": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the team. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *OIDCGroupMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.data = data
}

func (r *OIDCGroupMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OIDCGroupMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.Group.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group UUID", fmt.Sprintf("Unable to parse OIDC group UUID: %s", err))
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	_, err = r.data.Client.OIDC.AddTeamMapping(ctx, dtrack.OIDCMappingRequest{
		Group: groupUUID,
		Team:  teamUUID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create OIDC group mapping, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", groupUUID.String(), teamUUID.String()))

	tflog.Trace(ctx, "created an OIDC group mapping resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OIDCGroupMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OIDCGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.Group.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group UUID", fmt.Sprintf("Unable to parse OIDC group UUID: %s", err))
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	teams, err := r.data.Client.OIDC.GetAllTeamsOf(ctx, dtrack.OIDCGroup{UUID: groupUUID})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read OIDC group mappings, got error: %s", err))
		return
	}

	found := false
	for i := range teams {
		if teams[i].UUID == teamUUID {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", groupUUID.String(), teamUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OIDCGroupMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both group and team have RequiresReplace, so this is never called.
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"OIDC group mappings cannot be updated. Both group and team changes require replacement.",
	)
}

func (r *OIDCGroupMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OIDCGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.Group.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group UUID", fmt.Sprintf("Unable to parse OIDC group UUID: %s", err))
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	err = r.data.Client.OIDC.RemoveTeamMapping2(ctx, groupUUID, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete OIDC group mapping, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an OIDC group mapping resource")
}

func (r *OIDCGroupMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: group_uuid/team_uuid.
	groupUUIDStr, teamUUIDStr, err := parseCompositeID(req.ID, "group_uuid", "team_uuid")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID: %s", err),
		)
		return
	}

	groupUUID, err := uuid.Parse(groupUUIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Group UUID",
			fmt.Sprintf("Unable to parse group UUID from import ID: %s\nError: %s", groupUUIDStr, err),
		)
		return
	}

	teamUUID, err := uuid.Parse(teamUUIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Team UUID",
			fmt.Sprintf("Unable to parse team UUID from import ID: %s\nError: %s", teamUUIDStr, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group"), groupUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), teamUUID.String())...)
}
