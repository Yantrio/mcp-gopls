package hover

import (
	"context"
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