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

type isValidDirection struct{}

func (v isValidDirection) Description(ctx context.Context) string {
	return fmt.Sprint("direction must be incoming or outgoing exactly")
}

func (v isValidDirection) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprint("direction must be `incoming` or `outgoing` exactly")
}

func (v isValidDirection) Validate(ctx context.Context, request tfsdk.ValidateAttributeRequest, response *tfsdk.ValidateAttributeResponse) {
	var str types.String
	diags := tfsdk.ValueAs(ctx, request.AttributeConfig, &str)
	response.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if !(str.Value == "incoming" || str.Value == "outgoing") {
		response.Diagnostics.AddAttributeError(
			request.AttributePath,
			"Invalid Direction Value",
			fmt.Sprintf("Direction must be either incoming or outgoing exactly, got: %s.", str.Value),
		)

		return
	}
}

type resourceMapperType struct{}

// GetSchema Resource schema
func (r resourceMapperType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
			"mapping": {
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
			"direction": {
				Type:       types.StringType,
				Required:   true,
				Validators: []tfsdk.AttributeValidator{isValidDirection{}},
			},
		},
	}, nil
}

// NewResource instance
func (r resourceMapperType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceMapper{
		p: *(p.(*provider)),
	}, nil
}

type resourceMapper struct {
	p provider
}

