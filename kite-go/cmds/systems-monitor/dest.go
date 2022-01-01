package main

import (
	"fmt"
)

// Data destinations

func sendAll(metrics []*Metric) error {
	// send to instrumental
	if err := sendInstrumental(formatInstrumental(metrics)); err != nil {
		return fmt.Errorf("error sending metrics to instrumental: %v", err)
	}

	return nil
}

func formatInstrumental(metrics []*Metric) []string {
	// separator for names
	sep := "."
	// message list
	var messages []string

	for _, met := range metrics {
		// metric name formatting
		name := ""
		for _, section := range met.Name {
			name = name + section + sep
		}
		// include unit as part of the name
		name = name + met.Unit

		// add message to list
		messages = append(messages, fmt.Sprintf("gauge %s %f", name, met.Value))
	}

	return messages
}
