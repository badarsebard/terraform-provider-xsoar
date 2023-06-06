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

func TestAccClassifierDataSource_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccClassifierDataSourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckClassifierDataSourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccClassifierDataSourceBasic(rName),
				Check:  resource.TestCheckResourceAttrPair("data.xsoar_classifier."+rName, "id", "xsoar_classifier."+rName, "id"),
			},
		},
	})
}

func testAccClassifierDataSourcePreCheck(t *testing.T) {}

func testAccCheckClassifierDataSourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_classifier."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		_, _, err := openapiClient.DefaultApi.GetClassifier(context.Background()).SetIdentifier(r).Execute()
		if err == nil {
			return fmt.Errorf("classifier returned when it should be destroyed")
		}
		return nil
	}
}

func testAccClassifierDataSourceBasic(name string) string {
	c := `
resource "xsoar_classifier" "{name}" {
  name = "{name}"
}

data "xsoar_classifier" "{name}" {
  name = xsoar_classifier.{name}.name
}
`
	c = strings.Replace(c, "{name}", name, -1)
	return c
}
