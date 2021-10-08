---
page_title: "account Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
account resource in the Terraform provider XSOAR.
---

# Resource `account`

Account resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "account" "example" {
  name               = "foo"
  host_group_name    = "server-name"
  propagation_labels = ["a", "b"]
  account_roles      = ["Administrator"]
}
```

## Schema
- **name** (String) Name of the account
- **propagation_labels** (List) List of propagation labels applied to the account
- **account_roles** (List) List of user roles applied to the account

### Optional
- **host_group_name** (String) Name of the HA group to which this belongs
