package provider

import (
	"context"
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
var _ resource.Resource = &ProjectPolicyResource{}
var _ resource.ResourceWithImportState = &ProjectPolicyResource{}

func NewProjectPolicyResource() resource.Resource {
	return &ProjectPolicyResource{}
}

// ProjectPolicyResource defines the resource implementation.
type ProjectPolicyResource struct {
	client *dtrack.Client
}

// ProjectPolicyResourceModel describes the resource data model.
type ProjectPolicyResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Policy  types.String `tfsdk:"policy"`
	Project types.String `tfsdk:"project"`
}

func (r *ProjectPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_policy"
}

func (r *ProjectPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the assignment of a policy to a project in Dependency-Track. This resource creates a relationship between a project and a policy, enabling policy enforcement for that specific project.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the project policy assignment in the format `policy_uuid/project_uuid`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the policy",
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

func (r *ProjectPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy UUID", fmt.Sprintf("Unable to parse policy UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	_, err = r.client.Policy.AddProject(ctx, policyUUID, projectUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add project to policy, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", policyUUID.String(), projectUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy UUID", fmt.Sprintf("Unable to parse policy UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	// Get the policy and check if the project is in its projects list
	policy, err := r.client.Policy.Get(ctx, policyUUID)
	if err != nil {
		if apiErr, ok := err.(*dtrack.APIError); ok && apiErr.StatusCode == 404 {
			// Policy doesn't exist anymore, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	// Check if the project is in the policy's projects list
	found := false
	for _, project := range policy.Projects {
		if project.UUID == projectUUID {
			found = true
			break
		}
	}

	if !found {
		// Project is not assigned to the policy anymore, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", policyUUID.String(), projectUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since both policy and project have RequiresReplace, this should never be called
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"Project policy assignments cannot be updated. Both policy and project changes require replacement.",
	)
}

func (r *ProjectPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy UUID", fmt.Sprintf("Unable to parse policy UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	_, err = r.client.Policy.DeleteProject(ctx, policyUUID, projectUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove project from policy, got error: %s", err))
		return
	}
}

func (r *ProjectPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: policy_uuid/project_uuid
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format: policy_uuid/project_uuid, got: %s", req.ID),
		)
		return
	}

	policyUUID, err := uuid.Parse(parts[0])
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Policy UUID",
			fmt.Sprintf("Unable to parse policy UUID from import ID: %s\nError: %s", parts[0], err),
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy"), policyUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), projectUUID.String())...)
}
