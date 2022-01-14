---
page_title: "xsoar_ha_group Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
ha_group resource in the Terraform provider XSOAR.
---

# Resource xsoar_ha_group

HA Group resource in the Terraform provider XSOAR.

## Example Usage
```terraform
resource "xsoar_ha_group" "example" {
  ha_group_name        = "foo"
  elasticsearch_url    = "http://elastic.cluster.local:9200"
  elastic_index_prefix = "foo_"
}
```

## Argument Reference
- **ha_group_name** (Required) Name of the HA group
- **elastic_cluster_url** (Optional) URL location of Elasticsearch cluster, including scheme and port
- **elastic_index_prefix** (Optional) string prefix for HA Group indexes, cannot be empty

## Attributes Reference
- **id** The ID of the resource

<!-- ## Timeouts -->

## Import
HA Groups can be imported using the resource `ha_group_name`, e.g.,
```shell
terraform import xsoar_ha_group.example foo
```