---
page_title: "host Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
host resource in the Terraform provider XSOAR.
---

# Resource `host`

Host resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "xsoar_host" "example" {
  name = "foobar"
  server_url = "hostname.bar:22"
  ssh_user = "sshuser"
  ssh_key = file("/home/sshuser/.ssh/id_rsa")
}

resource "xsoar_host" "es_example" {
  name = "foobar"
  elasticsearch_url = "http://elastic.foo:9200"
  server_url = "hostname.bar:22"
  ssh_user = "sshuser"
  ssh_key = file("/home/sshuser/.ssh/id_rsa")
}

resource "xsoar_host" "ha_example" {
  name = "hostname"
  ha_group_name = "foo"
  server_url = "hostname.foo:22"
  ssh_user = "sshuser"
  ssh_key = file("/home/sshuser/.ssh/id_rsa")
}

resource "ha_group" "ha_group_example" {
  ha_group_name = "foo"
  elastic_cluster_url = "http://elastic.cluster.local:9200"
  elastic_index_prefix = "foo_"
}
```

## Schema
- **name** (String) Name of the host, will be used as XSOAR "external address"
- **server_url** (String) URL and port of the host for an SSH connection
- **ssh_user** (String) Username for the SSH connection
- **ssh_key** (String) SSH private key content

### Optional
- **ha_group_name** (String, Optional) The name of the HA group this host belongs to.
- **elasticsearch_url** (String, Optional) The URL with scheme and port of the elasticsearch cluster

