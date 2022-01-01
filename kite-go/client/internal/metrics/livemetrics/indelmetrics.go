package livemetrics

// # Overview
//
// These metrics were intended to help estimate coding speed as a proxy for productivity.
// Comments are included in the measuring of coding speed.
//
// Two major considerations were the effects of whitespace and pasted code on the final metric.
// The former is simple, the latter non-trivial. For pasted code, a heuristic is used.
// The metrics are additive/nondestructive in the event the heuristic does not yield sensible data.
//
// # Pasted Code Heuristic
//
// Multi-line insertions indicates pasted code or multi-line completions,
// the latter of which we do not return as of 2020/03/16.
// Multi-line deletions indicates deletion from pasted code, a heuristic that
// has strong limitations. For example a user might decide to refactor a
// manually typed block of code by selecting and deleting.
// See issue 9960 for more details and considerations during conception.
//
// # Naming
//
// The metric keys take the form `indel_<language>_<indelMetricType>_<diffType>`
// e.g. `indel_go_total_inserts`, `indel_javascript_multiline_deletes`,
// or `indel_python_occurences_difftype_none`
//
// For quick implementation, this file also tracks occurences of
// edit events without diffs in `indel_<language>_occurences_no_diffs`

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
)

const indelPrefix = "indel_"

type indelMetricType string

const (
	total               indelMetricType = "total"
	whitespace          indelMetricType = "whitespace"
	multiline           indelMetricType = "multiline"
	multilineWhitespace indelMetricType = "multiline_whitespace"
	occurences          indelMetricType = "occurences_difftype"
)

// indelMetric mainly tracks DiffTypes INSERT and DELETE.
// For DiffType_NONE, it separately tracks occurences.
type indelMetric struct {
	m indelMetricType
	d event.DiffType
}

// Substring of final key into metrics output map.
func (im indelMetric) String() string {
	if im.d == event.DiffType_NONE {
		return strings.ToLower(fmt.Sprintf("%s_%s", im.m, im.d))
	}
	return strings.ToLower(fmt.Sprintf("%s_%ss", im.m, im.d))
}

type indelStore map[indelMetric]uint64

func newIndelStore() indelStore {
	return make(map[indelMetric]uint64)
}

func (is indelStore) flatten(prefix string, out map[string]interface{}) {
	write := func(k string, val interface{}) {
		if val, ok := val.(uint64); ok && val == 0 {
			// save space by not writing 0 values
			return
		}
		out[prefix+"_"+k] = val
	}

	for met, val := range is {
		write(met.String(), val)
	}
}

// For debug printing only; map iteration is not reproducible
func (is indelStore) String() string {
	var kvs []string
	for k, v := range is {
		kvs = append(kvs, fmt.Sprintf("%v: %v", k, v))
	}
	return strings.Join(kvs, "\n")
}

type indelMetrics struct {
	sync.Mutex
	store   indelStore
	noDiffs int
}

func newIndelMetrics() *indelMetrics {
	return &indelMetrics{store: newIndelStore()}
}

func (im *indelMetrics) read(clear bool) (indelStore, int) {
	im.Lock()
	defer im.Unlock()

	out := newIndelStore()
	nd := im.noDiffs

	if clear {
		im.store, out = out, im.store
		im.noDiffs = 0
	} else {
		for k, v := range im.store {
			out[k] = v
		}
	}

	return out, nd
}

func (im *indelMetrics) update(diffs []*event.Diff) {
	if len(diffs) == 0 {
		im.Lock()
		im.noDiffs++
		im.Unlock()
	}

	for _, d := range diffs {
		text := d.GetText()
		rc := uint64(utf8.RuneCountInString(text))
		et := d.GetType()

		im.Lock()

		switch et {
		case event.DiffType_NONE:
			im.store[indelMetric{occurences, et}]++
		case event.DiffType_INSERT, event.DiffType_DELETE:
			wsc := whitespaceCountInString(text)
			txtIsMultLin := isMultiline(text, rc)

			im.store[indelMetric{total, et}] += rc
			im.store[indelMetric{whitespace, et}] += wsc

			if txtIsMultLin {
				im.store[indelMetric{multiline, et}] += rc
				im.store[indelMetric{multilineWhitespace, et}] += wsc
			}
		}

		im.Unlock()
	}
}

type indelMetricsByLang struct {
	sync.Map
}

func newIndelMetricsByLang() *indelMetricsByLang {
	return &indelMetricsByLang{}
}

func (im *indelMetricsByLang) get(l lang.Language) *indelMetrics {
	if im == nil {
		return nil
	}

	shard, ok := im.Load(l)
	if !ok {
		m := newIndelMetrics()
		shard, _ = im.LoadOrStore(l, m)
	}
	return shard.(*indelMetrics)
}

// ReadAndFlatten metrics to send
func (im *indelMetricsByLang) readAndFlatten(clear bool, out map[string]interface{}) map[string]interface{} {
	if out == nil {
		out = make(map[string]interface{})
	}
	im.Range(func(k, v interface{}) bool {
		l := k.(lang.Language)
		m := v.(*indelMetrics)

		store, noDiffs := m.read(clear)
		store.flatten(indelPrefix+l.Name(), out)
		out[indelPrefix+l.Name()+"_occurences_no_diffs"] = noDiffs

		return true
	})
	return out
}

/* String Utils */

var (
	nonWhitespaceRegex = regexp.MustCompile(`\S`)
	whitespaceRegex    = regexp.MustCompile(`\s`)
)

// Pressing 'enter' to create a newline and indentation will not be counted
// as multiline for the purposes of determining pasted code.
func isMultiline(s string, runeCount uint64) bool {
	return runeCount > 1 && hasLineBreak(s) && hasNonWhitespace(s)
}

func whitespaceCountInString(s string) uint64 {
	var wsCount uint64
	for _, r := range s {
		c := string(r)
		if whitespaceRegex.MatchString(c) {
			wsCount++
		}
	}
	return wsCount
}

func hasNonWhitespace(s string) bool {
	return nonWhitespaceRegex.FindStringIndex(s) != nil
}

func hasLineBreak(s string) bool {
	return strings.ContainsAny(s, "\n\r")
}
