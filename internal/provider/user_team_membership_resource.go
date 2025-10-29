package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UserTeamMembershipResource{}
var _ resource.ResourceWithImportState = &UserTeamMembershipResource{}

func NewUserTeamMembershipResource() resource.Resource {
	return &UserTeamMembershipResource{}
}

// UserTeamMembershipResource defines the resource implementation.
type UserTeamMembershipResource struct {
	client      *dtrack.Client
	baseURL     string
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// UserTeamMembershipResourceModel describes the resource data model.
type UserTeamMembershipResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Team     types.String `tfsdk:"team"`
}

// IdentifiableObject represents an object with a UUID field.
type IdentifiableObject struct {
	UUID uuid.UUID `json:"uuid"`
}

func (r *UserTeamMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_team_membership"
}

func (r *UserTeamMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user's membership in a team in Dependency-Track. This resource associates a user with a team.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier in the format `username/team_uuid`",
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
			"team": schema.StringAttribute{
				MarkdownDescription: "The UUID of the team",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *UserTeamMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = providerData.Client
	r.baseURL = providerData.Endpoint
	r.apiKey = providerData.ApiKey
	r.bearerToken = providerData.BearerToken
	r.httpClient = &http.Client{}
}

func (r *UserTeamMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UserTeamMembershipResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse team UUID
	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Add team to user via API
	err = r.addTeamToUser(ctx, data.Username.ValueString(), teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add team to user, got error: %s", err))
		return
	}

	// Set the ID as a composite key
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.Username.ValueString(), data.Team.ValueString()))

	tflog.Trace(ctx, "created a user team membership resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserTeamMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UserTeamMembershipResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse team UUID
	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Verify membership exists
	exists, err := r.verifyMembership(ctx, data.Username.ValueString(), teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to verify team membership, got error: %s", err))
		return
	}

	if !exists {
		// If membership doesn't exist, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserTeamMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UserTeamMembershipResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Since both username and team require replacement, this should never be called
	// But we'll implement it for completeness
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Updating a user team membership requires replacing the resource. Both username and team require replacement.",
	)
}

func (r *UserTeamMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UserTeamMembershipResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse team UUID
	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Remove team from user via API
	err = r.removeTeamFromUser(ctx, data.Username.ValueString(), teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove team from user, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a user team membership resource")
}

func (r *UserTeamMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID should be in the format "username/team_uuid"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'username/team_uuid', got: %s", req.ID),
		)
		return
	}

	username := parts[0]
	teamUUID := parts[1]

	// Validate team UUID format
	if _, err := uuid.Parse(teamUUID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Team UUID",
			fmt.Sprintf("The team UUID in the import ID is not valid: %s", err),
		)
		return
	}

	// Set the attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), teamUUID)...)
}

// Helper methods for API calls

func (r *UserTeamMembershipResource) addTeamToUser(ctx context.Context, username string, teamUUID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/user/%s/membership", r.baseURL, username)

	identifiable := IdentifiableObject{UUID: teamUUID}
	body, err := json.Marshal(identifiable)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	return r.doRequest(req, nil)
}

func (r *UserTeamMembershipResource) removeTeamFromUser(ctx context.Context, username string, teamUUID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/user/%s/membership", r.baseURL, username)

	identifiable := IdentifiableObject{UUID: teamUUID}
	body, err := json.Marshal(identifiable)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	return r.doRequest(req, nil)
}

func (r *UserTeamMembershipResource) verifyMembership(ctx context.Context, username string, teamUUID uuid.UUID) (bool, error) {
	// Get the user's teams and check if the team is in the list
	// We'll use the managed user endpoint to get the user details
	url := fmt.Sprintf("%s/api/v1/user/managed", r.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	var users []struct {
		Username string `json:"username"`
		Teams    []struct {
			UUID uuid.UUID `json:"uuid"`
		} `json:"teams"`
	}

	if err := r.doRequest(req, &users); err != nil {
		return false, err
	}

	// Find the user by username
	for _, user := range users {
		if user.Username == username {
			// Check if the team is in the user's teams
			for _, team := range user.Teams {
				if team.UUID == teamUUID {
					return true, nil
				}
			}
			return false, nil
		}
	}

	// User not found - might be LDAP or OIDC user
	// Try LDAP users endpoint
	url = fmt.Sprintf("%s/api/v1/user/ldap", r.baseURL)
	req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	var ldapUsers []struct {
		Username string `json:"username"`
		Teams    []struct {
			UUID uuid.UUID `json:"uuid"`
		} `json:"teams"`
	}

	if err := r.doRequest(req, &ldapUsers); err != nil {
		// LDAP might not be configured, continue to OIDC
		tflog.Debug(ctx, "Failed to fetch LDAP users, trying OIDC", map[string]interface{}{"error": err.Error()})
	} else {
		for _, user := range ldapUsers {
			if user.Username == username {
				for _, team := range user.Teams {
					if team.UUID == teamUUID {
						return true, nil
					}
				}
				return false, nil
			}
		}
	}

	// Try OIDC users endpoint
	url = fmt.Sprintf("%s/api/v1/user/oidc", r.baseURL)
	req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	var oidcUsers []struct {
		Username string `json:"username"`
		Teams    []struct {
			UUID uuid.UUID `json:"uuid"`
		} `json:"teams"`
	}

	if err := r.doRequest(req, &oidcUsers); err != nil {
		// OIDC might not be configured
		tflog.Debug(ctx, "Failed to fetch OIDC users", map[string]interface{}{"error": err.Error()})
		return false, fmt.Errorf("user not found in managed, LDAP, or OIDC users: %s", username)
	}

	for _, user := range oidcUsers {
		if user.Username == username {
			for _, team := range user.Teams {
				if team.UUID == teamUUID {
					return true, nil
				}
			}
			return false, nil
		}
	}

	return false, fmt.Errorf("user not found: %s", username)
}

func (r *UserTeamMembershipResource) doRequest(req *http.Request, result interface{}) error {
	req.Header.Set("Content-Type", "application/json")

	// Set authentication header based on available credentials
	if r.apiKey != "" {
		req.Header.Set("X-API-Key", r.apiKey)
	} else if r.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.bearerToken)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}
