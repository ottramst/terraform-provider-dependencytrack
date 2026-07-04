package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ExtensionConfigResource{}
var _ resource.ResourceWithImportState = &ExtensionConfigResource{}

func NewExtensionConfigResource() resource.Resource {
	return &ExtensionConfigResource{}
}

// ExtensionConfigResource defines the resource implementation.
type ExtensionConfigResource struct {
	data *Data
}

// ExtensionConfigResourceModel describes the resource data model.
type ExtensionConfigResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ExtensionPoint types.String `tfsdk:"extension_point"`
	Extension      types.String `tfsdk:"extension"`
	Config         types.String `tfsdk:"config"`
}

// extensionConfigEnvelope is the request and response body of the
// GET/PUT /api/v2/extension-points/{point}/extensions/{name}/config
// endpoints: the extension's configuration object wrapped in a "config" key.
type extensionConfigEnvelope struct {
	Config json.RawMessage `json:"config"`
}

func (r *ExtensionConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_extension_config"
}

func (r *ExtensionConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the configuration of a Dependency-Track extension. " +
			"Extensions are Dependency-Track v5's plugin mechanism for vulnerability analyzers " +
			"(extension point `vuln-analyzer`: `internal`, `oss-index`, `snyk`, `trivy`, `vuln-db`), " +
			"vulnerability data sources (`vuln-data-source`: `github`, `nvd`, `osv`), " +
			"notification publishers (`notification-publisher`: e.g. `email`, `jira`, `kafka`), " +
			"and package metadata resolvers (`package-metadata-resolver`). " +
			"Extension configurations always exist with server-side defaults and cannot be created or deleted, " +
			"only updated: this resource adopts the configuration into Terraform state and manages it. " +
			"When destroyed, the configuration is only removed from Terraform state and keeps its last value. " +
			"`config` is validated server-side against the extension's JSON schema " +
			"(retrievable at `/api/v2/extension-points/{point}/extensions/{name}/config-schema`). " +
			"**Never put clear-text credentials in `config`**: fields annotated `x-secret-ref` in the schema " +
			"(e.g. `apiToken`) take the *name* of a managed secret — see `dependencytrack_secret`. " +
			"Requires Dependency-Track v5 and the `SYSTEM_CONFIGURATION` permission.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the extension config in the format `extension_point/extension`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"extension_point": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the extension point (e.g. `vuln-analyzer`, `vuln-data-source`, `notification-publisher`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"extension": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the extension (e.g. `trivy`, `github`, `osv`, `email`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The full configuration of the extension as a JSON object (use `jsonencode()`). " +
					"Updates replace the whole configuration, so include every field you want set. " +
					"Semantically equal JSON (differing only in formatting or key order) is treated as unchanged",
			},
		},
	}
}

func (r *ExtensionConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// extensionConfigPath builds the /api/v2 config path for the given extension
// point and extension.
func extensionConfigPath(extensionPoint, extension string) string {
	return fmt.Sprintf("/api/v2/extension-points/%s/extensions/%s/config",
		url.PathEscape(extensionPoint), url.PathEscape(extension))
}

// putExtensionConfig validates and submits the desired configuration, mapping
// the API's error shapes onto actionable diagnostics. It returns false when a
// diagnostic was added.
func (r *ExtensionConfigResource) putExtensionConfig(ctx context.Context, data *ExtensionConfigResourceModel, diags *diag.Diagnostics) bool {
	configStr := data.Config.ValueString()
	if !json.Valid([]byte(configStr)) {
		diags.AddAttributeError(
			path.Root("config"),
			"Invalid JSON",
			"The config attribute must be a valid JSON object. Use jsonencode() to build it from HCL.",
		)
		return false
	}

	envelope := extensionConfigEnvelope{Config: json.RawMessage(configStr)}

	err := r.data.API().Do(ctx, http.MethodPut, extensionConfigPath(data.ExtensionPoint.ValueString(), data.Extension.ValueString()), envelope, nil)
	if err != nil && !isNotModified(err) {
		if isNotFound(err) {
			diags.AddError(
				"Extension Not Found",
				fmt.Sprintf("No extension %q exists under extension point %q. "+
					"List available extensions at GET /api/v2/extension-points/%s/extensions.",
					data.Extension.ValueString(), data.ExtensionPoint.ValueString(), data.ExtensionPoint.ValueString()),
			)
			return false
		}
		if apiErrorStatusCode(err) == http.StatusBadRequest {
			diags.AddAttributeError(
				path.Root("config"),
				"Extension Config Rejected",
				fmt.Sprintf("Dependency-Track rejected the configuration, likely because it does not match "+
					"the extension's config schema (GET /api/v2/extension-points/%s/extensions/%s/config-schema). "+
					"Error: %s",
					data.ExtensionPoint.ValueString(), data.Extension.ValueString(), err),
			)
			return false
		}
		diags.AddError("Client Error", fmt.Sprintf("Unable to update extension config, got error: %s", err))
		return false
	}

	return true
}

func (r *ExtensionConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireV5(r.data, "dependencytrack_extension_config", &resp.Diagnostics) {
		return
	}

	if !r.putExtensionConfig(ctx, &data, &resp.Diagnostics) {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.ExtensionPoint.ValueString(), data.Extension.ValueString()))

	tflog.Trace(ctx, "adopted an extension config", map[string]any{"id": data.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExtensionConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireV5(r.data, "dependencytrack_extension_config", &resp.Diagnostics) {
		return
	}

	var envelope extensionConfigEnvelope
	err := r.data.API().Do(ctx, http.MethodGet, extensionConfigPath(data.ExtensionPoint.ValueString(), data.Extension.ValueString()), nil, &envelope)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read extension config, got error: %s", err))
		return
	}

	serverConfig := string(envelope.Config)

	// Keep the state value when it is semantically the same JSON, so
	// formatting or key-order differences in the server's serialization do
	// not show up as drift. A real difference is surfaced in the server's
	// canonical form.
	if data.Config.IsNull() || !jsonStringsEquivalent(data.Config.ValueString(), serverConfig) {
		data.Config = types.StringValue(canonicalJSONString(serverConfig))
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.ExtensionPoint.ValueString(), data.Extension.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExtensionConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireV5(r.data, "dependencytrack_extension_config", &resp.Diagnostics) {
		return
	}

	if !r.putExtensionConfig(ctx, &data, &resp.Diagnostics) {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.ExtensionPoint.ValueString(), data.Extension.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExtensionConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExtensionConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extension configurations cannot be deleted from Dependency-Track; they
	// always exist with (at minimum) server-side defaults. Simply remove the
	// resource from Terraform state; the configuration keeps its last value.
}

func (r *ExtensionConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "extension_point/extension"
	extensionPoint, extension, err := parseCompositeID(req.ID, "extension_point", "extension")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("extension_point"), extensionPoint)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("extension"), extension)...)
}
