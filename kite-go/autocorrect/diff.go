package autocorrect

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// TODO: this needs some work to handle insertions/deletions of newlines and multiline changes
func diffs(old, new string) []editorapi.AutocorrectDiff {
	differ := diffmatchpatch.New()

	rawDiffs := differ.DiffMain(old, new, true)

	lmo := linenumber.NewMap([]byte(old))
	lmn := linenumber.NewMap([]byte(new))

	bytesToRunes := stringindex.NewConverter(new)

	calcDiff := func(offset int) editorapi.AutocorrectDiff {
		var diff editorapi.AutocorrectDiff

		// deleted line in old buffer
		lo := lmo.Line(offset)
		lbo, leo := lmo.LineBounds(lo)
		deletedDiffLine := editorapi.AutocorrectDiffLine{
			Line: lo,
			Text: old[lbo:leo],
		}

		// inserted line in new buffer
		ln := lmn.Line(offset)
		lbn, len := lmn.LineBounds(ln)
		insertedDiffLine := editorapi.AutocorrectDiffLine{
			Line: ln,
			Text: new[lbn:len],
		}

		deletedDiffLine.Emphasis, insertedDiffLine.Emphasis = calcEmphasis(deletedDiffLine.Text, insertedDiffLine.Text)

		diff.Deleted = append(diff.Deleted, deletedDiffLine)
		diff.Inserted = append(diff.Inserted, insertedDiffLine)

		diff.NewBufferOffsetBytes = uint64(offset)
		diff.NewBufferOffsetRunes = uint64(bytesToRunes.RunesFromBytes(offset))
		return diff
	}

	var diffs []editorapi.AutocorrectDiff
	var offset int
	for _, rd := range rawDiffs {
		var diff editorapi.AutocorrectDiff
		switch rd.Type {
		case diffmatchpatch.DiffEqual:
			offset += len(rd.Text)
		case diffmatchpatch.DiffDelete:
			diff = calcDiff(offset)
		case diffmatchpatch.DiffInsert:
			diff = calcDiff(offset)

			offset += len(rd.Text)
		}
		if len(diff.Deleted) > 0 || len(diff.Inserted) > 0 {
			diffs = append(diffs, diff)
		}
	}

	return diffs
}

func calcEmphasis(old, new string) ([]editorapi.AutocorrectLineEmphasis, []editorapi.AutocorrectLineEmphasis) {
	if old == new {
		return nil, nil
	}

	// TODO(juan): only supports point corrections
	switch {
	case len(old) < len(new):
		// insertion
		offset := -1
		for i := 0; i < len(old); i++ {
			if old[i] != new[i] {
				offset = i
				break
			}
		}

		// must have been the last character
		if offset < 0 {
			offset = len(new) - 1
		}

		conv := stringindex.NewConverter(new)
		return nil, []editorapi.AutocorrectLineEmphasis{
			editorapi.AutocorrectLineEmphasis{
				StartBytes: uint64(offset),
				StartRunes: uint64(conv.RunesFromBytes(offset)),
				EndBytes:   uint64(offset + 1),
				EndRunes:   uint64(conv.RunesFromBytes(offset + 1)),
			},
		}
	case len(new) < len(old):
		// deletion
		offset := -1
		for i := 0; i < len(new); i++ {
			if old[i] != new[i] {
				offset = i
				break
			}
		}

		// must have been the last character
		if offset < 0 {
			offset = len(old) - 1
		}

		conv := stringindex.NewConverter(old)

		return []editorapi.AutocorrectLineEmphasis{
			editorapi.AutocorrectLineEmphasis{
				StartBytes: uint64(offset),
				StartRunes: uint64(conv.RunesFromBytes(offset)),
				EndBytes:   uint64(offset + 1),
				EndRunes:   uint64(conv.RunesFromBytes(offset + 1)),
			},
		}, nil
	default:
		// substitution
		conv := stringindex.NewConverter(old)
		for i := 0; i < len(new); i++ {
			if old[i] != new[i] {
				emphasis := editorapi.AutocorrectLineEmphasis{
					StartBytes: uint64(i),
					StartRunes: uint64(conv.RunesFromBytes(i)),
					EndBytes:   uint64(i + 1),
					EndRunes:   uint64(conv.RunesFromBytes(i + 1)),
				}
				return []editorapi.AutocorrectLineEmphasis{emphasis}, []editorapi.AutocorrectLineEmphasis{emphasis}
			}
		}

		// bad times
		rollbar.Error(fmt.Errorf("invalid calc emphasis state"), fmt.Sprintf("old: `%s`", old), fmt.Sprintf("new: `%s`", new))
		return nil, nil
	}
}
