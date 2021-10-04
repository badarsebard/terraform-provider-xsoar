---
page_title: "integration_instance Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
integration_instance resource in the Terraform provider XSOAR.
---

# Resource `integration_instance`

integration_instance resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "integration_instance" "example" {
  sample_attribute = "foo"
}
```

## Schema

### Optional

- **id** (String, Optional) The ID of this resource.
- **sample_attribute** (String, Optional) Sample attribute.

