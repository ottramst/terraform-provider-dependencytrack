package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	data *Data
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

// NotificationPublisher represents the API model. It doubles as the request
// body on DT v4 and as the response shape on both major versions: v4 servers
// populate publisherClass, while v5 servers populate extensionName instead
// (see class()). v5 requests use notificationPublisherV5Request.
type NotificationPublisher struct {
	UUID             uuid.UUID `json:"uuid,omitempty"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	PublisherClass   string    `json:"publisherClass"`
	ExtensionName    string    `json:"extensionName,omitempty"`
	Template         string    `json:"template,omitempty"`
	TemplateMimeType string    `json:"templateMimeType"`
	DefaultPublisher bool      `json:"defaultPublisher,omitempty"`
}

// class returns the value carried by the publisher_class Terraform attribute:
// the v5 extension name when the server returned one, otherwise the v4 fully
// qualified publisher class name.
func (p NotificationPublisher) class() string {
	if p.ExtensionName != "" {
		return p.ExtensionName
	}
	return p.PublisherClass
}

// notificationPublisherV5Request is the request body accepted by DT v5's
// notification publisher endpoints (hyades-apiserver's
// Create/UpdateNotificationPublisherRequest DTOs). Unlike v4, v5 identifies
// the publisher implementation by extensionName rather than publisherClass,
// rejects create bodies carrying a uuid, and requires the uuid on update.
type notificationPublisherV5Request struct {
	UUID             *uuid.UUID `json:"uuid,omitempty"` // update only
	Name             string     `json:"name"`
	ExtensionName    string     `json:"extensionName"`
	Description      string     `json:"description,omitempty"`
	Template         string     `json:"template,omitempty"`
	TemplateMimeType string     `json:"templateMimeType"`
}

// warnOnPublisherClassVersionMismatch appends a warning diagnostic when the
// configured publisher_class value does not match the naming convention of
// the server's major version: DT v4 identifies publishers by fully qualified
// Java class name (e.g. org.dependencytrack.notification.publisher.WebhookPublisher),
// while DT v5 identifies them by short extension name (e.g. "webhook", "email").
func warnOnPublisherClassVersionMismatch(diags *diag.Diagnostics, isV5 bool, class string) {
	looksLikeFQCN := strings.Contains(class, ".")

	switch {
	case isV5 && looksLikeFQCN:
		diags.AddAttributeWarning(
			path.Root("publisher_class"),
			"Publisher class looks like a Dependency-Track v4 class name",
			fmt.Sprintf("The configured publisher_class %q looks like a fully qualified Java class name, "+
				"but the server is running Dependency-Track v5, which identifies notification publishers "+
				"by extension name (e.g. \"webhook\" or \"email\"). The server will likely reject this value.", class),
		)
	case !isV5 && !looksLikeFQCN:
		diags.AddAttributeWarning(
			path.Root("publisher_class"),
			"Publisher class looks like a Dependency-Track v5 extension name",
			fmt.Sprintf("The configured publisher_class %q looks like a v5 extension name, "+
				"but the server is running Dependency-Track v4, which identifies notification publishers "+
				"by fully qualified Java class name (e.g. \"org.dependencytrack.notification.publisher.WebhookPublisher\").", class),
		)
	}
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
				MarkdownDescription: "The publisher implementation to use. On Dependency-Track v4 this is a fully qualified class name (e.g., org.dependencytrack.notification.publisher.WebhookPublisher); on v5 it is an extension name (e.g., webhook, email)",
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

	r.data = providerData
}

func (r *NotificationPublisherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationPublisherResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	warnOnPublisherClassVersionMismatch(&resp.Diagnostics, r.data.IsV5(), data.PublisherClass.ValueString())

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
	data.PublisherClass = types.StringValue(createdPublisher.class())
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
		if isNotFound(err) {
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
	data.PublisherClass = types.StringValue(publisher.class())
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

	warnOnPublisherClassVersionMismatch(&resp.Diagnostics, r.data.IsV5(), data.PublisherClass.ValueString())

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
	data.PublisherClass = types.StringValue(updatedPublisher.class())
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
	// v4 accepts the publisher model as-is; v5 expects extensionName instead
	// of publisherClass and no uuid in the create body.
	var body any = publisher
	if r.data.IsV5() {
		body = notificationPublisherV5Request{
			Name:             publisher.Name,
			ExtensionName:    publisher.PublisherClass,
			Description:      publisher.Description,
			Template:         publisher.Template,
			TemplateMimeType: publisher.TemplateMimeType,
		}
	}

	var result NotificationPublisher
	if err := r.data.API().Do(ctx, http.MethodPut, "/api/v1/notification/publisher", body, &result); err != nil {
		return NotificationPublisher{}, err
	}

	return result, nil
}

func (r *NotificationPublisherResource) getPublisher(ctx context.Context, publisherUUID uuid.UUID) (NotificationPublisher, error) {
	publishers, err := apiGetAllPages[NotificationPublisher](ctx, r.data.API(), "/api/v1/notification/publisher", nil)
	if err != nil {
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
	// v4 accepts the publisher model as-is; v5 expects extensionName instead
	// of publisherClass, and requires the uuid in the update body.
	var body any = publisher
	if r.data.IsV5() {
		publisherUUID := publisher.UUID
		body = notificationPublisherV5Request{
			UUID:             &publisherUUID,
			Name:             publisher.Name,
			ExtensionName:    publisher.PublisherClass,
			Description:      publisher.Description,
			Template:         publisher.Template,
			TemplateMimeType: publisher.TemplateMimeType,
		}
	}

	var result NotificationPublisher
	if err := r.data.API().Do(ctx, http.MethodPost, "/api/v1/notification/publisher", body, &result); err != nil {
		return NotificationPublisher{}, err
	}

	return result, nil
}

func (r *NotificationPublisherResource) deletePublisher(ctx context.Context, publisherUUID uuid.UUID) error {
	return r.data.API().Do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/notification/publisher/%s", publisherUUID), nil, nil)
}
