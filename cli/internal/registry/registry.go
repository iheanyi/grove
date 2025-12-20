package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/iheanyi/wt/internal/config"
)

// Registry manages the server registry
type Registry struct {
	path    string
	mu      sync.RWMutex
	Servers map[string]*Server `json:"servers"`
	Proxy   *ProxyInfo         `json:"proxy,omitempty"`
}

// New creates a new registry instance
func New() *Registry {
	return &Registry{
		path:    config.RegistryPath(),
		Servers: make(map[string]*Server),
		Proxy:   &ProxyInfo{},
	}
}

// Load loads the registry from disk
func Load() (*Registry, error) {
	r := New()
	return r, r.load()
}

// load reads the registry from disk
func (r *Registry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			// No registry file, start fresh
			return nil
		}
		return fmt.Errorf("failed to read registry: %w", err)
	}

	if err := json.Unmarshal(data, r); err != nil {
		return fmt.Errorf("failed to parse registry: %w", err)
	}

	return nil
}

// Save saves the registry to disk
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// Get returns a server by name
func (r *Registry) Get(name string) (*Server, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	server, ok := r.Servers[name]
	return server, ok
}

// Set adds or updates a server
func (r *Registry) Set(server *Server) error {
	r.mu.Lock()
	r.Servers[server.Name] = server
	r.mu.Unlock()

	return r.Save()
}

// Remove removes a server from the registry
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	delete(r.Servers, name)
	r.mu.Unlock()

	return r.Save()
}

// List returns all servers
func (r *Registry) List() []*Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]*Server, 0, len(r.Servers))
	for _, server := range r.Servers {
		servers = append(servers, server)
	}
	return servers
}

// ListRunning returns all running servers
func (r *Registry) ListRunning() []*Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]*Server, 0)
	for _, server := range r.Servers {
		if server.IsRunning() {
			servers = append(servers, server)
		}
	}
	return servers
}

// GetUsedPorts returns a map of ports that are in use
func (r *Registry) GetUsedPorts() map[int]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ports := make(map[int]bool)
	for _, server := range r.Servers {
		if server.IsRunning() {
			ports[server.Port] = true
		}
	}
	return ports
}

// UpdateProxy updates the proxy information
func (r *Registry) UpdateProxy(proxy *ProxyInfo) error {
	r.mu.Lock()
	r.Proxy = proxy
	r.mu.Unlock()

	return r.Save()
}

// GetProxy returns the proxy information
func (r *Registry) GetProxy() *ProxyInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.Proxy == nil {
		return &ProxyInfo{}
	}
	return r.Proxy
}

// Cleanup removes stale entries (servers with dead PIDs)
func (r *Registry) Cleanup() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := []string{}

	for name, server := range r.Servers {
		if server.PID > 0 && !isProcessRunning(server.PID) {
			server.Status = StatusStopped
			server.PID = 0
			removed = append(removed, name)
		}
	}

	if len(removed) > 0 {
		r.mu.Unlock()
		err := r.Save()
		r.mu.Lock()
		return removed, err
	}

	return removed, nil
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
