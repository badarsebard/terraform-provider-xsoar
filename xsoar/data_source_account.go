package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceAccountType struct{}

func (r dataSourceAccountType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Type:     types.StringType,
				Required: true,
			},
			"host_group_name": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"host_group_id": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"account_roles": {
				Type:     types.ListType{ElemType: types.StringType},
				Computed: true,
				Optional: false,
			},
			"propagation_labels": {
				Type:     types.ListType{ElemType: types.StringType},
				Computed: true,
				Optional: false,
			},
		},
	}, nil
}

func (r dataSourceAccountType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceAccount{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceAccount struct {
	p provider
}

func (r dataSourceAccount) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Declare struct that this function will set to this data source's state
	var state Account
	diags := req.Config.Get(ctx, &state)
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

	var propagationLabels = []string{}
	for _, prop := range account["propagationLabels"].([]interface{}) {
		propagationLabels = append(propagationLabels, prop.(string))
	}

	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(ctx).Execute()
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
