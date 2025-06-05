package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yantrio/mcp-gopls/internal/server"
)

func main() {
	var (
		goplsPath     string
		workspaceRoot string
		version       bool
	)

	flag.StringVar(&goplsPath, "gopls", "", "Path to gopls binary (defaults to 'gopls' in PATH)")
	flag.StringVar(&workspaceRoot, "workspace", "", "Workspace root directory (defaults to current directory)")
	flag.BoolVar(&version, "version", false, "Print version and exit")
	flag.Parse()

	if version {
		fmt.Println("mcp-gopls version 1.0.0")
		os.Exit(0)
	}

	// Use environment variables if flags not provided
	if goplsPath == "" {
		goplsPath = os.Getenv("GOPLS_PATH")
	}
	if workspaceRoot == "" {
		workspaceRoot = os.Getenv("MCP_GOPLS_WORKSPACE")
	}

	// Create and start server
	srv, err := server.New(goplsPath, workspaceRoot)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Println("Starting mcp-gopls server...")
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
