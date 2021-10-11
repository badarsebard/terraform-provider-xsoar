# Terraform XSOAR Provider
_**This is an early Alpha release and not recommended for production usage**_

The Terraform XSOAR provider is a plugin for Terraform that allows for the management and configuration of Palo Alto's Cortex XSOAR platform. This provider can be used to manage single server and multi-tenant architectures, hosts with elastic backends, and high availability groups. 


## Example Single Tenant Usage
The provider relies on an existing XSOAR host and API key. There is only one resource of concern when it comes to a single tenant deployment and that is an instance of an integration.

```terraform
provider "xsoar" {
  main_host = "https://your_main_host"
  # you should use override.tf to keep this value out of version control
  api_key   = "your_api_key"
  insecure  = true
}

resource "xsoar_integration_instance" "threatcentral_1" {
  name               = "threatcentral_instance_1"
  integration_name   = "threatcentral"
  config = {
    APIAddress : "https://threatcentral.io/tc/rest/summaries"
    APIKey : "your_tc_api_key"
    useproxy : "false"
  }
}
```
The instance integration attribute `config` is a map of all integration settings as `key : value` pairs.

## Example Multi-Tenant Usage
Like a single-tenant deployment, multi-tenant requires an existing XSOAR host and API key. This must be the Main host. There three additional resource types of interest to a multi-tenant deployment: HA Group, Host, and Account.
```terraform
provider "xsoar" {
  main_host = "https://your_main_host"
  # you should use override.tf to keep this value out of version control
  api_key   = "your_api_key"
  insecure  = true
}

resource "xsoar_ha_group" "ha1" {
  name                 = "ha_1"
  elasticsearch_url    = "http://elastic.xsoar.local:9200"
  elastic_index_prefix = "ha_1_"
}

resource "xsoar_host" "host1" {
  name          = "host1.xsoar.local"
  ha_group_name = xsoar_ha_group.ha1.name
  server_url    = "host1.xsoar.local:22"
  ssh_user      = "vagrant"
  ssh_key_file  = "/path/to/file"
}

resource "xsoar_account" "acc1" {
  name               = "acc1"
  host_group_name    = xsoar_ha_group.ha1.name
  account_roles      = ["Administrator"]
  propagation_labels = [""]
  depends_on         = [xsoar_host.host1]
}
```
The `xsoar_ha_group` is an XSOAR construct representing a group of host servers sharing a common configuration and elastic backend. 

Each `xsoar_host` resource represents an installation of the XSOAR host installer on a server. The actual server must exist prior to deploying the resource and SSH configuration must be supplied via the `server_url`, `ssh_user`, and `ssh_key_file` attributes. The Terraform plugin will download the correct host installer from the Main host, transfer the installer via SSH, and execute the installation of XSOAR on the host. Once the installation is complete the host automatically joins the multi-tenant deployment and can be seen from the Main host. Hosts can belong to an HA Group, or they can be standalone instances. Standalone hosts can be configured to use either elastic or boltdb depending on whether the `elasticsearch_url` attribute is present.

Each `xsoar_account` represents an individual tenant within the multi-tenant deployment. Each account must be assigned to an HA group or a host using the `host_group_name` attribute. Account roles such as `Administrator` and `Analyst` must be assigned as well as, optionally, propagation labels. In addition, the use of the `depends_on` meta-argument is strongly recommended, to ensure Terraform does not attempt to create an account within a host or HA group that doesn't yet exist.