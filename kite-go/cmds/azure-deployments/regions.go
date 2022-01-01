package main

var (
	// WestUS2 contains the configurations for the West US 2 region
	WestUS2 = Region{
		Location:       "westus2",
		ResourceGroup:  "prod-westus2-0",
		SubnetID:       "/subscriptions/XXXXXXX/resourceGroups/prod-westus2-0/providers/Microsoft.Network/virtualNetworks/prod/subnets/subnet-private",
		SubscriptionID: "XXXXXXX",

		ScaleSets: map[string]ScaleSet{
			"user-node": ScaleSet{
				InstanceCount: 2,
				SkuName:       "Standard_E4s_v3",
				SkuTier:       "Standard",
			},
			"user-mux": ScaleSet{
				InstanceCount: 2,
				SkuName:       "Standard_A1",
				SkuTier:       "Standard",
			},
		},
		VMUser:               "ubuntu",
		VHDContainer:         "https://XXXXXXX.blob.core.windows.net/vhds",
		SSHKey:               "ssh-rsa XXXXXXX",
		VMSetupScriptURIBase: "https://XXXXXXX.blob.core.windows.net/provisioning",
		VMSetupCommandBase:   "sudo bash",
		StorageName:          "XXXXXXX",
		StorageContainer:     "provisioning",

		HAProxyVMSSNames: map[string]string{
			"staging": "haproxy-staging",
			"prod":    "haproxy-prod",
		},

		ProdAG:     "agw-prod",
		StagingAG:  "agw-staging",
		AGPoolName: "lbpool",

		LBNames:    []string{"deployments-0", "deployments-1", "deployments-2"},
		LBFIPCName: "deployments-frontend",
		LBPoolName: "deployments-pool",
		LBRuleName: "deployments-rule",
		AWSRegion:  "us-west-1",
	}

	// EastUS contains the configurations for the East US region
	EastUS = Region{
		Location:       "eastus",
		ResourceGroup:  "prod-eastus-0",
		SubnetID:       "/subscriptions/XXXXXXX/resourceGroups/prod-eastus-0/providers/Microsoft.Network/virtualNetworks/prod/subnets/subnet-private",
		SubscriptionID: "XXXXXXX",

		ScaleSets: map[string]ScaleSet{
			"user-node": ScaleSet{
				InstanceCount: 2,
				SkuName:       "Standard_E4s_v3",
				SkuTier:       "Standard",
			},
			"user-mux": ScaleSet{
				InstanceCount: 2,
				SkuName:       "Standard_A1",
				SkuTier:       "Standard",
			},
		},
		VMUser:               "ubuntu",
		VHDContainer:         "https://XXXXXXX.blob.core.windows.net/vhds",
		SSHKey:               "ssh-rsa XXXXXXX",
		VMSetupScriptURIBase: "https://XXXXXXX.blob.core.windows.net/provisioning",
		VMSetupCommandBase:   "sudo bash",
		StorageName:          "XXXXXXX",
		StorageContainer:     "provisioning",

		HAProxyVMSSNames: map[string]string{
			"staging": "haproxy-staging",
			"prod":    "haproxy-prod",
		},

		ProdAG:     "agw-prod",
		StagingAG:  "agw-staging",
		AGPoolName: "lbpool",

		LBNames:    []string{"deployments-0", "deployments-1", "deployments-2"},
		LBFIPCName: "deployments-frontend",
		LBPoolName: "deployments-pool",
		LBRuleName: "deployments-rule",
		AWSRegion:  "us-east-1",
	}

	// WestEU contains the configurations for the West Europe region
	WestEU = Region{
		Location:       "westeurope",
		ResourceGroup:  "prod-westeurope-0",
		SubnetID:       "/subscriptions/XXXXXXX/resourceGroups/prod-westeurope-0/providers/Microsoft.Network/virtualNetworks/prod/subnets/subnet-private",
		SubscriptionID: "XXXXXXX",

		ScaleSets: map[string]ScaleSet{
			"user-node": ScaleSet{
				InstanceCount: 2,
				SkuName:       "Standard_E4s_v3",
				SkuTier:       "Standard",
			},
			"user-mux": ScaleSet{
				InstanceCount: 2,
				SkuName:       "Standard_A1",
				SkuTier:       "Standard",
			},
		},
		VMUser:               "ubuntu",
		VHDContainer:         "https://XXXXXXX.blob.core.windows.net/vhds",
		SSHKey:               "ssh-rsa XXXXXXX",
		VMSetupScriptURIBase: "https://XXXXXXX.blob.core.windows.net/provisioning",
		VMSetupCommandBase:   "sudo bash",
		StorageName:          "XXXXXXX",
		StorageContainer:     "provisioning",

		HAProxyVMSSNames: map[string]string{
			"staging": "haproxy-staging",
			"prod":    "haproxy-prod",
		},

		ProdAG:     "agw-prod",
		StagingAG:  "agw-staging",
		AGPoolName: "lbpool",

		LBNames:    []string{"deployments-0", "deployments-1", "deployments-2"},
		LBFIPCName: "deployments-frontend",
		LBPoolName: "deployments-pool",
		LBRuleName: "deployments-rule",
		AWSRegion:  "eu-west-1",
	}

	// Regions is the list of all regions
	Regions = map[string]Region{
		"westus2":    WestUS2,
		"eastus":     EastUS,
		"westeurope": WestEU,
	}
)

// Region contains values from terraform that is needed to create and interact with deployments
type Region struct {
	Location             string
	ResourceGroup        string
	SubnetID             string
	SubscriptionID       string
	ScaleSets            map[string]ScaleSet
	VMUser               string
	VHDContainer         string
	SSHKey               string
	VMSetupScriptURIBase string
	VMSetupCommandBase   string
	StorageName          string
	StorageContainer     string
	ProdAG               string
	StagingAG            string
	AGPoolName           string
	LBNames              []string
	HAProxyVMSSNames     map[string]string
	LBFIPCName           string
	LBPoolName           string
	LBRuleName           string
	AWSRegion            string
}

// ScaleSet contains values specific to a scale set
type ScaleSet struct {
	InstanceCount int
	SkuName       string
	SkuTier       string
}

// Credentials contains credentials loaded from the local machine
type Credentials struct {
	AWSID      string
	AWSKey     string
	StorageKey string
}

// VMSSData contains both Region and Credentials as well as additional data needed for creating
// scale sets that are computed at runtime
type VMSSData struct {
	Region
	Credentials
	Release    string
	ReleaseRG  string
	Process    string
	ScriptName string
	LBPoolID   string
}

// ProvisionScriptTemplateData contains both Region and Credentials as well as additional data
// needed for generating template data that are computed at runtime
type ProvisionScriptTemplateData struct {
	Region
	Credentials
	Release       string
	ReleaseNoDots string
	ReleaseMD5    string
	Process       string
	IsProduction  bool
}
