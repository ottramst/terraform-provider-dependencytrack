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
var _ datasource.DataSource = &ProjectMetricsDataSource{}

func NewProjectMetricsDataSource() datasource.DataSource {
	return &ProjectMetricsDataSource{}
}

// ProjectMetricsDataSource defines the data source implementation.
type ProjectMetricsDataSource struct {
	data *Data
}

// ProjectMetricsDataSourceModel describes the data source data model.
type ProjectMetricsDataSourceModel struct {
	ID                                   types.String  `tfsdk:"id"`
	Project                              types.String  `tfsdk:"project"`
	FirstOccurrence                      types.Int64   `tfsdk:"first_occurrence"`
	LastOccurrence                       types.Int64   `tfsdk:"last_occurrence"`
	InheritedRiskScore                   types.Float64 `tfsdk:"inherited_risk_score"`
	Vulnerabilities                      types.Int64   `tfsdk:"vulnerabilities"`
	Components                           types.Int64   `tfsdk:"components"`
	VulnerableComponents                 types.Int64   `tfsdk:"vulnerable_components"`
	Suppressed                           types.Int64   `tfsdk:"suppressed"`
	Critical                             types.Int64   `tfsdk:"critical"`
	High                                 types.Int64   `tfsdk:"high"`
	Medium                               types.Int64   `tfsdk:"medium"`
	Low                                  types.Int64   `tfsdk:"low"`
	Unassigned                           types.Int64   `tfsdk:"unassigned"`
	FindingsTotal                        types.Int64   `tfsdk:"findings_total"`
	FindingsAudited                      types.Int64   `tfsdk:"findings_audited"`
	FindingsUnaudited                    types.Int64   `tfsdk:"findings_unaudited"`
	PolicyViolationsTotal                types.Int64   `tfsdk:"policy_violations_total"`
	PolicyViolationsFail                 types.Int64   `tfsdk:"policy_violations_fail"`
	PolicyViolationsWarn                 types.Int64   `tfsdk:"policy_violations_warn"`
	PolicyViolationsInfo                 types.Int64   `tfsdk:"policy_violations_info"`
	PolicyViolationsAudited              types.Int64   `tfsdk:"policy_violations_audited"`
	PolicyViolationsUnaudited            types.Int64   `tfsdk:"policy_violations_unaudited"`
	PolicyViolationsSecurityTotal        types.Int64   `tfsdk:"policy_violations_security_total"`
	PolicyViolationsSecurityAudited      types.Int64   `tfsdk:"policy_violations_security_audited"`
	PolicyViolationsSecurityUnaudited    types.Int64   `tfsdk:"policy_violations_security_unaudited"`
	PolicyViolationsLicenseTotal         types.Int64   `tfsdk:"policy_violations_license_total"`
	PolicyViolationsLicenseAudited       types.Int64   `tfsdk:"policy_violations_license_audited"`
	PolicyViolationsLicenseUnaudited     types.Int64   `tfsdk:"policy_violations_license_unaudited"`
	PolicyViolationsOperationalTotal     types.Int64   `tfsdk:"policy_violations_operational_total"`
	PolicyViolationsOperationalAudited   types.Int64   `tfsdk:"policy_violations_operational_audited"`
	PolicyViolationsOperationalUnaudited types.Int64   `tfsdk:"policy_violations_operational_unaudited"`
}

func (d *ProjectMetricsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_metrics"
}

func (d *ProjectMetricsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the most recent metrics snapshot for a project from Dependency-Track. " +
			"If the server has never computed metrics for the project (typical for a freshly created project on v4), " +
			"this data source triggers a metrics refresh, waits briefly for it to complete, and reports zero " +
			"values if metrics are still unavailable.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source result (the project UUID)",
			},
			"project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the project",
			},
			"first_occurrence": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Timestamp (epoch milliseconds) when this metrics snapshot was first recorded",
			},
			"last_occurrence": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Timestamp (epoch milliseconds) when this metrics snapshot was last confirmed",
			},
			"inherited_risk_score": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "The inherited risk score of the project",
			},
			"vulnerabilities": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of vulnerabilities in the project",
			},
			"components": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of components in the project",
			},
			"vulnerable_components": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of components with at least one vulnerability",
			},
			"suppressed": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of suppressed findings",
			},
			"critical": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of critical severity vulnerabilities",
			},
			"high": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of high severity vulnerabilities",
			},
			"medium": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of medium severity vulnerabilities",
			},
			"low": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of low severity vulnerabilities",
			},
			"unassigned": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of vulnerabilities with unassigned severity",
			},
			"findings_total": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of findings",
			},
			"findings_audited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of audited findings",
			},
			"findings_unaudited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of unaudited findings",
			},
			"policy_violations_total": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of policy violations",
			},
			"policy_violations_fail": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of policy violations with a FAIL violation state",
			},
			"policy_violations_warn": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of policy violations with a WARN violation state",
			},
			"policy_violations_info": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of policy violations with an INFO violation state",
			},
			"policy_violations_audited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of audited policy violations",
			},
			"policy_violations_unaudited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of unaudited policy violations",
			},
			"policy_violations_security_total": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of security policy violations",
			},
			"policy_violations_security_audited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of audited security policy violations",
			},
			"policy_violations_security_unaudited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of unaudited security policy violations",
			},
			"policy_violations_license_total": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of license policy violations",
			},
			"policy_violations_license_audited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of audited license policy violations",
			},
			"policy_violations_license_unaudited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of unaudited license policy violations",
			},
			"policy_violations_operational_total": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of operational policy violations",
			},
			"policy_violations_operational_audited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of audited operational policy violations",
			},
			"policy_violations_operational_unaudited": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of unaudited operational policy violations",
			},
		},
	}
}

