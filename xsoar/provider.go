package xsoar

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"

	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ = os.Stderr

func New() tfsdk.Provider {
	return &provider{}
}

type provider struct {
	configured bool
	client     *openapi.APIClient
}

// GetSchema
func (p *provider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"main_host": {
				Type:     types.StringType,
				Required: true,
			},
			"api_key": {
				Type:     types.StringType,
				Required: true,
			},
			"insecure": {
				Type:     types.BoolType,
				Optional: true,
			},
		},
	}, nil
}

// Provider schema struct
type providerData struct {
	Apikey   types.String `tfsdk:"api_key"`
	MainHost types.String `tfsdk:"main_host"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	// Retrieve provider data from configuration
	var config providerData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// User must provide an api key to the provider
	var apikey string
	if config.Apikey.Unknown {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as API key",
		)
		return
	}

	if config.Apikey.Null {
		apikey = os.Getenv("DEMISTO_API_KEY")
	} else {
		apikey = config.Apikey.Value
	}

	if apikey == "" {
		// Error vs warning - empty value must stop execution
		resp.Diagnostics.AddError(
			"Unable to find API key",
			"API key cannot be an empty string",
		)
		return
	}

	// User must specify a host
	var mainhost string
	if config.MainHost.Unknown {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Cannot use unknown value as main host",
		)
		return
	}

	if config.MainHost.Null {
		mainhost = os.Getenv("DEMISTO_BASE_URL")
	} else {
		mainhost = config.MainHost.Value
	}

	if mainhost == "" {
		// Error vs warning - empty value must stop execution
		resp.Diagnostics.AddError(
			"Unable to find main host",
			"Main host cannot be an empty string",
		)
		return
	}

	// Create a new xsoar client and set it to the provider client
	openapiConfig := openapi.NewConfiguration()
	openapiConfig.Servers[0].URL = mainhost
	openapiConfig.AddDefaultHeader("Authorization", apikey)
	openapiConfig.AddDefaultHeader("Accept", "application/json,*/*")
	if config.Insecure.Value {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		openapiConfig.HTTPClient = client
	}
	c := openapi.NewAPIClient(openapiConfig)

	p.client = c
	p.configured = true
}

// GetResources - Defines provider resources
func (p *provider) GetResources(_ context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"xsoar_account":              resourceAccountType{},
		"xsoar_ha_group":             resourceHAGroupType{},
		"xsoar_host":                 resourceHostType{},
		"xsoar_integration_instance": resourceIntegrationInstanceType{},
	}, nil
}

// GetDataSources - Defines provider data sources
func (p *provider) GetDataSources(_ context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"xsoar_account":              dataSourceAccountType{},
		"xsoar_ha_group":             dataSourceHAGroupType{},
		"xsoar_host":                 dataSourceHostType{},
		"xsoar_integration_instance": dataSourceIntegrationInstanceType{},
	}, nil
}
