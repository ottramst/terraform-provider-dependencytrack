package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &LicenseDataSource{}

func NewLicenseDataSource() datasource.DataSource {
	return &LicenseDataSource{}
}

// LicenseDataSource defines the data source implementation.
type LicenseDataSource struct {
	data *Data
}

// LicenseDataSourceModel describes the data source data model.
type LicenseDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	UUID        types.String `tfsdk:"uuid"`
	LicenseID   types.String `tfsdk:"license_id"`
	Name        types.String `tfsdk:"name"`
	Text        types.String `tfsdk:"text"`
	Template    types.String `tfsdk:"template"`
	Header      types.String `tfsdk:"header"`
	Comment     types.String `tfsdk:"comment"`
	OSIApproved types.Bool   `tfsdk:"osi_approved"`
	FSFLibre    types.Bool   `tfsdk:"fsf_libre"`
	SeeAlso     types.List   `tfsdk:"see_also"`
}

func (d *LicenseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license"
}

func (d *LicenseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single license from Dependency-Track by its SPDX-style license ID. Works for both built-in and custom licenses.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the license (same as `license_id`)",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the license",
			},
			"license_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The SPDX-style license ID to look up (e.g. `Apache-2.0`)",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the license",
			},
			"text": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full text of the license",
			},
			"template": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The standard license template",
			},
			"header": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The standard license header",
			},
			"comment": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "A comment about the license",
			},
			"osi_approved": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the license is approved by the Open Source Initiative",
			},
			"fsf_libre": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the license is considered libre by the Free Software Foundation",
			},
			"see_also": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "A list of URLs with more information about the license",
			},
		},
	}
}

func (d *LicenseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LicenseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LicenseDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	license, err := d.data.Client.License.Get(ctx, data.LicenseID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read license, got error: %s", err))
		return
	}

	data.ID = types.StringValue(license.LicenseID)
	data.UUID = types.StringValue(license.UUID.String())
	data.LicenseID = types.StringValue(license.LicenseID)
	data.Name = types.StringValue(license.Name)
	data.Text = types.StringValue(license.Text)
	data.Template = types.StringValue(license.Template)
	data.Header = types.StringValue(license.Header)
	data.Comment = types.StringValue(license.Comment)
	data.OSIApproved = types.BoolValue(license.OSIApproved)
	data.FSFLibre = types.BoolValue(license.FSFLibre)

	seeAlso, diags := types.ListValueFrom(ctx, types.StringType, license.SeeAlso)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SeeAlso = seeAlso

	tflog.Trace(ctx, "read a license data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
