package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &NotificationPublisherDataSource{}

func NewNotificationPublisherDataSource() datasource.DataSource {
	return &NotificationPublisherDataSource{}
}

// NotificationPublisherDataSource defines the data source implementation.
type NotificationPublisherDataSource struct {
	client      *dtrack.Client
	baseURL     string
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// NotificationPublisherDataSourceModel describes the data source data model.
type NotificationPublisherDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	PublisherClass   types.String `tfsdk:"publisher_class"`
	Template         types.String `tfsdk:"template"`
	TemplateMimeType types.String `tfsdk:"template_mime_type"`
	DefaultPublisher types.Bool   `tfsdk:"default_publisher"`
}

func (d *NotificationPublisherDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_publisher"
}

func (d *NotificationPublisherDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a notification publisher from Dependency-Track by UUID or name. Either `uuid` or `name` must be specified.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the notification publisher (same as UUID)",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the notification publisher. Either `uuid` or `name` must be specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the notification publisher. Either `uuid` or `name` must be specified.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.MatchRoot("uuid")),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the notification publisher",
				Computed:            true,
			},
			"publisher_class": schema.StringAttribute{
				MarkdownDescription: "The fully qualified class name of the publisher implementation",
				Computed:            true,
			},
			"template": schema.StringAttribute{
				MarkdownDescription: "The template content for the notification",
				Computed:            true,
			},
			"template_mime_type": schema.StringAttribute{
				MarkdownDescription: "The MIME type of the template",
				Computed:            true,
			},
			"default_publisher": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a default publisher",
				Computed:            true,
			},
		},
	}
}

func (d *NotificationPublisherDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = providerData.Client
	d.baseURL = providerData.Endpoint
	d.apiKey = providerData.ApiKey
	d.bearerToken = providerData.BearerToken
	d.httpClient = &http.Client{}
}

func (d *NotificationPublisherDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NotificationPublisherDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	hasUUID := !data.UUID.IsNull() && data.UUID.ValueString() != ""
	hasName := !data.Name.IsNull() && data.Name.ValueString() != ""

	if !hasUUID && !hasName {
		resp.Diagnostics.AddError(
			"Missing Search Criteria",
			"Either 'uuid' or 'name' must be specified to look up a notification publisher.",
		)
		return
	}

	// Fetch all publishers (API only supports list endpoint)
	publishers, err := d.getAllPublishers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification publishers, got error: %s", err))
		return
	}

	var publisher *NotificationPublisher

	if hasUUID {
		publisherUUID, err := uuid.Parse(data.UUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse UUID: %s", err))
			return
		}

		for _, p := range publishers {
			if p.UUID == publisherUUID {
				publisher = &p
				break
			}
		}

		if publisher == nil {
			resp.Diagnostics.AddError(
				"Notification Publisher Not Found",
				fmt.Sprintf("No notification publisher found with UUID: %s", publisherUUID),
			)
			return
		}

		tflog.Trace(ctx, "read notification publisher data source by UUID")
	} else {
		searchName := data.Name.ValueString()

		for _, p := range publishers {
			if p.Name == searchName {
				publisher = &p
				break
			}
		}

		if publisher == nil {
			resp.Diagnostics.AddError(
				"Notification Publisher Not Found",
				fmt.Sprintf("No notification publisher found with name: %s", searchName),
			)
			return
		}

		tflog.Trace(ctx, "read notification publisher data source by name")
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

// Helper methods for API calls.

func (d *NotificationPublisherDataSource) getAllPublishers(ctx context.Context) ([]NotificationPublisher, error) {
	url := fmt.Sprintf("%s/api/v1/notification/publisher", d.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var publishers []NotificationPublisher
	if err := d.doRequest(req, &publishers); err != nil {
		return nil, err
	}

	return publishers, nil
}

func (d *NotificationPublisherDataSource) doRequest(req *http.Request, result interface{}) error {
	req.Header.Set("Content-Type", "application/json")

	if d.apiKey != "" {
		req.Header.Set("X-API-Key", d.apiKey)
	} else if d.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+d.bearerToken)
	}

	resp, err := d.httpClient.Do(req)
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
