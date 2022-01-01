package main

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
)

func stopCluster(clusterName string) error {
	initAzureCreds()

	client := resources.NewGroupsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	rgName := resourceGroupName(clusterName)

	_, err := client.Delete(context.Background(), rgName)
	return err
}
