---
page_title: "xsoar_host Data Source - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_host data source in the Terraform provider XSOAR.
---

# Data Source xsoar_host

Host data source in the Terraform provider XSOAR.

## Example Usage
```terraform
data "xsoar_host" "example" {
  name = "foo"
}
```

## Argument Reference
- **name** (Required) The name of the host.
- **server_url** (Optional) Setting this argument will place the value into the state file.
- **ssh_user** (Optional) Setting this argument will place the value into the state file.
- **ssh_key** (Optional) Setting this argument will place the value into the state file.

## Attributes Reference
- **id** The ID of the resource.
- **ha_group_name** The name of the HA group this host should join. Changing this will force a new resource.
- **elasticsearch_url** The URL with scheme and port of the elasticsearch cluster.
