package stubs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/lsp"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

func NewFindImplementersTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "FindImplementers",
		Description: "Find all types that implement an interface",
		InputSchema: mcp.ToolInputSchema{Type: "object"},
	}
}

func NewFindImplementersHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Not implemented"), nil
	}
}

func NewListDocumentSymbolsTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "ListDocumentSymbols",
		Description: "Get an outline of symbols defined in the current file",
		InputSchema: mcp.ToolInputSchema{Type: "object"},
	}
}

func NewListDocumentSymbolsHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Not implemented"), nil
	}
}

func NewSearchSymbolTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "SearchSymbol",
		Description: "Search for symbols by name across the workspace",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Symbol name to search for (supports partial matching)",
				},
			},
			Required: []string{"query"},
		},
	}
}

func NewSearchSymbolHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := request.RequireString("query")
		if err != nil {
			return nil, err
		}

		if query == "" {
			return nil, fmt.Errorf("query cannot be empty")
		}

		client, err := manager.GetClient()
		if err != nil {
			return nil, err
		}

		symbols, err := client.WorkspaceSymbol(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("workspace symbol search failed: %w", err)
		}

		results := make([]map[string]interface{}, 0)
		for _, symbol := range symbols {
			symPath, err := utils.URIToPath(symbol.Location.URI)
			if err != nil {
				continue
			}

			symLine, symColumn := utils.ConvertToUserPosition(symbol.Location.Range.Start)

			symbolKind := getSymbolKindName(symbol.Kind)

			results = append(results, map[string]interface{}{
				"name":          symbol.Name,
				"kind":          symbolKind,
				"file":          symPath,
				"line":          symLine,
				"column":        symColumn,
				"containerName": symbol.ContainerName,
			})
		}

		result, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Found %d symbol(s):\n%s", len(results), string(result))), nil
	}
}

func NewFormatCodeTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "FormatCode",
		Description: "Format Go source code according to gofmt standards",
		InputSchema: mcp.ToolInputSchema{Type: "object"},
	}
}

func NewFormatCodeHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Not implemented"), nil
	}
}

func NewOrganizeImportsTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "OrganizeImports",
		Description: "Organize import statements",
		InputSchema: mcp.ToolInputSchema{Type: "object"},
	}
}

func NewOrganizeImportsHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Not implemented"), nil
	}
}

// getSymbolKindName converts a SymbolKind to a human-readable string
func getSymbolKindName(kind lsp.SymbolKind) string {
	switch kind {
	case lsp.SymbolKindFile:
		return "file"
	case lsp.SymbolKindModule:
		return "module"
	case lsp.SymbolKindNamespace:
		return "namespace"
	case lsp.SymbolKindPackage:
		return "package"
	case lsp.SymbolKindClass:
		return "class"
	case lsp.SymbolKindMethod:
		return "method"
	case lsp.SymbolKindProperty:
		return "property"
	case lsp.SymbolKindField:
		return "field"
	case lsp.SymbolKindConstructor:
		return "constructor"
	case lsp.SymbolKindEnum:
		return "enum"
	case lsp.SymbolKindInterface:
		return "interface"
	case lsp.SymbolKindFunction:
		return "function"
	case lsp.SymbolKindVariable:
		return "variable"
	case lsp.SymbolKindConstant:
		return "constant"
	case lsp.SymbolKindString:
		return "string"
	case lsp.SymbolKindNumber:
		return "number"
	case lsp.SymbolKindBoolean:
		return "boolean"
	case lsp.SymbolKindArray:
		return "array"
	case lsp.SymbolKindObject:
		return "object"
	case lsp.SymbolKindKey:
		return "key"
	case lsp.SymbolKindNull:
		return "null"
	case lsp.SymbolKindEnumMember:
		return "enumMember"
	case lsp.SymbolKindStruct:
		return "struct"
	case lsp.SymbolKindEvent:
		return "event"
	case lsp.SymbolKindOperator:
		return "operator"
	case lsp.SymbolKindTypeParameter:
		return "typeParameter"
	default:
		return "unknown"
	}
}