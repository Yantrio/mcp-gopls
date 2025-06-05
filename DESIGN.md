# MCP-GOPLS Design Document

## Overview

This document outlines the design for an MCP (Model Context Protocol) server that wraps the Go Language Server (gopls) to expose its functionality through MCP tools. The server will provide programmatic access to gopls features including code navigation, diagnostics, refactoring, and code organization capabilities.

### Goals

- Provide a clean MCP interface to gopls functionality
- Support multiple workspace directories
- Enable AI models to interact with Go codebases effectively
- Maintain gopls performance and accuracy

### Non-Goals

- Reimplementing gopls functionality
- Supporting non-Go languages
- Providing a GUI or editor interface

## Architecture

### High-Level Design

```
┌─────────────┐     ┌─────────────────────────────────────┐     ┌─────────────┐
│   MCP       │     │           MCP-GOPLS Server          │     │    GOPLS    │
│  Client     │────▶│  ┌─────────────┐  ┌─────────────┐  │────▶│   Process   │
└─────────────┘     │  │ Tool Handler│  │ LSP Client  │  │     └─────────────┘
                    │  └─────────────┘  └─────────────┘  │
                    └─────────────────────────────────────┘
       │                       │                 │                      │
       │   MCP Protocol        │                 │    LSP Protocol     │
       └───────────────────────┴─────────────────┴──────────────────────┘
```

### Components

1. **MCP Server**: Handles MCP protocol communication
2. **LSP Client**: Abstracts LSP protocol complexity and provides a clean interface
3. **Tool Handlers**: Translate MCP tool calls to LSP client methods
4. **GOPLS Manager**: Manages gopls process lifecycle
5. **File URI Manager**: Handles file path to URI conversions

### LSP Client Design

The LSP Client serves as an abstraction layer that:
- Manages the JSON-RPC communication with gopls
- Handles request/response correlation
- Provides type-safe methods for each LSP operation
- Manages document state and synchronization
- Implements error handling and retries

Example interface:
```go
type LSPClient interface {
    Initialize(workspaceRoot string) error
    Shutdown() error
    
    // Document operations
    OpenDocument(uri string, content string) error
    CloseDocument(uri string) error
    
    // Language features
    Definition(uri string, position Position) ([]Location, error)
    References(uri string, position Position, includeDecl bool) ([]Location, error)
    Hover(uri string, position Position) (*HoverResult, error)
    Rename(uri string, position Position, newName string) (*WorkspaceEdit, error)
    // ... other methods
}
```

### Dependencies

- `mark3lbs/mcp-go`: MCP server framework
- `golang.org/x/tools/gopls`: Go language server
- `github.com/sourcegraph/jsonrpc2`: JSON-RPC client for LSP communication
- Standard Go libraries for process management

## Project Structure

Following Go best practices, the project should be organized as:

```
mcp-gopls/
├── cmd/
│   └── mcp-gopls/
│       └── main.go              # Entry point
├── internal/
│   ├── lsp/
│   │   ├── client.go           # LSP client implementation
│   │   ├── types.go            # LSP protocol types
│   │   ├── jsonrpc.go          # JSON-RPC communication
│   │   └── client_test.go      # LSP client tests
│   ├── gopls/
│   │   ├── manager.go          # GOPLS process management
│   │   └── manager_test.go     
│   ├── tools/                  # MCP tool implementations
│   │   ├── definition.go       # GoToDefinition tool
│   │   ├── references.go       # FindReferences tool
│   │   ├── diagnostics.go      # GetDiagnostics tool
│   │   ├── hover.go            # Hover tool
│   │   ├── rename.go           # RenameSymbol tool
│   │   ├── implementations.go  # FindImplementers tool
│   │   ├── symbols.go          # Document/Workspace symbols
│   │   ├── formatting.go       # Format and organize imports
│   │   └── tools_test.go       
│   ├── server/
│   │   ├── server.go           # MCP server setup
│   │   └── config.go           # Configuration handling
│   └── utils/
│       ├── uri.go              # URI/path conversions
│       └── position.go         # Position calculations
├── pkg/
│   └── types/
│       └── types.go            # Shared types for API
├── configs/
│   └── config.yaml             # Default configuration
├── scripts/
│   ├── install.sh              # Installation script
│   └── test.sh                 # Test runner
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── DESIGN.md                   # This document
└── LICENSE
```

