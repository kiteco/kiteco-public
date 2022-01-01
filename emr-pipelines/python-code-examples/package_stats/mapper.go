package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRReader(os.Stdin)
	w := csv.NewWriter(os.Stdout)
	w.Comma = '\t'
	defer w.Flush()

	for {
		_, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var snippet pythoncode.Snippet
		err = json.Unmarshal(value, &snippet)
		if err != nil {
			log.Fatal(err)
		}

		// Emit the method name of every incantation, keyed by the
		// the root module name of the method.
		for _, attr := range snippet.Attributes {
			parts := strings.Split(attr, ".")
			if len(parts) == 0 {
				continue
			}
			pkg := parts[0]
			w.Write([]string{pkg, attr})
		}
	}
}
