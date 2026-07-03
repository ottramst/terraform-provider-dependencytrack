package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &TagsDataSource{}

func NewTagsDataSource() datasource.DataSource {
	return &TagsDataSource{}
}

// TagsDataSource defines the data source implementation.
type TagsDataSource struct {
	data *Data
}

// TagsDataSourceModel describes the data source data model.
type TagsDataSourceModel struct {
	ID   types.String   `tfsdk:"id"`
	Tags []TagDataModel `tfsdk:"tags"`
}

// TagDataModel describes an individual tag along with its usage counts.
type TagDataModel struct {
	Name                  types.String `tfsdk:"name"`
	ProjectCount          types.Int64  `tfsdk:"project_count"`
	PolicyCount           types.Int64  `tfsdk:"policy_count"`
	NotificationRuleCount types.Int64  `tfsdk:"notification_rule_count"`
}

func (d *TagsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *TagsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all tags from Dependency-Track along with their usage counts.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source result (always `tags`).",
			},
			"tags": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of tags",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the tag",
						},
						"project_count": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The number of projects tagged with this tag",
						},
						"policy_count": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The number of policies tagged with this tag",
						},
						"notification_rule_count": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The number of notification rules tagged with this tag",
						},
					},
				},
			},
		},
	}
}

func (d *TagsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TagsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tags, err := fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.TagListResponseItem], error) {
		return d.data.Client.Tag.GetAll(ctx, po, dtrack.SortOptions{})
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tags, got error: %s", err))
		return
	}

	data.ID = types.StringValue("tags")
	data.Tags = make([]TagDataModel, 0, len(tags))
	for _, tag := range tags {
		data.Tags = append(data.Tags, TagDataModel{
			Name:                  types.StringValue(tag.Name),
			ProjectCount:          types.Int64Value(tag.ProjectCount),
			PolicyCount:           types.Int64Value(tag.PolicyCount),
			NotificationRuleCount: types.Int64Value(tag.NotificationRuleCount),
		})
	}

	tflog.Trace(ctx, "read a tags data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
