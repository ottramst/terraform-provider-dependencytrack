package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RepositoriesDataSource{}

func NewRepositoriesDataSource() datasource.DataSource {
	return &RepositoriesDataSource{}
}

// RepositoriesDataSource defines the data source implementation.
type RepositoriesDataSource struct {
	data *Data
}

// RepositoriesDataSourceModel describes the data source data model.
type RepositoriesDataSourceModel struct {
	ID           types.String          `tfsdk:"id"`
	Type         types.String          `tfsdk:"type"`
	Repositories []RepositoryDataModel `tfsdk:"repositories"`
}

// RepositoryDataModel describes an individual repository. Passwords are never
// exposed.
type RepositoryDataModel struct {
	ID                     types.String `tfsdk:"id"`
	Type                   types.String `tfsdk:"type"`
	Identifier             types.String `tfsdk:"identifier"`
	URL                    types.String `tfsdk:"url"`
	ResolutionOrder        types.Int64  `tfsdk:"resolution_order"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	Internal               types.Bool   `tfsdk:"internal"`
	AuthenticationRequired types.Bool   `tfsdk:"authentication_required"`
}

func (d *RepositoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repositories"
}

func (d *RepositoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves component repositories from Dependency-Track, optionally filtered by type. API key values (passwords) are never returned.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source result: the type filter when set, otherwise `all`.",
			},
			"type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Only return repositories of this type. One of: CARGO, COMPOSER, CPAN, GEM, GITHUB, GO_MODULES, HEX, MAVEN, NPM, NUGET, PYPI. When omitted, all repositories are returned.",
				Validators: []validator.String{
					stringvalidator.OneOf(repositoryTypes...),
				},
			},
			"repositories": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of repositories",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the repository",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The type of the repository",
						},
						"identifier": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unique identifier of the repository",
						},
						"url": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The URL of the repository",
						},
						"resolution_order": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The resolution order of the repository",
						},
						"enabled": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the repository is enabled",
						},
						"internal": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the repository is internal",
						},
						"authentication_required": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether authentication is required to access the repository",
						},
					},
				},
			},
		},
	}
}

func (d *RepositoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.data = data
}

func (d *RepositoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RepositoriesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	hasType := !data.Type.IsNull() && data.Type.ValueString() != ""

	var repos []dtrack.Repository
	var err error

	if hasType {
		repoType := dtrack.RepositoryType(data.Type.ValueString())
		repos, err = fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.Repository], error) {
			return d.data.Client.Repository.GetByType(ctx, repoType, po)
		})
		data.ID = data.Type
	} else {
		repos, err = fetchAllPages(ctx, d.data.Client.Repository.GetAll)
		data.ID = types.StringValue("all")
	}

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read repositories, got error: %s", err))
		return
	}

	data.Repositories = make([]RepositoryDataModel, 0, len(repos))
	for _, repo := range repos {
		data.Repositories = append(data.Repositories, RepositoryDataModel{
			ID:                     types.StringValue(repo.UUID.String()),
			Type:                   types.StringValue(string(repo.Type)),
			Identifier:             types.StringValue(repo.Identifier),
			URL:                    types.StringValue(repo.Url),
			ResolutionOrder:        types.Int64Value(int64(repo.ResolutionOrder)),
			Enabled:                types.BoolValue(repo.Enabled),
			Internal:               types.BoolValue(repo.Internal),
			AuthenticationRequired: types.BoolValue(repo.AuthenticationRequired),
		})
	}

	tflog.Trace(ctx, "read a repositories data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
