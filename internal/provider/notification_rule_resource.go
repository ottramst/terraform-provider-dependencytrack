package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NotificationRuleResource{}
var _ resource.ResourceWithImportState = &NotificationRuleResource{}

func NewNotificationRuleResource() resource.Resource {
	return &NotificationRuleResource{}
}

// NotificationRuleResource defines the resource implementation.
type NotificationRuleResource struct {
	client      *dtrack.Client
	baseURL     string
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// NotificationRuleResourceModel describes the resource data model.
type NotificationRuleResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	NotifyChildren       types.Bool   `tfsdk:"notify_children"`
	LogSuccessfulPublish types.Bool   `tfsdk:"log_successful_publish"`
	Scope                types.String `tfsdk:"scope"`
	NotificationLevel    types.String `tfsdk:"notification_level"`
	Projects             types.Set    `tfsdk:"projects"`
	Teams                types.Set    `tfsdk:"teams"`
	NotifyOn             types.Set    `tfsdk:"notify_on"`
	Publisher            types.String `tfsdk:"publisher"`
	PublisherConfig      types.String `tfsdk:"publisher_config"`
}

// NotificationRule represents the API model.
type NotificationRule struct {
	UUID                 uuid.UUID                 `json:"uuid,omitempty"`
	Name                 string                    `json:"name"`
	Enabled              bool                      `json:"enabled"`
	NotifyChildren       bool                      `json:"notifyChildren"`
	LogSuccessfulPublish bool                      `json:"logSuccessfulPublish"`
	Scope                string                    `json:"scope"`
	NotificationLevel    string                    `json:"notificationLevel,omitempty"`
	Projects             []NotificationRuleProject `json:"projects,omitempty"`
	Teams                []NotificationRuleTeam    `json:"teams,omitempty"`
	NotifyOn             []string                  `json:"notifyOn,omitempty"`
	Publisher            NotificationRulePublisher `json:"publisher"`
	PublisherConfig      string                    `json:"publisherConfig,omitempty"`
}

type NotificationRuleProject struct {
	UUID uuid.UUID `json:"uuid"`
}

type NotificationRuleTeam struct {
	UUID uuid.UUID `json:"uuid"`
}

type NotificationRulePublisher struct {
	UUID uuid.UUID `json:"uuid"`
}

func (r *NotificationRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_rule"
}

