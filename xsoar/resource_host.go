package xsoar

import (
	"context"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type resourceHostType struct{}

// GetSchema Resource schema
func (r resourceHostType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	var planModifiers []tfsdk.AttributePlanModifier
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Type:          types.StringType,
				Required:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
				Optional: false,
			},
			"ha_group_name": {
				Type:          types.StringType,
				Optional:      true,
				PlanModifiers: append(planModifiers, tfsdk.RequiresReplace()),
			},
			"elasticsearch_url": {
				Type:          types.StringType,
				Computed:      true,
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
	if isHA || len(plan.ElasticsearchUrl.Value) > 0 {
		isElastic = true
	} else {
		isElastic = false
	}
	// Creation is a multi-step process
	// 1) Trigger or confirm the build of the host installer
	var haGroupId string
	if isHA {
		haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
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
		_, _, err = r.p.client.DefaultApi.CreateHAInstaller(ctx, haGroupId).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating HA installer",
				"Could not create HA installer: "+err.Error(),
			)
			return
		}
	} else {
		_, _, err := r.p.client.DefaultApi.CreateHostInstaller(ctx).Execute()
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
		installer, _, err = r.p.client.DefaultApi.GetHAInstaller(ctx, haGroupId).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error downloading HA installer",
				"Could not download HA installer: "+err.Error(),
			)
			return
		}
	} else {
		installer, _, err = r.p.client.DefaultApi.GetHostInstaller(ctx).Execute()
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
	var host map[string]interface{}
	c1 := make(chan map[string]interface{}, 1)
	go func() {
		for host == nil {
			host, _, err = r.p.client.DefaultApi.GetHost(ctx, plan.Name.Value).Execute()
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

	var result Host
	result = Host{
		Name: types.String{Value: hostName},
		Id:   types.String{Value: hostId},
	}
	if isHA {
		var haGroup openapi.CreateUpdateHAGroup
		haGroup, _, err = r.p.client.DefaultApi.GetHAGroup(ctx, haGroupId).Execute()
		result.HAGroupName.Value = haGroup.GetName()
	}
	if isElastic {
		var elasticsearchAddress string
		elasticsearchAddress = host["elasticsearchAddress"].(string)
		if !isHA || (!isHA && len(plan.ElasticsearchUrl.Value) > 0) {
			result.ElasticsearchUrl.Value = elasticsearchAddress
		}
	}
	result.ServerUrl = plan.ServerUrl
	result.SSHUser = plan.SSHUser
	result.SSHKeyFile = plan.SSHKeyFile

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r resourceHost) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Get current state
	var state Host
	diags := req.State.Get(ctx, &state)
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

	var result Host
	result = Host{
		Name: types.String{Value: hostName},
		Id:   types.String{Value: hostId},
	}

	haGroupId := host["hostGroupId"].(string)
	haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(ctx, haGroupId).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting HA group",
			"Could not get HA group: "+err.Error(),
		)
		return
	}

	var isHA = false
	if host["host"].(string) != haGroup.GetName() {
		isHA = true
	}
	if isHA {
		result.HAGroupName.Value = haGroup.GetName()
	}

	var isElastic = false
	if len(host["elasticsearchAddress"].(string)) > 0 {
		isElastic = true
	}
	if isElastic {
		var elasticsearchAddress string
		elasticsearchAddress = host["elasticsearchAddress"].(string)
		if !isHA || (!isHA && len(state.ElasticsearchUrl.Value) > 0) {
			result.ElasticsearchUrl.Value = elasticsearchAddress
		}
	}

	result.ServerUrl = state.ServerUrl
	result.SSHUser = state.SSHUser
	result.SSHKeyFile = state.SSHKeyFile

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r resourceHost) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan values
	var plan Host
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state Host
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Most attributes require a resource to be recreated,
	// the only attributes which are changeable are ones not available through the API about the host itself
	result := plan
	result.Id = state.Id

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r resourceHost) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state Host
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var isHA bool
	if !state.HAGroupName.Null && len(state.HAGroupName.Value) > 0 {
		isHA = true
	} else {
		isHA = false
	}

	// Delete Host by calling API
	// 1) Trigger or confirm the build of the host installer
	var haGroupId string
	if isHA {
		haGroups, _, err := r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error listing HA groups",
				"Could not list HA groups: "+err.Error(),
			)
			return
		}
		for _, group := range haGroups {
			if group["name"].(string) == state.HAGroupName.Value {
				haGroupId = group["id"].(string)
			}
		}
		_, _, err = r.p.client.DefaultApi.CreateHAInstaller(ctx, haGroupId).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating HA installer",
				"Could not create HA installer: "+err.Error(),
			)
			return
		}
	} else {
		_, _, err := r.p.client.DefaultApi.CreateHostInstaller(ctx).Execute()
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
		installer, _, err = r.p.client.DefaultApi.GetHAInstaller(ctx, haGroupId).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error downloading HA installer",
				"Could not download HA installer: "+err.Error(),
			)
			return
		}
	} else {
		installer, _, err = r.p.client.DefaultApi.GetHostInstaller(ctx).Execute()
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
		state.SSHUser.Value,
		state.SSHKeyFile.Value,
		ssh.InsecureIgnoreHostKey(),
	)
	client := scp.NewClient(state.ServerUrl.Value, &clientConfig)

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
	puKeyFile, err := ioutil.ReadFile(state.SSHKeyFile.Value)
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
		User: state.SSHUser.Value,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(pubKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", state.ServerUrl.Value, config)
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

	err = session.Run("sudo /tmp/installer.sh -- -purge -y")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error running installer",
			"Could not run installer: "+err.Error(),
		)
		return
	}

	// Delete host from main
	_, _, err = r.p.client.DefaultApi.DeleteHost(ctx, state.Id.Value).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting host",
			"Could not delete host: "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

func (r resourceHost) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	var diags diag.Diagnostics
	name := req.ID

	var host map[string]interface{}
	var err error
	c1 := make(chan map[string]interface{}, 1)
	go func() {
		for host == nil {
			host, _, err = r.p.client.DefaultApi.GetHost(ctx, name).Execute()
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

	var hostName = host["host"].(string)
	var hostId = host["id"].(string)
	var hostGroupId = host["hostGroupId"].(string)

	haGroup, _, err := r.p.client.DefaultApi.GetHAGroup(ctx, hostGroupId).Execute()
	var haGroupName = haGroup.GetName()

	// Map response body to resource schema attribute
	var result Host
	result = Host{
		Name: types.String{Value: hostName},
		Id:   types.String{Value: hostId},
	}

	if haGroupName != hostName {
		result.HAGroupName = types.String{Value: haGroupName}
	} else {
		result.ElasticsearchUrl = types.String{Value: host["elasticsearchAddress"].(string)}
	}

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
