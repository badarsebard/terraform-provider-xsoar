---
page_title: "xsoar_integration_instance Resource - terraform-provider-xsoar"
subcategory: ""
description: |-
integration_instance resource in the Terraform provider XSOAR.
---

# Resource xsoar_integration_instance

Integration instance resource in the Terraform provider XSOAR.

## Example Usage

```terraform
resource "xsoar_integration_instance" "example" {
  name               = "foo"
  integration_name   = "threatcentral"
  propagation_labels = ["all"]
  incoming_mapper_id = "c0a4bb6d-4799-4818-8cc2-9cc343ad8ad7a"
  config = {
    APIAddress : "https://threatcentral.io/tc/rest/summaries"
    APIKey : "123"
    useproxy : "true"
  }
}

resource "xsoar_integration_instance" "example" {
  name               = "bar"
  integration_name   = "AWS - SQS"
  propagation_labels = ["all"]
  account            = "StarkIndustries"
  config = {
    mappingId = "ed76f483-5df2-442b-8b19-bc9f189b7a18"
    defaultRegion = "us-west-2"
    queueUrl = "https://sqs.us-west-2.amazonaws.com/091411462528/wt-threat-12345678-int-incident_queue"
    isFetch = true
  }
}
```

## Argument Reference
- **name** (Required) The name of the integration instance.
- **integration_name** (Required) The name of the integration to be used. This represents the kind of integration to be configured, not the individual instance.
- **config** (Required) A map of keys and values that configure the integration. The keys and their accepted values are dependent on the integration itself.
- **account** (Optional) The name of the multi-tenant account for the instance of the integration.
- **propagation_labels** (Optional) A list of strings to apply to the resource as propagation labels.
- **incoming_mapper_id** (Optional) The ID of the incoming mapper to use for the integration.

## Attributes Reference
- **id** The ID of this resource.

<!-- ## Timeouts -->

## Import
Integration instances can be imported using the resource `name`, e.g.,
```shell
terraform import xsoar_integration_instance.example foo
```
Integration instances that are account-specific require the `account` to be prefixed to the `name` with a period (`.`), e.g.,
```shell
terraform import xsoar_integration_instance.example2 StarkIndustries.bar
```