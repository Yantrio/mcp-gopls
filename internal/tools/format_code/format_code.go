package format_code

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/lsp"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

func NewTool(manager *gopls.Manager) mcp.Tool {
	return mcp.Tool{
		Name:        "FormatCode",
		Description: "Format Go source code according to gofmt standards",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Absolute path to the Go source file to format",
				},
			},
			Required: []string{"file"},
		},
	}
}

func NewHandler(manager *gopls.Manager) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		file, err := request.RequireString("file")
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

		if err := client.OpenDocument(ctx, uri, string(content)); err != nil {
			return nil, err
		}
		defer client.CloseDocument(ctx, uri)

		// Request formatting from gopls
		textEdits, err := client.DocumentFormatting(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("formatting request failed: %w", err)
		}

		if len(textEdits) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("File %s is already properly formatted", file)), nil
		}

		// Apply the formatting edits to the file
		if err := applyTextEdits(file, textEdits); err != nil {
			return nil, fmt.Errorf("failed to apply formatting: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully formatted %s", file)), nil
	}
}

// applyTextEdits applies LSP text edits to a file
func applyTextEdits(filePath string, edits []lsp.TextEdit) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// For formatting, typically there's only one edit that replaces the entire content
	// But let's handle the general case
	text := string(content)
	
	// Apply edits in reverse order to avoid offset issues
	for i := len(edits) - 1; i >= 0; i-- {
		edit := edits[i]
		
		// Calculate offsets
		startOffset, err := utils.CalculateOffset(text, edit.Range.Start)
		if err != nil {
			return fmt.Errorf("failed to calculate start offset: %w", err)
		}
		
		endOffset, err := utils.CalculateOffset(text, edit.Range.End)
		if err != nil {
			return fmt.Errorf("failed to calculate end offset: %w", err)
		}
		
		// Apply the edit
		text = text[:startOffset] + edit.NewText + text[endOffset:]
	}

	// Write back to file
	if err := os.WriteFile(filePath, []byte(text), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}