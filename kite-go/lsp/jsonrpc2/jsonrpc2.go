package jsonrpc2

// Derived from:
// https://github.com/golang/tools/blob/master/internal/jsonrpc2/jsonrpc2.go

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// RPCConnection represents a single bi-directional jsonrpc2 connection.
type RPCConnection struct {
	transport TransportConnection
	handler   RPCHandler
}

// NewRPCConnection returns a new RPCConnection
func NewRPCConnection(transport TransportConnection, handler RPCHandler) *RPCConnection {
	rpc := &RPCConnection{
		transport: transport,
		handler:   handler,
	}
	return rpc
}

// Reply sends a reply to the given request.
// Reply should be called by the handler once for every request that is not a notification.
func (req *Request) Reply(result interface{}, err error) error {
	if req.IsNotification() {
		return errors.New("reply called on notification: %v, %v", req.Method, req.Params)
	}

	// We marshal the inner type separately from the whole message.
	// This means if the inner type fails to marshal, we are able to send an error instead of being unable to respond.
	var raw *json.RawMessage
	if err == nil {
		raw, err = marshalToRaw(result)
	}
	response := &Response{
		Result: raw,
		ID:     req.ID,
	}
	if err != nil {
		// Determine if the given err is already an ResponseError
		callErr, ok := err.(*ResponseError)
		if ok {
			response.Error = callErr
		} else {
			response.Error = &ResponseError{
				Code:    CodeInternalError,
				Message: fmt.Sprintf("%s", err),
			}
		}
		log.Printf("%+v", errors.WithStack(err))
	}
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	err = req.conn.transport.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// Run blocks until the connection is terminated.
// It reads in requests, validates them, and invokes the handler on them.
func (c *RPCConnection) Run() error {
	for {
		data, err := c.transport.Read()
		if err != nil {
			return err
		}
		reqBody := RequestBody{}
		err = json.Unmarshal(data, &reqBody)
		req := &Request{
			conn:        c,
			RequestBody: reqBody,
		}
		if err != nil {
			//TODO handle a bad message
			return err
		}

		if c.handler != nil {
			go c.handler.Handle(req)
		}
	}
}

// SendParseError replies to the given request with a jsonrpc2 ParseError
func (req *Request) SendParseError(err error) {
	_, ok := err.(*ResponseError)
	if !ok {
		err = Errorf(CodeParseError, "%v", err)
	}
	err = req.Reply(nil, err)
	if err != nil {
		log.Printf("%q\n", err)
	}
}

func marshalToRaw(obj interface{}) (*json.RawMessage, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(data)
	return &raw, nil
}
