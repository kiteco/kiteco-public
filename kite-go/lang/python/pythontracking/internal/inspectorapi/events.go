package inspectorapi

import (
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
)

// ListingsMetadata describes information about a listings of events.
type ListingsMetadata struct {
	Date    string                   `json:"date"`
	Type    pythontracking.EventType `json:"type"`
	Failure string                   `json:"failure"`
	// AvailableTypes is a list of acceptable event types that can be queried to produce listings
	AvailableTypes []pythontracking.EventType `json:"available_types"`
	// FailureCounts contains the counts of each encountered failure for the currently selected event type.
	FailureCounts map[string]int `json:"failure_counts"`
}

// EventListings describes a list of events for a specific date.
type EventListings struct {
	Metadata ListingsMetadata `json:"metadata"`
	Events   []EventListing   `json:"events"`
}

// GroupedEventListings describes a list of events for a specific date,
// grouped by (user id, machine id, filename).
type GroupedEventListings struct {
	Metadata ListingsMetadata      `json:"metadata"`
	Groups   []GroupedEventListing `json:"groups"`
}

// GroupedEventListing describes a group of events for a specific (user id, machine id, filename) tuple.
type GroupedEventListing struct {
	UserID    int64          `json:"user_id"`
	MachineID string         `json:"machine_id"`
	Filename  string         `json:"filename"`
	Events    []EventListing `json:"events"`
}

// EventListing is a compact representation of an event.
type EventListing struct {
	URI       string    `json:"uri"`
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`

	UserID    int64  `json:"user_id"`
	MachineID string `json:"machine_id"`
	Filename  string `json:"filename"`

	Failure string `json:"failure"`
}

// EventDetail describes an event along with extra information taken from the rebuilt context.
type EventDetail struct {
	Type      pythontracking.EventType `json:"type"`
	URI       string                   `json:"uri"`
	MessageID string                   `json:"message_id"`
	Timestamp time.Time                `json:"timestamp"`

	UserID       int64             `json:"user_id"`
	MachineID    string            `json:"machine"`
	Filename     string            `json:"filename"`
	Buffer       string            `json:"buffer"`
	IndexedFiles map[string]string `json:"indexed_files"`
	IndexError   string            `json:"index_error"`

	Failure string `json:"failure"`

	// Fields related to the user's cursor
	Cursor       int64 `json:"cursor"`
	LineNumber   int   `json:"line_number"`
	ColumnNumber int   `json:"column_number"`

	// Fields used for expression browsing
	Exprs      []ExprListing `json:"exprs"`
	ExprDetail ExprDetail    `json:"expr_detail"`

	Callee *CalleeDetail `json:"callee,omitempty"`
}

// CalleeDetail contains information about the callee if it is relevant to the event.
type CalleeDetail struct {
	OutsideParens     bool                      `json:"outside_parens"`
	CalleeID          string                    `json:"callee_id"`
	CalleeResponse    *editorapi.CalleeResponse `json:"callee_response"`
	ReproducedFailure string                    `json:"reproduced_failure"`
	FuncType          string                    `json:"func_type"`
	Function          string                    `json:"function"`
}

// EventInfo has basic information about an event, not including local files or information from the context.
type EventInfo struct {
	URI       string               `json:"uri"`
	MessageID string               `json:"message_id"`
	Timestamp time.Time            `json:"timestamp"`
	Event     pythontracking.Event `json:"event"`
}
