package jsonrpc2

// Derived from:
// https://github.com/golang/tools/blob/master/internal/jsonrpc2/wire.go

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	// CodeUnknownError should be used for all non coded errors.
	CodeUnknownError = -32001
	// CodeParseError is used when invalid JSON was received by the server.
	CodeParseError = -32700
	//CodeInvalidRequest is used when the JSON sent is not a valid Request object.
	CodeInvalidRequest = -32600
	// CodeMethodNotFound should be returned by the handler when the method does
	// not exist / is not available.
	CodeMethodNotFound = -32601
	// CodeInvalidParams should be returned by the handler when method
	// parameter(s) were invalid.
	CodeInvalidParams = -32602
	// CodeInternalError is not currently returned but defined for completeness.
	CodeInternalError = -32603

	//CodeServerOverloaded is returned when a message was refused due to a
	//server being temporarily unable to accept any new messages.
	CodeServerOverloaded = -32000
)

// Request is an augmented form of a raw jsonrpc2 request.
type Request struct {
	conn *RPCConnection
	RequestBody
}

// RequestBody represents a jsonrpc2 request
type RequestBody struct {
	Version Version          `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params,omitempty"`
	ID      *ID              `json:"id,omitempty"`
}

// Response represents a jsonrpc2 response.
type Response struct {
	Version Version          `json:"jsonrpc"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError   `json:"error,omitempty"`
	ID      *ID              `json:"id,omitempty"`
}

// ResponseError represents the inner error object of a jsonrpc2 response.
type ResponseError struct {
	Code    int64
	Message string
	Data    *json.RawMessage
}

func (err *ResponseError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

// Errorf builds a Error struct for the supplied message and code.
// If args is not empty, message and args will be passed to Sprintf.
func Errorf(code int64, format string, args ...interface{}) *ResponseError {
	return &ResponseError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// ID is the id component of a jsonrpc2 message.
// Only one of Name, Number should be set.
type ID struct {
	Name   string
	Number int64
}

func (id *ID) String() string {
	if id == nil {
		return ""
	}
	if id.Name != "" {
		return strconv.Quote(id.Name)
	}
	return "#" + strconv.FormatInt(id.Number, 10)
}

// MarshalJSON marshals an ID
func (id *ID) MarshalJSON() ([]byte, error) {
	if id.Name != "" {
		return json.Marshal(id.Name)
	}
	return json.Marshal(id.Number)
}

// UnmarshalJSON evaluates the type of a json id and sets it in an ID
func (id *ID) UnmarshalJSON(data []byte) error {
	*id = ID{}
	err := json.Unmarshal(data, &id.Number)
	if err == nil {
		return nil
	}
	return json.Unmarshal(data, &id.Name)
}

// IsNotification returns true if the given request is a notification.
// Notifications do not expect a response
func (req *Request) IsNotification() bool {
	return req.ID == nil
}

// Version is a placeholder tag that represents the version in a jsonrpc2 message.
// It must be received as "2.0" and will automatically be filled with that when sent.
type Version struct{}

// MarshalJSON sets the version of outgoing messages to "2.0"
func (Version) MarshalJSON() ([]byte, error) {
	return json.Marshal("2.0")
}

// UnmarshalJSON parses the version information of a received jsonrpc2 message,
// returning an error if the version isn't "2.0"
func (Version) UnmarshalJSON(data []byte) error {
	var version string
	err := json.Unmarshal(data, &version)
	if err != nil {
		return err
	}
	if version != "2.0" {
		return errors.New("invalid RPC version %v", version)
	}
	return nil
}
