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
var _ resource.Resource = &LDAPMappingResource{}
var _ resource.ResourceWithImportState = &LDAPMappingResource{}

func NewLDAPMappingResource() resource.Resource {
	return &LDAPMappingResource{}
}

// LDAPMappingResource defines the resource implementation.
type LDAPMappingResource struct {
	data *Data
}

// LDAPMappingResourceModel describes the resource data model.
type LDAPMappingResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Team types.String `tfsdk:"team"`
	DN   types.String `tfsdk:"dn"`
}

func (r *LDAPMappingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ldap_mapping"
}

func (r *LDAPMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Maps an LDAP group (by distinguished name) to a team in Dependency-Track. Users authenticated via LDAP inherit the permissions of every team their groups map to. The mapping is a plain database entity and can be managed without an LDAP server configured; Dependency-Track only validates that the team exists and that the mapping is not a duplicate.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the mapping, assigned by Dependency-Track",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the team. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dn": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The distinguished name of the LDAP group (e.g. `CN=Developers,OU=Groups,DC=example,DC=com`). Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *LDAPMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LDAPMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LDAPMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	mapping, err := r.data.Client.LDAP.AddMapping(ctx, dtrack.MappedLdapGroupRequest{
		Team:              teamUUID,
		DistinguishedName: data.DN.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create LDAP mapping, got error: %s", err))
		return
	}

	data.ID = types.StringValue(mapping.UUID.String())
	data.DN = types.StringValue(mapping.DistinguishedName)

	tflog.Trace(ctx, "created an LDAP mapping resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LDAPMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LDAPMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	mappingUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Mapping UUID", fmt.Sprintf("Unable to parse mapping UUID: %s", err))
		return
	}

	mappings, err := r.data.Client.LDAP.GetTeamMappings(ctx, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read LDAP mappings, got error: %s", err))
		return
	}

	var found *dtrack.MappedLdapGroup
	for i := range mappings {
		if mappings[i].UUID == mappingUUID {
			found = &mappings[i]
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(found.UUID.String())
	data.DN = types.StringValue(found.DistinguishedName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LDAPMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both team and dn have RequiresReplace, so this is never called.
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"LDAP mappings cannot be updated. Both team and dn changes require replacement.",
	)
}

func (r *LDAPMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LDAPMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	mappingUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Mapping UUID", fmt.Sprintf("Unable to parse mapping UUID: %s", err))
		return
	}

	err = r.data.Client.LDAP.RemoveMapping(ctx, mappingUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete LDAP mapping, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an LDAP mapping resource")
}

func (r *LDAPMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: team_uuid/mapping_uuid. The DN is not part of the ID
	// because distinguished names commonly contain '/'. The dn attribute is
	// populated by the subsequent Read.
	teamUUIDStr, mappingUUIDStr, err := parseCompositeID(req.ID, "team_uuid", "mapping_uuid")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID: %s", err),
		)
		return
	}

	teamUUID, err := uuid.Parse(teamUUIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Team UUID",
			fmt.Sprintf("Unable to parse team UUID from import ID: %s\nError: %s", teamUUIDStr, err),
		)
		return
	}

	mappingUUID, err := uuid.Parse(mappingUUIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Mapping UUID",
			fmt.Sprintf("Unable to parse mapping UUID from import ID: %s\nError: %s", mappingUUIDStr, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), mappingUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), teamUUID.String())...)
}
