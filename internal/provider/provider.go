package provider

import (
	"context"
	"fmt"

	dtrack "github.com/DependencyTrack/client-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Data contains the client and API configuration.
type Data struct {
	Client        *dtrack.Client
	Endpoint      string
	ApiKey        string
	BearerToken   string
	ServerVersion ServerVersion
	api           *apiClient
}

// IsV5 reports whether the configured Dependency-Track server is running
// version 5.x or newer.
func (d *Data) IsV5() bool {
	return d.ServerVersion.IsV5()
}

// API returns the shared HTTP client used by resources/data sources that
// call Dependency-Track endpoints not covered by client-go's typed methods.
func (d *Data) API() *apiClient {
	return d.api
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
	Endpoint         types.String           `tfsdk:"endpoint"`
	ApiKey           types.String           `tfsdk:"api_key"`
	Username         types.String           `tfsdk:"username"`
	Password         types.String           `tfsdk:"password"`
	WorkloadIdentity *WorkloadIdentityModel `tfsdk:"workload_identity"`
}

// WorkloadIdentityModel describes the provider's workload_identity block.
type WorkloadIdentityModel struct {
	ProviderName   types.String `tfsdk:"provider_name"`
	ServiceAccount types.String `tfsdk:"service_account"`
	Token          types.String `tfsdk:"token"`
}

func (p *DependencyTrackProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dependencytrack"
	resp.Version = p.version
}

func (p *DependencyTrackProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for [OWASP Dependency-Track](https://dependencytrack.org/). " +
			"It supports both Dependency-Track v4 (tested against 4.14.x, until its upstream end-of-life in December 2026) and v5 (tested against the latest minor, currently 5.2); end-of-life Dependency-Track versions get best-effort support only. " +
			"At configure time the provider queries the unauthenticated `GET /api/version` endpoint to detect the server's major version and automatically adapts version-dependent behavior (for example, notification publisher identifiers and the deprecated project `author` field); there is no version attribute to set. " +
			"If that probe fails, provider configuration fails with an actionable error instead of silently guessing a version. " +
			"Authenticate with exactly one of: an `api_key`, a `username`/`password` pair, or a `workload_identity` block (Dependency-Track 5.2+).",
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
			"workload_identity": schema.SingleNestedAttribute{
				MarkdownDescription: "Authenticate as a service account through workload identity federation (Dependency-Track 5.2+): " +
					"the provider exchanges a workload's identity token, such as a GitHub Actions OIDC token, for a Dependency-Track session " +
					"at configure time. The service account needs a `dependencytrack_workload_identity_binding` that matches the token. " +
					"The session lasts the workload identity provider's `session_lifetime_seconds` (1 hour by default) and is not refreshed, " +
					"so a run that takes longer fails once it expires. Conflicts with `api_key` and `username`/`password`.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"provider_name": schema.StringAttribute{
						MarkdownDescription: "The name of the Dependency-Track workload identity provider that issued `token`",
						Required:            true,
					},
					"service_account": schema.StringAttribute{
						MarkdownDescription: "The name of the service account to authenticate as (without the `svc:` prefix)",
						Required:            true,
					},
					"token": schema.StringAttribute{
						MarkdownDescription: "The workload's identity token (a JWT). Read it from a file with `file()` if the platform provides one",
						Required:            true,
						Sensitive:           true,
					},
				},
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
	hasWorkloadIdentity := data.WorkloadIdentity != nil

	// Check for mutually exclusive authentication methods
	if hasApiKey && (hasUsername || hasPassword) {
		resp.Diagnostics.AddError(
			"Conflicting Authentication Configuration",
			"api_key and username/password authentication are mutually exclusive. "+
				"Please provide either an api_key OR username and password, not both.",
		)
		return
	}

	if hasWorkloadIdentity && (hasApiKey || hasUsername || hasPassword) {
		resp.Diagnostics.AddAttributeError(
			path.Root("workload_identity"),
			"Conflicting Authentication Configuration",
			"workload_identity cannot be combined with api_key or username/password authentication.",
		)
		return
	}

	// Check that an authentication method is provided
	if !hasWorkloadIdentity && !hasApiKey && (!hasUsername || !hasPassword) {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"The provider requires authentication credentials. "+
				"Provide one of:\n"+
				"  - api_key for API key authentication,\n"+
				"  - username AND password for username/password authentication, OR\n"+
				"  - a workload_identity block for workload identity federation (Dependency-Track 5.2+)",
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

	endpoint := data.Endpoint.ValueString()

	// Detect the Dependency-Track server version via the unauthenticated
	// GET /api/version endpoint (exposed by both v4 and v5) before
	// authenticating, since workload identity federation requires 5.2+. This
	// is required: there is no silent fallback.
	probeClient, err := dtrack.NewClient(endpoint)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Dependency-Track Client",
			"An unexpected error occurred when creating the Dependency-Track client. Error: "+err.Error(),
		)
		return
	}

	about, err := probeClient.About.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Detect Dependency-Track Server Version",
			"The provider requires GET "+endpoint+"/api/version to be reachable. "+
				"Check the endpoint configuration and any proxies between Terraform and the Dependency-Track server. "+
				"Error: "+err.Error(),
		)
		return
	}

	serverVersion, err := parseServerVersion(about.Version)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Detect Dependency-Track Server Version",
			fmt.Sprintf(
				"The provider requires GET %s/api/version to return a valid Dependency-Track version, "+
					"but got %q which could not be parsed. Check the endpoint configuration and any proxies "+
					"between Terraform and the Dependency-Track server. Error: %s",
				endpoint, about.Version, err,
			),
		)
		return
	}

	var apiKey string
	var bearerToken string
	var clientOption dtrack.ClientOption

	switch {
	case hasApiKey:
		apiKey = data.ApiKey.ValueString()
		clientOption = dtrack.WithAPIKey(apiKey)

	case hasWorkloadIdentity:
		if !requireAtLeast(&Data{ServerVersion: serverVersion}, 5, 2, "Workload identity federation", &resp.Diagnostics) {
			return
		}

		wi := data.WorkloadIdentity
		var expiresIn int
		bearerToken, expiresIn, err = exchangeWorkloadIdentityToken(ctx, endpoint,
			wi.ProviderName.ValueString(), wi.ServiceAccount.ValueString(), wi.Token.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("workload_identity"),
				"Authentication Failed",
				"Unable to exchange the workload identity token for a Dependency-Track session. "+
					"Check that the workload identity provider and service account exist and that a binding of the "+
					"service account matches the token. Error: "+err.Error(),
			)
			return
		}
		tflog.Info(ctx, "authenticated through workload identity federation", map[string]interface{}{
			"service_account":    wi.ServiceAccount.ValueString(),
			"expires_in_seconds": expiresIn,
		})
		clientOption = dtrack.WithBearerToken(bearerToken)

	default:
		// Username/password: log in to get a bearer token.
		bearerToken, err = probeClient.User.Login(ctx, data.Username.ValueString(), data.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Authentication Failed",
				"Unable to authenticate with username and password. "+
					"Error: "+err.Error(),
			)
			return
		}
		clientOption = dtrack.WithBearerToken(bearerToken)
	}

	client, err := dtrack.NewClient(endpoint, clientOption)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Dependency-Track Client",
			"An unexpected error occurred when creating the authenticated Dependency-Track client. "+
				"Error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "detected Dependency-Track server version", map[string]interface{}{
		"version": serverVersion.Raw,
		"major":   serverVersion.Major,
	})

	// Create provider data with client and API configuration
	providerData := &Data{
		Client:        client,
		Endpoint:      endpoint,
		ApiKey:        apiKey,
		BearerToken:   bearerToken,
		ServerVersion: serverVersion,
		api:           newAPIClient(endpoint, apiKey, bearerToken),
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
		NewUserTeamMembershipResource,
		NewProjectPolicyResource,
		NewNotificationPublisherResource,
		NewNotificationRuleResource,
		NewNotificationRuleProjectResource,
		NewNotificationRuleTeamResource,
		NewRepositoryResource,
		NewOIDCGroupResource,
		NewTagResource,
		NewLicenseGroupResource,
		NewProjectPropertyResource,
		NewOIDCGroupMappingResource,
		NewLDAPMappingResource,
		NewLicenseResource,
		NewLicenseGroupLicenseResource,
		NewPolicyTagResource,
		NewNotificationRuleTagResource,
		NewSecretResource,
		NewExtensionConfigResource,
		NewServiceAccountResource,
		NewServiceAccountAPIKeyResource,
		NewWorkloadIdentityProviderResource,
		NewWorkloadIdentityBindingResource,
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
		NewNotificationPublisherDataSource,
		NewRepositoriesDataSource,
		NewOIDCGroupDataSource,
		NewTagsDataSource,
		NewLicenseGroupDataSource,
		NewLicenseDataSource,
		NewLicensesDataSource,
		NewPortfolioMetricsDataSource,
		NewProjectMetricsDataSource,
		NewProjectViolationsDataSource,
		NewProjectFindingsDataSource,
		NewServiceAccountDataSource,
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
