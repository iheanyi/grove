# Grove - Git Worktree & Dev Server Manager

A powerful CLI tool with TUI and macOS menubar app for managing git worktrees and their dev servers with clean localhost URLs.

## What is Grove?

Grove is a **worktree-first development workflow tool** that combines:

1. **Git worktree management** - Create, switch, and manage worktrees effortlessly
2. **Dev server orchestration** - Start, stop, and monitor servers across worktrees
3. **Clean local URLs** - Access servers at `http://localhost:PORT` or `https://branch.localhost`
4. **Interactive TUI** - Beautiful terminal dashboard for managing everything
5. **macOS Menubar App** - Native app for quick server access without leaving your flow

## Features

### Worktree Management
- **Clone for worktrees**: Clone repos with optimal structure for multiple worktrees
- **Create worktrees**: Create new worktrees with `grove new feature-name`
- **Switch worktrees**: Open terminal tabs in any worktree with `grove switch`
- **Auto-discovery**: Scan directories and find all your worktrees
- **Prune worktrees**: Clean up stale worktrees with `grove prune`

### Server Management
- **Simple by default**: Access servers at `http://localhost:PORT` with zero configuration
- **Optional subdomain mode**: Enable `https://feature-branch.localhost` URLs when needed
- **Automatic port allocation**: Hash-based port assignment means the same worktree always gets the same port
- **Works with any framework**: Rails, Node, Python, Go, or anything else
- **Attach external servers**: Register already-running servers with `grove attach`
- **Syntax-highlighted logs**: Colorized log output for Rails, JSON, and common patterns
- **Health checks**: Automatic server health monitoring

### Interfaces
- **Interactive TUI**: Beautiful terminal dashboard with real-time updates
- **fzf-style selector**: Quick fuzzy-find picker for server selection
- **macOS Menubar App**: Native menubar app with search, notifications, and quick actions
- **GitHub integration**: View CI status and PR links for each worktree
- **MCP Integration**: Claude Code can manage your dev servers directly

### Power User Features
- **Shell completion**: Tab completion for bash, zsh, fish, and PowerShell
- **JSON output**: Machine-readable output for scripting and automation
- **Project configs**: `.grove.yaml` files for project-specific settings
- **Hooks**: Run commands before/after server start/stop
- **Idle timeout**: Auto-stop servers after period of inactivity

## Installation

### Homebrew (macOS)

```bash
brew install iheanyi/tap/grove
```

### From source

```bash
go install github.com/iheanyi/grove/cli/cmd/grove@latest
```

### Build locally

```bash
git clone https://github.com/iheanyi/grove.git
cd grove/cli
make build
```

### Enable shell completion

```bash
# Bash
grove completion bash > /usr/local/etc/bash_completion.d/grove

# Zsh
grove completion zsh > "${fpath[1]}/_grove"

# Fish
grove completion fish > ~/.config/fish/completions/grove.fish
```

## Quick Start

### For a new project (worktree-optimized clone)

```bash
# Clone with worktree-friendly structure
grove clone https://github.com/user/repo.git

# Creates:
# repo/
# ├── .bare/     # Shared git objects
# └── main/      # Main branch worktree

cd repo/main
grove start npm run dev
```

### For an existing project

```bash
# Navigate to your project (must be a git repo)
cd ~/projects/myapp

# Start the dev server
grove start bin/dev

# Your server is now available at http://localhost:PORT
```

### Create a feature branch worktree

```bash
# Create new worktree for a feature
grove new feature-auth

# Switch to it (opens new terminal tab)
grove switch myapp-feature-auth

# Start the dev server
grove start
```

## CLI Reference

### Worktree Commands

```bash
# Clone a repo with worktree-friendly structure
grove clone <repo-url> [directory]
grove clone https://github.com/user/repo.git --branch develop

# Create a new worktree
grove new <branch-name> [base-branch]
grove new feature-auth              # From main/master
grove new bugfix-123 develop        # From develop branch

# Switch to a worktree (opens new terminal)
grove switch <worktree-name>
grove switch myapp-feature-auth --start  # Also start dev server

# Prune stale worktrees
grove prune           # Interactive selection
grove prune --all     # Remove all stale entries
grove prune --dry-run # Preview what would be removed

# Discover worktrees in a directory
grove discover                    # Scan current directory
grove discover ~/development      # Scan specific directory
grove discover --register         # Register all discovered worktrees
grove discover --register --start # Register and start all

# Show project information
grove info            # Comprehensive project overview
grove info --json     # JSON output
```

