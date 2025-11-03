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
var _ resource.Resource = &NotificationPublisherResource{}
var _ resource.ResourceWithImportState = &NotificationPublisherResource{}

func NewNotificationPublisherResource() resource.Resource {
	return &NotificationPublisherResource{}
}

// NotificationPublisherResource defines the resource implementation.
type NotificationPublisherResource struct {
	client      *dtrack.Client
	baseURL     string
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// NotificationPublisherResourceModel describes the resource data model.
type NotificationPublisherResourceModel struct {
	ID               types.String `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	PublisherClass   types.String `tfsdk:"publisher_class"`
	Template         types.String `tfsdk:"template"`
	TemplateMimeType types.String `tfsdk:"template_mime_type"`
	DefaultPublisher types.Bool   `tfsdk:"default_publisher"`
}

// NotificationPublisher represents the API model.
type NotificationPublisher struct {
	UUID             uuid.UUID `json:"uuid,omitempty"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	PublisherClass   string    `json:"publisherClass"`
	Template         string    `json:"template,omitempty"`
	TemplateMimeType string    `json:"templateMimeType"`
	DefaultPublisher bool      `json:"defaultPublisher,omitempty"`
}

func (r *NotificationPublisherResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_publisher"
}

func (r *NotificationPublisherResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a notification publisher in Dependency-Track. Notification publishers are used to send notifications to external systems.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the notification publisher (same as UUID)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the notification publisher",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the notification publisher",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the notification publisher",
				Optional:            true,
				Computed:            true,
			},
			"publisher_class": schema.StringAttribute{
				MarkdownDescription: "The fully qualified class name of the publisher implementation (e.g., org.dependencytrack.notification.publisher.WebhookPublisher)",
				Required:            true,
			},
			"template": schema.StringAttribute{
				MarkdownDescription: "The template content for the notification",
				Optional:            true,
				Computed:            true,
			},
			"template_mime_type": schema.StringAttribute{
				MarkdownDescription: "The MIME type of the template (e.g., application/json, text/plain)",
				Required:            true,
			},
			"default_publisher": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a default publisher (read-only, cannot be modified or deleted)",
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *NotificationPublisherResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NotificationPublisherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	publisher := NotificationPublisher{
		Name:             data.Name.ValueString(),
		PublisherClass:   data.PublisherClass.ValueString(),
		TemplateMimeType: data.TemplateMimeType.ValueString(),
	}

	// Only include optional fields if they are set
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		publisher.Description = data.Description.ValueString()
	}
	if !data.Template.IsNull() && !data.Template.IsUnknown() {
		publisher.Template = data.Template.ValueString()
	}

	createdPublisher, err := r.createPublisher(ctx, publisher)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create notification publisher, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdPublisher.UUID.String())
	data.UUID = types.StringValue(createdPublisher.UUID.String())
	data.Name = types.StringValue(createdPublisher.Name)
	data.Description = types.StringValue(createdPublisher.Description)
	data.PublisherClass = types.StringValue(createdPublisher.PublisherClass)
	data.Template = types.StringValue(createdPublisher.Template)
	data.TemplateMimeType = types.StringValue(createdPublisher.TemplateMimeType)
	data.DefaultPublisher = types.BoolValue(createdPublisher.DefaultPublisher)

	tflog.Trace(ctx, "created a notification publisher resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationPublisherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	publisherUUID, err := uuid.Parse(data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse UUID: %s", err))
		return
	}

	publisher, err := r.getPublisher(ctx, publisherUUID)
	if err != nil {
		if isNotFoundError(err) {
			// Publisher doesn't exist anymore, remove from state
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification publisher, got error: %s", err))
		return
	}

	data.ID = types.StringValue(publisher.UUID.String())
	data.UUID = types.StringValue(publisher.UUID.String())
	data.Name = types.StringValue(publisher.Name)
	data.Description = types.StringValue(publisher.Description)
	data.PublisherClass = types.StringValue(publisher.PublisherClass)
	data.Template = types.StringValue(publisher.Template)
	data.TemplateMimeType = types.StringValue(publisher.TemplateMimeType)
	data.DefaultPublisher = types.BoolValue(publisher.DefaultPublisher)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationPublisherResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	publisherUUID, err := uuid.Parse(data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse UUID: %s", err))
		return
	}

	publisher := NotificationPublisher{
		UUID:             publisherUUID,
		Name:             data.Name.ValueString(),
		PublisherClass:   data.PublisherClass.ValueString(),
		TemplateMimeType: data.TemplateMimeType.ValueString(),
	}

	// Only include optional fields if they are set
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		publisher.Description = data.Description.ValueString()
	}
	if !data.Template.IsNull() && !data.Template.IsUnknown() {
		publisher.Template = data.Template.ValueString()
	}

	updatedPublisher, err := r.updatePublisher(ctx, publisher)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update notification publisher, got error: %s", err))
		return
	}

	data.ID = types.StringValue(updatedPublisher.UUID.String())
	data.UUID = types.StringValue(updatedPublisher.UUID.String())
	data.Name = types.StringValue(updatedPublisher.Name)
	data.Description = types.StringValue(updatedPublisher.Description)
	data.PublisherClass = types.StringValue(updatedPublisher.PublisherClass)
	data.Template = types.StringValue(updatedPublisher.Template)
	data.TemplateMimeType = types.StringValue(updatedPublisher.TemplateMimeType)
	data.DefaultPublisher = types.BoolValue(updatedPublisher.DefaultPublisher)

	tflog.Trace(ctx, "updated a notification publisher resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationPublisherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	publisherUUID, err := uuid.Parse(data.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse UUID: %s", err))
		return
	}

	err = r.deletePublisher(ctx, publisherUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification publisher, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a notification publisher resource")
}

