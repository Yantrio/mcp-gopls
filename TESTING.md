# Testing MCP-GOPLS with Claude

This document describes how to test the mcp-gopls server functionality using Claude.

## Prerequisites

1. Install mcp-gopls:
   ```bash
   go install github.com/yantrio/mcp-gopls/cmd/mcp-gopls@latest
   ```

2. Add to Claude:
   ```bash
   claude mcp add mcp-gopls mcp-gopls
   ```

3. Restart Claude to load the new MCP server

## Test Cases

### 1. Test GoToDefinition

Ask Claude to find the definition of a function or type in this repository:

```
"In the mcp-gopls repository, find the definition of the NewClient function"
```

Expected: Claude should use the GoToDefinition tool to locate where `NewClient` is defined (internal/lsp/client.go:26)

### 2. Test FindReferences

Ask Claude to find all uses of a type or function:

```
"Find all references to the Manager struct in the mcp-gopls codebase"
```

Expected: Claude should find references in:
- internal/gopls/manager.go (definition)
- internal/server/server.go (usage)
- internal/tools/tools.go (multiple usages)

### 3. Test GetDiagnostics

Ask Claude to check for compilation errors:

```
"Are there any compilation errors in internal/lsp/client.go?"
```

Expected: Claude should report no errors if the file compiles cleanly

### 4. Test Hover Information

Ask Claude to explain what a function does:

```
"What does the PathToURI function in internal/utils/uri.go do? Show me its signature"
```

Expected: Claude should use Hover to get the function signature and documentation

## Testing Multiple Tools Together

Ask Claude to perform analysis that requires multiple tools:

```
"I want to understand how the LSP client works in mcp-gopls. Can you:
1. Find the Client struct definition
2. Show me all the methods on the Client struct
3. Find where the Client is created and used
4. Check if there are any compilation errors in the client code"
```

This tests:
- GoToDefinition (finding the struct)
- Hover (getting method information)
- FindReferences (finding usage)
- GetDiagnostics (checking for errors)

## Common Issues to Test

### 1. File Path Handling
```
"Find the definition of Position in the types.go file"
```
Tests that the tool correctly handles file paths

### 2. Cross-File Navigation
```
"Find where the Tool type from mcp-go is used in this project"
```
Tests navigation to external dependencies

### 3. Error Handling
```
"Find the definition of NonExistentFunction"
```
Should gracefully handle when symbols don't exist

## Performance Testing

For larger codebases, test with:
```
"Find all references to Context in the entire project"
```

This tests how well the server handles multiple files and results.

## Debugging Tips

If tools aren't working:

1. Check if gopls is installed:
   ```bash
   gopls version
   ```

2. Verify mcp-gopls is in PATH:
   ```bash
   which mcp-gopls
   ```

3. Test mcp-gopls directly:
   ```bash
   echo '{}' | mcp-gopls
   ```
   Should output MCP protocol messages

4. Check Claude's MCP configuration:
   ```bash
   claude mcp list
   ```

## Expected Tool Behavior

- **GoToDefinition**: Returns file path, line, column, and preview of definition
- **FindReferences**: Returns all locations where a symbol is used
- **GetDiagnostics**: Returns compilation errors/warnings with severity
- **Hover**: Returns type signature and documentation
- Other tools return "Not implemented" (stubs for future work)