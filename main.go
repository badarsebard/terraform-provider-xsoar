package main

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"terraform-provider-xsoar/xsoar"
)

func main() {
	_ = tfsdk.Serve(context.Background(), xsoar.New, tfsdk.ServeOpts{
		Name: "xsoar",
	})
}
