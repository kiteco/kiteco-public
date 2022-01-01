package livemetrics

import (
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/visibility"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
)

// visibilityMetrics records whether the sidebar was visible, and when users were coding
type visibilityMetrics struct {
	visibleCodingCount        int
	occludedCodingCount       int
	visiblePythonCodingCount  int
	occludedPythonCodingCount int
	visibleSlackingCount      int
	occludedSlackingCount     int

	hasBeenCoding       bool
	hasBeenPythonCoding bool

	visibleCount int
	hiddenCount  int

	mu sync.Mutex
}

func newVisibilityMetrics() *visibilityMetrics {
	v := &visibilityMetrics{}
	v.zero()
	return v
}

func (v *visibilityMetrics) zero() {
	v.visibleCodingCount = 0
	v.occludedCodingCount = 0
	v.visiblePythonCodingCount = 0
	v.occludedPythonCodingCount = 0
	v.visibleSlackingCount = 0
	v.occludedSlackingCount = 0
	v.hasBeenCoding = false
	v.hasBeenPythonCoding = false
	v.visibleCount = 0
	v.hiddenCount = 0
}

func (v *visibilityMetrics) lockedZero() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.zero()
}

func (v *visibilityMetrics) TrackCoding(filename string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.hasBeenCoding = true
}

func (v *visibilityMetrics) TrackPythonCoding(filename string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.hasBeenPythonCoding = true
}

func (v *visibilityMetrics) TrackVisible() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.hasBeenCoding {
		v.visibleCodingCount++
	} else {
		v.visibleSlackingCount++
	}
	if v.hasBeenPythonCoding {
		v.visiblePythonCodingCount++
	}
	v.hasBeenCoding = false
	v.hasBeenPythonCoding = false

	v.visibleCount++
}

func (v *visibilityMetrics) TrackOccluded() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.hasBeenCoding {
		v.occludedCodingCount++
	} else {
		v.occludedSlackingCount++
	}
	if v.hasBeenPythonCoding {
		v.occludedPythonCodingCount++
	}
	v.hasBeenCoding = false
	v.hasBeenPythonCoding = false

	v.hiddenCount++
}

func (v *visibilityMetrics) Track(visibleNow, visibleRecently bool) {
	if visibleNow {
		v.TrackVisible()
	} else {
		v.TrackOccluded()
	}
}

func (v *visibilityMetrics) get() map[string]int {
	v.mu.Lock()
	defer v.mu.Unlock()
	m := int(visibility.Interval / time.Second)
	n := map[string]int{
		"sidebar_visible_time": v.visibleCount * m,
		"sidebar_hidden_time":  v.hiddenCount * m,
	}
	return n
}

func (v *visibilityMetrics) dump() map[string]int {
	v.mu.Lock()
	defer v.mu.Unlock()
	m := int(visibility.Interval / time.Second)
	n := map[string]int{
		// FIXME these metrics do not work because some editors are not sending
		// events for non-whitelisted files or non-Python files
		"sidebar_visible_coding_time":        v.visibleCodingCount * m,
		"sidebar_hidden_coding_time":         v.occludedCodingCount * m,
		"sidebar_visible_slacking_time":      v.visibleSlackingCount * m,
		"sidebar_hidden_slacking_time":       v.occludedSlackingCount * m,
		"sidebar_visible_python_coding_time": v.visiblePythonCodingCount * m,
		"sidebar_hidden_python_coding_time":  v.occludedPythonCodingCount * m,

		// These work
		"sidebar_visible_time": v.visibleCount * m,
		"sidebar_hidden_time":  v.hiddenCount * m,
	}
	v.zero()
	return n
}

func (v *visibilityMetrics) TrackEvent(evt *event.Event) {
	v.TrackCoding(evt.GetFilename())
	if lang.FromFilename(evt.GetFilename()) == lang.Python && evt.GetSource() != "terminal" {
		v.TrackPythonCoding(evt.GetFilename())
	}
}
