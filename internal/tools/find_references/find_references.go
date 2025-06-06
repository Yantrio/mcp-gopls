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