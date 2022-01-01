package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/mitchellh/cli"
)

type deployCommand struct {
	templates *templateset.Set
}

func newDeployCommand() (*deployCommand, error) {
	// initialize templateset using bindata for the provisioning script
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(staticfs, "templates", nil)
	if err := templates.Validate(); err != nil {
		return nil, err
	}
	return &deployCommand{templates: templates}, nil
}

func (d *deployCommand) Help() string {
	return "run this command with a region and release, e.g 'deploy westus release_0'"
}

func (d *deployCommand) Synopsis() string {
	return "deploy release to a specified region"
}

func (d *deployCommand) Run(args []string) int {
	// init cli ui
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	// make sure all args are present
	if len(args) != 2 {
		return cli.RunResultHelp
	}

	// read command line args
	region, ok := Regions[args[0]]
	if !ok {
		ui.Error(fmt.Sprintf("region %s does not exist", args[0]))
	}
	release := args[1]

	// use region for logging prefix
	prefix := fmt.Sprintf("[%s] ", region.Location)
	ui = &cli.PrefixedUi{
		AskPrefix:       prefix,
		AskSecretPrefix: prefix,
		OutputPrefix:    prefix,
		InfoPrefix:      prefix,
		ErrorPrefix:     prefix,
		WarnPrefix:      prefix,
		Ui:              ui,
	}

	// get binary names from region scale sets
	var binaries []string
	for name := range region.ScaleSets {
		binaries = append(binaries, name)
	}
	// check that the release is a valid release
	isRelease, err := checkReleaseBinaries(release, binaries)
	if err != nil {
		ui.Error(fmt.Sprintf("error while checking for release binaries: %v", err))
		return 1
	}
	if !isRelease {
		ui.Error(fmt.Sprintf("%s is not a valid release", release))
		return 1
	}

	// create new resource group for scale sets using the release name and location
	rg, err := createReleaseRG(release, region.Location)
	if err != nil {
		ui.Error(fmt.Sprintf("%s", err))
		return 1
	}
	ui.Info(fmt.Sprintf("created new resource group %s", rg))

	// read in credentials
	creds := Credentials{
		AWSID:      envutil.MustGetenv("AWS_ACCESS_KEY_ID"),
		AWSKey:     envutil.MustGetenv("AWS_SECRET_ACCESS_KEY"),
		StorageKey: envutil.MustGetenv(strings.ToUpper(fmt.Sprintf("AZURE_%s_STORAGE_ACCESS_KEY", region.Location))),
	}

	// struct for provision script
	provTData := ProvisionScriptTemplateData{
		Region:        region,
		Credentials:   creds,
		Release:       release,
		ReleaseNoDots: strings.Replace(release, ".", "_", -1),
		ReleaseMD5:    fmt.Sprintf("%x", md5.Sum([]byte(release))),
		IsProduction:  strings.HasPrefix(release, "release_"),
	}

	// struct for new release scale set
	vmssData := VMSSData{
		Region:      region,
		Credentials: creds,
		Release:     release,
		ReleaseRG:   rg,
	}

	// create deployment scale sets in parallel
	resps := make(chan procErr, len(region.ScaleSets))
	wg := sync.WaitGroup{}
	wg.Add(len(region.ScaleSets))
	for process := range region.ScaleSets {
		ui.Info(fmt.Sprintf("creating new %s scale set for %s", process, release))
		go func(p string) {
			defer wg.Done()
			err := d.createVMSSForProcess(p, vmssData, provTData)

			resps <- procErr{
				proc: p,
				err:  err,
			}
		}(process)
	}

	// wait for create to finish
	go func() {
		wg.Wait()
		close(resps)
	}()

	// check errors
	var failed bool
	for resp := range resps {
		if resp.err != nil {
			ui.Error(fmt.Sprintf("failed to create %s:\n\t%v", resp.proc, resp.err))
			failed = true
		}
	}
	if failed {
		return 1
	}

	return 0
}

// checkReleaseBinaries checks S3 to make sure that the release name exists in the bucket and has
// the required binaries
func checkReleaseBinaries(release string, binaries []string) (bool, error) {
	// authenticate aws
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	listResp, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String("kite-deploys"),
		Prefix: aws.String(release),
	})
	if err != nil {
		return false, fmt.Errorf("error getting listing from s3: %v", err)
	}

	// if empty, release does not exist so return false
	if len(listResp.Contents) == 0 {
		return false, nil
	}

	// map for bucket contents
	contents := make(map[string]bool)
	for _, key := range listResp.Contents {
		contents[*key.Key] = true
	}

	// check that the binaries are present
	for _, bin := range binaries {
		key := fmt.Sprintf("%s/%s", release, bin)
		if !contents[key] {
			return false, fmt.Errorf("could not find %s for %s", bin, release)
		}
	}

	// return true if all required binaries are present
	return true, nil
}

// createReleaseRG creates a new resource group which will contain the resources for the release
// deployment
func createReleaseRG(release string, location string) (string, error) {
	// init client
	client := resources.NewGroupsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	// check if resource group exists, return error if it does.
	//
	// NOTE: Get() will return an error if the resource group is not found, so we check to see if
	// there was no error to see if the resource group already exists
	rgName := releaseRGName(release, location)
	_, err := client.Get(context.Background(), rgName)
	if err == nil {
		return "", fmt.Errorf("resource group %s already exists", rgName)
	}

	// create new resource group
	rg := resources.Group{Name: &rgName, Location: &location}

	// submit changes
	_, err = client.CreateOrUpdate(context.Background(), rgName, rg)
	if err != nil {
		return "", err
	}

	return rgName, nil
}

