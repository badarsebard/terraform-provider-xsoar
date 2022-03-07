package xsoar

import (
	"context"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceHAGroupType struct{}

// GetSchema Resource schema
func (r resourceHAGroupType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
			},
			"elasticsearch_url": {
				Type:          types.StringType,
				Required:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			"elastic_index_prefix": {
				Type:          types.StringType,
				Required:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			// todo: add missing ES parameters
			"account_ids": {
				Type:     types.ListType{ElemType: types.StringType},
				Computed: true,
			},
			"host_ids": {
				Type:     types.ListType{ElemType: types.StringType},
				Computed: true,
			},
		},
	}, nil
}

// NewResource instance
func (r resourceHAGroupType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceHAGroup{
		p: *(p.(*provider)),
	}, nil
}

type resourceHAGroup struct {
	p provider
}

// Create a new resource
func (r resourceHAGroup) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Retrieve values from plan
	var plan HAGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createHAGroupRequest := *openapi.NewCreateHAGroupRequest()
	createHAGroupRequest.SetName(plan.Name.Value)
	createHAGroupRequest.SetElasticIndexPrefix(plan.ElasticIndexPrefix.Value)
	createHAGroupRequest.SetElasticsearchAddress(plan.ElasticsearchUrl.Value)

	// Create new HA group
	haGroup, _, err := r.p.client.DefaultApi.CreateHAGroup(ctx).CreateHAGroupRequest(createHAGroupRequest).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating HA group",
			"Could not create HA group "+plan.Name.Value+": "+err.Error(),
		)
		return
	}

	// todo: trigger the host installer build

	var accountIds []attr.Value
	for _, a := range haGroup.GetAccountIds() {
		accountIds = append(accountIds, types.String{Value: a})
	}
	var hostIds []attr.Value
	for _, h := range haGroup.GetAccountIds() {
		hostIds = append(hostIds, types.String{Value: h})
	}

	// Map response body to resource schema attribute
	var result HAGroup
	result = HAGroup{
		Name:               types.String{Value: haGroup.GetName()},
		Id:                 types.String{Value: haGroup.GetId()},
		ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
		ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
		AccountIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: types.StringType,
		},
		HostIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: nil,
		},
	}

	if len(accountIds) > 0 {
		result.AccountIds.Null = false
		result.AccountIds.Elems = accountIds
	}
	if len(hostIds) > 0 {
		result.HostIds.Null = false
		result.HostIds.Elems = hostIds
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceHAGroup) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Get current state
	var state HAGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get HA group from API and then update what is in state from what the API returns
	haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(ctx, state.Id.Value).Execute()
	if err != nil {
		resp.State.RemoveResource(ctx)
		//resp.Diagnostics.AddError(
		//	"Error getting HA group",
		//	"Could not get HA group "+state.Name.Value+": "+err.Error(),
		//)
		return
	}

	var accountIds []attr.Value
	for _, a := range haGroup.GetAccountIds() {
		accountIds = append(accountIds, types.String{Value: a})
	}
	var hostIds []attr.Value
	for _, h := range haGroup.GetHostIds() {
		hostIds = append(hostIds, types.String{Value: h})
	}

	// Map response body to resource schema attribute
	var result HAGroup
	result = HAGroup{
		Name:               types.String{Value: haGroup.GetName()},
		Id:                 types.String{Value: haGroup.GetId()},
		ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
		ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
		AccountIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: types.StringType,
		},
		HostIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: types.StringType,
		},
	}

	if len(accountIds) > 0 {
		result.AccountIds.Null = false
		result.AccountIds.Elems = accountIds
	}
	if len(hostIds) > 0 {
		result.HostIds.Null = false
		result.HostIds.Elems = hostIds
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r resourceHAGroup) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan values
	var plan HAGroup
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state HAGroup
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	updateHAGroupRequest := *openapi.NewCreateHAGroupRequest()
	updateHAGroupRequest.SetId(state.Id.Value)
	updateHAGroupRequest.SetName(plan.Name.Value)
	updateHAGroupRequest.SetElasticsearchAddress(plan.ElasticsearchUrl.Value)
	updateHAGroupRequest.SetElasticIndexPrefix(plan.ElasticIndexPrefix.Value)
	haGroup, _, err := r.p.client.DefaultApi.CreateHAGroup(ctx).CreateHAGroupRequest(updateHAGroupRequest).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating HA group",
			"Could not update HA group "+plan.Name.Value+": "+err.Error(),
		)
		return
	}

	var accountIds []attr.Value
	for _, a := range haGroup.GetAccountIds() {
		accountIds = append(accountIds, types.String{Value: a})
	}
	var hostIds []attr.Value
	for _, h := range haGroup.GetAccountIds() {
		hostIds = append(hostIds, types.String{Value: h})
	}

	// Map response body to resource schema attribute
	var result HAGroup
	result = HAGroup{
		Name:               types.String{Value: haGroup.GetName()},
		Id:                 types.String{Value: haGroup.GetId()},
		ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
		ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
		AccountIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: types.StringType,
		},
		HostIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: nil,
		},
	}

	if len(accountIds) > 0 {
		result.AccountIds.Null = false
		result.AccountIds.Elems = accountIds
	}
	if len(hostIds) > 0 {
		result.HostIds.Null = false
		result.HostIds.Elems = hostIds
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceHAGroup) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state HAGroup
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Verify existence
	_, _, err := r.p.client.DefaultApi.GetHAGroup(ctx, state.Id.Value).Execute()
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	// Delete HA group by calling API
	_, _, err = r.p.client.DefaultApi.DeleteHAGroup(ctx, state.Id.Value).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting HA group",
			"Could not delete HA group "+state.Name.Value+": "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r resourceHAGroup) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	name := req.ID
	// Get HA group current value
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}
	var id string
	for _, group := range haGroups {
		if group["name"].(string) == name {
			id = group["id"].(string)
			break
		}
	}
	haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(ctx, id).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting HA group",
			"Could not read HA group "+name+": "+err.Error(),
		)
		return
	}

	var accountIds []attr.Value
	for _, a := range haGroup.GetAccountIds() {
		accountIds = append(accountIds, types.String{Value: a})
	}
	var hostIds []attr.Value
	for _, h := range haGroup.GetAccountIds() {
		hostIds = append(hostIds, types.String{Value: h})
	}

	// Map response body to resource schema attribute
	var result HAGroup
	result = HAGroup{
		Name:               types.String{Value: haGroup.GetName()},
		Id:                 types.String{Value: haGroup.GetId()},
		ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
		ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
		AccountIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: types.StringType,
		},
		HostIds: types.List{
			Unknown:  false,
			Null:     true,
			ElemType: nil,
		},
	}

	if len(accountIds) > 0 {
		result.AccountIds.Null = false
		result.AccountIds.Elems = accountIds
	}
	if len(hostIds) > 0 {
		result.HostIds.Null = false
		result.HostIds.Elems = hostIds
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
