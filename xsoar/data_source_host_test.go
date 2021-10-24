package xsoar

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"os"
	"strings"
	"testing"
)

func TestAccHostDataSource_basic(t *testing.T) {
	rName := acctest.RandStringFromCharSet(5, acctest.CharSetAlpha)
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccHostDataSourcePreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"xsoar": func() (tfprotov6.ProviderServer, error) {
				return tfsdk.NewProtocol6Server(New()), nil
			},
		},
		CheckDestroy: testAccCheckHostDataSourceDestroy(rName),
		Steps: []resource.TestStep{
			{
				Config: testAccHostDataSourceBasic(rName),
				Check:  resource.TestCheckResourceAttrPair("data.xsoar_host."+rName, "id", "xsoar_host."+rName, "id"),
			},
		},
	})
}

func testAccHostDataSourcePreCheck(t *testing.T) {}

func testAccCheckHostDataSourceExists(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_host."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		resp, _, err := openapiClient.DefaultApi.GetHost(context.Background(), rs.Primary.Attributes["name"]).Execute()
		if err != nil {
			return fmt.Errorf("Error getting Host: " + err.Error())
		}
		if rsid := resp["id"].(string); rsid != rs.Primary.ID {
			return fmt.Errorf("Host ID created (" + rsid + ") did not match state (" + rs.Primary.ID + ")")
		}
		return nil
	}
}

func testAccCheckHostDataSourceDestroy(r string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources["xsoar_host."+r]
		if !ok {
			return fmt.Errorf("not found: %s in %s", r, state.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		host, _, err := openapiClient.DefaultApi.GetHost(context.Background(), r).Execute()
		if err != nil {
			return fmt.Errorf("Error getting host: " + err.Error())
		}
		if host != nil {
			return fmt.Errorf("found host when none was expected")
		}
		return nil
	}
}

func testAccHostDataSourceBasic(name string) string {
	keyfile := os.Getenv("DEMISTO_HOST_KEYFILE")
	host := os.Getenv("DEMISTO_HOST")
	c := `
resource "xsoar_host" "{name}" {
  name          = "{host}"
  server_url    = "{host}:22"
  ssh_user      = "vagrant"
  ssh_key_file  = "{keyfile}"
}

data "xsoar_host" "{name}" {
  name = xsoar_host.{name}.name
}
`
	c = strings.Replace(c, "{name}", name, -1)
	c = strings.Replace(c, "{keyfile}", keyfile, -1)
	c = strings.Replace(c, "{host}", host, -1)
	return c
}
