package cli

import (
	"fmt"
	"os/exec"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose common issues",
	Long: `Diagnose common issues with wt.

This command checks:
- Configuration directory exists
- Caddy is installed
- Proxy is running
- Ports are available
- Registered servers are healthy`,
	RunE: runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println("grove doctor")
	fmt.Println("=========")
	fmt.Println()

	allGood := true

	// Check 1: Config directory
	fmt.Print("Config directory... ")
	if err := config.EnsureDirectories(); err != nil {
		fmt.Printf("FAIL (%v)\n", err)
		allGood = false
	} else {
		fmt.Printf("OK (%s)\n", config.ConfigDir())
	}

	// Check 2: Caddy installed
	fmt.Print("Caddy installed... ")
	caddyPath, err := exec.LookPath("caddy")
	if err != nil {
		fmt.Println("NOT FOUND")
		fmt.Println("  Run: brew install caddy (macOS) or apt install caddy (Linux)")
		allGood = false
	} else {
		fmt.Printf("OK (%s)\n", caddyPath)
	}

	// Check 3: Registry loadable
	fmt.Print("Registry... ")
	reg, err := registry.Load()
	if err != nil {
		fmt.Printf("FAIL (%v)\n", err)
		allGood = false
	} else {
		fmt.Printf("OK (%d servers registered)\n", len(reg.List()))
	}

	// Check 4: Proxy status
	fmt.Print("Proxy... ")
	if reg != nil {
		proxy := reg.GetProxy()
		if proxy.IsRunning() && isProcessRunning(proxy.PID) {
			fmt.Printf("RUNNING (PID: %d)\n", proxy.PID)
		} else {
			fmt.Println("NOT RUNNING")
			fmt.Println("  Run: grove proxy start")
			allGood = false
		}
	} else {
		fmt.Println("UNKNOWN (registry not loaded)")
	}

	// Check 5: HTTP port available (or in use by proxy)
	fmt.Printf("HTTP port (%d)... ", cfg.ProxyHTTPPort)
	if port.IsAvailable(cfg.ProxyHTTPPort) {
		fmt.Println("AVAILABLE")
	} else if reg != nil && reg.GetProxy().IsRunning() {
		fmt.Println("IN USE (by proxy)")
	} else {
		fmt.Println("IN USE (by another process)")
		fmt.Println("  Another process is using this port. Check with: lsof -i :80")
		allGood = false
	}

	// Check 6: HTTPS port available (or in use by proxy)
	fmt.Printf("HTTPS port (%d)... ", cfg.ProxyHTTPSPort)
	if port.IsAvailable(cfg.ProxyHTTPSPort) {
		fmt.Println("AVAILABLE")
	} else if reg != nil && reg.GetProxy().IsRunning() {
		fmt.Println("IN USE (by proxy)")
	} else {
		fmt.Println("IN USE (by another process)")
		fmt.Println("  Another process is using this port. Check with: lsof -i :443")
		allGood = false
	}

	// Check 7: Running servers health
	if reg != nil {
		running := reg.ListRunning()
		if len(running) > 0 {
			fmt.Println()
			fmt.Println("Running servers:")
			for _, s := range running {
				fmt.Printf("  %s (port %d)... ", s.Name, s.Port)
				if isProcessRunning(s.PID) {
					if port.IsListening(s.Port) {
						fmt.Println("HEALTHY")
					} else {
						fmt.Println("PROCESS RUNNING, PORT NOT LISTENING")
					}
				} else {
					fmt.Println("PROCESS NOT RUNNING (stale entry)")
					allGood = false
				}
			}
		}
	}

	fmt.Println()
	if allGood {
		fmt.Println("All checks passed!")
	} else {
		fmt.Println("Some issues found. See above for details.")
	}

	return nil
}
