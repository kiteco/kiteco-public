package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func main() {
	flag.Parse()
	n := flag.NArg()
	if n != 1 {
		fmt.Println("Usage: kfscat PATH")
		os.Exit(1)
	}

	// Avoid polluting output stream with log
	log.SetOutput(ioutil.Discard)

	// Open the file
	path := flag.Arg(0)
	r, err := fileutil.NewCachedReader(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Copy to stdout
	_, err = io.Copy(os.Stdout, r)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
