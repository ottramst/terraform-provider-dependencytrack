package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ServiceAccountAPIKeyResource{}
var _ resource.ResourceWithImportState = &ServiceAccountAPIKeyResource{}

func NewServiceAccountAPIKeyResource() resource.Resource {
	return &ServiceAccountAPIKeyResource{}
}

// ServiceAccountAPIKeyResource defines the resource implementation.
type ServiceAccountAPIKeyResource struct {
	data *Data
}

// ServiceAccountAPIKeyResourceModel describes the resource data model.
type ServiceAccountAPIKeyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ServiceAccount types.String `tfsdk:"service_account"`
	Comment        types.String `tfsdk:"comment"`
	ExpiresInDays  types.Int64  `tfsdk:"expires_in_days"`
	PublicID       types.String `tfsdk:"public_id"`
	Key            types.String `tfsdk:"key"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
}

// createServiceAccountAPIKeyRequest is the request body for POST
// /api/v2/service-accounts/{name}/api-keys.
type createServiceAccountAPIKeyRequest struct {
	Comment       string `json:"comment,omitempty"`
	ExpiresInDays *int64 `json:"expires_in_days,omitempty"`
}

// createServiceAccountAPIKeyResponse is the response body of POST
// /api/v2/service-accounts/{name}/api-keys. The plain text key is returned
// only here, exactly once.
type createServiceAccountAPIKeyResponse struct {
	PublicID  string `json:"public_id"`
	Key       string `json:"key"`
	ExpiresAt int64  `json:"expires_at"`
}

// serviceAccountAPIKey is an item of GET /api/v2/service-accounts/{name}/api-keys.
// Timestamps are epoch milliseconds.
type serviceAccountAPIKey struct {
	PublicID  string `json:"public_id"`
	Comment   string `json:"comment,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// epochMillisString formats an epoch-milliseconds timestamp as RFC 3339, or
// returns null for zero (absent).
func epochMillisString(ms int64) types.String {
	if ms == 0 {
		return types.StringNull()
	}
	return types.StringValue(time.UnixMilli(ms).UTC().Format(time.RFC3339))
}

func (r *ServiceAccountAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_api_key"
}

func (r *ServiceAccountAPIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an API key of a Dependency-Track service account. " +
			"Service account API keys always expire, and the plain text key is only returned when the key is created. " +
			"API keys cannot be modified: changing any argument replaces the key. To rotate a key on a schedule, " +
			"tie it to a `time_rotating` resource with `replace_triggered_by`. " +
			"Requires Dependency-Track 5.2 or newer and the `ACCESS_MANAGEMENT` permission.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the API key in the format `service_account/public_id`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_account": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the service account the key belongs to (without the `svc:` prefix)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A comment describing the key (max 255 characters)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"expires_in_days": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "Number of days after creation when the key expires. Must not exceed the server's maximum " +
					"key lifetime (`dt.api-key.max-lifetime-days`, 366 days by default). When unset, the server uses 30 days, " +
					"or the maximum lifetime if that is shorter. Cannot be read back, so it is not set after import",
				PlanModifiers: []planmodifier.Int64{
					// Imported keys have no expires_in_days in state; setting it in
					// configuration afterwards must not replace the key.
					int64planmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = !req.StateValue.IsNull()
						},
						"Replaces the key, unless the key was imported without this attribute.",
						"Replaces the key, unless the key was imported without this attribute.",
					),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"public_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The public, non-secret part of the key",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The plain text API key. Only available when the key is created; null after import",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the key was created (RFC 3339)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the key expires (RFC 3339)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ServiceAccountAPIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceAccountAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceAccountAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_service_account_api_key resource", &resp.Diagnostics) {
		return
	}

	createReq := createServiceAccountAPIKeyRequest{Comment: data.Comment.ValueString()}
	if !data.ExpiresInDays.IsNull() && !data.ExpiresInDays.IsUnknown() {
		days := data.ExpiresInDays.ValueInt64()
		createReq.ExpiresInDays = &days
	}

	var created createServiceAccountAPIKeyResponse
	err := r.data.API().Do(ctx, http.MethodPost, serviceAccountPath(data.ServiceAccount.ValueString())+"/api-keys", createReq, &created)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service account API key, got error: %s", err))
		return
	}

	data.ID = types.StringValue(data.ServiceAccount.ValueString() + "/" + created.PublicID)
	data.PublicID = types.StringValue(created.PublicID)
	data.Key = types.StringValue(created.Key)
	data.ExpiresAt = epochMillisString(created.ExpiresAt)

	// created_at is only part of the list response.
	found, err := r.read(ctx, &data)
	if err != nil || !found {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account API key after create (found: %t), got error: %v", found, err))
		return
	}

	tflog.Trace(ctx, "created a service account API key", map[string]any{"service_account": data.ServiceAccount.ValueString(), "public_id": created.PublicID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceAccountAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_service_account_api_key resource", &resp.Diagnostics) {
		return
	}

	found, err := r.read(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account API key, got error: %s", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update only runs when expires_in_days is set on an imported key (every other
// change replaces the key), so it just records the planned value.
func (r *ServiceAccountAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceAccountAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceAccountAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_service_account_api_key resource", &resp.Diagnostics) {
		return
	}

	keyPath := serviceAccountPath(data.ServiceAccount.ValueString()) + "/api-keys/" + url.PathEscape(data.PublicID.ValueString())
	err := r.data.API().Do(ctx, http.MethodDelete, keyPath, nil, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service account API key, got error: %s", err))
		return
	}
}

func (r *ServiceAccountAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serviceAccount, publicID, err := parseCompositeID(req.ID, "service_account", "public_id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_account"), serviceAccount)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("public_id"), publicID)...)
}

// read refreshes data from the service account's key list and reports whether
// the key (and its service account) still exist. There is no endpoint for a
// single key.
func (r *ServiceAccountAPIKeyResource) read(ctx context.Context, data *ServiceAccountAPIKeyResourceModel) (bool, error) {
	keys, err := apiGetAllV2Pages[serviceAccountAPIKey](ctx, r.data.API(), serviceAccountPath(data.ServiceAccount.ValueString())+"/api-keys")
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, k := range keys {
		if k.PublicID != data.PublicID.ValueString() {
			continue
		}
		if k.Comment == "" {
			data.Comment = types.StringNull()
		} else {
			data.Comment = types.StringValue(k.Comment)
		}
		data.CreatedAt = epochMillisString(k.CreatedAt)
		data.ExpiresAt = epochMillisString(k.ExpiresAt)
		return true, nil
	}

	return false, nil
}
