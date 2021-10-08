package xsoar

import (
	"context"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
)

type resourceIntegrationInstanceType struct{}

// GetSchema Resource schema
func (r resourceIntegrationInstanceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	var planModifiers []tfsdk.AttributePlanModifier
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Type:     types.StringType,
				Required: true,
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"integration_name": {
				Type:          types.StringType,
				Required:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			"config": {
				Type:     types.MapType{ElemType: types.StringType},
				Required: true,
			},
			"propagation_labels": {
				Type:     types.ListType{ElemType: types.StringType},
				Optional: true,
			},
			"account": {
				Type:          types.StringType,
				Optional:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
		},
	}, nil
}

// NewResource instance
func (r resourceIntegrationInstanceType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceIntegrationInstance{
		p: *(p.(*provider)),
	}, nil
}

type resourceIntegrationInstance struct {
	p provider
}

// Create a new resource
func (r resourceIntegrationInstance) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Retrieve values from plan
	var plan IntegrationInstance
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create
	// list integrations
	size := *openapi.NewInlineObject1()
	size.SetSize(500)
	integrations, _, err := r.p.client.DefaultApi.ListIntegrations(context.Background()).Size(size).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing integration",
			"Could not list integrations: "+err.Error(),
		)
		return
	}
	var moduleConfiguration []interface{}
	var moduleInstance = make(map[string]interface{})
	configurations := integrations["configurations"].([]interface{})
	for _, configuration := range configurations {
		if config := configuration.(map[string]interface{}); config["name"].(string) == plan.IntegrationName.Value {
			moduleConfiguration = config["configuration"].([]interface{})
			moduleInstance["brand"] = config["name"].(string)
			moduleInstance["canSample"] = false
			moduleInstance["category"] = config["category"].(string)
			moduleInstance["configuration"] = configuration
			moduleInstance["data"] = make([]map[string]interface{}, 0)
			moduleInstance["defultIgnore"] = false
			moduleInstance["enabled"] = "true"
			moduleInstance["engine"] = "" // todo: add this as a config option
			moduleInstance["engineGroup"] = ""
			moduleInstance["id"] = ""
			moduleInstance["incomingMapperId"] = ""
			moduleInstance["integrationLogLevel"] = ""
			moduleInstance["isIntegrationScript"] = false // todo: add this as a config option (byoi)
			moduleInstance["isLongRunning"] = false
			moduleInstance["mappingId"] = ""
			moduleInstance["name"] = plan.Name.Value
			moduleInstance["outgoingMapperId"] = ""
			moduleInstance["passwordProtected"] = false
			moduleInstance["propagationLabels"] = plan.PropagationLabels
			moduleInstance["resetContext"] = false
			moduleInstance["version"] = -1
			break
		}
	}
	for _, parameter := range moduleConfiguration {
		param := parameter.(map[string]interface{})
		param["hasvalue"] = false
		for configName, configValue := range plan.Config.Elems {
			if param["display"].(string) == configName || param["name"].(string) == configName {
				param["value"], _ = configValue.ToTerraformValue(context.Background())
				param["hasvalue"] = true
				break
			}
		}
		if !param["hasvalue"].(bool) {
			param["value"] = param["defaultValue"].(string)
		}
		moduleInstance["data"] = append(moduleInstance["data"].([]map[string]interface{}), param)
	}

	var integrationsResponse map[string]interface{}
	if plan.Account.Null || len(plan.Account.Value) == 0 {
		integrationsResponse, _, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstance(context.Background()).CreateIntegrationRequest(moduleInstance).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating integration instance",
				"Could not create integration instance: "+err.Error(),
			)
			return
		}
	} else {
		integrationsResponse, _, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstanceAccount(context.Background(), "acc_"+plan.Account.Value).CreateIntegrationRequest(moduleInstance).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating integration instance",
				"Could not create integration instance: "+err.Error(),
			)
			return
		}
	}
	var propLabels []string
	for _, label := range integrationsResponse["propagationLabels"].([]interface{}) {
		propLabels = append(propLabels, label.(string))
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integrationsResponse["name"].(string)},
		Id:                types.String{Value: integrationsResponse["id"].(string)},
		IntegrationName:   types.String{Value: integrationsResponse["brand"].(string)},
		Account:           plan.Account,
		PropagationLabels: propLabels,
		Config:            plan.Config,
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceIntegrationInstance) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Get current state
	var state IntegrationInstance
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from API
	var integrationInstance map[string]interface{}
	var err error
	if state.Account.Null || len(state.Account.Value) == 0 {
		integrationInstance, _, err = r.p.client.DefaultApi.GetIntegrationInstance(context.Background()).Identifier(state.Id.Value).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting integration instance",
				"Could not get integration instance: "+err.Error(),
			)
			return
		}
	} else {
		integrationInstance, _, err = r.p.client.DefaultApi.GetIntegrationInstanceAccount(context.Background(), "acc_"+state.Account.Value).Identifier(state.Id.Value).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting integration instance",
				"Could not get integration instance: "+err.Error(),
			)
			return
		}
	}
	var propLabels []string
	for _, label := range integrationInstance["propagationLabels"].([]interface{}) {
		propLabels = append(propLabels, label.(string))
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integrationInstance["name"].(string)},
		Id:                types.String{Value: integrationInstance["id"].(string)},
		IntegrationName:   types.String{Value: integrationInstance["brand"].(string)},
		Account:           state.Account,
		PropagationLabels: propLabels,
		Config:            state.Config,
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r resourceIntegrationInstance) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan values
	var plan IntegrationInstance
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state IntegrationInstance
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build request
	// list integrations
	size := *openapi.NewInlineObject1()
	size.SetSize(500)
	integrations, _, err := r.p.client.DefaultApi.ListIntegrations(context.Background()).Size(size).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing integration",
			"Could not list integrations: "+err.Error(),
		)
		return
	}
	var moduleConfiguration []interface{}
	var moduleInstance = make(map[string]interface{})
	configurations := integrations["configurations"].([]interface{})
	for _, configuration := range configurations {
		if config := configuration.(map[string]interface{}); config["name"].(string) == plan.IntegrationName.Value {
			moduleConfiguration = config["configuration"].([]interface{})
			moduleInstance["brand"] = config["name"].(string)
			moduleInstance["canSample"] = false
			moduleInstance["category"] = config["category"].(string)
			moduleInstance["configuration"] = configuration
			moduleInstance["data"] = make([]map[string]interface{}, 0)
			moduleInstance["defultIgnore"] = false
			moduleInstance["enabled"] = "true"
			moduleInstance["engine"] = "" // todo: add this as a config option
			moduleInstance["engineGroup"] = ""
			moduleInstance["id"] = state.Id.Value
			moduleInstance["incomingMapperId"] = ""
			moduleInstance["integrationLogLevel"] = ""
			moduleInstance["isIntegrationScript"] = false // todo: add this as a config option (byoi)
			moduleInstance["isLongRunning"] = false
			moduleInstance["mappingId"] = ""
			moduleInstance["name"] = plan.Name.Value
			moduleInstance["outgoingMapperId"] = ""
			moduleInstance["passwordProtected"] = false
			moduleInstance["propagationLabels"] = plan.PropagationLabels
			moduleInstance["resetContext"] = false
			moduleInstance["version"] = -1
			break
		}
	}
	for _, parameter := range moduleConfiguration {
		param := parameter.(map[string]interface{})
		param["hasvalue"] = false
		for configName, configValue := range plan.Config.Elems {
			if param["display"].(string) == configName || param["name"].(string) == configName {
				param["value"], _ = configValue.ToTerraformValue(context.Background())
				param["hasvalue"] = true
				break
			}
		}
		if !param["hasvalue"].(bool) {
			param["value"] = param["defaultValue"].(string)
		}
		moduleInstance["data"] = append(moduleInstance["data"].([]map[string]interface{}), param)
	}
	var integrationsResponse map[string]interface{}
	if state.Account.Null || len(state.Account.Value) == 0 {
		integrationsResponse, _, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstance(context.Background()).CreateIntegrationRequest(moduleInstance).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating integration instance",
				"Could not create integration instance: "+err.Error(),
			)
			return
		}
	} else {
		integrationsResponse, _, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstanceAccount(context.Background(), "acc_"+plan.Account.Value).CreateIntegrationRequest(moduleInstance).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating integration instance",
				"Could not create integration instance: "+err.Error(),
			)
			return
		}
	}
	var propLabels []string
	for _, label := range integrationsResponse["propagationLabels"].([]interface{}) {
		propLabels = append(propLabels, label.(string))
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integrationsResponse["name"].(string)},
		Id:                types.String{Value: integrationsResponse["id"].(string)},
		IntegrationName:   types.String{Value: integrationsResponse["brand"].(string)},
		Account:           plan.Account,
		PropagationLabels: propLabels,
		Config:            plan.Config,
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceIntegrationInstance) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	// Get state
	var state IntegrationInstance
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete
	if state.Account.Null || len(state.Account.Value) == 0 {
		_, err := r.p.client.DefaultApi.DeleteIntegrationInstance(context.Background(), state.Id.Value).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating integration instance",
				"Could not create integration instance: "+err.Error(),
			)
			return
		}
	} else {
		_, err := r.p.client.DefaultApi.DeleteIntegrationInstanceAccount(context.Background(), state.Id.Value, "acc_"+state.Account.Value).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating integration instance",
				"Could not create integration instance: "+err.Error(),
			)
			return
		}
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r resourceIntegrationInstance) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	accname := strings.Split(req.ID, ".")
	var acc, name string
	var integrationInstance map[string]interface{}
	var err error
	if len(accname) == 1 {
		name = req.ID
		integrationInstance, _, err = r.p.client.DefaultApi.GetIntegrationInstance(context.Background()).Identifier(name).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting integration instance",
				"Could not get integration instance: "+err.Error(),
			)
			return
		}
		if integrationInstance == nil {
			resp.Diagnostics.AddError(
				"Integration instance not found",
				"Could not find integration instance: "+name,
			)
			return
		}
	} else {
		acc, name = accname[0], accname[1]
		integrationInstance, _, err = r.p.client.DefaultApi.GetIntegrationInstanceAccount(context.Background(), "acc_"+acc).Identifier(name).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting integration instance",
				"Could not get integration instance: "+err.Error(),
			)
			return
		}
		if integrationInstance == nil {
			resp.Diagnostics.AddError(
				"Integration instance not found",
				"Could not find integration instance: "+name,
			)
			return
		}
	}
	var propLabels []string
	for _, label := range integrationInstance["propagationLabels"].([]interface{}) {
		propLabels = append(propLabels, label.(string))
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integrationInstance["name"].(string)},
		Id:                types.String{Value: integrationInstance["id"].(string)},
		IntegrationName:   types.String{Value: integrationInstance["brand"].(string)},
		PropagationLabels: propLabels,
		Config:            types.Map{},
	}

	if acc != "" {
		result.Account = types.String{Value: acc}
	} else {
		result.Account = types.String{Null: true}
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
