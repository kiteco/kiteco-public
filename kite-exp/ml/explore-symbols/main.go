package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

func checkForObjectCanonicalisation() {
	datadeps.Enable()
	opts := pythonresource.SmallOptions
	rm, errc := pythonresource.NewManager(opts)
	<-errc
	result := make(map[string]int)
	errorList := rm.RangeSemiCanonicalSymbols(func(sym pythonresource.Symbol) bool {
		if strings.Contains(sym.Canonical().PathString(), "object") {
			result[sym.Canonical().PathString()] = result[sym.Canonical().PathString()] + 1
		}
		return true
	})
	var errorCount int
	if errorList != nil {
		errorCount = errorList.(errors.Errors).Len()
	}
	errorStrings := make([]string, 0, errorCount)
	if errorList != nil {
		for _, err := range errorList.(errors.Errors).Slice() {
			errorStrings = append(errorStrings, fmt.Sprint(err))
		}
	}

	fmt.Println("Errors : \n", strings.Join(errorStrings, "\n"), "\nNumber of errors : ", errorCount)
	resultSlice := make([]string, 0, len(result))
	for k, c := range result {
		if c > 10 {
			resultSlice = append(resultSlice, k)
		}
	}
	sort.Slice(resultSlice, func(i, j int) bool {
		return result[resultSlice[i]] > result[resultSlice[j]]
	})
	for _, s := range resultSlice {
		fmt.Println(s, " : ", result[s])
	}
}

func main() {
	checkForObjectCanonicalisation()
}