// createVMSSForProcess creates a scale set for the process using the given data
func (d *deployCommand) createVMSSForProcess(process string, vd VMSSData, pd ProvisionScriptTemplateData) error {
	// update provision script
	pd.Process = process
	scriptName := fmt.Sprintf("%s-%s-provision.sh", pd.Release, pd.Process)
	if err := d.updateProvisionScript(pd, scriptName); err != nil {
		return err
	}

	// create scale set
	vd.Process = process
	vd.ScriptName = scriptName
	// createVMSS either returns error or nil
	return createVMSS(vd)
}

// updateProvisionScript generates and uploads the provisioning script with the given name
func (d *deployCommand) updateProvisionScript(tData ProvisionScriptTemplateData, scriptName string) error {
	// render template to string with template data
	text, err := readFromTemplate(d.templates, "provision.sh", tData)
	if err != nil {
		return err
	}

	// initialize new storage blob client
	sc, err := storage.NewBasicClient(tData.StorageName, tData.StorageKey)
	if err != nil {
		return err
	}
	blobCli := sc.GetBlobService()
	ctn := blobCli.GetContainerReference(tData.StorageContainer)

	// upload
	err = uploadFile(ctn, scriptName, text)
	if err != nil {
		return err
	}

	return nil
}

// createVMSS creates a scale set with the given data
func createVMSS(data VMSSData) error {
	// initialize load balancer pool if this is user-mux
	var lbpool *[]compute.SubResource

	// init vmss client
	client := compute.NewVirtualMachineScaleSetsClient(subID)
	client.Authorizer = autorest.NewBearerAuthorizer(auth)

	// create vmss struct from template data
	//
	// NOTE: to find the lines that have actual values, look for either `data.` or `to.`
	vmss := compute.VirtualMachineScaleSet{
		Name:     &data.Process,
		Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
		Location: &data.Location,
		Sku: &compute.Sku{
			Name:     to.StringPtr(data.ScaleSets[data.Process].SkuName),
			Tier:     to.StringPtr(data.ScaleSets[data.Process].SkuTier),
			Capacity: to.Int64Ptr(int64(data.ScaleSets[data.Process].InstanceCount)),
		},
		VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
			UpgradePolicy:        &compute.UpgradePolicy{Mode: compute.Manual},
			Overprovision:        to.BoolPtr(false),
			SinglePlacementGroup: to.BoolPtr(true),
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
				OsProfile: &compute.VirtualMachineScaleSetOSProfile{
					ComputerNamePrefix: &data.Process,
					AdminUsername:      &data.VMUser,
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(true),
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								compute.SSHPublicKey{
									Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", data.VMUser)),
									KeyData: &data.SSHKey,
								},
							},
						},
					},
				},
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr("Canonical"),
						Offer:     to.StringPtr("UbuntuServer"),
						Sku:       to.StringPtr("16.04-LTS"),
						Version:   to.StringPtr("latest"),
					},
					OsDisk: &compute.VirtualMachineScaleSetOSDisk{
						Name:         to.StringPtr(fmt.Sprintf("%s_os_disk", data.Release)),
						Caching:      compute.CachingTypesReadWrite,
						CreateOption: compute.DiskCreateOptionTypesFromImage,
						VhdContainers: &[]string{
							data.VHDContainer,
						},
					},
				},
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
						compute.VirtualMachineScaleSetNetworkConfiguration{
							Name: to.StringPtr(fmt.Sprintf("%s-nic", data.Process)),
							VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
								Primary: to.BoolPtr(true),
								IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
									compute.VirtualMachineScaleSetIPConfiguration{
										Name: to.StringPtr(fmt.Sprintf("%s-ipc", data.Process)),
										VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
											Subnet: &compute.APIEntityReference{
												ID: &data.SubnetID,
											},
											LoadBalancerBackendAddressPools: lbpool,
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
									"fileUris": []string{fmt.Sprintf("%s/%s", data.VMSetupScriptURIBase, data.ScriptName)},
								},
								ProtectedSettings: &map[string]interface{}{
									"commandToExecute":   fmt.Sprintf("%s %s 2> /var/log/%s-provision-errors.log", data.VMSetupCommandBase, data.ScriptName, data.Process),
									"storageAccountKey":  data.StorageKey,
									"storageAccountName": data.StorageName,
								},
							},
						},
					},
				},
			},
		},
	}

	// send create request and wait for response
	rg := data.ReleaseRG
	vmssName := data.Process
	resp, err := client.CreateOrUpdate(context.Background(), rg, vmssName, vmss)
	_ = resp // the returned struct only contains an ID; we don't care about it for now

	return err
}

// get the vmss pool ID given the resource group and load balancer name
func getLBPoolID(rg string, lb *network.LoadBalancer, poolName string) (string, error) {
	// nil pointer check
	if lb.BackendAddressPools == nil {
		return "", fmt.Errorf("nil pointer: lb.BackendAddressPools")
	}
	// find the backend address pool corresponding to the given pool name and return the ID
	pool, err := findLBPool(*lb.BackendAddressPools, poolName)
	if err != nil {
		return "", err
	}
	return *pool.ID, nil
}
