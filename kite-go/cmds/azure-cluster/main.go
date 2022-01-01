//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"log"
	"strconv"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	"github.com/spf13/cobra"
)

const defaultInstanceType = "Standard_B2MS"

var (
	staticfs  = &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates = templateset.NewSet(staticfs, "templates", nil)
)

func init() {
	log.SetPrefix("[azure-cluster] ")
}

func bundleCmd() *cobra.Command {
	var bundleGoBinaries *[]string
	var bundleKitecoPaths *[]string
	var bundleKiteML *bool
	var installCUDA *bool
	var dockerML *bool

	bundleCmd := cobra.Command{
		Use:   "bundle BUNDLE_FILE RUN_SCRIPT",
		Short: "create a bundle that can be deployed to an instance",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			bundleFile := args[0]
			runScript := args[1]

			conf := bundleConfig{
				GoBinaries:   *bundleGoBinaries,
				KitecoPaths:  *bundleKitecoPaths,
				BundleKiteML: *bundleKiteML,
				InstallCUDA:  *installCUDA,
				DockerML:     *dockerML,
			}

			if err := createBundle(bundleFile, runScript, conf); err != nil {
				log.Fatalln(err)
			}
		},
	}

	bundleGoBinaries = bundleCmd.Flags().StringSlice("go-binary", nil,
		"list of package paths for Go binaries that should be built and included in the bundle")

	bundleKitecoPaths = bundleCmd.Flags().StringSlice("kiteco-path", nil,
		"List of paths relative to kiteco root that should be included in the bundle")

	bundleKiteML = bundleCmd.Flags().Bool("kite-ml", false,
		"include kite_ml code and virtualenv")

	installCUDA = bundleCmd.Flags().Bool("cuda", false,
		"configure bundle to install NVIDIA GPU driver and CUDA library")

	dockerML = bundleCmd.Flags().Bool("docker-ml", false,
		"setup nvidia drivers, docker, and nvidia-container-toolkit")

	return &bundleCmd
}

func startCmd() *cobra.Command {
	var waitReady *bool
	var instanceType *string
	var bionic *bool

	startCmd := cobra.Command{
		Use:   "start CLUSTER_PREFIX COUNT",
		Short: "start a new cluster, returning the resulting cluster name",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			clusterPrefix := args[0]
			count, err := strconv.Atoi(args[1])
			if err != nil {
				log.Fatalf("error parsing count: %v", err)
			}

			clusterName := newClusterName(clusterPrefix)

			log.Printf("creating cluster with name %s", clusterName)
			if *bionic {
				log.Println("using ubuntu 18.04")
			}

			if err := startCluster(clusterName, count, *instanceType, *bionic); err != nil {
				log.Fatalln(err)
			}

			if *waitReady {
				wait(clusterName)
			}

			// NOTE: This is used by scripts to store the cluster name
			fmt.Println(clusterName)
		},
	}

	waitReady = startCmd.Flags().BoolP(
		"wait", "w", false, "wait for instances to be ready for provisioning")
	bionic = startCmd.Flags().BoolP(
		"bionic", "b", false, "use ubuntu 18.04 LTS (bionic)")
	instanceType = startCmd.Flags().StringP(
		"instance_type", "t", defaultInstanceType, "instance type to use")

	return &startCmd
}

func readyCmd() *cobra.Command {
	readyCmd := cobra.Command{
		Use:   "ready CLUSTER_NAME",
		Short: "wait for a cluster to be provisioned",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			clusterName := args[0]
			wait(clusterName)
		},
	}

	return &readyCmd
}

func deployCmd() *cobra.Command {
	var cleanupClusters *[]string

	deployCmd := cobra.Command{
		Use:   "deploy BUNDLE_FILE CLUSTER_NAME",
		Short: "deploy a bundle to each instance of a cluster and start executing the bundle's run script",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			bundleFile := args[0]
			clusterName := args[1]

			if err := deployBundle(bundleFile, clusterName, *cleanupClusters); err != nil {
				log.Fatalln(err)
			}
		},
	}

	cleanupClusters = deployCmd.Flags().StringSlice("cleanup", nil,
		"list of cluster names to stop once the bundle finishes running on any of the instances")

	return &deployCmd
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop CLUSTER_NAME",
		Short: "stop a cluster with the given name",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			clusterName := args[0]
			log.Printf("stopping cluster %s", clusterName)
			if err := stopCluster(clusterName); err != nil {
				log.Fatalln(err)
			}
		},
	}
}

func ipsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ips CLUSTER_NAME",
		Short: "return the IP addresses of all instances in the cluster",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			clusterName := args[0]
			ips, err := getClusterIPs(clusterName)
			if err != nil {
				log.Fatalln(err)
			}
			for _, ip := range ips {
				fmt.Println(ip)
			}
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "return the names of all clusters that have been created",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			clusters, err := listClusters()
			if err != nil {
				log.Fatalln(err)
			}
			for _, cluster := range clusters {
				fmt.Println(cluster)
			}
		},
	}
}

func main() {
	rootCmd := &cobra.Command{Use: "azure-cluster"}
	rootCmd.AddCommand(bundleCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(readyCmd())
	rootCmd.AddCommand(deployCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(ipsCmd())
	rootCmd.AddCommand(listCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
