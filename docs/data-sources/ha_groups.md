---
page_title: "xsoar_ha_groups Data Source - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_ha_groups data source in the Terraform provider XSOAR.
---

# Data Source xsoar_ha_groups

A list of ha_group data source in the Terraform provider XSOAR.

## Example Usage
```terraform
data "xsoar_ha_groups" "example" {}
```

## Argument Reference
- **name** (Optional) HA groups whose names do not match the pattern will be excluded from the results.
- **max_accounts** (Optional) HA groups that have a number of accounts greater than `max_accounts` will be excluded from the results.

## Attributes Reference
- **groups** List of maps representing the HA groups
