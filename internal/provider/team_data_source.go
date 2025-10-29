package provider

import (
	"context"
	"errors"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &TeamDataSource{}

func NewTeamDataSource() datasource.DataSource {
	return &TeamDataSource{}
}

// TeamDataSource defines the data source implementation.
type TeamDataSource struct {
	client *dtrack.Client
}

// TeamDataSourceModel describes the data source data model.
type TeamDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *TeamDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *TeamDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a team from Dependency-Track by ID or name. Either `id` or `name` must be specified.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the team. Either `id` or `name` must be specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the team. Either `id` or `name` must be specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRoot("id")),
				},
			},
		},
	}
}

func (d *TeamDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = providerData.Client
}

func (d *TeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TeamDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that at least one identifier is provided
	hasID := !data.ID.IsNull() && data.ID.ValueString() != ""
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing Search Criteria",
			"Either 'id' or 'name' must be specified to look up a team.",
		)
		return
	}

	var team dtrack.Team
	var err error

	// If ID is provided, use it for direct lookup
	if hasID {
		var teamUUID uuid.UUID
		teamUUID, err = uuid.Parse(data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse team ID: %s", err))
			return
		}

		team, err = d.client.Team.Get(ctx, teamUUID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team by ID, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "read team data source by ID")
	} else {
		// Search by name - fetch all teams and filter
		searchName := data.Name.ValueString()
		tflog.Debug(ctx, fmt.Sprintf("searching for team by name: %s", searchName))

		// Use ForEach to iterate through all teams with automatic pagination
		var foundTeam *dtrack.Team
		errTeamFound := errors.New("team found") // Sentinel error to break out of ForEach

		err = dtrack.ForEach(func(po dtrack.PageOptions) (dtrack.Page[dtrack.Team], error) {
			return d.client.Team.GetAll(ctx, po)
		}, func(t dtrack.Team) error {
			if t.Name == searchName {
				foundTeam = &t
				return errTeamFound // Return sentinel error to stop iteration
			}
			return nil
		})

		// Check if iteration was stopped because team was found
		if err != nil && !errors.Is(err, errTeamFound) {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch teams, got error: %s", err))
			return
		}

		if foundTeam == nil {
			resp.Diagnostics.AddError(
				"Team Not Found",
				fmt.Sprintf("No team found with name: %s", searchName),
			)
			return
		}

		team = *foundTeam
		tflog.Trace(ctx, "read team data source by name")
	}

	// Set data from API response
	data.ID = types.StringValue(team.UUID.String())
	data.Name = types.StringValue(team.Name)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
