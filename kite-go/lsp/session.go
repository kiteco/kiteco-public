package lsp

import (
	"os"

	"github.com/kiteco/kiteco/kite-go/lsp/jsonrpc2"

	"github.com/kiteco/kiteco/kite-go/lsp/types"
)

func (lsp *Server) initialize(params *types.ParamInitialize) (*types.InitializeResult, error) {
	if lsp.state != serverCreated {
		return nil, jsonrpc2.Errorf(jsonrpc2.CodeInvalidRequest, "server is already initialized")
	}
	lsp.state = serverInitializing

	triggerString := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789,.[]{}()'\""
	triggerChars := make([]string, 0)
	for _, char := range triggerString {
		triggerChars = append(triggerChars, string(char))
	}

	lsp.handlers.Options = params.InitializationOptions

	var result *types.InitializeResult
	result = &types.InitializeResult{
		Capabilities: types.ServerCapabilities{
			TextDocumentSync: 1,
			CompletionProvider: types.CompletionOptions{
				TriggerCharacters:   triggerChars,
				AllCommitCharacters: nil,
				ResolveProvider:     false,
				WorkDoneProgressOptions: types.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			HoverProvider: false,
			SignatureHelpProvider: types.SignatureHelpOptions{
				TriggerCharacters:   nil,
				RetriggerCharacters: nil,
				WorkDoneProgressOptions: types.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			DeclarationProvider:       nil,
			DefinitionProvider:        false,
			TypeDefinitionProvider:    nil,
			ImplementationProvider:    nil,
			ReferencesProvider:        false,
			DocumentHighlightProvider: false,
			DocumentSymbolProvider:    false,
			CodeActionProvider:        nil,
			CodeLensProvider: types.CodeLensOptions{
				ResolveProvider: false,
				WorkDoneProgressOptions: types.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			DocumentLinkProvider: types.DocumentLinkOptions{
				ResolveProvider: false,
				WorkDoneProgressOptions: types.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			ColorProvider:                   nil,
			WorkspaceSymbolProvider:         false,
			DocumentFormattingProvider:      false,
			DocumentRangeFormattingProvider: false,
			DocumentOnTypeFormattingProvider: types.DocumentOnTypeFormattingOptions{
				FirstTriggerCharacter: "",
				MoreTriggerCharacter:  nil,
			},
			RenameProvider:         nil,
			FoldingRangeProvider:   nil,
			SelectionRangeProvider: nil,
			ExecuteCommandProvider: types.ExecuteCommandOptions{
				Commands: nil,
				WorkDoneProgressOptions: types.WorkDoneProgressOptions{
					WorkDoneProgress: false,
				},
			},
			KiteProvider: true,
			Experimental: nil,
			Workspace: types.WorkspaceGn{
				WorkspaceFolders: types.WorkspaceFoldersGn{
					Supported:           false,
					ChangeNotifications: "",
				},
			},
		},
		ServerInfo: struct {
			Name    string `json:"name"`
			Version string `json:"version,omitempty"`
		}{
			Name:    "kite-lsp",
			Version: "",
		},
	}
	return result, nil
}

func (lsp *Server) initialized(params *types.InitializedParams) error {
	lsp.state = serverInitialized

	return nil
}

func (lsp *Server) shutdown() error {
	if lsp.state < serverInitialized {
		return jsonrpc2.Errorf(jsonrpc2.CodeInvalidRequest, "server not initialized")
	}
	lsp.state = serverShutDown
	return nil
}

func (lsp *Server) exit() error {
	if lsp.state != serverShutDown {
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}