### Server Commands

```bash
# Start a dev server
grove start                   # Use command from .grove.yaml
grove start bin/dev           # Explicit command
grove start rails s
grove start npm run dev
grove start --foreground      # Run in foreground (for debugging)

# Stop servers
grove stop              # Stop current worktree's server
grove stop feature-auth # Stop by name
grove stop --all        # Stop all servers

# Restart
grove restart

# List all servers
grove ls
grove ls --full  # Include CI status and PR links
grove ls --json  # Machine-readable output

# Server URLs
grove url               # Print URL for current worktree
grove url --json        # JSON output

# Open in browser
grove open
grove open feature-auth

# View logs with syntax highlighting
grove logs              # Current worktree
grove logs feature-auth # Named worktree
grove logs -f           # Follow mode (like tail -f)
grove logs --no-color   # Disable highlighting

# Status and health
grove status
```

### Attach External Servers

```bash
# Register an already-running server
grove attach 3000                    # Attach to port 3000
grove attach 3000 --name my-server   # Custom name
grove attach 8080 --url /api         # Only route /api paths

# Remove from tracking without stopping
grove detach
grove detach my-server
```

### Interactive Selection

```bash
# fzf-style server picker
grove select                    # Pick a server interactively
grove open $(grove select)      # Open selected server in browser
grove logs $(grove select)      # View logs for selected server
grove stop $(grove select)      # Stop selected server
```

### Project Configuration

```bash
# Create .grove.yaml interactively or from template
grove init              # Interactive
grove init rails        # Rails template
grove init node         # Node.js template
grove init python       # Python template
grove init go           # Go template
```

### Proxy Management (for subdomain mode)

```bash
grove proxy start   # Start the reverse proxy
grove proxy stop    # Stop the proxy
grove proxy status  # Check status
grove proxy routes  # List all registered routes
```

### Diagnostics

```bash
grove doctor   # Diagnose common issues
grove cleanup  # Remove stale registry entries
grove setup    # One-time setup (trust CA cert for HTTPS)
```

## TUI Dashboard

Launch the interactive dashboard:

```bash
grove      # or
grove ui
```

**Keyboard shortcuts:**
| Key | Action |
|-----|--------|
| `enter` / `space` | Start/stop selected server |
| `o` | Open in browser |
| `l` | View logs |
| `p` | Toggle proxy |
| `/` | Filter servers |
| `?` | Help |
| `q` | Quit |

Features:
- Real-time server status updates
- Log streaming with syntax highlighting
- Fuzzy search for quick server selection
- Spinner animations during operations
- Toast notifications for actions

## Shell Integration

Add these to your shell config for quick worktree navigation:

```bash
# Bash/Zsh (~/.bashrc or ~/.zshrc)
grovecd() { cd "$(grove cd "$@")" }

# Fish (~/.config/fish/config.fish)
function grovecd; cd (grove cd $argv); end
```

Then use `grovecd feature-auth` to jump to a worktree's directory.

## Project Configuration

Create a `.grove.yaml` in your project root:

```yaml
# .grove.yaml
name: myapp                    # Override auto-detected name
command: bin/dev               # Default command for `grove start`
port: 3000                     # Optional: override auto-allocated port

env:
  RAILS_ENV: development
  DATABASE_URL: postgres://localhost/myapp_dev

health_check:
  path: /health                # Endpoint to ping
  timeout: 30s                 # Max wait time

hooks:
  before_start:
    - bundle install
    - rails db:migrate
  after_start:
    - echo "Server ready!"
```

## macOS Menubar App

A native macOS menubar app for quick server management without the terminal.

### Installation

**Via Homebrew (recommended):**

```bash
# Install both CLI and menubar app
brew install iheanyi/tap/grove
brew install --cask iheanyi/tap/grove-menubar
```

