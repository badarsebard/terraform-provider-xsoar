---
page_title: "xsoar_mapper Data Source - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_mapper data source in the Terraform provider XSOAR.
---

# Data Source xsoar_mapper

Mapper data source in the Terraform provider XSOAR.

## Example Usage
```terraform
data "xsoar_mapper" "example" {
  name = "foo"
}
```

## Argument Reference
- **name** (Required) The name of the mapper.

## Attributes Reference
- **id** The ID of the resource.
- **direction** The direction of the mapper. It must be either `incoming` or `outgoing`.
- **mapping** A JSON string representing a mapping between fields.
- **account** The account name of the XSOAR tenant (do not include the `acc_` prefix).
- **propagation_labels** A list of strings to be used as propagation labels for the classifier.