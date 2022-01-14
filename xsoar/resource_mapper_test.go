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

func TestAccMapper_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccMapperResourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckMapperResourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccMapperResourceBasic(rName),
				Check:  testAccCheckMapperResourceExists(rName),
			},
			{
				ResourceName:      "xsoar_mapper." + rName,
				ImportStateId:     rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccMapperResourcePreCheck(t *testing.T) {}

func testAccCheckMapperResourceExists(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_mapper."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		resp, _, err := openapiClient.DefaultApi.GetClassifier(context.Background()).SetIdentifier(rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("Error getting Mapper: " + err.Error())
		}
		if rsid := *resp.Id; rsid != rs.Primary.ID {
			return fmt.Errorf("Mapper ID created (" + rsid + ") did not match state (" + rs.Primary.ID + ")")
		}
		return nil
	}
}

func testAccCheckMapperResourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_mapper."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, _, err := openapiClient.DefaultApi.GetClassifier(context.Background()).SetIdentifier(r).Execute()
		if err != nil {
			return nil
		}
		return fmt.Errorf("found mapper when none was expected: " + err.Error())
	}
}

func testAccMapperResourceBasic(name string) string {
	c := `
resource "xsoar_mapper" "{name}" {
  name      = "{name}"
  direction = "incoming"
}`
	c = strings.Replace(c, "{name}", name, -1)
	return c
}
