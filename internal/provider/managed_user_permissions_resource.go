package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ManagedUserPermissionsResource{}
var _ resource.ResourceWithImportState = &ManagedUserPermissionsResource{}

func NewManagedUserPermissionsResource() resource.Resource {
	return &ManagedUserPermissionsResource{}
}

// ManagedUserPermissionsResource defines the resource implementation.
type ManagedUserPermissionsResource struct {
	client      *dtrack.Client
	baseURL     string
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// ManagedUserPermissionsResourceModel describes the resource data model.
type ManagedUserPermissionsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	User        types.String `tfsdk:"user"`
	Permissions types.Set    `tfsdk:"permissions"`
}

// ManagedUserWithPermissions represents a managed user with permissions from the API.
type ManagedUserWithPermissions struct {
	Username            string              `json:"username"`
	Fullname            string              `json:"fullname,omitempty"`
	Email               string              `json:"email,omitempty"`
	Suspended           bool                `json:"suspended"`
	ForcePasswordChange bool                `json:"forcePasswordChange"`
	NonExpiryPassword   bool                `json:"nonExpiryPassword"`
	Permissions         []dtrack.Permission `json:"permissions"`
}

func (r *ManagedUserPermissionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_user_permissions"
}

func (r *ManagedUserPermissionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages permissions for a managed user in Dependency-Track. This resource manages the complete set of permissions assigned to a managed user. Available permissions: ACCESS_MANAGEMENT, BOM_UPLOAD, POLICY_MANAGEMENT, POLICY_VIOLATION_ANALYSIS, PORTFOLIO_MANAGEMENT, PROJECT_CREATION_UPLOAD, SYSTEM_CONFIGURATION, TAG_MANAGEMENT, VIEW_BADGES, VIEW_POLICY_VIOLATION, VIEW_PORTFOLIO, VIEW_VULNERABILITY, VULNERABILITY_ANALYSIS, VULNERABILITY_MANAGEMENT.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The username of the user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The username of the managed user to manage permissions for",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Set of permission names to assign to the user (e.g., BOM_UPLOAD, PORTFOLIO_MANAGEMENT)",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ManagedUserPermissionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = data.Client
	r.baseURL = data.Endpoint
	r.apiKey = data.ApiKey
	r.bearerToken = data.BearerToken
	r.httpClient = &http.Client{}
}

func (r *ManagedUserPermissionsResource) addPermissionToUser(ctx context.Context, username, permission string) error {
	url := fmt.Sprintf("%s/api/v1/permission/%s/user/%s", r.baseURL, permission, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if r.apiKey != "" {
		req.Header.Set("X-Api-Key", r.apiKey)
	} else if r.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.bearerToken)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (r *ManagedUserPermissionsResource) removePermissionFromUser(ctx context.Context, username, permission string) error {
	url := fmt.Sprintf("%s/api/v1/permission/%s/user/%s", r.baseURL, permission, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if r.apiKey != "" {
		req.Header.Set("X-Api-Key", r.apiKey)
	} else if r.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.bearerToken)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (r *ManagedUserPermissionsResource) getUserPermissions(ctx context.Context, username string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/user/managed", r.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if r.apiKey != "" {
		req.Header.Set("X-Api-Key", r.apiKey)
	} else if r.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.bearerToken)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var users []ManagedUserWithPermissions
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	// Find the user by username
	for _, user := range users {
		if user.Username == username {
			permissions := make([]string, 0, len(user.Permissions))
			for _, perm := range user.Permissions {
				permissions = append(permissions, perm.Name)
			}
			return permissions, nil
		}
	}

	// User not found
	return nil, nil
}

func (r *ManagedUserPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ManagedUserPermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	username := data.User.ValueString()

	// Get desired permissions from plan
	var desiredPermissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &desiredPermissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Add each permission to the user
	for _, permName := range desiredPermissions {
		err := r.addPermissionToUser(ctx, username, permName)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add permission %s to user, got error: %s", permName, err))
			return
		}
	}

	// Read back the actual permissions from the API to ensure state consistency
	actualPermissions, err := r.getUserPermissions(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user permissions after create, got error: %s", err))
		return
	}

	// Convert to Set type
	permissionsSet, diags := types.SetValueFrom(ctx, types.StringType, actualPermissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(username)
	data.Permissions = permissionsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedUserPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ManagedUserPermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	username := data.User.ValueString()

	permissions, err := r.getUserPermissions(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user permissions, got error: %s", err))
		return
	}

	// Check if user exists
	if permissions == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Convert to Set type
	permissionsSet, diags := types.SetValueFrom(ctx, types.StringType, permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(username)
	data.Permissions = permissionsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedUserPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ManagedUserPermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	username := plan.User.ValueString()

	// Get current and desired permissions
	var currentPermissions, desiredPermissions []string
	resp.Diagnostics.Append(state.Permissions.ElementsAs(ctx, &currentPermissions, false)...)
	resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &desiredPermissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to maps for easier lookup
	currentMap := make(map[string]bool)
	desiredMap := make(map[string]bool)
	for _, p := range currentPermissions {
		currentMap[p] = true
	}
	for _, p := range desiredPermissions {
		desiredMap[p] = true
	}

	// Add permissions that are in desired but not in current
	for _, permName := range desiredPermissions {
		if !currentMap[permName] {
			err := r.addPermissionToUser(ctx, username, permName)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add permission %s to user, got error: %s", permName, err))
				return
			}
		}
	}

	// Remove permissions that are in current but not in desired
	for _, permName := range currentPermissions {
		if !desiredMap[permName] {
			err := r.removePermissionFromUser(ctx, username, permName)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove permission %s from user, got error: %s", permName, err))
				return
			}
		}
	}

	// Read back the actual permissions from the API to ensure state consistency
	actualPermissions, err := r.getUserPermissions(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user permissions after update, got error: %s", err))
		return
	}

	// Convert to Set type
	permissionsSet, diags := types.SetValueFrom(ctx, types.StringType, actualPermissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(username)
	plan.Permissions = permissionsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ManagedUserPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ManagedUserPermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	username := data.User.ValueString()

	// Get all permissions to remove
	var permissions []string
	resp.Diagnostics.Append(data.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove each permission from the user
	for _, permName := range permissions {
		err := r.removePermissionFromUser(ctx, username, permName)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove permission %s from user, got error: %s", permName, err))
			return
		}
	}
}

func (r *ManagedUserPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using username
	username := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user"), username)...)
}