**Manual download:**

1. Download `Grove-X.X.X-macos.zip` from [GitHub Releases](https://github.com/iheanyi/grove/releases)
2. Extract and move `Grove.app` to `/Applications`
3. Right-click → Open (first time only, to bypass Gatekeeper)

**Build from source:**

```bash
git clone https://github.com/iheanyi/grove.git
cd grove
make install-menubar  # Builds and installs to /Applications
```

### Features

- **Server list**: See all running/stopped servers at a glance
- **Quick actions**: Start/stop servers with one click
- **Search**: Filter servers with ⌘K
- **Open in browser**: One-click to open server URL
- **View logs**: Quick access to server logs
- **Copy URL**: Copy server URL to clipboard
- **Toast notifications**: Get notified when servers start/stop/crash
- **Preferences**: Auto-start, notification settings, dock icon toggle

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `⌘K` | Open search |
| `⌘,` | Open preferences |
| `⌘Q` | Quit |

### Requirements

- macOS 14.0 (Sonoma) or later
- Grove CLI installed

The menubar app communicates with the `grove` CLI. The app looks for `grove` in these locations:
- `/opt/homebrew/bin/grove` (Homebrew on Apple Silicon)
- `/usr/local/bin/grove` (Homebrew on Intel / manual install)
- `~/go/bin/grove` (Go install)

## MCP Server for Claude Code

The `grove mcp` command runs grove as an MCP server, allowing Claude Code to manage your dev servers directly.

### Installation

```bash
grove mcp install
```

This automatically registers grove with Claude Code. Verify with:

```bash
claude mcp list
```

### Available MCP Tools

| Tool | Description |
|------|-------------|
| `grove_list` | List all registered dev servers and their URLs |
| `grove_start` | Start a dev server for a git worktree |
| `grove_stop` | Stop a running dev server by name |
| `grove_url` | Get the URL for a worktree's dev server |
| `grove_status` | Get detailed status of a dev server |

## Configuration

Global config: `~/.config/grove/config.yaml`

```yaml
# URL mode: "port" (default) or "subdomain"
url_mode: port

# Port allocation range
port_min: 3000
port_max: 3999

# TLD for local domains (only used in subdomain mode)
tld: localhost

# Server behavior
idle_timeout: 30m          # Auto-stop after inactivity (0 to disable)
health_check_timeout: 60s

# Notifications
notifications:
  enabled: true
  on_start: true
  on_stop: true
  on_crash: true
```

### URL Modes

**Port Mode (default)**
- URLs: `http://localhost:3042`
- Simple and works out of the box
- No proxy required
- Best for most development workflows

**Subdomain Mode**
- URLs: `https://feature-auth.localhost`
- Wildcard subdomains: `https://tenant.feature-auth.localhost`
- Requires running `grove proxy start`
- HTTPS with automatic local certificates

## JSON Output

The `--json` flag provides machine-readable output for scripting:

```bash
grove ls --json
grove info --json
grove url --json
```

Example output:
```json
{
  "servers": [
    {
      "name": "feature-auth",
      "url": "http://localhost:3042",
      "port": 3042,
      "status": "running",
      "path": "/Users/you/projects/myapp-feature-auth",
      "uptime": "2h 15m"
    }
  ],
  "proxy": null,
  "url_mode": "port"
}
```

## Troubleshooting

### Docker Desktop Port Conflict

Docker Desktop on macOS may bind to ports 80/443. Options:

**Option 1: Use alternate ports for grove**

Edit `~/.config/grove/config.yaml`:

```yaml
proxy_http_port: 8080
proxy_https_port: 8443
```

**Option 2: Disable Docker's port bindings**

Open Docker Desktop → Settings → Resources → Network

### DNS Resolution for *.localhost

On most systems, `*.localhost` should resolve to `127.0.0.1` automatically. If not:

**macOS**: Add to `/etc/hosts`:
```
127.0.0.1 myapp.localhost
```

Or use [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) for wildcard DNS.

## Requirements

- Go 1.21+
- [Caddy](https://caddyserver.com/) (for subdomain mode only)
- macOS or Linux

## License

MIT
