package search

import (
	"errors"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/codesearch"
)

// Results returned by search
type Results struct {
	Files []File
	Pulls []Pull
}

// File ...
type File struct {
	URL   string
	Lines []Line
}

// Line ...
type Line struct {
	Number  int
	Content string
}

// Pull ...
type Pull struct {
	Meta          PullMeta
	CommentGroups []CommentGroup
	FileDiffs     []FileDiff
}

// PullMeta ...
type PullMeta struct {
	URL    string
	Author string
	Title  string
	Body   string
	Number int
}

// CommentGroup ...
type CommentGroup struct {
	Diff     string
	URL      string
	Comments []Comment
}

// Comment ...
type Comment struct {
	Author  string
	Content string
}

// FileDiff ...
type FileDiff struct {
	URL  string
	Diff string
}

// Options ...
type Options struct {
	PullsDir      string
	RegExp        bool
	CaseSensitive bool
}

// Search indexed files for query
func Search(query string, opts Options) (Results, error) {
	if !opts.CaseSensitive {
		query = strings.ToLower(query)
	}
	if !opts.RegExp {
		query = regexp.QuoteMeta(query)
	}
	flags := codesearch.SearchOptions{
		N:     true,
		IFlag: !opts.CaseSensitive,
	}
	outBuf, errBuf := codesearch.Search(flags, query)
	if errBuf.String() != "" {
		return Results{}, errors.New(errBuf.String())
	}

	agg := newAggregator(opts.PullsDir)
	for _, raw := range strings.Split(outBuf.String(), "\n") {
		d, ok, err := process(raw)
		if err != nil {
			return Results{}, err
		}
		if !ok {
			continue
		}
		err = agg.add(d)
		if err != nil {
			return Results{}, err
		}
	}
	aggregated, err := agg.aggregate()
	if err != nil {
		return Results{}, err
	}

	appraiser, err := newAppraiser(query, opts)
	if err != nil {
		return Results{}, err
	}
	appraiser.rankPulls(aggregated.Pulls)
	appraiser.rankFiles(aggregated.Files)
	for _, pull := range aggregated.Pulls {
		appraiser.rankCommentGroups(pull.CommentGroups)
		appraiser.rankFileDiffs(pull.FileDiffs)
		appraiser.sampleFileDiffs(pull.FileDiffs)
	}

	return aggregated, nil
}
