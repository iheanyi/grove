package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TasukuTask represents a task from the .tasuku/tasks/ directory
type TasukuTask struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	ParentID    string   `json:"parent_id,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// FindTasukuDir locates the .tasuku directory for a given path.
// It searches up the directory tree from the given path.
func FindTasukuDir(startPath string) string {
	path := startPath
	for {
		tasukuDir := filepath.Join(path, ".tasuku")
		if info, err := os.Stat(tasukuDir); err == nil && info.IsDir() {
			return tasukuDir
		}

		parent := filepath.Dir(path)
		if parent == path {
			// Reached root, not found
			return ""
		}
		path = parent
	}
}

// GetActiveTask finds the current in_progress task for a worktree.
// Returns the task ID and description, or empty strings if none found.
func GetActiveTask(worktreePath string) (taskID string, description string) {
	tasukuDir := FindTasukuDir(worktreePath)
	if tasukuDir == "" {
		return "", ""
	}

	tasksDir := filepath.Join(tasukuDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskPath := filepath.Join(tasksDir, entry.Name())
		task, err := readTasukuTask(taskPath)
		if err != nil {
			continue
		}

		if task.Status == "in_progress" {
			return task.ID, task.Description
		}
	}

	return "", ""
}

// readTasukuTask reads and parses a single Tasuku task file
func readTasukuTask(path string) (*TasukuTask, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var task TasukuTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// ListTasks returns all tasks from a .tasuku directory
func ListTasks(tasukuDir string) ([]*TasukuTask, error) {
	tasksDir := filepath.Join(tasukuDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, err
	}

	var tasks []*TasukuTask
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskPath := filepath.Join(tasksDir, entry.Name())
		task, err := readTasukuTask(taskPath)
		if err != nil {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// FindTask finds a task by ID in the .tasuku directory
func FindTask(tasukuDir string, taskID string) (*TasukuTask, string, error) {
	tasksDir := filepath.Join(tasukuDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, "", err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskPath := filepath.Join(tasksDir, entry.Name())
		task, err := readTasukuTask(taskPath)
		if err != nil {
			continue
		}

		if task.ID == taskID {
			return task, taskPath, nil
		}
	}

	return nil, "", fmt.Errorf("task not found: %s", taskID)
}

// UpdateTaskStatus updates the status of a task
func UpdateTaskStatus(tasukuDir string, taskID string, newStatus string) error {
	task, taskPath, err := FindTask(tasukuDir, taskID)
	if err != nil {
		return err
	}

	task.Status = newStatus
	task.UpdatedAt = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}
