package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/iheanyi/wt/internal/config"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "One-time setup for wt",
	Long: `Perform one-time setup for wt.

This command will:
1. Create necessary directories
2. Generate a local CA certificate
3. Install the CA certificate to your system keychain (requires sudo)

After setup, your browser will trust HTTPS connections to *.localhost domains.`,
	RunE: runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("Setting up wt...")
	fmt.Println()

	// Step 1: Create directories
	fmt.Print("Creating directories... ")
	if err := config.EnsureDirectories(); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("failed to create directories: %w", err)
	}
	fmt.Println("done")

	// Step 2: Create default config if not exists
	fmt.Print("Creating default config... ")
	configPath := config.ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := config.Default()
		if err := cfg.Save(configPath); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("failed to create config: %w", err)
		}
		fmt.Println("done")
	} else {
		fmt.Println("skipped (already exists)")
	}

	// Step 3: Check for Caddy
	fmt.Print("Checking for Caddy... ")
	if _, err := exec.LookPath("caddy"); err != nil {
		fmt.Println("NOT FOUND")
		fmt.Println()
		fmt.Println("Caddy is required for the reverse proxy.")
		fmt.Println("Install it with:")
		switch runtime.GOOS {
		case "darwin":
			fmt.Println("  brew install caddy")
		case "linux":
			fmt.Println("  sudo apt install caddy")
			fmt.Println("  # or visit https://caddyserver.com/docs/install")
		default:
			fmt.Println("  Visit https://caddyserver.com/docs/install")
		}
		fmt.Println()
		fmt.Println("After installing Caddy, run 'wt setup' again.")
		return nil
	}
	fmt.Println("found")

	// Step 4: Trust Caddy's local CA (if needed)
	fmt.Println()
	fmt.Println("To enable HTTPS for local domains, Caddy needs to install a local CA certificate.")
	fmt.Println("This requires administrator privileges.")
	fmt.Println()
	fmt.Print("Trust local CA certificate? [y/N] ")

	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "Y" {
		// First, we need to start Caddy briefly to generate the CA certificate
		fmt.Print("Generating CA certificate... ")
		startCmd := exec.Command("caddy", "start")
		if err := startCmd.Run(); err != nil {
			fmt.Println("FAILED")
			fmt.Printf("Could not start Caddy to generate CA: %v\n", err)
			fmt.Println("Try running these commands manually:")
			fmt.Println("  caddy start")
			fmt.Println("  sudo caddy trust")
			fmt.Println("  caddy stop")
		} else {
			fmt.Println("done")

			// Now trust the CA (requires sudo)
			fmt.Print("Installing CA certificate (may require password)... ")
			trustCmd := exec.Command("sudo", "caddy", "trust")
			trustCmd.Stdin = os.Stdin
			trustCmd.Stdout = os.Stdout
			trustCmd.Stderr = os.Stderr
			if err := trustCmd.Run(); err != nil {
				fmt.Println("FAILED")
				fmt.Printf("Could not install CA certificate: %v\n", err)
				fmt.Println("Try running 'sudo caddy trust' manually")
			} else {
				fmt.Println("done")
			}

			// Stop Caddy so wt proxy can manage it
			fmt.Print("Stopping Caddy... ")
			stopCmd := exec.Command("caddy", "stop")
			if err := stopCmd.Run(); err != nil {
				// Not critical, just warn
				fmt.Printf("warning: %v\n", err)
			} else {
				fmt.Println("done")
			}
		}
	} else {
		fmt.Println("Skipped. Run these commands manually to enable HTTPS:")
		fmt.Println("  caddy start")
		fmt.Println("  sudo caddy trust")
		fmt.Println("  caddy stop")
	}

	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Start the proxy: wt proxy start")
	fmt.Println("  2. Start a server: wt start bin/dev")
	fmt.Println("  3. Open in browser: wt open")

	return nil
}
