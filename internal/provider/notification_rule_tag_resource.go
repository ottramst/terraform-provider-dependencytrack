package provider

import (
	"context"
	"fmt"
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
var _ resource.Resource = &NotificationRuleTagResource{}
var _ resource.ResourceWithImportState = &NotificationRuleTagResource{}

func NewNotificationRuleTagResource() resource.Resource {
	return &NotificationRuleTagResource{}
}

// NotificationRuleTagResource defines the resource implementation.
type NotificationRuleTagResource struct {
	data *Data
}

// NotificationRuleTagResourceModel describes the resource data model.
type NotificationRuleTagResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Tag              types.String `tfsdk:"tag"`
	NotificationRule types.String `tfsdk:"notification_rule"`
}

func (r *NotificationRuleTagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_rule_tag"
}

func (r *NotificationRuleTagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the assignment of a tag to a notification rule in Dependency-Track. The tag must already exist (see the `dependencytrack_tag` resource).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the notification rule tag assignment in the format `tag_name/notification_rule_uuid`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the tag. Dependency-Track normalizes tag names to lowercase, so a mixed-case name is matched case-insensitively (using a lowercase name is recommended). Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"notification_rule": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the notification rule",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *NotificationRuleTagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.data = data
}

func (r *NotificationRuleTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NotificationRuleTagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.NotificationRule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Notification Rule UUID", fmt.Sprintf("Unable to parse notification rule UUID: %s", err))
		return
	}

	// Dependency-Track stores tag names lowercased; normalize before using the
	// name in the request path so lookups resolve consistently.
	tag := strings.ToLower(data.Tag.ValueString())

	err = r.data.Client.Tag.TagNotificationRules(ctx, tag, []uuid.UUID{ruleUUID})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to tag notification rule, got error: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.Tag.ValueString(), ruleUUID.String()))

	tflog.Trace(ctx, "created a notification rule tag resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationRuleTagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.NotificationRule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Notification Rule UUID", fmt.Sprintf("Unable to parse notification rule UUID: %s", err))
		return
	}

	tag := strings.ToLower(data.Tag.ValueString())

	// The tagged-rules listing of a nonexistent tag is an empty list (not an
	// error) on both DT v4 and v5, so a deleted tag also falls into the
	// not-found path below.
	rules, err := fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.TaggedPolicyListResponseItem], error) {
		return r.data.Client.Tag.GetNotificationRules(ctx, tag, po, dtrack.SortOptions{})
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tagged notification rules, got error: %s", err))
		return
	}

	found := false
	for i := range rules {
		if rules[i].UUID == ruleUUID {
			found = true
			break
		}
	}

	if !found {
		// Tag is not assigned to the notification rule anymore, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.Tag.ValueString(), ruleUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NotificationRuleTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Since both tag and notification_rule have RequiresReplace, this should never be called
	resp.Diagnostics.AddError(
		"Unexpected Update Call",
		"Notification rule tag assignments cannot be updated. Both tag and notification_rule changes require replacement.",
	)
}

func (r *NotificationRuleTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationRuleTagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ruleUUID, err := uuid.Parse(data.NotificationRule.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Notification Rule UUID", fmt.Sprintf("Unable to parse notification rule UUID: %s", err))
		return
	}

	tag := strings.ToLower(data.Tag.ValueString())

	err = r.data.Client.Tag.UntagNotificationRules(ctx, tag, []uuid.UUID{ruleUUID})
	if err != nil {
		// The tag or rule being gone means there is nothing left to delete.
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to untag notification rule, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a notification rule tag resource")
}

func (r *NotificationRuleTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: tag_name/notification_rule_uuid
	tag, ruleID, err := parseCompositeID(req.ID, "tag_name", "notification_rule_uuid")
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	ruleUUID, err := uuid.Parse(ruleID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Notification Rule UUID",
			fmt.Sprintf("Unable to parse notification rule UUID from import ID: %s\nError: %s", ruleID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag"), tag)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("notification_rule"), ruleUUID.String())...)
}
