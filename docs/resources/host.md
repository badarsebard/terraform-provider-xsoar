---
page_title: "host Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
host resource in the Terraform provider XSOAR.
---

# Resource `host`

host resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "host" "example" {}

resource "host" "ha_example" {
  ha_group_name = "foo"
}

resource "ha_group" "ha_group_example" {
  ha_group_name = "foo"
  elastic_cluster_url = "http://elastic.cluster.local:9200"
  elastic_index_prefix = "foo_"
}
```

## Schema

### Optional
- **ha_group_name** (String, Optional) The name of the HA group this host belongs to.


