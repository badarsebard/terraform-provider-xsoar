package xsoar

import (
	"context"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type resourceHostType struct{}

// GetSchema Resource schema
func (r resourceHostType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
			"ha_group_name": {
				Type:     types.StringType,
				Optional: true,
			},
			"elasticsearch_url": {
				Type:          types.StringType,
				Optional:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			// todo: add in other elastic settings, especially index prefix
			"server_url": {
				Type:     types.StringType,
				Required: true,
			},
			"ssh_user": {
				Type:     types.StringType,
				Required: true,
			},
			"ssh_key_file": {
				Type:     types.StringType,
				Required: true,
			},
		},
	}, nil
}

// NewResource instance
func (r resourceHostType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return resourceHost{
		p: *(p.(*provider)),
	}, nil
}

type resourceHost struct {
	p provider
}

// Create a new resource
func (r resourceHost) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	if !r.p.configured {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Retrieve values from plan
	var plan Host
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var isHA bool
	if !plan.HAGroupName.Null && len(plan.HAGroupName.Value) > 0 {
		isHA = true
	} else {
		isHA = false
	}

	var isElastic bool
	if !plan.ElasticsearchUrl.Null && len(plan.ElasticsearchUrl.Value) > 0 {
		isElastic = true
	} else {
		isElastic = false
	}
	// Creation is a multi-step process
	// 1) Trigger or confirm the build of the host installer
	var haGroupId string
	if isHA {
		haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(context.Background()).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing HA groups",
				"Could not list HA groups: "+err.Error(),
			)
			return
		}
		for _, group := range haGroups {
			if group["name"].(string) == plan.HAGroupName.Value {
				haGroupId = group["id"].(string)
			}
		}
		_, _, err = r.p.client.DefaultApi.CreateHAInstaller(context.Background(), haGroupId).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating HA installer",
				"Could not create HA installer: "+err.Error(),
			)
			return
		}
	} else {
		_, _, err := r.p.client.DefaultApi.CreateHostInstaller(context.Background()).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating host installer",
				"Could not create host installer: "+err.Error(),
			)
			return
		}
	}

	// 2) Download the installer
	var installer *os.File
	var err error
	if isHA {
		installer, _, err = r.p.client.DefaultApi.GetHAInstaller(context.Background(), haGroupId).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error downloading HA installer",
				"Could not download HA installer: "+err.Error(),
			)
			return
		}
	} else {
		installer, _, err = r.p.client.DefaultApi.GetHostInstaller(context.Background()).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error downloading host installer",
				"Could not download host installer: "+err.Error(),
			)
			return
		}
	}

	// 3) Transfer installer to host server
	clientConfig, _ := auth.PrivateKey(
		plan.SSHUser.Value,
		plan.SSHKeyFile.Value,
		ssh.InsecureIgnoreHostKey(),
	)
	client := scp.NewClient(plan.ServerUrl.Value, &clientConfig)

	err = client.Connect()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating scp connection",
			"Could not create scp connection: "+err.Error(),
		)
		return
	}
	defer client.Close()

	err = client.CopyFile(installer, "/tmp/installer.sh", "0755")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error copying file",
			"Could not copy file: "+err.Error(),
		)
		return
	}

	// 4) Execute installer
	puKeyFile, err := ioutil.ReadFile(plan.SSHKeyFile.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading key file",
			"Could not read key file: "+err.Error(),
		)
		return
	}
	pubKey, err := ssh.ParsePrivateKey(puKeyFile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing key file",
			"Could not parse key file: "+err.Error(),
		)
		return
	}
	config := &ssh.ClientConfig{
		User: plan.SSHUser.Value,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(pubKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", plan.ServerUrl.Value, config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ssh connection",
			"Could not create ssh connection: "+err.Error(),
		)
		return
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ssh session",
			"Could not create ssh session: "+err.Error(),
		)
		return
	}
	defer session.Close()

	var args = []string{
		"-y",
		"-external-address=" + plan.Name.Value,
	}
	if isElastic && !isHA {
		args = append(args, "-elasticsearch-url="+plan.ElasticsearchUrl.Value)
	}
	argsString := strings.Join(args, " ")

	err = session.Run("sudo /tmp/installer.sh -- " + argsString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error running installer",
			"Could not run installer: "+err.Error(),
		)
		return
	}

	// Verify host details
	// todo: this needs to wait a bit until the host can register with main; must crete retry/timeout logic
	hosts, _, err := r.p.client.DefaultApi.ListHosts(context.Background()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing HA groups",
			"Could not list HA groups: "+err.Error(),
		)
		return
	}
	var hostName string
	var hostId string
	for _, host := range hosts {
		log.Println(host)
		if host["host"].(string) == plan.Name.Value {
			hostName = host["host"].(string)
			hostId = host["id"].(string)
		}
	}

	// Map response body to resource schema attribute
	var result Host
	result = Host{
		Name:             types.String{Value: hostName},
		Id:               types.String{Value: hostId},
		HAGroupName:      plan.HAGroupName,
		ElasticsearchUrl: plan.ElasticsearchUrl,
		ServerUrl:        plan.ServerUrl,
		SSHUser:          plan.SSHUser,
		SSHKeyFile:       plan.SSHKeyFile,
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceHost) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	//// Get current state
	//var state HAGroup
	//diags := req.State.Get(ctx, &state)
	//resp.Diagnostics.Append(diags...)
	//if resp.Diagnostics.HasError() {
	//	return
	//}
	//
	//// Get HA group from API and then update what is in state from what the API returns
	//haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(context.Background(), state.Id.Value).Execute()
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		"Error getting HA group",
	//		"Could not get HA group "+state.Name.Value+": "+err.Error(),
	//	)
	//	return
	//}
	//
	//// Map response body to resource schema attribute
	//state = HAGroup{
	//	Name:               types.String{Value: haGroup.GetName()},
	//	Id:                 types.String{Value: haGroup.GetId()},
	//	ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
	//	ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
	//}
	//
	//// Set state
	//diags = resp.State.Set(ctx, &state)
	//resp.Diagnostics.Append(diags...)
	//if resp.Diagnostics.HasError() {
	//	return
	//}
}

