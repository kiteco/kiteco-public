package jsonrpc2

// RPCHandler is an interface that defines the hooks that will be called by an RPCConnection when processing messages.
type RPCHandler interface {
	// Handle is called once a json-rpc2 request has been received and correctly parsed.
	// It is expected that the RPCHandler calls Reply on any request that is not a notification
	Handle(req *Request)
}
