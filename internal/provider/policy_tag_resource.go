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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyTagResource{}
var _ resource.ResourceWithImportState = &PolicyTagResource{}

func NewPolicyTagResource() resource.Resource {
	return &PolicyTagResource{}
}

// PolicyTagResource defines the resource implementation.
type PolicyTagResource struct {
	data *Data
}

// PolicyTagResourceModel describes the resource data model.
type PolicyTagResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Tag    types.String `tfsdk:"tag"`
	Policy types.String `tfsdk:"policy"`
}

func (r *PolicyTagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_tag"
}

func (r *PolicyTagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the assignment of a tag to a policy in Dependency-Track. The tag must already exist (see the `dependencytrack_tag` resource).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the policy tag assignment in the format `tag_name/policy_uuid`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the tag. Dependency-Track normalizes tag names to lowercase, so a mixed-case name is matched case-insensitively (using a lowercase name is recommended). Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the policy",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *PolicyTagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicyTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyTagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy UUID", fmt.Sprintf("Unable to parse policy UUID: %s", err))
		return
	}

	// Dependency-Track stores tag names lowercased; normalize before using the
	// name in the request path so lookups resolve consistently.
	tag := strings.ToLower(data.Tag.ValueString())

	err = r.data.Client.Tag.TagPolicies(ctx, tag, []uuid.UUID{policyUUID})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to tag policy, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.Tag.ValueString(), policyUUID.String()))

	tflog.Trace(ctx, "created a policy tag resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyTagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy UUID", fmt.Sprintf("Unable to parse policy UUID: %s", err))
		return
	}

	tag := strings.ToLower(data.Tag.ValueString())

	// The tagged-policies listing of a nonexistent tag is an empty list (not
	// an error) on both DT v4 and v5, so a deleted tag also falls into the
	// not-found path below.
	policies, err := fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.TaggedPolicyListResponseItem], error) {
		return r.data.Client.Tag.GetPolicies(ctx, tag, po, dtrack.SortOptions{})
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tagged policies, got error: %s", err))
		return
	}

	found := false
	for i := range policies {
		if policies[i].UUID == policyUUID {
			found = true
			break
		}
	}

	if !found {
		// Tag is not assigned to the policy anymore, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.Tag.ValueString(), policyUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since both tag and policy have RequiresReplace, this should never be called
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"Policy tag assignments cannot be updated. Both tag and policy changes require replacement.",
	)
}

func (r *PolicyTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyTagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.Policy.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Policy UUID", fmt.Sprintf("Unable to parse policy UUID: %s", err))
		return
	}

	tag := strings.ToLower(data.Tag.ValueString())

	err = r.data.Client.Tag.UntagPolicies(ctx, tag, []uuid.UUID{policyUUID})
	if err != nil {
		// The tag or policy being gone means there is nothing left to delete.
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to untag policy, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a policy tag resource")
}

func (r *PolicyTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: tag_name/policy_uuid
	tag, policyID, err := parseCompositeID(req.ID, "tag_name", "policy_uuid")
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Policy UUID",
			fmt.Sprintf("Unable to parse policy UUID from import ID: %s\nError: %s", policyID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag"), tag)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy"), policyUUID.String())...)
}
