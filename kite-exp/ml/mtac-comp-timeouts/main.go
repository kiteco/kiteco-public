package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncompletions"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
)

func maybeQuit(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type data struct {
	Metrics pythoncompletions.Metrics `json:"metrics"`
}

func main() {
	args := struct {
		Out string
	}{
		Out: "data.json",
	}
	arg.MustParse(&args)

	start := time.Now()

	out, err := os.Create(args.Out)
	maybeQuit(err)
	defer out.Close()

	enc := json.NewEncoder(out)

	end := analyze.Today()
	begin := end.Add(0, 0, -2)

	listing, err := analyze.ListRange(segmentsrc.Production, begin, end)
	maybeQuit(err)

	var URIs []string
	for _, d := range listing.Dates {
		URIs = append(URIs, d.URIs...)
	}
	fmt.Printf("found %d URIs within the provided date range", len(URIs))

	var count int
	analyze.Analyze(URIs, 2, "completion_computation_metrics", func(meta analyze.Metadata, data *data) {
		if data == nil {
			return
		}
		count++

		maybeQuit(enc.Encode(data.Metrics))
	})

	fmt.Printf("Done! took %v to encode %d records\n", time.Since(start), count)
}
