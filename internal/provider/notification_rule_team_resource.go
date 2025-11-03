package provider

import (
	"context"
	"fmt"
	"net/http"

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
var _ resource.Resource = &NotificationRuleTeamResource{}
var _ resource.ResourceWithImportState = &NotificationRuleTeamResource{}

func NewNotificationRuleTeamResource() resource.Resource {
	return &NotificationRuleTeamResource{}
}

// NotificationRuleTeamResource defines the resource implementation.
type NotificationRuleTeamResource struct {
	client      *dtrack.Client
	baseURL     string
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// NotificationRuleTeamResourceModel describes the resource data model.
type NotificationRuleTeamResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Rule types.String `tfsdk:"rule"`
	Team types.String `tfsdk:"team"`
}

func (r *NotificationRuleTeamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_rule_team"
}

func (r *NotificationRuleTeamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Associates a team with a notification rule. **IMPORTANT**: This only works with notification rules using the EMAIL publisher (SendMailPublisher). Teams receive email notifications when the rule is triggered.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the association (format: rule/team)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rule": schema.StringAttribute{
				MarkdownDescription: "The UUID of the notification rule",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"team": schema.StringAttribute{
				MarkdownDescription: "The UUID of the team to associate with the rule",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NotificationRuleTeamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationRuleTeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationRuleTeamResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.Rule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Rule UUID", fmt.Sprintf("Unable to parse rule UUID: %s", err))
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	if err := r.addTeamToRule(ctx, ruleUUID, teamUUID); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add team to notification rule, got error: %s", err))
		return
	}

	// Set the ID as a composite of rule/team
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", ruleUUID, teamUUID))

	tflog.Trace(ctx, "created a notification rule team association")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleTeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationRuleTeamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.Rule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Rule UUID", fmt.Sprintf("Unable to parse rule UUID: %s", err))
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Check if the association still exists
	exists, err := r.teamAssociationExists(ctx, ruleUUID, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check notification rule team association, got error: %s", err))
		return
	}

	if !exists {
		// Association doesn't exist anymore, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleTeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since both rule and team are marked as RequiresReplace, this should never be called
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Updating a notification rule team association is not supported. Please delete and recreate the association.",
	)
}

func (r *NotificationRuleTeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationRuleTeamResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.Rule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Rule UUID", fmt.Sprintf("Unable to parse rule UUID: %s", err))
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	if err := r.removeTeamFromRule(ctx, ruleUUID, teamUUID); err != nil {
		// If we get a 404, it means the association (or the rule/team itself) no longer exists.
		// This is the desired end state, so we can consider the deletion successful.
		if isNotFoundError(err) {
			tflog.Debug(ctx, "notification rule or team already deleted, considering team association deletion successful")
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove team from notification rule, got error: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "deleted a notification rule team association")
}

func (r *NotificationRuleTeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: rule/team
	rule, team, err := parseCompositeID(req.ID, "rule", "team")
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Unable to parse import ID: %s\nExpected format: rule/team", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule"), rule)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team"), team)...)
}

// Helper methods

func (r *NotificationRuleTeamResource) addTeamToRule(ctx context.Context, ruleUUID, teamUUID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/notification/rule/%s/team/%s", r.baseURL, ruleUUID, teamUUID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	return r.doRequest(req)
}

func (r *NotificationRuleTeamResource) removeTeamFromRule(ctx context.Context, ruleUUID, teamUUID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/notification/rule/%s/team/%s", r.baseURL, ruleUUID, teamUUID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	return r.doRequest(req)
}

func (r *NotificationRuleTeamResource) teamAssociationExists(ctx context.Context, ruleUUID, teamUUID uuid.UUID) (bool, error) {
	// Get the notification rule and check if the team is in its teams list
	url := fmt.Sprintf("%s/api/v1/notification/rule", r.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	var rules []NotificationRule
	if err := r.doRequestWithResponse(req, &rules); err != nil {
		return false, err
	}

	// Find the rule by UUID
	for _, rule := range rules {
		if rule.UUID == ruleUUID {
			// Check if the team is in the rule's teams list
			for _, team := range rule.Teams {
				if team.UUID == teamUUID {
					return true, nil
				}
			}
			return false, nil
		}
	}

	return false, fmt.Errorf("notification rule not found: %s", ruleUUID)
}

func (r *NotificationRuleTeamResource) doRequest(req *http.Request) error {
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
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

func (r *NotificationRuleTeamResource) doRequestWithResponse(req *http.Request, result interface{}) error {
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
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		return decodeJSON(resp.Body, result)
	}

	return nil
}
