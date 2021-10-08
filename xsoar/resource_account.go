package xsoar

import (
	"context"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceAccountType struct{}

// GetSchema Resource schema
func (r resourceAccountType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	var planModifiers []tfsdk.AttributePlanModifier
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"account_roles": {
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Required: true,
			},
			"host_group_name": {
				Type:     types.StringType,
				Required: true,
			},
			"host_group_id": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"name": {
				Type:          types.StringType,
				Required:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			"propagation_labels": {
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Required: true,
			},
		},
	}, nil
}

// NewResource instance
func (r resourceAccountType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceAccount{
		p: *(p.(*provider)),
	}, nil
}

type resourceAccount struct {
	p provider
}

// Create a new resource
func (r resourceAccount) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Retrieve values from plan
	var plan Account
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createAccountRequest := *openapi.NewCreateAccountRequest()
	createAccountRequest.SetAccountRoles(plan.AccountRoles)
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not list HA groups: "+err.Error(),
		)
		return
	}
	var hostGroupId = ""
	for _, group := range haGroups {
		if group["name"].(string) == plan.HostGroupName.Value {
			hostGroupId = group["id"].(string)
			break
		}
	}
	createAccountRequest.SetHostGroupId(hostGroupId)
	createAccountRequest.SetName(plan.Name.Value)
	createAccountRequest.SetPropagationLabels(plan.PropagationLabels)
	createAccountRequest.SetSyncOnCreation(true)

	// Create new account
	accounts, _, err := r.p.client.DefaultApi.CreateAccount(context.Background()).CreateAccountRequest(createAccountRequest).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading order",
			"Could not create account "+plan.Name.Value+": "+err.Error(),
		)
		return
	}

	// Map response body to resource schema attribute
	var result Account
	for _, account := range accounts {
		if account["displayName"].(string) == plan.Name.Value {
			var propLabels = []string{}
			for _, label := range account["propagationLabels"].([]interface{}) {
				propLabels = append(propLabels, label.(string))
			}
			var hostGroupName string
			for _, group := range haGroups {
				if group["id"].(string) == account["hostGroupId"].(string) {
					hostGroupName = group["name"].(string)
					break
				}
			}
			result = Account{
				Name:              types.String{Value: account["displayName"].(string)},
				HostGroupName:     types.String{Value: hostGroupName},
				HostGroupId:       types.String{Value: hostGroupId},
				PropagationLabels: propLabels,
				AccountRoles:      plan.AccountRoles,
			}
			break
		}
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceAccount) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Get current state
	var state Account
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get account from API and then update what is in state from what the API returns
	accName := "acc_" + state.Name.Value

	// Get account current value
	account, _, err := r.p.client.DefaultApi.GetAccount(context.Background(), accName).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting account",
			"Could not read account "+accName+": "+err.Error(),
		)
		return
	}

	var propagationLabels = []string{}
	for _, prop := range account["propagationLabels"].([]interface{}) {
		propagationLabels = append(propagationLabels, prop.(string))
	}

	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing account details",
			"Could not read account details"+err.Error(),
		)
		return
	}
	var roles []string
	for _, detail := range details {
		castDetail := detail.(map[string]interface{})
		if account["name"].(string) == castDetail["name"].(string) {
			roleObjects := castDetail["roles"].([]interface{})
			for _, roleObject := range roleObjects {
				roleName := roleObject.(map[string]interface{})["name"]
				roles = append(roles, roleName.(string))
			}
		}
	}
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}
	var hostGroupName = ""
	for _, group := range haGroups {
		if group["id"].(string) == account["hostGroupId"].(string) {
			hostGroupName = group["name"].(string)
			break
		}
	}
	// Map response body to resource schema attribute
	state = Account{
		Name:              types.String{Value: account["displayName"].(string)},
		HostGroupName:     types.String{Value: hostGroupName},
		HostGroupId:       types.String{Value: account["hostGroupId"].(string)},
		PropagationLabels: propagationLabels,
		AccountRoles:      roles,
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r resourceAccount) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan values
	var plan Account
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state Account
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	// This requires up to two requests: roles and propagation labels, and host migration
	// RolesAndPropagationLabels
	if equalSliceString(plan.AccountRoles, state.AccountRoles) || equalSliceString(plan.PropagationLabels, state.PropagationLabels) {
		updateRolesAndPropagationLabelsRequest := *openapi.NewUpdateRolesAndPropagationLabelsRequest()
		updateRolesAndPropagationLabelsRequest.SetSelectedRoles(plan.AccountRoles)
		updateRolesAndPropagationLabelsRequest.SetSelectedPropagationLabels(plan.PropagationLabels)
		_, _, err := r.p.client.DefaultApi.UpdateAccount(context.Background(), plan.Name.Value).UpdateRolesAndPropagationLabelsRequest(updateRolesAndPropagationLabelsRequest).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error update account",
				"Could not update account "+plan.Name.Value+": "+err.Error(),
			)
			return
		}
	}

	// Host
	// todo: implement after updating sdk with account host migration capability
	if plan.HostGroupName.Value != state.HostGroupName.Value {
		haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(context.Background()).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing HA groups",
				"Could not read HA groups"+err.Error(),
			)
			return
		}
		var targetHostGroupId = ""
		for _, group := range haGroups {
			if group["name"].(string) == plan.HostGroupName.Value {
				targetHostGroupId = group["id"].(string)
				break
			}
		}
		_, _, err = r.p.client.DefaultApi.UpdateAccountHost(context.Background(), "acc_"+plan.Name.Value, targetHostGroupId).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating account host",
				"Could not update account host for "+plan.Name.Value+": "+err.Error(),
			)
			return
		}
	}

	// Get account from API and then update what is in state from what the API returns
	accName := "acc_" + state.Name.Value

	// Get account current value
	account, _, err := r.p.client.DefaultApi.GetAccount(context.Background(), accName).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting account",
			"Could not read account "+accName+": "+err.Error(),
		)
		return
	}

	var propagationLabels = []string{}
	for _, prop := range account["propagationLabels"].([]interface{}) {
		propagationLabels = append(propagationLabels, prop.(string))
	}

	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing account details",
			"Could not read account details"+err.Error(),
		)
		return
	}
	var roles []string
	for _, detail := range details {
		castDetail := detail.(map[string]interface{})
		if account["name"].(string) == castDetail["name"].(string) {
			roleObjects := castDetail["roles"].([]interface{})
			for _, roleObject := range roleObjects {
				roleName := roleObject.(map[string]interface{})["name"]
				roles = append(roles, roleName.(string))
			}
		}
	}
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}
	var hostGroupName = ""
	for _, group := range haGroups {
		if group["id"].(string) == account["hostGroupId"].(string) {
			hostGroupName = group["name"].(string)
			break
		}
	}
	// Map response body to resource schema attribute
	result := Account{
		Name:              types.String{Value: account["displayName"].(string)},
		HostGroupName:     types.String{Value: hostGroupName},
		HostGroupId:       types.String{Value: account["hostGroupId"].(string)},
		PropagationLabels: propagationLabels,
		AccountRoles:      roles,
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceAccount) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state Account
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get order ID from state
	accName := "acc_" + state.Name.Value

	// Delete order by calling API
	_, _, err := r.p.client.DefaultApi.DeleteAccount(context.Background(), accName).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting account",
			"Could not delete account "+state.Name.Value+": "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r resourceAccount) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	accName := "acc_" + req.ID
	// Get account current value
	account, _, err := r.p.client.DefaultApi.GetAccount(context.Background(), accName).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting account",
			"Could not read account "+accName+": "+err.Error(),
		)
		return
	}

	var propagationLabels = []string{}
	for _, prop := range account["propagationLabels"].([]interface{}) {
		propagationLabels = append(propagationLabels, prop.(string))
	}

	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing account details",
			"Could not read account details"+err.Error(),
		)
		return
	}
	var roles []string
	for _, detail := range details {
		castDetail := detail.(map[string]interface{})
		if account["name"].(string) == castDetail["name"].(string) {
			roleObjects := castDetail["roles"].([]interface{})
			for _, roleObject := range roleObjects {
				roleName := roleObject.(map[string]interface{})["name"]
				roles = append(roles, roleName.(string))
			}
		}
	}
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}
	var hostGroupName = ""
	for _, group := range haGroups {
		if group["id"].(string) == account["hostGroupId"].(string) {
			hostGroupName = group["name"].(string)
			break
		}
	}
	// Map response body to resource schema attribute
	var state = Account{
		Name:              types.String{Value: account["displayName"].(string)},
		HostGroupName:     types.String{Value: hostGroupName},
		PropagationLabels: propagationLabels,
		AccountRoles:      roles,
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
