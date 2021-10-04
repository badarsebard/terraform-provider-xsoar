---
page_title: "account Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
account resource in the Terraform provider XSOAR.
---

# Resource `account`

account resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "account" "example" {
  name               = "foo"
  host_group_name    = "server-name"
  propagation_labels = ["a", "b"]
  roles              = ["Administrator"]
}
```

## Schema

### Optional

- **id** (String, Optional) The ID of this resource.
- **sample_attribute** (String, Optional) Sample attribute.

