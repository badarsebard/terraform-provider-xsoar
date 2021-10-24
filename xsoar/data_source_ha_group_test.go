package xsoar

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"strings"
	"testing"
)

func TestAccHAGroupDataSource_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccHAGroupDataSourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckHAGroupDataSourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccHAGroupDataSourceBasic(rName),
				Check:  resource.TestCheckResourceAttrPair("data.xsoar_ha_group."+rName, "id", "xsoar_ha_group."+rName, "id"),
			},
		},
	})
}

func testAccHAGroupDataSourcePreCheck(t *testing.T) {}

func testAccCheckHAGroupDataSourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_ha_group."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, _, err := openapiClient.DefaultApi.GetHAGroup(context.Background(), r).Execute()
		if err != nil {
			return nil
		}
		return fmt.Errorf("found HA group when none was expected: " + err.Error())
	}
}

func testAccHAGroupDataSourceBasic(name string) string {
	c := `
resource "xsoar_ha_group" "{name}" {
  name                 = "{name}"
  elasticsearch_url    = "http://elastic.xsoar.local:9200"
  elastic_index_prefix = "{name}_"
}

data "xsoar_ha_group" "{name}" {
  name = xsoar_ha_group.{name}.name
}
`
	c = strings.Replace(c, "{name}", name, -1)
	return c
}
