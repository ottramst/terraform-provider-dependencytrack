package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &LicenseResource{}
var _ resource.ResourceWithImportState = &LicenseResource{}

func NewLicenseResource() resource.Resource {
	return &LicenseResource{}
}

// LicenseResource defines the resource implementation.
type LicenseResource struct {
	data *Data
}

// LicenseResourceModel describes the resource data model.
type LicenseResourceModel struct {
	ID          types.String `tfsdk:"id"`
	UUID        types.String `tfsdk:"uuid"`
	LicenseID   types.String `tfsdk:"license_id"`
	Name        types.String `tfsdk:"name"`
	Text        types.String `tfsdk:"text"`
	Template    types.String `tfsdk:"template"`
	Header      types.String `tfsdk:"header"`
	Comment     types.String `tfsdk:"comment"`
	OSIApproved types.Bool   `tfsdk:"osi_approved"`
	FSFLibre    types.Bool   `tfsdk:"fsf_libre"`
	SeeAlso     types.List   `tfsdk:"see_also"`
}

func (r *LicenseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license"
}

func (r *LicenseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom license in Dependency-Track. Dependency-Track has no endpoint to update a license, so any change forces the license to be replaced.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the license (same as `license_id`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the license, assigned by Dependency-Track",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"license_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The SPDX-style license ID (the natural key, e.g. `Acme-1.0`). Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the license. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"text": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The full text of the license. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The standard license template. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"header": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The standard license header. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A comment about the license. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"osi_approved": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the license is approved by the Open Source Initiative. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"fsf_libre": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the license is considered libre by the Free Software Foundation. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"see_also": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A list of URLs with more information about the license. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *LicenseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LicenseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LicenseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// see_also is Optional+Computed, so it is unknown when not configured; only
	// decode a concrete list.
	var seeAlso []string
	if !data.SeeAlso.IsNull() && !data.SeeAlso.IsUnknown() {
		resp.Diagnostics.Append(data.SeeAlso.ElementsAs(ctx, &seeAlso, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	license := dtrack.License{
		LicenseID:   data.LicenseID.ValueString(),
		Name:        data.Name.ValueString(),
		Text:        data.Text.ValueString(),
		Template:    data.Template.ValueString(),
		Header:      data.Header.ValueString(),
		Comment:     data.Comment.ValueString(),
		OSIApproved: data.OSIApproved.ValueBool(),
		FSFLibre:    data.FSFLibre.ValueBool(),
		SeeAlso:     seeAlso,
	}

	createdLicense, err := r.data.Client.License.Create(ctx, license)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create license, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(r.setState(ctx, &data, createdLicense)...)

	tflog.Trace(ctx, "created a license resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LicenseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LicenseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	license, err := r.data.Client.License.Get(ctx, data.LicenseID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read license, got error: %s", err))
		return
	}

	resp.Diagnostics.Append(r.setState(ctx, &data, license)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LicenseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every attribute has RequiresReplace and Dependency-Track has no license
	// update endpoint, so this is never called.
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"Licenses cannot be updated. Every attribute change requires replacement.",
	)
}

func (r *LicenseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LicenseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.data.Client.License.Delete(ctx, data.LicenseID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete license, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a license resource")
}

func (r *LicenseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the license ID (the natural key).
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("license_id"), req.ID)...)
}

// setState maps a license returned by the API into the resource model.
func (r *LicenseResource) setState(ctx context.Context, data *LicenseResourceModel, license dtrack.License) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(license.LicenseID)
	data.UUID = types.StringValue(license.UUID.String())
	data.LicenseID = types.StringValue(license.LicenseID)
	data.Name = types.StringValue(license.Name)
	data.Text = types.StringValue(license.Text)
	data.Template = types.StringValue(license.Template)
	data.Header = types.StringValue(license.Header)
	data.Comment = types.StringValue(license.Comment)
	data.OSIApproved = types.BoolValue(license.OSIApproved)
	data.FSFLibre = types.BoolValue(license.FSFLibre)

	seeAlso, d := types.ListValueFrom(ctx, types.StringType, license.SeeAlso)
	diags.Append(d...)
	data.SeeAlso = seeAlso

	return diags
}
