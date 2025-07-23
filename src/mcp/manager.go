package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// manager implements the Manager interface
type manager struct {
	servers  map[string]Server
	configs  map[string]ServerConfig
	mu       sync.RWMutex
	
	// Auto-initialization
	autoInit bool
	initParams *InitializeParams
}

// NewManager creates a new MCP manager
func NewManager() Manager {
	return &manager{
		servers:  make(map[string]Server),
		configs:  make(map[string]ServerConfig),
		autoInit: true,
		initParams: &InitializeParams{
			ProtocolVersion: ProtocolVersion,
			Capabilities: ClientCapability{
				// We support sampling by default
				Sampling: &SamplingCapability{},
			},
			ClientInfo: &ClientInfo{
				Name:    "gofer",
				Version: "0.1.0",
			},
		},
	}
}

// SetInitializeParams sets the initialization parameters for auto-init
func (m *manager) SetInitializeParams(params *InitializeParams) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initParams = params
}

// SetAutoInit enables/disables automatic initialization
func (m *manager) SetAutoInit(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoInit = enabled
}

// AddServer adds a server
func (m *manager) AddServer(config ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if server already exists
	if _, exists := m.servers[config.Name]; exists {
		return fmt.Errorf("server '%s' already exists", config.Name)
	}
	
	// Create the server
	server, err := NewServer(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	
	// Store server and config
	m.servers[config.Name] = server
	m.configs[config.Name] = config
	
	// Auto-initialize if enabled
	if m.autoInit {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if _, err := server.Initialize(ctx, m.initParams); err != nil {
			// Clean up on failure
			server.Close()
			delete(m.servers, config.Name)
			delete(m.configs, config.Name)
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}
	
	slog.Info("MCP server added", "name", config.Name, "transport", config.TransportType)
	return nil
}

// RemoveServer removes a server
func (m *manager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	server, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}
	
	// Close the server
	if err := server.Close(); err != nil {
		slog.Error("error closing server", "name", name, "error", err)
	}
	
	// Remove from maps
	delete(m.servers, name)
	delete(m.configs, name)
	
	slog.Info("MCP server removed", "name", name)
	return nil
}

// GetServer gets a server by name
func (m *manager) GetServer(name string) Server {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.servers[name]
}

// ListServers lists all servers
func (m *manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.servers))
	for name := range m.servers {
		names = append(names, name)
	}
	return names
}

// Close closes all servers
func (m *manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var errs []error
	
	// Close all servers
	for name, server := range m.servers {
		if err := server.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close server '%s': %w", name, err))
		}
	}
	
	// Clear maps
	m.servers = make(map[string]Server)
	m.configs = make(map[string]ServerConfig)
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing servers: %v", errs)
	}
	
	return nil
}

// Discovery provides MCP server discovery functionality
type Discovery struct {
	manager Manager
	configs []ServerConfig
	mu      sync.RWMutex
}

// NewDiscovery creates a new discovery instance
func NewDiscovery(manager Manager) *Discovery {
	return &Discovery{
		manager: manager,
		configs: make([]ServerConfig, 0),
	}
}

// LoadFromConfigs loads servers from configuration
func (d *Discovery) LoadFromConfigs(configs []ServerConfig) error {
	d.mu.Lock()
	d.configs = configs
	d.mu.Unlock()
	
	var errs []error
	
	for _, config := range configs {
		if err := d.manager.AddServer(config); err != nil {
			errs = append(errs, fmt.Errorf("failed to add server '%s': %w", config.Name, err))
			slog.Error("failed to add MCP server", "name", config.Name, "error", err)
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors loading servers: %v", errs)
	}
	
	return nil
}

// DiscoverFromEnvironment discovers MCP servers from environment
func (d *Discovery) DiscoverFromEnvironment() error {
	// This is a placeholder for environment-based discovery
	// In a real implementation, this might:
	// - Check for MCP_SERVERS environment variable
	// - Look for configuration files in standard locations
	// - Use service discovery mechanisms
	
	slog.Info("MCP server discovery from environment not yet implemented")
	return nil
}

// GetConfigs returns the loaded configurations
func (d *Discovery) GetConfigs() []ServerConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	result := make([]ServerConfig, len(d.configs))
	copy(result, d.configs)
	return result
}