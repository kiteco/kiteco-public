package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/diskmapindex"
)

func merge(bufs [][]byte) ([]byte, error) {
	var all []pythoncode.HashCounts
	for _, buf := range bufs {
		hcs, err := pythoncode.DecodeHashes(buf)
		if err != nil {
			return nil, err
		}
		all = append(all, hcs...)
	}

	return pythoncode.EncodeHashes(all)
}

func main() {
	args := struct {
		In  string
		Out string
	}{}
	arg.MustParse(&args)

	if err := os.MkdirAll(args.Out, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("merging %s -> %s\n", args.In, args.Out)

	start := time.Now()
	defer func() {
		fmt.Println("Done, took", time.Since(start))
	}()

	err := diskmapindex.Merge(diskmapindex.MergeOptions{
		Builder: diskmapindex.BuilderOptions{
			Compress: true,
		},
		MaxBlockSizeBytes:  5e9,
		WaitOnBlockWriting: true,
	}, args.In, args.Out, merge)

	if err != nil {
		log.Fatal(err)
	}
}
