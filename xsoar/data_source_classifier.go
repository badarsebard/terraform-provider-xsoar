package xsoar

import (
	"context"
	"encoding/json"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
	"log"
	"net/http"
)

type dataSourceClassifierType struct{}

func (r dataSourceClassifierType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
			"default_incident_type": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"key_type_map": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"transformer": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"propagation_labels": {
				Type:     types.SetType{ElemType: types.StringType},
				Computed: true,
				Optional: false,
			},
			"account": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
		},
	}, nil
}

func (r dataSourceClassifierType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceClassifier{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceClassifier struct {
	p provider
}

func (r dataSourceClassifier) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Declare struct that this function will set to this data source's config
	var config Classifier
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from API
	var classifier openapi.InstanceClassifier
	var httpResponse *http.Response
	var err error
	if config.Account.Null || len(config.Account.Value) == 0 {
		classifier, httpResponse, err = r.p.client.DefaultApi.GetClassifier(ctx).SetIdentifier(config.Name.Value).Execute()
	} else {
		classifier, httpResponse, err = r.p.client.DefaultApi.GetClassifierAccount(ctx, "acc_"+config.Account.Value).SetIdentifier(config.Name.Value).Execute()
	}
	if httpResponse != nil {
		getBody, _ := httpResponse.Request.GetBody()
		b, _ := io.ReadAll(getBody)
		log.Println(string(b))
	}
	if err != nil {
		log.Println(err.Error())
		resp.Diagnostics.AddError(
			"Error getting classifier",
			"Could not get classifier: "+err.Error(),
		)
		return
	}

	var propLabels []attr.Value
	for _, label := range classifier.GetPropagationLabels() {
		propLabels = append(propLabels, types.String{Value: label})
	}

	// Map response body to resource schema attribute
	defaultIncidentType, err := json.Marshal(classifier.GetDefaultIncidentType())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling json",
			"Could not marshal json: "+err.Error(),
		)
		return
	}
	keyTypeMap, err := json.Marshal(classifier.GetKeyTypeMap())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling json",
			"Could not marshal json: "+err.Error(),
		)
		return
	}
	transformer, err := json.Marshal(classifier.GetTransformer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling json",
			"Could not marshal json: "+err.Error(),
		)
		return
	}
	result := Classifier{
		Name:              types.String{Value: classifier.GetName()},
		Id:                types.String{Value: classifier.GetId()},
		PropagationLabels: types.Set{Elems: propLabels, ElemType: types.StringType},
		Account:           config.Account,
	}
	if v := string(defaultIncidentType); v == "null" {
		result.DefaultIncidentType = types.String{Null: true}
	} else {
		result.DefaultIncidentType = types.String{Value: v}
	}
	if v := string(keyTypeMap); v == "null" {
		result.KeyTypeMap = types.String{Null: true}
	} else {
		result.KeyTypeMap = types.String{Value: v}
	}
	if v := string(transformer); v == "null" {
		result.Transformer = types.String{Null: true}
	} else {
		result.Transformer = types.String{Value: v}
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
