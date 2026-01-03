package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/port"
)

// Registry manages the server registry
type Registry struct {
	path      string
	mu        sync.RWMutex
	Servers   map[string]*Server             `json:"servers"`
	Worktrees map[string]*discovery.Worktree `json:"worktrees,omitempty"`
	Proxy     *ProxyInfo                     `json:"proxy,omitempty"`
}

// New creates a new registry instance
func New() *Registry {
	return &Registry{
		path:      config.RegistryPath(),
		Servers:   make(map[string]*Server),
		Worktrees: make(map[string]*discovery.Worktree),
		Proxy:     &ProxyInfo{},
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

// CleanupResult holds the results of a cleanup operation
type CleanupResult struct {
	Stopped          []string // Servers whose PIDs are no longer running
	RemovedServers   []string // Servers whose paths no longer exist
	RemovedWorktrees []string // Worktrees whose paths no longer exist
}

// Cleanup removes stale entries (servers/worktrees with missing paths, dead PIDs)
func (r *Registry) Cleanup() (*CleanupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := &CleanupResult{
		Stopped:          []string{},
		RemovedServers:   []string{},
		RemovedWorktrees: []string{},
	}

	serversToDelete := []string{}
	worktreesToDelete := []string{}

	// Check servers
	for name, server := range r.Servers {
		// Check if the path still exists
		if server.Path != "" {
			if _, err := os.Stat(server.Path); os.IsNotExist(err) {
				serversToDelete = append(serversToDelete, name)
				result.RemovedServers = append(result.RemovedServers, name)
				continue
			}
		}

		// Check if PID is still running
		if server.PID > 0 && !isProcessRunning(server.PID) {
			server.Status = StatusStopped
			server.PID = 0
			result.Stopped = append(result.Stopped, name)
			continue
		}

		// For "running" servers with no PID, check if port is actually in use
		if server.Status == StatusRunning && server.PID == 0 && server.Port > 0 {
			if !port.IsListening(server.Port) {
				server.Status = StatusStopped
				result.Stopped = append(result.Stopped, name)
			}
		}
	}

	// Check worktrees
	for name, wt := range r.Worktrees {
		if wt.Path != "" {
			if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
				worktreesToDelete = append(worktreesToDelete, name)
				result.RemovedWorktrees = append(result.RemovedWorktrees, name)
			}
		}
	}

	// Remove entries with missing paths
	for _, name := range serversToDelete {
		delete(r.Servers, name)
	}
	for _, name := range worktreesToDelete {
		delete(r.Worktrees, name)
	}

	if len(result.Stopped) > 0 || len(result.RemovedServers) > 0 || len(result.RemovedWorktrees) > 0 {
		r.mu.Unlock()
		err := r.Save()
		r.mu.Lock()
		return result, err
	}

	return result, nil
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

// GetWorktree returns a worktree by name
func (r *Registry) GetWorktree(name string) (*discovery.Worktree, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wt, ok := r.Worktrees[name]
	return wt, ok
}

// SetWorktree adds or updates a worktree
func (r *Registry) SetWorktree(wt *discovery.Worktree) error {
	r.mu.Lock()
	r.Worktrees[wt.Name] = wt
	r.mu.Unlock()

	return r.Save()
}

// RemoveWorktree removes a worktree from the registry
func (r *Registry) RemoveWorktree(name string) error {
	r.mu.Lock()
	delete(r.Worktrees, name)
	r.mu.Unlock()

	return r.Save()
}

// ListWorktrees returns all worktrees
func (r *Registry) ListWorktrees() []*discovery.Worktree {
	r.mu.RLock()
	defer r.mu.RUnlock()

	worktrees := make([]*discovery.Worktree, 0, len(r.Worktrees))
	for _, wt := range r.Worktrees {
		worktrees = append(worktrees, wt)
	}
	return worktrees
}

// UpdateWorktreeActivities updates all worktrees with their current activity status
func (r *Registry) UpdateWorktreeActivities() error {
	r.mu.Lock()
	for _, wt := range r.Worktrees {
		if err := discovery.DetectActivity(wt); err != nil {
			// Continue on error, just log it
			continue
		}
	}
	r.mu.Unlock()

	return r.Save()
}
