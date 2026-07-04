package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

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
var _ resource.Resource = &SecretResource{}
var _ resource.ResourceWithImportState = &SecretResource{}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

// SecretResource defines the resource implementation.
type SecretResource struct {
	data *Data
}

// SecretResourceModel describes the resource data model.
type SecretResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
}

// createSecretRequest is the request body for POST /api/v2/secrets.
type createSecretRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value"`
}

// updateSecretRequest is the request body for PATCH /api/v2/secrets/{name}.
// Omitted fields retain their current server-side value, so both fields are
// always sent to keep the secret fully described by configuration.
type updateSecretRequest struct {
	Description *string `json:"description,omitempty"`
	Value       *string `json:"value,omitempty"`
}

// secretMetadata is the response body of GET /api/v2/secrets/{name}. The
// secret value itself is write-only and never returned by the API.
type secretMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (r *SecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a secret in Dependency-Track's secret manager. " +
			"Secrets are referenced by name from configuration fields annotated as secret references, " +
			"such as extension configuration fields (see `dependencytrack_extension_config`) and " +
			"`dependencytrack_repository.password`. " +
			"The secret value is write-only: Dependency-Track never returns it, so Terraform cannot detect " +
			"drift of the value itself (only of its metadata). " +
			"Requires Dependency-Track v5 and the `SECRET_MANAGEMENT` permission " +
			"(plus `SYSTEM_CONFIGURATION` or `SYSTEM_CONFIGURATION_READ` to read metadata).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the secret (same as `name`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the secret. Must match `^[a-zA-Z0-9_-]{1,64}$`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(secretNameRegex, "must contain only letters, digits, underscores, and hyphens (max 64 characters)"),
				},
			},
			"value": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "The value of the secret (1-4096 characters). Write-only: never read back from the API",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 4096),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the secret (max 255 characters)",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
		},
	}
}

func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireV5(r.data, "dependencytrack_secret", &resp.Diagnostics) {
		return
	}

	createReq := createSecretRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Value:       data.Value.ValueString(),
	}

	if err := r.data.API().Do(ctx, http.MethodPost, "/api/v2/secrets", createReq, nil); err != nil {
		if apiErrorStatusCode(err) == http.StatusConflict {
			resp.Diagnostics.AddError(
				"Secret Already Exists",
				fmt.Sprintf("A secret named %q already exists in Dependency-Track. "+
					"Import it with `terraform import` to manage it with this resource.", data.Name.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create secret, got error: %s", err))
		return
	}

	data.ID = data.Name

	tflog.Trace(ctx, "created a secret", map[string]any{"name": data.Name.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireV5(r.data, "dependencytrack_secret", &resp.Diagnostics) {
		return
	}

	var meta secretMetadata
	err := r.data.API().Do(ctx, http.MethodGet, "/api/v2/secrets/"+url.PathEscape(data.Name.ValueString()), nil, &meta)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read secret, got error: %s", err))
		return
	}

	data.ID = data.Name
	if meta.Description == "" {
		data.Description = types.StringNull()
	} else {
		data.Description = types.StringValue(meta.Description)
	}
	// The secret value is never returned by the API; the state value is
	// preserved as-is (write-only semantics).

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireV5(r.data, "dependencytrack_secret", &resp.Diagnostics) {
		return
	}

	// Always send both fields: PATCH retains omitted fields, but the
	// Terraform configuration is the full desired state. An unset
	// description is sent as an empty string to clear it on the server.
	value := data.Value.ValueString()
	description := data.Description.ValueString()
	updateReq := updateSecretRequest{
		Description: &description,
		Value:       &value,
	}

	err := r.data.API().Do(ctx, http.MethodPatch, "/api/v2/secrets/"+url.PathEscape(data.Name.ValueString()), updateReq, nil)
	if err != nil && !isNotModified(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update secret, got error: %s", err))
		return
	}

	data.ID = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireV5(r.data, "dependencytrack_secret", &resp.Diagnostics) {
		return
	}

	err := r.data.API().Do(ctx, http.MethodDelete, "/api/v2/secrets/"+url.PathEscape(data.Name.ValueString()), nil, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete secret, got error: %s", err))
		return
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the secret name. The secret value cannot be read from the
	// API; the first apply after import updates it to the configured value.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
