package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &TeamAPIKeysDataSource{}

func NewTeamAPIKeysDataSource() datasource.DataSource {
	return &TeamAPIKeysDataSource{}
}

// TeamAPIKeysDataSource defines the data source implementation.
type TeamAPIKeysDataSource struct {
	client *dtrack.Client
}

// TeamAPIKeysDataSourceModel describes the data source data model.
type TeamAPIKeysDataSourceModel struct {
	ID      types.String          `tfsdk:"id"`
	Team    types.String          `tfsdk:"team"`
	APIKeys []TeamAPIKeyDataModel `tfsdk:"api_keys"`
}

// TeamAPIKeyDataModel describes an individual API key.
type TeamAPIKeyDataModel struct {
	PublicID  types.String `tfsdk:"public_id"`
	Comment   types.String `tfsdk:"comment"`
	MaskedKey types.String `tfsdk:"masked_key"`
	Legacy    types.Bool   `tfsdk:"legacy"`
}

func (d *TeamAPIKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_api_keys"
}

func (d *TeamAPIKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all API keys for a Dependency-Track team. Note that the actual API key values are not returned, only metadata.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the team",
			},
			"team": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the team to retrieve API keys for",
			},
			"api_keys": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of API keys for the team",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"public_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The public ID of the API key",
						},
						"comment": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Comment or description for the API key",
						},
						"masked_key": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The masked version of the API key",
						},
						"legacy": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether this is a legacy API key",
						},
					},
				},
			},
		},
	}
}

func (d *TeamAPIKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TeamAPIKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TeamAPIKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Get all API keys for the team
	apiKeys, err := d.client.Team.GetAPIKeys(ctx, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team API keys, got error: %s", err))
		return
	}

	// Convert to Terraform model
	data.ID = types.StringValue(teamUUID.String())
	data.APIKeys = make([]TeamAPIKeyDataModel, 0, len(apiKeys))

	for _, key := range apiKeys {
		data.APIKeys = append(data.APIKeys, TeamAPIKeyDataModel{
			PublicID:  types.StringValue(key.PublicId),
			Comment:   types.StringValue(key.Comment),
			MaskedKey: types.StringValue(key.MaskedKey),
			Legacy:    types.BoolValue(key.Legacy),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
