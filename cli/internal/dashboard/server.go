package dashboard

import (
	"embed"
	"fmt"
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

//go:embed web/build/*
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

	// Check if the file exists
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

	resp, err := http.Get(targetURL)
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

	// Stream the body
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

// Start starts the dashboard server
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Start background update goroutine
	go s.backgroundUpdates()

	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listeners = append(s.listeners, listener)

	log.Printf("Dashboard server starting on http://localhost:%d", s.port)

	return s.server.Serve(listener)
}

// Stop stops the dashboard server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// URL returns the dashboard URL
func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}

// backgroundUpdates periodically updates the registry and broadcasts changes
func (s *Server) backgroundUpdates() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
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
