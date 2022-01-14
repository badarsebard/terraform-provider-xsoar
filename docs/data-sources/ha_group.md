---
page_title: "xsoar_ha_group Data Source - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_ha_group data source in the Terraform provider XSOAR.
---

# Data Source xsoar_ha_group

ha_group data source in the Terraform provider XSOAR.

## Example Usage
```terraform
data "xsoar_ha_group" "example" {
  name = "foo"
}
```

## Argument Reference
- **name** (Required) The name of the ha_group.

## Attributes Reference
- **id** The ID of the resource.
- **elastic_cluster_url** URL location of Elasticsearch cluster, including scheme and port.
- **elastic_index_prefix** String prefix for HA Group indexes.
