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

func TestAccMapperDataSource_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccMapperDataSourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckMapperDataSourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccMapperDataSourceBasic(rName),
				Check:  resource.TestCheckResourceAttrPair("data.xsoar_mapper."+rName, "id", "xsoar_mapper."+rName, "id"),
			},
		},
	})
}

func testAccMapperDataSourcePreCheck(t *testing.T) {}

func testAccCheckMapperDataSourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_mapper."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		_, _, err := openapiClient.DefaultApi.GetClassifier(context.Background()).SetIdentifier(r).Execute()
		if err == nil {
			return fmt.Errorf("mapper returned when it should be destroyed")
		}
		return nil
	}
}

func testAccMapperDataSourceBasic(name string) string {
	c := `
resource "xsoar_mapper" "{name}" {
  name = "{name}"
}

data "xsoar_mapper" "{name}" {
  name = xsoar_mapper.{name}.name
}
`
	c = strings.Replace(c, "{name}", name, -1)
	return c
}
