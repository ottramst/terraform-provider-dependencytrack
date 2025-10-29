package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource defines the resource implementation.
type ProjectResource struct {
	client *dtrack.Client
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Version     types.String `tfsdk:"version"`
	Description types.String `tfsdk:"description"`
	Group       types.String `tfsdk:"group"`
	Publisher   types.String `tfsdk:"publisher"`
	Author      types.String `tfsdk:"author"`
	Classifier  types.String `tfsdk:"classifier"`
	Active      types.Bool   `tfsdk:"active"`
	CPE         types.String `tfsdk:"cpe"`
	PURL        types.String `tfsdk:"purl"`
	SWIDTagID   types.String `tfsdk:"swid_tag_id"`
	ParentUUID  types.String `tfsdk:"parent_uuid"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Dependency-Track project.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the project",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the project",
			},
			"version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The version of the project",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The description of the project",
			},
			"group": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The group of the project",
			},
			"publisher": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The publisher of the project",
			},
			"author": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The author of the project",
			},
			"classifier": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The classifier of the project (APPLICATION, FRAMEWORK, LIBRARY, CONTAINER, OPERATING_SYSTEM, DEVICE, FIRMWARE, FILE, PLATFORM, DEVICE_DRIVER, MACHINE_LEARNING_MODEL, DATA)",
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the project is active",
			},
			"cpe": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The Common Platform Enumeration (CPE) of the project",
			},
			"purl": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The Package URL (PURL) of the project",
			},
			"swid_tag_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The SWID tag ID of the project",
			},
			"parent_uuid": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The UUID of the parent project",
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project := dtrack.Project{
		Name:        data.Name.ValueString(),
		Version:     data.Version.ValueString(),
		Description: data.Description.ValueString(),
		Group:       data.Group.ValueString(),
		Publisher:   data.Publisher.ValueString(),
		Author:      data.Author.ValueString(),
		Classifier:  data.Classifier.ValueString(),
		Active:      data.Active.ValueBool(),
		CPE:         data.CPE.ValueString(),
		PURL:        data.PURL.ValueString(),
		SWIDTagID:   data.SWIDTagID.ValueString(),
	}

	if !data.ParentUUID.IsNull() && !data.ParentUUID.IsUnknown() {
		parentUUID, err := uuid.Parse(data.ParentUUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Parent UUID", fmt.Sprintf("Unable to parse parent UUID: %s", err))
			return
		}
		project.ParentRef = &dtrack.ParentRef{UUID: parentUUID}
	}

	createdProject, err := r.client.Project.Create(ctx, project)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
		return
	}

	data.ID = types.StringValue(createdProject.UUID.String())
	data.Name = types.StringValue(createdProject.Name)
	data.Version = types.StringValue(createdProject.Version)
	data.Description = types.StringValue(createdProject.Description)
	data.Group = types.StringValue(createdProject.Group)
	data.Publisher = types.StringValue(createdProject.Publisher)
	data.Author = types.StringValue(createdProject.Author)
	data.Classifier = types.StringValue(createdProject.Classifier)
	data.Active = types.BoolValue(createdProject.Active)
	data.CPE = types.StringValue(createdProject.CPE)
	data.PURL = types.StringValue(createdProject.PURL)
	data.SWIDTagID = types.StringValue(createdProject.SWIDTagID)

	if createdProject.ParentRef != nil {
		data.ParentUUID = types.StringValue(createdProject.ParentRef.UUID.String())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	project, err := r.client.Project.Get(ctx, projectUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	// If project not found, remove from state
	if project.UUID == uuid.Nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.ID = types.StringValue(project.UUID.String())
	data.Name = types.StringValue(project.Name)
	data.Version = types.StringValue(project.Version)
	data.Description = types.StringValue(project.Description)
	data.Group = types.StringValue(project.Group)
	data.Publisher = types.StringValue(project.Publisher)
	data.Author = types.StringValue(project.Author)
	data.Classifier = types.StringValue(project.Classifier)
	data.Active = types.BoolValue(project.Active)
	data.CPE = types.StringValue(project.CPE)
	data.PURL = types.StringValue(project.PURL)
	data.SWIDTagID = types.StringValue(project.SWIDTagID)

	if project.ParentRef != nil {
		data.ParentUUID = types.StringValue(project.ParentRef.UUID.String())
	} else {
		data.ParentUUID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	project := dtrack.Project{
		UUID:        projectUUID,
		Name:        data.Name.ValueString(),
		Version:     data.Version.ValueString(),
		Description: data.Description.ValueString(),
		Group:       data.Group.ValueString(),
		Publisher:   data.Publisher.ValueString(),
		Author:      data.Author.ValueString(),
		Classifier:  data.Classifier.ValueString(),
		Active:      data.Active.ValueBool(),
		CPE:         data.CPE.ValueString(),
		PURL:        data.PURL.ValueString(),
		SWIDTagID:   data.SWIDTagID.ValueString(),
	}

	if !data.ParentUUID.IsNull() && !data.ParentUUID.IsUnknown() {
		parentUUID, err := uuid.Parse(data.ParentUUID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Parent UUID", fmt.Sprintf("Unable to parse parent UUID: %s", err))
			return
		}
		project.ParentRef = &dtrack.ParentRef{UUID: parentUUID}
	}

	updatedProject, err := r.client.Project.Update(ctx, project)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
		return
	}

	data.ID = types.StringValue(updatedProject.UUID.String())
	data.Name = types.StringValue(updatedProject.Name)
	data.Version = types.StringValue(updatedProject.Version)
	data.Description = types.StringValue(updatedProject.Description)
	data.Group = types.StringValue(updatedProject.Group)
	data.Publisher = types.StringValue(updatedProject.Publisher)
	data.Author = types.StringValue(updatedProject.Author)
	data.Classifier = types.StringValue(updatedProject.Classifier)
	data.Active = types.BoolValue(updatedProject.Active)
	data.CPE = types.StringValue(updatedProject.CPE)
	data.PURL = types.StringValue(updatedProject.PURL)
	data.SWIDTagID = types.StringValue(updatedProject.SWIDTagID)

	if updatedProject.ParentRef != nil {
		data.ParentUUID = types.StringValue(updatedProject.ParentRef.UUID.String())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse project UUID: %s", err))
		return
	}

	err = r.client.Project.Delete(ctx, projectUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using UUID
	projectUUID, err := uuid.Parse(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse UUID. Expected a valid UUID, got: %s\nError: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), projectUUID.String())...)
}
