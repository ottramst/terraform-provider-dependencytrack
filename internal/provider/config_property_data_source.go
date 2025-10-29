package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ConfigPropertyDataSource{}

func NewConfigPropertyDataSource() datasource.DataSource {
	return &ConfigPropertyDataSource{}
}

// ConfigPropertyDataSource defines the data source implementation.
type ConfigPropertyDataSource struct {
	client *dtrack.Client
}

// ConfigPropertyDataSourceModel describes the data source data model.
type ConfigPropertyDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	GroupName   types.String `tfsdk:"group_name"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
}

func (d *ConfigPropertyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_property"
}

func (d *ConfigPropertyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Dependency-Track configuration property.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the config property in the format `group_name/property_name`",
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The group name of the config property",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the config property",
			},
			"value": schema.StringAttribute{
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

func (d *ConfigPropertyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = data.Client
}

func (d *ConfigPropertyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConfigPropertyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	prop, err := d.client.Config.Get(ctx, data.GroupName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read config property, got error: %s", err))
		return
	}

	// If the property doesn't exist, return an error
	if prop.Name == "" {
		resp.Diagnostics.AddError(
			"Config Property Not Found",
			fmt.Sprintf("Config property with group_name=%q and name=%q does not exist.", data.GroupName.ValueString(), data.Name.ValueString()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", prop.GroupName, prop.Name))
	data.Value = types.StringValue(prop.Value)
	data.Type = types.StringValue(prop.Type)
	data.Description = types.StringValue(prop.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
