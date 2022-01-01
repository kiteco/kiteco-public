package livemetrics

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/stretchr/testify/assert"
)

func TestLanguageMetrics(t *testing.T) {
	lm := newLanguageMetrics()
	for l, lt := range lang.LanguageTags {
		if l == lang.Text {
			continue
		}
		lm.TrackEvent(&event.Event{
			Filename: proto.String("file." + lt.Ext),
			Action:   proto.String("edit"),
		})
		lm.TrackEvent(&event.Event{
			Filename: proto.String("file." + lt.Ext),
			Action:   proto.String("selection"),
		})
	}

	d := lm.dump()
	for l, lt := range lang.LanguageTags {
		if l == lang.Text {
			continue
		}
		editKey := lt.Name + "_edit"
		v, found := d[editKey]
		assert.True(t, found, "unable to find edit key for %s", lt.Name)
		assert.Equal(t, uint64(1), v, "expected to find 1 edit event for %s", lt.Name)

		selectKey := lt.Name + "_select"
		v, found = d[selectKey]
		assert.True(t, found, "unable to find select key for %s", lt.Name)
		assert.Equal(t, uint64(1), v, "expected to find 1 select event for %s", lt.Name)

		eventKey := lt.Name + "_events"
		v, found = d[eventKey]
		assert.True(t, found, "unable to find events key for %s", lt.Name)
		assert.Equal(t, uint64(2), v, "expected to find 2 total events for %s", lt.Name)
	}

	// make sure everything is properly zeroed out
	d = lm.dump()
	for l, lt := range lang.LanguageTags {
		if l == lang.Text {
			continue
		}
		editKey := lt.Name + "_edit"
		v, found := d[editKey]
		assert.True(t, found, "unable to find edit key for %s", lt.Name)
		assert.Equal(t, uint64(0), v, "expected to find 0 edit events for %s", lt.Name)

		selectKey := lt.Name + "_select"
		v, found = d[selectKey]
		assert.True(t, found, "unable to find select key for %s", lt.Name)
		assert.Equal(t, uint64(0), v, "expected to find 0 select events for %s", lt.Name)

		eventKey := lt.Name + "_events"
		v, found = d[eventKey]
		assert.True(t, found, "unable to find events key for %s", lt.Name)
		assert.Equal(t, uint64(0), v, "expected to find 0 total events for %s", lt.Name)
	}

}
