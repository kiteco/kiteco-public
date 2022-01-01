package diff

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/stretchr/testify/assert"
)

func TestDiffThenPatch(t *testing.T) {
	testdata := [][2]string{
		[2]string{"", ""},
		[2]string{"", "a"},
		[2]string{"", "☃"},
		[2]string{"a", ""},
		[2]string{"☃", ""},
		[2]string{"☃", "✈"},
		[2]string{"☃", "✈a"},
		[2]string{"☃", "a✈"},
		[2]string{"☃", "☃✈"},
		[2]string{"☃", "☃a"},
		[2]string{"☃", "✈☃"},
		[2]string{"☃", "a☃"},
		[2]string{"☃✈", "☃"},
		[2]string{"☃a", "☃"},
		[2]string{"✈☃", "☃"},
		[2]string{"a☃", "☃"},
		[2]string{"☃☁☁☁✈", "☃★☁☁☁"},
		[2]string{"xxx ☃☁☁☁✈ yyy", "xxx ☃☁☁☁✈ yyy"},
		[2]string{"xxx ☃☁☁☁✈ yyy", "xxx ☃☁☁☁✈ yyy zzz"},
	}

	for i, pair := range testdata {
		before, after := pair[0], pair[1]
		t.Run(fmt.Sprintf("Pair %d", i), func(t *testing.T) {
			diff := NewDiffer().Diff(before, after)

			// convert from []event.Diff to []*event.Diff
			var diffs []*event.Diff
			for i := range diff {
				diffs = append(diffs, &diff[i])
			}

			patcher := NewPatcher([]byte(before))
			patcher.Apply(diffs)
			assert.Equal(t, after, string(patcher.Bytes()))
		})
	}
}
