package curation

import (
	"fmt"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/sandbox"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRunManager() *RunManager {
	db := GormDB("sqlite3", ":memory:")
	runs := NewRunManager(db)
	runs.Migrate()
	return runs
}

func TestCreateRun(t *testing.T) {
	m := setupRunManager()

	result := sandbox.Result{}
	agg, err := m.Create(&result, 123, time.Now())
	require.NoError(t, err)
	assert.EqualValues(t, 123, agg.Run.SnippetID)

	run, err := m.Lookup(agg.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, agg.Run.ID, run.ID)
	assert.EqualValues(t, 123, run.SnippetID)
}

func TestLookupLatestForSnippet(t *testing.T) {
	m := setupRunManager()

	result := sandbox.Result{}
	agg, err := m.Create(&result, 123, time.Now())
	require.NoError(t, err)
	assert.EqualValues(t, 123, agg.Run.SnippetID)

	run, err := m.LookupLatestForSnippet(123)
	require.NoError(t, err)
	assert.Equal(t, agg.Run.ID, run.ID)
	assert.EqualValues(t, 123, run.SnippetID)
}

func TestLookupLatestForSnippet_MultipleResults(t *testing.T) {
	m := setupRunManager()

	_, err := m.Create(&sandbox.Result{}, 123, time.Unix(0, 0))
	require.NoError(t, err)

	newAgg, err := m.Create(&sandbox.Result{}, 123, time.Unix(1, 0))
	require.NoError(t, err)

	run, err := m.LookupLatestForSnippet(123)
	require.NoError(t, err)
	assert.Equal(t, newAgg.Run.ID, run.ID)
	assert.EqualValues(t, 123, run.SnippetID)
}

func TestLookupProblems(t *testing.T) {
	m := setupRunManager()

	for i := 0; i < 3; i++ {
		problem := CodeProblem{
			RunID:   1,
			Level:   "info",
			Segment: "prelude",
			Message: fmt.Sprintf("Test code problem %d", i),
			Line:    i,
		}
		err := m.db.Create(&problem).Error
		require.NoError(t, err)
	}

	problems, err := m.LookupProblems(1)
	require.NoError(t, err)
	assert.Equal(t, 3, len(problems))
	for idx, problem := range problems {
		assert.EqualValues(t, 1, problem.RunID)
		assert.Equal(t, fmt.Sprintf("Test code problem %d", idx), problem.Message)
		assert.Equal(t, idx, problem.Line)
	}
}

func TestLookupHTTPOutputs(t *testing.T) {
	m := setupRunManager()

	for i := 0; i < 3; i++ {
		output := HTTPOutput{
			RunID:              1,
			RequestMethod:      "GET",
			RequestURL:         "http://localhost",
			RequestHeaders:     "a:b\nc:d",
			RequestBody:        []byte(fmt.Sprintf("Request body %d", i)),
			ResponseStatus:     "Status OK",
			ResponseStatusCode: 200,
			ResponseHeaders:    "e:f\ng:h",
			ResponseBody:       []byte(fmt.Sprintf("Response body %d", i)),
		}
		err := m.db.Create(&output).Error
		require.NoError(t, err)
	}

	outputs, err := m.LookupHTTPOutputs(1)
	require.NoError(t, err)
	assert.Equal(t, 3, len(outputs))
	for idx, output := range outputs {
		assert.EqualValues(t, 1, output.RunID)
		assert.Equal(t, fmt.Sprintf("Request body %d", idx), string(output.RequestBody))
		assert.Equal(t, fmt.Sprintf("Response body %d", idx), string(output.ResponseBody))
	}
}

func TestLookupOutputFiles(t *testing.T) {
	m := setupRunManager()

	for i := 0; i < 3; i++ {
		output := OutputFile{
			RunID:       1,
			Path:        fmt.Sprintf("Test path %d", i),
			ContentType: "text",
			Contents:    []byte(fmt.Sprintf("Output %d", i)),
		}
		err := m.db.Create(&output).Error
		require.NoError(t, err)
	}

	outputs, err := m.LookupOutputFiles(1)
	require.NoError(t, err)
	assert.Equal(t, 3, len(outputs))
	for idx, output := range outputs {
		assert.EqualValues(t, 1, output.RunID)
		assert.Equal(t, fmt.Sprintf("Test path %d", idx), string(output.Path))
		assert.Equal(t, fmt.Sprintf("Output %d", idx), string(output.Contents))
	}
}
