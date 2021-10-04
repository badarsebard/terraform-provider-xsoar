---
page_title: "ha_group Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
ha_group resource in the Terraform provider XSOAR.
---

# Resource `ha_group`

ha_group resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "ha_group" "example" {
  ha_group_name = "foo"
  elastic_cluster_url = "http://elastic.cluster.local:9200"
  elastic_index_prefix = "foo_"
}
```

## Schema
- **ha_group_name** (string) Name of the HA group
- **elastic_cluster_url** (string:url) URL location of Elasticsearch cluster, including scheme and port
- **elastic_index_prefix** (string) string prefix for ha group indexes, cannot be empty 

### Optional
