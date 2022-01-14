---
page_title: "xsoar_account Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_account resource in the Terraform provider XSOAR.
---

# Resource: xsoar_account

Account resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "xsoar_account" "example" {
  name               = "foo"
  host_group_name    = "server-name"
  propagation_labels = ["a", "b"]
  account_roles      = ["Administrator"]
}
```

## Argument Reference
The following arguments are supported:
- **name** (Required) Name of the account
- **propagation_labels** (Optional) List of propagation labels applied to the account
- **account_roles** (Optional) List of user roles applied to the account
- **host_group_name** (Optional) Name of the HA group to which this belongs

## Attributes Reference
The following attributes are exported:
- **id** The ID of the resource

<!--## Timeouts-->

## Import
Accounts can be imported using the resource `name`, e.g.,
```shell
terraform import xsoar_account.example foo
```