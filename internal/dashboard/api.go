package dashboard

import (
	"encoding/json"
	"net/http"
	"time"
)

// WorkspaceResponse represents a workspace in API responses
type WorkspaceResponse struct {
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	Branch    string          `json:"branch"`
	MainRepo  string          `json:"main_repo,omitempty"`
	GitDirty  bool            `json:"git_dirty"`
	HasClaude bool            `json:"has_claude"`
	HasVSCode bool            `json:"has_vscode"`
	Tags      []string        `json:"tags,omitempty"`
	Server    *ServerResponse `json:"server,omitempty"`
}

// ServerResponse represents server state in API responses
type ServerResponse struct {
	Port      int       `json:"port"`
	Status    string    `json:"status"`
	URL       string    `json:"url"`
	Health    string    `json:"health,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
}

// AgentResponse represents an agent in API responses
type AgentResponse struct {
	Worktree  string    `json:"worktree"`
	Path      string    `json:"path"`
	Branch    string    `json:"branch"`
	Type      string    `json:"type"`
	PID       int       `json:"pid"`
	StartTime time.Time `json:"start_time,omitempty"`
	Duration  string    `json:"duration,omitempty"`
}

// HealthResponse represents the API health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// handleWorkspaces handles GET /api/workspaces
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaces := s.getWorkspacesData()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(workspaces); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleAgents handles GET /api/agents
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents := s.getAgentsData()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleHealth handles GET /api/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
