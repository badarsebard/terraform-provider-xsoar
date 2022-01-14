---
page_title: "xsoar_account Data Source - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_account data source in the Terraform provider XSOAR.
---

# Data Source xsoar_account

Account data source in the Terraform provider XSOAR.

## Example Usage
```terraform
data "xsoar_account" "example" {
  name = "foo"
}
```

## Argument Reference
- **name** (Required) The name of the account.

## Attributes Reference
- **id** The ID of the resource.
- **propagation_labels** List of propagation labels assigned to the account.
- **account_roles** List of user roles assigned to the account.
- **host_group_name** Name of the HA group to which this belongs