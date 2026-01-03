package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/port"
)

// Workspace represents a unified view of a git worktree with optional server state.
// This is the primary data structure for tracking development environments.
type Workspace struct {
	// Identity
	Name string `json:"name"`
	Path string `json:"path"`

	// Git state
	Branch   string `json:"branch"`
	MainRepo string `json:"main_repo,omitempty"`
	GitDirty bool   `json:"git_dirty,omitempty"`

	// Activity detection
	HasClaude    bool      `json:"has_claude,omitempty"`
	HasVSCode    bool      `json:"has_vscode,omitempty"`
	LastActivity time.Time `json:"last_activity,omitempty"`

	// Server (optional - nil means no server configured)
	Server *ServerState `json:"server,omitempty"`

	// Metadata
	Tags         []string  `json:"tags,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
}

// ServerState represents the state of a dev server within a workspace.
type ServerState struct {
	Port            int          `json:"port"`
	PID             int          `json:"pid,omitempty"`
	Status          ServerStatus `json:"status"`
	URL             string       `json:"url"`
	Command         []string     `json:"command,omitempty"`
	LogFile         string       `json:"log_file,omitempty"`
	StartedAt       time.Time    `json:"started_at,omitempty"`
	StoppedAt       time.Time    `json:"stopped_at,omitempty"`
	Health          HealthStatus `json:"health,omitempty"`
	LastHealthCheck time.Time    `json:"last_health_check,omitempty"`
}

// IsRunning returns true if the workspace has a running server
func (w *Workspace) IsRunning() bool {
	if w.Server == nil {
		return false
	}
	return w.Server.Status == StatusRunning || w.Server.Status == StatusStarting
}

// HasServerState returns true if the workspace has server configuration
func (w *Workspace) HasServerState() bool {
	return w.Server != nil
}

// GetPort returns the server port, or 0 if no server
func (w *Workspace) GetPort() int {
	if w.Server == nil {
		return 0
	}
	return w.Server.Port
}

// GetURL returns the server URL, or empty string if no server
func (w *Workspace) GetURL() string {
	if w.Server == nil {
		return ""
	}
	return w.Server.URL
}

// Uptime returns the duration the server has been running
func (w *Workspace) Uptime() time.Duration {
	if w.Server == nil || w.Server.StartedAt.IsZero() {
		return 0
	}
	if !w.IsRunning() {
		if w.Server.StoppedAt.IsZero() {
			return 0
		}
		return w.Server.StoppedAt.Sub(w.Server.StartedAt)
	}
	return time.Since(w.Server.StartedAt)
}

// UptimeString returns a human-readable uptime string
func (w *Workspace) UptimeString() string {
	uptime := w.Uptime()
	if uptime == 0 {
		return "-"
	}

	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60

	if hours > 0 {
		return formatDuration(hours, "h") + " " + formatDuration(minutes, "m")
	}
	return formatDuration(minutes, "m")
}

// HasTag returns true if the workspace has the specified tag
func (w *Workspace) HasTag(tag string) bool {
	for _, t := range w.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the workspace if it doesn't already exist
func (w *Workspace) AddTag(tag string) bool {
	if w.HasTag(tag) {
		return false
	}
	w.Tags = append(w.Tags, tag)
	return true
}

// RemoveTag removes a tag from the workspace, returns true if tag was removed
func (w *Workspace) RemoveTag(tag string) bool {
	for i, t := range w.Tags {
		if t == tag {
			w.Tags = append(w.Tags[:i], w.Tags[i+1:]...)
			return true
		}
	}
	return false
}

// ToServer converts a Workspace to a Server for backward compatibility
func (w *Workspace) ToServer() *Server {
	if w == nil {
		return nil
	}

	server := &Server{
		Name:   w.Name,
		Path:   w.Path,
		Branch: w.Branch,
		Tags:   w.Tags,
	}

	if w.Server != nil {
		server.Port = w.Server.Port
		server.PID = w.Server.PID
		server.Status = w.Server.Status
		server.URL = w.Server.URL
		server.Command = w.Server.Command
		server.LogFile = w.Server.LogFile
		server.StartedAt = w.Server.StartedAt
		server.StoppedAt = w.Server.StoppedAt
		server.Health = w.Server.Health
		server.LastHealthCheck = w.Server.LastHealthCheck
	} else {
		server.Status = StatusStopped
	}

	return server
}

// WorkspaceFromServer creates a Workspace from an existing Server
func WorkspaceFromServer(s *Server) *Workspace {
	if s == nil {
		return nil
	}

	ws := &Workspace{
		Name:      s.Name,
		Path:      s.Path,
		Branch:    s.Branch,
		Tags:      s.Tags,
		CreatedAt: s.StartedAt,
	}

	// Only create ServerState if the server has meaningful data
	if s.Port > 0 || s.PID > 0 || s.Status != "" {
		ws.Server = &ServerState{
			Port:            s.Port,
			PID:             s.PID,
			Status:          s.Status,
			URL:             s.URL,
			Command:         s.Command,
			LogFile:         s.LogFile,
			StartedAt:       s.StartedAt,
			StoppedAt:       s.StoppedAt,
			Health:          s.Health,
			LastHealthCheck: s.LastHealthCheck,
		}
	}

	return ws
}

// WorkspaceFromWorktree creates a Workspace from a discovery.Worktree
func WorkspaceFromWorktree(wt *discovery.Worktree) *Workspace {
	if wt == nil {
		return nil
	}

	return &Workspace{
		Name:         wt.Name,
		Path:         wt.Path,
		Branch:       wt.Branch,
		MainRepo:     wt.MainRepo,
		GitDirty:     wt.GitDirty,
		HasClaude:    wt.HasClaude,
		HasVSCode:    wt.HasVSCode,
		LastActivity: wt.LastActivity,
		DiscoveredAt: wt.DiscoveredAt,
	}
}

// Registry manages the server registry
type Registry struct {
	path string
	mu   sync.RWMutex

	// New unified model
	Workspaces map[string]*Workspace `json:"workspaces,omitempty"`

	// Legacy fields (kept for backward compatibility during migration)
	Servers   map[string]*Server             `json:"servers,omitempty"`
	Worktrees map[string]*discovery.Worktree `json:"worktrees,omitempty"`

	Proxy *ProxyInfo `json:"proxy,omitempty"`

	// Internal flag to track if we migrated
	migrated bool
}

// New creates a new registry instance
func New() *Registry {
	return &Registry{
		path:       config.RegistryPath(),
		Workspaces: make(map[string]*Workspace),
		Servers:    make(map[string]*Server),
		Worktrees:  make(map[string]*discovery.Worktree),
		Proxy:      &ProxyInfo{},
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

	// Ensure maps are initialized after unmarshal
	if r.Workspaces == nil {
		r.Workspaces = make(map[string]*Workspace)
	}
	if r.Servers == nil {
		r.Servers = make(map[string]*Server)
	}
	if r.Worktrees == nil {
		r.Worktrees = make(map[string]*discovery.Worktree)
	}

	// Migrate old format to new if needed
	if len(r.Workspaces) == 0 && (len(r.Servers) > 0 || len(r.Worktrees) > 0) {
		r.migrateToWorkspaces()
	}

	return nil
}

// migrateToWorkspaces converts old Servers and Worktrees to unified Workspaces
func (r *Registry) migrateToWorkspaces() {
	// First, create workspaces from servers
	for name, server := range r.Servers {
		ws := WorkspaceFromServer(server)
		r.Workspaces[name] = ws
	}

	// Then merge in worktree data (may add new workspaces or enrich existing)
	for name, wt := range r.Worktrees {
		if existing, ok := r.Workspaces[name]; ok {
			// Merge worktree data into existing workspace
			existing.MainRepo = wt.MainRepo
			existing.GitDirty = wt.GitDirty
			existing.HasClaude = wt.HasClaude
			existing.HasVSCode = wt.HasVSCode
			existing.LastActivity = wt.LastActivity
			existing.DiscoveredAt = wt.DiscoveredAt
			if existing.Branch == "" {
				existing.Branch = wt.Branch
			}
			if existing.Path == "" {
				existing.Path = wt.Path
			}
		} else {
			// Create new workspace from worktree
			r.Workspaces[name] = WorkspaceFromWorktree(wt)
		}
	}

	r.migrated = true
}

// Save saves the registry to disk
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	// Sync workspaces back to legacy maps for backward compatibility
	r.syncToLegacy()

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// syncToLegacy updates the legacy Servers and Worktrees maps from Workspaces
// This ensures backward compatibility with older code/tools that read the registry
func (r *Registry) syncToLegacy() {
	// Clear and rebuild legacy maps
	r.Servers = make(map[string]*Server)
	r.Worktrees = make(map[string]*discovery.Worktree)

	for name, ws := range r.Workspaces {
		// Always create a Server entry
		r.Servers[name] = ws.ToServer()

		// Create Worktree entry
		r.Worktrees[name] = &discovery.Worktree{
			Name:         ws.Name,
			Path:         ws.Path,
			Branch:       ws.Branch,
			MainRepo:     ws.MainRepo,
			GitDirty:     ws.GitDirty,
			HasClaude:    ws.HasClaude,
			HasVSCode:    ws.HasVSCode,
			LastActivity: ws.LastActivity,
			DiscoveredAt: ws.DiscoveredAt,
			HasServer:    ws.HasServerState(),
		}
	}
}

// GetWorkspace returns a workspace by name
func (r *Registry) GetWorkspace(name string) (*Workspace, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ws, ok := r.Workspaces[name]
	return ws, ok
}

// SetWorkspace adds or updates a workspace
func (r *Registry) SetWorkspace(ws *Workspace) error {
	r.mu.Lock()
	r.Workspaces[ws.Name] = ws
	r.mu.Unlock()

	return r.Save()
}

// SetWorkspaceWithoutSave adds or updates a workspace without saving (for batch operations)
func (r *Registry) SetWorkspaceWithoutSave(ws *Workspace) {
	r.mu.Lock()
	r.Workspaces[ws.Name] = ws
	r.mu.Unlock()
}

// RemoveWorkspace removes a workspace from the registry
func (r *Registry) RemoveWorkspace(name string) error {
	r.mu.Lock()
	delete(r.Workspaces, name)
	r.mu.Unlock()

	return r.Save()
}

// RemoveWorkspaceWithoutSave removes a workspace without saving (for batch operations)
func (r *Registry) RemoveWorkspaceWithoutSave(name string) {
	r.mu.Lock()
	delete(r.Workspaces, name)
	r.mu.Unlock()
}

// ListWorkspaces returns all workspaces
func (r *Registry) ListWorkspaces() []*Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workspaces := make([]*Workspace, 0, len(r.Workspaces))
	for _, ws := range r.Workspaces {
		workspaces = append(workspaces, ws)
	}
	return workspaces
}

// ListRunningWorkspaces returns all workspaces with running servers
func (r *Registry) ListRunningWorkspaces() []*Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workspaces := make([]*Workspace, 0)
	for _, ws := range r.Workspaces {
		if ws.IsRunning() {
			workspaces = append(workspaces, ws)
		}
	}
	return workspaces
}

// =============================================================================
// Backward-compatible Server methods (delegate to Workspace operations)
// =============================================================================

// Get returns a server by name (backward compatible wrapper)
func (r *Registry) Get(name string) (*Server, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if ws, ok := r.Workspaces[name]; ok {
		return ws.ToServer(), true
	}
	return nil, false
}

// Set adds or updates a server (backward compatible wrapper)
func (r *Registry) Set(server *Server) error {
	r.mu.Lock()

	// Check if workspace exists
	if ws, ok := r.Workspaces[server.Name]; ok {
		// Update existing workspace's server state
		ws.Path = server.Path
		ws.Branch = server.Branch
		ws.Tags = server.Tags
		ws.Server = &ServerState{
			Port:            server.Port,
			PID:             server.PID,
			Status:          server.Status,
			URL:             server.URL,
			Command:         server.Command,
			LogFile:         server.LogFile,
			StartedAt:       server.StartedAt,
			StoppedAt:       server.StoppedAt,
			Health:          server.Health,
			LastHealthCheck: server.LastHealthCheck,
		}
	} else {
		// Create new workspace from server
		r.Workspaces[server.Name] = WorkspaceFromServer(server)
	}

	r.mu.Unlock()
	return r.Save()
}

// Remove removes a server from the registry (backward compatible wrapper)
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	delete(r.Workspaces, name)
	r.mu.Unlock()

	return r.Save()
}

// RemoveWithoutSave removes a server without saving (for batch operations)
func (r *Registry) RemoveWithoutSave(name string) {
	r.mu.Lock()
	delete(r.Workspaces, name)
	r.mu.Unlock()
}

// List returns all servers (backward compatible wrapper)
func (r *Registry) List() []*Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]*Server, 0, len(r.Workspaces))
	for _, ws := range r.Workspaces {
		servers = append(servers, ws.ToServer())
	}
	return servers
}

// ListRunning returns all running servers (backward compatible wrapper)
func (r *Registry) ListRunning() []*Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]*Server, 0)
	for _, ws := range r.Workspaces {
		if ws.IsRunning() {
			servers = append(servers, ws.ToServer())
		}
	}
	return servers
}

// GetUsedPorts returns a map of ports that are in use
func (r *Registry) GetUsedPorts() map[int]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ports := make(map[int]bool)
	for _, ws := range r.Workspaces {
		if ws.IsRunning() && ws.Server != nil {
			ports[ws.Server.Port] = true
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

// Cleanup removes stale entries (workspaces with missing paths, dead PIDs)
func (r *Registry) Cleanup() (*CleanupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := &CleanupResult{
		Stopped:          []string{},
		RemovedServers:   []string{},
		RemovedWorktrees: []string{},
	}

	workspacesToDelete := []string{}

	// Check workspaces
	for name, ws := range r.Workspaces {
		// Check if the path still exists
		if ws.Path != "" {
			if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
				workspacesToDelete = append(workspacesToDelete, name)
				result.RemovedServers = append(result.RemovedServers, name)
				result.RemovedWorktrees = append(result.RemovedWorktrees, name)
				continue
			}
		}

		// Check server state if present
		if ws.Server != nil {
			// Check if PID is still running
			if ws.Server.PID > 0 && !isProcessRunning(ws.Server.PID) {
				ws.Server.Status = StatusStopped
				ws.Server.PID = 0
				result.Stopped = append(result.Stopped, name)
				continue
			}

			// For "running" servers with no PID, check if port is actually in use
			if ws.Server.Status == StatusRunning && ws.Server.PID == 0 && ws.Server.Port > 0 {
				if !port.IsListening(ws.Server.Port) {
					ws.Server.Status = StatusStopped
					result.Stopped = append(result.Stopped, name)
				}
			}
		}
	}

	// Remove entries with missing paths
	for _, name := range workspacesToDelete {
		delete(r.Workspaces, name)
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

// =============================================================================
// Backward-compatible Worktree methods (delegate to Workspace operations)
// =============================================================================

// GetWorktree returns a worktree by name (backward compatible wrapper)
func (r *Registry) GetWorktree(name string) (*discovery.Worktree, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if ws, ok := r.Workspaces[name]; ok {
		return &discovery.Worktree{
			Name:         ws.Name,
			Path:         ws.Path,
			Branch:       ws.Branch,
			MainRepo:     ws.MainRepo,
			GitDirty:     ws.GitDirty,
			HasClaude:    ws.HasClaude,
			HasVSCode:    ws.HasVSCode,
			LastActivity: ws.LastActivity,
			DiscoveredAt: ws.DiscoveredAt,
			HasServer:    ws.HasServerState(),
		}, true
	}
	return nil, false
}

// SetWorktree adds or updates a worktree (backward compatible wrapper)
func (r *Registry) SetWorktree(wt *discovery.Worktree) error {
	r.mu.Lock()

	// Check if workspace exists
	if ws, ok := r.Workspaces[wt.Name]; ok {
		// Update existing workspace's worktree data
		ws.Path = wt.Path
		ws.Branch = wt.Branch
		ws.MainRepo = wt.MainRepo
		ws.GitDirty = wt.GitDirty
		ws.HasClaude = wt.HasClaude
		ws.HasVSCode = wt.HasVSCode
		ws.LastActivity = wt.LastActivity
		ws.DiscoveredAt = wt.DiscoveredAt
	} else {
		// Create new workspace from worktree
		r.Workspaces[wt.Name] = WorkspaceFromWorktree(wt)
	}

	r.mu.Unlock()
	return r.Save()
}

// RemoveWorktree removes a worktree from the registry (backward compatible wrapper)
func (r *Registry) RemoveWorktree(name string) error {
	r.mu.Lock()
	delete(r.Workspaces, name)
	r.mu.Unlock()

	return r.Save()
}

// RemoveWorktreeWithoutSave removes a worktree without saving (for batch operations)
func (r *Registry) RemoveWorktreeWithoutSave(name string) {
	r.mu.Lock()
	delete(r.Workspaces, name)
	r.mu.Unlock()
}

// ListWorktrees returns all worktrees (backward compatible wrapper)
func (r *Registry) ListWorktrees() []*discovery.Worktree {
	r.mu.RLock()
	defer r.mu.RUnlock()

	worktrees := make([]*discovery.Worktree, 0, len(r.Workspaces))
	for _, ws := range r.Workspaces {
		worktrees = append(worktrees, &discovery.Worktree{
			Name:         ws.Name,
			Path:         ws.Path,
			Branch:       ws.Branch,
			MainRepo:     ws.MainRepo,
			GitDirty:     ws.GitDirty,
			HasClaude:    ws.HasClaude,
			HasVSCode:    ws.HasVSCode,
			LastActivity: ws.LastActivity,
			DiscoveredAt: ws.DiscoveredAt,
			HasServer:    ws.HasServerState(),
		})
	}
	return worktrees
}

// UpdateWorktreeActivities updates all workspaces with their current activity status.
// Activity detection is parallelized across all workspaces for performance.
func (r *Registry) UpdateWorktreeActivities() error {
	r.mu.RLock()
	workspaces := make([]*Workspace, 0, len(r.Workspaces))
	for _, ws := range r.Workspaces {
		workspaces = append(workspaces, ws)
	}
	r.mu.RUnlock()

	if len(workspaces) == 0 {
		return nil
	}

	// Create a channel to receive results
	type activityResult struct {
		ws       *Workspace
		dirty    bool
		claude   bool
		vscode   bool
		activity time.Time
	}

	results := make(chan activityResult, len(workspaces))
	var wg sync.WaitGroup

	// Launch goroutines for each workspace (parallelized for speed)
	for _, ws := range workspaces {
		wg.Add(1)
		go func(ws *Workspace) {
			defer wg.Done()

			// Create a temporary worktree to use with DetectActivity
			wt := &discovery.Worktree{
				Name:     ws.Name,
				Path:     ws.Path,
				Branch:   ws.Branch,
				MainRepo: ws.MainRepo,
			}
			if err := discovery.DetectActivity(wt); err != nil {
				return
			}

			results <- activityResult{
				ws:       ws,
				dirty:    wt.GitDirty,
				claude:   wt.HasClaude,
				vscode:   wt.HasVSCode,
				activity: wt.LastActivity,
			}
		}(ws)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and update workspaces
	r.mu.Lock()
	for result := range results {
		result.ws.GitDirty = result.dirty
		result.ws.HasClaude = result.claude
		result.ws.HasVSCode = result.vscode
		result.ws.LastActivity = result.activity
	}
	r.mu.Unlock()

	return r.Save()
}
