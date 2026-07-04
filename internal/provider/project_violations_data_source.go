package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ProjectViolationsDataSource{}

func NewProjectViolationsDataSource() datasource.DataSource {
	return &ProjectViolationsDataSource{}
}

// ProjectViolationsDataSource defines the data source implementation.
type ProjectViolationsDataSource struct {
	data *Data
}

// ProjectViolationsDataSourceModel describes the data source data model.
type ProjectViolationsDataSourceModel struct {
	ID         types.String            `tfsdk:"id"`
	Project    types.String            `tfsdk:"project"`
	Suppressed types.Bool              `tfsdk:"suppressed"`
	Violations []ProjectViolationModel `tfsdk:"violations"`
}

// ProjectViolationModel describes an individual policy violation.
type ProjectViolationModel struct {
	UUID                 types.String `tfsdk:"uuid"`
	Type                 types.String `tfsdk:"type"`
	Text                 types.String `tfsdk:"text"`
	PolicyName           types.String `tfsdk:"policy_name"`
	PolicyViolationState types.String `tfsdk:"policy_violation_state"`
	ComponentUUID        types.String `tfsdk:"component_uuid"`
	ComponentName        types.String `tfsdk:"component_name"`
	ComponentVersion     types.String `tfsdk:"component_version"`
}

func (d *ProjectViolationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_violations"
}

func (d *ProjectViolationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the policy violations of a project from Dependency-Track.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source result (the project UUID)",
			},
			"project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the project",
			},
			"suppressed": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to include suppressed violations in the result. Defaults to `false`.",
			},
			"violations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of policy violations",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the policy violation",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The type of the violation (`LICENSE`, `SECURITY` or `OPERATIONAL`)",
						},
						"text": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Additional text describing the violation, if available",
						},
						"policy_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the violated policy",
						},
						"policy_violation_state": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The violation state of the violated policy (`INFO`, `WARN` or `FAIL`)",
						},
						"component_uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the component that violates the policy",
						},
						"component_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the component that violates the policy",
						},
						"component_version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The version of the component that violates the policy",
						},
					},
				},
			},
		},
	}
}

func (d *ProjectViolationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectViolationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectViolationsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	suppressed := data.Suppressed.ValueBool()

	violations, err := fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.PolicyViolation], error) {
		return d.data.Client.PolicyViolation.GetAllForProject(ctx, projectUUID, suppressed, po)
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project policy violations, got error: %s", err))
		return
	}

	data.ID = types.StringValue(projectUUID.String())
	data.Violations = make([]ProjectViolationModel, 0, len(violations))
	for i := range violations {
		v := &violations[i]

		item := ProjectViolationModel{
			UUID:                 types.StringValue(v.UUID.String()),
			Type:                 types.StringValue(v.Type),
			Text:                 types.StringNull(),
			PolicyName:           types.StringNull(),
			PolicyViolationState: types.StringNull(),
			ComponentUUID:        types.StringValue(v.Component.UUID.String()),
			ComponentName:        types.StringValue(v.Component.Name),
			ComponentVersion:     types.StringValue(v.Component.Version),
		}

		if v.Text != "" {
			item.Text = types.StringValue(v.Text)
		}

		if v.PolicyCondition != nil && v.PolicyCondition.Policy != nil {
			item.PolicyName = types.StringValue(v.PolicyCondition.Policy.Name)
			item.PolicyViolationState = types.StringValue(string(v.PolicyCondition.Policy.ViolationState))
		}

		data.Violations = append(data.Violations, item)
	}

	tflog.Trace(ctx, "read a project violations data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
