package organize_imports

import (
	"context"
	"encoding/json"
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
		Name:        "OrganizeImports",
		Description: "Organize import statements",
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
		// Parse arguments
		args, err := json.Marshal(arguments)
		if err != nil {
			return nil, err
		}

		var input struct {
			File string `json:"file"`
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

		// Count lines in the file to get proper range
		lines := strings.Count(string(content), "\n")
		
		// Request code actions for organizing imports
		codeActions, err := client.CodeActionForRange(ctx, uri, lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: lines, Character: 0},
		})
		if err != nil {
			return nil, fmt.Errorf("code action request failed: %w", err)
		}

		// Find the organize imports action
		var organizeImportsAction *lsp.CodeAction
		for _, action := range codeActions {
			if action.Kind == lsp.CodeActionKindSourceOrganizeImports {
				organizeImportsAction = &action
				break
			}
		}

		if organizeImportsAction == nil {
			return mcp.NewToolResultText(fmt.Sprintf("No import organization needed for %s", input.File)), nil
		}

		// Apply the workspace edit if available
		if organizeImportsAction.Edit != nil {
			if err := applyWorkspaceEdit(input.File, organizeImportsAction.Edit); err != nil {
				return nil, fmt.Errorf("failed to apply import organization: %w", err)
			}
			return mcp.NewToolResultText(fmt.Sprintf("Successfully organized imports in %s", input.File)), nil
		}

		// If there's no edit but a command, we can't execute it directly
		if organizeImportsAction.Command != nil {
			return mcp.NewToolResultText("Import organization requires command execution, which is not supported"), nil
		}

		return mcp.NewToolResultText("No changes needed for import organization"), nil
	}
}

// applyWorkspaceEdit applies a workspace edit to files
func applyWorkspaceEdit(targetFile string, edit *lsp.WorkspaceEdit) error {
	// Handle document changes format
	if len(edit.DocumentChanges) > 0 {
		for _, docEdit := range edit.DocumentChanges {
			filePath, err := utils.URIToPath(docEdit.TextDocument.URI)
			if err != nil {
				return fmt.Errorf("failed to parse URI: %w", err)
			}
			
			if filePath == targetFile {
				if err := applyTextEdits(filePath, docEdit.Edits); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Handle changes format
	for fileURI, edits := range edit.Changes {
		filePath, err := utils.URIToPath(fileURI)
		if err != nil {
			return fmt.Errorf("failed to parse URI: %w", err)
		}
		
		if filePath == targetFile {
			if err := applyTextEdits(filePath, edits); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyTextEdits applies LSP text edits to a file
func applyTextEdits(filePath string, edits []lsp.TextEdit) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to lines for easier manipulation
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