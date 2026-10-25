package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkloadIdentityProviderResource{}
var _ resource.ResourceWithImportState = &WorkloadIdentityProviderResource{}

func NewWorkloadIdentityProviderResource() resource.Resource {
	return &WorkloadIdentityProviderResource{}
}

// WorkloadIdentityProviderResource defines the resource implementation.
type WorkloadIdentityProviderResource struct {
	data *Data
}

// WorkloadIdentityProviderResourceModel describes the resource data model.
type WorkloadIdentityProviderResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	Issuer                 types.String `tfsdk:"issuer"`
	Audience               types.String `tfsdk:"audience"`
	JWKSURL                types.String `tfsdk:"jwks_url"`
	JWKS                   types.String `tfsdk:"jwks"`
	JWKSKeyIDs             types.List   `tfsdk:"jwks_key_ids"`
	SessionLifetimeSeconds types.Int64  `tfsdk:"session_lifetime_seconds"`
}

// workloadIdentityProviderRequest is the request body for POST
// /api/v2/workload-identity-providers and (without name and type) PATCH
// /api/v2/workload-identity-providers/{name}, where omitted fields retain
// their current value.
type workloadIdentityProviderRequest struct {
	Name                   string          `json:"name,omitempty"`
	Type                   string          `json:"type,omitempty"`
	Issuer                 string          `json:"issuer"`
	Audience               string          `json:"audience"`
	JWKSURL                string          `json:"jwks_url,omitempty"`
	JWKS                   json.RawMessage `json:"jwks,omitempty"`
	SessionLifetimeSeconds int64           `json:"session_lifetime_seconds"`
}

// workloadIdentityProviderResponse is the response body of GET
// /api/v2/workload-identity-providers/{name}. An inline key set is never
// returned, only its key IDs.
type workloadIdentityProviderResponse struct {
	Name                   string   `json:"name"`
	Type                   string   `json:"type"`
	Issuer                 string   `json:"issuer"`
	Audience               string   `json:"audience"`
	JWKSURL                string   `json:"jwks_url,omitempty"`
	JWKSKeyIDs             []string `json:"jwks_key_ids,omitempty"`
	SessionLifetimeSeconds int64    `json:"session_lifetime_seconds"`
}

func workloadIdentityProviderPath(name string) string {
	return "/api/v2/workload-identity-providers/" + url.PathEscape(name)
}

func (r *WorkloadIdentityProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workload_identity_provider"
}

func (r *WorkloadIdentityProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Dependency-Track workload identity provider: an external token issuer, such as GitHub Actions, " +
			"whose tokens service accounts can exchange for Dependency-Track sessions. Which tokens map to which service account is " +
			"defined by `dependencytrack_workload_identity_binding`. " +
			"Signing keys come from `jwks` (inline), `jwks_url`, or, for `OIDC` providers when both are unset, the issuer's discovery document. " +
			"Requires Dependency-Track 5.2 or newer and the `ACCESS_MANAGEMENT` permission.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the workload identity provider (same as `name`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the workload identity provider (max 63 characters)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 63),
				},
			},
			"type": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The type of issuer: `OIDC` for an OpenID Connect issuer (tokens' `iss` must equal `issuer`), " +
					"or `SPIFFE` for a SPIFFE trust domain (`sub` must be a SPIFFE ID of that domain)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("OIDC", "SPIFFE"),
				},
			},
			"issuer": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The expected `iss` claim for `OIDC` providers, or the SPIFFE trust domain (such as `example.org`) for `SPIFFE` providers",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"audience": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The value the token's `aud` claim must contain",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"jwks_url": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "The `https` URL to fetch the issuer's signing keys from. Conflicts with `jwks`. " +
					"Leave both unset on an `OIDC` provider to resolve the URL from the issuer's discovery document",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 2048),
					stringvalidator.ConflictsWith(path.MatchRoot("jwks")),
				},
			},
			"jwks": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "An inline RFC 7517 JSON Web Key Set (JSON string, e.g. from `jsonencode()` or `file()`) holding the issuer's public signing keys. " +
					"Dependency-Track does not return it, so changes made outside Terraform are only detected through `jwks_key_ids`",
			},
			"jwks_key_ids": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The key IDs of the inline key set. Null when the keys are fetched from a URL",
			},
			"session_lifetime_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3600),
				MarkdownDescription: "The lifetime in seconds (60-86400) of sessions created through this provider. Default: `3600`",
				Validators: []validator.Int64{
					int64validator.Between(60, 86400),
				},
			},
		},
	}
}

