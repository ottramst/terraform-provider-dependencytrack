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
var _ datasource.DataSource = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

// ProjectDataSource defines the data source implementation.
type ProjectDataSource struct {
	client *dtrack.Client
}

// ProjectDataSourceModel describes the data source data model.
type ProjectDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Group       types.String `tfsdk:"group"`
	Publisher   types.String `tfsdk:"publisher"`
	Author      types.String `tfsdk:"author"`
	Classifier  types.String `tfsdk:"classifier"`
	Active      types.Bool   `tfsdk:"active"`
	CPE         types.String `tfsdk:"cpe"`
	PURL        types.String `tfsdk:"purl"`
	SWIDTagID   types.String `tfsdk:"swid_tag_id"`
	ParentUUID  types.String `tfsdk:"parent_uuid"`
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Dependency-Track project. You can look up a project by ID (UUID), or by name and version.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The UUID of the project. Either `id` or both `name` and `version` must be specified.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the project. Required when `id` is not specified.",
			},
			"version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The version of the project. Required when `id` is not specified.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the project",
			},
			"group": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The group of the project",
			},
			"publisher": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The publisher of the project",
			},
			"author": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The author of the project",
			},
			"classifier": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The classifier of the project",
			},
			"active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the project is active",
			},
			"cpe": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Common Platform Enumeration (CPE) of the project",
			},
			"purl": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Package URL (PURL) of the project",
			},
			"swid_tag_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The SWID tag ID of the project",
			},
			"parent_uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the parent project",
			},
		},
	}
}

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var project dtrack.Project
	var err error

	// Validate input: either ID or (name and version) must be specified
	hasID := !data.ID.IsNull() && !data.ID.IsUnknown()
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown()
	hasVersion := !data.Version.IsNull() && !data.Version.IsUnknown()

	if hasID {
		// Lookup by ID (UUID)
		projectUUID, parseErr := uuid.Parse(data.ID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Unable to parse project ID: %s", parseErr))
			return
		}

		project, err = d.client.Project.Get(ctx, projectUUID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
			return
		}
	} else if hasName && hasVersion {
		// Lookup by name and version
		project, err = d.client.Project.Lookup(ctx, data.Name.ValueString(), data.Version.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to lookup project, got error: %s", err))
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Either 'id' or both 'name' and 'version' must be specified.",
		)
		return
	}

	// Check if project was found
	if project.UUID == uuid.Nil {
		resp.Diagnostics.AddError(
			"Project Not Found",
			"The specified project does not exist.",
		)
		return
	}

	data.ID = types.StringValue(project.UUID.String())
	data.Name = types.StringValue(project.Name)
	data.Version = types.StringValue(project.Version)
	data.Description = types.StringValue(project.Description)
	data.Group = types.StringValue(project.Group)
	data.Publisher = types.StringValue(project.Publisher)
	data.Author = types.StringValue(project.Author)
	data.Classifier = types.StringValue(project.Classifier)
	data.Active = types.BoolValue(project.Active)
	data.CPE = types.StringValue(project.CPE)
	data.PURL = types.StringValue(project.PURL)
	data.SWIDTagID = types.StringValue(project.SWIDTagID)

	if project.ParentRef != nil {
		data.ParentUUID = types.StringValue(project.ParentRef.UUID.String())
	} else {
		data.ParentUUID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