// Create a new resource
func (r resourceMapper) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Retrieve values from plan
	var plan Mapper
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create
	mapperRequest := *openapi.NewCreateUpdateClassifierRequest()
	mapperRequest.SetType("mapping-" + plan.Direction.Value)
	mapperRequest.SetName(plan.Name.Value)
	var err error
	if !plan.Mapping.Unknown {
		var mapping map[string]interface{}
		err = json.Unmarshal([]byte(plan.Mapping.Value), &mapping)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error unmarshalling json",
				"Could not unmarshal json: "+err.Error(),
			)
			return
		}
		mapperRequest.SetKeyTypeMap(mapping)
	}
	if !plan.PropagationLabels.Unknown {
		var props []string
		plan.PropagationLabels.ElementsAs(ctx, props, true)
		mapperRequest.SetPropagationLabels(props)
	}
	var mapper openapi.InstanceClassifier
	var httpResponse *http.Response
	if plan.Account.Null || len(plan.Account.Value) == 0 {
		mapper, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifier(ctx).CreateUpdateClassifierRequest(mapperRequest).Execute()
	} else {
		mapper, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifierAccount(ctx, "acc_"+plan.Account.Value).CreateUpdateClassifierAccountRequest(mapperRequest).Execute()
	}
	if err != nil {
		getBody := httpResponse.Body
		b, _ := io.ReadAll(getBody)
		fmt.Println(string(b))
		resp.Diagnostics.AddError(
			"Error creating mapper",
			"Could not create mapper: "+err.Error()+" "+string(b),
		)
		return
	}

	var propLabels []attr.Value
	for _, label := range mapper.GetPropagationLabels() {
		propLabels = append(propLabels, types.String{Value: label})
	}

	// Map response body to resource schema attribute
	mapping, err := json.Marshal(mapper.GetMapping())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling json",
			"Could not marshal json: "+err.Error(),
		)
		return
	}
	result := Mapper{
		Name:              types.String{Value: mapper.GetName()},
		Id:                types.String{Value: mapper.GetId()},
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
		Account:           plan.Account,
		Direction:         plan.Direction,
	}
	if m := string(mapping); m == "null" {
		result.Mapping = types.String{Null: true}
	} else {
		result.Mapping = types.String{Value: m}
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceMapper) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Get current state
	var state Mapper
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from API
	var mapper openapi.InstanceClassifier
	var httpResponse *http.Response
	var err error
	if state.Account.Null || len(state.Account.Value) == 0 {
		mapper, httpResponse, err = r.p.client.DefaultApi.GetClassifier(ctx).SetIdentifier(state.Id.Value).Execute()
	} else {
		mapper, httpResponse, err = r.p.client.DefaultApi.GetClassifierAccount(ctx, "acc_"+state.Account.Value).SetIdentifier(state.Id.Value).Execute()
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
	for _, label := range mapper.GetPropagationLabels() {
		propLabels = append(propLabels, types.String{Value: label})
	}

	// Map response body to resource schema attribute
	mapping, err := json.Marshal(mapper.GetMapping())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling json",
			"Could not marshal json: "+err.Error(),
		)
		return
	}
	result := Mapper{
		Name:              types.String{Value: mapper.GetName()},
		Id:                types.String{Value: mapper.GetId()},
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
		Account:           state.Account,
		Direction:         state.Direction,
	}
	if m := string(mapping); m == "null" {
		result.Mapping = types.String{Null: true}
	} else {
		result.Mapping = types.String{Value: m}
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r resourceMapper) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan values
	var plan Mapper
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state Mapper
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build request
	mapperRequest := *openapi.NewCreateUpdateClassifierRequest()
	mapperRequest.SetType("mapping-" + plan.Direction.Value)
	mapperRequest.SetId(state.Id.Value)
	mapperRequest.SetName(plan.Name.Value)
	var err error
	if !plan.Mapping.Unknown {
		var mapping map[string]interface{}
		err = json.Unmarshal([]byte(plan.Mapping.Value), &mapping)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error unmarshalling json",
				"Could not unmarshal json: "+err.Error(),
			)
			return
		}
		mapperRequest.SetKeyTypeMap(mapping)
	}
	if !plan.PropagationLabels.Unknown {
		var props []string
		plan.PropagationLabels.ElementsAs(ctx, props, true)
		mapperRequest.SetPropagationLabels(props)
	}
	var mapper openapi.InstanceClassifier
	var httpResponse *http.Response
	if plan.Account.Null || len(plan.Account.Value) == 0 {
		mapper, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifier(ctx).CreateUpdateClassifierRequest(mapperRequest).Execute()
	} else {
		mapper, httpResponse, err = r.p.client.DefaultApi.CreateUpdateClassifierAccount(ctx, "acc_"+plan.Account.Value).CreateUpdateClassifierAccountRequest(mapperRequest).Execute()
	}
	if err != nil {
		getBody, _ := httpResponse.Request.GetBody()
		b, _ := io.ReadAll(getBody)
		log.Println(string(b))
		resp.Diagnostics.AddError(
			"Error updating mapper",
			"Could not update mapper: "+err.Error(),
		)
		return
	}

	var propLabels []attr.Value
	for _, label := range mapper.GetPropagationLabels() {
		propLabels = append(propLabels, types.String{Value: label})
	}

	// Map response body to resource schema attribute
	mapping, err := json.Marshal(mapper.GetMapping())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling json",
			"Could not marshal json: "+err.Error(),
		)
		return
	}
	result := Mapper{
		Name:              types.String{Value: mapper.GetName()},
		Id:                types.String{Value: mapper.GetId()},
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
		Account:           plan.Account,
		Direction:         plan.Direction,
	}
	if m := string(mapping); m == "null" {
		result.Mapping = types.String{Null: true}
	} else {
		result.Mapping = types.String{Value: m}
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceMapper) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	// Get state
	var state Mapper
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete
	var err error
	var httpResponse *http.Response
	if state.Account.Null || len(state.Account.Value) == 0 {
		httpResponse, err = r.p.client.DefaultApi.DeleteClassifier(ctx, state.Id.Value).Execute()
	} else {
		httpResponse, err = r.p.client.DefaultApi.DeleteClassifierAccount(ctx, state.Id.Value, "acc_"+state.Account.Value).Execute()
	}
	if err != nil {
		getBody := httpResponse.Body
		b, _ := io.ReadAll(getBody)
		log.Println(string(b))
		resp.Diagnostics.AddError(
			"Error deleting mapper",
			"Could not delete mapper: "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r resourceMapper) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	accname := strings.Split(req.ID, ".")
	var acc, name string
	var mapper openapi.InstanceClassifier
	var err error
	if len(accname) == 1 {
		name = req.ID
		mapper, _, err = r.p.client.DefaultApi.GetClassifier(ctx).SetIdentifier(name).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error importing mapper",
				"Could not import mapper: "+err.Error(),
			)
			return
		}
	} else {
		acc, name = accname[0], accname[1]
		mapper, _, err = r.p.client.DefaultApi.GetClassifierAccount(ctx, "acc_"+acc).SetIdentifier(name).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error importing mapper",
				"Could not import mapper: "+err.Error(),
			)
			return
		}
	}
	var propLabels []attr.Value
	for _, label := range mapper.GetPropagationLabels() {
		propLabels = append(propLabels, types.String{Value: label})
	}

	// Map response body to resource schema attribute
	mapping, err := json.Marshal(mapper.GetMapping())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshalling json",
			"Could not marshal json: "+err.Error(),
		)
		return
	}
	classificationType := mapper.GetType()
	splitClassification := strings.Split(classificationType, "-")
	direction := splitClassification[1]
	result := Mapper{
		Name:              types.String{Value: mapper.GetName()},
		Id:                types.String{Value: mapper.GetId()},
		PropagationLabels: types.List{Elems: propLabels, ElemType: types.StringType},
		Direction:         types.String{Value: direction},
	}
	if m := string(mapping); m == "null" {
		result.Mapping = types.String{Null: true}
	} else {
		result.Mapping = types.String{Value: m}
	}
	if acc == "" {
		result.Account = types.String{Null: true}
	} else {
		result.Account = types.String{Value: acc}
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
