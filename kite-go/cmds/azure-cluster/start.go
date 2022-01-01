package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	provisionScriptName = "provision.sh"
	resourceGroupPrefix = "cluster_"
	// all resource groups/VMSSes will be tagged with <clusterTag>=<clusterName>
	clusterTag = "cluster"
)

func startCluster(clusterName string, count int, instanceType string, use1804 bool) error {
	initAzureCreds()

	uris, err := uploadProvisionFiles(clusterName)
	if err != nil {
		return err
	}

	if err := createResourceGroup(clusterName); err != nil {
		return fmt.Errorf("error creating resource group: %v", err)
	}

	if err := createVMSS(clusterName, instanceType, count, uris, use1804); err != nil {
		// We don't want the resource group to stick around if the VMSS couldn't be created
		if err := stopCluster(clusterName); err != nil {
			log.Printf("encountered error in stopping cluster: %v", err)
		}
		return fmt.Errorf("error creating VMSS: %v", err)
	}

	return nil
}

func createVMSS(clusterName string, instanceType string, instanceCount int, fileURIs []string, use1804 bool) error {
	// init vmss client
	client := compute.NewVirtualMachineScaleSetsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	sku := "16.04-LTS"
	if use1804 {
		sku = "18.04-LTS"
	}

	vmss := compute.VirtualMachineScaleSet{
		Name: to.StringPtr(clusterName),
		Tags: map[string]*string{
			clusterTag: to.StringPtr(clusterName),
		},
		Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
		Location: &conf.Location,
		Sku: &compute.Sku{
			Name:     to.StringPtr(instanceType),
			Tier:     to.StringPtr(conf.SkuTier),
			Capacity: to.Int64Ptr(int64(instanceCount)),
		},
		VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
			UpgradePolicy:        &compute.UpgradePolicy{Mode: compute.Manual},
			Overprovision:        to.BoolPtr(false),
			SinglePlacementGroup: to.BoolPtr(true),
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
				OsProfile: &compute.VirtualMachineScaleSetOSProfile{
					ComputerNamePrefix: to.StringPtr(clusterName),
					AdminUsername:      &conf.VMUser,
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(true),
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								compute.SSHPublicKey{
									Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", conf.VMUser)),
									KeyData: &conf.SSHKey,
								},
							},
						},
					},
				},
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr("Canonical"),
						Offer:     to.StringPtr("UbuntuServer"),
						Sku:       to.StringPtr(sku),
						Version:   to.StringPtr("latest"),
					},
					OsDisk: &compute.VirtualMachineScaleSetOSDisk{
						Name:         to.StringPtr(fmt.Sprintf("%s_os_disk", clusterName)),
						Caching:      compute.CachingTypesReadWrite,
						CreateOption: compute.DiskCreateOptionTypesFromImage,
						VhdContainers: &[]string{
							conf.VHDContainer,
						},
					},
				},
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
						compute.VirtualMachineScaleSetNetworkConfiguration{
							Name: to.StringPtr(fmt.Sprintf("%s-nic", clusterName)),
							VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
								Primary: to.BoolPtr(true),
								IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
									compute.VirtualMachineScaleSetIPConfiguration{
										Name: to.StringPtr(fmt.Sprintf("%s-ipc", clusterName)),
										VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
											Subnet: &compute.APIEntityReference{
												ID: &conf.SubnetID,
											},
										},
									},
								},
							},
						},
					},
				},
				ExtensionProfile: &compute.VirtualMachineScaleSetExtensionProfile{
					Extensions: &[]compute.VirtualMachineScaleSetExtension{
						compute.VirtualMachineScaleSetExtension{
							Name: to.StringPtr("customScript"),
							VirtualMachineScaleSetExtensionProperties: &compute.VirtualMachineScaleSetExtensionProperties{
								Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
								Type:                    to.StringPtr("CustomScript"),
								TypeHandlerVersion:      to.StringPtr("2.0"),
								AutoUpgradeMinorVersion: to.BoolPtr(true),
								Settings: &map[string]interface{}{
									"fileUris": fileURIs,
								},
								ProtectedSettings: &map[string]interface{}{
									"commandToExecute": fmt.Sprintf(
										"sudo bash %s > /var/log/provision.log 2>&1",
										prefixedFilename(clusterName, provisionScriptName)),
									"storageAccountKey":  storageKey,
									"storageAccountName": conf.StorageName,
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := client.CreateOrUpdate(context.Background(), resourceGroupName(clusterName), clusterName, vmss)
	_ = resp
	return err
}

func createResourceGroup(clusterName string) error {
	// init client
	client := resources.NewGroupsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	rgName := resourceGroupName(clusterName)

	_, err := client.Get(context.Background(), rgName)
	if err == nil {
		return fmt.Errorf("resource group %s already exists", rgName)
	}

	// create new resource group
	rg := resources.Group{
		Name:     &rgName,
		Location: &conf.Location,
		Tags: map[string]*string{
			clusterTag: to.StringPtr(clusterName),
		},
	}

	// submit changes
	_, err = client.CreateOrUpdate(context.Background(), rgName, rg)
	if err != nil {
		return err
	}

	return nil
}

// uploadProvisionFiles uploads the files necessary for provisioning to blob storage and returns the resulting URIs
func uploadProvisionFiles(clusterName string) ([]string, error) {
	provisionScript, err := renderProvisionScript()
	if err != nil {
		return nil, err
	}

	sc, err := storage.NewBasicClient(conf.StorageName, storageKey)
	if err != nil {
		return nil, err
	}
	blobCli := sc.GetBlobService()
	ctn := blobCli.GetContainerReference(conf.StorageContainer)

	toUpload := map[string][]byte{
		provisionScriptName: provisionScript,
	}

	uris := make([]string, 0, len(toUpload))

	for filename, data := range toUpload {
		prefixed := prefixedFilename(clusterName, filename)
		err = uploadToBlobStorage(ctn, prefixed, data)
		if err != nil {
			return nil, err
		}
		uris = append(uris, fmt.Sprintf("%s/%s", conf.ProvisionDataURIBase, prefixed))
	}

	return uris, nil
}

func renderProvisionScript() ([]byte, error) {
	envVars, err := envVarsToForward()
	if err != nil {
		return nil, err
	}

	envVars["KITE_USE_AZURE_MIRROR"] = "1"

	var buf bytes.Buffer
	if err := templates.RenderText(&buf, "provision.sh", map[string]interface{}{
		"EnvVars": envVars,
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// prefixedFilename returns the filename prefixed by the cluster name so that the same filenames between different
// clusters don't collide in blob storage
func prefixedFilename(clusterName, filename string) string {
	return fmt.Sprintf("%s_%s", clusterName, filename)
}

func resourceGroupName(clusterName string) string {
	return resourceGroupPrefix + clusterName
}
