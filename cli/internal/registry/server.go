package registry

import (
	"time"
)

// ServerStatus represents the status of a server
type ServerStatus string

const (
	StatusRunning  ServerStatus = "running"
	StatusStopped  ServerStatus = "stopped"
	StatusStarting ServerStatus = "starting"
	StatusStopping ServerStatus = "stopping"
	StatusCrashed  ServerStatus = "crashed"
)

// HealthStatus represents the health of a server
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

// Server represents a registered server
type Server struct {
	// Name is the sanitized worktree name (used as key)
	Name string `json:"name"`

	// Port is the allocated port number
	Port int `json:"port"`

	// PID is the process ID of the running server
	PID int `json:"pid,omitempty"`

	// Command is the command used to start the server
	Command []string `json:"command"`

	// Path is the working directory
	Path string `json:"path"`

	// URL is the full URL to access the server
	URL string `json:"url"`

	// Status is the current server status
	Status ServerStatus `json:"status"`

	// Health is the current health status
	Health HealthStatus `json:"health,omitempty"`

	// StartedAt is when the server was started
	StartedAt time.Time `json:"started_at,omitempty"`

	// StoppedAt is when the server was stopped
	StoppedAt time.Time `json:"stopped_at,omitempty"`

	// LastHealthCheck is when the last health check was performed
	LastHealthCheck time.Time `json:"last_health_check,omitempty"`

	// Branch is the git branch name
	Branch string `json:"branch,omitempty"`

	// LogFile is the path to the log file
	LogFile string `json:"log_file,omitempty"`

	// Tags is a list of user-defined tags for categorization
	Tags []string `json:"tags,omitempty"`
}

// IsRunning returns true if the server is currently running
func (s *Server) IsRunning() bool {
	return s.Status == StatusRunning || s.Status == StatusStarting
}

// HasTag returns true if the server has the specified tag
func (s *Server) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the server if it doesn't already exist
func (s *Server) AddTag(tag string) bool {
	if s.HasTag(tag) {
		return false
	}
	s.Tags = append(s.Tags, tag)
	return true
}

// RemoveTag removes a tag from the server, returns true if tag was removed
func (s *Server) RemoveTag(tag string) bool {
	for i, t := range s.Tags {
		if t == tag {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			return true
		}
	}
	return false
}

// Uptime returns the duration the server has been running
func (s *Server) Uptime() time.Duration {
	if s.StartedAt.IsZero() {
		return 0
	}
	if !s.IsRunning() {
		if s.StoppedAt.IsZero() {
			return 0
		}
		return s.StoppedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
}

// UptimeString returns a human-readable uptime string
func (s *Server) UptimeString() string {
	uptime := s.Uptime()
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

func formatDuration(value int, unit string) string {
	if value == 0 {
		return ""
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10)) + unit
}

// ProxyInfo contains information about the proxy daemon
type ProxyInfo struct {
	PID       int       `json:"pid,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	HTTPPort  int       `json:"http_port"`
	HTTPSPort int       `json:"https_port"`
}

// IsRunning returns true if the proxy is running
func (p *ProxyInfo) IsRunning() bool {
	return p.PID > 0
}
