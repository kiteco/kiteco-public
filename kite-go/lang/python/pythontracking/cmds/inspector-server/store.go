package main

import (
	"fmt"
	"log"
	"sort"

	lru "github.com/hashicorp/golang-lru"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/internal/inspectorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
)

var (
	eventSources = map[pythontracking.EventType]segmentsrc.Source{
		pythontracking.ServerCompletionsFailureEvent: segmentsrc.CompletionsTracking,
		pythontracking.ServerSignatureFailureEvent:   segmentsrc.CalleeTracking,
	}
	// map of EC2 regions to corresponding localfiles S3 buckets.
	// TODO(damian): switch to just getting the buckets from the logs once that's included
	bucketsByRegion = map[string]string{
		"us-west-1":      "kite-local-content",
		"us-west-2":      "kite-local-content-us-west-2",
		"us-east-1":      "kite-local-content-us-east-1",
		"eu-west-1":      "kite-local-content-eu-west-1",
		"ap-southeast-1": "kite-local-content-ap-southeast-1",
		"eastus":         "kite-local-content-us-east-1",
		"westus2":        "kite-local-content",
		"westeurope":     "kite-local-content-eu-west-1",
	}
)

type event struct {
	metadata analyze.Metadata
	track    *pythontracking.Event
}

type eventWithContext struct {
	metadata     analyze.Metadata
	track        *pythontracking.Event
	ctx          *python.Context
	indexedFiles map[string]string
	// Callee-specific cached information
	calleeResult *python.CalleeResult
}

// store is responsible for retrieving and caching information used by the inspector.
type store struct {
	eventListingsByURI    *lru.Cache
	eventsByID            *lru.Cache
	eventsWithContextByID *lru.Cache

	recreator *servercontext.Recreator
}

func newStore() (*store, error) {
	log.Println("Creating context recreator")
	recreator, err := servercontext.NewRecreator(bucketsByRegion)
	if err != nil {
		return nil, fmt.Errorf("could not create store: %v", err)
	}

	eventListingsByURI, err := lru.New(5000)
	if err != nil {
		return nil, fmt.Errorf("couldn't create eventListingsByURI cache")
	}

	eventsByID, err := lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("couldn't create eventsByID cache")
	}

	eventsWithContextByID, err := lru.New(100)
	if err != nil {
		return nil, fmt.Errorf("couldn't create eventsWithContextByID cache")
	}

	return &store{
		recreator:             recreator,
		eventListingsByURI:    eventListingsByURI,
		eventsByID:            eventsByID,
		eventsWithContextByID: eventsWithContextByID,
	}, nil
}

// Return a list of EventListings for a given event type. Currently, this gets the events for the latest available day.
// If failure is non-empty, filter events for that specific failure type.
// TODO(damian): Allow for the ability to get listings for an arbitrary day.
func (s *store) getEventListings(eventType pythontracking.EventType, failure string) (*inspectorapi.EventListings, error) {
	source, ok := eventSources[eventType]
	if !ok {
		return nil, fmt.Errorf("no bucket registered for event type: %s", string(eventType))
	}

	log.Printf("Getting URI listing, event type: %s, failure: %s", eventType, failure)
	uriListing, err := analyze.List(source)
	if err != nil {
		return nil, err
	}

	if len(uriListing.Dates) == 0 {
		return nil, fmt.Errorf("no events found for type: %s", eventType)
	}

	day := uriListing.Dates[len(uriListing.Dates)-1]

	var types []pythontracking.EventType
	for typ := range eventSources {
		types = append(types, typ)
	}
	sort.Slice(types, func(i, j int) bool {
		return string(types[i]) < string(types[j])
	})

	result := &inspectorapi.EventListings{
		Metadata: inspectorapi.ListingsMetadata{
			Date:           day.Date.String(),
			Type:           eventType,
			Failure:        failure,
			AvailableTypes: types,
			FailureCounts:  make(map[string]int),
		},
		Events: make([]inspectorapi.EventListing, 0),
	}

	var URIsToGet []string
	for _, uri := range day.URIs {
		events, ok := s.eventListingsByURI.Get(uri)
		if !ok {
			URIsToGet = append(URIsToGet, uri)
			continue
		}
		for _, event := range events.([]inspectorapi.EventListing) {
			result.Events = append(result.Events, event)
		}
	}

	log.Printf("%d out out of %d URIs were found in cache", len(day.URIs)-len(URIsToGet), len(day.URIs))
	if len(URIsToGet) != 0 {
		newEvents := s.retrieveEventListingsFromURIs(URIsToGet, eventType)
		result.Events = append(result.Events, newEvents...)
	}

	// Count the failure types and filuter out the ones that don't match the desired failure.
	var filtered []inspectorapi.EventListing
	for _, event := range result.Events {
		result.Metadata.FailureCounts[event.Failure]++
		if failure == "" || failure == event.Failure {
			filtered = append(filtered, event)
		}
	}
	result.Events = filtered

	// Sort the resulting events in descending order of time
	sort.Slice(result.Events, func(i, j int) bool {
		return result.Events[i].Timestamp.After(result.Events[j].Timestamp)
	})

	return result, nil
}

