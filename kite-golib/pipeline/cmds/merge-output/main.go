package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
)

func noErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		In    string
		Out   string
		Force bool
	}{}
	arg.MustParse(&args)

	out, err := os.Create(args.Out)
	noErr(err)
	defer out.Close()

	fs, err := fileutil.ListDir(args.In)
	noErr(err)

	if len(fs) == 0 {
		log.Fatal(fmt.Errorf("no files found in %s", args.In))
	}

	var foundDone bool
	for _, f := range fs {
		if strings.HasSuffix(f, "/"+aggregator.DoneFilename) {
			foundDone = true
			break
		}
	}

	if !foundDone && !args.Force {
		log.Fatal(fmt.Errorf("no DONE file in %s", args.In))
	}

	var totalSize int64
	for _, f := range fs {
		if strings.HasSuffix(f, "/"+aggregator.DoneFilename) {
			continue
		}
		r, err := fileutil.NewReader(f)
		noErr(err)
		size, err := io.Copy(out, r)
		noErr(err)
		totalSize += size
		noErr(r.Close())
	}

	log.Printf("wrote %d bytes from %d files in %s to %s", totalSize, len(fs), args.In, args.Out)
}
