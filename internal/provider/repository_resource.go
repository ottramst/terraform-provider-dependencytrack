package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// repositoryTypes are the RepositoryType enum values accepted by
// Dependency-Track, mirroring the constants exported by client-go
// (UNSUPPORTED is intentionally omitted: it is a sentinel the server returns
// for unknown types, not a type a repository can be created with).
var repositoryTypes = []string{
	dtrack.RepositoryTypeCargo,
	dtrack.RepositoryTypeComposer,
	dtrack.RepositoryTypeCpan,
	dtrack.RepositoryTypeGem,
	dtrack.RepositoryTypeGithub,
	dtrack.RepositoryTypeGoModules,
	dtrack.RepositoryTypeHex,
	dtrack.RepositoryTypeMaven,
	dtrack.RepositoryTypeNpm,
	dtrack.RepositoryTypeNuget,
	dtrack.RepositoryTypePypi,
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RepositoryResource{}
var _ resource.ResourceWithImportState = &RepositoryResource{}

func NewRepositoryResource() resource.Resource {
	return &RepositoryResource{}
}

// RepositoryResource defines the resource implementation.
type RepositoryResource struct {
	data *Data
}

// RepositoryResourceModel describes the resource data model.
type RepositoryResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Type                   types.String `tfsdk:"type"`
	Identifier             types.String `tfsdk:"identifier"`
	URL                    types.String `tfsdk:"url"`
	ResolutionOrder        types.Int64  `tfsdk:"resolution_order"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	Internal               types.Bool   `tfsdk:"internal"`
	AuthenticationRequired types.Bool   `tfsdk:"authentication_required"`
	Username               types.String `tfsdk:"username"`
	Password               types.String `tfsdk:"password"`
}

func (r *RepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *RepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a component repository in Dependency-Track.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the repository",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of the repository. One of: CARGO, COMPOSER, CPAN, GEM, GITHUB, GO_MODULES, HEX, MAVEN, NPM, NUGET, PYPI. Changing this forces a new resource to be created.",
				Validators: []validator.String{
					stringvalidator.OneOf(repositoryTypes...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"identifier": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier of the repository (unique per repository type)",
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The URL of the repository",
			},
			"resolution_order": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The resolution order of the repository. This is assigned and managed by Dependency-Track (based on creation sequence per type) and cannot be set; any value supplied in a request is ignored by the server.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the repository is enabled",
			},
			"internal": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the repository is internal",
			},
			"authentication_required": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether authentication is required to access the repository",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The username used to authenticate with the repository",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The password used to authenticate with the repository. Dependency-Track never returns this value, so it is preserved from configuration/state and is not refreshed on read. Note: on Dependency-Track v5 this must be the name of an existing secret rather than a literal password; on v4 it is the literal password.",
			},
		},
	}
}

func (r *RepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// resolution_order is computed: Dependency-Track assigns it and ignores any
	// value supplied in the request, so it is intentionally not sent here.
	repo := dtrack.Repository{
		Type:                   dtrack.RepositoryType(data.Type.ValueString()),
		Identifier:             data.Identifier.ValueString(),
		Url:                    data.URL.ValueString(),
		Enabled:                data.Enabled.ValueBool(),
		Internal:               data.Internal.ValueBool(),
		AuthenticationRequired: data.AuthenticationRequired.ValueBool(),
		Username:               data.Username.ValueString(),
		Password:               data.Password.ValueString(),
	}

	createdRepo, err := r.data.Client.Repository.Create(ctx, repo)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create repository, got error: %s", err))
		return
	}

	r.updateModelFromRepository(&data, createdRepo)

	tflog.Trace(ctx, "created a repository resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	repoUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse repository UUID: %s", err))
		return
	}

	// There is no get-by-uuid endpoint for repositories, so list them all and
	// match by UUID.
	repo, found, err := r.findRepository(ctx, repoUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read repository, got error: %s", err))
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Password is never returned by the API; preserve the prior state value.
	r.updateModelFromRepository(&data, repo)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	repoUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse repository UUID: %s", err))
		return
	}

	// resolution_order is computed and server-managed (see Create); not sent.
	repo := dtrack.Repository{
		UUID:                   repoUUID,
		Type:                   dtrack.RepositoryType(data.Type.ValueString()),
		Identifier:             data.Identifier.ValueString(),
		Url:                    data.URL.ValueString(),
		Enabled:                data.Enabled.ValueBool(),
		Internal:               data.Internal.ValueBool(),
		AuthenticationRequired: data.AuthenticationRequired.ValueBool(),
		Username:               data.Username.ValueString(),
		Password:               data.Password.ValueString(),
	}

	updatedRepo, err := r.data.Client.Repository.Update(ctx, repo)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update repository, got error: %s", err))
		return
	}

	r.updateModelFromRepository(&data, updatedRepo)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	repoUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Unable to parse repository UUID: %s", err))
		return
	}

	err = r.data.Client.Repository.Delete(ctx, repoUUID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete repository, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a repository resource")
}

func (r *RepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	repoUUID, err := uuid.Parse(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse UUID. Expected a valid UUID, got: %s\nError: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), repoUUID.String())...)
}

// updateModelFromRepository copies API-sourced fields into the model. The
// Username is normalized to null when empty (so an unauthenticated repository
// does not drift against a null configuration value), and the Password is left
// untouched because the API never returns it.
func (r *RepositoryResource) updateModelFromRepository(data *RepositoryResourceModel, repo dtrack.Repository) {
	data.ID = types.StringValue(repo.UUID.String())
	data.Type = types.StringValue(string(repo.Type))
	data.Identifier = types.StringValue(repo.Identifier)
	data.URL = types.StringValue(repo.Url)
	data.ResolutionOrder = types.Int64Value(int64(repo.ResolutionOrder))
	data.Enabled = types.BoolValue(repo.Enabled)
	data.Internal = types.BoolValue(repo.Internal)
	data.AuthenticationRequired = types.BoolValue(repo.AuthenticationRequired)

	if repo.Username != "" {
		data.Username = types.StringValue(repo.Username)
	} else {
		data.Username = types.StringNull()
	}
}

// findRepository lists all repositories and returns the one matching repoUUID.
func (r *RepositoryResource) findRepository(ctx context.Context, repoUUID uuid.UUID) (dtrack.Repository, bool, error) {
	repos, err := fetchAllPages(ctx, r.data.Client.Repository.GetAll)
	if err != nil {
		return dtrack.Repository{}, false, err
	}

	for i := range repos {
		if repos[i].UUID == repoUUID {
			return repos[i], true, nil
		}
	}

	return dtrack.Repository{}, false, nil
}
