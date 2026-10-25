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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ServiceAccountResource{}
var _ resource.ResourceWithImportState = &ServiceAccountResource{}

func NewServiceAccountResource() resource.Resource {
	return &ServiceAccountResource{}
}

// ServiceAccountResource defines the resource implementation.
type ServiceAccountResource struct {
	data *Data
}

// ServiceAccountResourceModel describes the resource data model.
type ServiceAccountResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Username  types.String `tfsdk:"username"`
	Email     types.String `tfsdk:"email"`
	Suspended types.Bool   `tfsdk:"suspended"`
}

// createServiceAccountRequest is the request body for POST /api/v2/service-accounts.
type createServiceAccountRequest struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// updateServiceAccountRequest is the request body for PATCH
// /api/v2/service-accounts/{name}. Omitted fields retain their current value,
// and an empty email removes the current address.
type updateServiceAccountRequest struct {
	Email     *string `json:"email,omitempty"`
	Suspended *bool   `json:"suspended,omitempty"`
}

// serviceAccountTeam is a team membership in a getServiceAccountResponse.
type serviceAccountTeam struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// getServiceAccountResponse is the response body of GET
// /api/v2/service-accounts/{name}. Permissions lists only the permissions
// granted to the account directly, not those inherited from its teams.
type getServiceAccountResponse struct {
	Name        string               `json:"name"`
	Username    string               `json:"username"`
	Email       string               `json:"email,omitempty"`
	Suspended   bool                 `json:"suspended"`
	Teams       []serviceAccountTeam `json:"teams"`
	Permissions []string             `json:"permissions"`
}

// serviceAccountPath returns the /api/v2 path of the named service account.
func serviceAccountPath(name string) string {
	return "/api/v2/service-accounts/" + url.PathEscape(name)
}

// getServiceAccount fetches the named service account, returning nil if it
// does not exist.
func getServiceAccount(ctx context.Context, data *Data, name string) (*getServiceAccountResponse, error) {
	var sa getServiceAccountResponse
	if err := data.API().Do(ctx, http.MethodGet, serviceAccountPath(name), nil, &sa); err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &sa, nil
}

func (r *ServiceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *ServiceAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Dependency-Track service account: a non-human identity for automation such as CI pipelines. " +
			"A service account authenticates with its own expiring API keys (see `dependencytrack_service_account_api_key`) " +
			"or through workload identity federation. Its username is its name with the reserved `svc:` prefix; use that username " +
			"with `dependencytrack_managed_user_permissions` and `dependencytrack_user_team_membership` to grant permissions and team memberships. " +
			"Requires Dependency-Track 5.2 or newer and the `ACCESS_MANAGEMENT` permission.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the service account (same as `name`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the service account, without the `svc:` prefix (max 59 characters). Must start with a letter or digit, followed by letters, digits, or `+=,.:@_-`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(59),
					stringvalidator.RegexMatches(serviceAccountNameRegex, "must start with a letter or digit, followed by letters, digits, or +=,.:@_-"),
					serviceAccountNameValidator{},
				},
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full username of the service account: its name with the `svc:` prefix",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The email address of the service account (max 255 characters)",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"suspended": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the service account is suspended. A suspended account cannot authenticate. Default: `false`",
			},
		},
	}
}

func (r *ServiceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_service_account resource", &resp.Diagnostics) {
		return
	}

	createReq := createServiceAccountRequest{
		Name:  data.Name.ValueString(),
		Email: data.Email.ValueString(),
	}

	if err := r.data.API().Do(ctx, http.MethodPost, "/api/v2/service-accounts", createReq, nil); err != nil {
		if apiErrorStatusCode(err) == http.StatusConflict {
			resp.Diagnostics.AddError(
				"Service Account Already Exists",
				fmt.Sprintf("A service account named %q already exists in Dependency-Track. "+
					"Import it with `terraform import` to manage it with this resource.", data.Name.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service account, got error: %s", err))
		return
	}

	// New service accounts start out active; suspend with a follow-up update.
	if data.Suspended.ValueBool() {
		suspended := true
		err := r.data.API().Do(ctx, http.MethodPatch, serviceAccountPath(data.Name.ValueString()), updateServiceAccountRequest{Suspended: &suspended}, nil)
		if err != nil && !isNotModified(err) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to suspend service account, got error: %s", err))
			return
		}
	}

	if _, err := r.read(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account after create, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a service account", map[string]any{"name": data.Name.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_service_account resource", &resp.Diagnostics) {
		return
	}

	found, err := r.read(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got error: %s", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_service_account resource", &resp.Diagnostics) {
		return
	}

	// Always send both fields so the account matches the configuration. An
	// unset email is sent as an empty string, which removes it on the server.
	email := data.Email.ValueString()
	suspended := data.Suspended.ValueBool()
	updateReq := updateServiceAccountRequest{
		Email:     &email,
		Suspended: &suspended,
	}

	err := r.data.API().Do(ctx, http.MethodPatch, serviceAccountPath(data.Name.ValueString()), updateReq, nil)
	if err != nil && !isNotModified(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update service account, got error: %s", err))
		return
	}

	if _, err := r.read(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account after update, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_service_account resource", &resp.Diagnostics) {
		return
	}

	err := r.data.API().Do(ctx, http.MethodDelete, serviceAccountPath(data.Name.ValueString()), nil, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete service account, got error: %s", err))
		return
	}
}

func (r *ServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

// read refreshes data from the server and reports whether the service account
// exists.
func (r *ServiceAccountResource) read(ctx context.Context, data *ServiceAccountResourceModel) (bool, error) {
	sa, err := getServiceAccount(ctx, r.data, data.Name.ValueString())
	if err != nil || sa == nil {
		return false, err
	}

	data.ID = types.StringValue(sa.Name)
	data.Name = types.StringValue(sa.Name)
	data.Username = types.StringValue(sa.Username)
	data.Suspended = types.BoolValue(sa.Suspended)
	if sa.Email == "" {
		data.Email = types.StringNull()
	} else {
		data.Email = types.StringValue(sa.Email)
	}
	return true, nil
}
