package predict

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

func toInt64(a []int) []int64 {
	var b []int64
	if len(a) > 0 {
		b = make([]int64, 0, len(a))
	}
	for _, e := range a {
		b = append(b, int64(e))
	}
	return b
}

func toInt(a []int64) []int {
	var b []int
	if len(a) > 0 {
		b = make([]int, 0, len(a))
	}
	for _, e := range a {
		b = append(b, int(e))
	}
	return b
}

func toInt2d(a [][]int64) [][]int {
	var b [][]int
	for _, aa := range a {
		b = append(b, toInt(aa))
	}
	return b
}

// Print gives a string representation of the predicted results
func Print(f *lexicalv0.FileEncoder, preds []Predicted) string {
	var s string
	for _, p := range preds {
		s += fmt.Sprintf("|%s|  (%.5f)\n", f.DecodeToStrings(p.TokenIDs), p.Prob)
	}
	return s
}

func copyAndAppend(base []int64, ext []int64) []int64 {
	cpy := make([]int64, 0, len(base)+len(ext))
	cpy = append(cpy, base...)
	cpy = append(cpy, ext...)
	return cpy
}

func idsMatchingPrefixSlice(enc *lexicalv0.FileEncoder, prefix string) []int64 {
	prefixLower := strings.ToLower(prefix)
	var validIDs []int64
	for i, str := range enc.IDToStringLower {
		if wordMatchesPrefixLower(str, prefixLower) {
			validIDs = append(validIDs, int64(i))
		}
	}
	return validIDs
}

func idsMatchingPrefix(enc *lexicalv0.FileEncoder, prefix string) map[int]bool {
	validIds := make(map[int]bool)
	prefixLower := strings.ToLower(prefix)
	for i, str := range enc.IDToStringLower {
		if wordMatchesPrefixLower(str, prefixLower) {
			validIds[i] = true
		}
	}
	return validIds
}

func wordMatchesPrefixLower(word, prefixLower string) bool {
	return strings.HasPrefix(word, prefixLower) || strings.HasPrefix(prefixLower, word)
}

func toStrings(ctx []int64) []string {
	strs := make([]string, 0, len(ctx))
	for _, c := range ctx {
		strs = append(strs, strconv.FormatInt(c, 10))
	}
	return strs
}

func handlePredictChanInitErr(err error) (chan Predicted, chan error) {
	errChan := make(chan error, 1)
	errChan <- err

	// nil channel blocks forever so we allocate a channel then close it immediately
	preds := make(chan Predicted)
	close(preds)

	return preds, errChan
}
