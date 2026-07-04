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
var _ datasource.DataSource = &LicensesDataSource{}

func NewLicensesDataSource() datasource.DataSource {
	return &LicensesDataSource{}
}

// LicensesDataSource defines the data source implementation.
type LicensesDataSource struct {
	data *Data
}

// LicensesDataSourceModel describes the data source data model.
type LicensesDataSourceModel struct {
	ID       types.String       `tfsdk:"id"`
	Licenses []LicenseDataModel `tfsdk:"licenses"`
}

// LicenseDataModel describes a single license in the concise list.
type LicenseDataModel struct {
	UUID        types.String `tfsdk:"uuid"`
	LicenseID   types.String `tfsdk:"license_id"`
	Name        types.String `tfsdk:"name"`
	OSIApproved types.Bool   `tfsdk:"osi_approved"`
	FSFLibre    types.Bool   `tfsdk:"fsf_libre"`
}

func (d *LicensesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_licenses"
}

func (d *LicensesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the concise list of all licenses known to Dependency-Track (built-in and custom).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source result (always `licenses`).",
			},
			"licenses": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of licenses",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the license",
						},
						"license_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The SPDX-style license ID",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the license",
						},
						"osi_approved": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the license is approved by the Open Source Initiative",
						},
						"fsf_libre": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the license is considered libre by the Free Software Foundation",
						},
					},
				},
			},
		},
	}
}

func (d *LicensesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LicensesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LicensesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	licenses, err := d.data.Client.License.GetConcise(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read licenses, got error: %s", err))
		return
	}

	data.ID = types.StringValue("licenses")
	data.Licenses = make([]LicenseDataModel, 0, len(licenses))
	for _, license := range licenses {
		data.Licenses = append(data.Licenses, LicenseDataModel{
			UUID:        types.StringValue(license.UUID.String()),
			LicenseID:   types.StringValue(license.LicenseID),
			Name:        types.StringValue(license.Name),
			OSIApproved: types.BoolValue(license.OSIApproved),
			FSFLibre:    types.BoolValue(license.FSFLibre),
		})
	}

	tflog.Trace(ctx, "read a licenses data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
