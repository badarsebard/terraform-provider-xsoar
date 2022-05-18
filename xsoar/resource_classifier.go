package xsoar

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
	"log"
	"net/http"
	"strings"
)

type resourceClassifierType struct{}

// GetSchema Resource schema
func (r resourceClassifierType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
			"default_incident_type": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
			"key_type_map": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
			"transformer": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
			"propagation_labels": {
				Type:     types.ListType{ElemType: types.StringType},
				Optional: true,
				Computed: true,
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
func (r resourceClassifierType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceClassifier{
		p: *(p.(*provider)),
	}, nil
}

type resourceClassifier struct {
	p provider
}

// Create a new resource
func (r resourceClassifier) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Retrieve values from plan
	var plan Classifier
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create
	classifierRequest := *openapi.NewCreateUpdateClassifierRequest()
	classifierRequest.SetType("classification")
	classifierRequest.SetName(plan.Name.Value)
	var err error
	if !plan.DefaultIncidentType.Unknown {
		classifierRequest.SetDefaultIncidentType(plan.DefaultIncidentType.Value)
	}
	if !plan.KeyTypeMap.Unknown {
		var keyTypeMap map[string]interface{}
		err = json.Unmarshal([]byte(plan.KeyTypeMap.Value), &keyTypeMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error unmarshalling json",
				"Could not unmarshal json: "+err.Error(),
			)
			return
		}
		classifierRequest.SetKeyTypeMap(keyTypeMap)
	}
	if !plan.Transformer.Unknown {
		var transformer map[string]interface{}
		err = json.Unmarshal([]byte(plan.Transformer.Value), &transformer)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error unmarshalling json",
				"Could not unmarshal json: "+err.Error(),
			)
			return
		}
		classifierRequest.SetTransformer(transformer)
	}
	if !plan.PropagationLabels.Unknown {
		var props []string
		plan.PropagationLabels.ElementsAs(ctx, props, true)
		classifierRequest.SetPropagationLabels(props)
	}
	var classifier openapi.InstanceClassifier
	var httpResponse *http.Response
	if plan.Account.Null || len(plan.Account.Value) == 0 {
		classifier, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifier(ctx).CreateUpdateClassifierRequest(classifierRequest).Execute()
	} else {
		classifier, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifierAccount(ctx, "acc_"+plan.Account.Value).CreateUpdateClassifierAccountRequest(classifierRequest).Execute()
	}
	if err != nil {
		getBody, _ := httpResponse.Request.GetBody()
		b, _ := io.ReadAll(getBody)
		log.Println(string(b))
		resp.Diagnostics.AddError(
			"Error creating classifier",
			"Could not create classifier: "+err.Error(),
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
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
		Account:           plan.Account,
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

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceClassifier) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Get current state
	var state Classifier
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from API
	var classifier openapi.InstanceClassifier
	var httpResponse *http.Response
	var err error
	if state.Account.Null || len(state.Account.Value) == 0 {
		classifier, httpResponse, err = r.p.client.DefaultApi.GetClassifier(ctx).SetIdentifier(state.Name.Value).Execute()
	} else {
		classifier, httpResponse, err = r.p.client.DefaultApi.GetClassifierAccount(ctx, "acc_"+state.Account.Value).SetIdentifier(state.Name.Value).Execute()
	}
	if err != nil {
		getBody, _ := httpResponse.Request.GetBody()
		b, _ := io.ReadAll(getBody)
		log.Println(string(b))
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
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
		Account:           state.Account,
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

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r resourceClassifier) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan values
	var plan Classifier
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state Classifier
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build request
	classifierRequest := *openapi.NewCreateUpdateClassifierRequest()
	classifierRequest.SetType("classification")
	classifierRequest.SetId(state.Id.Value)
	classifierRequest.SetName(plan.Name.Value)
	var err error
	if !plan.DefaultIncidentType.Unknown {
		classifierRequest.SetDefaultIncidentType(plan.DefaultIncidentType.Value)
	}
	if !plan.KeyTypeMap.Unknown {
		var keyTypeMap map[string]interface{}
		err = json.Unmarshal([]byte(plan.KeyTypeMap.Value), &keyTypeMap)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error unmarshalling json",
				"Could not unmarshal json: "+err.Error(),
			)
			return
		}
		classifierRequest.SetKeyTypeMap(keyTypeMap)
	}
	if !plan.Transformer.Unknown {
		var transformer map[string]interface{}
		err = json.Unmarshal([]byte(plan.Transformer.Value), &transformer)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error unmarshalling json",
				"Could not unmarshal json: "+err.Error(),
			)
			return
		}
		classifierRequest.SetTransformer(transformer)
	}
	if !plan.PropagationLabels.Null {
		var props []string
		plan.PropagationLabels.ElementsAs(ctx, props, true)
		classifierRequest.SetPropagationLabels(props)
	}
	var classifier openapi.InstanceClassifier
	var httpResponse *http.Response
	if plan.Account.Null || len(plan.Account.Value) == 0 {
		classifier, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifier(ctx).CreateUpdateClassifierRequest(classifierRequest).Execute()
	} else {
		classifier, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifierAccount(ctx, "acc_"+plan.Account.Value).CreateUpdateClassifierAccountRequest(classifierRequest).Execute()
	}
	if err != nil {
		getBody, _ := httpResponse.Request.GetBody()
		b, _ := io.ReadAll(getBody)
		log.Println(string(b))
		resp.Diagnostics.AddError(
			"Error updating classifier",
			"Could not update classifier: "+err.Error(),
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
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
		Account:           plan.Account,
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

// Delete resource
func (r resourceClassifier) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	// Get state
	var state Classifier
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete
	var httpResponse *http.Response
	var err error
	if state.Account.Null || len(state.Account.Value) == 0 {
		httpResponse, err = r.p.client.DefaultApi.DeleteClassifier(ctx, state.Id.Value).Execute()
	} else {
		httpResponse, err = r.p.client.DefaultApi.DeleteClassifierAccount(ctx, state.Id.Value, "acc_"+state.Account.Value).Execute()
	}
	if err != nil {
		getBody := httpResponse.Body
		b, _ := io.ReadAll(getBody)
		fmt.Println(string(b))
		resp.Diagnostics.AddError(
			"Error deleting mapper",
			"Could not delete mapper: "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r resourceClassifier) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	accname := strings.Split(req.ID, ".")
	var acc, name string
	var classifier openapi.InstanceClassifier
	var err error
	if len(accname) == 1 {
		name = req.ID
		classifier, _, err = r.p.client.DefaultApi.GetClassifier(ctx).SetIdentifier(name).Execute()
	} else {
		acc, name = accname[0], accname[1]
		classifier, _, err = r.p.client.DefaultApi.GetClassifierAccount(ctx, "acc_"+acc).SetIdentifier(name).Execute()
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing classifier",
			"Could not import classifier: "+err.Error(),
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
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
	}
	if len(accname) == 1 {
		result.Account = types.String{Null: true}
	} else {
		result.Account = types.String{Value: acc}
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

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
