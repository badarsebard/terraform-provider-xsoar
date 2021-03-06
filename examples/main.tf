terraform {
  required_providers {
    xsoar = {
      version = "~> 0.1"
      source  = "badarsebard/xsoar"
    }
  }
}

provider "xsoar" {
  main_host = "https://your.main.hostname.fqdn"
  api_key   = "your_api_key"
  insecure  = true
}

resource "xsoar_ha_group" "ha1" {
  name                 = "ha_1"
  elastic_index_prefix = "ha_1_"
  elasticsearch_url    = "https://elastic.example.com:9200"
}

resource "xsoar_host" "host1" {
  name          = "host.example.com"
  ha_group_name = xsoar_ha_group.ha1.name
  server_url    = "host.example.com:22"
  ssh_user      = "vagrant"
  ssh_key       = file("/path/to/file")
}

resource "xsoar_account" "acc1" {
  name               = "acc1"
  host_group_name    = xsoar_ha_group.ha1.name
  account_roles      = ["Administrator"]
  propagation_labels = [""]
  depends_on         = [xsoar_host.host1]
}

resource "xsoar_integration_instance" "threatcentral1" {
  name               = "threatcentral_instance_1"
  integration_name   = "threatcentral"
  propagation_labels = ["all"]
  account            = xsoar_account.acc1.name
  config = {
    APIAddress : "https://threatcentral.io/tc/rest/summaries"
    APIKey : "123"
    useproxy : "true"
  }
}