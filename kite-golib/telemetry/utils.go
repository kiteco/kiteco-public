package telemetry

import (
	"runtime"
	"time"

	uuid "github.com/satori/go.uuid"
)

// AugmentProps augments props with sent_at, os, and client_version
func AugmentProps(props map[string]interface{}) map[string]interface{} {
	if props == nil {
		props = make(map[string]interface{}, 3)
	}
	if props["sent_at"] == nil {
		props["sent_at"] = time.Now().Unix()
	}
	if props["os"] == nil {
		props["os"] = runtime.GOOS
	}
	if clientVersion != "" && props["client_version"] == nil {
		props["client_version"] = clientVersion
	}
	return props
}

func createMessage(userID, event string, props map[string]interface{}) Message {
	id, _ := uuid.NewV4()
	now := time.Now()

	return Message{
		MessageID:         id.String(),
		Version:           3,
		UserID:            userID,
		Event:             event,
		Timestamp:         now,
		OriginalTimestamp: now,
		Properties:        props,
	}
}
