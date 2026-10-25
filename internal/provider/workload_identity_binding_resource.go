package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkloadIdentityBindingResource{}
var _ resource.ResourceWithImportState = &WorkloadIdentityBindingResource{}

func NewWorkloadIdentityBindingResource() resource.Resource {
	return &WorkloadIdentityBindingResource{}
}

// workloadIdentitySubjectRegex is the server-side pattern for binding
// subjects: an optional trailing "*" makes a prefix match, whose prefix must
// end with ":" or "/".
var workloadIdentitySubjectRegex = regexp.MustCompile(`^[^*]+(?:[:/]\*)?$`)

// WorkloadIdentityBindingResource defines the resource implementation.
type WorkloadIdentityBindingResource struct {
	data *Data
}

// WorkloadIdentityBindingResourceModel describes the resource data model.
type WorkloadIdentityBindingResourceModel struct {
	ID             types.String `tfsdk:"id"`
	UUID           types.String `tfsdk:"uuid"`
	ServiceAccount types.String `tfsdk:"service_account"`
	ProviderName   types.String `tfsdk:"provider_name"`
	Subject        types.String `tfsdk:"subject"`
	Condition      types.String `tfsdk:"condition"`
}

// workloadIdentityBinding is the request body for POST
// /api/v2/service-accounts/{name}/workload-identity-bindings and an item of
// the corresponding list response.
type workloadIdentityBinding struct {
	UUID         string `json:"uuid,omitempty"`
	ProviderName string `json:"provider_name"`
	Subject      string `json:"subject"`
	Condition    string `json:"condition,omitempty"`
}

func workloadIdentityBindingsPath(serviceAccount string) string {
	return serviceAccountPath(serviceAccount) + "/workload-identity-bindings"
}

func (r *WorkloadIdentityBindingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_identity_binding"
}

func (r *WorkloadIdentityBindingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Binds a workload identity to a Dependency-Track service account: tokens from `provider_name` whose `sub` claim matches " +
			"`subject` (and whose claims satisfy `condition`, if set) can be exchanged for a session of the service account. " +
			"Bindings cannot be modified: changing any argument replaces the binding. " +
			"Requires Dependency-Track 5.2 or newer and the `ACCESS_MANAGEMENT` permission.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the binding in the format `service_account/uuid`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the binding",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_account": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the service account (without the `svc:` prefix)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the workload identity provider whose tokens to trust",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subject": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The `sub` claim to match, such as `repo:acme/app:ref:refs/heads/main`. A trailing `*` makes it a prefix match; " +
					"the prefix must not be empty and must end with `:` or `/`, such as `repo:acme/app:*`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 512),
					stringvalidator.RegexMatches(workloadIdentitySubjectRegex, "a \"*\" is only allowed at the end, directly after \":\" or \"/\""),
				},
			},
			"condition": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "A CEL expression over the token's claims (the `claims` map) that must evaluate to `true`, " +
					"such as `claims.ref == \"refs/heads/main\"`. Use it when the subject alone cannot express the rule",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 2048),
				},
			},
		},
	}
}

func (r *WorkloadIdentityBindingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkloadIdentityBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkloadIdentityBindingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_workload_identity_binding resource", &resp.Diagnostics) {
		return
	}

	body := workloadIdentityBinding{
		ProviderName: data.ProviderName.ValueString(),
		Subject:      data.Subject.ValueString(),
		Condition:    data.Condition.ValueString(),
	}

	var created workloadIdentityBinding
	if err := r.data.API().Do(ctx, http.MethodPost, workloadIdentityBindingsPath(data.ServiceAccount.ValueString()), body, &created); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workload identity binding, got error: %s", err))
		return
	}
	if created.UUID == "" {
		resp.Diagnostics.AddError("Client Error", "Dependency-Track did not return the UUID of the created workload identity binding.")
		return
	}

	data.UUID = types.StringValue(created.UUID)
	data.ID = types.StringValue(data.ServiceAccount.ValueString() + "/" + created.UUID)

	tflog.Trace(ctx, "created a workload identity binding", map[string]any{"service_account": data.ServiceAccount.ValueString(), "uuid": created.UUID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadIdentityBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkloadIdentityBindingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_workload_identity_binding resource", &resp.Diagnostics) {
		return
	}

	// There is no endpoint for a single binding; find it in the list.
	bindings, err := apiGetAllV2Pages[workloadIdentityBinding](ctx, r.data.API(), workloadIdentityBindingsPath(data.ServiceAccount.ValueString()))
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workload identity bindings, got error: %s", err))
		return
	}

	for _, b := range bindings {
		if b.UUID != data.UUID.ValueString() {
			continue
		}
		data.ID = types.StringValue(data.ServiceAccount.ValueString() + "/" + b.UUID)
		data.ProviderName = types.StringValue(b.ProviderName)
		data.Subject = types.StringValue(b.Subject)
		if b.Condition == "" {
			data.Condition = types.StringNull()
		} else {
			data.Condition = types.StringValue(b.Condition)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.State.RemoveResource(ctx)
}

// Update is never called: every argument requires replacement.
func (r *WorkloadIdentityBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkloadIdentityBindingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadIdentityBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkloadIdentityBindingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_workload_identity_binding resource", &resp.Diagnostics) {
		return
	}

	bindingPath := workloadIdentityBindingsPath(data.ServiceAccount.ValueString()) + "/" + url.PathEscape(data.UUID.ValueString())
	err := r.data.API().Do(ctx, http.MethodDelete, bindingPath, nil, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete workload identity binding, got error: %s", err))
		return
	}
}

func (r *WorkloadIdentityBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serviceAccount, bindingUUID, err := parseCompositeID(req.ID, "service_account", "uuid")
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account"), serviceAccount)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), bindingUUID)...)
}
