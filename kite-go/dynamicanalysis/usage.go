package dynamicanalysis

import "fmt"

// PassContext represents information about how an object was passed to a function
type PassContext struct {
	Function string // canonical name of function passed to
	Keyword  string // name of the argument, if it was passed with a keyword, or "*" or "**"
	Position int    // Position in the argument list (including other keyword args)
}

// Usage represents an object that had zero or more attributes accessed on it while being traced
type Usage struct {
	Object       *FirstObservation // information about the object from its first observation
	Type         string            // canonical name of the object's type
	ReturnedFrom string            // canonical name of function that this object was returned from
	PassedTo     []PassContext     // functions that this object was passed to
	Attributes   []string          // attributes that were accessed, in order of access
}

// UsagesFromEvents converts an event stream to a set of usages for each observed object
func UsagesFromEvents(events []Event) ([]*Usage, error) {
	// First construct the map from ID to observation
	usages := make(map[int64]*Usage)
	for _, event := range events {
		if event, ok := event.(*FirstObservation); ok {
			if _, seen := usages[event.ID]; seen {
				return nil, fmt.Errorf("duplicate observations for ID %d", event.ID)
			}
			usages[event.ID] = &Usage{Object: event}
		}
	}

	// Fill in type info
	for _, usage := range usages {
		usage.Type = usages[usage.Object.TypeID].Object.CanonicalName
	}

	// Now process the rest of the events
	for _, event := range events {
		switch event := event.(type) {
		case *Call:
			fun := usages[event.FunctionID].Object.CanonicalName
			usages[event.ResultID].ReturnedFrom = fun
			for i, arg := range event.Arguments {
				usages[arg.ValueID].PassedTo = append(usages[arg.ValueID].PassedTo, PassContext{
					Position: i,
					Function: fun,
					Keyword:  arg.Name,
				})
			}
			if event.VarargID != 0 {
				usages[event.VarargID].PassedTo = append(usages[event.VarargID].PassedTo, PassContext{
					Function: fun,
					Keyword:  "*",
				})
			}
			if event.KwargID != 0 {
				usages[event.KwargID].PassedTo = append(usages[event.KwargID].PassedTo, PassContext{
					Function: fun,
					Keyword:  "**",
				})
			}
		case *AttributeLookup:
			usages[event.ObjectID].Attributes = append(usages[event.ObjectID].Attributes, event.Attribute)
		}
	}

	// Now flatten the map into a list
	var out []*Usage
	for _, usage := range usages {
		out = append(out, usage)
	}
	return out, nil
}
