---
page_title: "ha_group Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
ha_group resource in the Terraform provider XSOAR.
---

# Resource `ha_group`

HA group resource in the Terraform provider XSOAR.

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

### Optional
- **elastic_cluster_url** (String, Optional) URL location of Elasticsearch cluster, including scheme and port
- **elastic_index_prefix** (String, Optional) string prefix for ha group indexes, cannot be empty 
