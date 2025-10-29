package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TeamAPIKeyResource{}
var _ resource.ResourceWithImportState = &TeamAPIKeyResource{}

func NewTeamAPIKeyResource() resource.Resource {
	return &TeamAPIKeyResource{}
}

// TeamAPIKeyResource defines the resource implementation.
type TeamAPIKeyResource struct {
	client *dtrack.Client
}

// TeamAPIKeyResourceModel describes the resource data model.
type TeamAPIKeyResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Team      types.String `tfsdk:"team"`
	Key       types.String `tfsdk:"key"`
	Comment   types.String `tfsdk:"comment"`
	MaskedKey types.String `tfsdk:"masked_key"`
}

func (r *TeamAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_api_key"
}

func (r *TeamAPIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an API key for a Dependency-Track team. The actual API key value is only available upon creation and cannot be retrieved later.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The public ID of the API key",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the team this API key belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The API key value. This is only available upon creation and cannot be retrieved later.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Comment or description for the API key (max 255 characters)",
			},
			"masked_key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The masked version of the API key",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TeamAPIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

func (r *TeamAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Generate the API key
	apiKey, err := r.client.Team.GenerateAPIKey(ctx, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to generate API key, got error: %s", err))
		return
	}

	// Set the values from the created API key
	data.ID = types.StringValue(apiKey.PublicId)
	data.Key = types.StringValue(apiKey.Key)
	data.MaskedKey = types.StringValue(apiKey.MaskedKey)

	// Update the comment if provided
	if !data.Comment.IsNull() && data.Comment.ValueString() != "" {
		_, err = r.client.Team.UpdateAPIKeyComment(ctx, apiKey.PublicId, data.Comment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update API key comment, got error: %s", err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	teamUUID, err := uuid.Parse(data.Team.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Team UUID", fmt.Sprintf("Unable to parse team UUID: %s", err))
		return
	}

	// Get all API keys for the team
	apiKeys, err := r.client.Team.GetAPIKeys(ctx, teamUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team API keys, got error: %s", err))
		return
	}

	// Find the API key by public ID
	var found bool
	for _, key := range apiKeys {
		if key.PublicId == data.ID.ValueString() {
			found = true
			// Update the comment and masked key from the API
			if key.Comment != "" {
				data.Comment = types.StringValue(key.Comment)
			} else {
				data.Comment = types.StringNull()
			}
			data.MaskedKey = types.StringValue(key.MaskedKey)
			// Note: The actual key is not returned by the API after creation
			break
		}
	}

	if !found {
		// API key no longer exists, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TeamAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Only the comment can be updated
	if !plan.Comment.Equal(state.Comment) {
		_, err := r.client.Team.UpdateAPIKeyComment(ctx, state.ID.ValueString(), plan.Comment.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update API key comment, got error: %s", err))
			return
		}
	}

	// Preserve the state values
	plan.ID = state.ID
	plan.Team = state.Team
	plan.Key = state.Key
	plan.MaskedKey = state.MaskedKey

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TeamAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TeamAPIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Team.DeleteAPIKey(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete API key, got error: %s", err))
		return
	}
}

func (r *TeamAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using format: team_uuid:public_id
	resp.Diagnostics.AddError(
		"Import Not Supported",
		"Importing team API keys is not supported because the actual API key value is only available upon creation and cannot be retrieved later. Please recreate the resource instead.",
	)
}
