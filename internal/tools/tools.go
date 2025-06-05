package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

// GoToDefinition implementation
func NewGoToDefinitionTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "GoToDefinition",
		Description: "Navigate to the definition of a symbol at a given position",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Absolute path to the Go source file",
				},
				"line": map[string]interface{}{
					"type":        "number",
					"description": "Line number (1-indexed)",
				},
				"column": map[string]interface{}{
					"type":        "number",
					"description": "Column number (1-indexed)",
				},
			},
			Required: []string{"file", "line", "column"},
		},
	}
}

func NewGoToDefinitionHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		// Parse arguments
		args, err := json.Marshal(arguments)
		if err != nil {
			return nil, err
		}

		var input struct {
			File   string `json:"file"`
			Line   int    `json:"line"`
			Column int    `json:"column"`
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

		position := utils.ConvertPosition(input.Line, input.Column)
		locations, err := client.Definition(ctx, uri, position)
		if err != nil {
			return nil, err
		}

		definitions := make([]map[string]interface{}, 0)
		for _, loc := range locations {
			defPath, err := utils.URIToPath(loc.URI)
			if err != nil {
				continue
			}

			defLine, defColumn := utils.ConvertToUserPosition(loc.Range.Start)

			preview := ""
			if defContent, err := os.ReadFile(defPath); err == nil {
				lines := strings.Split(string(defContent), "\n")
				if defLine <= len(lines) {
					preview = strings.TrimSpace(lines[defLine-1])
				}
			}

			definitions = append(definitions, map[string]interface{}{
				"file":    defPath,
				"line":    defLine,
				"column":  defColumn,
				"preview": preview,
			})
		}

		// Format result as JSON
		result, _ := json.MarshalIndent(definitions, "", "  ")

		return mcp.NewToolResultText(fmt.Sprintf("Found %d definition(s):\n%s", len(definitions), string(result))), nil
	}
}

// FindReferences implementation
func NewFindReferencesTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "FindReferences",
		Description: "Find all references to a symbol at a given position",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Absolute path to the Go source file",
				},
				"line": map[string]interface{}{
					"type":        "number",
					"description": "Line number (1-indexed)",
				},
				"column": map[string]interface{}{
					"type":        "number",
					"description": "Column number (1-indexed)",
				},
				"includeDeclaration": map[string]interface{}{
					"type":        "boolean",
					"description": "Include the declaration in results",
					"default":     false,
				},
			},
			Required: []string{"file", "line", "column"},
		},
	}
}

func NewFindReferencesHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		args, _ := json.Marshal(arguments)

		var input struct {
			File               string `json:"file"`
			Line               int    `json:"line"`
			Column             int    `json:"column"`
			IncludeDeclaration bool   `json:"includeDeclaration"`
		}
		json.Unmarshal(args, &input)

		client, _ := manager.GetClient()
		uri, _ := utils.PathToURI(input.File)
		content, _ := os.ReadFile(input.File)

		ctx := context.Background()
		client.OpenDocument(ctx, uri, string(content))
		defer client.CloseDocument(ctx, uri)

		position := utils.ConvertPosition(input.Line, input.Column)
		locations, err := client.References(ctx, uri, position, input.IncludeDeclaration)
		if err != nil {
			return nil, err
		}

		references := make([]map[string]interface{}, 0)
		for _, loc := range locations {
			refPath, _ := utils.URIToPath(loc.URI)
			refLine, refColumn := utils.ConvertToUserPosition(loc.Range.Start)

			preview := ""
			if refContent, err := os.ReadFile(refPath); err == nil {
				lines := strings.Split(string(refContent), "\n")
				if refLine <= len(lines) {
					preview = strings.TrimSpace(lines[refLine-1])
				}
			}

			references = append(references, map[string]interface{}{
				"file":    refPath,
				"line":    refLine,
				"column":  refColumn,
				"preview": preview,
			})
		}

		result, _ := json.MarshalIndent(references, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Found %d reference(s):\n%s", len(references), string(result))), nil
	}
}

