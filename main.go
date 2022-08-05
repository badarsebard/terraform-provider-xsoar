package main

import (
	"context"
	"log"
	"terraform-provider-xsoar/xsoar"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/badarsebard/xsoar",
	}

	err := providerserver.Serve(context.Background(), xsoar.New(), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
