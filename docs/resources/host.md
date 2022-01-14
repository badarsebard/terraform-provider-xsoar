---
page_title: "xsoar_host Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
host resource in the Terraform provider XSOAR.
---

# Resource xsoar_host

Host resource in the Terraform provider XSOAR. Hosts in XSOAR are the individual servers that join a multi-tenant architecture. They consist of two components: the physical infrastructure and the XSOAR host application. The host application is obtained through the Main Account. See the Palo Alto documentation for more information on how to obtain the installer manually. The Terraform provider for XSOAR manages the creation, download, installation, and uninstallation of the host application on to an existing server via an SSH connection. The sequence of events is roughly this:
1. The provider initiates the build of the host installer via the API
2. The provider connects to the host server via SSH
3. The host server downloads the installer via the API
4. The host server executes the installer to either install on create, or uninstall on destroy
5. The provider waits for the host to appear or disappear from the API and updates the Terraform state file 

## Example Usage

```terraform
resource "xsoar_host" "example" {
  name = "foo"
  server_url = "foo.example.com:22"
  ssh_user = "sshuser"
  ssh_key = file("/home/sshuser/.ssh/id_rsa")
}

resource "xsoar_host" "es_example" {
  name = "foo"
  elasticsearch_url = "http://elastic.example.com:9200"
  server_url = "foo.example.com:22"
  ssh_user = "sshuser"
  ssh_key = file("/home/sshuser/.ssh/id_rsa")
}

resource "xsoar_host" "ha_example" {
  name = "foo"
  ha_group_name = "bar"
  server_url = "foo.example.com:22"
  ssh_user = "sshuser"
  ssh_key = file("/home/sshuser/.ssh/id_rsa")
}
```

## Argument Reference
- **name** (Required) Name of the host, will be used as XSOAR "external address". Usually the hostname of the underlying server. Changing this will force a new resource.
- **server_url** (Required) FQDN or IP and the SSH port of the host.
- **ssh_user** (Required) Username for the SSH connection.
- **ssh_key** (Required) SSH private key content.
- **ha_group_name** (Optional) The name of the HA group this host should join. Changing this will force a new resource.
- **elasticsearch_url** (Optional) The URL with scheme and port of the elasticsearch cluster. Not needed if using `ha_group_name`. Changing this will force a new resource.

## Attributes Reference
- **id** The ID of the resource

<!-- ## Timeouts -->

## Import
Hosts can be imported using the resource `name`, e.g.,
```shell
terraform import xsoar_host.example foo
```
If a host is imported it will not capture the `server_url`, `ssh_user`, and `ssh_key` attributes as these are not contained within the API. The next time Terraform is run they will be shown in the plan and added to the state. All other attributes force a re-creation of the resource.