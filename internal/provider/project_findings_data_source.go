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
var _ datasource.DataSource = &ProjectFindingsDataSource{}

func NewProjectFindingsDataSource() datasource.DataSource {
	return &ProjectFindingsDataSource{}
}

// ProjectFindingsDataSource defines the data source implementation.
type ProjectFindingsDataSource struct {
	data *Data
}

// ProjectFindingsDataSourceModel describes the data source data model.
type ProjectFindingsDataSourceModel struct {
	ID         types.String          `tfsdk:"id"`
	Project    types.String          `tfsdk:"project"`
	Suppressed types.Bool            `tfsdk:"suppressed"`
	Findings   []ProjectFindingModel `tfsdk:"findings"`
}

// ProjectFindingModel describes an individual finding.
type ProjectFindingModel struct {
	ComponentUUID     types.String `tfsdk:"component_uuid"`
	ComponentName     types.String `tfsdk:"component_name"`
	ComponentVersion  types.String `tfsdk:"component_version"`
	VulnerabilityUUID types.String `tfsdk:"vulnerability_uuid"`
	VulnID            types.String `tfsdk:"vuln_id"`
	Source            types.String `tfsdk:"source"`
	Severity          types.String `tfsdk:"severity"`
	CWEs              types.List   `tfsdk:"cwes"`
	AnalysisState     types.String `tfsdk:"analysis_state"`
	IsSuppressed      types.Bool   `tfsdk:"is_suppressed"`
	AnalyzerIdentity  types.String `tfsdk:"analyzer_identity"`
	AttributedOn      types.Int64  `tfsdk:"attributed_on"`
}

func (d *ProjectFindingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_findings"
}

func (d *ProjectFindingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the vulnerability findings of a project from Dependency-Track.",

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
				MarkdownDescription: "Whether to include suppressed findings in the result. Defaults to `false`.",
			},
			"findings": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of findings",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"component_uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the affected component",
						},
						"component_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the affected component",
						},
						"component_version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The version of the affected component",
						},
						"vulnerability_uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the vulnerability",
						},
						"vuln_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The identifier of the vulnerability (e.g. a CVE ID)",
						},
						"source": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The source of the vulnerability (e.g. `NVD`, `GITHUB`, `INTERNAL`)",
						},
						"severity": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The severity of the vulnerability",
						},
						"cwes": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.Int64Type,
							MarkdownDescription: "The CWE IDs associated with the vulnerability",
						},
						"analysis_state": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The audit analysis state of the finding, if analyzed",
						},
						"is_suppressed": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the finding is suppressed",
						},
						"analyzer_identity": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The analyzer that attributed the finding",
						},
						"attributed_on": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Timestamp (epoch milliseconds) when the finding was attributed",
						},
					},
				},
			},
		},
	}
}

func (d *ProjectFindingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectFindingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectFindingsDataSourceModel

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

	// Both Dependency-Track v4 and v5 serialize a finding's vulnerability.cwes
	// as a list of {cweId, name} objects (verified live against 4.14.2 and
	// 5.0.2), which client-go's Finding struct decodes on either version.
	findings, err := fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.Finding], error) {
		return d.data.Client.Finding.GetAll(ctx, projectUUID, suppressed, po)
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project findings, got error: %s", err))
		return
	}

	data.ID = types.StringValue(projectUUID.String())
	data.Findings = make([]ProjectFindingModel, 0, len(findings))
	for i := range findings {
		f := &findings[i]

		cweIDs := make([]types.Int64, 0, len(f.Vulnerability.CWEs))
		for _, cwe := range f.Vulnerability.CWEs {
			cweIDs = append(cweIDs, types.Int64Value(int64(cwe.ID)))
		}
		cwes, diags := types.ListValueFrom(ctx, types.Int64Type, cweIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		item := ProjectFindingModel{
			ComponentUUID:     types.StringValue(f.Component.UUID.String()),
			ComponentName:     types.StringValue(f.Component.Name),
			ComponentVersion:  types.StringValue(f.Component.Version),
			VulnerabilityUUID: types.StringValue(f.Vulnerability.UUID.String()),
			VulnID:            types.StringValue(f.Vulnerability.VulnID),
			Source:            types.StringValue(f.Vulnerability.Source),
			Severity:          types.StringValue(f.Vulnerability.Severity),
			CWEs:              cwes,
			AnalysisState:     types.StringNull(),
			IsSuppressed:      types.BoolValue(f.Analysis.Suppressed),
			AnalyzerIdentity:  types.StringValue(f.Attribution.AnalyzerIdentity),
			AttributedOn:      types.Int64Value(int64(f.Attribution.AttributedOn)),
		}

		if f.Analysis.State != "" {
			item.AnalysisState = types.StringValue(f.Analysis.State)
		}

		data.Findings = append(data.Findings, item)
	}

	tflog.Trace(ctx, "read a project findings data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
