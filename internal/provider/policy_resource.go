package provider

import (
	"context"
	"fmt"
	"net/http"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
// The Dependency-Track API ignores includeChildren and global on policy
// creation, so both are read-only (Computed) here. Policy updates, however,
// overwrite includeChildren, onlyLatestProjectVersion and (on 5.2+)
// invertTagMatch with whatever the request carries, and client-go's Policy
// cannot express all of them. Updates therefore go through updatePolicy,
// which reads the stored policy first and sends back the server's values for
// every field this resource does not manage.
type PolicyResource struct {
	data *Data
}

// PolicyResourceModel describes the resource data model.
type PolicyResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Operator        types.String `tfsdk:"operator"`
	ViolationState  types.String `tfsdk:"violation_state"`
	IncludeChildren types.Bool   `tfsdk:"include_children"`
	Global          types.Bool   `tfsdk:"global"`
	InvertTagMatch  types.Bool   `tfsdk:"invert_tag_match"`
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
			"invert_tag_match": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the policy's tag scope is inverted, so that it applies to projects that do **not** have any of the policy's tags. Requires Dependency-Track 5.2 or newer; on older servers it can only be `false`. Default: `false`",
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

	r.data = data
}

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() || !r.checkInvertTagMatchSupported(&data, &resp.Diagnostics) {
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
	createdPolicy, err := r.data.Client.Policy.Create(ctx, policy)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy, got error: %s", err))
		return
	}

	// Policy creation ignores invertTagMatch, so set it with a follow-up update.
	if data.InvertTagMatch.ValueBool() {
		if err := r.updatePolicy(ctx, createdPolicy.UUID, &data); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set invert_tag_match on policy, got error: %s", err))
			return
		}
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

			createdCondition, err := r.data.Client.PolicyCondition.Create(ctx, createdPolicy.UUID, apiCondition)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy condition, got error: %s", err))
				return
			}

			conditions[i].UUID = types.StringValue(createdCondition.UUID.String())
		}
	}

	// Read back the policy to get complete state
	readPolicy, err := r.data.Client.Policy.Get(ctx, createdPolicy.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy after create, got error: %s", err))
		return
	}

	// Update model with created values
	r.updateModelFromAPI(&data, &readPolicy)
	if err := r.readInvertTagMatch(ctx, createdPolicy.UUID, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy after create, got error: %s", err))
		return
	}

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

	policy, err := r.data.Client.Policy.Get(ctx, policyUUID)
	if err != nil {
		// If policy not found, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update model with values from API
	r.updateModelFromAPI(&data, &policy)
	if err := r.readInvertTagMatch(ctx, policyUUID, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() || !r.checkInvertTagMatchSupported(&plan, &resp.Diagnostics) {
		return
	}

	policyUUID, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Invalid policy UUID: %s", err))
		return
	}

	if err := r.updatePolicy(ctx, policyUUID, &plan); err != nil {
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
		err = r.data.Client.PolicyCondition.Delete(ctx, condUUID)
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

		createdCondition, err := r.data.Client.PolicyCondition.Create(ctx, policyUUID, apiCondition)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create policy condition, got error: %s", err))
			return
		}

		planConditions[i].UUID = types.StringValue(createdCondition.UUID.String())
	}

	// Read back the policy to get complete state
	readPolicy, err := r.data.Client.Policy.Get(ctx, policyUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy after update, got error: %s", err))
		return
	}

	// Update model with updated values
	r.updateModelFromAPI(&plan, &readPolicy)
	if err := r.readInvertTagMatch(ctx, policyUUID, &plan); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy after update, got error: %s", err))
		return
	}

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

	err = r.data.Client.Policy.Delete(ctx, policyUUID)
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

// policyUpdateBody holds the fields Dependency-Track's policy update endpoint
// (POST /api/v1/policy) writes. Every field is sent explicitly, because the
// server overwrites each of them from the request.
type policyUpdateBody struct {
	UUID                     string `json:"uuid"`
	Name                     string `json:"name"`
	Operator                 string `json:"operator"`
	ViolationState           string `json:"violationState"`
	IncludeChildren          bool   `json:"includeChildren"`
	OnlyLatestProjectVersion bool   `json:"onlyLatestProjectVersion"`
	InvertTagMatch           bool   `json:"invertTagMatch"`
}

// updatePolicy updates the managed policy attributes from data while keeping
// the server's current values for the fields this resource does not manage
// (includeChildren, onlyLatestProjectVersion, and invertTagMatch before 5.2).
func (r *PolicyResource) updatePolicy(ctx context.Context, policyUUID uuid.UUID, data *PolicyResourceModel) error {
	var body policyUpdateBody
	if err := r.data.API().Do(ctx, http.MethodGet, "/api/v1/policy/"+policyUUID.String(), nil, &body); err != nil {
		return fmt.Errorf("read current policy: %w", err)
	}

	body.UUID = policyUUID.String()
	body.Name = data.Name.ValueString()
	body.Operator = data.Operator.ValueString()
	body.ViolationState = data.ViolationState.ValueString()
	if r.data.ServerVersion.AtLeast(5, 2) {
		body.InvertTagMatch = data.InvertTagMatch.ValueBool()
	}

	return r.data.API().Do(ctx, http.MethodPost, "/api/v1/policy", body, nil)
}

// readPolicyInvertTagMatch returns a policy's invertTagMatch flag. client-go's
// Policy has no such field, so it is read with a raw request on 5.2+; older
// servers have no inverted tag matching and always report false.
func readPolicyInvertTagMatch(ctx context.Context, data *Data, policyUUID uuid.UUID) (types.Bool, error) {
	if !data.ServerVersion.AtLeast(5, 2) {
		return types.BoolValue(false), nil
	}

	var body policyUpdateBody
	if err := data.API().Do(ctx, http.MethodGet, "/api/v1/policy/"+policyUUID.String(), nil, &body); err != nil {
		return types.BoolNull(), err
	}
	return types.BoolValue(body.InvertTagMatch), nil
}

// readInvertTagMatch sets data.InvertTagMatch from the server.
func (r *PolicyResource) readInvertTagMatch(ctx context.Context, policyUUID uuid.UUID, data *PolicyResourceModel) error {
	v, err := readPolicyInvertTagMatch(ctx, r.data, policyUUID)
	if err != nil {
		return err
	}
	data.InvertTagMatch = v
	return nil
}

// checkInvertTagMatchSupported rejects invert_tag_match = true on servers
// older than Dependency-Track 5.2, which have no inverted tag matching.
func (r *PolicyResource) checkInvertTagMatchSupported(data *PolicyResourceModel, diags *diag.Diagnostics) bool {
	if !data.InvertTagMatch.ValueBool() {
		return true
	}
	return requireAtLeast(r.data, 5, 2, "Setting invert_tag_match", diags)
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
