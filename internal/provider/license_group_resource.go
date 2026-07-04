package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &LicenseGroupResource{}
var _ resource.ResourceWithImportState = &LicenseGroupResource{}

func NewLicenseGroupResource() resource.Resource {
	return &LicenseGroupResource{}
}

// LicenseGroupResource defines the resource implementation.
type LicenseGroupResource struct {
	data *Data
}

// LicenseGroupResourceModel describes the resource data model.
type LicenseGroupResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	RiskWeight types.Int64  `tfsdk:"risk_weight"`
}

func (r *LicenseGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_group"
}

func (r *LicenseGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a license group in Dependency-Track.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the license group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the license group",
			},
			"risk_weight": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The risk weight of the license group. Dependency-Track manages this value and ignores any value supplied in a request (it reads back as 0), so it is exposed as read-only.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *LicenseGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LicenseGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LicenseGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// risk_weight is computed: Dependency-Track ignores any supplied value, so
	// it is intentionally not sent here.
	createdGroup, err := r.data.Client.LicenseGroup.Create(ctx, dtrack.LicenseGroup{
		Name: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create license group, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdGroup.UUID.String())
	data.Name = types.StringValue(createdGroup.Name)
	data.RiskWeight = types.Int64Value(int64(createdGroup.RiskWeight))

	tflog.Trace(ctx, "created a license group resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LicenseGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LicenseGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse license group UUID: %s", err))
		return
	}

	group, err := r.data.Client.LicenseGroup.Get(ctx, groupUUID)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read license group, got error: %s", err))
		return
	}

	data.ID = types.StringValue(group.UUID.String())
	data.Name = types.StringValue(group.Name)
	data.RiskWeight = types.Int64Value(int64(group.RiskWeight))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LicenseGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LicenseGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse license group UUID: %s", err))
		return
	}

	// risk_weight is computed and server-managed (see Create); not sent.
	updatedGroup, err := r.data.Client.LicenseGroup.Update(ctx, dtrack.LicenseGroup{
		UUID: groupUUID,
		Name: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update license group, got error: %s", err))
		return
	}

	data.ID = types.StringValue(updatedGroup.UUID.String())
	data.Name = types.StringValue(updatedGroup.Name)
	data.RiskWeight = types.Int64Value(int64(updatedGroup.RiskWeight))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LicenseGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LicenseGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse license group UUID: %s", err))
		return
	}

	err = r.data.Client.LicenseGroup.Delete(ctx, groupUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete license group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a license group resource")
}

func (r *LicenseGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupUUID, err := uuid.Parse(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse UUID. Expected a valid UUID, got: %s\nError: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), groupUUID.String())...)
}
