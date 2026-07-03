package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// projectPropertyTypes are the property types Dependency-Track accepts for a
// project property. ENCRYPTEDSTRING is only supported on v4 (v5 rejects it).
var projectPropertyTypes = []string{
	"BOOLEAN", "INTEGER", "NUMBER", "STRING", "ENCRYPTEDSTRING", "TIMESTAMP", "URL", "UUID",
}

// encryptedStringPlaceholder is the sentinel Dependency-Track returns instead
// of the real value for ENCRYPTEDSTRING properties.
const encryptedStringPlaceholder = "HiddenDecryptedPropertyPlaceholder"

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectPropertyResource{}
var _ resource.ResourceWithImportState = &ProjectPropertyResource{}

func NewProjectPropertyResource() resource.Resource {
	return &ProjectPropertyResource{}
}

// ProjectPropertyResource defines the resource implementation.
type ProjectPropertyResource struct {
	data *Data
}

// ProjectPropertyResourceModel describes the resource data model.
type ProjectPropertyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Project     types.String `tfsdk:"project"`
	Group       types.String `tfsdk:"group"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
}

func (r *ProjectPropertyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_property"
}

func (r *ProjectPropertyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a project property in Dependency-Track. Project properties are arbitrary key/value pairs attached to a project, grouped by a group name.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the project property in the format `project_uuid/group/name`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the project the property belongs to. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The group name of the property. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the property. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The value of the property.",
			},
			"type": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The type of the property (BOOLEAN, INTEGER, NUMBER, STRING, ENCRYPTEDSTRING, TIMESTAMP, URL, UUID). " +
					"The ENCRYPTEDSTRING type is only supported on Dependency-Track v4; v5 rejects it. " +
					"Changing this forces a new resource to be created.",
				Validators: []validator.String{
					stringvalidator.OneOf(projectPropertyTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The description of the property. Dependency-Track only sets the description when the property is created and ignores it on update, so changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ProjectPropertyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// warnOnEncryptedStringOnV5 appends a warning diagnostic when an ENCRYPTEDSTRING
// property is configured against a Dependency-Track v5 server, which rejects
// that type.
func warnOnEncryptedStringOnV5(diags *diag.Diagnostics, isV5 bool, propertyType string) {
	if isV5 && propertyType == "ENCRYPTEDSTRING" {
		diags.AddAttributeWarning(
			path.Root("type"),
			"ENCRYPTEDSTRING is not supported on Dependency-Track v5",
			"The property type ENCRYPTEDSTRING is only supported on Dependency-Track v4. "+
				"The server is running v5, which will reject this property.",
		)
	}
}

func (r *ProjectPropertyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectPropertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	warnOnEncryptedStringOnV5(&resp.Diagnostics, r.data.IsV5(), data.Type.ValueString())

	property := dtrack.ProjectProperty{
		Group:       data.Group.ValueString(),
		Name:        data.Name.ValueString(),
		Value:       data.Value.ValueString(),
		Type:        data.Type.ValueString(),
		Description: data.Description.ValueString(),
	}

	createdProperty, err := r.data.Client.ProjectProperty.Create(ctx, projectUUID, property)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project property, got error: %s", err))
		return
	}

	data.ID = types.StringValue(projectPropertyID(projectUUID, createdProperty.Group, createdProperty.Name))
	data.Type = types.StringValue(createdProperty.Type)
	data.Description = types.StringValue(createdProperty.Description)
	r.setValue(&data, createdProperty)

	tflog.Trace(ctx, "created a project property resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectPropertyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	property, found, err := r.findProperty(ctx, projectUUID, data.Group.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project property, got error: %s", err))
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(projectPropertyID(projectUUID, property.Group, property.Name))
	data.Type = types.StringValue(property.Type)
	data.Description = types.StringValue(property.Description)
	r.setValue(&data, property)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectPropertyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectPropertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	warnOnEncryptedStringOnV5(&resp.Diagnostics, r.data.IsV5(), data.Type.ValueString())

	// Only the value is updatable in place; group, name, type and description
	// all force replacement, so they are unchanged here.
	property := dtrack.ProjectProperty{
		Group:       data.Group.ValueString(),
		Name:        data.Name.ValueString(),
		Value:       data.Value.ValueString(),
		Type:        data.Type.ValueString(),
		Description: data.Description.ValueString(),
	}

	updatedProperty, err := r.data.Client.ProjectProperty.Update(ctx, projectUUID, property)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project property, got error: %s", err))
		return
	}

	data.ID = types.StringValue(projectPropertyID(projectUUID, updatedProperty.Group, updatedProperty.Name))
	data.Type = types.StringValue(updatedProperty.Type)
	data.Description = types.StringValue(updatedProperty.Description)
	r.setValue(&data, updatedProperty)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectPropertyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	err = r.data.Client.ProjectProperty.Delete(ctx, projectUUID, data.Group.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project property, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a project property resource")
}

func (r *ProjectPropertyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "project_uuid/group/name".
	projectUUIDStr, group, name, err := parseCompositeID3(req.ID, "project_uuid", "group", "name")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID: %s", err),
		)
		return
	}

	projectUUID, err := uuid.Parse(projectUUIDStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Project UUID",
			fmt.Sprintf("Unable to parse project UUID from import ID: %s\nError: %s", projectUUIDStr, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), projectUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group"), group)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

// setValue records the property value in state. For ENCRYPTEDSTRING properties
// the API returns a placeholder instead of the real value, so the configured
// value already in state is preserved (mirroring config_property).
func (r *ProjectPropertyResource) setValue(data *ProjectPropertyResourceModel, property dtrack.ProjectProperty) {
	if property.Type == "ENCRYPTEDSTRING" && property.Value == encryptedStringPlaceholder {
		return
	}
	data.Value = types.StringValue(property.Value)
}

// findProperty returns the project property matching group and name, paging
// through the project's full property list.
func (r *ProjectPropertyResource) findProperty(ctx context.Context, projectUUID uuid.UUID, group, name string) (dtrack.ProjectProperty, bool, error) {
	properties, err := fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.ProjectProperty], error) {
		return r.data.Client.ProjectProperty.GetAll(ctx, projectUUID, po)
	})
	if err != nil {
		return dtrack.ProjectProperty{}, false, err
	}

	for i := range properties {
		if properties[i].Group == group && properties[i].Name == name {
			return properties[i], true, nil
		}
	}

	return dtrack.ProjectProperty{}, false, nil
}

// projectPropertyID builds the composite resource ID.
func projectPropertyID(projectUUID uuid.UUID, group, name string) string {
	return fmt.Sprintf("%s/%s/%s", projectUUID.String(), group, name)
}
