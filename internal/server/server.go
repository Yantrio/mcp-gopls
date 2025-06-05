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
	// Register GoToDefinition tool
	s.mcpServer.AddTool(
		tools.NewGoToDefinitionTool(s.manager),
		tools.NewGoToDefinitionHandler(s.manager),
	)

	// Register FindReferences tool
	s.mcpServer.AddTool(
		tools.NewFindReferencesTool(s.manager),
		tools.NewFindReferencesHandler(s.manager),
	)

	// Register GetDiagnostics tool
	s.mcpServer.AddTool(
		tools.NewGetDiagnosticsTool(s.manager),
		tools.NewGetDiagnosticsHandler(s.manager),
	)

	// Register Hover tool
	s.mcpServer.AddTool(
		tools.NewHoverTool(s.manager),
		tools.NewHoverHandler(s.manager),
	)

	// Register RenameSymbol tool
	s.mcpServer.AddTool(
		tools.NewRenameSymbolTool(s.manager),
		tools.NewRenameSymbolHandler(s.manager),
	)

	// Register FindImplementers tool
	s.mcpServer.AddTool(
		tools.NewFindImplementersTool(s.manager),
		tools.NewFindImplementersHandler(s.manager),
	)

	// Register ListDocumentSymbols tool
	s.mcpServer.AddTool(
		tools.NewListDocumentSymbolsTool(s.manager),
		tools.NewListDocumentSymbolsHandler(s.manager),
	)

	// Register SearchSymbol tool
	s.mcpServer.AddTool(
		tools.NewSearchSymbolTool(s.manager),
		tools.NewSearchSymbolHandler(s.manager),
	)

	// Register FormatCode tool
	s.mcpServer.AddTool(
		tools.NewFormatCodeTool(s.manager),
		tools.NewFormatCodeHandler(s.manager),
	)

	// Register OrganizeImports tool
	s.mcpServer.AddTool(
		tools.NewOrganizeImportsTool(s.manager),
		tools.NewOrganizeImportsHandler(s.manager),
	)
}

func (s *Server) Shutdown() error {
	ctx := context.Background()
	return s.manager.Shutdown(ctx)
}
