package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type dataSourceHAGroupType struct{}

func (r dataSourceHAGroupType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
				Type:     types.StringType,
				Computed: true,
			},
			"elastic_index_prefix": {
				Type:     types.StringType,
				Computed: true,
			},
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

func (r dataSourceHAGroupType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceHAGroup{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceHAGroup struct {
	p provider
}

func (r dataSourceHAGroup) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Declare struct that this function will set to this data source's config
	var config HAGroup
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
	var haGroupId string
	for _, group := range haGroups {
		if config.Name.Value == group["name"].(string) {
			haGroupId = group["id"].(string)
			break
		}
	}
	haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(ctx, haGroupId).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting HA group",
			"Could not get HA group "+config.Name.Value+": "+err.Error(),
		)
		return
	}

	// Map response body to resource schema attribute
	config = HAGroup{
		Name:               types.String{Value: haGroup.GetName()},
		Id:                 types.String{Value: haGroup.GetId()},
		ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
		ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
		AccountIds: types.List{
			Unknown:  false,
			Null:     false,
			Elems:    nil,
			ElemType: nil,
		},
		HostIds: types.List{
			Unknown:  false,
			Null:     false,
			Elems:    nil,
			ElemType: nil,
		},
	}

	// Set state
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
