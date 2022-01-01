package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

// forwardedEnvVars whose values are taken from this machine (running azure-cluster) and copied over to the
// provisioned instances
var forwardedEnvVars = []string{
	"AWS_REGION",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",

	"AZURE_SUBSCRIPTION_ID",
	"AZURE_SERVICE_PRINCIPAL_TENANT",
	"AZURE_SERVICE_PRINCIPAL_NAME",
	"AZURE_SERVICE_PRINCIPAL_PASSWORD",
	"AZURE_DEV_STORAGE_ACCESS_KEY",

	"KITE_AZUREUTIL_STORAGE_NAME",
	"KITE_AZUREUTIL_STORAGE_KEY",
}

var (
	auth       *adal.ServicePrincipalToken // used for azure ARM clients
	subID      string                      // azure subscription ID
	storageKey string                      // azure blob storage key
)

func initAzureCreds() {
	// get azure credentials from environment
	spName, spPass, spTenant :=
		envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_NAME"),
		envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_PASSWORD"),
		envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_TENANT")

	subID = envutil.MustGetenv("AZURE_SUBSCRIPTION_ID")
	storageKey = envutil.MustGetenv("AZURE_DEV_STORAGE_ACCESS_KEY")

	// authenticate to get service principal token
	oauth, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, spTenant)
	if err != nil {
		log.Fatalln(err)
	}

	auth, err = adal.NewServicePrincipalToken(*oauth, spName, spPass, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		log.Fatalln(err)
	}
}

// envVarsToForward to the instance we want to provision, along with their values
func envVarsToForward() (map[string]string, error) {
	vars := make(map[string]string, len(forwardedEnvVars))

	for _, v := range forwardedEnvVars {
		val := os.Getenv(v)
		if val == "" {
			return nil, fmt.Errorf("missing env var: %s", v)
		}
		vars[v] = val
	}

	return vars, nil
}
