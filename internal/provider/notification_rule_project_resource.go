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
var _ resource.Resource = &NotificationRuleProjectResource{}
var _ resource.ResourceWithImportState = &NotificationRuleProjectResource{}

func NewNotificationRuleProjectResource() resource.Resource {
	return &NotificationRuleProjectResource{}
}

// NotificationRuleProjectResource defines the resource implementation.
type NotificationRuleProjectResource struct {
	client      *dtrack.Client
	baseURL     string
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// NotificationRuleProjectResourceModel describes the resource data model.
type NotificationRuleProjectResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Rule    types.String `tfsdk:"rule"`
	Project types.String `tfsdk:"project"`
}

func (r *NotificationRuleProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_rule_project"
}

func (r *NotificationRuleProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Associates a project with a notification rule. This is only valid for notification rules with PORTFOLIO scope.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the association (format: rule/project)",
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
			"project": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project to associate with the rule",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NotificationRuleProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationRuleProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationRuleProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.Rule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Rule UUID", fmt.Sprintf("Unable to parse rule UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	if err := r.addProjectToRule(ctx, ruleUUID, projectUUID); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to add project to notification rule, got error: %s", err))
		return
	}

	// Set the ID as a composite of rule/project
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", ruleUUID, projectUUID))

	tflog.Trace(ctx, "created a notification rule project association")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationRuleProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.Rule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Rule UUID", fmt.Sprintf("Unable to parse rule UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	// Check if the association still exists
	exists, err := r.projectAssociationExists(ctx, ruleUUID, projectUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check notification rule project association, got error: %s", err))
		return
	}

	if !exists {
		// Association doesn't exist anymore, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since both rule and project are marked as RequiresReplace, this should never be called
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Updating a notification rule project association is not supported. Please delete and recreate the association.",
	)
}

func (r *NotificationRuleProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationRuleProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.Rule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Rule UUID", fmt.Sprintf("Unable to parse rule UUID: %s", err))
		return
	}

	projectUUID, err := uuid.Parse(data.Project.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Project UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	if err := r.removeProjectFromRule(ctx, ruleUUID, projectUUID); err != nil {
		// If we get a 404, it means the association (or the rule/project itself) no longer exists.
		// This is the desired end state, so we can consider the deletion successful.
		if isNotFoundError(err) {
			tflog.Debug(ctx, "notification rule or project already deleted, considering project association deletion successful")
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove project from notification rule, got error: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "deleted a notification rule project association")
}

func (r *NotificationRuleProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: rule/project
	rule, project, err := parseCompositeID(req.ID, "rule", "project")
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Unable to parse import ID: %s\nExpected format: rule/project", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rule"), rule)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
}

// Helper methods

func (r *NotificationRuleProjectResource) addProjectToRule(ctx context.Context, ruleUUID, projectUUID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/notification/rule/%s/project/%s", r.baseURL, ruleUUID, projectUUID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	return r.doRequest(req)
}

func (r *NotificationRuleProjectResource) removeProjectFromRule(ctx context.Context, ruleUUID, projectUUID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/notification/rule/%s/project/%s", r.baseURL, ruleUUID, projectUUID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	return r.doRequest(req)
}

func (r *NotificationRuleProjectResource) projectAssociationExists(ctx context.Context, ruleUUID, projectUUID uuid.UUID) (bool, error) {
	// Get the notification rule and check if the project is in its projects list
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
			// Check if the project is in the rule's projects list
			for _, project := range rule.Projects {
				if project.UUID == projectUUID {
					return true, nil
				}
			}
			return false, nil
		}
	}

	return false, fmt.Errorf("notification rule not found: %s", ruleUUID)
}

func (r *NotificationRuleProjectResource) doRequest(req *http.Request) error {
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

func (r *NotificationRuleProjectResource) doRequestWithResponse(req *http.Request, result interface{}) error {
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