func (r *NotificationPublisherResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using UUID
	publisherUUID, err := uuid.Parse(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid UUID",
			fmt.Sprintf("Unable to parse UUID: %s\nError: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), publisherUUID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), publisherUUID.String())...)
}

// Helper methods for API calls

func (r *NotificationPublisherResource) createPublisher(ctx context.Context, publisher NotificationPublisher) (NotificationPublisher, error) {
	url := fmt.Sprintf("%s/api/v1/notification/publisher", r.baseURL)

	body, err := json.Marshal(publisher)
	if err != nil {
		return NotificationPublisher{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return NotificationPublisher{}, err
	}

	var result NotificationPublisher
	if err := r.doRequest(req, &result); err != nil {
		return NotificationPublisher{}, err
	}

	return result, nil
}

func (r *NotificationPublisherResource) getPublisher(ctx context.Context, publisherUUID uuid.UUID) (NotificationPublisher, error) {
	url := fmt.Sprintf("%s/api/v1/notification/publisher", r.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return NotificationPublisher{}, err
	}

	var publishers []NotificationPublisher
	if err := r.doRequest(req, &publishers); err != nil {
		return NotificationPublisher{}, err
	}

	// Find the publisher by UUID
	for _, p := range publishers {
		if p.UUID == publisherUUID {
			return p, nil
		}
	}

	return NotificationPublisher{}, fmt.Errorf("notification publisher not found: %s", publisherUUID)
}

func (r *NotificationPublisherResource) updatePublisher(ctx context.Context, publisher NotificationPublisher) (NotificationPublisher, error) {
	url := fmt.Sprintf("%s/api/v1/notification/publisher", r.baseURL)

	body, err := json.Marshal(publisher)
	if err != nil {
		return NotificationPublisher{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return NotificationPublisher{}, err
	}

	var result NotificationPublisher
	if err := r.doRequest(req, &result); err != nil {
		return NotificationPublisher{}, err
	}

	return result, nil
}

func (r *NotificationPublisherResource) deletePublisher(ctx context.Context, publisherUUID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/notification/publisher/%s", r.baseURL, publisherUUID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	return r.doRequest(req, nil)
}

func (r *NotificationPublisherResource) doRequest(req *http.Request, result interface{}) error {
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

func isNotFoundError(err error) bool {
	return err != nil && (err.Error() == "notification publisher not found" ||
		(len(err.Error()) > 0 && err.Error()[0:3] == "API" && err.Error()[len(err.Error())-3:] == "404"))
}
