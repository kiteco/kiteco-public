package normalize

import (
	"encoding/csv"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type matchMetrics struct {
	characters   int
	placeholders int
	identifiers  int
	keywords     int
}

func (m matchMetrics) header() []string {
	return []string{
		"match_chars",
		"match_placeholders",
		"match_identifiers",
		"match_keywords",
	}
}

func (m matchMetrics) toStrings() []string {
	return []string{
		strconv.Itoa(m.characters),
		strconv.Itoa(m.placeholders),
		strconv.Itoa(m.identifiers),
		strconv.Itoa(m.keywords),
	}
}

// ProviderDataCollector ...
type ProviderDataCollector struct {
	cursor     string
	limit      int
	dataSet    map[data.ProviderName][]providerData
	ErrorCount int
	m          sync.Mutex
}

// NewProviderDataCollector ...
func NewProviderDataCollector(cursor string, limit int) ProviderDataCollector {
	return ProviderDataCollector{
		cursor:  cursor,
		limit:   limit,
		dataSet: make(map[data.ProviderName][]providerData),
	}
}

type providerData struct {
	sampleID          int
	providerName      data.ProviderName
	score             float64
	experimentalScore float64
	matchMetrics      matchMetrics
}

func (p providerData) header() []string {
	cols := []string{
		"sample_id",
		"provider",
		"score",
		"experimental_score",
	}
	cols = append(cols, p.matchMetrics.header()...)
	return cols
}

func (p providerData) toStrings() []string {
	values := []string{
		strconv.Itoa(p.sampleID),
		strconv.Itoa(int(p.providerName)),
		strconv.FormatFloat(p.score, 'E', 6, 64),
		strconv.FormatFloat(p.experimentalScore, 'E', 6, 64),
	}
	values = append(values, p.matchMetrics.toStrings()...)
	return values
}

// Collect data from providers
func (p *ProviderDataCollector) Collect(provider pythonproviders.Provider, global pythonproviders.Global, code string, sampleID int) {
	buf, before, after := process(code, p.cursor)
	providerName := provider.Name()
	var count int
	ctx := kitectx.Background()
	inputs, err := pythonproviders.NewInputs(ctx, global, buf, false, false)
	if err != nil {
		log.Fatal(err)
	}

	out := func(ctx kitectx.Context, b data.SelectedBuffer, m pythonproviders.MetaCompletion) {
		p.m.Lock()
		defer p.m.Unlock()
		count++
		if count > p.limit {
			return
		}
		matchMetrics, err := match(m.Completion.Snippet, before, after)
		if err != nil {
			p.ErrorCount++
			return
		}
		p.dataSet[providerName] = append(p.dataSet[providerName], providerData{
			providerName:      providerName,
			score:             m.Score,
			experimentalScore: m.ExperimentalScore,
			matchMetrics:      matchMetrics,
			sampleID:          sampleID,
		})
	}

	provider.Provide(ctx, global, inputs, out)
}

// Write data to CSV
func (p *ProviderDataCollector) Write(path string) error {
	time.Sleep(10 * time.Second)
	var providerRows [][]string
	for _, group := range p.dataSet {
		for _, completion := range group {
			providerRows = append(providerRows, completion.toStrings())
		}
	}
	return write(path, providerData{}.header(), providerRows)
}

// APIDataCollector ...
type APIDataCollector struct {
	cursor     string
	topN       int
	dataSet    []apiData
	ErrorCount int
	m          sync.Mutex
}

// NewAPIDataCollector ...
func NewAPIDataCollector(cursor string, topN int) APIDataCollector {
	return APIDataCollector{
		cursor: cursor,
		topN:   topN,
	}
}

type apiData struct {
	sampleID     int
	rank         int
	cohort       string
	matchMetrics matchMetrics
	source       response.EditorCompletionSource
}

func (a apiData) header() []string {
	cols := []string{
		"sample_id",
		"rank",
		"cohort",
		"source",
	}
	cols = append(cols, a.matchMetrics.header()...)
	return cols
}

func (a apiData) toStrings() []string {
	vals := []string{
		strconv.Itoa(a.sampleID),
		strconv.Itoa(a.rank),
		a.cohort,
		string(a.source),
	}
	vals = append(vals, a.matchMetrics.toStrings()...)
	return vals
}

// Collect data from the API
func (a *APIDataCollector) Collect(completer api.API, opts api.CompleteOptions, sampleID int, code, cohort string) {
	buf, before, after := process(code, a.cursor)
	req := data.APIRequest{
		UMF:            data.UMF{Filename: "/sample.py"},
		SelectedBuffer: buf,
	}
	resp := completer.Complete(kitectx.Background(), opts, req, nil, nil)
	for i, completion := range resp.Completions {
		if i >= a.topN {
			break
		}
		matchMetrics, err := match(completion.Snippet, before, after)
		result := apiData{
			sampleID:     sampleID,
			rank:         i,
			matchMetrics: matchMetrics,
			cohort:       cohort,
			source:       completion.Source,
		}
		a.addData(result, err)
	}
}

// Write data to CSV
func (a *APIDataCollector) Write(path string) error {
	var apiRows [][]string
	for _, completion := range a.dataSet {
		apiRows = append(apiRows, completion.toStrings())
	}
	return write(path, apiData{}.header(), apiRows)
}

func (a *APIDataCollector) addData(result apiData, err error) {
	a.m.Lock()
	defer a.m.Unlock()
	if err != nil {
		a.ErrorCount++
		return
	}
	a.dataSet = append(a.dataSet, result)
}

func write(path string, header []string, rows [][]string) error {
	if path == "" {
		return errors.New("path is empty")
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()
	writer.Write(header)
	for _, row := range rows {
		err := writer.Write(row)
		if err != nil {
			return err
		}
	}
	return nil
}
