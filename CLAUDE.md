# Development Guidelines for MCP-GOPLS

## Project Information
- **Module**: `github.com/yantrio/mcp-gopls`
- **Purpose**: MCP server that wraps gopls to provide Go language server features

## Git Commit Guidelines
- Make frequent, atomic commits for each major feature
- Use simple, clear commit messages
- Example: "Add LSP client implementation"
- Commit after completing each significant component

## Development Workflow
1. Follow the design document (DESIGN.md)
2. Implement features incrementally
3. Test each component before moving to the next
4. Commit after each working feature

## Code Quality
- Run `go fmt` before committing
- Ensure all tests pass
- Follow Go best practices and idioms

## Testing Commands
- `go test ./...` - Run all tests
- `go fmt ./...` - Format all code
- `go vet ./...` - Run static analysis

## Key Implementation Notes
- Use `github.com/sourcegraph/jsonrpc2` for JSON-RPC communication
- Follow the directory structure defined in DESIGN.md
- Keep internal packages private, only expose necessary APIs