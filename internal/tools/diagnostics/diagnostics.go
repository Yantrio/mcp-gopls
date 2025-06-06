package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

func NewTool(manager *gopls.Manager) mcp.Tool {
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

func NewHandler(manager *gopls.Manager) server.ToolHandlerFunc {
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