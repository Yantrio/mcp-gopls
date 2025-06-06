package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/yantrio/mcp-gopls/internal/gopls"
	"github.com/yantrio/mcp-gopls/internal/tools"
)

type Server struct {
	mcpServer *server.MCPServer
	manager   *gopls.Manager
}

func New(goplsPath, workspaceRoot string) (*Server, error) {
	manager, err := gopls.NewManager(goplsPath, workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create gopls manager: %w", err)
	}

	mcpServer := server.NewMCPServer(
		"mcp-gopls",
		"1.0.0",
		server.WithInstructions(
			"Go language server integration via gopls. "+
				"Use these tools to interact with Go code for accurate, context-aware analysis and refactoring. "+
				"\n\n"+
				"gopls is the official Go language server that understands your entire codebase, making it far more reliable than grep/search for:\n"+
				"• Finding references - gopls understands Go semantics, not just text matching\n"+
				"• Renaming symbols - safely renames across packages with type awareness\n"+
				"• Navigation - jumps to actual definitions, not just similar names\n"+
				"• Code analysis - provides real compiler errors and type information\n"+
				"\n"+
				"For Go code tasks, always prefer these tools over generic file search/edit operations.",
		),
	)

	s := &Server{
		mcpServer: mcpServer,
		manager:   manager,
	}

	// Register all tools
	s.registerTools()

	return s, nil
}

func (s *Server) Start() error {
	// Initialize gopls when server starts
	ctx := context.Background()
	if err := s.manager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize gopls: %w", err)
	}

	// Start the MCP server
	return server.ServeStdio(s.mcpServer)
}

func (s *Server) registerTools() {
	// Get all tools and handlers
	toolList := tools.GetTools(s.manager)
	handlers := tools.GetToolHandlers(s.manager)

	// Register each tool with its handler
	for _, tool := range toolList {
		if handler, ok := handlers[tool.Name]; ok {
			s.mcpServer.AddTool(tool, handler)
		}
	}
}

func (s *Server) Shutdown() error {
	ctx := context.Background()
	return s.manager.Shutdown(ctx)
}
