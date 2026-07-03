package provider

import (
	"context"
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
var _ datasource.DataSource = &LicenseGroupDataSource{}

func NewLicenseGroupDataSource() datasource.DataSource {
	return &LicenseGroupDataSource{}
}

// LicenseGroupDataSource defines the data source implementation.
type LicenseGroupDataSource struct {
	data *Data
}

// LicenseGroupDataSourceModel describes the data source data model.
type LicenseGroupDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	RiskWeight types.Int64  `tfsdk:"risk_weight"`
}

func (d *LicenseGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_group"
}

func (d *LicenseGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a license group from Dependency-Track by UUID or name. Exactly one of `id` or `name` must be specified.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The UUID of the license group. Exactly one of `id` or `name` must be specified.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the license group. Exactly one of `id` or `name` must be specified.",
			},
			"risk_weight": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The risk weight of the license group",
			},
		},
	}
}

func (d *LicenseGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LicenseGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LicenseGroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull() && data.ID.ValueString() != ""

	var group dtrack.LicenseGroup

	if hasID {
		groupUUID, err := uuid.Parse(data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse license group UUID: %s", err))
			return
		}

		group, err = d.data.Client.LicenseGroup.Get(ctx, groupUUID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read license group by ID, got error: %s", err))
			return
		}

		tflog.Trace(ctx, "read license group data source by ID")
	} else {
		searchName := data.Name.ValueString()

		found, err := d.findGroupByName(ctx, searchName)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read license groups, got error: %s", err))
			return
		}

		if found == nil {
			resp.Diagnostics.AddError(
				"License Group Not Found",
				fmt.Sprintf("No license group found with name: %s", searchName),
			)
			return
		}

		group = *found
		tflog.Trace(ctx, "read license group data source by name")
	}

	data.ID = types.StringValue(group.UUID.String())
	data.Name = types.StringValue(group.Name)
	data.RiskWeight = types.Int64Value(int64(group.RiskWeight))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// findGroupByName returns the license group with the given name, or nil if
// none matches. The client-go GetAll endpoint returns every license group in a
// single (unpaginated) response, so a single call suffices.
func (d *LicenseGroupDataSource) findGroupByName(ctx context.Context, name string) (*dtrack.LicenseGroup, error) {
	page, err := d.data.Client.LicenseGroup.GetAll(ctx, dtrack.PageOptions{}, dtrack.SortOptions{})
	if err != nil {
		return nil, err
	}

	for i := range page.Items {
		if page.Items[i].Name == name {
			return &page.Items[i], nil
		}
	}

	return nil, nil
}
