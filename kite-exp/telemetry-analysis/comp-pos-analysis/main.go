package main

import (
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

func isCompSampleEvent(s pipeline.Sample) bool {
	ev := s.(pythonpipeline.Event)
	return ev.Completions.Failure == pythontracking.CompletionsSample
}

type record struct {
	Length             int64   `json:"length"`
	Cursor             int64   `json:"cursor"`
	ColPercentOfLine   float64 `json:"col_percent_of_line"`
	LinePercentOfTotal float64 `json:"line_percent_of_total"`
	BlankLine          bool    `json:"blank_line"`
}

func (record) SampleTag() {}

func getRecord(s pipeline.Sample) pipeline.Sample {
	ev := s.(pythonpipeline.Event)

	lm := linenumber.NewMap([]byte(ev.Buffer))

	lineBounds := func(line int) (begin, end int) {
		begin, end = lm.LineBounds(line)
		// If the file has windows-styles newlines, then we want to account for that
		if ev.Buffer[end-1] == '\r' {
			end--

		}
		return
	}

	// Count the number of lines in the source, discounting any empty newlines at the end
	numLines := lm.LineCount()
	for {
		lastBegin, lastEnd := lineBounds(numLines - 1)
		if lastEnd > lastBegin || numLines == 1 {
			break
		}
		numLines--
	}

	line, col := lm.LineCol(int(ev.Offset))

	// This can happen when the user's cursor is on an empty line at the end of the file and there are more
	// empty lines after the cursor. For the purpose of analysis, we we can count this is as being on the last line.
	if line >= numLines {
		line = numLines - 1
	}

	begin, end := lineBounds(line)
	lineLength := end - begin

	// In windows files, sometimes the cursor is between the \r and \n.
	if col > lineLength {
		col = lineLength
	}

	colPercent := 1.0
	blankLine := true
	if lineLength != 0 {
		colPercent = float64(col) / float64(lineLength)
		blankLine = false
	}

	r := record{
		Length:             int64(len(ev.Buffer)),
		Cursor:             ev.Offset,
		ColPercentOfLine:   colPercent,
		LinePercentOfTotal: float64(line+1) / float64(numLines),
		BlankLine:          blankLine,
	}

	return r
}

func main() {
	args := struct {
		Filename string
	}{
		Filename: "comp-pos-info.json",
	}

	compEvents := pythonpipeline.NewTrackingEvents(
		analyze.NewDate(2018, 12, 12),
		analyze.NewDate(2019, 01, 15),
		pythontracking.ServerCompletionsFailureEvent,
		pythonpipeline.DefaultTrackingEventsOpts)

	outf, err := os.Create(args.Filename)
	if err != nil {
		log.Fatalln(err)
	}
	writer := dependent.NewJSONWriter("writeLogs", outf)

	p := make(pipeline.ParentMap)
	p.Chain(
		compEvents,
		transform.NewFilter("compSampleEvents", isCompSampleEvent),
		transform.NewOneInOneOut("getRecord", getRecord),
		writer)

	pipe := pipeline.Pipeline{
		Name:    "comp-pos-analysis",
		Parents: p,
		Sources: []pipeline.Source{compEvents},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.DefaultEngineOptions)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = engine.Run()
	if err != nil {
		log.Fatalln(err)
	}

	if writer.Errors != 0 {
		log.Fatalf("%d errors encountered when writing to %s", writer.Errors, args.Filename)
	}
	log.Printf("%d record written to %s", writer.Written, args.Filename)
}
