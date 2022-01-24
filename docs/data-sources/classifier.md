---
page_title: "xsoar_classifier Data Source - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_classifier data source in the Terraform provider XSOAR.
---

# Data Source xsoar_classifier

Classifier data source in the Terraform provider XSOAR.

## Example Usage
```terraform
data "xsoar_classifier" "example" {
  name = "foo"
}
```

## Argument Reference
- **name** (Required) The name of the classifier.

## Attributes Reference
- **id** The ID of this resource.
- **account** The account name of the XSOAR tenant.
- **propagation_labels** A list of propagation labels for the classifier
- **default_incident_type** Classification type for incidents that do not match any others in key_type_map.
- **key_type_map** A mapping between a key of the incident data and the incident type.
- **transformer** The transformations to be applied to the incident data to generate the keys used in `key_type_map`.