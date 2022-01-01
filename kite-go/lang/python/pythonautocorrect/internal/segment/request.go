package segment

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect/internal/api"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kr/pretty"
)

// Request tracking information.
type Request struct {
	Timestamp time.Time

	UserID  int64
	Machine string

	Req editorapi.AutocorrectRequest

	LexErr errors.Errors

	ParseErr errors.Errors

	Proposals []api.Proposal

	Selected *api.Proposal

	EndState string

	Opts pythonautocorrect.Options
}

type requestProperties struct {
	Event   string                       `json:"event"`
	Machine string                       `json:"machine"`
	Request editorapi.AutocorrectRequest `json:"request"`
	Region  string                       `json:"region"`
	Options pythonautocorrect.Options    `json:"options"`

	LexErr    errors.Errors  `json:"lex_err"`
	ParseErr  errors.Errors  `json:"parse_err"`
	Proposals []api.Proposal `json:"proposals"`
	Selected  *api.Proposal  `json:"selected"`
	EndState  string         `json:"end_state"`
}

type rawRequest struct {
	commonFields
	Properties requestProperties `json:"properties"`
}

// RequestIterator iterates over segment tracking events
// associated with an autocorrect request.
type RequestIterator struct {
	downloader *downloader
	req        Request
	err        error
}

// NewRequestIterator returns a new iterator
// over segment tracking events associated with an
// autocorrect request.
func NewRequestIterator(bucket, key string) (*RequestIterator, error) {
	downloader, err := newDownloader(bucket, key)
	if err != nil {
		return nil, err
	}
	return &RequestIterator{
		downloader: downloader,
	}, nil
}

// Next advances the iterator. Next will return false on completion or error. Be sure
// to check Err() to determine if there was a non-EOF error.
func (i *RequestIterator) Next() bool {
	if i.err != nil || i.downloader.Err() != nil {
		return false
	}

	for i.downloader.Next() {
		var evt event
		if err := json.Unmarshal(i.downloader.Value(), &evt); err != nil {
			i.quit(fmt.Errorf("error unmarshaling event: %v", err))
			return false
		}

		if evt.Event != "autocorrect" || evt.Properties.Event != "request_funnel" {
			continue
		}

		var rr rawRequest
		if err := json.Unmarshal(i.downloader.Value(), &rr); err != nil {
			i.quit(fmt.Errorf("error unmarshalling raw request: %v", err))
			return false
		}

		req, err := i.fromRaw(rr)
		if err != nil {
			i.quit(err)
			return false
		}
		i.req = req

		return true
	}
	i.quit(io.EOF)
	return false
}

func (i *RequestIterator) quit(err error) {
	i.err = err
	i.downloader.Close()
}

func (i *RequestIterator) fromRaw(rr rawRequest) (Request, error) {
	ts, err := parseTimestamp(rr.Timestamp)
	if err != nil {
		return Request{}, fmt.Errorf("error parsing timestamp from `%s`: %v", pretty.Sprintf("%#v", rr), err)
	}

	uid, err := parseUserID(rr.UserID)
	if err != nil {
		return Request{}, fmt.Errorf("error parsing userid from `%s`: %v", pretty.Sprintf("%#v", rr), err)
	}

	return Request{
		Timestamp: ts,
		UserID:    uid,
		Machine:   rr.Properties.Machine,
		ParseErr:  rr.Properties.ParseErr,
		LexErr:    rr.Properties.LexErr,
		Proposals: rr.Properties.Proposals,
		Selected:  rr.Properties.Selected,
		EndState:  rr.Properties.EndState,
		Req:       rr.Properties.Request,
	}, nil
}

// Err returns any non-EOF errors.
func (i *RequestIterator) Err() error {
	if i.err != nil && i.err != io.EOF {
		return i.err
	}
	return i.downloader.Err()
}

// Request returns the request the iterator is currently on.
func (i *RequestIterator) Request() Request {
	return i.req
}
