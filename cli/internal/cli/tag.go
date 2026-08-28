package cli

import (
	"fmt"
	"strings"

	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag <name> [tags...]",
	Short: "Manage tags for a worktree/server",
	Long: `Add, remove, or list tags for a worktree/server.

Tags help you organize and filter worktrees. You can use tags to group
related projects, mark priority, or any other categorization.

Examples:
  grove tag my-feature frontend api      # Add 'frontend' and 'api' tags
  grove tag my-feature --remove api      # Remove 'api' tag
  grove tag my-feature --list            # List all tags for my-feature
  grove ls --tag frontend                # List worktrees with 'frontend' tag
  grove ls --tag frontend --tag api      # List with 'frontend' OR 'api' tag`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTag,
}

func init() {
	tagCmd.Flags().StringSlice("remove", nil, "Tags to remove")
	tagCmd.Flags().Bool("list", false, "List tags for the worktree")
}

func runTag(cmd *cobra.Command, args []string) error {
	name := args[0]
	tagsToAdd := args[1:]

	removeFlag, _ := cmd.Flags().GetStringSlice("remove")
	listFlag, _ := cmd.Flags().GetBool("list")

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get the server
	server, exists := reg.Get(name)
	if !exists {
		return fmt.Errorf("server '%s' not found in registry", name)
	}

	// Handle --list flag
	if listFlag {
		if len(server.Tags) == 0 {
			fmt.Printf("%s has no tags\n", name)
		} else {
			fmt.Printf("Tags for %s: %s\n", name, strings.Join(server.Tags, ", "))
		}
		return nil
	}

	// Handle --remove flag
	if len(removeFlag) > 0 {
		for _, tag := range removeFlag {
			if server.RemoveTag(tag) {
				fmt.Printf("Removed tag '%s' from %s\n", tag, name)
			} else {
				fmt.Printf("Tag '%s' not found on %s\n", tag, name)
			}
		}
		return reg.Set(server)
	}

	// Add new tags
	if len(tagsToAdd) == 0 {
		// No tags specified and no flags, show current tags
		if len(server.Tags) == 0 {
			fmt.Printf("%s has no tags\n", name)
		} else {
			fmt.Printf("Tags for %s: %s\n", name, strings.Join(server.Tags, ", "))
		}
		return nil
	}

	for _, tag := range tagsToAdd {
		// Normalize tag (lowercase, no spaces)
		tag = normalizeTag(tag)
		if tag == "" {
			continue
		}

		if server.AddTag(tag) {
			fmt.Printf("Added tag '%s' to %s\n", tag, name)
		} else {
			fmt.Printf("Tag '%s' already exists on %s\n", tag, name)
		}
	}

	return reg.Set(server)
}

// normalizeTag normalizes a tag string (lowercase, alphanumeric and hyphens only)
func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.ToLower(tag)

	// Replace spaces and underscores with hyphens
	tag = strings.ReplaceAll(tag, " ", "-")
	tag = strings.ReplaceAll(tag, "_", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, r := range tag {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}
