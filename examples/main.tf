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

#resource "xsoar_account" "test" {
#  name               = "test1"
#  host_group_name    = "ha_1"
#  account_roles      = ["Administrator"]
#  propagation_labels = ["ada"]
#}

#resource "xsoar_ha_group" "test" {
#  name                 = "test1"
#  elasticsearch_url    = "http://elastic.xsoar.local:9200"
#  elastic_index_prefix = "test1_"
#}

resource "xsoar_host" "test" {
  name          = "host1.xsoar.local"
#  ha_group_name = "test2"
  server_url    = "host1.xsoar.local:22"
  ssh_user      = "vagrant"
  ssh_key_file  = "/path/to/file"
}