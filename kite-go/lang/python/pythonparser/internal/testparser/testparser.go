// Package testparser implements parser testing utilities common
// to the specialized python parsers.
package testparser

import (
	"encoding/json"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// ParserUpToDate checks if the parser is up-to-date by running the
// default make target as a dry-run and checking if it reported
// that nothing had to be done.
func ParserUpToDate(t *testing.T, pegPath string) {
	// This os.Stat call may seem weird, but it is required because of the
	// cached test results in Go1.10+. Go caches test results if the code doesn't
	// change, so changing the PEG grammar (a .peg file, not a .go file) after
	// running the tests successfully would still return success because of
	// cached results (when it should fail because the generated parser is not
	// up-to-date anymore).
	//
	// The test caching mechanism records the files that are accessed by a test
	// so that those tests are re-run when those external dependencies change.
	// Calling os.Stat makes sure that the caching mechanism is aware that our
	// test relies on this external peg file.
	//
	// See https://golang.org/doc/go1.10#test
	_, err := os.Stat(pegPath)
	require.NoError(t, err)

	c := exec.Command("make", "--dry-run")
	b, err := c.Output()

	require.NoError(t, err)

	msg := string(b)
	require.Regexp(t, "make([^:]+)?\\s*:\\s*Nothing to be done for [`']all[`']", msg, "generated parser not updated, run `make`")
}

// WithDataSet runs the test function fn for each case in the dataset
// file identified by the environment variable KITE_EPYTEXT_DATASET.
// The test is skipped if no file is specified. The test function
// receives the *testing.T, the index of the case in the dataset
// (-1 when a unique key is run), the key value and the source text
// of that key.
//
// The environment variables KITE_EPYTEXT_DATASET_OFFSET and
// KITE_EPYTEXT_DATASET_LIMIT can specify an offset and limit number
// of cases to run in the dataset. If the KITE_EPYTEXT_DATASET_KEY
// environment variable is set, only the case with that specific key
// will be executed.
func WithDataSet(t *testing.T, fn func(*testing.T, int, string, string)) {
	filename := os.Getenv("KITE_EPYTEXT_DATASET")
	if filename == "" {
		t.Skip("no dataset file specified; set environment variable KITE_EPYTEXT_DATASET to run\n\toptions: KITE_EPYTEXT_DATASET_OFFSET=n KITE_EPYTEXT_DATASET_LIMIT=n KITE_EPYTEXT_DATASET_KEY=key")
	}
	offset, _ := strconv.Atoi(os.Getenv("KITE_EPYTEXT_DATASET_OFFSET"))
	limit, _ := strconv.Atoi(os.Getenv("KITE_EPYTEXT_DATASET_LIMIT"))
	uniqueKey := os.Getenv("KITE_EPYTEXT_DATASET_KEY")

	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	var cases map[string]string
	require.NoError(t, json.NewDecoder(f).Decode(&cases))

	if uniqueKey != "" {
		// run that unique key, no need to sort
		src, ok := cases[uniqueKey]
		require.True(t, ok, "specified key does not exist")
		fn(t, -1, uniqueKey, src)
		return
	}

	if offset >= len(cases) {
		t.Fatalf("specified offset of %d is greater than the number of cases (%d)", offset, len(cases))
	}

	// sort the keys so offset and limit are meaningful (otherwise
	// range access to the map is random).
	keys := make([]string, 0, len(cases))
	for k := range cases {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var count int
	for i, key := range keys {
		if i < offset {
			continue
		}
		fn(t, i, key, cases[key])
		count++
		if limit > 0 && count >= limit {
			break
		}
	}
}
