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

func TestAccAccount_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccAccountResourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckAccountResourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccAccountResourceBasic(rName),
				Check:  testAccCheckAccountResourceExists(rName),
			},
			{
				ResourceName:      "xsoar_account." + rName,
				ImportStateId:     rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAccountResourcePreCheck(t *testing.T) {}

func testAccCheckAccountResourceExists(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_account."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		resp, _, err := openapiClient.DefaultApi.GetAccount(context.Background(), "acc_"+r).Execute()
		if err != nil {
			return fmt.Errorf("Error getting account: " + err.Error())
		}
		if resp == nil {
			return fmt.Errorf("no account returned")
		}
		if rsid := resp["id"].(string); rsid != rs.Primary.ID {
			return fmt.Errorf("Account ID created (" + rsid + ") did not match state (" + rs.Primary.ID + ")")
		}
		return nil
	}
}

func testAccCheckAccountResourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_account."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
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

func testAccAccountResourceBasic(name string) string {
	c := `
resource "xsoar_account" "{name}" {
  name               = "{name}"
  host_group_name    = ""
}`
	c = strings.Replace(c, "{name}", name, -1)
	return c
}
