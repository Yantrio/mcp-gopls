package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

type Client struct {
	process      *exec.Cmd
	conn         *jsonrpc2.Conn
	capabilities ServerCapabilities
	handler      *serverHandler

	mu          sync.Mutex
	initialized bool
	openDocs    map[string]bool
	rootURI     string
}

func NewClient(goplsPath string) (*Client, error) {
	if goplsPath == "" {
		goplsPath = "gopls"
	}

	cmd := exec.Command(goplsPath, "serve")
	cmd.Stderr = os.Stderr

	handler := &serverHandler{
		diagnostics: make(map[string][]Diagnostic),
	}

	conn, err := newProcessConnection(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	client := &Client{
		process:  cmd,
		conn:     conn,
		handler:  handler,
		openDocs: make(map[string]bool),
	}

	return client, nil
}

func (c *Client) Initialize(ctx context.Context, rootURI string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return fmt.Errorf("client already initialized")
	}

	params := InitializeParams{
		ProcessID: os.Getpid(),
		RootURI:   rootURI,
		Capabilities: ClientCapabilities{
			TextDocument: TextDocumentClientCapabilities{
				Synchronization: TextDocumentSyncClientCapabilities{
					DidSave: true,
				},
				Definition: DefinitionClientCapabilities{},
				References: ReferenceClientCapabilities{},
				Hover:      HoverClientCapabilities{},
				Rename:     RenameClientCapabilities{
					PrepareSupport: true,
				},
			},
			Workspace: WorkspaceClientCapabilities{
				ApplyEdit: true,
				WorkspaceEdit: WorkspaceEditClientCapabilities{
					DocumentChanges: true,
				},
				Symbol: WorkspaceSymbolClientCapabilities{},
			},
		},
	}

	var result InitializeResult
	if err := c.conn.Call(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	c.capabilities = result.Capabilities
	c.rootURI = rootURI

	// Send initialized notification
	if err := c.conn.Notify(ctx, "initialized", &InitializedParams{}); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	c.initialized = true
	return nil
}

func (c *Client) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}

	// Close all open documents
	for uri := range c.openDocs {
		_ = c.closeDocument(ctx, uri)
	}

	// Send shutdown request
	if err := c.conn.Call(ctx, "shutdown", nil, nil); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	// Send exit notification
	if err := c.conn.Notify(ctx, "exit", nil); err != nil {
		return fmt.Errorf("exit notification failed: %w", err)
	}

	// Close connection
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	// Wait for process to exit
	if err := c.process.Wait(); err != nil {
		// Ignore error if process was already terminated
		if _, ok := err.(*exec.ExitError); !ok {
			return fmt.Errorf("failed to wait for process: %w", err)
		}
	}

	c.initialized = false
	return nil
}

func (c *Client) OpenDocument(ctx context.Context, uri string, content string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return fmt.Errorf("client not initialized")
	}

	if c.openDocs[uri] {
		return nil // Already open
	}

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       content,
		},
	}

	if err := c.conn.Notify(ctx, "textDocument/didOpen", params); err != nil {
		return fmt.Errorf("didOpen notification failed: %w", err)
	}

	c.openDocs[uri] = true
	return nil
}

func (c *Client) CloseDocument(ctx context.Context, uri string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.closeDocument(ctx, uri)
}

func (c *Client) closeDocument(ctx context.Context, uri string) error {
	if !c.openDocs[uri] {
		return nil // Not open
	}

	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{
			URI: uri,
		},
	}

	if err := c.conn.Notify(ctx, "textDocument/didClose", params); err != nil {
		return fmt.Errorf("didClose notification failed: %w", err)
	}

	delete(c.openDocs, uri)
	return nil
}

func (c *Client) Definition(ctx context.Context, uri string, position Position) ([]Location, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := DefinitionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}

	var result json.RawMessage
	if err := c.conn.Call(ctx, "textDocument/definition", params, &result); err != nil {
		return nil, fmt.Errorf("definition request failed: %w", err)
	}

	// Handle both single Location and []Location responses
	var locations []Location
	if err := json.Unmarshal(result, &locations); err != nil {
		var singleLocation Location
		if err := json.Unmarshal(result, &singleLocation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal definition result: %w", err)
		}
		locations = []Location{singleLocation}
	}

	return locations, nil
}

func (c *Client) References(ctx context.Context, uri string, position Position, includeDeclaration bool) ([]Location, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := ReferenceParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
		Context: ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}

	var locations []Location
	if err := c.conn.Call(ctx, "textDocument/references", params, &locations); err != nil {
		return nil, fmt.Errorf("references request failed: %w", err)
	}

	return locations, nil
}

func (c *Client) Hover(ctx context.Context, uri string, position Position) (*Hover, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}

	var result Hover
	if err := c.conn.Call(ctx, "textDocument/hover", params, &result); err != nil {
		return nil, fmt.Errorf("hover request failed: %w", err)
	}

	return &result, nil
}

func (c *Client) PrepareRename(ctx context.Context, uri string, position Position) (*PrepareRenameResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := PrepareRenameParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}

	var result *PrepareRenameResult
	if err := c.conn.Call(ctx, "textDocument/prepareRename", params, &result); err != nil {
		return nil, fmt.Errorf("prepareRename request failed: %w", err)
	}

	return result, nil
}

