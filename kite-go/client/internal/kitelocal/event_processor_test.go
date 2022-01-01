package kitelocal

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/permissions"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
	"github.com/stretchr/testify/require"
)

var (
	testpath      string
	testwhitelist string
	badpath       string
)

func init() {
	if runtime.GOOS == "windows" {
		//lowercase because these have to be sanitized paths
		testpath = `c:\src.py`
		testwhitelist = `c:\`
		badpath = `c:\src.bad`
	} else {
		testpath = `/src.py`
		testwhitelist = `/`
		badpath = `/src.bad`
	}
}

func testpathN(n int) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`C:\src%d.py`, n)
	}
	return fmt.Sprintf(`/src%d.py`, n)
}

// --

func TestUnsavedFile(t *testing.T) {
	_, processor := newTestEventProcessor()
	for i := 0; i < 10; i++ {
		event, err := processor.processEvent(&component.EditorEvent{})
		require.Error(t, err)
		require.Nil(t, event)
		require.Equal(t, errUnsavedFile, err)
	}
}

func TestUnsupportedFile(t *testing.T) {
	_, processor := newTestEventProcessor()
	for i := 0; i < 10; i++ {
		event, err := processor.processEvent(&component.EditorEvent{
			Filename: badpath,
		})
		require.Error(t, err)
		require.Nil(t, event)
		require.Equal(t, errUnsupportedFile, err)
	}
}

func TestSkipEvent(t *testing.T) {
	_, processor := newTestEventProcessor()
	for i := 0; i < 10; i++ {
		event, err := processor.processEvent(&component.EditorEvent{
			Action:   "skip",
			Filename: testpath,
		})
		require.Error(t, err)
		require.Nil(t, event)
		require.Equal(t, errSkipped, err)
	}
}

func TestFileTooLarge(t *testing.T) {
	_, processor := newTestEventProcessor()
	for i := 0; i < 10; i++ {
		event, err := processor.processEvent(&component.EditorEvent{
			Filename: testpath,
			Text:     strings.Repeat("d", processor.maxFileSizeBytes()),
		})
		require.Error(t, err)
		require.Nil(t, event)
		require.Equal(t, errFileTooLarge, err)
	}
}

func TestLostFocus(t *testing.T) {
	_, processor := newTestEventProcessor()
	for i := 0; i < 10; i++ {
		event, err := processor.processEvent(&component.EditorEvent{
			Action:   "lost_focus",
			Filename: testpath,
		})
		require.Error(t, err)
		require.Nil(t, event)
		require.Equal(t, errUnused, err)
	}
}

func TestDuplicateProcessing(t *testing.T) {
	_, processor := newTestEventProcessor()

	text := strings.Repeat("d", 10)
	for i := 0; i < 10; i++ {
		event, err := processor.processEvent(&component.EditorEvent{
			Action:   "edit",
			Filename: testpath,
			Source:   "atom",
			Text:     text,
		})

		if i == 0 {
			require.NoError(t, err)
			require.Equal(t, text, event.GetText())
			require.Equal(t, "edit", event.GetAction())
		} else {
			require.Error(t, err)
			require.Nil(t, event)
			require.Equal(t, errDuplicate, err)
		}

		// Set response so we stop sending full text
		processor.updateLatestResponse("atom", testpath, text, event.GetSelections(), false)
	}
}

func TestFocus(t *testing.T) {
	_, processor := newTestEventProcessor()
	for i := 0; i < 10; i++ {
		text := strings.Repeat("d", i+1)
		event, err := processor.processEvent(&component.EditorEvent{
			Source:   "atom",
			Action:   "focus",
			Filename: testpath,
			Text:     text,
		})

		require.NoError(t, err)
		if i == 0 {
			// The first focus will become an edit because we haven't seen this before
			require.Equal(t, "edit", event.GetAction())
		} else {
			// Subsequent focus events should force a focus event with full text
			require.Equal(t, "focus", event.GetAction())
		}
		require.Equal(t, text, event.GetText())
		require.Empty(t, event.GetDiffs())
		require.Empty(t, event.GetReferenceState())

		// Set response
		processor.updateLatestResponse("atom", testpath, text, event.GetSelections(), false)
	}
}

func TestSelectionChanged(t *testing.T) {
	_, processor := newTestEventProcessor()

	text := strings.Repeat("d", 10)
	hash := textMD5(text)
	checksum := textChecksum(text)

	for i := 0; i < 10; i++ {
		event, err := processor.processEvent(&component.EditorEvent{
			Action:   "edit",
			Source:   "atom",
			Filename: testpath,
			Text:     text,
			Selections: []*component.Selection{
				{Start: int64(i + 1), End: int64(i + 1)},
			},
		})

		require.NoError(t, err)

		if i == 0 {
			require.Equal(t, text, event.GetText())
			require.Equal(t, "edit", event.GetAction())
			require.Empty(t, event.GetReferenceState())
		} else {
			require.Equal(t, "selection", event.GetAction())
			require.Equal(t, text, event.GetText())
			require.Equal(t, hash, event.GetReferenceState())
		}

		require.Equal(t, hash, event.GetTextMD5())
		require.Equal(t, checksum, event.GetTextChecksum())
		require.Empty(t, event.GetDiffs())

		// Set response so we stop sending full text
		processor.updateLatestResponse("atom", testpath, text, event.GetSelections(), false)
	}
}

func TestUnicodeSelection(t *testing.T) {
	_, processor := newTestEventProcessor()

	// 1, 2, 3, and 4 utf-8 bytes, respectively
	// 1, 1, 1, and 2 utf-16 code units, respectively
	text := "$£ई𠜎"

	byteCnt := int64(0)
	idx := int64(0)
	for range text {
		idx++
		byteCnt += idx

		event, err := processor.processEvent(&component.EditorEvent{
			Action:   "edit",
			Source:   "atom",
			Filename: testpath,
			Text:     text,
			Selections: []*component.Selection{
				{Start: idx, End: idx, Encoding: stringindex.UTF32},
			},
		})
		require.NoError(t, err)
		require.Len(t, event.Selections, 1)
		require.Equal(t, byteCnt, *event.Selections[0].Start)
	}

	event, err := processor.processEvent(&component.EditorEvent{
		Action:   "edit",
		Source:   "atom",
		Filename: testpath,
		Text:     text,
		Selections: []*component.Selection{
			{Start: 6, End: 6, Encoding: stringindex.UTF8},
		},
	})
	require.NoError(t, err)
	require.Len(t, event.Selections, 1)
	require.Equal(t, int64(6), *event.Selections[0].Start)

	event, err = processor.processEvent(&component.EditorEvent{
		Action:   "edit",
		Source:   "atom",
		Filename: testpath,
		Text:     text,
		Selections: []*component.Selection{
			{Start: 5, End: 5, Encoding: stringindex.UTF16},
		},
	})
	require.NoError(t, err)
	require.Len(t, event.Selections, 1)
	require.Equal(t, int64(10), *event.Selections[0].Start)

	event, err = processor.processEvent(&component.EditorEvent{
		Action:   "edit",
		Source:   "atom",
		Filename: testpath,
		Text:     text,
		Selections: []*component.Selection{
			{Start: -1, End: 5, Encoding: stringindex.UTF16},
		},
	})
	require.Error(t, err)

	event, err = processor.processEvent(&component.EditorEvent{
		Action:   "edit",
		Source:   "atom",
		Filename: testpath,
		Text:     text,
		Selections: []*component.Selection{
			{Start: 5, End: 10, Encoding: stringindex.UTF16},
		},
	})
	require.Error(t, err)

	// Set response so we stop sending full text
	processor.updateLatestResponse("atom", testpath, text, event.GetSelections(), false)
}

func TestResend(t *testing.T) {
	var ref string
	var resend bool
	_, processor := newTestEventProcessor()
	for i := 0; i < 100; i++ {
		text := strings.Repeat("d", i+1)
		hash := textMD5(text)
		checksum := textChecksum(text)

		// Process the event
		event, err := processor.processEvent(&component.EditorEvent{
			Action:   "edit",
			Filename: testpath,
			Source:   "atom",
			Text:     strings.Repeat("d", i+1),
		})

		// Check checksums
		require.NoError(t, err)
		require.Equal(t, hash, event.GetTextMD5())
		require.Equal(t, checksum, event.GetTextChecksum())
		require.Equal(t, "edit", event.GetAction())

		if i == 0 || resend {
			require.Equal(t, text, event.GetText())
			require.Empty(t, event.GetReferenceState())
			require.Empty(t, event.GetDiffs())
		} else {
			require.Equal(t, text, event.GetText())
			require.Equal(t, ref, event.GetReferenceState())
			require.NotEmpty(t, event.GetDiffs())
		}

		// Randomly decide whether the "backend" requests a resend.
		ref = hash
		resend = rand.Intn(10) < 5
		processor.updateLatestResponse("atom", testpath, text, event.GetSelections(), resend)
	}
}

func TestReference(t *testing.T) {
	var ref string
	var gotResponse bool
	_, processor := newTestEventProcessor()
	for i := 0; i < 100; i++ {
		text := strings.Repeat("d", i+1)
		hash := textMD5(text)
		checksum := textChecksum(text)

		// Process the event
		event, err := processor.processEvent(&component.EditorEvent{
			Action:   "edit",
			Filename: testpath,
			Source:   "atom",
			Text:     strings.Repeat("d", i+1),
		})

		// Check checksums
		require.NoError(t, err)
		require.Equal(t, hash, event.GetTextMD5())
		require.Equal(t, checksum, event.GetTextChecksum())
		require.Equal(t, "edit", event.GetAction())

		if i == 0 {
			// Initial event requires full text
			require.Equal(t, text, event.GetText())
			require.Empty(t, event.GetReferenceState())
			require.Empty(t, event.GetDiffs())
		} else if gotResponse {
			// If we've received a response then we can start using reference states
			require.Equal(t, text, event.GetText())
			require.NotEmpty(t, event.GetDiffs())
			require.Equal(t, ref, event.GetReferenceState())
		} else {
			// If we have not received a response, we must continue to send full text
			require.Equal(t, text, event.GetText())
			require.Empty(t, event.GetReferenceState())
			require.Empty(t, event.GetDiffs())
		}

		// Randomly decide whether the backend sends a response, updating the reference state
		updateref := rand.Intn(10) < 5
		gotResponse = gotResponse || updateref
		if updateref {
			processor.updateLatestResponse("atom", testpath, text, event.GetSelections(), false)
			ref = hash
		}
	}
}

func TestReferenceConcurrent(t *testing.T) {
	_, processor := newTestEventProcessor()

	var wg sync.WaitGroup
	wg.Add(10)
	for fn := 0; fn < 10; fn++ {
		filepath := testpathN(fn)
		go func(wg *sync.WaitGroup, fn string) {
			defer wg.Done()

			var ref string
			var gotResponse bool
			for i := 0; i < 100; i++ {
				text := strings.Repeat("d", i+1)
				hash := textMD5(text)
				checksum := textChecksum(text)

				// Process the event
				event, err := processor.processEvent(&component.EditorEvent{
					Action:   "edit",
					Filename: fn,
					Source:   "atom",
					Text:     strings.Repeat("d", i+1),
				})

				// Check checksums
				require.NoError(t, err)
				require.Equal(t, hash, event.GetTextMD5())
				require.Equal(t, checksum, event.GetTextChecksum())
				require.Equal(t, "edit", event.GetAction())

				if i == 0 {
					// Initial event requires full text
					require.Equal(t, text, event.GetText())
					require.Empty(t, event.GetReferenceState())
					require.Empty(t, event.GetDiffs())
				} else if gotResponse {
					// If we've received a response then we can start using reference states
					require.Equal(t, text, event.GetText())
					require.NotEmpty(t, event.GetDiffs())
					require.Equal(t, ref, event.GetReferenceState())
				} else {
					// If we have not received a response, we must continue to send full text
					// Indel metrics depends on an empty diffs array when full text is sent
					require.Equal(t, text, event.GetText())
					require.Empty(t, event.GetReferenceState())
					require.Empty(t, event.GetDiffs())
				}

				// Randomly decide whether the backend sends a response, updating the reference state
				updateref := rand.Intn(10) < 5
				gotResponse = gotResponse || updateref
				if updateref {
					processor.updateLatestResponse("atom", fn, text, event.GetSelections(), false)
					ref = hash
				}
			}
		}(&wg, filepath)
	}

	wg.Wait()
}

func Test_EventOffsetConversion(t *testing.T) {
	_, processor := newTestEventProcessor()

	// offset 20 in utf-16 is the end of the string
	// this is 40 in utf-8
	event, err := processor.processEvent(&component.EditorEvent{
		Source:   "intellij",
		Action:   "edit",
		Filename: testpath,
		Text: `print("史史史史史史史史史史")
`,
		Selections: []*component.Selection{
			{
				Start:    20,
				End:      20,
				Encoding: stringindex.UTF16,
			},
		},
	})

	require.NoError(t, err)
	require.EqualValues(t, 40, *(event.Selections[0].Start))
	require.EqualValues(t, 40, *(event.Selections[0].End))
}

// --

func newTestEventProcessor() (*permissions.Manager, *eventProcessor) {
	f := filepath.Join(os.TempDir(), "test_permissions.json")
	os.RemoveAll(f)
	p := permissions.NewTestManager(lang.Python)
	return p, newEventProcessor(p, func() int {
		return 1024
	})
}

// --

func textMD5(text string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(text)))
}

func textChecksum(text string) uint64 {
	return spooky.Hash64([]byte(text))
}