### Package Guidelines

1. **cmd/**: Contains the main application entry point
2. **internal/**: Private application code that cannot be imported by other projects
   - `lsp/`: LSP client and protocol handling
   - `gopls/`: GOPLS process management
   - `tools/`: Individual MCP tool implementations
   - `server/`: MCP server setup and configuration
   - `utils/`: Shared utilities
3. **pkg/**: Public packages that can be imported (if needed)
4. **configs/**: Configuration files
5. **scripts/**: Build and deployment scripts

## Tool Specifications

### 1. GoToDefinition

**Description**: Navigate to the definition of a symbol at a given position.

**Parameters**:
- `file`: string - Absolute path to the Go source file
- `line`: number - Line number (1-indexed)
- `column`: number - Column number (1-indexed)

**Returns**:
```json
{
  "definitions": [
    {
      "file": "path/to/file.go",
      "line": 42,
      "column": 15,
      "preview": "func MyFunction() error {"
    }
  ]
}
```

**LSP Method**: `textDocument/definition`

### 2. FindReferences

**Description**: Find all references to a symbol at a given position.

**Parameters**:
- `file`: string - Absolute path to the Go source file
- `line`: number - Line number (1-indexed)
- `column`: number - Column number (1-indexed)
- `includeDeclaration`: boolean - Include the declaration in results

**Returns**:
```json
{
  "references": [
    {
      "file": "path/to/file.go",
      "line": 10,
      "column": 5,
      "preview": "result := MyFunction()"
    }
  ]
}
```

**LSP Method**: `textDocument/references`

### 3. GetDiagnostics

**Description**: Get compile errors and static analysis findings for a file.

**Parameters**:
- `file`: string - Absolute path to the Go source file

**Returns**:
```json
{
  "diagnostics": [
    {
      "severity": "error",
      "message": "undefined: someVariable",
      "line": 25,
      "column": 10,
      "endLine": 25,
      "endColumn": 22
    }
  ]
}
```

**LSP Method**: `textDocument/publishDiagnostics` (passive notification)

### 4. Hover

**Description**: Get information about the symbol under the cursor.

**Parameters**:
- `file`: string - Absolute path to the Go source file
- `line`: number - Line number (1-indexed)
- `column`: number - Column number (1-indexed)

**Returns**:
```json
{
  "content": "func fmt.Printf(format string, a ...interface{}) (n int, err error)",
  "documentation": "Printf formats according to a format specifier and writes to standard output.",
  "signature": "func(format string, a ...interface{}) (n int, err error)"
}
```

**LSP Method**: `textDocument/hover`

### 5. RenameSymbol

**Description**: Rename a symbol across the workspace.

**Parameters**:
- `file`: string - Absolute path to the Go source file
- `line`: number - Line number (1-indexed)
- `column`: number - Column number (1-indexed)
- `newName`: string - New name for the symbol

**Returns**:
```json
{
  "changes": [
    {
      "file": "path/to/file.go",
      "edits": [
        {
          "line": 10,
          "column": 5,
          "endLine": 10,
          "endColumn": 15,
          "newText": "newSymbolName"
        }
      ]
    }
  ]
}
```

**LSP Method**: `textDocument/rename`

### 6. FindImplementers

**Description**: Find all types that implement an interface.

**Parameters**:
- `file`: string - Absolute path to the Go source file
- `line`: number - Line number (1-indexed)
- `column`: number - Column number (1-indexed)

**Returns**:
```json
{
  "implementations": [
    {
      "file": "path/to/impl.go",
      "line": 30,
      "column": 6,
      "typeName": "MyStruct",
      "preview": "type MyStruct struct {"
    }
  ]
}
```

**LSP Method**: `textDocument/implementation`

### 7. ListDocumentSymbols

**Description**: Get an outline of symbols defined in the current file.

**Parameters**:
- `file`: string - Absolute path to the Go source file

**Returns**:
```json
{
  "symbols": [
    {
      "name": "MyFunction",
      "kind": "function",
      "line": 10,
      "column": 1,
      "children": []
    },
    {
      "name": "MyStruct",
      "kind": "struct",
      "line": 20,
      "column": 1,
      "children": [
        {
          "name": "Field1",
          "kind": "field",
          "line": 21,
          "column": 2
        }
      ]
    }
  ]
}
```

**LSP Method**: `textDocument/documentSymbol`

### 8. SearchSymbol

**Description**: Search for symbols by name across the workspace.

**Parameters**:
- `query`: string - Search query (supports fuzzy matching)
- `kind`: string (optional) - Symbol kind filter ("function", "type", "variable", etc.)

**Returns**:
```json
{
  "symbols": [
    {
      "name": "MyFunction",
      "kind": "function",
      "file": "path/to/file.go",
      "line": 10,
      "column": 1,
      "containerName": "main"
    }
  ]
}
```

**LSP Method**: `workspace/symbol`

### 9. FormatCode

**Description**: Format Go source code according to gofmt standards.

**Parameters**:
- `file`: string - Absolute path to the Go source file

**Returns**:
```json
{
  "formatted": true,
  "edits": [
    {
      "line": 5,
      "column": 1,
      "endLine": 5,
      "endColumn": 10,
      "newText": "\t"
    }
  ]
}
```

**LSP Method**: `textDocument/formatting`

### 10. OrganizeImports

**Description**: Organize import statements (add missing, remove unused, sort).

**Parameters**:
- `file`: string - Absolute path to the Go source file

**Returns**:
```json
{
  "organized": true,
  "edits": [
    {
      "line": 3,
      "column": 1,
      "endLine": 5,
      "endColumn": 1,
      "newText": "import (\n\t\"fmt\"\n\t\"strings\"\n)\n"
    }
  ]
}
```

**LSP Method**: `textDocument/codeAction` (with source.organizeImports)

## Implementation Details

### LSP Client Implementation

The LSP client will leverage an existing JSON-RPC library to handle the protocol complexity:

**Recommended JSON-RPC Libraries**:
1. **github.com/sourcegraph/jsonrpc2** - Battle-tested, used by many LSP implementations
2. **github.com/creachadair/jrpc2** - Modern, feature-rich with good error handling
3. **go.lsp.dev/jsonrpc2** - Specifically designed for LSP implementations

Using `github.com/sourcegraph/jsonrpc2` as an example:

```go
// internal/lsp/client.go
type Client struct {
    process  *exec.Cmd
    conn     *jsonrpc2.Conn
    capabilities ServerCapabilities
}

func NewClient() (*Client, error) {
    cmd := exec.Command("gopls", "serve")
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    stream := jsonrpc2.NewBufferedStream(
        readWriteCloser{stdout, stdin},
        jsonrpc2.VSCodeObjectCodec{},
    )
    
    client := &Client{process: cmd}
    client.conn = jsonrpc2.NewConn(
        context.Background(),
        stream,
        client, // Handler for server-initiated requests
    )
    
    return client, nil
}

func (c *Client) Initialize(rootURI string) error {
    var result InitializeResult
    err := c.conn.Call(
        context.Background(),
        "initialize",
        InitializeParams{
            RootURI: rootURI,
            Capabilities: ClientCapabilities{
                // ... capabilities
            },
        },
        &result,
    )
    if err != nil {
        return err
    }
    
    c.capabilities = result.Capabilities
    
    // Send initialized notification
    return c.conn.Notify(
        context.Background(),
        "initialized",
        &InitializedParams{},
    )
}

// High-level methods that hide JSON-RPC complexity
func (c *Client) Definition(uri string, pos Position) ([]Location, error) {
    var locations []Location
    err := c.conn.Call(
        context.Background(),
        "textDocument/definition",
        DefinitionParams{
            TextDocumentPositionParams: TextDocumentPositionParams{
                TextDocument: TextDocumentIdentifier{URI: uri},
                Position:     pos,
            },
        },
        &locations,
    )
    return locations, err
}
```

**Benefits of using an off-the-shelf JSON-RPC client**:
1. **Proven reliability**: These libraries are battle-tested in production
2. **Protocol compliance**: Handles JSON-RPC 2.0 spec correctly
3. **Built-in features**: Request IDs, error handling, concurrent requests
4. **Less code to maintain**: Focus on LSP logic, not protocol details
5. **Better debugging**: Most libraries have excellent logging support

### GOPLS Process Management

1. **Initialization**:
   - Start gopls process with `gopls serve` command
   - Initialize LSP connection with workspace folders
   - Wait for initialization confirmation

2. **Connection**:
   - Use stdin/stdout for JSON-RPC communication
   - Implement request/response correlation with unique IDs
   - Handle notifications separately

3. **Lifecycle**:
   - Start gopls on first tool use
   - Keep alive for session duration
   - Graceful shutdown on server termination

### File Management

1. **URI Conversion**:
   ```go
   func filePathToURI(path string) string {
       return "file://" + filepath.ToSlash(filepath.Clean(path))
   }
   ```

2. **Document Synchronization**:
   - Open documents before operations
   - Track open document state
   - Close documents when no longer needed

### Error Handling

1. **GOPLS Errors**:
   - Parse LSP error responses
   - Map to user-friendly messages
   - Include diagnostic context

2. **File System Errors**:
   - Validate file existence
   - Check read permissions
   - Handle workspace boundaries

3. **Protocol Errors**:
   - Timeout handling for LSP requests
   - Retry logic for transient failures
   - Graceful degradation

### Performance Considerations

1. **Caching**:
   - Cache file contents for repeated operations
   - Invalidate on file system changes
   - Limit cache size

2. **Batching**:
   - Group related operations
   - Minimize round trips to gopls
   - Efficient document state management

3. **Resource Limits**:
   - Limit concurrent gopls processes
   - Monitor memory usage
   - Implement request throttling

## Configuration

### Server Configuration

```yaml
server:
  name: "mcp-gopls"
  description: "MCP server providing gopls functionality"
  version: "1.0.0"

gopls:
  path: "gopls"  # Path to gopls binary
  args: ["serve"]
  timeout: 30s   # Request timeout
  
workspace:
  root: "/path/to/workspace"
  followSymlinks: true
```

### Environment Variables

- `GOPLS_PATH`: Override gopls binary location
- `MCP_GOPLS_WORKSPACE`: Default workspace directory
- `MCP_GOPLS_LOG_LEVEL`: Logging verbosity (debug, info, warn, error)

## Security Considerations

1. **File Access**:
   - Validate all file paths
   - Prevent directory traversal
   - Respect workspace boundaries

2. **Process Isolation**:
   - Run gopls with limited permissions
   - Sanitize all inputs
   - Prevent command injection

3. **Resource Protection**:
   - Limit memory usage
   - Timeout long-running operations
   - Rate limit requests

## Testing Strategy

1. **Unit Tests**:
   - Tool handler logic
   - URI conversion
   - Error handling

2. **Integration Tests**:
   - Full tool workflows
   - GOPLS communication
   - Edge cases

3. **Performance Tests**:
   - Large codebases
   - Concurrent requests
   - Memory usage

## Future Enhancements

1. **Additional Tools**:
   - Code completion
   - Quick fixes
   - Call hierarchy

2. **Performance**:
   - Persistent gopls daemon
   - Distributed caching
   - Parallel processing

3. **Features**:
   - Multi-module support
   - Custom analyzers
   - Workspace-wide refactoring