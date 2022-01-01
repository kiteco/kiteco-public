package segment

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kr/pretty"
)

type onSaveEvent struct {
	UserID     string                       `json:"userId"`
	Timestamp  string                       `json:"timestamp"`
	Properties editorapi.AutocorrectRequest `json:"properties"`
}

// OnSaveEvent stored on s3 from segment.
type OnSaveEvent struct {
	UserID    int64
	Timestamp time.Time
	Request   editorapi.AutocorrectRequest
}

// OnSaveEventIterator for segment on save events
type OnSaveEventIterator struct {
	downloader *downloader

	err   error
	event OnSaveEvent
}

// NewOnSaveEventIterator returns a reader ready to read on save events
// from s3.
func NewOnSaveEventIterator(bucket, key string) (*OnSaveEventIterator, error) {
	downloader, err := newDownloader(bucket, key)
	if err != nil {
		return nil, err
	}

	return &OnSaveEventIterator{
		downloader: downloader,
	}, nil
}

// Next advances the iterator. Next will return false on completion or error. Be sure
// to check Err() to determine if there was a non-EOF error.
func (i *OnSaveEventIterator) Next() bool {
	if i.err != nil || i.downloader.Err() != nil {
		return false
	}

	if i.downloader.Next() {
		var event onSaveEvent
		if err := json.Unmarshal(i.downloader.Value(), &event); err != nil {
			i.quit(fmt.Errorf("error unmarshalling event: %v", err))
			return false
		}

		// parse user id and timestamp
		uid, err := parseUserID(event.UserID)
		if err != nil {
			i.quit(fmt.Errorf("error parsing user id from event `%s`: %v", pretty.Sprintf("%#v", event), err))
			return false
		}

		ts, err := parseTimestamp(event.Timestamp)
		if err != nil {
			i.quit(fmt.Errorf("error parsing timestamp from event `%s`: %v", pretty.Sprintf("%#v", event), err))
			return false
		}

		i.event = OnSaveEvent{
			UserID:    uid,
			Timestamp: ts,
			Request:   event.Properties,
		}
		return true
	}

	i.quit(io.EOF)
	return false
}

func (i *OnSaveEventIterator) quit(err error) {
	i.downloader.Close()
	i.err = err
}

// Err returns any non-EOF errors.
func (i *OnSaveEventIterator) Err() error {
	if i.err != nil && i.err != io.EOF {
		return i.err
	}
	return i.downloader.Err()
}

// Event returns the event the iterator is currently on.
func (i *OnSaveEventIterator) Event() OnSaveEvent {
	return i.event
}
