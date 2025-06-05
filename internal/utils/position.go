package utils

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/yantrio/mcp-gopls/internal/lsp"
)

// ConvertPosition converts 1-indexed line/column to LSP 0-indexed position
func ConvertPosition(line, column int) lsp.Position {
	return lsp.Position{
		Line:      line - 1,
		Character: column - 1,
	}
}

// ConvertToUserPosition converts LSP 0-indexed position to 1-indexed line/column
func ConvertToUserPosition(pos lsp.Position) (line, column int) {
	return pos.Line + 1, pos.Character + 1
}

// GetLineContent reads the content of a specific line from a reader
func GetLineContent(reader io.Reader, lineNumber int) (string, error) {
	scanner := bufio.NewScanner(reader)
	currentLine := 0

	for scanner.Scan() {
		currentLine++
		if currentLine == lineNumber {
			return scanner.Text(), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return "", fmt.Errorf("line %d not found", lineNumber)
}

// GetLinePreview gets a preview of a line with surrounding context
func GetLinePreview(reader io.Reader, lineNumber int, contextLines int) (string, error) {
	scanner := bufio.NewScanner(reader)
	lines := make([]string, 0)
	currentLine := 0

	startLine := lineNumber - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := lineNumber + contextLines

	for scanner.Scan() {
		currentLine++
		if currentLine >= startLine && currentLine <= endLine {
			prefix := "  "
			if currentLine == lineNumber {
				prefix = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%d: %s", prefix, currentLine, scanner.Text()))
		}
		if currentLine > endLine {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}

// CalculateOffset calculates the byte offset for a position in a text
func CalculateOffset(text string, pos lsp.Position) (int, error) {
	lines := strings.Split(text, "\n")
	if pos.Line >= len(lines) {
		return -1, fmt.Errorf("line %d exceeds document length %d", pos.Line, len(lines))
	}

	offset := 0
	// Add lengths of all lines before the target line
	for i := 0; i < pos.Line; i++ {
		offset += len(lines[i]) + 1 // +1 for newline
	}

	// Add character offset within the line
	line := lines[pos.Line]
	if pos.Character > len(line) {
		return -1, fmt.Errorf("character %d exceeds line length %d", pos.Character, len(line))
	}

	offset += pos.Character
	return offset, nil
}

// OffsetToPosition converts a byte offset to an LSP position
func OffsetToPosition(text string, offset int) (lsp.Position, error) {
	if offset < 0 || offset > len(text) {
		return lsp.Position{}, fmt.Errorf("offset %d out of range", offset)
	}

	line := 0
	character := 0

	for i := 0; i < offset; i++ {
		if text[i] == '\n' {
			line++
			character = 0
		} else {
			character++
		}
	}

	return lsp.Position{
		Line:      line,
		Character: character,
	}, nil
}
