package cli

import (
	"fmt"

	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	loadRegistry          = registry.Load
	getListeningPortByPID = port.GetPrimaryListeningPortByPID
	setServerInRegistry   = func(reg *registry.Registry, server *registry.Server) error {
		return reg.Set(server)
	}
)

var syncPortsCmd = &cobra.Command{
	Use:   "sync-ports [name]",
	Short: "Sync registered ports with runtime listening ports",
	Long: `Sync registered server ports with the process' actual listening ports.

This is useful when a process starts on a different port than the one currently
stored in Grove's registry.

Examples:
  grove sync-ports                # Sync current worktree server
  grove sync-ports my-server      # Sync a named server
  grove sync-ports --all          # Sync all running servers`,
	RunE: runSyncPorts,
}

func init() {
	syncPortsCmd.Flags().Bool("all", false, "Sync all running servers")
}

func runSyncPorts(cmd *cobra.Command, args []string) error {
	syncAll, _ := cmd.Flags().GetBool("all")

	reg, err := loadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	var targets []*registry.Server
	if syncAll {
		targets = reg.ListRunning()
		if len(targets) == 0 {
			fmt.Println("No running servers to sync.")
			return nil
		}
	} else {
		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			wt, detectErr := worktree.Detect()
			if detectErr != nil {
				return fmt.Errorf("failed to detect worktree: %w\nRun this command from inside a worktree, pass a server name, or use --all", detectErr)
			}
			name = wt.Name
		}

		server, ok := reg.Get(name)
		if !ok {
			return fmt.Errorf("no server registered for '%s'", name)
		}
		targets = []*registry.Server{server}
	}

	var updated, unchanged, skipped int
	for _, s := range targets {
		if !s.IsRunning() {
			skipped++
			fmt.Printf("~ %s: skipped (not running)\n", s.Name)
			continue
		}
		if s.PID <= 0 {
			skipped++
			fmt.Printf("~ %s: skipped (missing PID)\n", s.Name)
			continue
		}

		actualPort := getListeningPortByPID(s.PID)
		if actualPort == 0 {
			skipped++
			fmt.Printf("~ %s: skipped (no LISTEN port detected for PID %d)\n", s.Name, s.PID)
			continue
		}

		if actualPort == s.Port {
			unchanged++
			fmt.Printf("= %s: already in sync at :%d\n", s.Name, s.Port)
			continue
		}

		// Guard against obvious conflicts with another running server.
		conflict := false
		for _, other := range reg.ListRunning() {
			if other.Name != s.Name && other.Port == actualPort {
				conflict = true
				break
			}
		}
		if conflict {
			skipped++
			fmt.Printf("! %s: detected :%d but skipped due to running-port conflict\n", s.Name, actualPort)
			continue
		}

		oldPort := s.Port
		s.Port = actualPort
		s.URL = cfg.ServerURL(s.Name, actualPort)
		if setErr := setServerInRegistry(reg, s); setErr != nil {
			skipped++
			fmt.Printf("! %s: failed to update from :%d to :%d (%v)\n", s.Name, oldPort, actualPort, setErr)
			continue
		}

		updated++
		fmt.Printf("✓ %s: synced :%d -> :%d\n", s.Name, oldPort, actualPort)
	}

	fmt.Printf("\nSummary: %d updated, %d unchanged, %d skipped\n", updated, unchanged, skipped)
	return nil
}
