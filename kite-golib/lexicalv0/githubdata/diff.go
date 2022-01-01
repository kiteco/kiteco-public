package githubdata

import (
	"bytes"
	"errors"

	"github.com/google/go-github/github"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func (e Extractor) extractSamplesFromDiff(pull *github.PullRequest, filename string, edits []diffmatchpatch.Diff) []PredictionSite {
	var sites []PredictionSite
	var group, before []diffmatchpatch.Diff

	for idx, edit := range edits {
		switch edit.Type {
		case diffmatchpatch.DiffEqual:
			if len(group) > 0 {
				after := edits[idx:]
				site, err := e.predictionSiteForGroup(filename, before, after, group, pull)
				if err == nil {
					sites = append(sites, site)
				}
			}
			for _, gedit := range group {
				before = append(before, gedit)
			}
			before = append(before, edit)

			group = nil
		case diffmatchpatch.DiffDelete:
			group = append(group, edit)
		case diffmatchpatch.DiffInsert:
			group = append(group, edit)
		}
	}

	if len(group) > 0 {
		site, err := e.predictionSiteForGroup(filename, before, nil, group, pull)
		if err == nil {
			sites = append(sites, site)
		}
	}

	return sites
}

func (e Extractor) predictionSiteForGroup(fn string, before, after, group []diffmatchpatch.Diff, pull *github.PullRequest) (PredictionSite, error) {
	var (
		srcContextBefore = &bytes.Buffer{}
		srcContextAfter  = &bytes.Buffer{}
		srcWindow        = &bytes.Buffer{}

		dstContextBefore = &bytes.Buffer{}
		dstContextAfter  = &bytes.Buffer{}
		dstWindow        = &bytes.Buffer{}
	)

	accumulateSrcDst := func(edits []diffmatchpatch.Diff, src *bytes.Buffer, dst *bytes.Buffer) {
		for _, edit := range edits {
			switch edit.Type {
			case diffmatchpatch.DiffEqual:
				src.WriteString(edit.Text)
				dst.WriteString(edit.Text)
			case diffmatchpatch.DiffDelete:
				src.WriteString(edit.Text)
			case diffmatchpatch.DiffInsert:
				dst.WriteString(edit.Text)
			}
		}
	}

	accumulateSrcDst(group, srcWindow, dstWindow)

	srcWindowStr, dstWindowStr := srcWindow.String(), dstWindow.String()
	if len(srcWindowStr)+len(dstWindowStr) == 0 {
		return PredictionSite{}, errors.New("skipping, invalid window")
	}

	accumulateSrcDst(before, srcContextBefore, dstContextBefore)
	accumulateSrcDst(after, srcContextAfter, dstContextAfter)

	return PredictionSite{
		PullNumber:       pull.GetNumber(),
		PullTime:         pull.GetClosedAt().Format("20060102150405"),
		FilePath:         fn,
		SrcContextBefore: srcContextBefore.String(),
		SrcContextAfter:  srcContextAfter.String(),
		SrcWindow:        srcWindowStr,
		DstContextBefore: dstContextBefore.String(),
		DstContextAfter:  dstContextAfter.String(),
		DstWindow:        dstWindowStr,
	}, nil
}

func computeDiffs(src, dst string) ([]diffmatchpatch.Diff, error) {
	dmp := diffmatchpatch.New()
	wSrc, wDst, warray := dmp.DiffLinesToRunes(src, dst)
	diffs := dmp.DiffMainRunes(wSrc, wDst, false)
	diffs = dmp.DiffCharsToLines(diffs, warray)
	return diffs, nil
}
