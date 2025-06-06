package goto_definition

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

func NewTool(manager *gopls.Manager) mcp.Tool {
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

func NewHandler(manager *gopls.Manager) server.ToolHandlerFunc {
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

		result, _ := json.MarshalIndent(definitions, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	}
}