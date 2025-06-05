package utils

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// PathToURI converts a file path to a file URI
func PathToURI(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Clean and convert to forward slashes
	absPath = filepath.ToSlash(absPath)

	// On Windows, we need to handle drive letters specially
	if runtime.GOOS == "windows" && len(absPath) > 1 && absPath[1] == ':' {
		// Convert C:/path to /C:/path for proper URI encoding
		absPath = "/" + absPath
	}

	// Encode the path
	u := &url.URL{
		Scheme: "file",
		Path:   absPath,
	}

	return u.String(), nil
}

// URIToPath converts a file URI to a file path
func URIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}

	if u.Scheme != "file" {
		return "", fmt.Errorf("expected file URI, got scheme: %s", u.Scheme)
	}

	path := u.Path

	// Decode percent-encoded characters
	path, err = url.PathUnescape(path)
	if err != nil {
		return "", fmt.Errorf("failed to unescape path: %w", err)
	}

	// On Windows, remove the leading slash before the drive letter
	if runtime.GOOS == "windows" && len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	// Convert to native path separators
	path = filepath.FromSlash(path)

	return path, nil
}

// IsFileURI checks if a string is a valid file URI
func IsFileURI(uri string) bool {
	return strings.HasPrefix(uri, "file://")
}

// NormalizeURI ensures a URI is properly formatted
func NormalizeURI(uri string) (string, error) {
	if !IsFileURI(uri) {
		// Assume it's a path and convert it
		return PathToURI(uri)
	}

	// Parse and re-encode to normalize
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}

	return u.String(), nil
}
