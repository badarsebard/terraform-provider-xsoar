package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
	"log"
	"net/http"
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
	integrations, _, err := r.p.client.DefaultApi.ListIntegrations(ctx).Execute()
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
		config := configuration.(map[string]interface{})
		if config["name"].(string) == plan.IntegrationName.Value {
			moduleConfiguration = config["configuration"].([]interface{})
			moduleInstance["brand"] = config["name"].(string)
			moduleInstance["canSample"] = config["canGetSamples"].(bool)
			moduleInstance["category"] = config["category"].(string)
			moduleInstance["configuration"] = configuration
			moduleInstance["data"] = make([]map[string]interface{}, 0)
			moduleInstance["defaultIgnore"] = false
			moduleInstance["enabled"] = "true"
			// todo: add this as a config option
			//moduleInstance["engine"] = ""
			//moduleInstance["engineGroup"] = ""
			//moduleInstance["id"] = ""
			//moduleInstance["incomingMapperId"] = ""
			//moduleInstance["integrationLogLevel"] = ""
			// todo: add this as a config option (byoi)
			var isIntegrationScript bool
			if val, ok := config["integrationScript"]; ok && val != nil {
				isIntegrationScript = true
			}
			moduleInstance["isIntegrationScript"] = isIntegrationScript
			//moduleInstance["isLongRunning"] = false
			//moduleInstance["mappingId"] = ""
			moduleInstance["name"] = plan.Name.Value
			//moduleInstance["outgoingMapperId"] = ""
			//moduleInstance["passwordProtected"] = false
			var propLabels []string
			plan.PropagationLabels.ElementsAs(ctx, propLabels, false)
			moduleInstance["propagationLabels"] = propLabels
			//moduleInstance["resetContext"] = false
			moduleInstance["version"] = -1
			break
		}
	}
	for _, parameter := range moduleConfiguration {
		param := parameter.(map[string]interface{})
		param["hasvalue"] = false
		for configName, configValue := range plan.Config.Elems {
			if param["display"].(string) == configName || param["name"].(string) == configName {
				param["value"], _ = configValue.ToTerraformValue(ctx)
				param["hasvalue"] = true
				break
			}
		}
		moduleInstance["data"] = append(moduleInstance["data"].([]map[string]interface{}), param)
	}

	var integration map[string]interface{}
	var httpResponse *http.Response
	if plan.Account.Null || len(plan.Account.Value) == 0 {
		integration, httpResponse, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstance(ctx).CreateIntegrationRequest(moduleInstance).Execute()
	} else {
		integration, httpResponse, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstanceAccount(ctx, "acc_"+plan.Account.Value).CreateIntegrationRequest(moduleInstance).Execute()
	}
	getBody := httpResponse.Body
	b, _ := io.ReadAll(getBody)
	log.Println(string(b))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating integration instance",
			"Could not create integration instance: "+err.Error(),
		)
		return
	}

	var propagationLabels []attr.Value
	if integration["propagationLabels"] == nil {
		propagationLabels = []attr.Value{}
	} else {
		for _, prop := range integration["propagationLabels"].([]interface{}) {
			propagationLabels = append(propagationLabels, types.String{
				Unknown: false,
				Null:    false,
				Value:   prop.(string),
			})
		}
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		Account:           plan.Account,
		PropagationLabels: types.List{Elems: propagationLabels, ElemType: types.StringType},
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
	var integration map[string]interface{}
	var httpResponse *http.Response
	var err error
	if state.Account.Null || len(state.Account.Value) == 0 {
		integration, httpResponse, err = r.p.client.DefaultApi.GetIntegrationInstance(ctx).SetIdentifier(state.Id.Value).Execute()
	} else {
		integration, httpResponse, err = r.p.client.DefaultApi.GetIntegrationInstanceAccount(ctx, "acc_"+state.Account.Value).SetIdentifier(state.Id.Value).Execute()
	}
	if err != nil {
		getBody := httpResponse.Body
		b, _ := io.ReadAll(getBody)
		log.Println(string(b))
		resp.Diagnostics.AddError(
			"Error getting integration instance",
			"Could not get integration instance: "+err.Error(),
		)
		return
	}

	var propagationLabels []attr.Value
	if integration["propagationLabels"] == nil {
		propagationLabels = []attr.Value{}
	} else {
		for _, prop := range integration["propagationLabels"].([]interface{}) {
			propagationLabels = append(propagationLabels, types.String{
				Unknown: false,
				Null:    false,
				Value:   prop.(string),
			})
		}
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		Account:           state.Account,
		PropagationLabels: types.List{Elems: propagationLabels, ElemType: types.StringType},
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
	integrations, _, err := r.p.client.DefaultApi.ListIntegrations(ctx).Execute()
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
		config := configuration.(map[string]interface{})
		if config["name"].(string) == plan.IntegrationName.Value {
			moduleConfiguration = config["configuration"].([]interface{})
			moduleInstance["brand"] = config["name"].(string)
			moduleInstance["canSample"] = config["canGetSamples"].(bool)
			moduleInstance["category"] = config["category"].(string)
			moduleInstance["configuration"] = configuration
			moduleInstance["data"] = make([]map[string]interface{}, 0)
			moduleInstance["defaultIgnore"] = false
			moduleInstance["enabled"] = "true"
			// todo: add this as a config option
			//moduleInstance["engine"] = ""
			//moduleInstance["engineGroup"] = ""
			moduleInstance["id"] = state.Id.Value
			//moduleInstance["incomingMapperId"] = ""
			//moduleInstance["integrationLogLevel"] = ""
			// todo: add this as a config option (byoi)
			var isIntegrationScript bool
			if val, ok := config["integrationScript"]; ok && val != nil {
				isIntegrationScript = true
			}
			moduleInstance["isIntegrationScript"] = isIntegrationScript
			//moduleInstance["isLongRunning"] = false
			//moduleInstance["mappingId"] = ""
			moduleInstance["name"] = plan.Name.Value
			//moduleInstance["outgoingMapperId"] = ""
			//moduleInstance["passwordProtected"] = false
			moduleInstance["propagationLabels"] = plan.PropagationLabels
			//moduleInstance["resetContext"] = false
			moduleInstance["version"] = -1
			break
		}
	}
	for _, parameter := range moduleConfiguration {
		param := parameter.(map[string]interface{})
		param["hasvalue"] = false
		for configName, configValue := range plan.Config.Elems {
			if param["display"].(string) == configName || param["name"].(string) == configName {
				param["value"], _ = configValue.ToTerraformValue(ctx)
				param["hasvalue"] = true
				break
			}
		}
		if !param["hasvalue"].(bool) {
			param["value"] = param["defaultValue"].(string)
		}
		moduleInstance["data"] = append(moduleInstance["data"].([]map[string]interface{}), param)
	}
	var integration map[string]interface{}
	if state.Account.Null || len(state.Account.Value) == 0 {
		integration, _, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstance(ctx).CreateIntegrationRequest(moduleInstance).Execute()
	} else {
		integration, _, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstanceAccount(ctx, "acc_"+plan.Account.Value).CreateIntegrationRequest(moduleInstance).Execute()
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration instance",
			"Could not update integration instance: "+err.Error(),
		)
		return
	}

	var propagationLabels []attr.Value
	if integration["propagationLabels"] == nil {
		propagationLabels = []attr.Value{}
	} else {
		for _, prop := range integration["propagationLabels"].([]interface{}) {
			propagationLabels = append(propagationLabels, types.String{
				Unknown: false,
				Null:    false,
				Value:   prop.(string),
			})
		}
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		Account:           plan.Account,
		PropagationLabels: types.List{Elems: propagationLabels, ElemType: types.StringType},
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
	var err error
	if state.Account.Null || len(state.Account.Value) == 0 {
		_, err = r.p.client.DefaultApi.DeleteIntegrationInstance(ctx, state.Id.Value).Execute()
	} else {
		_, err = r.p.client.DefaultApi.DeleteIntegrationInstanceAccount(ctx, state.Id.Value, "acc_"+state.Account.Value).Execute()
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting integration instance",
			"Could not delete integration instance: "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r resourceIntegrationInstance) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	accname := strings.Split(req.ID, ".")
	var acc, name string
	var integration map[string]interface{}
	var err error
	if len(accname) == 1 {
		name = req.ID
		integration, _, err = r.p.client.DefaultApi.GetIntegrationInstance(ctx).SetIdentifier(name).Execute()
	} else {
		acc, name = accname[0], accname[1]
		integration, _, err = r.p.client.DefaultApi.GetIntegrationInstanceAccount(ctx, "acc_"+acc).SetIdentifier(name).Execute()
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting integration instance",
			"Could not get integration instance: "+err.Error(),
		)
		return
	}
	if integration == nil {
		resp.Diagnostics.AddError(
			"Integration instance not found",
			"Could not find integration instance: "+name,
		)
		return
	}

	var propagationLabels []attr.Value
	if integration["propagationLabels"] == nil {
		propagationLabels = []attr.Value{}
	} else {
		for _, prop := range integration["propagationLabels"].([]interface{}) {
			propagationLabels = append(propagationLabels, types.String{
				Unknown: false,
				Null:    false,
				Value:   prop.(string),
			})
		}
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		PropagationLabels: types.List{Elems: propagationLabels, ElemType: types.StringType},
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
