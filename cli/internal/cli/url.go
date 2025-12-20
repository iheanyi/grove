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
		// Server not registered, but we can still generate the URL
		url := fmt.Sprintf("https://%s.%s", name, cfg.TLD)
		if outputJSON {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"name":       name,
				"url":        url,
				"subdomains": fmt.Sprintf("https://*.%s.%s", name, cfg.TLD),
				"status":     "not_registered",
			})
		}
		fmt.Println(url)
		return nil
	}

	if outputJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"name":       server.Name,
			"url":        server.URL,
			"subdomains": fmt.Sprintf("https://*.%s.%s", server.Name, cfg.TLD),
			"port":       server.Port,
			"status":     server.Status,
		})
	}

	fmt.Println(server.URL)
	return nil
}
