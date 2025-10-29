package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ACLMappingResource{}
var _ resource.ResourceWithImportState = &ACLMappingResource{}

func NewACLMappingResource() resource.Resource {
	return &ACLMappingResource{}
}

// ACLMappingResource defines the resource implementation.
type ACLMappingResource struct {
	client *dtrack.Client
}

// ACLMappingResourceModel describes the resource data model.
type ACLMappingResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Team    types.String `tfsdk:"team"`
	Project types.String `tfsdk:"project"`
}

func (r *ACLMappingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_mapping"
}

func (r *ACLMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an ACL mapping between a team and a project in Dependency-Track. ACL mappings control which teams have access to which projects when portfolio access control is enabled.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the ACL mapping in the format `team_uuid/project_uuid`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the team",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the project",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ACLMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ACLMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ACLMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	mapping := dtrack.ACLMappingRequest{
		Team:    teamUUID,
		Project: projectUUID,
	}

	err = r.client.ACL.AddProjectMapping(ctx, mapping)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ACL mapping, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", teamUUID.String(), projectUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ACLMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	// Use ForEach to check if the project is in the team's ACL with automatic pagination
	found := false
	errProjectFound := errors.New("project found") // Sentinel error to break out of ForEach

	err = dtrack.ForEach(func(po dtrack.PageOptions) (dtrack.Page[dtrack.Project], error) {
		return r.client.ACL.GetAllProjects(ctx, teamUUID, po)
	}, func(project dtrack.Project) error {
		if project.UUID == projectUUID {
			found = true
			return errProjectFound // Return sentinel error to stop iteration
		}
		return nil
	})

	// Check if iteration was stopped because project was found
	if err != nil && !errors.Is(err, errProjectFound) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACL mappings, got error: %s", err))
		return
	}

	if !found {
		// Mapping doesn't exist anymore, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", teamUUID.String(), projectUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since both team and project have RequiresReplace, this should never be called
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"ACL mappings cannot be updated. Both team and project changes require replacement.",
	)
}

func (r *ACLMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ACLMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	err = r.client.ACL.RemoveProjectMapping(ctx, teamUUID, projectUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ACL mapping, got error: %s", err))
		return
	}
}

func (r *ACLMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: team_uuid/project_uuid
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: team_uuid/project_uuid, got: %s", req.ID),
		)
		return
	}

	teamUUID, err := uuid.Parse(parts[0])
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Team UUID",
			fmt.Sprintf("Unable to parse team UUID from import ID: %s\nError: %s", parts[0], err),
		)
		return
	}

	projectUUID, err := uuid.Parse(parts[1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Project UUID",
			fmt.Sprintf("Unable to parse project UUID from import ID: %s\nError: %s", parts[1], err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), teamUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), projectUUID.String())...)
}
