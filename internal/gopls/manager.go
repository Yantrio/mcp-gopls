package gopls

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/yantrio/mcp-gopls/internal/lsp"
)

type Manager struct {
	client        *lsp.Client
	goplsPath     string
	workspaceRoot string

	mu          sync.RWMutex
	initialized bool
}

func NewManager(goplsPath, workspaceRoot string) (*Manager, error) {
	if workspaceRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		workspaceRoot = cwd
	}

	absWorkspace, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &Manager{
		goplsPath:     goplsPath,
		workspaceRoot: absWorkspace,
	}, nil
}

func (m *Manager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	client, err := lsp.NewClient(m.goplsPath)
	if err != nil {
		return fmt.Errorf("failed to create LSP client: %w", err)
	}

	rootURI := pathToURI(m.workspaceRoot)
	if err := client.Initialize(ctx, rootURI); err != nil {
		_ = client.Shutdown(ctx)
		return fmt.Errorf("failed to initialize LSP client: %w", err)
	}

	m.client = client
	m.initialized = true
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized || m.client == nil {
		return nil
	}

	err := m.client.Shutdown(ctx)
	m.client = nil
	m.initialized = false
	return err
}

func (m *Manager) GetClient() (*lsp.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized || m.client == nil {
		return nil, fmt.Errorf("manager not initialized")
	}

	return m.client, nil
}

func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

func (m *Manager) WorkspaceRoot() string {
	return m.workspaceRoot
}

func pathToURI(path string) string {
	absPath, _ := filepath.Abs(path)
	return "file://" + filepath.ToSlash(absPath)
}
