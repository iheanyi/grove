package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List active AI agent sessions",
	Long: `List active AI agent sessions (Claude Code, etc.) across all worktrees.

Shows agent type, working directory, duration, and process info.

Examples:
  grove agents              # List all active agents
  grove agents --json       # Output as JSON
  grove agents --watch      # Continuously update (every 2s)`,
	RunE: runAgents,
}

func init() {
	agentsCmd.Flags().Bool("json", false, "Output in JSON format")
	agentsCmd.Flags().Bool("watch", false, "Continuously update the list")
	rootCmd.AddCommand(agentsCmd)
}

func runAgents(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	watchMode, _ := cmd.Flags().GetBool("watch")

	if watchMode {
		return runAgentsWatch(jsonOutput)
	}

	return runAgentsOnce(jsonOutput)
}

func runAgentsOnce(jsonOutput bool) error {
	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get all worktrees and detect activity
	worktrees := reg.ListWorktrees()

	// Detect activity for each worktree in parallel
	var agents []*agentView
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
			agents = append(agents, &agentView{
				Worktree: wt.Name,
				Path:     wt.Path,
				Branch:   wt.Branch,
				Agent:    wtCopy.Agent,
			})
		}
	}

	if jsonOutput {
		return outputAgentsJSON(agents)
	}
	return outputAgentsTable(agents)
}

func runAgentsWatch(jsonOutput bool) error {
	// Clear screen and move cursor to top
	fmt.Print("\033[2J\033[H")

	for {
		// Move cursor to top
		fmt.Print("\033[H")

		if err := runAgentsOnce(jsonOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		fmt.Printf("\nLast updated: %s (press Ctrl+C to exit)\n", time.Now().Format("15:04:05"))
		time.Sleep(2 * time.Second)
	}
}

type agentView struct {
	Worktree string
	Path     string
	Branch   string
	Agent    *discovery.AgentInfo
}

func outputAgentsJSON(agents []*agentView) error {
	type jsonAgent struct {
		Worktree    string `json:"worktree"`
		Path        string `json:"path"`
		Branch      string `json:"branch"`
		Type        string `json:"type"`
		PID         int    `json:"pid"`
		StartTime   string `json:"start_time,omitempty"`
		Duration    string `json:"duration,omitempty"`
		ActiveTask  string `json:"active_task,omitempty"`
		TaskSummary string `json:"task_summary,omitempty"`
	}

	var out []jsonAgent
	for _, a := range agents {
		ja := jsonAgent{
			Worktree:    a.Worktree,
			Path:        a.Path,
			Branch:      a.Branch,
			Type:        a.Agent.Type,
			PID:         a.Agent.PID,
			ActiveTask:  a.Agent.ActiveTask,
			TaskSummary: a.Agent.TaskSummary,
		}
		if !a.Agent.StartTime.IsZero() {
			ja.StartTime = a.Agent.StartTime.Format(time.RFC3339)
			ja.Duration = formatDuration(time.Since(a.Agent.StartTime))
		}
		out = append(out, ja)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputAgentsTable(agents []*agentView) error {
	if len(agents) == 0 {
		fmt.Println("No active agents found.")
		fmt.Println("\nAgents are detected by finding running Claude Code processes.")
		return nil
	}

	fmt.Printf("Active Agents (%d):\n\n", len(agents))

	// Build table
	var rows [][]string
	for _, a := range agents {
		duration := "-"
		if !a.Agent.StartTime.IsZero() {
			duration = formatDuration(time.Since(a.Agent.StartTime))
		}

		// Shorten path
		displayPath := a.Path
		if home, err := os.UserHomeDir(); err == nil {
			if strings.HasPrefix(displayPath, home) {
				displayPath = "~" + strings.TrimPrefix(displayPath, home)
			}
		}

		// Get task display (truncate if needed)
		taskDisplay := "-"
		if a.Agent.ActiveTask != "" {
			taskDisplay = a.Agent.ActiveTask
			if len(taskDisplay) > 25 {
				taskDisplay = taskDisplay[:22] + "..."
			}
		}

		rows = append(rows, []string{
			a.Agent.Type,
			a.Worktree,
			taskDisplay,
			duration,
			fmt.Sprintf("%d", a.Agent.PID),
		})
	}

	// Create styled table
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle()
		}).
		Headers("TYPE", "WORKTREE", "TASK", "DURATION", "PID").
		Rows(rows...)

	fmt.Println(t)
	return nil
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
