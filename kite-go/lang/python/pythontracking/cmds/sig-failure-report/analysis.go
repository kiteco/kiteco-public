package main

import (
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/localfiles/offlineconf"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

type analyzer struct {
	recreator *servercontext.Recreator
}

func newAnalyzer(recreator *servercontext.Recreator) analyzer {
	return analyzer{
		recreator: recreator,
	}
}

// Analyzed is a struct that is written to json for post-analysis.
type Analyzed struct {
	MessageID              string                 `json:"message_id"`
	Timestamp              time.Time              `json:"timestamp"`
	Failure                string                 `json:"failure"`
	ReproducedFailure      string                 `json:"reproduced_failure"`
	UserID                 int64                  `json:"user_id"`
	Offset                 int64                  `json:"offset"`
	BufferSize             int                    `json:"buffer_size"`
	IndexFileCount         int                    `json:"index_file_count"`
	IndexValueCount        int                    `json:"index_value_count"`
	MissingIndexedFiles    int                    `json:"missing_indexed_files"`   // Indexed files unreadable when log was created
	RecreatedMissingFiles  int                    `json:"recreated_missing_files"` // Indexed files unreadable during analysis
	Platform               string                 `json:"platform"`                // osx, windows
	Region                 string                 `json:"region"`
	FuncType               string                 `json:"func_type"` // type of relevant CallExpr's func
	Function               string                 `json:"function"`  // buffer's contents that correspond to the CallExpr's func
	Category               *category              `json:"category"`
	CalleeID               string                 `json:"callee_id"`
	UnresolvedFileAnalysis UnresolvedFileAnalysis `json:"unresolved_file_analysis"`
}

func (a analyzer) analyze(metadata analyze.Metadata, track *pythontracking.Event) (*Analyzed, error) {
	ctx, err := a.recreator.RecreateContext(track, true)
	if err != nil {
		return nil, err
	}

	platform := "osx"
	if strings.Contains(track.Filename, "/windows") {
		platform = "windows"
	}

	var indexValueCount int
	var recreatedMissingFiles int
	if ctx.LocalIndex != nil {
		indexValueCount = len(ctx.LocalIndex.ValuesCount)
		recreatedMissingFiles = len(ctx.LocalIndex.ArtifactMetadata.MissingHashes)
	}

	in := python.NewCalleeInputs(ctx, int64(track.Offset), a.recreator.Services)

	result := python.GetCallee(kitectx.Background(), in)

	analyzed := Analyzed{
		MessageID:             metadata.ID.String(),
		Timestamp:             metadata.Timestamp,
		Failure:               string(track.Failure()),
		ReproducedFailure:     string(result.Failure),
		UserID:                track.UserID,
		Offset:                track.Offset,
		BufferSize:            len(track.Buffer),
		IndexFileCount:        len(track.ArtifactMeta.FileHashes),
		IndexValueCount:       indexValueCount,
		MissingIndexedFiles:   len(track.ArtifactMeta.MissingHashes),
		RecreatedMissingFiles: recreatedMissingFiles,
		Platform:              platform,
		Region:                track.Region,
	}

	if result.CallExpr != nil {
		funcExpr := result.CallExpr.Func
		analyzed.FuncType = reflect.TypeOf(funcExpr).String()
		analyzed.Function = track.Buffer[funcExpr.Begin():funcExpr.End()]
	}

	if result.Failure == pythontracking.UnresolvedValueFailure {
		analyzed.Category = a.categorize(metadata.ID, track, ctx)
	}

	if result.Response != nil && result.Response.Callee != nil {
		analyzed.CalleeID = result.Response.Callee.ID.String()
	}

	if manager := offlineconf.GetFileManager(track.Region); manager != nil {
		unresolvedFileAnalysis, err := doUnresolvedFileAnalysis(track, metadata.Timestamp, ctx, manager)
		if err != nil {
			log.Printf("error in performing unresolved import analysis: %v", err)
		} else {
			analyzed.UnresolvedFileAnalysis = unresolvedFileAnalysis
		}
	}

	return &analyzed, nil
}

// getValueName returns a name and a true iff val is a global value
func getValueName(val pythontype.Value, graph pythonresource.Manager) (string, bool) {
	val = pythontype.TranslateNoCtx(val, graph)
	if val == nil {
		return "", false
	}
	if val, ok := val.(pythontype.GlobalValue); ok {
		switch val := val.(type) {
		case pythontype.ExternalInstance:
			return val.TypeExternal.Symbol().Canonical().String(), true
		case pythontype.External:
			return val.Symbol().Canonical().String(), true
		case pythontype.ExternalRoot:
			return "", true
		default:
			panic("unhandled GlobalValue type")
		}
	}
	return val.Address().Path.String(), false
}
