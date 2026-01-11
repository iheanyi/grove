package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/x/ansi"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/styles"
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

	// Get all worktrees
	worktrees := reg.ListWorktrees()

	// Use batch detection for all agents at once (much faster)
	allAgents := discovery.DetectAllAgents()

	// Match agents to worktrees, de-duplicating by PID
	// Multiple worktrees can share the same path (different branches), but we only want to show each agent once
	seenPIDs := make(map[int]bool)
	var agents []*agentView
	for _, wt := range worktrees {
		if agent, exists := allAgents[wt.Path]; exists {
			// Skip if we've already seen this PID
			if seenPIDs[agent.PID] {
				continue
			}
			seenPIDs[agent.PID] = true

			// Check for active Tasuku task
			taskID, taskDesc := discovery.GetActiveTask(wt.Path)
			if taskID != "" {
				agent.ActiveTask = taskID
				agent.TaskSummary = taskDesc
			}

			agents = append(agents, &agentView{
				Worktree: wt.Name,
				Path:     wt.Path,
				Branch:   wt.Branch,
				Agent:    agent,
			})
		}
	}

	// Sort: agents with active tasks first, then by worktree name
	sort.Slice(agents, func(i, j int) bool {
		iHasTask := agents[i].Agent.ActiveTask != ""
		jHasTask := agents[j].Agent.ActiveTask != ""
		if iHasTask != jHasTask {
			return iHasTask
		}
		return agents[i].Worktree < agents[j].Worktree
	})

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

		// Get task display (truncate if needed)
		taskDisplay := "-"
		if a.Agent.ActiveTask != "" {
			taskDisplay = ansi.Truncate(a.Agent.ActiveTask, styles.TruncateShort, styles.TruncateTail)
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
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(styles.BorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return styles.LinkHeader
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
