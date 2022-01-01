//go:generate go-bindata -o bindata.go templates
package main

import (
	"log"
	"os"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/mitchellh/cli"
)

// procErr is used for error reporting on parallelized commands
type procErr struct {
	proc string
	err  error
}

var (
	auth  *adal.ServicePrincipalToken // used for azure ARM clients
	subID string                      // azure subscription ID
)

func init() {
	// get azure credentials from environment
	spName, spPass, spTenant :=
		envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_NAME"),
		envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_PASSWORD"),
		envutil.MustGetenv("AZURE_SERVICE_PRINCIPAL_TENANT")

	subID = envutil.MustGetenv("AZURE_SUBSCRIPTION_ID")

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

func main() {
	c := cli.NewCLI("azure-deployments", "0.0.1")
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"deploy": func() (cli.Command, error) {
			return newDeployCommand()
		},
		"deployregions": func() (cli.Command, error) {
			return &deployregionsCommand{}, nil
		},
		"terminate": func() (cli.Command, error) {
			return &terminateCommand{}, nil
		},
		"terminateregions": func() (cli.Command, error) {
			return &terminateregionsCommand{}, nil
		},
		"switch": func() (cli.Command, error) {
			return &switchCommand{}, nil
		},
		"switchregions": func() (cli.Command, error) {
			return &switchregionsCommand{}, nil
		},
		"describe": func() (cli.Command, error) {
			return &describeCommand{}, nil
		},
		"describeregions": func() (cli.Command, error) {
			return &describeregionsCommand{}, nil
		},
		"cleanup": func() (cli.Command, error) {
			return &cleanupCommand{}, nil
		},
		"cleanupregions": func() (cli.Command, error) {
			return &cleanupregionsCommand{}, nil
		},
		"list": func() (cli.Command, error) {
			return &listCommand{}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(exitStatus)
}
