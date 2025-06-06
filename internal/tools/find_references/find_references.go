package find_references

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

func NewTool(manager *gopls.Manager) mcp.Tool {
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

func NewHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		file, err := request.RequireString("file")
		if err != nil {
			return nil, err
		}
		line, err := request.RequireInt("line")
		if err != nil {
			return nil, err
		}
		column, err := request.RequireInt("column")
		if err != nil {
			return nil, err
		}
		includeDeclaration := request.GetBool("includeDeclaration", false)

		client, err := manager.GetClient()
		if err != nil {
			return nil, err
		}
		uri, err := utils.PathToURI(file)
		if err != nil {
			return nil, err
		}
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		err = client.OpenDocument(ctx, uri, string(content))
		if err != nil {
			return nil, err
		}
		defer client.CloseDocument(ctx, uri)

		position := utils.ConvertPosition(line, column)
		locations, err := client.References(ctx, uri, position, includeDeclaration)
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