package xsoar

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

// IntegrationInstance -
type IntegrationInstance struct {
	Name              types.String `tfsdk:"name"`
	Id                types.String `tfsdk:"id"`
	IntegrationName   types.String `tfsdk:"integration_name"`
	Config            types.Map    `tfsdk:"config"`
	PropagationLabels []string     `tfsdk:"propagation_labels"`
	Account           types.String `tfsdk:"account"`
}

type propLabel struct {
	Value string
}

func (p propLabel) Type(ctx context.Context) attr.Type {
	return types.StringType
}

func (p propLabel) ToTerraformValue(ctx context.Context) (interface{}, error) {
	return p.Value, nil
}

func (p propLabel) Equal(value attr.Value) bool {
	ctx := context.Background()
	pVal, _ := p.ToTerraformValue(ctx)
	vVal, _ := value.ToTerraformValue(ctx)
	if pVal == vVal {
		return true
	} else {
		return false
	}
}
