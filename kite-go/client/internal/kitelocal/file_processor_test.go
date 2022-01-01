package kitelocal

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

func TestFileProcessor_SequentialEvents(t *testing.T) {
	fp := newTestFileProcessor()
	var prevEvt *event.Event
	for i := 0; i < 1000; i++ {
		text := strings.Repeat("a", i+1)
		hash := fmt.Sprintf("%x", md5.Sum([]byte(text)))
		evt := &event.Event{
			Action:       proto.String("edit"),
			Source:       proto.String("atom"),
			Filename:     proto.String("test.py"),
			Text:         proto.String(text),
			TextMD5:      proto.String(hash),
			TextChecksum: proto.Uint64(spooky.Hash64([]byte(text))),
		}

		if prevEvt != nil {
			evt.Text = nil
			evt.Diffs = []*event.Diff{
				{
					Type:   event.DiffType.Enum(event.DiffType_INSERT),
					Offset: proto.Int32(int32(len(prevEvt.GetText()))),
					Text:   proto.String("a"),
				},
			}
		}

		results, err := fp.handleEvent(kitectx.Background(), evt)
		require.NoError(t, err)
		require.False(t, results.resend)
		require.Equal(t, hash, results.state)
		prevEvt = evt

		require.True(t, len(fp.guardedStates) <= maxStates)
	}
}

func TestFileProcessor_EventsWithReference(t *testing.T) {
	type eventReference struct {
		text  string
		state string
	}

	var refs []eventReference

	fp := newTestFileProcessor()
	for i := 0; i < 1000; i++ {
		text := strings.Repeat("a", i+1)
		hash := fmt.Sprintf("%x", md5.Sum([]byte(text)))
		evt := &event.Event{
			Action:       proto.String("edit"),
			Source:       proto.String("atom"),
			Filename:     proto.String("test.py"),
			Text:         proto.String(text),
			TextMD5:      proto.String(hash),
			TextChecksum: proto.Uint64(spooky.Hash64([]byte(text))),
		}

		if len(refs) > 0 {
			// Select an index within the last N states, so that we hit an existing reference
			var idx int
			switch {
			case len(refs) > maxStates:
				idx = len(refs) - rand.Intn(maxStates) - 1
			default:
				idx = rand.Intn(len(refs))
			}

			ref := refs[idx]

			evt.Text = nil
			evt.ReferenceState = proto.String(ref.state)
			evt.Diffs = []*event.Diff{
				{
					Type:   event.DiffType.Enum(event.DiffType_INSERT),
					Offset: proto.Int32(int32(len(ref.text))),
					Text:   proto.String(strings.Repeat("a", i-idx)),
				},
			}
		}

		results, err := fp.handleEvent(kitectx.Background(), evt)
		require.NoError(t, err)
		require.Equal(t, hash, results.state)
		require.False(t, results.resend)

		require.True(t, len(fp.guardedStates) <= maxStates)

		refs = append(refs, eventReference{
			text:  text,
			state: hash,
		})
	}
}

func TestFileProcessor_Mixed(t *testing.T) {
	type eventReference struct {
		text  string
		state string
	}

	var refs []eventReference

	fp := newTestFileProcessor()
	for i := 0; i < 1000; i++ {
		text := strings.Repeat("a", i+1)
		hash := fmt.Sprintf("%x", md5.Sum([]byte(text)))
		evt := &event.Event{
			Action:       proto.String("edit"),
			Source:       proto.String("atom"),
			Filename:     proto.String("test.py"),
			Text:         proto.String(text),
			TextMD5:      proto.String(hash),
			TextChecksum: proto.Uint64(spooky.Hash64([]byte(text))),
		}

		if len(refs) > 0 {
			var idx int

			switch rand.Intn(6) {
			case 0:
				// Select previous event as reference
				idx = len(refs) - 1
			case 1:
				// Select previous event as reference, no text
				evt.Text = nil
				idx = len(refs) - 1
			case 2:
				// Select an ref event within maxStates
				switch {
				case len(refs) > maxStates:
					idx = len(refs) - rand.Intn(maxStates) - 1
				default:
					idx = rand.Intn(len(refs))
				}
			case 3:
				// Select an ref event within maxStates, without text
				evt.Text = nil
				switch {
				case len(refs) > maxStates:
					idx = len(refs) - rand.Intn(maxStates) - 1
				default:
					idx = rand.Intn(len(refs))
				}
			case 4:
				// Select an older ref outside of maxStates, with text
				switch {
				case len(refs) > maxStates:
					idx = rand.Intn(len(refs) - maxStates)
				default:
					idx = rand.Intn(len(refs))
				}
			case 5:
				// Select an older ref outside of maxStates, without text
				evt.Text = nil
				switch {
				case len(refs) > maxStates:
					idx = rand.Intn(len(refs) - maxStates)
				default:
					idx = rand.Intn(len(refs))
				}
			}

			ref := refs[idx]

			evt.ReferenceState = proto.String(ref.state)
			evt.Diffs = []*event.Diff{
				{
					Type:   event.DiffType.Enum(event.DiffType_INSERT),
					Offset: proto.Int32(int32(len(ref.text))),
					Text:   proto.String(strings.Repeat("a", i-idx)),
				},
			}

			// Error expected if reference is not found and text is nil
			sk := stateKey{evt.GetFilename(), evt.GetSource(), ref.state}
			_, ok := fp.guardedStates[sk]
			errExpected := !ok && evt.Text == nil

			results, err := fp.handleEvent(kitectx.Background(), evt)
			if errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.False(t, results.resend)
				require.Equal(t, hash, results.state)
			}

			require.True(t, len(fp.guardedStates) <= maxStates)
		}

		refs = append(refs, eventReference{
			text:  text,
			state: hash,
		})
	}
}

// --

func newTestFileProcessor() *fileProcessor {
	return &fileProcessor{
		guardedStates: make(map[stateKey]*fileDriver),
		testing:       true,
	}
}
