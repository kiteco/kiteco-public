package main

import (
	"encoding/json"
	"log"
	"os"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmetrics"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
)

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type Data struct {
	Trace          pythonmetrics.ASTTraceEvent `json:"trace"`
	Handled        bool                        `json:"handled"`
	NumCompletions int                         `json:"num_completions"`
	UserID         string                      `json:"user_id"`
	SentAt         int64                       `json:"sent_at"`
}

func main() {
	today := analyze.Today()
	lastWeek := today.Add(0, 0, -7)
	args := struct {
		Start *analyze.Date
		End   *analyze.Date
	}{
		Start: &lastWeek,
		End:   &today,
	}
	arg.MustParse(&args)

	listing, err := analyze.ListRange(segmentsrc.Production, *args.Start, *args.End)
	fail(err)

	var URIs []string
	for _, d := range listing.Dates {
		URIs = append(URIs, d.URIs...)
	}
	log.Printf("found %d URIs within the provided date range", len(URIs))

	stdoutJSON := json.NewEncoder(os.Stdout)
	analyze.Analyze(URIs, 16, "asttrace_completions", func(meta analyze.Metadata, data *Data) {
		if data == nil {
			return
		}
		fail(stdoutJSON.Encode(*data))
	})
}
