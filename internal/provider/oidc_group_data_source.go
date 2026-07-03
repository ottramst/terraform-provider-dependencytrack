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
var _ datasource.DataSource = &OIDCGroupDataSource{}

func NewOIDCGroupDataSource() datasource.DataSource {
	return &OIDCGroupDataSource{}
}

// OIDCGroupDataSource defines the data source implementation.
type OIDCGroupDataSource struct {
	data *Data
}

// OIDCGroupDataSourceModel describes the data source data model.
type OIDCGroupDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *OIDCGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_group"
}

func (d *OIDCGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches an OpenID Connect (OIDC) group from Dependency-Track by name. OIDC groups are plain database entities and can be looked up without an identity provider configured.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the OIDC group",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the OIDC group to look up",
			},
		},
	}
}

func (d *OIDCGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OIDCGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OIDCGroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	searchName := data.Name.ValueString()

	groups, err := d.data.Client.OIDC.GetAllGroups(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read OIDC groups, got error: %s", err))
		return
	}

	for i := range groups {
		if groups[i].Name == searchName {
			data.ID = types.StringValue(groups[i].UUID.String())
			data.Name = types.StringValue(groups[i].Name)

			tflog.Trace(ctx, "read an OIDC group data source")

			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"OIDC Group Not Found",
		fmt.Sprintf("No OIDC group found with name: %s", searchName),
	)
}
