package curation

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/sandbox"
)

// RunManager is responsible for executing code examples, storing the results in the database,
// and retrieving results of runs previously stored in the database.
type RunManager struct {
	db gorm.DB
}

// NewRunManager creates a new run manager using the provided dbmap
func NewRunManager(db gorm.DB) *RunManager {
	return &RunManager{db: db}
}

// Migrate will auto-migrate relevant tables in the db.
func (m *RunManager) Migrate() error {
	if err := m.db.AutoMigrate(&Run{}, &HTTPOutput{}, &OutputFile{}, &CodeProblem{}).Error; err != nil {
		return fmt.Errorf("error creating tables in db: %v", err)
	}
	return nil
}

func headerString(headers http.Header) string {
	var lines []string
	for name, values := range headers {
		lines = append(lines, name+": "+strings.Join(values, ", "))
	}
	return strings.Join(lines, "\n")
}

// Create inserts records into the database for the given execution result
func (m *RunManager) Create(result *sandbox.Result, snippetID int64, timestamp time.Time) (*ExecutionResult, error) {
	// Insert the run
	run := Run{
		SnippetID: snippetID,
		Timestamp: timestamp.Unix(),
		Stdin:     result.Stdin,
		Stdout:    result.Stdout,
		Stderr:    result.Stderr,
		Succeeded: result.Succeeded,
	}

	if result.SandboxError != nil {
		switch err := result.SandboxError.(type) {
		case *sandbox.UncleanExit:
			run.SandboxError = "Process exited with non-zero exit status"
		case *sandbox.TimeLimitExceeded:
			run.SandboxError = "Time limit exceeded"
		case *sandbox.OutputLimitExceeded:
			run.SandboxError = "Output limit exceeded for " + err.Stream
		default:
			run.SandboxError = err.Error()
		}
	}

	if err := m.db.Create(&run).Error; err != nil {
		return nil, fmt.Errorf("error inserting run: %v", err)
	}

	agg := &ExecutionResult{
		Run: &run,
	}

	// Insert the output files
	for _, output := range result.OutputFiles {
		contenttype := mime.TypeByExtension(filepath.Ext(output.Path))
		outputfile := OutputFile{
			RunID:       run.ID,
			Path:        output.Path,
			ContentType: contenttype,
			Contents:    output.Contents,
		}
		agg.OutputFiles = append(agg.OutputFiles, &outputfile)
		if err := m.db.Create(&outputfile).Error; err != nil {
			return nil, fmt.Errorf("error inserting output file: %v", err)
		}
	}

	// Insert the HTTP outputs
	for _, httpOutput := range result.HTTPOutputs {
		if httpOutput.Response == nil {
			continue
		}
		httpoutput := HTTPOutput{
			RunID:              run.ID,
			RequestMethod:      httpOutput.Request.Method,
			RequestURL:         httpOutput.Request.URL.String(),
			RequestHeaders:     headerString(httpOutput.Request.Header),
			RequestBody:        httpOutput.RequestBody,
			ResponseStatus:     httpOutput.Response.Status,
			ResponseStatusCode: httpOutput.Response.StatusCode,
			ResponseHeaders:    headerString(httpOutput.Response.Header),
			ResponseBody:       httpOutput.ResponseBody,
		}
		agg.HTTPOutputs = append(agg.HTTPOutputs, &httpoutput)
		if err := m.db.Create(&httpoutput).Error; err != nil {
			return nil, fmt.Errorf("error inserting http outputs: %v", err)
		}
	}

	return agg, nil
}

// Lookup gets the run with the given ID
func (m *RunManager) Lookup(id int64) (*Run, error) {
	var run Run
	err := m.db.First(&run, id).Error
	return &run, err
}

// LookupLatestForSnippet gets the most recent run for the given snippetID, or nil
// if there are no runs for that snippet.
func (m *RunManager) LookupLatestForSnippet(snippetID int64) (*Run, error) {
	var run Run
	err := m.db.Where("SnippetID=?", snippetID).Order("Timestamp DESC").Limit(1).Find(&run).Error
	if err != nil {
		if err == gorm.RecordNotFound {
			// There are no runs for this snippet (this is not an error)
			err = nil
		}
		return nil, err
	}
	return &run, err
}

// LookupProblems gets the problems associated with the given run ID
func (m *RunManager) LookupProblems(runID int64) ([]*CodeProblem, error) {
	problems := []*CodeProblem{}
	err := m.db.Where("RunID=?", runID).Find(&problems).Error
	return problems, err
}

// LookupHTTPOutputs gets the HTTP outputs associated with the given run ID
func (m *RunManager) LookupHTTPOutputs(runID int64) ([]*HTTPOutput, error) {
	httpOutputs := []*HTTPOutput{}
	err := m.db.Where("RunID=?", runID).Find(&httpOutputs).Error
	return httpOutputs, err
}

// LookupOutputFiles gets the output files associated with the given run ID
func (m *RunManager) LookupOutputFiles(runID int64) ([]*OutputFile, error) {
	outputFiles := []*OutputFile{}
	err := m.db.Where("RunID=?", runID).Find(&outputFiles).Error
	return outputFiles, err
}

// LookupAggregate gets all records associated with a run
func (m *RunManager) LookupAggregate(id int64) (*ExecutionResult, error) {
	var err error
	var agg ExecutionResult

	agg.Run, err = m.Lookup(id)
	if err != nil {
		return nil, err
	}

	agg.Problems, err = m.LookupProblems(agg.Run.ID)
	if err != nil {
		return nil, err
	}

	agg.HTTPOutputs, err = m.LookupHTTPOutputs(agg.Run.ID)
	if err != nil {
		return nil, err
	}

	agg.OutputFiles, err = m.LookupOutputFiles(agg.Run.ID)
	if err != nil {
		return nil, err
	}

	return &agg, nil
}

// LookupLatestForSnippetAggregate gets all records associated with the latest run for a given snippet ID
func (m *RunManager) LookupLatestForSnippetAggregate(snippetID int64) (*ExecutionResult, error) {
	var err error
	var agg ExecutionResult

	agg.Run, err = m.LookupLatestForSnippet(snippetID)
	if err != nil {
		return nil, err
	}
	if agg.Run == nil {
		// There are no runs for this snippet (this is not an error)
		return nil, nil
	}

	agg.Problems, err = m.LookupProblems(agg.Run.ID)
	if err != nil {
		return nil, err
	}

	agg.HTTPOutputs, err = m.LookupHTTPOutputs(agg.Run.ID)
	if err != nil {
		return nil, err
	}

	agg.OutputFiles, err = m.LookupOutputFiles(agg.Run.ID)
	if err != nil {
		return nil, err
	}

	return &agg, nil
}
