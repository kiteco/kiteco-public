package render

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

func TestTransformSelectionWithPrettifiedMappings(t *testing.T) {
	cases := []struct {
		sel      data.Selection
		mappings []OffsetMapping
		match    MatchOption
		want     data.Selection
	}{
		{
			data.Selection{Begin: 0, End: 1},
			nil,
			MatchEnd,
			data.Selection{Begin: -1, End: -1},
		},
		{
			data.Selection{Begin: 0, End: 2},
			[]OffsetMapping{{StartBefore: 0, StartAfter: 1, EndBefore: 2, EndAfter: 3}},
			MatchStart,
			data.Selection{Begin: 1, End: 3},
		},
		{
			// actual transformation case from TestJavascript_NoSyntaxEndingInParen
			data.Selection{Begin: 114, End: 122},
			[]OffsetMapping{
				{105, 104, 110, 109},
				{110, 109, 111, 110},
				{112, 111, 113, 112},
				{114, 113, 122, 121},
				{123, 122, 124, 123},
				{127, 126, 128, 127},
				{129, 128, 130, 129},
				{130, 129, 131, 130},
			},
			MatchStart,
			data.Selection{Begin: 113, End: 121},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			gotBegin := transformPosition(c.sel.Begin, c.match, c.mappings)
			gotEnd := transformPosition(c.sel.End, false, c.mappings)
			got := data.Selection{Begin: gotBegin, End: gotEnd}
			if got != c.want {
				t.Fatalf("want %v, got %v", c.want, got)
			}
		})
	}
}
