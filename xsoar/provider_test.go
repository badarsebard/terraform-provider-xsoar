package xsoar

import (
	"crypto/tls"
	"github.com/badarsebard/xsoar-sdk-go/openapi"
	"net/http"
	"os"
)

var openapiClient *openapi.APIClient

func init() {
	apikey := os.Getenv("DEMISTO_API_KEY")
	mainhost := os.Getenv("DEMISTO_BASE_URL")
	openapiConfig := openapi.NewConfiguration()
	openapiConfig.Servers[0].URL = mainhost
	openapiConfig.AddDefaultHeader("Authorization", apikey)
	openapiConfig.AddDefaultHeader("Accept", "application/json,*/*")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	openapiConfig.HTTPClient = client
	openapiClient = openapi.NewAPIClient(openapiConfig)
}
