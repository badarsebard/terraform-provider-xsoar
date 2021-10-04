package xsoar

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Account -
type Account struct {
	AccountRoles      []string     `tfsdk:"account_roles"`
	HostGroupName     types.String `tfsdk:"host_group_name"`
	HostGroupId       types.String `tfsdk:"host_group_id"`
	Name              types.String `tfsdk:"name"`
	PropagationLabels []string     `tfsdk:"propagation_labels"`
}

// HAGroup -
type HAGroup struct {
	Name               types.String `tfsdk:"name"`
	Id                 types.String `tfsdk:"id"`
	ElasticsearchUrl   types.String `tfsdk:"elasticsearch_url"`
	ElasticIndexPrefix types.String `tfsdk:"elastic_index_prefix"`
}

// Host -
type Host struct {
	Name             types.String `tfsdk:"name"`
	Id               types.String `tfsdk:"id"`
	HAGroupName      types.String `tfsdk:"ha_group_name"`
	ElasticsearchUrl types.String `tfsdk:"elasticsearch_url"`
	ServerUrl        types.String `tfsdk:"server_url"`
	SSHUser          types.String `tfsdk:"ssh_user"`
	SSHKeyFile       types.String `tfsdk:"ssh_key_file"`
}
