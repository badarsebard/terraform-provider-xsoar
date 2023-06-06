package xsoar

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
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
				Optional: true,
				Computed: true,
			},
			"propagation_labels": {
				Type:     types.SetType{ElemType: types.StringType},
				Computed: true,
				Optional: true,
			},
			"account": {
				Type:          types.StringType,
				Optional:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			"incoming_mapper_id": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
			// aka classifier
			"mapping_id": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
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
			var IncomingMapperId string
			if ok := plan.IncomingMapperId.Value; ok != "" {
				IncomingMapperId = plan.IncomingMapperId.Value
			} else {
				IncomingMapperId = ""
			}
			moduleInstance["incomingMapperId"] = IncomingMapperId
			var MappingId string
			if ok := plan.MappingId.Value; ok != "" {
				MappingId = plan.MappingId.Value
			} else {
				MappingId = ""
			}
			moduleInstance["mappingId"] = MappingId
			//moduleInstance["integrationLogLevel"] = ""
			// todo: add this as a config option (byoi)
			var isIntegrationScript bool
			if val, ok := config["integrationScript"]; ok && val != nil {
				isIntegrationScript = true
			}
			moduleInstance["isIntegrationScript"] = isIntegrationScript
			//moduleInstance["isLongRunning"] = false
			moduleInstance["mappingId"] = MappingId
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
	var configs map[string]string
	plan.Config.ElementsAs(ctx, &configs, true)
	for _, parameter := range moduleConfiguration {
		param := parameter.(map[string]interface{})
		param["hasvalue"] = false
		for configName, configValue := range configs {
			if param["display"].(string) == configName || param["name"].(string) == configName {
				param["value"] = configValue
				param["hasvalue"] = true
				break
			}
		}
		moduleInstance["data"] = append(moduleInstance["data"].([]map[string]interface{}), param)
	}

	var integration map[string]interface{}
	var httpResponse *http.Response
	var body []byte
	err = resource.RetryContext(ctx, 10*time.Minute, func() *resource.RetryError {
		if plan.Account.Null || len(plan.Account.Value) == 0 {
			integration, httpResponse, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstance(ctx).CreateIntegrationRequest(moduleInstance).Execute()
		} else {
			integration, httpResponse, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstanceAccount(ctx, "acc_"+plan.Account.Value).CreateIntegrationRequest(moduleInstance).Execute()
		}
		if httpResponse != nil {
			body, _ = io.ReadAll(httpResponse.Body)
			log.Printf("code: %d status: %s headers: %s body: %s\n", httpResponse.StatusCode, httpResponse.Status, httpResponse.Header, string(body))
		}
		if err != nil {
			log.Println(err.Error())
			time.Sleep(60 * time.Second)
			return resource.RetryableError(fmt.Errorf("error message: %s, http response: %s", err, body))
		}

		return nil
	})
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

	//var integrationConfigs map[string]attr.Value
	integrationConfigs := make(map[string]attr.Value)
	if integration["data"] == nil {
		integrationConfigs = map[string]attr.Value{}
		log.Println(integrationConfigs)
	} else {
		var integrationConfig map[string]interface{}
		var valueattr attr.Value
		switch reflect.TypeOf(integration["data"]).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(integration["data"])
				for i := 0; i < s.Len(); i++ {
					integrationConfig = s.Index(i).Interface().(map[string]interface{})
					log.Println(integrationConfig)

					valueconf, ok := integrationConfig["value"].(string)
					if ok {
						valueattr = types.String{ Value: valueconf,}
					} else {
						valueattr = types.String{ Value: "",}
					}

					nameconf, ok := integrationConfig["name"].(string)
					if ok {
						integrationConfigs[nameconf] = valueattr.(attr.Value)
					} else {
						break
					}
				}
		}
	}

	log.Println(integrationConfigs)

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		Account:           plan.Account,
		PropagationLabels: types.Set{Elems: propagationLabels, ElemType: types.StringType},
		Config:			   types.Map{Elems: integrationConfigs, ElemType: types.StringType},
	}

	IncomingMapperId, ok := integration["incomingMapperId"].(string)
	if ok {
		result.IncomingMapperId = types.String{Value: IncomingMapperId}
	} else {
		result.IncomingMapperId = types.String{Null: true}
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
		var account map[string]interface{}
		account, httpResponse, err = r.p.client.DefaultApi.GetAccount(ctx, "acc_"+state.Account.Value).Execute()
		if err != nil {
			log.Println(err.Error())
			if httpResponse != nil {
				body, _ := io.ReadAll(httpResponse.Body)
				payload, _ := io.ReadAll(httpResponse.Request.Body)
				log.Printf("code: %d status: %s headers: %s body: %s payload: %s\n", httpResponse.StatusCode, httpResponse.Status, httpResponse.Header, string(body), string(payload))
			}
			resp.Diagnostics.AddError(
				"Error getting integration instance",
				"Could not verify account existence: "+err.Error(),
			)
			return
		}
		if account == nil {
			resp.State.RemoveResource(ctx)
			return
		}
		integration, httpResponse, err = r.p.client.DefaultApi.GetIntegrationInstanceAccount(ctx, "acc_"+state.Account.Value).SetIdentifier(state.Id.Value).Execute()
	}
	if err != nil {
		log.Println(err.Error())
		if httpResponse != nil {
			body, _ := io.ReadAll(httpResponse.Body)
			payload, _ := io.ReadAll(httpResponse.Request.Body)
			log.Printf("code: %d status: %s headers: %s body: %s payload: %s\n", httpResponse.StatusCode, httpResponse.Status, httpResponse.Header, string(body), string(payload))
		}
		resp.Diagnostics.AddError(
			"Error getting integration instance",
			"Could not get integration instance: "+err.Error(),
		)
		return
	}

	if integration == nil {
		resp.State.RemoveResource(ctx)
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

	//var integrationConfigs map[string]attr.Value
	integrationConfigs := make(map[string]attr.Value)
	if integration["data"] == nil {
		integrationConfigs = map[string]attr.Value{}
		log.Println(integrationConfigs)
	} else {
		var integrationConfig map[string]interface{}
		var valueattr attr.Value
		switch reflect.TypeOf(integration["data"]).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(integration["data"])
				for i := 0; i < s.Len(); i++ {
					integrationConfig = s.Index(i).Interface().(map[string]interface{})
					log.Println(integrationConfig)

					valueconf, ok := integrationConfig["value"].(string)
					if ok {
						valueattr = types.String{ Value: valueconf,}
					} else {
						valueattr = types.String{ Value: "",}
					}

					nameconf, ok := integrationConfig["name"].(string)
					if ok {
						integrationConfigs[nameconf] = valueattr.(attr.Value)
					} else {
						break
					}
				}
		}
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		Account:           state.Account,
		PropagationLabels: types.Set{Elems: propagationLabels, ElemType: types.StringType},
		Config:			   types.Map{Elems: integrationConfigs, ElemType: types.StringType},
	}

	IncomingMapperId, ok := integration["incomingMapperId"].(string)
	if ok {
		result.IncomingMapperId = types.String{Value: IncomingMapperId}
	} else {
		result.IncomingMapperId = types.String{Null: true}
	}
	MappingId, ok := integration["mappingId"].(string)
	if ok {
		result.MappingId = types.String{Value: MappingId}
	} else {
		result.MappingId = types.String{Null: true}
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
			var IncomingMapperId string
			if ok := plan.IncomingMapperId.Value; ok != "" {
				IncomingMapperId = plan.IncomingMapperId.Value
			} else {
				IncomingMapperId = ""
			}
			moduleInstance["incomingMapperId"] = IncomingMapperId
			var MappingId string
			if ok := plan.MappingId.Value; ok != "" {
				MappingId = plan.MappingId.Value
			} else {
				MappingId = ""
			}
			moduleInstance["mappingId"] = MappingId
			//moduleInstance["integrationLogLevel"] = ""
			// todo: add this as a config option (byoi)
			var isIntegrationScript bool
			if val, ok := config["integrationScript"]; ok && val != nil {
				isIntegrationScript = true
			}
			moduleInstance["isIntegrationScript"] = isIntegrationScript
			//moduleInstance["isLongRunning"] = false
			moduleInstance["mappingId"] = MappingId
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
		if !param["hasvalue"].(bool) {
			param["value"] = param["defaultValue"].(string)
		}
		moduleInstance["data"] = append(moduleInstance["data"].([]map[string]interface{}), param)
	}
	var integration map[string]interface{}
	var httpResponse *http.Response
	if state.Account.Null || len(state.Account.Value) == 0 {
		integration, httpResponse, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstance(ctx).CreateIntegrationRequest(moduleInstance).Execute()
	} else {
		integration, httpResponse, err = r.p.client.DefaultApi.CreateUpdateIntegrationInstanceAccount(ctx, "acc_"+plan.Account.Value).CreateIntegrationRequest(moduleInstance).Execute()
	}
	if err != nil {
		log.Println(err.Error())
		if httpResponse != nil {
			body, _ := io.ReadAll(httpResponse.Body)
			payload, _ := io.ReadAll(httpResponse.Request.Body)
			log.Printf("code: %d status: %s headers: %s body: %s payload: %s\n", httpResponse.StatusCode, httpResponse.Status, httpResponse.Header, string(body), string(payload))
		}
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

	//var integrationConfigs map[string]attr.Value
	integrationConfigs := make(map[string]attr.Value)
	if integration["data"] == nil {
		integrationConfigs = map[string]attr.Value{}
		log.Println(integrationConfigs)
	} else {
		var integrationConfig map[string]interface{}
		var valueattr attr.Value
		switch reflect.TypeOf(integration["data"]).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(integration["data"])
				for i := 0; i < s.Len(); i++ {
					integrationConfig = s.Index(i).Interface().(map[string]interface{})
					log.Println(integrationConfig)

					valueconf, ok := integrationConfig["value"].(string)
					if ok {
						valueattr = types.String{ Value: valueconf,}
					} else {
						valueattr = types.String{ Value: "",}
					}

					nameconf, ok := integrationConfig["name"].(string)
					if ok {
						integrationConfigs[nameconf] = valueattr.(attr.Value)
					} else {
						break
					}
				}
		}
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		Account:           plan.Account,
		PropagationLabels: types.Set{Elems: propagationLabels, ElemType: types.StringType},
		Config:			   types.Map{Elems: integrationConfigs, ElemType: types.StringType},
	}

	IncomingMapperId, ok := integration["incomingMapperId"].(string)
	if ok {
		result.IncomingMapperId = types.String{Value: IncomingMapperId}
	} else {
		result.IncomingMapperId = types.String{Null: true}
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

	//var integrationConfigs map[string]attr.Value
	integrationConfigs := make(map[string]attr.Value)
	if integration["data"] == nil {
		integrationConfigs = map[string]attr.Value{}
		log.Println(integrationConfigs)
	} else {
		var integrationConfig map[string]interface{}
		var valueattr attr.Value
		switch reflect.TypeOf(integration["data"]).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(integration["data"])
				for i := 0; i < s.Len(); i++ {
					integrationConfig = s.Index(i).Interface().(map[string]interface{})
					log.Println(integrationConfig)

					valueconf, ok := integrationConfig["value"].(string)
					if ok {
						valueattr = types.String{ Value: valueconf,}
					} else {
						valueattr = types.String{ Value: "",}
					}

					nameconf, ok := integrationConfig["name"].(string)
					if ok {
						integrationConfigs[nameconf] = valueattr.(attr.Value)
					} else {
						break
					}
				}
		}
	}

	// Map response body to resource schema attribute
	result := IntegrationInstance{
		Name:              types.String{Value: integration["name"].(string)},
		Id:                types.String{Value: integration["id"].(string)},
		IntegrationName:   types.String{Value: integration["brand"].(string)},
		PropagationLabels: types.Set{Elems: propagationLabels, ElemType: types.StringType},
		Config:			   types.Map{Elems: integrationConfigs, ElemType: types.StringType},
	}

	IncomingMapperId, ok := integration["incomingMapperId"].(string)
	if ok {
		result.IncomingMapperId = types.String{Value: IncomingMapperId}
	} else {
		result.IncomingMapperId = types.String{Null: true}
	}
	MappingId, ok := integration["mappingId"].(string)
	if ok {
		result.MappingId = types.String{Value: MappingId}
	} else {
		result.MappingId = types.String{Null: true}
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