func (c *Client) Rename(ctx context.Context, uri string, position Position, newName string) (*WorkspaceEdit, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := RenameParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
		NewName: newName,
	}

	var result json.RawMessage
	if err := c.conn.Call(ctx, "textDocument/rename", params, &result); err != nil {
		return nil, fmt.Errorf("rename request failed: %w", err)
	}

	// Debug: log the raw response
	fmt.Fprintf(os.Stderr, "DEBUG: Rename raw response: %s\n", string(result))

	// Check if result is null
	if string(result) == "null" || len(result) == 0 {
		return &WorkspaceEdit{Changes: make(map[string][]TextEdit)}, nil
	}

	// Parse the workspace edit
	var workspaceEdit WorkspaceEdit
	if err := json.Unmarshal(result, &workspaceEdit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rename result: %w", err)
	}

	return &workspaceEdit, nil
}

func (c *Client) GetDiagnostics(uri string) []Diagnostic {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handler.diagnostics == nil {
		return nil
	}

	return c.handler.diagnostics[uri]
}

func (c *Client) Implementation(ctx context.Context, uri string, position Position) ([]Location, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := ImplementationParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}

	var result json.RawMessage
	if err := c.conn.Call(ctx, "textDocument/implementation", params, &result); err != nil {
		return nil, fmt.Errorf("implementation request failed: %w", err)
	}

	// Handle both single Location and []Location responses
	var locations []Location
	if err := json.Unmarshal(result, &locations); err != nil {
		var singleLocation Location
		if err := json.Unmarshal(result, &singleLocation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal implementation result: %w", err)
		}
		locations = []Location{singleLocation}
	}

	return locations, nil
}

func (c *Client) DocumentSymbols(ctx context.Context, uri string) ([]DocumentSymbol, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	var rawResult json.RawMessage
	if err := c.conn.Call(ctx, "textDocument/documentSymbol", params, &rawResult); err != nil {
		return nil, fmt.Errorf("documentSymbol request failed: %w", err)
	}

	// Try to unmarshal as DocumentSymbol[]
	var docSymbols []DocumentSymbol
	if err := json.Unmarshal(rawResult, &docSymbols); err == nil {
		return docSymbols, nil
	}

	// If that fails, try SymbolInformation[] and convert
	var symInfos []SymbolInformation
	if err := json.Unmarshal(rawResult, &symInfos); err != nil {
		return nil, fmt.Errorf("failed to parse document symbols: %w", err)
	}

	// Convert SymbolInformation to DocumentSymbol
	result := make([]DocumentSymbol, 0, len(symInfos))
	for _, info := range symInfos {
		result = append(result, DocumentSymbol{
			Name:           info.Name,
			Kind:           info.Kind,
			Range:          info.Location.Range,
			SelectionRange: info.Location.Range,
			Children:       []DocumentSymbol{},
		})
	}

	return result, nil
}

func (c *Client) DocumentFormatting(ctx context.Context, uri string) ([]TextEdit, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Options: FormattingOptions{
			TabSize:      4,
			InsertSpaces: false, // Use tabs for Go
		},
	}

	var edits []TextEdit
	if err := c.conn.Call(ctx, "textDocument/formatting", params, &edits); err != nil {
		return nil, fmt.Errorf("formatting request failed: %w", err)
	}

	return edits, nil
}

func (c *Client) CodeActionForRange(ctx context.Context, uri string, r Range) ([]CodeAction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        r,
		Context: CodeActionContext{
			Diagnostics: []Diagnostic{},
			Only:        []CodeActionKind{CodeActionKindSourceOrganizeImports},
		},
	}

	var actions []CodeAction
	if err := c.conn.Call(ctx, "textDocument/codeAction", params, &actions); err != nil {
		return nil, fmt.Errorf("code action request failed: %w", err)
	}

	return actions, nil
}

func (c *Client) WorkspaceSymbol(ctx context.Context, query string) ([]SymbolInformation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := WorkspaceSymbolParams{
		Query: query,
	}

	var result []SymbolInformation
	if err := c.conn.Call(ctx, "workspace/symbol", params, &result); err != nil {
		return nil, fmt.Errorf("workspace/symbol request failed: %w", err)
	}

	return result, nil
}

func (c *Client) Format(ctx context.Context, uri string) ([]TextEdit, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := DocumentFormattingParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Options: FormattingOptions{
			TabSize:      4,
			InsertSpaces: false, // Use tabs for Go
		},
	}

	var result []TextEdit
	if err := c.conn.Call(ctx, "textDocument/formatting", params, &result); err != nil {
		return nil, fmt.Errorf("formatting request failed: %w", err)
	}

	return result, nil
}

func (c *Client) OrganizeImports(ctx context.Context, uri string) ([]TextEdit, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		},
		Context: CodeActionContext{
			Only: []CodeActionKind{CodeActionKindSourceOrganizeImports},
		},
	}

	var result []CodeAction
	if err := c.conn.Call(ctx, "textDocument/codeAction", params, &result); err != nil {
		return nil, fmt.Errorf("codeAction request failed: %w", err)
	}

	// Extract edits from the first organize imports action
	for _, action := range result {
		if action.Kind == CodeActionKindSourceOrganizeImports && action.Edit != nil {
			for _, edits := range action.Edit.Changes {
				return edits, nil
			}
		}
	}

	return nil, nil
}