func (d *ProjectMetricsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ProjectMetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectMetricsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	metrics, _, err := currentMetricsWithRefresh(ctx,
		func(ctx context.Context) (dtrack.ProjectMetrics, error) {
			return d.data.Client.Metrics.LatestProjectMetrics(ctx, projectUUID)
		},
		func(ctx context.Context) error {
			return d.data.Client.Metrics.RefreshProjectMetrics(ctx, projectUUID)
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project metrics, got error: %s", err))
		return
	}

	data.ID = types.StringValue(projectUUID.String())
	data.FirstOccurrence = types.Int64Value(int64(metrics.FirstOccurrence))
	data.LastOccurrence = types.Int64Value(int64(metrics.LastOccurrence))
	data.InheritedRiskScore = types.Float64Value(metrics.InheritedRiskScore)
	data.Vulnerabilities = types.Int64Value(int64(metrics.Vulnerabilities))
	data.Components = types.Int64Value(int64(metrics.Components))
	data.VulnerableComponents = types.Int64Value(int64(metrics.VulnerableComponents))
	data.Suppressed = types.Int64Value(int64(metrics.Suppressed))
	data.Critical = types.Int64Value(int64(metrics.Critical))
	data.High = types.Int64Value(int64(metrics.High))
	data.Medium = types.Int64Value(int64(metrics.Medium))
	data.Low = types.Int64Value(int64(metrics.Low))
	data.Unassigned = types.Int64Value(int64(metrics.Unassigned))
	data.FindingsTotal = types.Int64Value(int64(metrics.FindingsTotal))
	data.FindingsAudited = types.Int64Value(int64(metrics.FindingsAudited))
	data.FindingsUnaudited = types.Int64Value(int64(metrics.FindingsUnaudited))
	data.PolicyViolationsTotal = types.Int64Value(int64(metrics.PolicyViolationsTotal))
	data.PolicyViolationsFail = types.Int64Value(int64(metrics.PolicyViolationsFail))
	data.PolicyViolationsWarn = types.Int64Value(int64(metrics.PolicyViolationsWarn))
	data.PolicyViolationsInfo = types.Int64Value(int64(metrics.PolicyViolationsInfo))
	data.PolicyViolationsAudited = types.Int64Value(int64(metrics.PolicyViolationsAudited))
	data.PolicyViolationsUnaudited = types.Int64Value(int64(metrics.PolicyViolationsUnaudited))
	data.PolicyViolationsSecurityTotal = types.Int64Value(int64(metrics.PolicyViolationsSecurityTotal))
	data.PolicyViolationsSecurityAudited = types.Int64Value(int64(metrics.PolicyViolationsSecurityAudited))
	data.PolicyViolationsSecurityUnaudited = types.Int64Value(int64(metrics.PolicyViolationsSecurityUnaudited))
	data.PolicyViolationsLicenseTotal = types.Int64Value(int64(metrics.PolicyViolationsLicenseTotal))
	data.PolicyViolationsLicenseAudited = types.Int64Value(int64(metrics.PolicyViolationsLicenseAudited))
	data.PolicyViolationsLicenseUnaudited = types.Int64Value(int64(metrics.PolicyViolationsLicenseUnaudited))
	data.PolicyViolationsOperationalTotal = types.Int64Value(int64(metrics.PolicyViolationsOperationalTotal))
	data.PolicyViolationsOperationalAudited = types.Int64Value(int64(metrics.PolicyViolationsOperationalAudited))
	data.PolicyViolationsOperationalUnaudited = types.Int64Value(int64(metrics.PolicyViolationsOperationalUnaudited))

	tflog.Trace(ctx, "read a project metrics data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
