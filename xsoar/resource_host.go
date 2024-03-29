package xsoar

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"golang.org/x/crypto/ssh"
	"hash/crc64"
	"io"
	"log"
	"math/rand"
	"net/http"
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
			"nfs_mount": {
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
			"ssh_key": {
				Type:      types.StringType,
				Required:  true,
				Sensitive: true,
			},
			"installation_timeout": {
				Type:     types.Int64Type,
				Optional: true,
			},
			"extra_flags": {
				Type:     types.ListType{ElemType: types.StringType},
				Optional: true,
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
	log.Println("Starting create")
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
	log.Printf("%+v\n", plan)

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

	// 1) connect to host server over ssh
	apikey := r.p.data.Apikey
	mainhost := r.p.data.MainHost
	signer, _ := ssh.ParsePrivateKey([]byte(plan.SSHKey.Value))
	clientConfig := ssh.ClientConfig{
		User: plan.SSHUser.Value,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var conn *ssh.Client
	var err error
	err = resource.RetryContext(ctx, 300*time.Second, func() *resource.RetryError {
		var conErr error
		conn, conErr = ssh.Dial("tcp", plan.ServerUrl.Value, &clientConfig)
		if conErr != nil {
			return resource.RetryableError(fmt.Errorf("error connecting to host over ssh: " + conErr.Error()))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating host",
			fmt.Sprintf("Error creating host: %s", err.Error()),
		)
		return
	}
	defer conn.Close()

	// 2) query main server with /host/build
	var haGroup string
	var httpResponse *http.Response
	if isHA {
		var haGroups []map[string]interface{}
		var haGroupId string
		log.Println("List ha groups")
		haGroups, _, err = r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
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
				haGroup = "/" + haGroupId
			}
		}
		_, httpResponse, err = r.p.client.DefaultApi.CreateHAInstaller(ctx, haGroupId).Execute()
		if err != nil {
			log.Println(err.Error())
			body, _ := io.ReadAll(httpResponse.Body)
			log.Printf("code: %d status: %s body: %s\n", httpResponse.StatusCode, httpResponse.Status, string(body))
			i := bytes.Index(body, []byte("Already building host for ha group"))
			if i > -1 {
				for true {
					_, httpResponse, err = r.p.client.DefaultApi.CreateHAInstaller(ctx, haGroupId).Execute()
					if err == nil {
						break
					}
				}
			} else {
				resp.Diagnostics.AddError(
					"Error creating HA installer",
					"Could not create HA installer: "+err.Error(),
				)
				return
			}
		}
	} else {
		_, httpResponse, err = r.p.client.DefaultApi.CreateHostInstaller(ctx).Execute()
		if err != nil {
			log.Println(err.Error())
			body, bodyErr := io.ReadAll(httpResponse.Body)
			if bodyErr != nil {
				log.Println("error reading body: " + bodyErr.Error())
				return
			}
			log.Printf("code: %d status: %s body: %s\n", httpResponse.StatusCode, httpResponse.Status, string(body))
			i := bytes.Index(body, []byte("Already building host installer"))
			if i > -1 {
				for true {
					_, httpResponse, err = r.p.client.DefaultApi.CreateHostInstaller(ctx).Execute()
					if err == nil {
						break
					}
				}
			} else {
				resp.Diagnostics.AddError(
					"Error creating host installer",
					"Could not create host installer: "+err.Error(),
				)
				return
			}
		}
	}

	// 3) download installer
	session, err := conn.NewSession()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ssh session",
			"Could not create ssh session: "+err.Error(),
		)
		return
	}
	defer session.Close()

	insecure := ""
	if r.p.data.Insecure.Value {
		insecure = "-k"
	}
	cmd := fmt.Sprintf(
		"sudo curl -s -o '/tmp/installer.sh' -H 'Authorization: %s' %s %s/host/download%s && "+
			"sudo chmod +x /tmp/installer.sh",
		apikey.Value, insecure, mainhost.Value, haGroup)
	err = session.Run(cmd)
	if err != nil {

		resp.Diagnostics.AddError(
			"Error downloading installer",
			"Could not download installer: "+err.Error(),
		)
		return
	}

	// 4) Check for lock
	if !plan.NFSMount.Null {
		// wait a random amount of time
		crcTable := crc64.MakeTable(crc64.ISO)
		seedInt := int64(crc64.Checksum([]byte(plan.Name.Value), crcTable))
		log.Printf("generated seed: %d\n", seedInt)
		randSource := rand.NewSource(seedInt)
		nrand := rand.New(randSource)
		randomTimeToWait := nrand.Intn(30) + 1
		log.Printf("sleeping for %d seconds\n", randomTimeToWait)
		time.Sleep(time.Duration(randomTimeToWait) * time.Second)
		// attempt to place lock
		session, err = conn.NewSession()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating ssh session",
				"Could not create ssh session: "+err.Error(),
			)
			return
		}
		defer session.Close()
		err = session.Run(fmt.Sprintf(
			`while [[ -f "%s/xsoar_host_install.lock" ]]; do sleep %d; done; sudo touch %s/xsoar_host_install.lock`,
			plan.NFSMount.Value, randomTimeToWait, plan.NFSMount.Value,
		))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for lock file",
				"Lock file error: "+err.Error(),
			)
			return
		}
	}

	// 5) Execute installer
	log.Println("Executing install")
	session, err = conn.NewSession()
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
		"-external-address='" + plan.Name.Value + "'",
	}
	if isElastic && !isHA {
		args = append(args, "-elasticsearch-url='"+plan.ElasticsearchUrl.Value+"'")
	}
	if isHA {
		args = append(args, "-temp-folder='/tmp/demisto'", "-ha")
	}
	if !plan.ExtraFlags.Null {
		var extraArgs []string
		flagErr := plan.ExtraFlags.ElementsAs(ctx, &extraArgs, false)
		if flagErr != nil {
			resp.Diagnostics.AddError(
				"Error extracting extra arguments",
				fmt.Sprintf("Could not extract %s into extraArgs with error: %s", plan.ExtraFlags.Elems, flagErr),
			)
			return
		}
		log.Printf("extra args: %s\n", extraArgs)
		// todo: there's a security flaw here where a user can inject arbitrary commands into the installer
		args = append(args, extraArgs...)
	}
	argsString := strings.Join(args, " ")

	err = session.Run("sudo /tmp/installer.sh -- " + argsString)
	log.Printf("args: %s", argsString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error running installer",
			"Could not run installer: "+err.Error(),
		)
		log.Println("remove lock file")
		session, err = conn.NewSession()
		defer session.Close()
		err = session.Run(fmt.Sprintf("sudo rm -f %s/xsoar_host_install.lock", plan.NFSMount.Value))
		if err != nil {
			log.Println("could not remove lock file")
			resp.Diagnostics.AddError(
				"Error removing lock file",
				"Could not remove lock file: "+err.Error(),
			)
		}
		return
	}

	// Verify host details
	log.Println("Verifying host details")
	var host = make(map[string]interface{})
	c1 := make(chan map[string]interface{}, 1)
	go func() {
		for len(host) == 0 || host["hostGroupId"] == "" {
			host, _, _ = r.p.client.DefaultApi.GetHost(ctx, plan.Name.Value).Execute()
			time.Sleep(time.Second)
		}
		c1 <- host
	}()
	timeout := time.Duration(300)
	if !plan.InstallationTimeout.Null {
		timeout = time.Duration(plan.InstallationTimeout.Value)
	}
	select {
	case _ = <-c1:
		log.Println(host)
		break
	case <-time.After(timeout * time.Second):
		resp.Diagnostics.AddError(
			"Error getting host",
			"Could not get host before timeout",
		)
		return
	}
	// delete lock file
	if !plan.NFSMount.Null {
		session, err = conn.NewSession()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating ssh session",
				"Could not create ssh session: "+err.Error(),
			)
			return
		}
		defer session.Close()
		err = session.Run(fmt.Sprintf(`sudo rm %s/xsoar_host_install.lock`, plan.NFSMount.Value))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting lock file",
				"Could not delete lock file: "+err.Error(),
			)
			return
		}
	}

	// Map response body to resource schema attribute
	var hostName = host["host"].(string)
	var hostId = host["id"].(string)
	var hostGroupId = host["hostGroupId"].(string)

	haGroupName, httpResponse, err := r.p.client.DefaultApi.GetHAGroup(ctx, hostGroupId).Execute()
	if err != nil {
		log.Println(err.Error())
		if httpResponse != nil {
			body, _ := io.ReadAll(httpResponse.Body)
			payload, _ := io.ReadAll(httpResponse.Request.Body)
			log.Printf("code: %d status: %s headers: %s body: %s payload: %s\n", httpResponse.StatusCode, httpResponse.Status, httpResponse.Header, string(body), string(payload))
		}
		resp.Diagnostics.AddError(
			"Error getting HA group",
			"Could not get HA group: "+err.Error(),
		)
		return
	}

	var result Host
	result = Host{
		Name:                types.String{Value: hostName},
		Id:                  types.String{Value: hostId},
		InstallationTimeout: plan.InstallationTimeout,
		ExtraFlags:          plan.ExtraFlags,
		NFSMount:            plan.NFSMount,
		ServerUrl:           plan.ServerUrl,
		SSHUser:             plan.SSHUser,
		SSHKey:              plan.SSHKey,
	}

	if host["host"].(string) != haGroupName.GetName() {
		result.HAGroupName.Value = haGroupName.GetName()
	} else {
		result.HAGroupName.Null = true
	}

	if len(host["elasticsearchAddress"].(string)) > 0 {
		result.ElasticsearchUrl.Value = host["elasticsearchAddress"].(string)
	} else {
		result.ElasticsearchUrl.Null = true
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
	var hostGroupId = host["hostGroupId"].(string)

	haGroupName, httpResponse, err := r.p.client.DefaultApi.GetHAGroup(ctx, hostGroupId).Execute()
	if err != nil {
		log.Println(err.Error())
		if httpResponse != nil {
			body, _ := io.ReadAll(httpResponse.Body)
			payload, _ := io.ReadAll(httpResponse.Request.Body)
			log.Printf("code: %d status: %s headers: %s body: %s payload: %s\n", httpResponse.StatusCode, httpResponse.Status, httpResponse.Header, string(body), string(payload))
		}
		resp.Diagnostics.AddError(
			"Error getting HA group",
			"Could not get HA group: "+err.Error(),
		)
		return
	}

	var result Host
	result = Host{
		Name:                types.String{Value: hostName},
		Id:                  types.String{Value: hostId},
		InstallationTimeout: state.InstallationTimeout,
		ExtraFlags:          state.ExtraFlags,
		NFSMount:            state.NFSMount,
		ServerUrl:           state.ServerUrl,
		SSHUser:             state.SSHUser,
		SSHKey:              state.SSHKey,
	}

	if host["host"].(string) != haGroupName.GetName() {
		result.HAGroupName.Value = haGroupName.GetName()
	} else {
		result.HAGroupName.Null = true
	}

	if len(host["elasticsearchAddress"].(string)) > 0 {
		result.ElasticsearchUrl.Value = host["elasticsearchAddress"].(string)
	} else {
		result.ElasticsearchUrl.Null = true
	}

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

	// Delete Host
	// 1) connect to host server over ssh
	apikey := r.p.data.Apikey
	mainhost := r.p.data.MainHost
	signer, _ := ssh.ParsePrivateKey([]byte(state.SSHKey.Value))
	clientConfig := ssh.ClientConfig{
		User: state.SSHUser.Value,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var conn *ssh.Client
	var err error
	err = resource.RetryContext(ctx, 300*time.Second, func() *resource.RetryError {
		var conErr error
		conn, conErr = ssh.Dial("tcp", state.ServerUrl.Value, &clientConfig)
		if conErr != nil {
			return resource.RetryableError(fmt.Errorf("error connecting to host over ssh: " + conErr.Error()))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting host",
			"Could not delete host: "+err.Error(),
		)
		return
	}
	defer conn.Close()

	// 2) query main server with /host/build
	var haGroup string
	var httpResponse *http.Response
	if isHA {
		var haGroups []map[string]interface{}
		var haGroupId string
		log.Println("List ha groups")
		haGroups, _, err = r.p.client.DefaultApi.ListHAGroups(ctx).Execute()
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
				haGroup = "/" + haGroupId
			}
		}
		_, httpResponse, err = r.p.client.DefaultApi.CreateHAInstaller(ctx, haGroupId).Execute()
		if err != nil {
			body, bodyErr := io.ReadAll(httpResponse.Body)
			if bodyErr != nil {
				log.Println("error reading body: " + bodyErr.Error())
				return
			}
			log.Printf("code: %d status: %s body: %s\n", httpResponse.StatusCode, httpResponse.Status, string(body))
			i := bytes.Index(body, []byte("Already building host for ha group"))
			if i > -1 {
				for true {
					_, httpResponse, err = r.p.client.DefaultApi.CreateHAInstaller(ctx, haGroupId).Execute()
					if err == nil {
						break
					}
				}
			} else {
				resp.Diagnostics.AddError(
					"Error creating HA installer",
					"Could not create HA installer: "+err.Error(),
				)
				return
			}
		}
	} else {
		_, _, err = r.p.client.DefaultApi.CreateHostInstaller(ctx).Execute()
		if err != nil {
			body, bodyErr := io.ReadAll(httpResponse.Body)
			if bodyErr != nil {
				log.Println("error reading body: " + bodyErr.Error())
				return
			}
			log.Printf("code: %d status: %s body: %s\n", httpResponse.StatusCode, httpResponse.Status, string(body))
			i := bytes.Index(body, []byte("Already building host for ha group"))
			if i > -1 {
				for true {
					_, httpResponse, err = r.p.client.DefaultApi.CreateHostInstaller(ctx).Execute()
					if err == nil {
						break
					}
				}
			} else {
				resp.Diagnostics.AddError(
					"Error creating host installer",
					"Could not create host installer: "+err.Error(),
				)
				return
			}
		}
	}

	// 3) download installer
	session, err := conn.NewSession()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ssh session",
			"Could not create ssh session: "+err.Error(),
		)
		return
	}
	defer session.Close()

	insecure := ""
	if r.p.data.Insecure.Value {
		insecure = "-k"
	}
	cmd := fmt.Sprintf(
		"sudo curl -s -o '/tmp/installer.sh' -H 'Authorization: %s' %s %s/host/download%s && "+
			"sudo chmod +x /tmp/installer.sh",
		apikey.Value, insecure, mainhost.Value, haGroup)
	err = session.Run(cmd)
	if err != nil {
		fmt.Println(cmd)
		resp.Diagnostics.AddError(
			"Error downloading installer",
			"Could not download installer: "+err.Error(),
		)
		return
	}

	// 4) Execute installer
	session, err = conn.NewSession()
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
					"Error getting host",
					"Could not get host: "+err.Error(),
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
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting HA group",
			"Could not get HA group: "+err.Error(),
		)
		return
	}

	// Map response body to resource schema attribute
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

	// Generate resource state struct
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
