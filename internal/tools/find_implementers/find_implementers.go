package find_implementers

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
		Name:        "FindImplementers",
		Description: "Find all types that implement an interface",
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
		locations, err := client.Implementation(ctx, uri, position)
		if err != nil {
			return nil, fmt.Errorf("implementation request failed: %w", err)
		}

		if len(locations) == 0 {
			return mcp.NewToolResultText("No implementations found"), nil
		}

		// Convert locations to human-readable format
		results := make([]map[string]interface{}, 0)
		for _, loc := range locations {
			locPath, err := utils.URIToPath(loc.URI)
			if err != nil {
				continue
			}

			startLine, startColumn := utils.ConvertToUserPosition(loc.Range.Start)
			
			// Read the line to get context
			fileContent, err := os.ReadFile(locPath)
			if err != nil {
				continue
			}
			
			lines := string(fileContent)
			lineText := ""
			currentLine := 1
			lineStart := 0
			for i, ch := range lines {
				if ch == '\n' {
					if currentLine == startLine {
						lineText = lines[lineStart:i]
						break
					}
					currentLine++
					lineStart = i + 1
				}
			}
			if currentLine == startLine && lineText == "" {
				lineText = lines[lineStart:]
			}

			results = append(results, map[string]interface{}{
				"file":    locPath,
				"line":    startLine,
				"column":  startColumn,
				"preview": lineText,
			})
		}

		// Format as JSON
		result, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(fmt.Sprintf("Found %d implementation(s):\n%s", len(results), string(result))), nil
	}
}