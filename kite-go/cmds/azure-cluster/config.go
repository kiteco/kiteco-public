package main

var (
	conf = config{
		Location:       "westus2",
		ResourceGroup:  "dev",
		SubnetID:       "/subscriptions/XXXXXXX/resourceGroups/dev/providers/Microsoft.Network/virtualNetworks/dev/subnets/subnet-private",
		SubscriptionID: "XXXXXXX",
		SkuTier:        "Standard",

		VMUser:       "ubuntu",
		VHDContainer: "XXXXXXX",
		SSHKey:       "ssh-rsa XXXXXXX",

		StorageName:          "XXXXXXX",
		StorageContainer:     "provisioning",
		ProvisionDataURIBase: "XXXXXXX",
	}
)

type config struct {
	Location       string
	ResourceGroup  string
	SubnetID       string
	SubscriptionID string
	SkuTier        string

	VMUser       string
	VHDContainer string
	SSHKey       string

	StorageName          string
	StorageContainer     string
	ProvisionDataURIBase string
}