// GetDiagnostics implementation
func NewGetDiagnosticsTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "GetDiagnostics",
		Description: "Get compile errors and static analysis findings for a file",
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

func NewGetDiagnosticsHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		args, _ := json.Marshal(arguments)

		var input struct {
			File string `json:"file"`
		}
		json.Unmarshal(args, &input)

		client, _ := manager.GetClient()
		uri, _ := utils.PathToURI(input.File)
		content, _ := os.ReadFile(input.File)

		ctx := context.Background()
		client.OpenDocument(ctx, uri, string(content))
		defer client.CloseDocument(ctx, uri)

		lspDiagnostics := client.GetDiagnostics(uri)

		diagnostics := make([]map[string]interface{}, 0)
		for _, diag := range lspDiagnostics {
			startLine, startColumn := utils.ConvertToUserPosition(diag.Range.Start)
			endLine, endColumn := utils.ConvertToUserPosition(diag.Range.End)

			severity := "error"
			switch diag.Severity {
			case 1:
				severity = "error"
			case 2:
				severity = "warning"
			case 3:
				severity = "information"
			case 4:
				severity = "hint"
			}

			diagnostics = append(diagnostics, map[string]interface{}{
				"severity":  severity,
				"message":   diag.Message,
				"line":      startLine,
				"column":    startColumn,
				"endLine":   endLine,
				"endColumn": endColumn,
			})
		}

		result, _ := json.MarshalIndent(diagnostics, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Found %d diagnostic(s):\n%s", len(diagnostics), string(result))), nil
	}
}

// Hover implementation
func NewHoverTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "Hover",
		Description: "Get information about the symbol under the cursor",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Absolute path to the Go source file",
				},
				"line": map[string]interface{}{
					"type":        "number",
					"description": "Line number (1-indexed)",
				},
				"column": map[string]interface{}{
					"type":        "number",
					"description": "Column number (1-indexed)",
				},
			},
			Required: []string{"file", "line", "column"},
		},
	}
}

func NewHoverHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		args, _ := json.Marshal(arguments)

		var input struct {
			File   string `json:"file"`
			Line   int    `json:"line"`
			Column int    `json:"column"`
		}
		json.Unmarshal(args, &input)

		client, _ := manager.GetClient()
		uri, _ := utils.PathToURI(input.File)
		content, _ := os.ReadFile(input.File)

		ctx := context.Background()
		client.OpenDocument(ctx, uri, string(content))
		defer client.CloseDocument(ctx, uri)

		position := utils.ConvertPosition(input.Line, input.Column)
		hover, err := client.Hover(ctx, uri, position)
		if err != nil {
			return nil, err
		}

		if hover == nil {
			return mcp.NewToolResultText("No hover information available"), nil
		}

		return mcp.NewToolResultText(hover.Contents.Value), nil
	}
}

// Stub implementations for remaining tools
func NewRenameSymbolTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "RenameSymbol",
		Description: "Rename a symbol across the workspace",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Absolute path to the Go source file",
				},
				"line": map[string]interface{}{
					"type":        "number",
					"description": "Line number (1-indexed)",
				},
				"column": map[string]interface{}{
					"type":        "number",
					"description": "Column number (1-indexed)",
				},
				"newName": map[string]interface{}{
					"type":        "string",
					"description": "New name for the symbol",
				},
			},
			Required: []string{"file", "line", "column", "newName"},
		},
	}
}

func NewRenameSymbolHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Rename functionality not yet implemented"), nil
	}
}

func NewFindImplementersTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "FindImplementers",
		Description: "Find all types that implement an interface",
		InputSchema: mcp.ToolInputSchema{Type: "object"},
	}
}

func NewFindImplementersHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
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
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Not implemented"), nil
	}
}

func NewSearchSymbolTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "SearchSymbol",
		Description: "Search for symbols by name across the workspace",
		InputSchema: mcp.ToolInputSchema{Type: "object"},
	}
}

func NewSearchSymbolHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Not implemented"), nil
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
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
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
	return func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Not implemented"), nil
	}
}
