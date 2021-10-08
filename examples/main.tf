terraform {
  required_providers {
    xsoar = {
      version = "~> 0.1.0"
      source  = "local/badarsebard/xsoar"
    }
  }
}

provider "xsoar" {
  main_host = "https://your_main_host"
  api_key   = "your_api_key"
  insecure  = true
}

data "xsoar_account" "test" {
  name = "fml"
}

#resource "xsoar_integration_instance" "test" {
#  name               = "threatcentral_instance_1"
#  integration_name   = "threatcentral"
#  propagation_labels = ["all", "fml"]
#  account            = "fml"
#  config = {
#    APIAddress : "https://threatcentral.io/tc/rest/summaries"
#    APIKey : "123na"
#    useproxy : "true"
#  }
#}

#resource "xsoar_account" "test" {
#  name               = "acc1"
#  host_group_name    = xsoar_ha_group.test.name
#  account_roles      = ["Administrator"]
#  propagation_labels = [""]
#  depends_on         = [xsoar_host.test]
#}

#resource "xsoar_host" "test" {
#  name          = "host1.xsoar.local"
#  server_url    = "host1.xsoar.local:22"
#  ha_group_name = xsoar_ha_group.test.name
#  ssh_user      = "vagrant"
#  ssh_key_file  = "/path/to/file"
#}

#resource "xsoar_ha_group" "test" {
#  name                 = "ha_1"
#  elasticsearch_url    = "http://elastic.xsoar.local:9200"
#  elastic_index_prefix = "ha_1_"
#}