func (r *WorkloadIdentityProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// request builds the create/update request body from data.
func (r *WorkloadIdentityProviderResource) request(data *WorkloadIdentityProviderResourceModel) (workloadIdentityProviderRequest, error) {
	body := workloadIdentityProviderRequest{
		Issuer:                 data.Issuer.ValueString(),
		Audience:               data.Audience.ValueString(),
		JWKSURL:                data.JWKSURL.ValueString(),
		SessionLifetimeSeconds: data.SessionLifetimeSeconds.ValueInt64(),
	}
	if jwks := data.JWKS.ValueString(); jwks != "" {
		if !json.Valid([]byte(jwks)) {
			return body, fmt.Errorf("jwks is not valid JSON")
		}
		body.JWKS = json.RawMessage(jwks)
	}
	return body, nil
}

func (r *WorkloadIdentityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkloadIdentityProviderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_workload_identity_provider resource", &resp.Diagnostics) {
		return
	}

	body, err := r.request(&data)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("jwks"), "Invalid JWKS", err.Error())
		return
	}
	body.Name = data.Name.ValueString()
	body.Type = data.Type.ValueString()

	if err := r.data.API().Do(ctx, http.MethodPost, "/api/v2/workload-identity-providers", body, nil); err != nil {
		if apiErrorStatusCode(err) == http.StatusConflict {
			resp.Diagnostics.AddError(
				"Workload Identity Provider Already Exists",
				fmt.Sprintf("A workload identity provider named %q already exists in Dependency-Track. "+
					"Import it with `terraform import` to manage it with this resource.", data.Name.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workload identity provider, got error: %s", err))
		return
	}

	if _, err := r.read(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workload identity provider after create, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created a workload identity provider", map[string]any{"name": data.Name.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadIdentityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkloadIdentityProviderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_workload_identity_provider resource", &resp.Diagnostics) {
		return
	}

	found, err := r.read(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workload identity provider, got error: %s", err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadIdentityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkloadIdentityProviderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_workload_identity_provider resource", &resp.Diagnostics) {
		return
	}

	body, err := r.request(&data)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("jwks"), "Invalid JWKS", err.Error())
		return
	}

	err = r.data.API().Do(ctx, http.MethodPatch, workloadIdentityProviderPath(data.Name.ValueString()), body, nil)
	if err != nil && !isNotModified(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update workload identity provider, got error: %s", err))
		return
	}

	if _, err := r.read(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workload identity provider after update, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkloadIdentityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkloadIdentityProviderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(r.data, 5, 2, "The dependencytrack_workload_identity_provider resource", &resp.Diagnostics) {
		return
	}

	err := r.data.API().Do(ctx, http.MethodDelete, workloadIdentityProviderPath(data.Name.ValueString()), nil, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete workload identity provider, got error: %s", err))
		return
	}
}

func (r *WorkloadIdentityProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

// read refreshes data from the server and reports whether the provider exists.
//
// jwks_url is only read back when it is set in state: for OIDC providers
// without configured keys, the server reports the URL it resolved from the
// discovery document, which is not configuration. jwks is never returned, so
// its state value is kept.
func (r *WorkloadIdentityProviderResource) read(ctx context.Context, data *WorkloadIdentityProviderResourceModel) (bool, error) {
	var p workloadIdentityProviderResponse
	if err := r.data.API().Do(ctx, http.MethodGet, workloadIdentityProviderPath(data.Name.ValueString()), nil, &p); err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	data.ID = types.StringValue(p.Name)
	data.Name = types.StringValue(p.Name)
	data.Type = types.StringValue(p.Type)
	data.Issuer = types.StringValue(p.Issuer)
	data.Audience = types.StringValue(p.Audience)
	data.SessionLifetimeSeconds = types.Int64Value(p.SessionLifetimeSeconds)
	if !data.JWKSURL.IsNull() {
		if p.JWKSURL == "" {
			data.JWKSURL = types.StringNull()
		} else {
			data.JWKSURL = types.StringValue(p.JWKSURL)
		}
	}

	if len(p.JWKSKeyIDs) == 0 {
		data.JWKSKeyIDs = types.ListNull(types.StringType)
	} else {
		keyIDs, diags := types.ListValueFrom(ctx, types.StringType, p.JWKSKeyIDs)
		if diags.HasError() {
			return false, fmt.Errorf("convert jwks_key_ids: %v", diags)
		}
		data.JWKSKeyIDs = keyIDs
	}
	return true, nil
}
