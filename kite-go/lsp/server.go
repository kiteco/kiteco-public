package lsp

import (
	"encoding/json"
	"log"

	"github.com/kiteco/kiteco/kite-go/lsp/handlers"
	"github.com/kiteco/kiteco/kite-go/lsp/jsonrpc2"
	"github.com/kiteco/kiteco/kite-go/lsp/types"
)

const (
	serverCreated      = 0
	serverInitializing = 1
	serverInitialized  = 2
	serverShutDown     = 3
)

// Server handles interactions between LSP clients and Kite.
type Server struct {
	handlers *handlers.Handlers
	state    serverState
}

type serverState int

// New creates a new Server.
func New() *Server {
	return &Server{
		handlers: handlers.New(),
		state:    serverCreated,
	}
}

// Handle is called when a jsonrpc2 request is received and correctly parsed.
// The server must first be registered with an RPCConnection as a handler.
// Handle must call req.Reply on every request that expects a response.
func (lsp Server) Handle(req *jsonrpc2.Request) {
	switch req.Method {
	//
	// General LSP Session Messages
	//
	case "initialize":
		var params types.ParamInitialize
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		resp, err := lsp.initialize(&params)
		err = req.Reply(resp, err)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "initialized":
		var params types.InitializedParams
		var err error
		// Added this check in case clients send params:null instead of params:{}
		// (jupyterlab-lsp does this)
		// This is fine because InitializedParams currently has no acceptable fields anyway
		if req.Params != nil {
			err = json.Unmarshal(*req.Params, &params)
			if err != nil {
				req.SendParseError(err)
				return
			}
		}
		err = lsp.initialized(&params)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "shutdown":
		if req.Params != nil {
			req.Reply(nil, jsonrpc2.Errorf(jsonrpc2.CodeInvalidParams, "expected no params"))
			return
		}
		err := lsp.shutdown()
		err = req.Reply(nil, err)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "exit":
		err := lsp.exit()
		if err != nil {
			log.Printf("%q", err)
		}
		return
	//
	// Text Synchronization Messages
	//
	case "textDocument/didOpen":
		var params types.DidOpenTextDocumentParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		err = lsp.handlers.DidOpen(params)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "textDocument/didChange":
		var params types.DidChangeTextDocumentParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		err = lsp.handlers.DidChange(params)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "textDocument/didClose":
		var params types.DidCloseTextDocumentParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		err = lsp.handlers.DidClose(params)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	//
	// Languages Features
	//
	case "textDocument/completion":
		var params types.CompletionParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		resp, err := lsp.handlers.Completion(params)
		err = req.Reply(resp, err)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	//
	// Kite Features
	//
	case "kite/onboarding":
		var params types.KiteOnboardingParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		resp, err := lsp.handlers.Onboarding(params)
		err = req.Reply(resp, err)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "kite/selection":
		var params types.KiteSelectionParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		err = lsp.handlers.Selection(params)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "kite/status":
		var params types.KiteStatusParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		resp, err := lsp.handlers.Status(params)
		err = req.Reply(resp, err)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	case "kite/track":
		var params types.KiteTrackParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			req.SendParseError(err)
			return
		}
		err = lsp.handlers.Track(params)
		if err != nil {
			log.Printf("%q", err)
		}
		return
	default:
	}
}
