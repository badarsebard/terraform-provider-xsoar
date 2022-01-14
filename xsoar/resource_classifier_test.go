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

func TestAccClassifier_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccClassifierResourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckClassifierResourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccClassifierResourceBasic(rName),
				Check:  testAccCheckClassifierResourceExists(rName),
			},
			{
				ResourceName:      "xsoar_classifier." + rName,
				ImportStateId:     rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccClassifierResourcePreCheck(t *testing.T) {}

func testAccCheckClassifierResourceExists(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_classifier."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		resp, _, err := openapiClient.DefaultApi.GetClassifier(context.Background()).SetIdentifier(rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("Error getting Classifier: " + err.Error())
		}
		if rsid := *resp.Id; rsid != rs.Primary.ID {
			return fmt.Errorf("Classifier ID created (" + rsid + ") did not match state (" + rs.Primary.ID + ")")
		}
		return nil
	}
}

func testAccCheckClassifierResourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_classifier."+r]
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
		return fmt.Errorf("found classifier when none was expected: " + err.Error())
	}
}

func testAccClassifierResourceBasic(name string) string {
	c := `
resource "xsoar_classifier" "{name}" {
  name = "{name}"
}`
	c = strings.Replace(c, "{name}", name, -1)
	return c
}
