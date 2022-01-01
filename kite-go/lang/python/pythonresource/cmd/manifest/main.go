package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func checkError(e error) {
	// "value of 1 prints details of caller of Output," so value of 2 for caller of checkError
	if e != nil {
		log.Output(2, e.Error())
		os.Exit(1)
	}
}

func loadManifest(filePath string) manifest.Manifest {
	f, err := os.Open(filePath)
	checkError(err)
	defer f.Close()

	m, err := manifest.New(f)
	checkError(err)

	return m
}

var outputPath string

func writeManifest(m manifest.Manifest) {
	out := os.Stdout
	if outputPath != "" {
		var err error
		out, err = os.Create(outputPath)
		checkError(err)
		defer out.Close()
	}

	checkError(m.Encode(out))
}

func init() {
	rewrite.Flags().StringVarP(&outputPath, "output", "o", "", "manifest output path (defaults to stdout)")
	merge.Flags().StringVarP(&outputPath, "output", "o", "", "manifest output path (defaults to stdout)")
	extract.Flags().StringVarP(&outputPath, "output", "o", "", "manifest output path (defaults to stdout)")
	remove.Flags().StringVarP(&outputPath, "output", "o", "", "manifest output path (defaults to stdout)")
}

var rewrite = cobra.Command{
	Use:   "rewrite orig_manifest.json /orig/prefix/ [/new/prefix/]",
	Short: "rewrite a manifest, updating the prefix of all paths",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		origManifest := loadManifest(args[0])

		newPrefix := builder.DefaultResourceRoot
		if len(args) > 2 {
			newPrefix = args[2]
		}
		if !strings.HasPrefix(newPrefix, "s3://") && !filepath.IsAbs(newPrefix) {
			log.Printf("WARNING: using non-absolute prefix %s for manifest; this should only be used for test manifests\n", newPrefix)
		}
		newPrefixURL, err := url.Parse(newPrefix)
		checkError(err)
		newPrefixPath := newPrefixURL.Path

		oldPrefix := args[1]
		// all manifests loaded through manifest.New have absolute paths, so make the input oldPrefix absolute as well
		if !strings.HasPrefix(oldPrefix, "s3://") {
			oldPrefix, err = filepath.Abs(oldPrefix)
			checkError(err)
		}
		prefixTransformer := func(p string) (string, error) {
			// s3:// will get converted to s3:/ here, but that's acceptable as
			// if only one of p and oldPrefix have s3://, then the following prefix check will fail, and
			// if both p and oldPrefix have s3://, then the s3:// will be dropped entirely from relpath
			relpath, err := filepath.Rel(oldPrefix, p)
			if err != nil {
				return "", errors.Wrap(err, "")
			}
			relpath = filepath.ToSlash(relpath)

			if strings.HasPrefix(relpath, "../") {
				return "", errors.Errorf("path %s does not have prefix %s", p, oldPrefix)
			}

			// join, maintaining a possible s3:// in newPrefix
			newPrefixURL.Path = path.Join(newPrefixPath, relpath)
			return newPrefixURL.String(), nil
		}

		newManifest, err := manifestTransformPaths(origManifest, prefixTransformer)
		checkError(err)

		writeManifest(newManifest)
	},
}

var copydata = cobra.Command{
	Use:   "copydata src_manifest.json dst_manifest.json",
	Short: "copy all files in a manifest to file locations in an isomorphic manifest",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		srcManifest := loadManifest(args[0])
		dstManifest := loadManifest(args[1])
		checkError(manifestCopyResources(srcManifest, dstManifest))
	},
}

var merge = cobra.Command{
	Use:   "merge base_manifest.json manifest1.json [manifest2.json ...]",
	Short: "create a new manifest by updating a manifest with the contents of other manifests",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		baseManifest := loadManifest(args[0])
		for _, mPath := range args[1:] {
			m := loadManifest(mPath)
			manifestMerge(baseManifest, m)
		}

		writeManifest(baseManifest)
	},
}

var extract = cobra.Command{
	Use:   "extract manifest.json distribution version",
	Short: "extract a singleton manifest containing only the specified distribution/version",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		m := loadManifest(args[0])

		v := ""
		if len(args) > 2 {
			v = args[2]
		}

		newM, err := manifestExtract(m, keytypes.Distribution{Name: args[1], Version: v})
		checkError(err)
		writeManifest(newM)
	},
}

var remove = cobra.Command{
	Use:   "removeKey manifest.json key_to_remove",
	Short: "Remove all entries for the specified key in the manifest",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		m := loadManifest(args[0])
		var removedEntries int
		removedKey := args[1]
		result := make(manifest.Manifest)
		for dist, loc := range m {
			if _, ok := loc[removedKey]; ok {

				delete(loc, removedKey)
				removedEntries++
			}
			result[dist] = loc
		}
		if removedEntries == 0 {
			fmt.Printf("No entries with the name %s found, are you sure it's the right name? (no output written)\n", removedKey)
			os.Exit(1)
		}
		writeManifest(result)
		fmt.Printf("%d %s entries removed\n", removedEntries, removedKey)
	},
}

func main() {
	rootCmd := &cobra.Command{Use: "manifest"}
	rootCmd.AddCommand(&rewrite)
	rootCmd.AddCommand(&copydata)
	rootCmd.AddCommand(&merge)
	rootCmd.AddCommand(&extract)
	rootCmd.AddCommand(&remove)

	rootCmd.Execute()
}
