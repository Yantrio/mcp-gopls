package rename

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/lsp"
	"github.com/yantrio/mcp-gopls/internal/utils"
)

func NewTool(manager *gopls.Manager) mcp.Tool {
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
		newName, err := request.RequireString("newName")
		if err != nil {
			return nil, err
		}

		if newName == "" {
			return nil, fmt.Errorf("newName cannot be empty")
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

		position := utils.ConvertPosition(line, column)
		
		// First, check if rename is possible at this location
		prepareResult, prepareErr := client.PrepareRename(ctx, uri, position)
		if prepareErr != nil {
			// If prepareRename fails, it might mean rename is not supported at this location
			// Let's still try the rename operation
		}
		
		// Debug info
		debugInfo := fmt.Sprintf("Debug - Position: line=%d, col=%d (0-indexed: line=%d, col=%d)\n", 
			line, column, position.Line, position.Character)
		if prepareResult != nil {
			debugInfo += fmt.Sprintf("PrepareRename result: placeholder=%s, range=[%d:%d-%d:%d]\n", 
				prepareResult.Placeholder, 
				prepareResult.Range.Start.Line, prepareResult.Range.Start.Character,
				prepareResult.Range.End.Line, prepareResult.Range.End.Character)
		}
		
		workspaceEdit, err := client.Rename(ctx, uri, position, newName)
		if err != nil {
			return nil, fmt.Errorf("rename failed: %w (debug: %s)", err, debugInfo)
		}

		if workspaceEdit == nil || (len(workspaceEdit.Changes) == 0 && len(workspaceEdit.DocumentChanges) == 0) {
			return mcp.NewToolResultText(fmt.Sprintf("No changes needed for rename\n%s", debugInfo)), nil
		}

		// Apply the edits to files
		filesModified := make(map[string]bool)
		var errors []string

		// Handle both changes and documentChanges formats
		if len(workspaceEdit.DocumentChanges) > 0 {
			// Process documentChanges
			for _, docEdit := range workspaceEdit.DocumentChanges {
				filePath, err := utils.URIToPath(docEdit.TextDocument.URI)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Failed to parse URI %s: %v", docEdit.TextDocument.URI, err))
					continue
				}

				if err := applyEditsToFile(filePath, docEdit.Edits); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to apply edits to %s: %v", filePath, err))
					continue
				}
				filesModified[filePath] = true
			}
		} else {
			// Process regular changes
			for fileURI, edits := range workspaceEdit.Changes {
				filePath, err := utils.URIToPath(fileURI)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Failed to parse URI %s: %v", fileURI, err))
					continue
				}

				if err := applyEditsToFile(filePath, edits); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to apply edits to %s: %v", filePath, err))
					continue
				}
				filesModified[filePath] = true
			}
		}

		// Prepare result message
		var resultMsg string
		if len(filesModified) > 0 {
			resultMsg = fmt.Sprintf("Successfully renamed '%s' to '%s' in %d file(s):\n", prepareResult.Placeholder, newName, len(filesModified))
			for file := range filesModified {
				resultMsg += fmt.Sprintf("  - %s\n", file)
			}
		} else {
			resultMsg = "No files were modified"
		}

		if len(errors) > 0 {
			resultMsg += "\nErrors:\n"
			for _, err := range errors {
				resultMsg += fmt.Sprintf("  - %s\n", err)
			}
		}

		return mcp.NewToolResultText(resultMsg), nil
	}
}

// applyEditsToFile applies text edits to a file
func applyEditsToFile(filePath string, edits []lsp.TextEdit) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert content to lines
	lines := strings.Split(string(content), "\n")

	// Sort edits in reverse order (from end to beginning) to avoid offset issues
	sortedEdits := make([]lsp.TextEdit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		if sortedEdits[i].Range.Start.Line != sortedEdits[j].Range.Start.Line {
			return sortedEdits[i].Range.Start.Line > sortedEdits[j].Range.Start.Line
		}
		return sortedEdits[i].Range.Start.Character > sortedEdits[j].Range.Start.Character
	})

	// Apply edits
	for _, edit := range sortedEdits {
		startLine := edit.Range.Start.Line
		startChar := edit.Range.Start.Character
		endLine := edit.Range.End.Line
		endChar := edit.Range.End.Character

		// Validate line numbers
		if startLine >= len(lines) || endLine >= len(lines) {
			return fmt.Errorf("invalid line number: start=%d, end=%d, total=%d", startLine, endLine, len(lines))
		}

		// Handle single-line edit
		if startLine == endLine {
			line := lines[startLine]
			if startChar > len(line) || endChar > len(line) {
				return fmt.Errorf("invalid character position: line=%d, start=%d, end=%d, length=%d", startLine, startChar, endChar, len(line))
			}
			lines[startLine] = line[:startChar] + edit.NewText + line[endChar:]
		} else {
			// Multi-line edit
			startLineContent := lines[startLine]
			endLineContent := lines[endLine]
			
			if startChar > len(startLineContent) || endChar > len(endLineContent) {
				return fmt.Errorf("invalid character position in multi-line edit")
			}

			// Create new content
			newContent := startLineContent[:startChar] + edit.NewText + endLineContent[endChar:]
			
			// Replace the lines
			newLines := append(lines[:startLine], newContent)
			if endLine+1 < len(lines) {
				newLines = append(newLines, lines[endLine+1:]...)
			}
			lines = newLines
		}
	}

	// Write back to file
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}