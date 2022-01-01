package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func main() {
	var format string
	flag.StringVar(&format, "format", "", "emr, json, or jsongz")
	flag.Parse()

	var r io.Reader
	switch flag.NArg() {
	case 0:
		r = os.Stdin
	case 1:
		input := flag.Arg(0)
		if format == "" {
			format = formatFromPath(input)
		}
		file, err := fileutil.NewCachedReader(input)
		if err != nil {
			log.Fatalln("unable to create reader for", input, "error:", err)
		}
		r = file
		defer file.Close()
	default:
		log.Fatalln("unexpected number of arguments")
	}

	if format == "" {
		// Attempt to deduce the format automatically
		bufr := bufio.NewReaderSize(r, 100)
		head, _ := bufr.Peek(100)
		r = bufr
		if len(head) > 0 && head[0] == '{' {
			format = "json"
		} else if isGzipHeader(head) {
			format = "jsongz"
		} else {
			log.Fatalln("-format option required (emr, json, or jsongz)")
		}
	}

	switch format {
	case "json":
		handleJSON(r)
	case "jsongz":
		handleGzippedJSON(r)
	case "emr":
		handleEMR(r)
	}
}

// determine whether buf constitutes the beginning of a valid gzip stream
func isGzipHeader(buf []byte) bool {
	_, err := gzip.NewReader(bytes.NewBuffer(buf))
	return err == nil
}

func formatFromPath(path string) string {
	if strings.HasSuffix(path, ".json") {
		return "json"
	}
	if strings.HasSuffix(path, ".json.gz") {
		return "jsongz"
	}
	if strings.HasSuffix(path, ".emr") {
		return "emr"
	}
	parts := strings.Split(path, "/")
	if len(parts) > 0 && strings.HasPrefix(parts[len(parts)-1], "part-") {
		return "emr"
	}
	return ""
}

func handleGzippedJSON(r io.Reader) {
	decomp, err := gzip.NewReader(r)
	if err != nil {
		log.Fatalln("error creating gzip reader:", err)
	}
	handleJSON(decomp)
}

func handleJSON(r io.Reader) {
	decoder := json.NewDecoder(r)
	for {
		var msg json.RawMessage
		err := decoder.Decode(&msg)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalln("error while reading:", err)
		}

		var out bytes.Buffer
		err = json.Indent(&out, []byte(msg), "", "  ")
		if err != nil {
			log.Fatalln("error formatting object:", err)
		}

		fmt.Println(out.String())
	}
}

func handleEMR(r io.Reader) {
	iter := awsutil.NewEMRIterator(r)
	for iter.Next() {
		var buf bytes.Buffer
		err := json.Indent(&buf, iter.Value(), "", "  ")
		if err != nil {
			log.Fatalln("unable to indent:", err)
		}

		fmt.Println("key:", iter.Key(), "value:", buf.String())
	}

	if err := iter.Err(); err != nil {
		log.Fatalln("error while reading:", err)
	}
}
