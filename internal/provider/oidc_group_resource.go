package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &OIDCGroupResource{}
var _ resource.ResourceWithImportState = &OIDCGroupResource{}

func NewOIDCGroupResource() resource.Resource {
	return &OIDCGroupResource{}
}

// OIDCGroupResource defines the resource implementation.
type OIDCGroupResource struct {
	data *Data
}

// OIDCGroupResourceModel describes the resource data model.
type OIDCGroupResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (r *OIDCGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_group"
}

func (r *OIDCGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an OpenID Connect (OIDC) group in Dependency-Track. OIDC groups are plain database entities and can be managed without an identity provider configured.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the OIDC group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the OIDC group",
			},
		},
	}
}

func (r *OIDCGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OIDCGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OIDCGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdGroup, err := r.data.Client.OIDC.CreateGroup(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create OIDC group, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdGroup.UUID.String())
	data.Name = types.StringValue(createdGroup.Name)

	tflog.Trace(ctx, "created an OIDC group resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OIDCGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OIDCGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse OIDC group UUID: %s", err))
		return
	}

	// There is no get-by-uuid endpoint for OIDC groups, so list them all and
	// match by UUID.
	group, found, err := r.findGroup(ctx, groupUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read OIDC group, got error: %s", err))
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(group.UUID.String())
	data.Name = types.StringValue(group.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OIDCGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OIDCGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse OIDC group UUID: %s", err))
		return
	}

	updatedGroup, err := r.data.Client.OIDC.UpdateGroup(ctx, dtrack.OIDCGroup{
		UUID: groupUUID,
		Name: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update OIDC group, got error: %s", err))
		return
	}

	data.ID = types.StringValue(updatedGroup.UUID.String())
	data.Name = types.StringValue(updatedGroup.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OIDCGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OIDCGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	groupUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse OIDC group UUID: %s", err))
		return
	}

	err = r.data.Client.OIDC.DeleteGroup(ctx, groupUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete OIDC group, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an OIDC group resource")
}

func (r *OIDCGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

// findGroup lists all OIDC groups and returns the one matching groupUUID.
//
// GetAllGroups issues a bare GET on /api/v1/oidc/group with no pagination
// parameters. This is safe: unlike /api/v1/licenseGroup, the OIDC group
// endpoint is not paginated. Empirically verified against Dependency-Track
// v5.0.2 by seeding 120 groups: a bare GET returned all 120 in a single
// response, and pageSize/pageNumber query parameters were ignored (page 1 and
// page 2 returned the identical full set, and no X-Total-Count header was
// emitted). Routing this through apiGetAllPages would in fact break it — with
// no X-Total-Count and every page returning the full (>=100-item) set, it
// would fetch identical pages up to the safety cap and error out.
func (r *OIDCGroupResource) findGroup(ctx context.Context, groupUUID uuid.UUID) (dtrack.OIDCGroup, bool, error) {
	groups, err := r.data.Client.OIDC.GetAllGroups(ctx)
	if err != nil {
		return dtrack.OIDCGroup{}, false, err
	}

	for i := range groups {
		if groups[i].UUID == groupUUID {
			return groups[i], true, nil
		}
	}

	return dtrack.OIDCGroup{}, false, nil
}
