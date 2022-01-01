package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	mtacutils "github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/internal/utils"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

const (
	s3Region    = "us-west-1"
	datasetPath = "s3://kite-emr/users/juan/python-dedupe-code/2018-07-26_13-53-43-PM/dedupe/output/"
)

var (
	// parseOpts should match the options in the driver (or whatever is running inference with the model)
	parseOpts = pythonparser.Options{
		ScanOptions: pythonscanner.Options{
			ScanComments: false,
			ScanNewLines: false,
		},
		ErrorMode: pythonparser.Recover,
	}
)

//find name expression sites under scenarios
func findNameSites(rast *pythonanalyzer.ResolvedAST) []int64 {
	var cursors []int64
	names := mtacutils.FindNameExprScenarios(rast)
	for _, name := range names {
		cursors = append(cursors, int64(name.Begin()))
	}
	return cursors
}

func getSample(src []byte, res resources) (sample, error) {
	ast, err := pythonparser.Parse(kitectx.Background(), src, parseOpts)
	if err != nil {
		return sample{}, fmt.Errorf("unable to parse file: %v", err)
	}

	rast, err := mtacutils.Resolve(ast, res.RM)
	if err != nil {
		return sample{}, fmt.Errorf("error resolving ast: %v", err)
	}

	cursors := findNameSites(rast)
	if len(cursors) == 0 {
		return sample{}, fmt.Errorf("no name expressions in call found")
	}

	return sample{
		Source:  src,
		PosList: cursors,
	}, nil
}

// need this to communicate with S3. This is to get keys associates with data partitions.
func bucketAndKeys(path string) (string, []string) {
	uri, err := awsutil.ValidateURI(path)
	if err != nil {
		log.Fatalln(err)
	}

	bucket := uri.Host
	prefix := uri.Path[1:]

	keys, err := awsutil.S3ListObjects(s3Region, bucket, prefix)
	if err != nil {
		log.Fatalln(err)
	}

	return bucket, keys
}

type sample struct {
	Source  []byte
	PosList []int64
}

type resources struct {
	RM     pythonresource.Manager
	Models *pythonmodels.Models
}

func main() {
	datadeps.Enable()
	args := struct {
		Out      string
		MaxFiles int
	}{
		Out:      "./gt_data.json",
		MaxFiles: 10000,
	}

	arg.MustParse(&args)
	outFile := args.Out
	maxFiles := args.MaxFiles

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}
	serviceOpts := python.DefaultServiceOptions

	models, err := pythonmodels.New(serviceOpts.ModelOptions)
	if err != nil {
		log.Fatalln(err)
	}

	res := resources{RM: rm, Models: models}

	start := time.Now()
	bucket, keys := bucketAndKeys(datasetPath)

	var skippedErr int
	var successful int
	var total int
	var done bool

	outf, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer outf.Close()

	encoder := json.NewEncoder(outf)
	for _, key := range keys {
		uri := fmt.Sprintf("s3://%s/%s", bucket, key)
		log.Printf("reading %s", uri)

		r, err := awsutil.NewCachedS3Reader(uri)
		if err != nil {
			log.Fatalln(err)
		}

		in := awsutil.NewEMRIterator(r)

		for in.Next() {
			total++
			if total > maxFiles {
				done = true
				break
			}

			sample, err := getSample(in.Value(), res)
			if err != nil {
				skippedErr++
				log.Printf("error getting sample: %v", err)
				continue
			}

			if err := encoder.Encode(sample); err != nil {
				log.Fatal(err)
			}
			successful++
		}
		if done {
			break
		}

	}

	log.Printf("Done! took %v, successful: %d, skipped (err): %d\n",
		time.Since(start), successful, skippedErr)

}
