package list_document_symbols

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/lsp"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

func NewTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "ListDocumentSymbols",
		Description: "Get an outline of symbols defined in the current file",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Absolute path to the Go source file",
				},
			},
			Required: []string{"file"},
		},
	}
}

func NewHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		// Parse arguments
		args, err := json.Marshal(arguments)
		if err != nil {
			return nil, err
		}

		var input struct {
			File string `json:"file"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			return nil, err
		}

		client, err := manager.GetClient()
		if err != nil {
			return nil, err
		}

		uri, err := utils.PathToURI(input.File)
		if err != nil {
			return nil, err
		}

		content, err := os.ReadFile(input.File)
		if err != nil {
			return nil, err
		}

		ctx := context.Background()
		if err := client.OpenDocument(ctx, uri, string(content)); err != nil {
			return nil, err
		}
		defer client.CloseDocument(ctx, uri)

		symbols, err := client.DocumentSymbols(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("document symbols request failed: %w", err)
		}

		if len(symbols) == 0 {
			return mcp.NewToolResultText("No symbols found in the document"), nil
		}

		// Convert symbols to human-readable format
		results := make([]string, 0)
		formatSymbols(symbols, "", &results)

		// Format as a tree structure
		return mcp.NewToolResultText(fmt.Sprintf("Document symbols for %s:\n\n%s", input.File, strings.Join(results, "\n"))), nil
	}
}

// formatSymbols recursively formats document symbols into a tree structure
func formatSymbols(symbols []lsp.DocumentSymbol, indent string, results *[]string) {
	for i, symbol := range symbols {
		// Determine the tree character
		treeChar := "├── "
		if i == len(symbols)-1 {
			treeChar = "└── "
		}

		// Format the symbol
		line := fmt.Sprintf("%s%s%s %s",
			indent,
			treeChar,
			getSymbolIcon(symbol.Kind),
			symbol.Name)

		// Add detail if available
		if symbol.Detail != "" {
			line += fmt.Sprintf(" (%s)", symbol.Detail)
		}

		// Add line number
		startLine, _ := utils.ConvertToUserPosition(symbol.Range.Start)
		line += fmt.Sprintf(" [line %d]", startLine)

		*results = append(*results, line)

		// Process children
		if len(symbol.Children) > 0 {
			childIndent := indent
			if i == len(symbols)-1 {
				childIndent += "    "
			} else {
				childIndent += "│   "
			}
			formatSymbols(symbol.Children, childIndent, results)
		}
	}
}

// getSymbolIcon returns a text indicator for the symbol kind
func getSymbolIcon(kind lsp.SymbolKind) string {
	switch kind {
	case lsp.SymbolKindFile:
		return "[file]"
	case lsp.SymbolKindModule:
		return "[module]"
	case lsp.SymbolKindNamespace:
		return "[namespace]"
	case lsp.SymbolKindPackage:
		return "[package]"
	case lsp.SymbolKindClass:
		return "[class]"
	case lsp.SymbolKindMethod:
		return "[method]"
	case lsp.SymbolKindProperty:
		return "[property]"
	case lsp.SymbolKindField:
		return "[field]"
	case lsp.SymbolKindConstructor:
		return "[constructor]"
	case lsp.SymbolKindEnum:
		return "[enum]"
	case lsp.SymbolKindInterface:
		return "[interface]"
	case lsp.SymbolKindFunction:
		return "[func]"
	case lsp.SymbolKindVariable:
		return "[var]"
	case lsp.SymbolKindConstant:
		return "[const]"
	case lsp.SymbolKindString:
		return "[string]"
	case lsp.SymbolKindNumber:
		return "[number]"
	case lsp.SymbolKindBoolean:
		return "[bool]"
	case lsp.SymbolKindArray:
		return "[array]"
	case lsp.SymbolKindObject:
		return "[object]"
	case lsp.SymbolKindKey:
		return "[key]"
	case lsp.SymbolKindNull:
		return "[null]"
	case lsp.SymbolKindEnumMember:
		return "[enum-member]"
	case lsp.SymbolKindStruct:
		return "[struct]"
	case lsp.SymbolKindEvent:
		return "[event]"
	case lsp.SymbolKindOperator:
		return "[operator]"
	case lsp.SymbolKindTypeParameter:
		return "[type-param]"
	default:
		return "[unknown]"
	}
}