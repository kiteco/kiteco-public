package dynamicanalysis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/sandbox"
)

// Load the code for python-side tracing
var eventCode = MustAsset("src/trace_events.py")

// Event represents something that was evaluated during tracing
type Event interface {
	event() // just an indicator to declare a struct as an event
}

// FirstObservation represents the first time an object was evaluated by the tracer
type FirstObservation struct {
	ID             int64    `json:"id"`
	TypeID         int64    `json:"type_id"`
	Str            string   `json:"str"`
	Repr           string   `json:"repr"`
	CanonicalName  string   `json:"canonical_name"`
	Classification string   `json:"classification"`
	Members        []string `json:"members"`
}

func (x *FirstObservation) event() {}

// Argument represents an argument in a traced function call
type Argument struct {
	Name    string `json:"name"`     // Name is the declared name of the keyword argument, or empty string
	ValueID int64  `json:"value_id"` // ValueID is the ID of the value for this argument
}

// Call represents a function that was called
type Call struct {
	FunctionID int64      `json:"function_id"`
	Arguments  []Argument `json:"arguments"`
	VarargID   int64      `json:"vararg_id"`
	KwargID    int64      `json:"kwarg_id"`
	ResultID   int64      `json:"result_id"`
}

func (x *Call) event() {}

// AttributeLookup represents an attribute that was accessed on an object
type AttributeLookup struct {
	Attribute string `json:"attribute"`
	ObjectID  int64  `json:"object_id"`
	ResultID  int64  `json:"result_id"`
}

func (x *AttributeLookup) event() {}

// TraceEvents executes the given python code and returns events associating
// expressions in the source file with their fully qualified typenames.
func TraceEvents(src string, opts TraceOptions) ([]Event, error) {
	// Construct the program
	prog := sandbox.NewContainerizedPythonProgram(string(eventCode), opts.DockerImage)
	prog.SupportingFiles["src.py"] = []byte(src)
	prog.EnvironmentVariables["PYTHONPATH"] = "."
	prog.EnvironmentVariables["SOURCE"] = "src.py"
	prog.EnvironmentVariables["TRACE_OUTPUT"] = "events.json"

	// Construct the apparatus
	apparatus, err := annotate.NewApparatusFromCode(src, lang.Python)
	if err != nil {
		return nil, fmt.Errorf("error constructing apparatus: %v", err)
	}

	// Run the program in the apparatus
	result, err := apparatus.Run(prog)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tracing code: %v", err)
	} else if !result.Succeeded {
		return nil, fmt.Errorf("tracing code exited uncleanly: %v\nPython said:\n%s", result.SandboxError, result.Stderr)
	} else if len(result.Stderr) > 0 {
		// Stderr is not empty but the process exited with status 0. In this case it may be
		// helpful to print what was received on stderr, but we should still continue anyway.
		log.Printf("Stderr from tracing: %s\n", string(result.Stderr))
	}

	// Look for the output file
	f := result.File("events.json")
	if f == nil {
		return nil, fmt.Errorf("tracing code did not generate events.json")
	}

	return LoadEvents(bytes.NewBuffer(f.Contents))
}

// LoadEvents parses a sequence of events from a json stream
func LoadEvents(r io.Reader) ([]Event, error) {
	// record is a struct used internally to deal with type information
	type record struct {
		Type  string          `json:"type"`
		Event json.RawMessage `json:"event"`
	}

	var events []Event
	d := json.NewDecoder(r)
	for {
		var r record
		err := d.Decode(&r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch r.Type {
		case "FirstObservation":
			events = append(events, &FirstObservation{})
		case "Call":
			events = append(events, &Call{})
		case "AttributeLookup":
			events = append(events, &AttributeLookup{})
		default:
			return nil, fmt.Errorf("unknown event type '%s'", r.Type)
		}
		err = json.Unmarshal(r.Event, events[len(events)-1])
		if err != nil {
			return nil, err
		}
	}
	return events, nil
}

// WriteEvents encodes a sequence of events to a json stream
func WriteEvents(w io.Writer, events []Event) error {
	// record is a struct used internally to deal with type information
	type record struct {
		Type  string `json:"type"`
		Event Event  `json:"event"`
	}

	enc := json.NewEncoder(w)
	for _, event := range events {
		r := record{Event: event}
		switch event.(type) {
		case *FirstObservation:
			r.Type = "FirstObservation"
		case *Call:
			r.Type = "Call"
		case *AttributeLookup:
			r.Type = "AttributeLookup"
		default:
			return fmt.Errorf("unknown event type %T", event)
		}

		err := enc.Encode(r)
		if err != nil {
			return err
		}
	}
	return nil
}
