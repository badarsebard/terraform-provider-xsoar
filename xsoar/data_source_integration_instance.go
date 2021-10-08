package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"log"
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
			"config": {
				Type:     types.MapType{ElemType: types.StringType},
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
	// Declare struct that this function will set to this data source's state
	var state IntegrationInstance
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from API
	var integrationInstance map[string]interface{}
	var err error
	if state.Account.Null || len(state.Account.Value) == 0 {
		integrationInstance, _, err = r.p.client.DefaultApi.GetIntegrationInstance(ctx).Identifier(state.Name.Value).Execute()
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
				"Could not find integration instance: "+state.Name.Value,
			)
			return
		}
	} else {
		integrationInstance, _, err = r.p.client.DefaultApi.GetIntegrationInstanceAccount(ctx, "acc_"+state.Account.Value).Identifier(state.Name.Value).Execute()
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
				"Could not find integration instance: "+state.Name.Value,
			)
			return
		}
	}
	log.Println(integrationInstance)
	var propLabels []string
	for _, label := range integrationInstance["propagationLabels"].([]interface{}) {
		propLabels = append(propLabels, label.(string))
	}

	// Map response body to resource schema attribute
	state.Name = types.String{Value: integrationInstance["name"].(string)}
	state.Id = types.String{Value: integrationInstance["id"].(string)}
	state.IntegrationName = types.String{Value: integrationInstance["brand"].(string)}
	state.PropagationLabels = propLabels

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
