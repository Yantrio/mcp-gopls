package hover

import (
	"context"
	"encoding/json"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

func NewTool(manager *gopls.Manager) mcp.Tool {
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

func NewHandler(manager *gopls.Manager) server.ToolHandlerFunc {
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