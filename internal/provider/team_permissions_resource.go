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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TeamPermissionsResource{}
var _ resource.ResourceWithImportState = &TeamPermissionsResource{}

func NewTeamPermissionsResource() resource.Resource {
	return &TeamPermissionsResource{}
}

// TeamPermissionsResource defines the resource implementation.
type TeamPermissionsResource struct {
	client *dtrack.Client
}

// TeamPermissionsResourceModel describes the resource data model.
type TeamPermissionsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Team        types.String `tfsdk:"team"`
	Permissions types.Set    `tfsdk:"permissions"`
}

func (r *TeamPermissionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_permissions"
}

func (r *TeamPermissionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages permissions for a Dependency-Track team. This resource manages the complete set of permissions assigned to a team. Available permissions: ACCESS_MANAGEMENT, BOM_UPLOAD, POLICY_MANAGEMENT, POLICY_VIOLATION_ANALYSIS, PORTFOLIO_MANAGEMENT, PROJECT_CREATION_UPLOAD, SYSTEM_CONFIGURATION, TAG_MANAGEMENT, VIEW_BADGES, VIEW_POLICY_VIOLATION, VIEW_PORTFOLIO, VIEW_VULNERABILITY, VULNERABILITY_ANALYSIS, VULNERABILITY_MANAGEMENT.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the team",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the team to manage permissions for",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Set of permission names to assign to the team (e.g., BOM_UPLOAD, PORTFOLIO_MANAGEMENT)",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TeamPermissionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = data.Client
}

func (r *TeamPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TeamPermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Get desired permissions from plan
	var desiredPermissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &desiredPermissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Add each permission to the team
	for _, permName := range desiredPermissions {
		permission := dtrack.Permission{
			Name: permName,
		}
		_, err = r.client.Permission.AddPermissionToTeam(ctx, permission, teamUUID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add permission %s to team, got error: %s", permName, err))
			return
		}
	}

	// Read back the team to get actual permissions from the API
	team, err := r.client.Team.Get(ctx, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team after create, got error: %s", err))
		return
	}

	// Extract current permissions from team
	actualPermissions := make([]string, 0, len(team.Permissions))
	for _, perm := range team.Permissions {
		actualPermissions = append(actualPermissions, perm.Name)
	}

	// Convert to Set type
	permissionsSet, diags := types.SetValueFrom(ctx, types.StringType, actualPermissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(teamUUID.String())
	data.Permissions = permissionsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamPermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	team, err := r.client.Team.Get(ctx, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
		return
	}

	// Check if team exists
	if team.UUID == uuid.Nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Extract current permissions from team
	currentPermissions := make([]string, 0, len(team.Permissions))
	for _, perm := range team.Permissions {
		currentPermissions = append(currentPermissions, perm.Name)
	}

	// Convert to Set type
	permissionsSet, diags := types.SetValueFrom(ctx, types.StringType, currentPermissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(team.UUID.String())
	data.Permissions = permissionsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamPermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(plan.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Get current and desired permissions
	var currentPermissions, desiredPermissions []string
	resp.Diagnostics.Append(state.Permissions.ElementsAs(ctx, &currentPermissions, false)...)
	resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &desiredPermissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to maps for easier lookup
	currentMap := make(map[string]bool)
	desiredMap := make(map[string]bool)
	for _, p := range currentPermissions {
		currentMap[p] = true
	}
	for _, p := range desiredPermissions {
		desiredMap[p] = true
	}

	// Add permissions that are in desired but not in current
	for _, permName := range desiredPermissions {
		if !currentMap[permName] {
			permission := dtrack.Permission{
				Name: permName,
			}
			_, err = r.client.Permission.AddPermissionToTeam(ctx, permission, teamUUID)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add permission %s to team, got error: %s", permName, err))
				return
			}
		}
	}

	// Remove permissions that are in current but not in desired
	for _, permName := range currentPermissions {
		if !desiredMap[permName] {
			permission := dtrack.Permission{
				Name: permName,
			}
			_, err = r.client.Permission.RemovePermissionFromTeam(ctx, permission, teamUUID)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove permission %s from team, got error: %s", permName, err))
				return
			}
		}
	}

	// Read back the team to get actual permissions from the API
	team, err := r.client.Team.Get(ctx, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team after update, got error: %s", err))
		return
	}

	// Extract current permissions from team
	actualPermissions := make([]string, 0, len(team.Permissions))
	for _, perm := range team.Permissions {
		actualPermissions = append(actualPermissions, perm.Name)
	}

	// Convert to Set type
	permissionsSet, diags := types.SetValueFrom(ctx, types.StringType, actualPermissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(teamUUID.String())
	plan.Permissions = permissionsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamPermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Get all permissions to remove
	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove each permission from the team
	for _, permName := range permissions {
		permission := dtrack.Permission{
			Name: permName,
		}
		_, err = r.client.Permission.RemovePermissionFromTeam(ctx, permission, teamUUID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove permission %s from team, got error: %s", permName, err))
			return
		}
	}
}

func (r *TeamPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using team UUID
	teamUUID, err := uuid.Parse(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse UUID. Expected a valid team UUID, got: %s\nError: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), teamUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), teamUUID.String())...)
}
