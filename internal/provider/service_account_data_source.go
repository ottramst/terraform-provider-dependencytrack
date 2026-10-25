package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ServiceAccountDataSource{}

func NewServiceAccountDataSource() datasource.DataSource {
	return &ServiceAccountDataSource{}
}

// ServiceAccountDataSource defines the data source implementation.
type ServiceAccountDataSource struct {
	data *Data
}

// ServiceAccountDataSourceModel describes the data source data model.
type ServiceAccountDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Username    types.String `tfsdk:"username"`
	Email       types.String `tfsdk:"email"`
	Suspended   types.Bool   `tfsdk:"suspended"`
	Teams       types.List   `tfsdk:"teams"`
	Permissions types.Set    `tfsdk:"permissions"`
}

var serviceAccountTeamAttrTypes = map[string]attr.Type{
	"uuid": types.StringType,
	"name": types.StringType,
}

func (d *ServiceAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (d *ServiceAccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Dependency-Track service account, including its team memberships and directly granted permissions. " +
			"Requires Dependency-Track 5.2 or newer and the `ACCESS_MANAGEMENT` permission.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the service account (same as `name`)",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the service account, without the `svc:` prefix",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full username of the service account: its name with the `svc:` prefix",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The email address of the service account",
			},
			"suspended": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the service account is suspended",
			},
			"teams": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The teams the service account is a member of, ordered by name",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the team",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the team",
						},
					},
				},
			},
			"permissions": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The permissions granted to the service account directly. Permissions inherited from its teams are not included",
			},
		},
	}
}

func (d *ServiceAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceAccountDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !requireAtLeast(d.data, 5, 2, "The dependencytrack_service_account data source", &resp.Diagnostics) {
		return
	}

	var sa getServiceAccountResponse
	if err := d.data.API().Do(ctx, http.MethodGet, serviceAccountPath(data.Name.ValueString()), nil, &sa); err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Service account %q not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service account, got error: %s", err))
		return
	}

	data.ID = types.StringValue(sa.Name)
	data.Username = types.StringValue(sa.Username)
	data.Suspended = types.BoolValue(sa.Suspended)
	if sa.Email == "" {
		data.Email = types.StringNull()
	} else {
		data.Email = types.StringValue(sa.Email)
	}

	teams := make([]attr.Value, 0, len(sa.Teams))
	for _, team := range sa.Teams {
		teams = append(teams, types.ObjectValueMust(serviceAccountTeamAttrTypes, map[string]attr.Value{
			"uuid": types.StringValue(team.UUID),
			"name": types.StringValue(team.Name),
		}))
	}
	teamList, diags := types.ListValue(types.ObjectType{AttrTypes: serviceAccountTeamAttrTypes}, teams)
	resp.Diagnostics.Append(diags...)

	if sa.Permissions == nil {
		sa.Permissions = []string{}
	}
	permissions, diags := types.SetValueFrom(ctx, types.StringType, sa.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Teams = teamList
	data.Permissions = permissions

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
