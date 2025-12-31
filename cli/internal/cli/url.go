package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var urlCmd = &cobra.Command{
	Use:   "url [name]",
	Short: "Print the URL for a server",
	Long: `Print the URL for the current worktree's server or a named server.

Examples:
  grove url              # Print URL for current worktree
  grove url feature-auth # Print URL for named server
  grove url --json       # Output as JSON`,
	RunE: runURL,
}

func init() {
	urlCmd.Flags().Bool("json", false, "Output as JSON")
}

func runURL(cmd *cobra.Command, args []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Determine which server
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use current worktree
		wt, err := worktree.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect worktree: %w", err)
		}
		name = wt.Name
	}

	server, ok := reg.Get(name)
	if !ok {
		// Server not registered - in port mode we can't know the URL without a port
		if !cfg.IsSubdomainMode() {
			return fmt.Errorf("server '%s' is not registered (port unknown)", name)
		}
		url := cfg.ServerURL(name, 0)
		if outputJSON {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"name":       name,
				"url":        url,
				"subdomains": cfg.SubdomainURL(name),
				"status":     "not_registered",
			})
		}
		fmt.Println(url)
		return nil
	}

	if outputJSON {
		result := map[string]interface{}{
			"name":   server.Name,
			"url":    server.URL,
			"port":   server.Port,
			"status": server.Status,
		}
		if cfg.IsSubdomainMode() {
			result["subdomains"] = cfg.SubdomainURL(server.Name)
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Println(server.URL)
	return nil
}
