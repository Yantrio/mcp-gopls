package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/tools/diagnostics"
	"github.com/yantrio/mcp-gopls/internal/tools/find_implementers"
	"github.com/yantrio/mcp-gopls/internal/tools/find_references"
	"github.com/yantrio/mcp-gopls/internal/tools/format_code"
	"github.com/yantrio/mcp-gopls/internal/tools/goto_definition"
	"github.com/yantrio/mcp-gopls/internal/tools/hover"
	"github.com/yantrio/mcp-gopls/internal/tools/list_document_symbols"
	"github.com/yantrio/mcp-gopls/internal/tools/rename"
	"github.com/yantrio/mcp-gopls/internal/tools/stubs"
)

// GetTools returns all available tools
func GetTools(manager *gopls.Manager) []mcp.Tool {
	return []mcp.Tool{
		goto_definition.NewTool(manager),
		find_references.NewTool(manager),
		diagnostics.NewTool(manager),
		hover.NewTool(manager),
		rename.NewTool(manager),
		find_implementers.NewTool(manager),
		list_document_symbols.NewTool(manager),
		stubs.NewSearchSymbolTool(manager),
		format_code.NewTool(manager),
		stubs.NewOrganizeImportsTool(manager),
	}
}

// GetToolHandlers returns all tool handlers
func GetToolHandlers(manager *gopls.Manager) map[string]server.ToolHandlerFunc {
	return map[string]server.ToolHandlerFunc{
		"GoToDefinition":      goto_definition.NewHandler(manager),
		"FindReferences":      find_references.NewHandler(manager),
		"GetDiagnostics":      diagnostics.NewHandler(manager),
		"Hover":               hover.NewHandler(manager),
		"RenameSymbol":        rename.NewHandler(manager),
		"FindImplementers":    find_implementers.NewHandler(manager),
		"ListDocumentSymbols": list_document_symbols.NewHandler(manager),
		"SearchSymbol":        stubs.NewSearchSymbolHandler(manager),
		"FormatCode":          format_code.NewHandler(manager),
		"OrganizeImports":     stubs.NewOrganizeImportsHandler(manager),
	}
}