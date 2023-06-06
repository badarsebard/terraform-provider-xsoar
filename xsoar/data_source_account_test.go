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

func TestAccAccountDataSource_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccAccountDataSourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckAccountDataSourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccAccountDataSourceBasic(rName),
				Check:  resource.TestCheckResourceAttrPair("data.xsoar_account."+rName, "id", "xsoar_account."+rName, "id"),
			},
		},
	})
}

func testAccAccountDataSourcePreCheck(t *testing.T) {}

func testAccCheckAccountDataSourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_account."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		resp, _, err := openapiClient.DefaultApi.GetAccount(context.Background(), "acc_"+r).Execute()
		if err != nil {
			return fmt.Errorf("Error getting account: " + err.Error())
		}
		if resp != nil {
			return fmt.Errorf("account returned when it should be destroyed")
		}
		return nil
	}
}

func testAccAccountDataSourceBasic(name string) string {
	c := `
resource "xsoar_account" "{name}" {
  name               = "{name}"
  host_group_name    = ""
}

data "xsoar_account" "{name}" {
  name = xsoar_account.{name}.name
}
`
	c = strings.Replace(c, "{name}", name, -1)
	return c
}
