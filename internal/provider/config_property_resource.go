package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ConfigPropertyResource{}
var _ resource.ResourceWithImportState = &ConfigPropertyResource{}

func NewConfigPropertyResource() resource.Resource {
	return &ConfigPropertyResource{}
}

// ConfigPropertyResource defines the resource implementation.
type ConfigPropertyResource struct {
	client *dtrack.Client
}

// ConfigPropertyResourceModel describes the resource data model.
type ConfigPropertyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	GroupName   types.String `tfsdk:"group_name"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
}

func (r *ConfigPropertyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_property"
}

func (r *ConfigPropertyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Dependency-Track configuration property. " +
			"Configuration properties are predefined in Dependency-Track and cannot be created or deleted, only updated. " +
			"This resource adopts an existing configuration property into Terraform state and manages its value. " +
			"When destroyed, the property is only removed from Terraform state and remains in Dependency-Track with its current value.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the config property in the format `group_name/property_name`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The group name of the config property",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the config property",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The value of the config property",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The type of the config property (BOOLEAN, INTEGER, NUMBER, STRING, ENCRYPTEDSTRING, TIMESTAMP, URL, UUID)",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the config property",
			},
		},
	}
}

func (r *ConfigPropertyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = data.Client
}

func (r *ConfigPropertyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Config properties cannot be created, only adopted
	// First, check if the property exists in Dependency-Track
	existingProp, err := r.client.Config.Get(ctx, data.GroupName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read config property, got error: %s", err))
		return
	}

	// If the property doesn't exist or has no name, it means it wasn't found
	if existingProp.Name == "" {
		resp.Diagnostics.AddError(
			"Config Property Not Found",
			fmt.Sprintf("Config property with group_name=%q and name=%q does not exist. Config properties must be predefined in Dependency-Track.", data.GroupName.ValueString(), data.Name.ValueString()),
		)
		return
	}

	// Property exists - adopt it into state and update with desired value if specified
	var updatedProp dtrack.ConfigProperty
	if !data.Value.IsNull() && !data.Value.IsUnknown() {
		// User specified a value - update the property
		updateProp := dtrack.ConfigProperty{
			GroupName: data.GroupName.ValueString(),
			Name:      data.Name.ValueString(),
			Type:      existingProp.Type,
			Value:     data.Value.ValueString(),
		}

		updatedProp, err = r.client.Config.Update(ctx, updateProp)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update config property, got error: %s", err))
			return
		}
	} else {
		// No value specified - just adopt current state
		updatedProp = existingProp
	}

	// Set the ID and all attributes in state
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", updatedProp.GroupName, updatedProp.Name))
	data.Value = types.StringValue(updatedProp.Value)
	data.Type = types.StringValue(updatedProp.Type)
	data.Description = types.StringValue(updatedProp.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConfigPropertyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	prop, err := r.client.Config.Get(ctx, data.GroupName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read config property, got error: %s", err))
		return
	}

	// If the property doesn't exist, remove from state
	if prop.Name == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Value = types.StringValue(prop.Value)
	data.Type = types.StringValue(prop.Type)
	data.Description = types.StringValue(prop.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConfigPropertyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current property to retrieve its type
	existingProp, err := r.client.Config.Get(ctx, data.GroupName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read config property, got error: %s", err))
		return
	}

	updateProp := dtrack.ConfigProperty{
		GroupName: data.GroupName.ValueString(),
		Name:      data.Name.ValueString(),
		Type:      existingProp.Type,
		Value:     data.Value.ValueString(),
	}

	updatedProp, err := r.client.Config.Update(ctx, updateProp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update config property, got error: %s", err))
		return
	}

	data.Value = types.StringValue(updatedProp.Value)
	data.Type = types.StringValue(updatedProp.Type)
	data.Description = types.StringValue(updatedProp.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConfigPropertyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConfigPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Config properties cannot be deleted from Dependency-Track
	// They are predefined and permanent
	// Simply remove from Terraform state without making any API calls
	// The property will remain in Dependency-Track with its current value
}

func (r *ConfigPropertyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "group_name/property_name"
	// Parse the ID
	groupName, propertyName, err := parseConfigPropertyID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID. Expected format: 'group_name/property_name', got: %s\nError: %s", req.ID, err),
		)
		return
	}

	// Set the ID and individual attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), groupName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), propertyName)...)
}

// parseConfigPropertyID parses a config property ID in the format "group_name/property_name".
func parseConfigPropertyID(id string) (groupName, propertyName string, err error) {
	// Find the first slash to split group_name and property_name
	var slashIndex = -1
	for i, c := range id {
		if c == '/' {
			slashIndex = i
			break
		}
	}

	if slashIndex == -1 {
		return "", "", fmt.Errorf("ID must contain a '/' separator")
	}

	groupName = id[:slashIndex]
	propertyName = id[slashIndex+1:]

	if groupName == "" {
		return "", "", fmt.Errorf("group_name cannot be empty")
	}
	if propertyName == "" {
		return "", "", fmt.Errorf("property_name cannot be empty")
	}

	return groupName, propertyName, nil
}