func (r *NotificationRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a notification rule in Dependency-Track. Notification rules define when and how notifications are sent.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the notification rule (same as UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the notification rule",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the notification rule",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the notification rule is enabled (defaults to true if not specified)",
				Optional:            true,
				Computed:            true,
			},
			"notify_children": schema.BoolAttribute{
				MarkdownDescription: "Whether to notify on child projects (defaults to true if not specified)",
				Optional:            true,
				Computed:            true,
			},
			"log_successful_publish": schema.BoolAttribute{
				MarkdownDescription: "Whether to log successful notification publishing (defaults to false if not specified)",
				Optional:            true,
				Computed:            true,
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "The scope of the notification rule (PORTFOLIO or SYSTEM)",
				Required:            true,
			},
			"notification_level": schema.StringAttribute{
				MarkdownDescription: "The notification level (INFORMATIONAL, WARNING, or ERROR)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("INFORMATIONAL"),
			},
			"projects": schema.SetAttribute{
				MarkdownDescription: "Set of project UUIDs associated with this rule (read-only, use dependencytrack_notification_rule_project to manage)",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"teams": schema.SetAttribute{
				MarkdownDescription: "Set of team UUIDs associated with this rule (read-only, use dependencytrack_notification_rule_team to manage)",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"notify_on": schema.SetAttribute{
				MarkdownDescription: "Set of notification groups to trigger on (e.g., NEW_VULNERABILITY, POLICY_VIOLATION, etc.)",
				Required:            true,
				ElementType:         types.StringType,
			},
			"publisher": schema.StringAttribute{
				MarkdownDescription: "The UUID of the notification publisher to use",
				Required:            true,
			},
			"publisher_config": schema.StringAttribute{
				MarkdownDescription: "Publisher-specific configuration (JSON string)",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *NotificationRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse publisher UUID
	publisherUUID, err := uuid.Parse(data.Publisher.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Publisher UUID", fmt.Sprintf("Unable to parse publisher UUID: %s", err))
		return
	}

	// Build notification rule
	rule := NotificationRule{
		Name:  data.Name.ValueString(),
		Scope: data.Scope.ValueString(),
		Publisher: NotificationRulePublisher{
			UUID: publisherUUID,
		},
	}

	// Handle optional boolean fields - use API defaults if not specified
	if !data.Enabled.IsNull() && !data.Enabled.IsUnknown() {
		rule.Enabled = data.Enabled.ValueBool()
	} else {
		rule.Enabled = true // API default
	}

	if !data.NotifyChildren.IsNull() && !data.NotifyChildren.IsUnknown() {
		rule.NotifyChildren = data.NotifyChildren.ValueBool()
	} else {
		rule.NotifyChildren = true // API default
	}

	if !data.LogSuccessfulPublish.IsNull() && !data.LogSuccessfulPublish.IsUnknown() {
		rule.LogSuccessfulPublish = data.LogSuccessfulPublish.ValueBool()
	} else {
		rule.LogSuccessfulPublish = false // API default
	}

	// Add optional fields
	if !data.NotificationLevel.IsNull() && !data.NotificationLevel.IsUnknown() {
		rule.NotificationLevel = data.NotificationLevel.ValueString()
	}
	if !data.PublisherConfig.IsNull() && !data.PublisherConfig.IsUnknown() {
		rule.PublisherConfig = data.PublisherConfig.ValueString()
	}

	// Add notify_on
	if !data.NotifyOn.IsNull() && !data.NotifyOn.IsUnknown() {
		var notifyOn []string
		resp.Diagnostics.Append(data.NotifyOn.ElementsAs(ctx, &notifyOn, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		rule.NotifyOn = notifyOn
	}

	createdRule, err := r.createRule(ctx, rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create notification rule, got error: %s", err))
		return
	}

	// API quirk: The PUT endpoint (create) ignores several fields and uses API defaults instead:
	// - notifyOn: always returns empty array (default: [])
	// - enabled: always returns true (default: true)
	// - notifyChildren: always returns true (default: true)
	// - logSuccessfulPublish: always returns false (default: false)
	// We need to immediately follow up with a POST (update) if any of these differ from defaults.
	needsUpdate := false
	if len(rule.NotifyOn) > 0 {
		needsUpdate = true
		createdRule.NotifyOn = rule.NotifyOn
	}
	if !rule.Enabled {
		needsUpdate = true
		createdRule.Enabled = rule.Enabled
	}
	if !rule.NotifyChildren {
		needsUpdate = true
		createdRule.NotifyChildren = rule.NotifyChildren
	}
	if rule.LogSuccessfulPublish {
		needsUpdate = true
		createdRule.LogSuccessfulPublish = rule.LogSuccessfulPublish
	}

	if needsUpdate {
		tflog.Debug(ctx, "Following up with update to set fields ignored by PUT endpoint due to API limitation")
		createdRule, err = r.updateRule(ctx, createdRule)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update notification rule fields, got error: %s", err))
			return
		}
	}

	// Update model with response from create/update
	// Note: We use the create response directly instead of reading back, as the API
	// may return empty arrays for relationships even when they are set
	resp.Diagnostics.Append(r.updateModelFromAPI(ctx, &data, &createdRule)...)

	tflog.Trace(ctx, "created a notification rule resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse UUID: %s", err))
		return
	}

	rule, err := r.getRule(ctx, ruleUUID)
	if err != nil {
		if isNotFoundError(err) {
			// Rule doesn't exist anymore, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification rule, got error: %s", err))
		return
	}

	// Update model with response
	resp.Diagnostics.Append(r.updateModelFromAPI(ctx, &data, &rule)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NotificationRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse UUID: %s", err))
		return
	}

	// Parse publisher UUID
	publisherUUID, err := uuid.Parse(data.Publisher.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Publisher UUID", fmt.Sprintf("Unable to parse publisher UUID: %s", err))
		return
	}

	// Build notification rule
	rule := NotificationRule{
		UUID:  ruleUUID,
		Name:  data.Name.ValueString(),
		Scope: data.Scope.ValueString(),
		Publisher: NotificationRulePublisher{
			UUID: publisherUUID,
		},
	}

	// Handle optional boolean fields - use API defaults if not specified
	if !data.Enabled.IsNull() && !data.Enabled.IsUnknown() {
		rule.Enabled = data.Enabled.ValueBool()
	} else {
		rule.Enabled = true // API default
	}

	if !data.NotifyChildren.IsNull() && !data.NotifyChildren.IsUnknown() {
		rule.NotifyChildren = data.NotifyChildren.ValueBool()
	} else {
		rule.NotifyChildren = true // API default
	}

	if !data.LogSuccessfulPublish.IsNull() && !data.LogSuccessfulPublish.IsUnknown() {
		rule.LogSuccessfulPublish = data.LogSuccessfulPublish.ValueBool()
	} else {
		rule.LogSuccessfulPublish = false // API default
	}

	// Add optional fields
	if !data.NotificationLevel.IsNull() && !data.NotificationLevel.IsUnknown() {
		rule.NotificationLevel = data.NotificationLevel.ValueString()
	}
	if !data.PublisherConfig.IsNull() && !data.PublisherConfig.IsUnknown() {
		rule.PublisherConfig = data.PublisherConfig.ValueString()
	}

	// Add notify_on
	if !data.NotifyOn.IsNull() && !data.NotifyOn.IsUnknown() {
		var notifyOn []string
		resp.Diagnostics.Append(data.NotifyOn.ElementsAs(ctx, &notifyOn, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		rule.NotifyOn = notifyOn
	}

	updatedRule, err := r.updateRule(ctx, rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update notification rule, got error: %s", err))
		return
	}

	// Update model with response from update
	// Note: We use the update response directly instead of reading back, as the API
	// may return empty arrays for relationships even when they are set
	resp.Diagnostics.Append(r.updateModelFromAPI(ctx, &data, &updatedRule)...)

	tflog.Trace(ctx, "updated a notification rule resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse UUID: %s", err))
		return
	}

	err = r.deleteRule(ctx, ruleUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification rule, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a notification rule resource")
}

func (r *NotificationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using UUID
	ruleUUID, err := uuid.Parse(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid UUID",
			fmt.Sprintf("Unable to parse UUID: %s\nError: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ruleUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), ruleUUID.String())...)
}

// Helper methods

func (r *NotificationRuleResource) updateModelFromAPI(ctx context.Context, model *NotificationRuleResourceModel, rule *NotificationRule) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(rule.UUID.String())
	model.UUID = types.StringValue(rule.UUID.String())
	model.Name = types.StringValue(rule.Name)
	model.Enabled = types.BoolValue(rule.Enabled)
	model.NotifyChildren = types.BoolValue(rule.NotifyChildren)
	model.LogSuccessfulPublish = types.BoolValue(rule.LogSuccessfulPublish)
	model.Scope = types.StringValue(rule.Scope)

	// Handle optional string fields - use null for empty strings
	if rule.NotificationLevel != "" {
		model.NotificationLevel = types.StringValue(rule.NotificationLevel)
	} else if model.NotificationLevel.IsNull() {
		model.NotificationLevel = types.StringNull()
	}

	model.Publisher = types.StringValue(rule.Publisher.UUID.String())

	if rule.PublisherConfig != "" {
		model.PublisherConfig = types.StringValue(rule.PublisherConfig)
	} else {
		model.PublisherConfig = types.StringNull()
	}

	// Convert projects to set - always set what API returns
	projectUUIDs := make([]string, 0, len(rule.Projects))
	for _, project := range rule.Projects {
		projectUUIDs = append(projectUUIDs, project.UUID.String())
	}
	projectsSet, d := types.SetValueFrom(ctx, types.StringType, projectUUIDs)
	diags.Append(d...)
	model.Projects = projectsSet

	// Convert teams to set - always set what API returns
	teamUUIDs := make([]string, 0, len(rule.Teams))
	for _, team := range rule.Teams {
		teamUUIDs = append(teamUUIDs, team.UUID.String())
	}
	teamsSet, d := types.SetValueFrom(ctx, types.StringType, teamUUIDs)
	diags.Append(d...)
	model.Teams = teamsSet

	// Convert notify_on to set - always set what API returns
	notifyOn := make([]string, 0, len(rule.NotifyOn))
	notifyOn = append(notifyOn, rule.NotifyOn...)
	notifyOnSet, d := types.SetValueFrom(ctx, types.StringType, notifyOn)
	diags.Append(d...)
	model.NotifyOn = notifyOnSet

	return diags
}

// API methods

// createRule creates a new notification rule using PUT /api/v1/notification/rule.
// Note: This endpoint has known limitations - it ignores several fields and uses API defaults:
//   - notifyOn: always returns empty array (default: [])
//   - enabled: always returns true (default: true)
//   - notifyChildren: always returns true (default: true)
//   - logSuccessfulPublish: always returns false (default: false)
//
// Callers should follow up with updateRule() if any of these fields need non-default values.
func (r *NotificationRuleResource) createRule(ctx context.Context, rule NotificationRule) (NotificationRule, error) {
	url := fmt.Sprintf("%s/api/v1/notification/rule", r.baseURL)

	body, err := json.Marshal(rule)
	if err != nil {
		return NotificationRule{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return NotificationRule{}, err
	}

	var result NotificationRule
	if err := r.doRequest(req, &result); err != nil {
		return NotificationRule{}, err
	}

	return result, nil
}

func (r *NotificationRuleResource) getRule(ctx context.Context, ruleUUID uuid.UUID) (NotificationRule, error) {
	url := fmt.Sprintf("%s/api/v1/notification/rule", r.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return NotificationRule{}, err
	}

	var rules []NotificationRule
	if err := r.doRequest(req, &rules); err != nil {
		return NotificationRule{}, err
	}

	// Find the rule by UUID
	for _, rule := range rules {
		if rule.UUID == ruleUUID {
			return rule, nil
		}
	}

	return NotificationRule{}, fmt.Errorf("notification rule not found: %s", ruleUUID)
}

func (r *NotificationRuleResource) updateRule(ctx context.Context, rule NotificationRule) (NotificationRule, error) {
	url := fmt.Sprintf("%s/api/v1/notification/rule", r.baseURL)

	body, err := json.Marshal(rule)
	if err != nil {
		return NotificationRule{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return NotificationRule{}, err
	}

	var result NotificationRule
	if err := r.doRequest(req, &result); err != nil {
		return NotificationRule{}, err
	}

	return result, nil
}

func (r *NotificationRuleResource) deleteRule(ctx context.Context, ruleUUID uuid.UUID) error {
	// First, get the full rule object - DELETE requires complete object with all required fields
	rule, err := r.getRule(ctx, ruleUUID)
	if err != nil {
		// If rule doesn't exist, consider it already deleted
		if isNotFoundError(err) {
			return nil
		}
		return err
	}

	url := fmt.Sprintf("%s/api/v1/notification/rule", r.baseURL)

	// DELETE requires the rule object in the body with all required fields
	body, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	return r.doRequest(req, nil)
}

func (r *NotificationRuleResource) doRequest(req *http.Request, result interface{}) error {
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