func (s *store) retrieveEventListingsFromURIs(URIs []string, eventType pythontracking.EventType) []inspectorapi.EventListing {
	var events []inspectorapi.EventListing
	numReadThreads := 16
	newEvents := make(map[string][]inspectorapi.EventListing)
	results := analyze.Analyze(URIs, numReadThreads, string(eventType),
		func(metadata analyze.Metadata, track *pythontracking.Event) bool {
			if track == nil {
				return false
			}
			if !shouldListEvent(track) {
				return true
			}
			eventFailure := track.Failure()
			event := inspectorapi.EventListing{
				MessageID: metadata.ID.ID,
				URI:       metadata.ID.URI,
				Timestamp: metadata.Timestamp,
				UserID:    track.UserID,
				MachineID: track.MachineID,
				Filename:  track.Filename,
				Failure:   eventFailure,
			}
			newEvents[metadata.ID.URI] = append(newEvents[metadata.ID.URI], event)
			events = append(events, event)
			return true
		})
	if results.Err != nil {
		log.Printf("error(s) ecountered in processing logs: %v", results.Err)
	}
	log.Printf("processed events: %d", results.ProcessedEvents)
	log.Printf("decode failures: %d", results.DecodeErrors)

	for uri, events := range newEvents {
		s.eventListingsByURI.Add(uri, events)
	}

	return events
}

type umf struct {
	User    int64
	Machine string
	File    string
}

// Return events grouped by user/machine/filename.
// If failure is non-empty, filter events for that specific failure type.
func (s *store) getGroupedEventListings(eventType pythontracking.EventType, failure string) (*inspectorapi.GroupedEventListings, error) {
	listings, err := s.getEventListings(eventType, failure)
	if err != nil {
		return nil, err
	}

	umfs := make(map[umf][]inspectorapi.EventListing)
	for _, entry := range listings.Events {
		umf := umf{
			User:    entry.UserID,
			Machine: entry.MachineID,
			File:    entry.Filename,
		}

		umfs[umf] = append(umfs[umf], entry)
	}

	grouped := &inspectorapi.GroupedEventListings{
		Metadata: listings.Metadata,
	}
	for umf, listings := range umfs {
		copied := append([]inspectorapi.EventListing{}, listings...)
		sort.Slice(copied, func(i, j int) bool {
			return copied[i].Timestamp.Before(copied[j].Timestamp)
		})
		grouped.Groups = append(grouped.Groups, inspectorapi.GroupedEventListing{
			UserID:    umf.User,
			MachineID: umf.Machine,
			Filename:  umf.File,
			Events:    copied,
		})
	}

	groups := grouped.Groups
	sort.Slice(groups, func(i, j int) bool {
		switch {
		case groups[i].UserID != groups[j].UserID:
			return groups[i].UserID < groups[j].UserID
		case groups[i].MachineID != groups[j].MachineID:
			return groups[i].MachineID < groups[j].MachineID
		default:
			return groups[i].Filename < groups[j].Filename
		}
	})

	return grouped, nil
}

func shouldListEvent(track *pythontracking.Event) bool {
	_, ok := bucketsByRegion[track.Region]
	if !ok {
		return false
	}
	if track.Type == pythontracking.ServerSignatureFailureEvent && track.Failure() == "call_expr_outside_parens" {
		return false
	}
	return true
}

// getEvent retrieves a pythontracking.Event event from S3 given a message ID.
// Returns nil if the event was not found.
func (s *store) getEvent(messageID analyze.MessageID) (*event, error) {
	cachedEvent, ok := s.eventsByID.Get(messageID)
	if ok {
		return cachedEvent.(*event), nil
	}

	track, metadata, results := analyze.GetSingleEvent(messageID, pythontracking.Event{})
	if results.Err != nil {
		return nil, results.Err
	}

	log.Printf("%d decode errors encountered in reading events from %s", results.DecodeErrors, messageID.URI)

	if track == nil {
		return nil, fmt.Errorf("event not found for URI: %s and ID: %s", messageID.URI, messageID.ID)
	}

	logEvent := track.(*pythontracking.Event)

	// Older messages will not have the Type field set, so we get it from top-level Segment log.
	// TODO(damian): We can remove this once there is a good backlog.
	if logEvent.Type == "" {
		logEvent.Type = pythontracking.EventType(metadata.EventName)
	}

	event := &event{
		metadata: metadata,
		track:    logEvent,
	}
	s.eventsByID.Add(messageID, event)
	return event, nil
}

// getEventWithContext retrieves a pythontracking.Event, along with the recreated python.Context, from S3 given a
// message ID. Returns nil if the event was not found.
func (s *store) getEventWithContext(messageID analyze.MessageID) (*eventWithContext, error) {
	cachedEvent, ok := s.eventsWithContextByID.Get(messageID)
	if ok {
		return cachedEvent.(*eventWithContext), nil
	}

	event, err := s.getEvent(messageID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve event with ID %+v: %v", messageID, err)
	}

	ctx, err := s.recreator.RecreateContext(event.track, true)
	if err != nil {
		return nil, fmt.Errorf("could not recreate context: %v", err)
	}

	indexedFiles, err := s.recreator.GetIndexedFiles(event.track)
	if err != nil {
		return nil, fmt.Errorf("could not get local files: %v", err)
	}

	var calleeResult *python.CalleeResult
	if event.track.Type == pythontracking.ServerSignatureFailureEvent {
		in := python.NewCalleeInputs(ctx, int64(event.track.Offset), s.recreator.Services)
		result := python.GetCallee(kitectx.Background(), in)
		calleeResult = &result
	}

	eventWithContext := &eventWithContext{
		metadata:     event.metadata,
		track:        event.track,
		ctx:          ctx,
		indexedFiles: indexedFiles,
		calleeResult: calleeResult,
	}
	s.eventsWithContextByID.Add(messageID, eventWithContext)
	return eventWithContext, nil
}
