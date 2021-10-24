package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"time"
)

type dataSourceHostType struct{}

func (r dataSourceHostType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
			"ha_group_name": {
				Type:     types.StringType,
				Computed: true,
				Optional: true,
			},
			"elasticsearch_url": {
				Type:     types.StringType,
				Computed: true,
				Optional: true,
			},
			"server_url": {
				Type:     types.StringType,
				Computed: false,
				Optional: true,
			},
			"ssh_user": {
				Type:     types.StringType,
				Computed: false,
				Optional: true,
			},
			"ssh_key_file": {
				Type:     types.StringType,
				Computed: false,
				Optional: true,
			},
		},
	}, nil
}

func (r dataSourceHostType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceHost{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceHost struct {
	p provider
}

func (r dataSourceHost) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Declare struct that this function will set to this data source's state
	var state Host
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var host map[string]interface{}
	var err error
	c1 := make(chan map[string]interface{}, 1)
	go func() {
		for host == nil {
			host, _, err = r.p.client.DefaultApi.GetHost(ctx, state.Name.Value).Execute()
			if err != nil {
				resp.Diagnostics.AddError(
					"Error listing HA groups",
					"Could not list HA groups: "+err.Error(),
				)
				return
			}
			time.Sleep(time.Second)
		}
		c1 <- host
	}()
	select {
	case _ = <-c1:
		break
	case <-time.After(60 * time.Second):
		resp.Diagnostics.AddError(
			"Error getting host",
			"Could not get host before timeout",
		)
		return
	}

	// Map response body to resource schema attribute
	var hostName = host["host"].(string)
	var hostId = host["id"].(string)
	var haGroupId = host["hostGroupId"].(string)

	haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(ctx, haGroupId).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting HA group",
			"Could not get HA group: "+err.Error(),
		)
		return
	}

	var result Host
	result = Host{
		Name: types.String{Value: hostName},
		Id:   types.String{Value: hostId},
	}

	var isHA = false
	if host["host"].(string) != haGroup.GetName() {
		isHA = true
		result.HAGroupName.Value = haGroup.GetName()
	} else {
		result.HAGroupName.Null = true
	}

	if len(host["elasticsearchAddress"].(string)) > 0 {
		if isHA {
			result.ElasticsearchUrl.Null = true
		} else {
			result.ElasticsearchUrl.Value = host["elasticsearchAddress"].(string)
		}
	} else {
		result.ElasticsearchUrl.Null = true
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
