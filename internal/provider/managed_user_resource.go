package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ManagedUserResource{}
var _ resource.ResourceWithImportState = &ManagedUserResource{}

func NewManagedUserResource() resource.Resource {
	return &ManagedUserResource{}
}

// ManagedUserResource defines the resource implementation.
type ManagedUserResource struct {
	data *Data
}

// ManagedUserResourceModel describes the resource data model.
type ManagedUserResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Username            types.String `tfsdk:"username"`
	Fullname            types.String `tfsdk:"fullname"`
	Email               types.String `tfsdk:"email"`
	Password            types.String `tfsdk:"password"`
	Suspended           types.Bool   `tfsdk:"suspended"`
	ForcePasswordChange types.Bool   `tfsdk:"force_password_change"`
	NonExpiryPassword   types.Bool   `tfsdk:"non_expiry_password"`
}

func (r *ManagedUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_user"
}

func (r *ManagedUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a managed user in Dependency-Track",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the user (same as username)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the user",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fullname": schema.StringAttribute{
				MarkdownDescription: "The full name of the user",
				Required:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email address of the user",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for the user (only used during create/update)",
				Optional:            true,
				Sensitive:           true,
			},
			"suspended": schema.BoolAttribute{
				MarkdownDescription: "Whether the user account is suspended",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"force_password_change": schema.BoolAttribute{
				MarkdownDescription: "Whether to force the user to change password on next login",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"non_expiry_password": schema.BoolAttribute{
				MarkdownDescription: "Whether the password never expires",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *ManagedUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.data = providerData
}

func (r *ManagedUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedUserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get password value to use for both newPassword and confirmPassword
	password := data.Password.ValueString()

	// Create managed user via API
	user := dtrack.ManagedUser{
		Username:            data.Username.ValueString(),
		Fullname:            data.Fullname.ValueString(),
		Email:               data.Email.ValueString(),
		NewPassword:         password,
		ConfirmPassword:     password,
		Suspended:           data.Suspended.ValueBool(),
		ForcePasswordChange: data.ForcePasswordChange.ValueBool(),
		NonExpiryPassword:   data.NonExpiryPassword.ValueBool(),
	}

	createdUser, err := r.data.Client.User.CreateManaged(ctx, user)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create managed user, got error: %s", err))
		return
	}

	// Update state with values from API
	data.ID = types.StringValue(createdUser.Username)
	data.Username = types.StringValue(createdUser.Username)
	data.Fullname = types.StringValue(createdUser.Fullname)
	data.Email = types.StringValue(createdUser.Email)
	data.Suspended = types.BoolValue(createdUser.Suspended)
	data.ForcePasswordChange = types.BoolValue(createdUser.ForcePasswordChange)
	data.NonExpiryPassword = types.BoolValue(createdUser.NonExpiryPassword)

	// Keep password from plan in state (it's already marked as sensitive)

	tflog.Trace(ctx, "created a managed user resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedUserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get user from API
	user, found, err := r.getManagedUser(ctx, data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read managed user, got error: %s", err))
		return
	}
	if !found {
		// User doesn't exist anymore, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with values from API
	data.Fullname = types.StringValue(user.Fullname)
	data.Email = types.StringValue(user.Email)
	data.Suspended = types.BoolValue(user.Suspended)
	data.ForcePasswordChange = types.BoolValue(user.ForcePasswordChange)
	data.NonExpiryPassword = types.BoolValue(user.NonExpiryPassword)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ManagedUserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get password value to use for both newPassword and confirmPassword
	password := data.Password.ValueString()

	// Update user via API
	user := dtrack.ManagedUser{
		Username:            data.Username.ValueString(),
		Fullname:            data.Fullname.ValueString(),
		Email:               data.Email.ValueString(),
		NewPassword:         password,
		ConfirmPassword:     password,
		Suspended:           data.Suspended.ValueBool(),
		ForcePasswordChange: data.ForcePasswordChange.ValueBool(),
		NonExpiryPassword:   data.NonExpiryPassword.ValueBool(),
	}

	updatedUser, err := r.data.Client.User.UpdateManaged(ctx, user)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update managed user, got error: %s", err))
		return
	}

	// Update state with values from API
	data.Fullname = types.StringValue(updatedUser.Fullname)
	data.Email = types.StringValue(updatedUser.Email)
	data.Suspended = types.BoolValue(updatedUser.Suspended)
	data.ForcePasswordChange = types.BoolValue(updatedUser.ForcePasswordChange)
	data.NonExpiryPassword = types.BoolValue(updatedUser.NonExpiryPassword)

	// Keep password from plan in state (it's already marked as sensitive)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedUserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete user via API
	user := dtrack.ManagedUser{
		Username: data.Username.ValueString(),
	}

	err := r.data.Client.User.DeleteManaged(ctx, user)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete managed user, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a managed user resource")
}

func (r *ManagedUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the username
	username := req.ID

	// Set both id and username to the import ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
}

// Helper methods for API calls

// getManagedUser lists all managed users and returns the one matching
// username. The managed user endpoint has no get-by-name variant, so a missing
// user is reported via found=false (not an error): the list call itself
// succeeds, so there is no HTTP 404 for isNotFound to key off, and the Read
// path relies on found to decide whether to remove the resource.
func (r *ManagedUserResource) getManagedUser(ctx context.Context, username string) (*dtrack.ManagedUser, bool, error) {
	users, err := fetchAllPages(ctx, r.data.Client.User.GetAllManaged)
	if err != nil {
		return nil, false, err
	}

	// Find user by username
	for i := range users {
		if users[i].Username == username {
			return &users[i], true, nil
		}
	}

	return nil, false, nil
}
