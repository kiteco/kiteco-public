package main

import (
	"bufio"
	"compress/gzip"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	comp, err := gzip.NewReader(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	var count int
	r := bufio.NewReader(comp)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		err = w.Emit(strconv.Itoa(count), line)
		if err != nil {
			log.Fatal(err)
		}
		count++
	}
}