// Update resource
func (r resourceHost) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	//// Get plan values
	//var plan HAGroup
	//diags := req.Plan.Get(ctx, &plan)
	//resp.Diagnostics.Append(diags...)
	//if resp.Diagnostics.HasError() {
	//	return
	//}
	//
	//// Get current state
	//var state HAGroup
	//diags = req.State.Get(ctx, &state)
	//resp.Diagnostics.Append(diags...)
	//if resp.Diagnostics.HasError() {
	//	return
	//}
	//
	//// Generate API request body from plan
	//updateHAGroupRequest := *openapi.NewCreateHAGroupRequest()
	//updateHAGroupRequest.SetId(state.Id.Value)
	//updateHAGroupRequest.SetName(plan.Name.Value)
	//updateHAGroupRequest.SetElasticsearchAddress(plan.ElasticsearchUrl.Value)
	//updateHAGroupRequest.SetElasticIndexPrefix(plan.ElasticIndexPrefix.Value)
	//haGroup, _, err := r.p.client.DefaultApi.CreateHAGroup(context.Background()).CreateHAGroupRequest(updateHAGroupRequest).Execute()
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		"Error updating HA group",
	//		"Could not update HA group "+plan.Name.Value+": "+err.Error(),
	//	)
	//	return
	//}
	//
	//// Map response body to resource schema attribute
	//result := Host{
	//	Name:               types.String{Value: haGroup.GetName()},
	//	Id:                 types.String{Value: haGroup.GetId()},
	//	HAG
	//	ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
	//	ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
	//}
	//
	//// Set state
	//diags = resp.State.Set(ctx, result)
	//resp.Diagnostics.Append(diags...)
	//if resp.Diagnostics.HasError() {
	//	return
	//}
}

// Delete resource
func (r resourceHost) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	//var state HAGroup
	//diags := req.State.Get(ctx, &state)
	//resp.Diagnostics.Append(diags...)
	//if resp.Diagnostics.HasError() {
	//	return
	//}
	//
	//// Delete HA group by calling API
	//_, _, err := r.p.client.DefaultApi.DeleteHAGroup(context.Background(), state.Id.Value).Execute()
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		"Error deleting HA group",
	//		"Could not delete HA group "+state.Name.Value+": "+err.Error(),
	//	)
	//	return
	//}
	//
	//// Remove resource from state
	//resp.State.RemoveResource(ctx)
}

func (r resourceHost) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	//var diags diag.Diagnostics
	//name := req.ID
	//// Get HA group current value
	//haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(context.Background()).Execute()
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		"Error listing HA groups",
	//		"Could not read HA groups"+err.Error(),
	//	)
	//	return
	//}
	//var id string
	//for _, group := range haGroups {
	//	if group["name"].(string) == name {
	//		id = group["id"].(string)
	//		break
	//	}
	//}
	//haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(context.Background(), id).Execute()
	//if err != nil {
	//	resp.Diagnostics.AddError(
	//		"Error getting HA group",
	//		"Could not read HA group "+name+": "+err.Error(),
	//	)
	//	return
	//}
	//
	//// Map response body to resource schema attribute
	//var state = HAGroup{
	//	Name:               types.String{Value: haGroup.GetName()},
	//	Id:                 types.String{Value: haGroup.GetId()},
	//	ElasticsearchUrl:   types.String{Value: haGroup.GetElasticsearchAddress()},
	//	ElasticIndexPrefix: types.String{Value: haGroup.GetElasticIndexPrefix()},
	//}
	//
	//// Set state
	//diags = resp.State.Set(ctx, &state)
	//resp.Diagnostics.Append(diags...)
	//if resp.Diagnostics.HasError() {
	//	return
	//}
}
