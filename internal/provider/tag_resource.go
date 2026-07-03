package provider

import (
	"context"
	"fmt"
	"strings"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TagResource{}
var _ resource.ResourceWithImportState = &TagResource{}

func NewTagResource() resource.Resource {
	return &TagResource{}
}

// TagResource defines the resource implementation.
type TagResource struct {
	data *Data
}

// TagResourceModel describes the resource data model.
type TagResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (r *TagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *TagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a tag in Dependency-Track. Tags have no UUID; the tag name is its identifier.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The identifier of the tag (same as its name)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the tag. Dependency-Track normalizes tag names to lowercase, so a mixed-case name is matched case-insensitively (using a lowercase name is recommended). Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *TagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	err := r.data.Client.Tag.Create(ctx, []string{name})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create tag, got error: %s", err))
		return
	}

	data.ID = types.StringValue(name)

	tflog.Trace(ctx, "created a tag resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	found, err := r.tagExists(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tags, got error: %s", err))
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(name)
	data.Name = types.StringValue(name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is a no-op: the only configurable attribute (name) forces
// replacement, so this method is never invoked in practice. It is implemented
// only to satisfy the resource.Resource interface.
func (r *TagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TagResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TagResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Dependency-Track stores tag names lowercased and its delete endpoint
	// requires the exact stored name, so normalize before deleting.
	err := r.data.Client.Tag.Delete(ctx, []string{strings.ToLower(data.Name.ValueString())})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tag, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a tag resource")
}

func (r *TagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the tag name.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

// tagExists reports whether a tag with the given name exists, paging through
// the full tag list.
func (r *TagResource) tagExists(ctx context.Context, name string) (bool, error) {
	tags, err := fetchAllPages(ctx, func(ctx context.Context, po dtrack.PageOptions) (dtrack.Page[dtrack.TagListResponseItem], error) {
		return r.data.Client.Tag.GetAll(ctx, po, dtrack.SortOptions{})
	})
	if err != nil {
		return false, err
	}

	for i := range tags {
		// Dependency-Track lowercases tag names on write; match
		// case-insensitively so a mixed-case configured name still resolves.
		if strings.EqualFold(tags[i].Name, name) {
			return true, nil
		}
	}

	return false, nil
}
