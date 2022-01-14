---
page_title: "xsoar_integration_instance Data Source - terraform-provider-xsoar"
subcategory: ""
description: |-
xsoar_integration_instance data source in the Terraform provider XSOAR.
---

# Data Source xsoar_integration_instance

Integration instance data source in the Terraform provider XSOAR.

## Example Usage
```terraform
data "xsoar_integration_instance" "example" {
  name = "foo"
}
```

## Argument Reference
- **name** (Required) The name of the integration_instance.

## Attributes Reference
- **id** The ID of the resource.
- **integration_name** The name of the integration to be used. This represents the kind of integration to be configured, not the individual instance.
- **account** The name of the multi-tenant account for the instance of the integration.
- **propagation_labels** A list of strings to apply to the resource as propagation labels.