---
page_title: "xsoar_mapper Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_mapper resource in the Terraform provider XSOAR.
---

# Resource xsoar_mapper

Mapper resource in the Terraform provider XSOAR.

## Example Usage
```terraform
resource "xsoar_mapper" "example" {
  name = "foo"
}
```

## Argument Reference
- **name** (Required) Name of the resource
- **direction** (Required) The direction of the mapper. It must be either `incoming` or `outgoing`.
- **mapping** (Optional) A JSON string representing a mapping between fields.
- **id** (Optional) The ID of this resource.
- **account** (Optional) The account name of the XSOAR tenant (do not include the `acc_` prefix).
- **propagation_labels** (Optional) A list of strings to be used as propagation labels for the classifier.

<!-- ## Attributes Reference -->

<!-- ## Timeouts -->

## Import
Mappers can be imported using the resource `name`, e.g.,
```shell
terraform import xsoar_mapper.example foo
```