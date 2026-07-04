package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
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
var _ resource.Resource = &LicenseGroupLicenseResource{}
var _ resource.ResourceWithImportState = &LicenseGroupLicenseResource{}

func NewLicenseGroupLicenseResource() resource.Resource {
	return &LicenseGroupLicenseResource{}
}

// LicenseGroupLicenseResource defines the resource implementation.
type LicenseGroupLicenseResource struct {
	data *Data
}

// LicenseGroupLicenseResourceModel describes the resource data model.
type LicenseGroupLicenseResourceModel struct {
	ID           types.String `tfsdk:"id"`
	LicenseGroup types.String `tfsdk:"license_group"`
	License      types.String `tfsdk:"license"`
}

func (r *LicenseGroupLicenseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_group_license"
}

func (r *LicenseGroupLicenseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the membership of a license in a license group in Dependency-Track.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the license group membership in the format `license_group_uuid/license_uuid`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"license_group": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the license group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"license": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the license",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *LicenseGroupLicenseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LicenseGroupLicenseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LicenseGroupLicenseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, licenseUUID, ok := r.parseUUIDs(&data, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := r.data.Client.LicenseGroup.AddLicense(ctx, groupUUID, licenseUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add license to license group, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", groupUUID.String(), licenseUUID.String()))

	tflog.Trace(ctx, "created a license group license resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LicenseGroupLicenseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LicenseGroupLicenseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, licenseUUID, ok := r.parseUUIDs(&data, &resp.Diagnostics)
	if !ok {
		return
	}

	group, err := r.data.Client.LicenseGroup.Get(ctx, groupUUID)
	if err != nil {
		if isNotFound(err) {
			// License group doesn't exist anymore, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read license group, got error: %s", err))
		return
	}

	found := false
	for i := range group.Licenses {
		if group.Licenses[i].UUID == licenseUUID {
			found = true
			break
		}
	}

	if !found {
		// License is no longer part of the group, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", groupUUID.String(), licenseUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LicenseGroupLicenseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since both license_group and license have RequiresReplace, this should never be called
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"License group memberships cannot be updated. Both license_group and license changes require replacement.",
	)
}

func (r *LicenseGroupLicenseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LicenseGroupLicenseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, licenseUUID, ok := r.parseUUIDs(&data, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := r.data.Client.LicenseGroup.RemoveLicense(ctx, groupUUID, licenseUUID)
	if err != nil {
		// The server answers 304 Not Modified when the license is not (or no
		// longer) part of the group, and 404 when the group or license is
		// gone entirely; both mean there is nothing left to delete.
		if isNotFound(err) || apiErrorStatusCode(err) == http.StatusNotModified {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove license from license group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a license group license resource")
}

func (r *LicenseGroupLicenseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: license_group_uuid/license_uuid
	groupID, licenseID, err := parseCompositeID(req.ID, "license_group_uuid", "license_uuid")
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid License Group UUID",
			fmt.Sprintf("Unable to parse license group UUID from import ID: %s\nError: %s", groupID, err),
		)
		return
	}

	licenseUUID, err := uuid.Parse(licenseID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid License UUID",
			fmt.Sprintf("Unable to parse license UUID from import ID: %s\nError: %s", licenseID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("license_group"), groupUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("license"), licenseUUID.String())...)
}

// parseUUIDs parses the license_group and license attributes as UUIDs, adding
// diagnostics and returning ok=false when either is invalid.
func (r *LicenseGroupLicenseResource) parseUUIDs(data *LicenseGroupLicenseResourceModel, diags *diag.Diagnostics) (uuid.UUID, uuid.UUID, bool) {
	groupUUID, err := uuid.Parse(data.LicenseGroup.ValueString())
	if err != nil {
		diags.AddError("Invalid License Group UUID", fmt.Sprintf("Unable to parse license group UUID: %s", err))
		return uuid.UUID{}, uuid.UUID{}, false
	}

	licenseUUID, err := uuid.Parse(data.License.ValueString())
	if err != nil {
		diags.AddError("Invalid License UUID", fmt.Sprintf("Unable to parse license UUID: %s", err))
		return uuid.UUID{}, uuid.UUID{}, false
	}

	return groupUUID, licenseUUID, true
}
