package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceAccountsType struct{}

func (r dataSourceAccountsType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"accounts": {
				Type: types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"name":               types.StringType,
							"host_group_name":    types.StringType,
							"host_group_id":      types.StringType,
							"account_roles":      types.ListType{ElemType: types.StringType},
							"propagation_labels": types.ListType{ElemType: types.StringType},
							"id":                 types.StringType,
						},
					},
				},
				Computed: true,
			},
		},
	}, nil
}

func (r dataSourceAccountsType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceAccounts{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceAccounts struct {
	p provider
}

func (r dataSourceAccounts) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Declare struct that this function will set to this data source's config
	var config Account
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get accounts current value
	accounts, _, err := r.p.client.DefaultApi.ListAccounts(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting accounts",
			"Could not read accounts: "+err.Error(),
		)
		return
	}
	details, _, err := r.p.client.DefaultApi.ListAccountsDetails(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing account details",
			"Could not read account details"+err.Error(),
		)
		return
	}
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not read HA groups"+err.Error(),
		)
		return
	}

	var accountsAccounts = types.List{
		Unknown: false,
		Null:    false,
		Elems:   nil,
		ElemType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":               types.StringType,
				"host_group_name":    types.StringType,
				"host_group_id":      types.StringType,
				"account_roles":      types.ListType{ElemType: types.StringType},
				"propagation_labels": types.ListType{ElemType: types.StringType},
				"id":                 types.StringType,
			},
		},
	}

	for _, account := range accounts {
		accountObject := types.Object{
			Unknown: false,
			Null:    false,
			Attrs: map[string]attr.Value{
				"name":               types.String{Null: true},
				"host_group_name":    types.String{Null: true},
				"host_group_id":      types.String{Null: true},
				"account_roles":      types.String{Null: true},
				"propagation_labels": types.String{Null: true},
				"id":                 types.String{Null: true},
			},
			AttrTypes: map[string]attr.Type{
				"name":               types.StringType,
				"host_group_name":    types.StringType,
				"host_group_id":      types.StringType,
				"account_roles":      types.ListType{ElemType: types.StringType},
				"propagation_labels": types.ListType{ElemType: types.StringType},
				"id":                 types.StringType,
			},
		}
		// assign the values from the response to the object
		accountName, ok := account["name"].(string)
		if ok {
			accountObject.Attrs["name"] = types.String{Value: accountName}
		}
		accountHostGroupId, ok := account["hostGroupId"].(string)
		if ok {
			accountObject.Attrs["host_group_id"] = types.String{Value: accountHostGroupId}
		}
		for _, group := range haGroups {
			if group["id"].(string) == accountHostGroupId {
				hostGroupName, ok := group["name"].(string)
				if ok {
					accountObject.Attrs["host_group_name"] = types.String{Value: hostGroupName}
					break
				}
			}
		}
		accountId, ok := account["id"].(string)
		if ok {
			accountObject.Attrs["id"] = types.String{Value: accountId}
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
		accountObject.Attrs["propagation_labels"] = types.List{
			Unknown:  false,
			Null:     false,
			Elems:    propagationLabels,
			ElemType: types.StringType,
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
		accountObject.Attrs["account_roles"] = types.List{
			Unknown:  false,
			Null:     false,
			Elems:    roles,
			ElemType: types.StringType,
		}
	}

	var result Accounts
	result = Accounts{
		Accounts: accountsAccounts,
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
