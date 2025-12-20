package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/iheanyi/wt/internal/project"
	"github.com/iheanyi/wt/internal/worktree"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [template]",
	Short: "Create a .wt.yaml configuration file",
	Long: `Create a .wt.yaml configuration file in the current directory.

Available templates:
  rails   - Ruby on Rails project
  node    - Node.js project
  python  - Python project
  go      - Go project

Examples:
  wt init           # Create basic .wt.yaml
  wt init rails     # Create Rails-specific .wt.yaml
  wt init node      # Create Node.js-specific .wt.yaml`,
	RunE:      runInit,
	ValidArgs: []string{"rails", "node", "python", "go"},
}

func init() {
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing .wt.yaml")
}

func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	// Check if .wt.yaml already exists
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath := filepath.Join(cwd, project.ConfigFileName)
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf(".wt.yaml already exists\nUse --force to overwrite")
	}

	// Detect worktree for name suggestion
	wt, _ := worktree.Detect()
	name := "myapp"
	if wt != nil {
		name = wt.Name
	}

	// Get template
	template := ""
	if len(args) > 0 {
		template = args[0]
	}

	// Generate config based on template
	cfg := generateConfig(template, name)

	// Save config
	if err := cfg.Save(cwd); err != nil {
		return fmt.Errorf("failed to write .wt.yaml: %w", err)
	}

	fmt.Printf("Created %s\n", configPath)
	if template != "" {
		fmt.Printf("Using template: %s\n", template)
	}

	return nil
}

func generateConfig(template, name string) *project.Config {
	switch template {
	case "rails":
		return &project.Config{
			Name:    name,
			Command: "bin/dev",
			Env: map[string]string{
				"RAILS_ENV": "development",
			},
			HealthCheck: project.HealthCheckConfig{
				Path: "/up",
			},
			Hooks: project.HooksConfig{
				BeforeStart: []string{
					"bundle install",
					"rails db:migrate",
				},
			},
		}

	case "node":
		return &project.Config{
			Name:    name,
			Command: "npm run dev",
			Env: map[string]string{
				"NODE_ENV": "development",
			},
			Hooks: project.HooksConfig{
				BeforeStart: []string{
					"npm install",
				},
			},
		}

	case "python":
		return &project.Config{
			Name:    name,
			Command: "python manage.py runserver 0.0.0.0:$PORT",
			Env: map[string]string{
				"DJANGO_SETTINGS_MODULE": "config.settings.development",
			},
			Hooks: project.HooksConfig{
				BeforeStart: []string{
					"pip install -r requirements.txt",
					"python manage.py migrate",
				},
			},
		}

	case "go":
		return &project.Config{
			Name:    name,
			Command: "go run .",
			Env: map[string]string{
				"GO_ENV": "development",
			},
			Hooks: project.HooksConfig{
				BeforeStart: []string{
					"go mod download",
				},
			},
		}

	default:
		// Basic config
		return &project.Config{
			Name:    name,
			Command: "",
			Env:     map[string]string{},
		}
	}
}
