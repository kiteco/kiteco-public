package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

const (
	minScore     = 100
	maxDepth     = 3
	minKwargFreq = 50
)

func main() {
	if err := datadeps.Enable(); err != nil {
		log.Fatal(err)
	}

	args := struct {
		Packages string `arg:"required"`
		Endpoint string
		Out      string `arg:"required"`
	}{
		Packages: "~/go/src/github.com/kiteco/kiteco/local-pipelines/python-ggnn-call-completion/packagelist.txt",
		Endpoint: "http://ml-training-0.kite.com:3039/symbol/scores",
	}

	arg.MustParse(&args)

	pkgs, err := traindata.LoadPackageList(args.Packages)
	if err != nil {
		log.Fatal(err)
	}

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	info, err := pythonexpr.ComputeMetaInfo(rm, args.Endpoint, pkgs, minScore, maxDepth, minKwargFreq)
	if err != nil {
		log.Fatal(err)
	}

	if err := info.Valid(); err != nil {
		log.Fatal(err)
	}

	outf, err := os.Create(args.Out)
	if err != nil {
		log.Fatal(err)
	}
	defer outf.Close()

	if err := json.NewEncoder(outf).Encode(info); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done! took %v\n", time.Since(start))
	fmt.Printf("Fuctions: %d\n", len(info.Call.Infos))
	fmt.Printf("Attrs: %d\n", len(info.Attr.Dist))
	fmt.Printf("Base attrs: %d\n", len(info.AttrBase.Dist))
	fmt.Printf("Name Subtokens: %d\n", len(info.NameSubtokenIndex))
	fmt.Printf("Type subtokens: %d\n", len(info.TypeSubtokenIndex))
	fmt.Printf("Productions: %d\n", len(info.ProductionIndex.Indices))
}
