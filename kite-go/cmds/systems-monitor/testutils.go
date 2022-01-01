package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// output options for the test function
type testopts struct {
	// set the following to true to print out the values

	include []string // only include metrics with the given strings in the name - ignored when empty
	exclude []string // exclude metrics with the given strings in the name - ignored when empty
}

// test function - useful when adding new nodes/sources/metrics
func test(opts testopts) {
	metrics, err := poll()
	if err != nil {
		log.Printf("error polling: %v", err)
		return
	}

	displayed := 0
	for _, met := range metrics {
		if testSkip(met, opts) {
			continue
		}

		buf, _ := json.MarshalIndent(met, "  ", "")
		fmt.Println(string(buf))
		displayed = displayed + 1
	}
	fmt.Printf("%d metrics shown\n", displayed)
}

// check testopts to see if a metric will be skipped for printing
func testSkip(met *Metric, opts testopts) bool {
	// include list
	if len(opts.include) > 0 {
		for _, word := range opts.include {
			contains := false
			for _, section := range met.Name {
				if strings.Contains(section, word) {
					contains = true
				}
			}
			if !contains {
				return true
			}
		}
	}
	// exclude list
	if len(opts.exclude) > 0 {
		for _, word := range opts.exclude {
			for _, section := range met.Name {
				if strings.Contains(section, word) {
					return true
				}
			}
		}
	}

	return false
}

// format a %+v struct print for slightly easier reading
//
// NOTE: this is only used for testing
func formatStruct(text string) string {
	// tabbed newline for list of structs
	text = strings.Replace(text, ":[{", ":\n\t[{", -1)
	// tabbed newline for nested struct
	text = strings.Replace(text, ":{", ":\n\t{", -1)
	// tabbed newline for list
	text = strings.Replace(text, ":[", ":\n\t[", -1)
	// newline after close brace
	text = strings.Replace(text, "} ", "}\n", -1)
	// newline after close bracket
	text = strings.Replace(text, "] ", "]\n", -1)

	return text
}
