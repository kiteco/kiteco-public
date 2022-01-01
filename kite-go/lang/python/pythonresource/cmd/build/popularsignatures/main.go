package main

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/spf13/cobra"
)

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}

const (
	manifestOutputPathDefault = "manifest_popularsignatures.json"
	newResourcesRootDefault   = "/tmp/manager-resources/popularsignatures/"
	// maximum number of signature patterns to include per symbol
	maxSignatures = 5
	// maximum number of examples to include per type
	maxExamples = 3
	// maximum number of types to include per pattern argument
	maxTypes = 4
	// minimum usage frequency for a pattern to be included
	minUsage = .02
)

// configurable constants
var (
	manifestPath                                      string
	outputRawDatasetPath, filteringResultPath         string
	manifestOutputPath, newResourcesRoot, distidxPath string
	callPatternsDataset                               string
	useDatadeps                                       bool
)

func init() {
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVar(&distidxPath, "distidx", "", "distribution index path (defaults to compiled-in KiteIndex)")
	cmd.Flags().StringVar(&manifestOutputPath, "manifest-output", manifestOutputPathDefault, "Location of the final manifest containing the list of resources for popular signatures, default to "+manifestOutputPathDefault)
	cmd.Flags().StringVar(&newResourcesRoot, "resources-output", newResourcesRootDefault, "Root folder where the new resources files will be written, it can be a s3 folder. Default to "+newResourcesRootDefault)
	cmd.Flags().StringVar(&callPatternsDataset, "call-patterns", pythoncode.CallPatterns, "Call patterns extracted from github")
}

func runCommand(cmd *cobra.Command, args []string) {
	if newResourcesRoot == newResourcesRootDefault {
		log.Println("The resources will be generated at the default location: " + newResourcesRootDefault + ". You can change it by using the flag '--resources-output'")
	}
	if manifestOutputPath == manifestOutputPathDefault {
		log.Println("The popular signature manifest will be generated at the default location: " + manifestOutputPathDefault + ". You can change it by using the flag '--manifest-output'")
	}

	buildResources()
}

// cli
var cmd = cobra.Command{
	Use:   "popularsignatures",
	Short: "Filter signature examples from resources raw dataset",
	Args:  cobra.ExactArgs(0),
	Run:   runCommand,
}

func main() {
	cmd.Execute()
}
