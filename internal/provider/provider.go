package provider

import (
	"context"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Data contains the client and API configuration.
type Data struct {
	Client      *dtrack.Client
	Endpoint    string
	ApiKey      string
	BearerToken string
}

// Ensure DependencyTrackProvider satisfies various provider interfaces.
var _ provider.Provider = &DependencyTrackProvider{}
var _ provider.ProviderWithFunctions = &DependencyTrackProvider{}
var _ provider.ProviderWithEphemeralResources = &DependencyTrackProvider{}

// DependencyTrackProvider defines the provider implementation.
type DependencyTrackProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// DependencyTrackProviderModel describes the provider data model.
type DependencyTrackProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	ApiKey   types.String `tfsdk:"api_key"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *DependencyTrackProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dependencytrack"
	resp.Version = p.version
}

func (p *DependencyTrackProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for OWASP Dependency-Track.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The URL of the Dependency-Track server (e.g., https://dtrack.example.com)",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for authenticating with Dependency-Track. Conflicts with username/password authentication.",
				Optional:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for authenticating with Dependency-Track. Must be used with password. Conflicts with api_key authentication.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for authenticating with Dependency-Track. Must be used with username. Conflicts with api_key authentication.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *DependencyTrackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data DependencyTrackProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate required configuration
	if data.Endpoint.IsNull() || data.Endpoint.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing Endpoint Configuration",
			"The provider requires an endpoint URL to be configured. "+
				"Set the endpoint attribute in the provider configuration.",
		)
		return
	}

	// Validate authentication configuration
	hasApiKey := !data.ApiKey.IsNull() && data.ApiKey.ValueString() != ""
	hasUsername := !data.Username.IsNull() && data.Username.ValueString() != ""
	hasPassword := !data.Password.IsNull() && data.Password.ValueString() != ""

	// Check for mutually exclusive authentication methods
	if hasApiKey && (hasUsername || hasPassword) {
		resp.Diagnostics.AddError(
			"Conflicting Authentication Configuration",
			"api_key and username/password authentication are mutually exclusive. "+
				"Please provide either an api_key OR username and password, not both.",
		)
		return
	}

	// Check that either api_key or username/password is provided
	if !hasApiKey && (!hasUsername || !hasPassword) {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"The provider requires authentication credentials. "+
				"Provide either:\n"+
				"  - api_key for API key authentication, OR\n"+
				"  - username AND password for username/password authentication",
		)
		return
	}

	// Check that username and password are provided together
	if hasUsername && !hasPassword {
		resp.Diagnostics.AddError(
			"Incomplete Authentication Configuration",
			"username requires password to be set as well.",
		)
		return
	}

	if hasPassword && !hasUsername {
		resp.Diagnostics.AddError(
			"Incomplete Authentication Configuration",
			"password requires username to be set as well.",
		)
		return
	}

	var client *dtrack.Client
	var apiKey string
	var bearerToken string
	var err error

	// Create DependencyTrack client based on authentication method
	if hasApiKey {
		// Use API key authentication
		apiKey = data.ApiKey.ValueString()
		client, err = dtrack.NewClient(data.Endpoint.ValueString(), dtrack.WithAPIKey(apiKey))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Dependency-Track Client",
				"An unexpected error occurred when creating the Dependency-Track client with API key. "+
					"Error: "+err.Error(),
			)
			return
		}
	} else {
		// Use username/password authentication - login to get the bearer token
		// Create a temporary client to perform the login
		tempClient, err := dtrack.NewClient(data.Endpoint.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Temporary Client",
				"An unexpected error occurred when creating a temporary Dependency-Track client for login. "+
					"Error: "+err.Error(),
			)
			return
		}

		// Use the client library's login method
		bearerToken, err = tempClient.User.Login(ctx, data.Username.ValueString(), data.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Authentication Failed",
				"Unable to authenticate with username and password. "+
					"Error: "+err.Error(),
			)
			return
		}

		// Create an authenticated client with the bearer token
		client, err = dtrack.NewClient(data.Endpoint.ValueString(), dtrack.WithBearerToken(bearerToken))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create Dependency-Track Client",
				"An unexpected error occurred when creating the Dependency-Track client with bearer token. "+
					"Error: "+err.Error(),
			)
			return
		}
	}

	// Create provider data with client and API configuration
	providerData := &Data{
		Client:      client,
		Endpoint:    data.Endpoint.ValueString(),
		ApiKey:      apiKey,
		BearerToken: bearerToken,
	}

	// Make the provider data available to data sources and resources
	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *DependencyTrackProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTeamResource,
		NewManagedUserResource,
		NewConfigPropertyResource,
		NewProjectResource,
		NewTeamPermissionsResource,
		NewManagedUserPermissionsResource,
		NewPolicyResource,
		NewACLMappingResource,
		NewTeamAPIKeyResource,
	}
}

func (p *DependencyTrackProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *DependencyTrackProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewTeamDataSource,
		NewManagedUserDataSource,
		NewConfigPropertyDataSource,
		NewProjectDataSource,
		NewPolicyDataSource,
		NewTeamAPIKeysDataSource,
	}
}

func (p *DependencyTrackProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DependencyTrackProvider{
			version: version,
		}
	}
}
