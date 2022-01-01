package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/local-pipelines/python-import-exploration/helpers"
	"github.com/spf13/cobra"
)

func build(cmd *cobra.Command, args []string) {
	buf, err := ioutil.ReadFile(args[0])
	if err != nil {
		log.Fatalln(errors.Wrapf(err, "unable to read input file %s", args[0]))
	}

	// make sure we can write file before fetching URLs.
	var out bytes.Buffer
	start := time.Now()

	byName := make(map[string]keytypes.Distribution)
	err = helpers.ReadLinesWithComments(string(buf), func(distStr, comment string) error {
		if distStr == "" {
			out.WriteString(comment)
			out.WriteString("\n")
			return nil
		}

		d, err := keytypes.ParseDistribution(distStr)
		if err != nil {
			return errors.Wrapf(err, "could not parse distribution string %s\n", distStr)
		}
		d = d.Normalize()

		if _, ok := byName[keytypes.NormalizeDistName(d.Name)]; ok {
			log.Printf("skipping duplicate distribution name %s\n", d.Name)
			if comment != "" {
				log.Printf("dropping comment %s\n", comment)
			}
			return nil
		}
		byName[d.Name] = d

		if len(byName)%100 == 0 {
			log.Printf("processed %d distributions in %v\n", len(byName), time.Since(start))
		}

		v, err := getLatestVersionFor(d)
		if err == nil {
			d.Version = v
		} else {
			log.Printf("error updating version for distribution %s: %v\n", d, err)
			if d.Version == "" {
				if comment != "" {
					log.Printf("dropping comment %s\n", comment)
				}
				return nil
			}
		}

		out.WriteString(d.String())
		if comment != "" {
			out.WriteString(" ")
			out.WriteString(comment)
		}
		out.WriteString("\n")
		return nil
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Done! took %v to process for %d packages\n", time.Since(start), len(byName))

	f := os.Stdout
	if inplace {
		f, err = os.Create(args[0])
		if err != nil {
			log.Fatalln(errors.Wrapf(err, "unable to create file %s", args[0]))
		}
		defer f.Close()
	}
	fmt.Fprint(f, out.String())
}

var cmd = cobra.Command{
	Use:   "packagelist INPUT_FILE",
	Short: "update package list versions from PyPI",
	Args:  cobra.ExactArgs(1),
	Run:   build,
}
var inplace bool

func init() {
	cmd.Flags().BoolVarP(&inplace, "inplace", "o", false, "write the output back to the provided file")
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
