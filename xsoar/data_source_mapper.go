package xsoar

import (
	"context"
	"encoding/json"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io"
	"log"
	"net/http"
)

type dataSourceMapperType struct{}

func (r dataSourceMapperType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
				Computed: true,
				Optional: false,
			},
			"propagation_labels": {
				Type:     types.ListType{ElemType: types.StringType},
				Computed: true,
				Optional: false,
			},
			"account": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"direction": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
		},
	}, nil
}

func (r dataSourceMapperType) NewDataSource(_ context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return dataSourceMapper{
		p: *(p.(*provider)),
	}, nil
}

type dataSourceMapper struct {
	p provider
}

func (r dataSourceMapper) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	// Declare struct that this function will set to this data source's config
	var config Mapper
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get resource from API
	var mapper openapi.InstanceClassifier
	var httpResponse *http.Response
	var err error
	if config.Account.Null || len(config.Account.Value) == 0 {
		mapper, httpResponse, err = r.p.client.DefaultApi.GetClassifier(ctx).SetIdentifier(config.Name.Value).Execute()
	} else {
		mapper, httpResponse, err = r.p.client.DefaultApi.GetClassifierAccount(ctx, "acc_"+config.Account.Value).SetIdentifier(config.Name.Value).Execute()
	}
	if err != nil {
		log.Println(err.Error())
		if httpResponse != nil {
			getBody, _ := httpResponse.Request.GetBody()
			b, _ := io.ReadAll(getBody)
			log.Println(string(b))
		}
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
		PropagationLabels: types.Set{Elems: propLabels, ElemType: types.StringType},
		Account:           config.Account,
		Direction:         config.Direction,
	}
	if m := string(mapping); m == "null" {
		result.Mapping = types.String{Null: true}
	} else {
		result.Mapping = types.String{Value: m}
	}

	// Set state
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
