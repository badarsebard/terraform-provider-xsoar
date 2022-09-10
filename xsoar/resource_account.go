package xsoar

import (
	"context"
	"fmt"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"hash/crc64"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type resourceAccountType struct{}

// GetSchema Resource schema
func (r resourceAccountType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	var planModifiers []tfsdk.AttributePlanModifier
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"account_roles": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				Optional: true,
				Computed: true,
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
				Type: types.SetType{
					ElemType: types.StringType,
				},
				Optional: true,
				Computed: true,
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
			},
			"timeout": {
				Type:     types.Int64Type,
				Optional: true,
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
		log.Printf("%+v\n", req.Plan)
		return
	}

	// Generate API request body from plan
	createAccountRequest := *openapi.NewCreateAccountRequest()
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
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
	if !plan.AccountRoles.Null && len(plan.AccountRoles.Elems) > 0 {
		var accountRoles []string
		plan.AccountRoles.ElementsAs(ctx, &accountRoles, true)
		createAccountRequest.SetAccountRoles(accountRoles)
	} else {
		createAccountRequest.SetAccountRoles([]string{"Administrator"})
	}
	if !plan.PropagationLabels.Null && len(plan.PropagationLabels.Elems) > 0 {
		var propagationLabels []string
		plan.PropagationLabels.ElementsAs(ctx, propagationLabels, true)
		createAccountRequest.SetPropagationLabels(propagationLabels)
	}
	createAccountRequest.SetSyncOnCreation(true)

	// Create new account
	var accounts []map[string]interface{}
	timeout := time.Duration(1800) * time.Second
	if !plan.Timeout.Null && plan.Timeout.Value > 0 {
		timeout = time.Duration(plan.Timeout.Value) * time.Second
	}
	err = resource.RetryContext(ctx, timeout, func() *resource.RetryError {
		var httpResponse *http.Response
		var body []byte
		// initial random delay
		crcTable := crc64.MakeTable(crc64.ISO)
		seedInt := int64(crc64.Checksum([]byte(plan.Name.Value), crcTable))
		log.Printf("generated seed: %d\n", seedInt)
		randSource := rand.NewSource(seedInt)
		nrand := rand.New(randSource)
		randomTimeToWait := nrand.Intn(90) + 1
		log.Printf("sleeping for %d seconds\n", randomTimeToWait)
		time.Sleep(time.Duration(randomTimeToWait) * time.Second)
		// wait until no other accounts are being created
		accounts, httpResponse, err = r.p.client.DefaultApi.ListAccounts(ctx).Execute()
		if httpResponse != nil {
			body, _ = io.ReadAll(httpResponse.Body)
			log.Printf("%s : %s\n", httpResponse.Status, body)
		}
		if err != nil {
			return resource.RetryableError(fmt.Errorf("error message: %s, http response: %s", err, body))
		}
		for _, account := range accounts {
			if account["status"].(string) == "" {
				time.Sleep(60 * time.Second)
				return resource.RetryableError(fmt.Errorf("waiting for account %s to finish creation", account["name"].(string)))
			}
		}
		// Create account
		log.Printf("creating account")

		accounts, httpResponse, err = r.p.client.DefaultApi.CreateAccount(ctx).CreateAccountRequest(createAccountRequest).Execute()
		if httpResponse != nil {
			body, _ = io.ReadAll(httpResponse.Body)
			payload, _ := io.ReadAll(httpResponse.Request.Body)
			log.Printf("%s : %s - %s\n", payload, httpResponse.Status, body)
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
			"Error creating account",
			"Could not create account: "+err.Error(),
		)
		return
	}

	// Map response body to resource schema attribute
	var result Account
	for _, account := range accounts {
		if account["displayName"].(string) == plan.Name.Value {
			var propagationLabels []attr.Value
			if account["propagationLabels"] == nil {
				propagationLabels = []attr.Value{}
			} else {
				for _, label := range account["propagationLabels"].([]interface{}) {
					propagationLabels = append(propagationLabels, types.String{
						Unknown: false,
						Null:    false,
						Value:   label.(string),
					})
				}
			}

			var hostGroupName string
			for _, group := range haGroups {
				hgi, ok := account["hostGroupId"].(string)
				if ok && group["id"].(string) == hgi {
					hostGroupName = group["name"].(string)
					break
				}
			}

			var roles []attr.Value
			if account["roles"] == nil {
				roles = []attr.Value{}
			} else {
				rolesMap := account["roles"].(map[string]interface{})
				rolesMapRoles := rolesMap["roles"].([]interface{})
				var role string
				var ok bool
				for _, roleInterface := range rolesMapRoles {
					role, ok = roleInterface.(string)
					if ok {
						roles = append(roles, types.String{
							Unknown: false,
							Null:    false,
							Value:   role,
						})
					}
				}
			}

			result = Account{
				Name:          types.String{Value: account["displayName"].(string)},
				HostGroupName: types.String{Value: hostGroupName},
				HostGroupId:   types.String{Value: hostGroupId},
				PropagationLabels: types.Set{
					Unknown:  false,
					Null:     false,
					Elems:    propagationLabels,
					ElemType: types.StringType,
				},
				AccountRoles: types.Set{
					Unknown:  false,
					Null:     false,
					Elems:    roles,
					ElemType: types.StringType,
				},
				Id:      types.String{Value: account["id"].(string)},
				Timeout: plan.Timeout,
			}
			break
		}
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, &result)
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
	account, _, err := r.p.client.DefaultApi.GetAccount(ctx, accName).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting account",
			"Could not read account "+accName+": "+err.Error(),
		)
		return
	}
	if account == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var propagationLabels []attr.Value
	if account["propagationLabels"] == nil {
		propagationLabels = []attr.Value{}
	} else {
		for _, prop := range account["propagationLabels"].([]interface{}) {
			propagationLabels = append(propagationLabels, types.String{
				Unknown: false,
				Null:    false,
				Value:   prop.(string),
			})
		}
	}

	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing account details",
			"Could not read account details"+err.Error(),
		)
		return
	}
	var roles []attr.Value
	for _, detail := range details {
		castDetail, ok1 := detail.(map[string]interface{})
		accountName, ok2 := account["name"].(string)
		if ok1 && ok2 && accountName == castDetail["name"].(string) {
			roleObjects := castDetail["roles"].([]interface{})
			for _, roleObject := range roleObjects {
				roleName := roleObject.(map[string]interface{})["name"]
				roles = append(roles, types.String{
					Unknown: false,
					Null:    false,
					Value:   roleName.(string),
				})
			}
		}
	}
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}
	var hostGroupName = ""
	for _, group := range haGroups {
		hostGroupId, ok := account["hostGroupId"].(string)
		if ok && group["id"].(string) == hostGroupId {
			hostGroupName = group["name"].(string)
			break
		}
	}

	// Map response body to resource schema attribute
	state = Account{
		Name:          types.String{Value: account["displayName"].(string)},
		HostGroupName: types.String{Value: hostGroupName},
		HostGroupId:   types.String{Value: account["hostGroupId"].(string)},
		PropagationLabels: types.Set{
			Unknown:  false,
			Null:     false,
			Elems:    propagationLabels,
			ElemType: types.StringType,
		},
		AccountRoles: types.Set{
			Unknown:  false,
			Null:     false,
			Elems:    roles,
			ElemType: types.StringType,
		},
		Id:      types.String{Value: account["id"].(string)},
		Timeout: state.Timeout,
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
	var err error
	if !plan.AccountRoles.Null || !plan.PropagationLabels.Null {
		updateRolesAndPropagationLabelsRequest := *openapi.NewUpdateRolesAndPropagationLabelsRequest()
		var updateRolesAndPropagationLabels = false
		if !plan.AccountRoles.Null && len(plan.AccountRoles.Elems) > 0 && !plan.AccountRoles.Equal(state.AccountRoles) {
			var roles []string
			for _, elem := range plan.AccountRoles.Elems {
				role, _ := elem.ToTerraformValue(ctx)
				if role.IsKnown() && !role.IsNull() {
					var roleToAppend string
					err = role.As(&roleToAppend)
					if err != nil {
						resp.Diagnostics.AddError(
							"Error converting role",
							"Could not convert role to string: "+err.Error(),
						)
						return
					}
					roles = append(roles, roleToAppend)
				}
			}
			updateRolesAndPropagationLabelsRequest.SetSelectedRoles(roles)
			updateRolesAndPropagationLabels = true
		} else {
			var roles []string
			for _, elem := range state.AccountRoles.Elems {
				role, _ := elem.ToTerraformValue(ctx)
				if role.IsKnown() && !role.IsNull() {
					var roleToAppend string
					err = role.As(&roleToAppend)
					if err != nil {
						resp.Diagnostics.AddError(
							"Error converting role",
							"Could not convert role to string: "+err.Error(),
						)
						return
					}
					roles = append(roles, roleToAppend)
				}
			}
			updateRolesAndPropagationLabelsRequest.SetSelectedRoles(roles)
		}
		if !plan.PropagationLabels.Null && len(plan.PropagationLabels.Elems) > 0 && !plan.PropagationLabels.Equal(state.PropagationLabels) {
			var propagationLabels []string
			for _, elem := range plan.PropagationLabels.Elems {
				label, _ := elem.ToTerraformValue(ctx)
				if label.IsKnown() && !label.IsNull() {
					var labelToAppend string
					err = label.As(&labelToAppend)
					if err != nil {
						resp.Diagnostics.AddError(
							"Error converting label",
							"Could not convert label to string: "+err.Error(),
						)
						return
					}
					propagationLabels = append(propagationLabels, labelToAppend)
				}
			}
			updateRolesAndPropagationLabelsRequest.SetSelectedPropagationLabels(propagationLabels)
			updateRolesAndPropagationLabels = true
		} else {
			var propagationLabels []string
			for _, elem := range state.PropagationLabels.Elems {
				label, _ := elem.ToTerraformValue(ctx)
				if label.IsKnown() && !label.IsNull() {
					var labelToAppend string
					err = label.As(&labelToAppend)
					if err != nil {
						resp.Diagnostics.AddError(
							"Error converting label",
							"Could not convert label to string: "+err.Error(),
						)
						return
					}
					propagationLabels = append(propagationLabels, labelToAppend)
				}
			}
			updateRolesAndPropagationLabelsRequest.SetSelectedPropagationLabels(propagationLabels)
		}
		if updateRolesAndPropagationLabels {
			_, _, err = r.p.client.DefaultApi.UpdateAccount(ctx, plan.Name.Value).UpdateRolesAndPropagationLabelsRequest(updateRolesAndPropagationLabelsRequest).Execute()
			if err != nil {
				resp.Diagnostics.AddError(
					"Error update account",
					"Could not update account "+plan.Name.Value+": "+err.Error(),
				)
				return
			}
		}
	}

	// Host
	// todo: implement after updating sdk with account host migration capability
	if plan.HostGroupName.Value != state.HostGroupName.Value {
		haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
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
		_, _, err = r.p.client.DefaultApi.UpdateAccountHost(ctx, "acc_"+plan.Name.Value, targetHostGroupId).Execute()
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
	account, _, err := r.p.client.DefaultApi.GetAccount(ctx, accName).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting account",
			"Could not read account "+accName+": "+err.Error(),
		)
		return
	}
	if account == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var propagationLabels []attr.Value
	if account["propagationLabels"] == nil {
		propagationLabels = []attr.Value{}
	} else {
		for _, prop := range account["propagationLabels"].([]interface{}) {
			propagationLabels = append(propagationLabels, types.String{
				Unknown: false,
				Null:    false,
				Value:   prop.(string),
			})
		}
	}

	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing account details",
			"Could not read account details"+err.Error(),
		)
		return
	}
	var roles []attr.Value
	for _, detail := range details {
		castDetail := detail.(map[string]interface{})
		if account["name"].(string) == castDetail["name"].(string) {
			roleObjects := castDetail["roles"].([]interface{})
			for _, roleObject := range roleObjects {
				roleName := roleObject.(map[string]interface{})["name"]
				roles = append(roles, types.String{
					Unknown: false,
					Null:    false,
					Value:   roleName.(string),
				})
			}
		}
	}
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}
	var hostGroupName = ""
	for _, group := range haGroups {
		hostGroupId, ok := account["hostGroupId"].(string)
		if ok && group["id"].(string) == hostGroupId {
			hostGroupName = group["name"].(string)
			break
		}
	}
	// Map response body to resource schema attribute
	result := Account{
		Name:          types.String{Value: account["displayName"].(string)},
		HostGroupName: types.String{Value: hostGroupName},
		HostGroupId:   types.String{Value: account["hostGroupId"].(string)},
		PropagationLabels: types.Set{
			Unknown:  false,
			Null:     false,
			Elems:    propagationLabels,
			ElemType: types.StringType,
		},
		AccountRoles: types.Set{
			Unknown:  false,
			Null:     false,
			Elems:    roles,
			ElemType: types.StringType,
		},
		Id:      types.String{Value: account["id"].(string)},
		Timeout: plan.Timeout,
	}

	// Set state
	diags = resp.State.Set(ctx, &result)
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

	accName := "acc_" + state.Name.Value

	err := resource.RetryContext(ctx, 300*time.Second, func() *resource.RetryError {
		// Get account current value
		account, _, _ := r.p.client.DefaultApi.GetAccount(ctx, accName).Execute()
		if account != nil {
			_, httpResponse, err := r.p.client.DefaultApi.DeleteAccount(ctx, accName).Execute()
			if err != nil {
				log.Println(err.Error())
				if httpResponse != nil {
					body, _ := io.ReadAll(httpResponse.Body)
					payload, _ := io.ReadAll(httpResponse.Request.Body)
					log.Printf("code: %d status: %s body: %s payload: %s\n", httpResponse.StatusCode, httpResponse.Status, string(body), string(payload))
				}
				return resource.RetryableError(fmt.Errorf("error deleting instance: %s", err))
			}
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting account",
			"Could not delete account: "+err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r resourceAccount) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	accName := "acc_" + req.ID
	// Get account current value
	account, _, err := r.p.client.DefaultApi.GetAccount(ctx, accName).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting account",
			"Could not read account "+accName+": "+err.Error(),
		)
		return
	}

	var propagationLabels []attr.Value
	if account["propagationLabels"] == nil {
		propagationLabels = []attr.Value{}
	} else {
		for _, prop := range account["propagationLabels"].([]interface{}) {
			propagationLabels = append(propagationLabels, types.String{
				Unknown: false,
				Null:    false,
				Value:   prop.(string),
			})
		}
	}

	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing account details",
			"Could not read account details"+err.Error(),
		)
		return
	}
	var roles []attr.Value
	for _, detail := range details {
		castDetail := detail.(map[string]interface{})
		if account["name"].(string) == castDetail["name"].(string) {
			roleObjects := castDetail["roles"].([]interface{})
			for _, roleObject := range roleObjects {
				roleName := roleObject.(map[string]interface{})["name"]
				roles = append(roles, types.String{
					Unknown: false,
					Null:    false,
					Value:   roleName.(string),
				})
			}
		}
	}
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}
	var hostGroupName = ""
	for _, group := range haGroups {
		hostGroupId, ok := account["hostGroupId"].(string)
		if ok && group["id"].(string) == hostGroupId {
			hostGroupName = group["name"].(string)
			break
		}
	}
	// Map response body to resource schema attribute
	var state = Account{
		Name:          types.String{Value: account["displayName"].(string)},
		HostGroupName: types.String{Value: hostGroupName},
		PropagationLabels: types.Set{
			Unknown:  false,
			Null:     false,
			Elems:    propagationLabels,
			ElemType: types.StringType,
		},
		AccountRoles: types.Set{
			Unknown:  false,
			Null:     false,
			Elems:    roles,
			ElemType: types.StringType,
		},
		Id:      types.String{Value: account["id"].(string)},
		Timeout: types.Int64{Value: 900},
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
