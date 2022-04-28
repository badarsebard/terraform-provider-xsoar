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
	"reflect"
)

type dataSourceIntegrationInstanceType struct{}

func (r dataSourceIntegrationInstanceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"propagation_labels": {
				Type:     types.ListType{ElemType: types.StringType},
				Computed: true,
				Optional: false,
			},
			"account": {
				Type:     types.StringType,
				Optional: true,
			},
			"config": {
				Type:     types.MapType{ElemType: types.StringType},
				Optional: true,
				Computed: true,
			},
			"incoming_mapper_id": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
		},
	}, nil
}

func (r dataSourceIntegrationInstanceType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceIntegrationInstance{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceIntegrationInstance struct {
	p provider
}

func (r dataSourceIntegrationInstance) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Get current config
	var config IntegrationInstance
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from API
	var integration map[string]interface{}
	var httpResponse *http.Response
	var err error
	if config.Account.Null || len(config.Account.Value) == 0 {
		integration, httpResponse, err = r.p.client.DefaultApi.GetIntegrationInstance(ctx).SetIdentifier(config.Name.Value).Execute()
	} else {
		integration, httpResponse, err = r.p.client.DefaultApi.GetIntegrationInstanceAccount(ctx, "acc_"+config.Account.Value).SetIdentifier(config.Name.Value).Execute()
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
		Account:           config.Account,
		PropagationLabels: types.List{Elems: propagationLabels, ElemType: types.StringType},
		Config:            types.Map{Elems: integrationConfigs, ElemType: types.StringType},
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
