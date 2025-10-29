package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

// PolicyResource defines the resource implementation.
//
// IMPORTANT NOTE: Testing has revealed that the Dependency-Track API (as of v4.x) ignores
// the includeChildren and global fields during policy creation and update operations.
// The API always defaults global to true and includeChildren to false, regardless
// of what values are sent. Therefore, these fields are read-only (Computed) in the
// Terraform schema to accurately reflect API behavior and prevent perpetual drift.
// Users who need to control these values should file an issue with the Dependency-Track project.
type PolicyResource struct {
	client *dtrack.Client
}

// PolicyResourceModel describes the resource data model.
type PolicyResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Operator        types.String `tfsdk:"operator"`
	ViolationState  types.String `tfsdk:"violation_state"`
	IncludeChildren types.Bool   `tfsdk:"include_children"`
	Global          types.Bool   `tfsdk:"global"`
	Conditions      types.List   `tfsdk:"conditions"`
}

// PolicyConditionModel describes a policy condition.
type PolicyConditionModel struct {
	UUID     types.String `tfsdk:"uuid"`
	Subject  types.String `tfsdk:"subject"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

func (r *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Dependency-Track policy. Policies are used to automatically audit components based on specific criteria and trigger alerts or fail builds.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the policy (same as uuid)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the policy",
			},
			"operator": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("ALL"),
				MarkdownDescription: "The operator to use when evaluating conditions (ALL or ANY). Default: ALL",
				Validators: []validator.String{
					stringvalidator.OneOf("ALL", "ANY"),
				},
			},
			"violation_state": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("INFO"),
				MarkdownDescription: "The violation state (INFO, WARN, or FAIL). Default: INFO",
				Validators: []validator.String{
					stringvalidator.OneOf("INFO", "WARN", "FAIL"),
				},
			},
			"include_children": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the policy applies to child projects. This field is read-only as the Dependency-Track API does not support setting it.",
			},
			"global": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is a global policy. This field is read-only as the Dependency-Track API always sets it to true and does not support changing it.",
			},
			"conditions": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "List of policy conditions that must be met",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(0),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the condition",
						},
						"subject": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The subject of the condition (AGE, COORDINATES, CPE, LICENSE, LICENSE_GROUP, PACKAGE_URL, SEVERITY, SWID_TAGID, VERSION, COMPONENT_HASH, CWE, VULNERABILITY_ID, VERSION_DISTANCE, EPSS)",
							Validators: []validator.String{
								stringvalidator.OneOf("AGE", "COORDINATES", "CPE", "LICENSE", "LICENSE_GROUP", "PACKAGE_URL", "SEVERITY", "SWID_TAGID", "VERSION", "COMPONENT_HASH", "CWE", "VULNERABILITY_ID", "VERSION_DISTANCE", "EPSS"),
							},
						},
						"operator": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The operator for the condition (IS, IS_NOT, MATCHES, NO_MATCH, NUMERIC_GREATER_THAN, NUMERIC_LESS_THAN, NUMERIC_EQUAL, NUMERIC_NOT_EQUAL, NUMERIC_GREATER_THAN_OR_EQUAL, NUMERIC_LESSER_THAN_OR_EQUAL, CONTAINS_ALL, CONTAINS_ANY)",
							Validators: []validator.String{
								stringvalidator.OneOf("IS", "IS_NOT", "MATCHES", "NO_MATCH", "NUMERIC_GREATER_THAN", "NUMERIC_LESS_THAN", "NUMERIC_EQUAL", "NUMERIC_NOT_EQUAL", "NUMERIC_GREATER_THAN_OR_EQUAL", "NUMERIC_LESSER_THAN_OR_EQUAL", "CONTAINS_ALL", "CONTAINS_ANY"),
							},
						},
						"value": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The value to compare against",
						},
					},
				},
			},
		},
	}
}

func (r *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform model to client library model
	// Note: We don't set IncludeChildren or Global because the API ignores these fields
	policy := dtrack.Policy{
		Name:           data.Name.ValueString(),
		Operator:       dtrack.PolicyOperator(data.Operator.ValueString()),
		ViolationState: dtrack.PolicyViolationState(data.ViolationState.ValueString()),
	}

	// Create policy using client library
	createdPolicy, err := r.client.Policy.Create(ctx, policy)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy, got error: %s", err))
		return
	}

	// Create conditions if any
	var conditions []PolicyConditionModel
	if !data.Conditions.IsNull() && !data.Conditions.IsUnknown() {
		resp.Diagnostics.Append(data.Conditions.ElementsAs(ctx, &conditions, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for i, condition := range conditions {
			apiCondition := dtrack.PolicyCondition{
				Subject:  dtrack.PolicyConditionSubject(condition.Subject.ValueString()),
				Operator: dtrack.PolicyConditionOperator(condition.Operator.ValueString()),
				Value:    condition.Value.ValueString(),
			}

			createdCondition, err := r.client.PolicyCondition.Create(ctx, createdPolicy.UUID, apiCondition)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy condition, got error: %s", err))
				return
			}

			conditions[i].UUID = types.StringValue(createdCondition.UUID.String())
		}
	}

	// Read back the policy to get complete state
	readPolicy, err := r.client.Policy.Get(ctx, createdPolicy.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy after create, got error: %s", err))
		return
	}

	// Update model with created values
	r.updateModelFromAPI(&data, &readPolicy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Invalid policy UUID: %s", err))
		return
	}

	policy, err := r.client.Policy.Get(ctx, policyUUID)
	if err != nil {
		// If policy not found, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with values from API
	r.updateModelFromAPI(&data, &policy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Invalid policy UUID: %s", err))
		return
	}

	// Update policy basic attributes
	// Note: We don't set IncludeChildren or Global because the API ignores these fields
	policy := dtrack.Policy{
		UUID:           policyUUID,
		Name:           plan.Name.ValueString(),
		Operator:       dtrack.PolicyOperator(plan.Operator.ValueString()),
		ViolationState: dtrack.PolicyViolationState(plan.ViolationState.ValueString()),
	}

	updatedPolicy, err := r.client.Policy.Update(ctx, policy)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update policy, got error: %s", err))
		return
	}

	// Handle conditions - delete old ones and create new ones
	var stateConditions, planConditions []PolicyConditionModel
	if !state.Conditions.IsNull() && !state.Conditions.IsUnknown() {
		resp.Diagnostics.Append(state.Conditions.ElementsAs(ctx, &stateConditions, false)...)
	}
	if !plan.Conditions.IsNull() && !plan.Conditions.IsUnknown() {
		resp.Diagnostics.Append(plan.Conditions.ElementsAs(ctx, &planConditions, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete all existing conditions
	for _, condition := range stateConditions {
		condUUID, err := uuid.Parse(condition.UUID.ValueString())
		if err != nil {
			continue // Skip invalid UUIDs
		}
		err = r.client.PolicyCondition.Delete(ctx, condUUID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy condition, got error: %s", err))
			return
		}
	}

	// Create new conditions
	for i, condition := range planConditions {
		apiCondition := dtrack.PolicyCondition{
			Subject:  dtrack.PolicyConditionSubject(condition.Subject.ValueString()),
			Operator: dtrack.PolicyConditionOperator(condition.Operator.ValueString()),
			Value:    condition.Value.ValueString(),
		}

		createdCondition, err := r.client.PolicyCondition.Create(ctx, updatedPolicy.UUID, apiCondition)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy condition, got error: %s", err))
			return
		}

		planConditions[i].UUID = types.StringValue(createdCondition.UUID.String())
	}

	// Read back the policy to get complete state
	readPolicy, err := r.client.Policy.Get(ctx, updatedPolicy.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy after update, got error: %s", err))
		return
	}

	// Update model with updated values
	r.updateModelFromAPI(&plan, &readPolicy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Invalid policy UUID: %s", err))
		return
	}

	err = r.client.Policy.Delete(ctx, policyUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete policy, got error: %s", err))
		return
	}
}

func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using UUID
	_, err := uuid.Parse(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse UUID. Expected a valid policy UUID, got: %s\nError: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper method to update model from client library Policy struct.
func (r *PolicyResource) updateModelFromAPI(data *PolicyResourceModel, policy *dtrack.Policy) {
	data.ID = types.StringValue(policy.UUID.String())
	data.Name = types.StringValue(policy.Name)
	data.Operator = types.StringValue(string(policy.Operator))
	data.ViolationState = types.StringValue(string(policy.ViolationState))
	data.IncludeChildren = types.BoolValue(policy.IncludeChildren)
	data.Global = types.BoolValue(policy.Global)

	// Update conditions
	if len(policy.PolicyConditions) > 0 {
		conditionElements := make([]attr.Value, 0, len(policy.PolicyConditions))
		for _, cond := range policy.PolicyConditions {
			conditionElements = append(conditionElements, types.ObjectValueMust(
				map[string]attr.Type{
					"uuid":     types.StringType,
					"subject":  types.StringType,
					"operator": types.StringType,
					"value":    types.StringType,
				},
				map[string]attr.Value{
					"uuid":     types.StringValue(cond.UUID.String()),
					"subject":  types.StringValue(string(cond.Subject)),
					"operator": types.StringValue(string(cond.Operator)),
					"value":    types.StringValue(cond.Value),
				},
			))
		}
		conditionsList, _ := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"uuid":     types.StringType,
					"subject":  types.StringType,
					"operator": types.StringType,
					"value":    types.StringType,
				},
			},
			conditionElements,
		)
		data.Conditions = conditionsList
	} else {
		data.Conditions = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"uuid":     types.StringType,
				"subject":  types.StringType,
				"operator": types.StringType,
				"value":    types.StringType,
			},
		})
	}
}
