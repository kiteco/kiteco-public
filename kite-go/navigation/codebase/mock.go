package codebase

import (
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type mockRecommender struct {
	files         []recommend.File
	shouldRebuild bool
}

func (r mockRecommender) Recommend(ctx kitectx.Context, request recommend.Request) ([]recommend.File, error) {
	return r.files, nil
}

func (r mockRecommender) RecommendBlocks(ctx kitectx.Context, request recommend.BlockRequest) ([]recommend.File, error) {
	return request.InspectFiles, nil
}

func (r mockRecommender) RankedFiles() ([]recommend.File, error) {
	return nil, nil
}

func (r mockRecommender) ShouldRebuild() (bool, error) {
	return r.shouldRebuild, nil
}

type mockIgnorer struct {
	shouldRebuild bool
}

func (i mockIgnorer) Ignore(pathname localpath.Absolute, isDir bool) bool {
	return false
}

func (i mockIgnorer) ShouldRebuild() (bool, error) {
	return i.shouldRebuild, nil
}
