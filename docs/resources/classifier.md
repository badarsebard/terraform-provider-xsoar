---
page_title: "xsoar_classifier Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_classifier resource in the Terraform provider XSOAR.
---

# Resource xsoar_classifier

Classifier resource in the Terraform provider XSOAR.

## Example Usage
```terraform
resource "xsoar_classifier" "example" {
  name = "foo"
}

resource "xsoar_classifier" "example2" {
  name    = "bar"
  account = "StarkIndustries"
}
```

## Argument Reference
- **name** (Required) Name of the resource
- **id** (Optional) The ID of this resource.
- **account** (Optional) The account name of the XSOAR tenant (do not include the `acc_` prefix).
- **propagation_labels** (Optional) A list of propagation labels to add to the classifier
- **default_incident_type** (Optional) classification type for incidents that do not match any others in key_type_map.
- **key_type_map** (Optional) A mapping between a key of the incident data and the incident type. This must be formatted as a JSON string.
- **transformer** (Optional) The transformations to be applied to the incident data to generate the keys used in `key_type_map`. This must be formatted as a JSON string.

<!-- ## Attributes Reference -->

<!-- ## Timeouts -->

## Import
Classifiers can be imported using the resource `name`, e.g.,
```shell
terraform import xsoar_classifier.example foo
```
Classifiers that are account-specific require the `account` to be prefixed to the `name` with a period (`.`), e.g.,
```shell
terraform import xsoar_classifier.example2 StarkIndustries.bar
```