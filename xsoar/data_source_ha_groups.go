package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ryanuber/go-glob"
)

type dataSourceHAGroupsType struct{}

func (r dataSourceHAGroupsType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Type:     types.StringType,
				Optional: true,
			},
			"max_accounts": {
				Type:     types.Int64Type,
				Optional: true,
			},
			"groups": {
				Type: types.SetType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"name":                 types.StringType,
							"id":                   types.StringType,
							"elasticsearch_url":    types.StringType,
							"elastic_index_prefix": types.StringType,
							"account_ids":          types.SetType{ElemType: types.StringType},
							"host_ids":             types.SetType{ElemType: types.StringType},
						},
					},
				},
				Computed: true,
			},
		},
	}, nil
}

func (r dataSourceHAGroupsType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceHAGroups{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceHAGroups struct {
	p provider
}

func (r dataSourceHAGroups) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Declare struct that this function will set to this data source's config
	var config HAGroups
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get HA group from API and then update what is in config from what the API returns
	haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not list HA groups: "+err.Error(),
		)
		return
	}

	var haGroupsGroups = types.Set{
		Unknown: false,
		Null:    false,
		Elems:   nil,
		ElemType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":                 types.StringType,
				"id":                   types.StringType,
				"elasticsearch_url":    types.StringType,
				"elastic_index_prefix": types.StringType,
				"account_ids":          types.SetType{ElemType: types.StringType},
				"host_ids":             types.SetType{ElemType: types.StringType},
			},
		},
	}
	for _, group := range haGroups {
		if !config.Name.Null {
			if !glob.Glob(config.Name.Value, group["name"].(string)) {
				continue
			}
		}
		if !config.MaxAccounts.Null {
			accountIds, ok := group["accountIds"].([]interface{})
			if ok {
				if int64(len(accountIds)) > config.MaxAccounts.Value {
					continue
				}
			}
		}
		haGroupsGroups.Null = false
		// initialize the group object
		groupObject := types.Object{
			Unknown: false,
			Null:    false,
			Attrs: map[string]attr.Value{
				"name":                 types.String{Null: true},
				"id":                   types.String{Null: true},
				"elasticsearch_url":    types.String{Null: true},
				"elastic_index_prefix": types.String{Null: true},
				"account_ids": types.Set{
					Unknown:  false,
					Null:     false,
					Elems:    nil,
					ElemType: types.StringType,
				},
				"host_ids": types.Set{
					Unknown:  false,
					Null:     false,
					Elems:    nil,
					ElemType: types.StringType,
				},
			},
			AttrTypes: map[string]attr.Type{
				"name":                 types.StringType,
				"id":                   types.StringType,
				"elasticsearch_url":    types.StringType,
				"elastic_index_prefix": types.StringType,
				"account_ids":          types.SetType{ElemType: types.StringType},
				"host_ids":             types.SetType{ElemType: types.StringType},
			},
		}
		// assign the values from the response to the object
		groupName, ok := group["name"].(string)
		if ok {
			groupObject.Attrs["name"] = types.String{Value: groupName}
		}
		groupId, ok := group["id"].(string)
		if ok {
			groupObject.Attrs["id"] = types.String{Value: groupId}
		}
		groupElasticsearchAddress, ok := group["elasticsearchAddress"].(string)
		if ok {
			groupObject.Attrs["elasticsearch_url"] = types.String{Value: groupElasticsearchAddress}
		}
		groupElasticIndexPrefix, ok := group["elasticIndexPrefix"].(string)
		if ok {
			groupObject.Attrs["elastic_index_prefix"] = types.String{Value: groupElasticIndexPrefix}
		}
		groupAccountIds, ok := group["accountIds"].([]interface{})
		if ok {
			var elems []attr.Value
			for _, a := range groupAccountIds {
				elems = append(elems, types.String{Value: a.(string)})
			}
			groupObject.Attrs["account_ids"] = types.Set{
				Unknown:  false,
				Null:     false,
				Elems:    elems,
				ElemType: types.StringType,
			}
		}
		groupHostIds, ok := group["hostIds"].([]interface{})
		if ok {
			var elems []attr.Value
			for _, h := range groupHostIds {
				elems = append(elems, types.String{Value: h.(string)})
			}

			groupObject.Attrs["host_ids"] = types.Set{
				Unknown:  false,
				Null:     false,
				Elems:    elems,
				ElemType: types.StringType,
			}
		}
		haGroupsGroups.Elems = append(haGroupsGroups.Elems, groupObject)
	}

	var result HAGroups
	result = HAGroups{
		Name:        config.Name,
		MaxAccounts: config.MaxAccounts,
		Groups:      haGroupsGroups,
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
