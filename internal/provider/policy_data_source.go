package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PolicyDataSource{}

func NewPolicyDataSource() datasource.DataSource {
	return &PolicyDataSource{}
}

// PolicyDataSource defines the data source implementation.
type PolicyDataSource struct {
	client *dtrack.Client
}

// PolicyDataSourceModel describes the data source data model.
type PolicyDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Operator        types.String `tfsdk:"operator"`
	ViolationState  types.String `tfsdk:"violation_state"`
	IncludeChildren types.Bool   `tfsdk:"include_children"`
	Global          types.Bool   `tfsdk:"global"`
	Conditions      types.List   `tfsdk:"conditions"`
}

func (d *PolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (d *PolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a Dependency-Track policy.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the policy (same as uuid)",
				Required:            true,
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the policy",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the policy",
			},
			"operator": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The operator used when evaluating conditions (ALL or ANY)",
			},
			"violation_state": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The violation state (INFO, WARN, or FAIL)",
			},
			"include_children": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the policy applies to child projects",
			},
			"global": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is a global policy",
			},
			"conditions": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of policy conditions",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the condition",
						},
						"subject": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The subject of the condition",
						},
						"operator": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The operator for the condition",
						},
						"value": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The value to compare against",
						},
					},
				},
			},
		},
	}
}

func (d *PolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*Data)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = data.Client
}

func (d *PolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	policyUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse policy UUID: %s", err))
		return
	}

	policy, err := d.client.Policy.Get(ctx, policyUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read policy, got error: %s", err))
		return
	}

	// Check if policy exists
	if policy.UUID == uuid.Nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Policy with UUID %s not found", policyUUID))
		return
	}

	// Update model with values from API
	data.UUID = types.StringValue(policy.UUID.String())
	data.Name = types.StringValue(policy.Name)
	data.Operator = types.StringValue(string(policy.Operator))
	data.ViolationState = types.StringValue(string(policy.ViolationState))
	data.IncludeChildren = types.BoolValue(policy.IncludeChildren)
	data.Global = types.BoolValue(policy.Global)

	// Build conditions list
	if len(policy.PolicyConditions) > 0 {
		conditionElements := make([]attr.Value, 0, len(policy.PolicyConditions))
		for _, cond := range policy.PolicyConditions {
			conditionElements = append(conditionElements, types.ObjectValueMust(
				map[string]attr.Type{
					"uuid":     types.StringType,
					"subject":  types.StringType,
					"operator": types.StringType,
					"value":    types.StringType,
				},
				map[string]attr.Value{
					"uuid":     types.StringValue(cond.UUID.String()),
					"subject":  types.StringValue(string(cond.Subject)),
					"operator": types.StringValue(string(cond.Operator)),
					"value":    types.StringValue(cond.Value),
				},
			))
		}
		conditionsList, diags := types.ListValue(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"uuid":     types.StringType,
					"subject":  types.StringType,
					"operator": types.StringType,
					"value":    types.StringType,
				},
			},
			conditionElements,
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Conditions = conditionsList
	} else {
		data.Conditions = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"uuid":     types.StringType,
				"subject":  types.StringType,
				"operator": types.StringType,
				"value":    types.StringType,
			},
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
