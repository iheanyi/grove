package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "View and manage Tasuku tasks",
	Long: `View and manage Tasuku tasks for the current project.

Reads from the .tasuku directory (Tasuku V3 format).

Examples:
  grove task              # List tasks (same as grove task list)
  grove task list         # List all tasks
  grove task current      # Show current in-progress task
  grove task start <id>   # Start working on a task
  grove task done <id>    # Mark a task as complete`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTaskList(cmd, args)
	},
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  `List all tasks from the .tasuku directory.`,
	RunE:  runTaskList,
}

var taskCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current in-progress task",
	Long:  `Display the currently active (in_progress) task.`,
	RunE:  runTaskCurrent,
}

var taskStartCmd = &cobra.Command{
	Use:   "start <task-id>",
	Short: "Start working on a task",
	Long:  `Mark a task as in_progress.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskStart,
}

var taskDoneCmd = &cobra.Command{
	Use:   "done <task-id>",
	Short: "Mark a task as complete",
	Long:  `Mark a task as done.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDone,
}

func init() {
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCurrentCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskDoneCmd)

	taskListCmd.Flags().Bool("json", false, "Output in JSON format")
	taskListCmd.Flags().String("status", "", "Filter by status (open, in_progress, done)")

	rootCmd.AddCommand(taskCmd)
}

func runTaskList(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	statusFilter, _ := cmd.Flags().GetString("status")

	// Find .tasuku directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	tasukuDir := discovery.FindTasukuDir(cwd)
	if tasukuDir == "" {
		fmt.Println("No .tasuku directory found.")
		fmt.Println("\nTo initialize Tasuku, run: tk init")
		return nil
	}

	tasks, err := discovery.ListTasks(tasukuDir)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter by status if specified
	if statusFilter != "" {
		var filtered []*discovery.TasukuTask
		for _, t := range tasks {
			if statusFilter == "open" && (t.Status == "ready" || t.Status == "blocked") {
				filtered = append(filtered, t)
			} else if t.Status == statusFilter {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tasks)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	// Build table
	var rows [][]string
	for _, t := range tasks {
		statusIcon := getStatusIcon(t.Status)
		desc := t.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		rows = append(rows, []string{
			statusIcon,
			t.ID,
			desc,
		})
	}

	// Create styled table
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle()
		}).
		Headers("", "ID", "DESCRIPTION").
		Rows(rows...)

	fmt.Println(tbl)
	return nil
}

func runTaskCurrent(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	taskID, taskDesc := discovery.GetActiveTask(cwd)
	if taskID == "" {
		fmt.Println("No task currently in progress.")
		fmt.Println("\nStart a task with: grove task start <task-id>")
		return nil
	}

	fmt.Printf("Current Task: %s\n", taskID)
	fmt.Printf("Description:  %s\n", taskDesc)
	return nil
}

func runTaskStart(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	tasukuDir := discovery.FindTasukuDir(cwd)
	if tasukuDir == "" {
		return fmt.Errorf("no .tasuku directory found")
	}

	if err := discovery.UpdateTaskStatus(tasukuDir, taskID, "in_progress"); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	fmt.Printf("Started: %s\n", taskID)
	return nil
}

func runTaskDone(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	tasukuDir := discovery.FindTasukuDir(cwd)
	if tasukuDir == "" {
		return fmt.Errorf("no .tasuku directory found")
	}

	if err := discovery.UpdateTaskStatus(tasukuDir, taskID, "done"); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	fmt.Printf("Completed: %s\n", taskID)
	return nil
}

func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "done":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("✓")
	case "in_progress":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("●")
	case "blocked":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("⊘")
	default: // ready, open
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("○")
	}
}
