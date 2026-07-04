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
var _ datasource.DataSource = &ManagedUserDataSource{}

func NewManagedUserDataSource() datasource.DataSource {
	return &ManagedUserDataSource{}
}

// ManagedUserDataSource defines the data source implementation.
type ManagedUserDataSource struct {
	data *Data
}

// ManagedUserDataSourceModel describes the data source data model.
type ManagedUserDataSourceModel struct {
	Username            types.String `tfsdk:"username"`
	Fullname            types.String `tfsdk:"fullname"`
	Email               types.String `tfsdk:"email"`
	Suspended           types.Bool   `tfsdk:"suspended"`
	ForcePasswordChange types.Bool   `tfsdk:"force_password_change"`
	NonExpiryPassword   types.Bool   `tfsdk:"non_expiry_password"`
}

func (d *ManagedUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_user"
}

func (d *ManagedUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a managed user from Dependency-Track by username",

		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the user to look up",
				Required:            true,
			},
			"fullname": schema.StringAttribute{
				MarkdownDescription: "The full name of the user",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the user",
				Computed:            true,
			},
			"suspended": schema.BoolAttribute{
				MarkdownDescription: "Whether the user account is suspended",
				Computed:            true,
			},
			"force_password_change": schema.BoolAttribute{
				MarkdownDescription: "Whether the user must change password on next login",
				Computed:            true,
			},
			"non_expiry_password": schema.BoolAttribute{
				MarkdownDescription: "Whether the password never expires",
				Computed:            true,
			},
		},
	}
}

func (d *ManagedUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.data = providerData
}

func (d *ManagedUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ManagedUserDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch managed user from API
	user, err := d.getManagedUser(ctx, data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed user, got error: %s", err))
		return
	}

	// Set data from API response
	data.Username = types.StringValue(user.Username)
	data.Fullname = types.StringValue(user.Fullname)
	data.Email = types.StringValue(user.Email)
	data.Suspended = types.BoolValue(user.Suspended)
	data.ForcePasswordChange = types.BoolValue(user.ForcePasswordChange)
	data.NonExpiryPassword = types.BoolValue(user.NonExpiryPassword)

	tflog.Trace(ctx, "read a managed user data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper methods for API calls

func (d *ManagedUserDataSource) getManagedUser(ctx context.Context, username string) (*dtrack.ManagedUser, error) {
	users, err := fetchAllPages(ctx, d.data.Client.User.GetAllManaged)
	if err != nil {
		return nil, err
	}

	// Find user by username
	for i := range users {
		if users[i].Username == username {
			return &users[i], nil
		}
	}

	return nil, fmt.Errorf("managed user not found: %s", username)
}
