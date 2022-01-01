package api

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/stretchr/testify/assert"
)

type pushEditorEventTC struct {
	oldEvents []*component.EditorEvent
	newEvents []*component.EditorEvent
	expected  []*component.EditorEvent
}

func TestPushCollectEvents(t *testing.T) {
	tcs := []pushEditorEventTC{
		pushEditorEventTC{
			nil,
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "first"},
				&component.EditorEvent{Text: "second"},
				&component.EditorEvent{Text: "third"},
			},
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "first"},
				&component.EditorEvent{Text: "second"},
				&component.EditorEvent{Text: "third"},
			},
		},
		pushEditorEventTC{
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "old one"},
				&component.EditorEvent{Text: "old two"},
			},
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "first"},
				&component.EditorEvent{Text: "second"},
				&component.EditorEvent{Text: "third"},
			},
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "old two"},
				&component.EditorEvent{Text: "first"},
				&component.EditorEvent{Text: "second"},
				&component.EditorEvent{Text: "third"},
			},
		},
		pushEditorEventTC{
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "old one"},
				&component.EditorEvent{Text: "old two"},
				&component.EditorEvent{Text: "old three"},
				&component.EditorEvent{Text: "old four"},
			},
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "first"},
				&component.EditorEvent{Text: "second"},
				&component.EditorEvent{Text: "third"},
				&component.EditorEvent{Text: "fourth"},
				&component.EditorEvent{Text: "fifth"},
			},
			[]*component.EditorEvent{
				&component.EditorEvent{Text: "second"},
				&component.EditorEvent{Text: "third"},
				&component.EditorEvent{Text: "fourth"},
				&component.EditorEvent{Text: "fifth"},
			},
		},
	}
	for _, tc := range tcs {
		events := NewEvents(4)
		events.evts = tc.oldEvents
		for _, event := range tc.newEvents {
			events.Push(event)
		}
		assert.Equal(t, tc.expected, events.Collect())
		assert.Equal(t, 0, len(events.Collect()))
	}
}
