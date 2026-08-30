package dashboard

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/registry"
)

//go:generate npm --prefix web ci
//go:generate npm --prefix web run build

//go:embed all:web/build
var webFS embed.FS

// Server represents the dashboard HTTP server
type Server struct {
	port      int
	devMode   bool
	devURL    string
	mux       *http.ServeMux
	wsHub     *Hub
	registry  *registry.Registry
	mu        sync.RWMutex
	server    *http.Server
	listeners []net.Listener
	stopOnce  sync.Once
	stopCh    chan struct{}
	bgDone    chan struct{}
}

// Config holds the server configuration
type Config struct {
	Port    int
	DevMode bool
	DevURL  string
}

// NewServer creates a new dashboard server
func NewServer(cfg Config) (*Server, error) {
	reg, err := registry.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	s := &Server{
		port:     cfg.Port,
		devMode:  cfg.DevMode,
		devURL:   cfg.DevURL,
		mux:      http.NewServeMux(),
		wsHub:    NewHub(),
		registry: reg,
		stopCh:   make(chan struct{}),
		bgDone:   make(chan struct{}),
	}

	s.setupRoutes()
	return s, nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	s.mux.HandleFunc("/api/agents", s.handleAgents)
	s.mux.HandleFunc("/api/health", s.handleHealth)

	// WebSocket route
	s.mux.HandleFunc("/ws", s.wsHub.HandleWebSocket)

	// Static files (SvelteKit build)
	if s.devMode {
		// In dev mode, proxy to Vite dev server
		s.mux.HandleFunc("/", s.proxyToDev)
	} else {
		// In production, serve embedded static files
		s.mux.HandleFunc("/", s.handleStatic)
	}
}

// handleStatic serves the embedded SvelteKit build
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Get the embedded filesystem, stripping the "web/build" prefix
	staticFS, err := fs.Sub(webFS, "web/build")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create a file server
	fileServer := http.FileServer(http.FS(staticFS))

	// Try to serve the requested file
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Check if the file exists.
	if _, err := fs.Stat(staticFS, strings.TrimPrefix(path, "/")); err != nil {
		// File doesn't exist, serve index.html for SPA routing
		r.URL.Path = "/"
	}

	fileServer.ServeHTTP(w, r)
}

// proxyToDev proxies requests to the Vite dev server
func (s *Server) proxyToDev(w http.ResponseWriter, r *http.Request) {
	// Simple proxy implementation for dev mode
	targetURL := s.devURL + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil) //nolint:gosec // G107: target URL is constructed from trusted local dev server config
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create proxy request: %v", err), http.StatusBadGateway)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to proxy to dev server: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, _ = io.Copy(w, resp.Body)
}

// Start starts the dashboard server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listeners = append(s.listeners, listener)

	log.Printf("Dashboard server starting on http://localhost:%d", s.port)

	// Start goroutines only after the listener is ready.
	go s.wsHub.Run()
	go s.backgroundUpdates(2 * time.Second)

	return s.server.Serve(listener)
}

// Stop stops the dashboard server
func (s *Server) Stop() error {
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
		if s.wsHub != nil {
			s.wsHub.Stop()
		}
	})

	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// URL returns the dashboard URL
func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// backgroundUpdates periodically updates the registry and broadcasts changes.
func (s *Server) backgroundUpdates(interval time.Duration) {
	defer close(s.bgDone)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
		}

		// Reload registry
		s.mu.Lock()
		reg, err := registry.Load()
		if err == nil {
			s.registry = reg
		}
		s.mu.Unlock()

		// Broadcast update to WebSocket clients
		workspaces := s.getWorkspacesData()
		s.wsHub.Broadcast(Message{
			Type:    "workspaces_updated",
			Payload: workspaces,
		})

		agents := s.getAgentsData()
		s.wsHub.Broadcast(Message{
			Type:    "agents_updated",
			Payload: agents,
		})
	}
}

// OpenBrowser opens the dashboard in the default browser
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// getWorkspacesData fetches workspace data from the registry
func (s *Server) getWorkspacesData() []WorkspaceResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workspaces := s.registry.ListWorkspaces()
	result := make([]WorkspaceResponse, 0, len(workspaces))

	for _, ws := range workspaces {
		resp := WorkspaceResponse{
			Name:     ws.Name,
			Path:     ws.Path,
			Branch:   ws.Branch,
			MainRepo: ws.MainRepo,
			GitDirty: ws.GitDirty,
			Tags:     ws.Tags,
		}

		if ws.Server != nil {
			resp.Server = &ServerResponse{
				Port:      ws.Server.Port,
				Status:    string(ws.Server.Status),
				URL:       ws.Server.URL,
				Health:    string(ws.Server.Health),
				StartedAt: ws.Server.StartedAt,
			}
		}

		result = append(result, resp)
	}

	return result
}

// getAgentsData fetches agent data from worktrees
func (s *Server) getAgentsData() []AgentResponse {
	s.mu.RLock()
	worktrees := s.registry.ListWorktrees()
	s.mu.RUnlock()

	var agents []AgentResponse

	for _, wt := range worktrees {
		// Create a copy for detection
		wtCopy := &discovery.Worktree{
			Name:   wt.Name,
			Path:   wt.Path,
			Branch: wt.Branch,
		}

		if err := discovery.DetectActivity(wtCopy); err != nil {
			continue
		}

		if wtCopy.Agent != nil {
			agents = append(agents, AgentResponse{
				Worktree:  wt.Name,
				Path:      wt.Path,
				Branch:    wt.Branch,
				Type:      wtCopy.Agent.Type,
				PID:       wtCopy.Agent.PID,
				StartTime: wtCopy.Agent.StartTime,
				Duration:  formatDuration(time.Since(wtCopy.Agent.StartTime)),
			})
		}
	}

	return agents
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}
