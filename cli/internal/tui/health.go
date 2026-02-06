package tui

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/iheanyi/grove/internal/registry"
)

// healthClient is a shared http.Client with connection pooling for health checks.
var healthClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 2,
		DialContext: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).DialContext,
	},
}

// HealthCheckMsg is sent when a health check completes
type HealthCheckMsg struct {
	ServerName string
	Health     registry.HealthStatus
	CheckTime  time.Time
}

// StartHealthChecks starts periodic health checks for all servers
func StartHealthChecks(reg *registry.Registry) tea.Cmd {
	return func() tea.Msg {
		servers := reg.ListRunning()
		for _, server := range servers {
			go checkServerHealth(server)
		}
		return nil
	}
}

// checkServerHealth performs a health check on a server
func checkServerHealth(server *registry.Server) tea.Msg {
	health := performHealthCheck(server.URL)
	return HealthCheckMsg{
		ServerName: server.Name,
		Health:     health,
		CheckTime:  time.Now(),
	}
}

// performHealthCheck performs an HTTP health check
func performHealthCheck(url string) registry.HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return registry.HealthUnknown
	}

	resp, err := healthClient.Do(req)
	if err != nil {
		return registry.HealthUnhealthy
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		return registry.HealthHealthy
	}

	return registry.HealthUnhealthy
}

// HealthCheckCmd creates a command to check health for a specific server
func HealthCheckCmd(server *registry.Server) tea.Cmd {
	return func() tea.Msg {
		return checkServerHealth(server)
	}
}

// HealthCheckTicker returns a command that periodically triggers health checks
func HealthCheckTicker(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return healthCheckTickMsg(t)
	})
}

// healthCheckTickMsg is sent periodically to trigger health checks
type healthCheckTickMsg time.Time

// FormatHealthStatus formats a health status with color
func FormatHealthStatus(health registry.HealthStatus) string {
	switch health {
	case registry.HealthHealthy:
		return healthyStyle.Render("✓ healthy")
	case registry.HealthUnhealthy:
		return unhealthyStyle.Render("✗ unhealthy")
	default:
		return unknownStyle.Render("? unknown")
	}
}

// FormatLastHealthCheck formats the last health check time
func FormatLastHealthCheck(lastCheck time.Time) string {
	if lastCheck.IsZero() {
		return "never"
	}

	duration := time.Since(lastCheck)
	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(duration.Hours()))
}
