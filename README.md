# grove - Worktree Server Manager

A CLI tool with TUI to automatically manage dev servers across git worktrees with clean localhost URLs.

## Features

- **Simple by default**: Access servers at `http://localhost:PORT` with zero configuration
- **Optional subdomain mode**: Enable `https://feature-branch.localhost` URLs when needed
- **Automatic port allocation**: Hash-based port assignment means the same worktree always gets the same port
- **Works with any framework**: Rails, Node, Python, Go, or anything else
- **Interactive TUI**: Beautiful terminal dashboard for managing all your servers
- **fzf-style selector**: Quick fuzzy-find picker for server selection
- **GitHub integration**: View CI status and PR links for each worktree
- **Syntax-highlighted logs**: Colorized log output for Rails, JSON, and common patterns
- **macOS Menubar App**: Native menubar app for quick server management
- **MCP Integration**: Claude Code can manage your dev servers directly
- **JSON output**: Machine-readable output for scripting and automation

## Installation

### Homebrew (macOS)

```bash
brew install iheanyi/tap/grove
```

### From source

```bash
go install github.com/iheanyi/grove/cli/cmd/wt@latest
```

### Build locally

```bash
git clone https://github.com/iheanyi/grove.git
cd grove/cli
make build
```

## Quick Start

```bash
# Navigate to your project (must be a git repo)
cd ~/projects/myapp

# Start the dev server
grove start bin/dev

# Your server is now available at http://localhost:PORT
# grove automatically allocates a consistent port based on worktree name
```

That's it! No proxy setup required in the default port mode.

### Check Status

```bash
# List all servers
grove ls

# Launch the interactive TUI
grove
```

### Optional: Subdomain Mode

If you want pretty URLs like `https://myapp.localhost`, enable subdomain mode:

```bash
# Install Caddy (reverse proxy)
brew install caddy

# Trust the local CA certificate
grove setup

# Edit config to enable subdomain mode
# ~/.config/grove/config.yaml -> url_mode: subdomain

# Start the proxy
grove proxy start

# Now servers are available at https://name.localhost
```

## Usage

### Starting servers

```bash
# Start with a command
grove start bin/dev
grove start rails s
grove start npm run dev

# Use .grove.yaml config (auto-detected)
grove start

# Run in foreground (useful for debugging)
grove start --foreground bin/dev
```

### Managing servers

```bash
# List all servers
grove ls
grove ls --full  # Include CI status and PR links (requires gh CLI)
grove ls --json  # MCP-friendly output

# Interactive selector (fzf-style)
grove select                    # Pick a server interactively
grove open $(grove select)      # Open selected server in browser
grove logs $(grove select)      # View logs for selected server

# Navigate to worktree directory
grove cd feature-auth           # Print path (use with shell function)

# Stop servers
grove stop              # Stop current worktree's server
grove stop feature-auth # Stop by name
grove stop --all        # Stop all servers

# Restart
grove restart

# View status
grove status
grove url

# Open in browser
grove open

# View logs with syntax highlighting
grove logs              # Current worktree
grove logs feature-auth # Named worktree
grove logs -f           # Follow mode (like tail -f)
grove logs --no-color   # Disable highlighting
```

### Shell Integration

Add these to your shell config for quick navigation:

```bash
# Bash/Zsh (~/.bashrc or ~/.zshrc)
grovecd() { cd "$(grove cd "$@")" }

# Fish (~/.config/fish/config.fish)
function grovecd; cd (grove cd $argv); end
```

Then use `grovecd feature-auth` to jump to a worktree's directory.

### Project configuration

Create a `.grove.yaml` in your project root:

```yaml
# Simple project
name: myapp
command: bin/dev
port: 3000  # Optional: override auto-allocated port

env:
  RAILS_ENV: development
  DATABASE_URL: postgres://localhost/myapp_dev

health_check:
  path: /health
  timeout: 30s

hooks:
  before_start:
    - bundle install
    - rails db:migrate
  after_start:
    - echo "Server ready!"
```

Or use a template:

```bash
grove init rails   # Rails project
grove init node    # Node.js project
grove init python  # Python project
grove init go      # Go project
```

### Proxy management

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
```

## TUI

Launch the interactive dashboard:

```bash
grove      # or
grove ui
```

Keyboard shortcuts:
- `enter` / `space` - Start/stop selected server
- `o` - Open in browser
- `l` - View logs
- `p` - Toggle proxy
- `/` - Filter servers
- `?` - Help
- `q` - Quit

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser                               │
│         https://tenant1.feature-auth.localhost              │
└─────────────────────┬───────────────────────────────────────┘
                      │ HTTPS (port 443)
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    grove proxy (Caddy)                          │
│  *.feature-auth.localhost → localhost:3042                  │
│  *.main.localhost → localhost:3000                          │
└─────────────────────┬───────────────────────────────────────┘
                      │ HTTP (internal ports)
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Your Dev Servers                          │
│  Rails on :3042, Node on :3000, etc.                        │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

Global config: `~/.config/grove/config.yaml`

```yaml
# URL mode: "port" (default) or "subdomain"
# - port: Access servers at http://localhost:PORT (simpler, no proxy needed)
# - subdomain: Access servers at https://name.localhost (requires proxy)
url_mode: port

# Port allocation range
port_min: 3000
port_max: 3999

# TLD for local domains (only used in subdomain mode)
tld: localhost

# Server behavior
idle_timeout: 30m
health_check_timeout: 60s

# Notifications
notifications:
  enabled: true
  on_start: true
  on_stop: true
  on_crash: true
```

### URL Modes

grove supports two URL modes:

**Port Mode (default)**
- URLs: `http://localhost:3042`
- Simple and works out of the box
- No proxy required
- Best for apps that use subdomains internally (multi-tenant apps)

**Subdomain Mode**
- URLs: `https://feature-auth.localhost`
- Wildcard subdomains: `https://tenant.feature-auth.localhost`
- Requires running `grove proxy start`
- HTTPS with automatic local certificates

To switch modes, edit `~/.config/grove/config.yaml`:

```yaml
# For port mode (default)
url_mode: port

# For subdomain mode
url_mode: subdomain
```

## MCP Server for Claude Code

The `grove mcp` command runs grove as an MCP server, allowing Claude Code to manage your dev servers directly. This enables seamless browser automation workflows where Claude can:

1. Start a dev server for your current worktree
2. Get the URL for browser testing
3. Stop servers when done

### Configuring Claude Code

The easiest way to install is using the built-in command:

```bash
grove mcp install
```

This automatically registers grove with Claude Code. Verify with:

```bash
claude mcp list
```

Alternatively, manually add to your Claude Code configuration:

```bash
claude mcp add -s user -t stdio grove /path/to/wt mcp
```

**Restart Claude Code** to load the MCP server.

### Available MCP Tools

| Tool | Description |
|------|-------------|
| `wt_list` | List all registered dev servers and their URLs |
| `wt_start` | Start a dev server for a git worktree |
| `wt_stop` | Stop a running dev server by name |
| `wt_url` | Get the URL for a worktree's dev server |
| `wt_status` | Get detailed status of a dev server |

### Example Claude Code Workflow

```
User: Start a dev server for this project and open it in the browser

Claude: [Uses wt_start to start the server]
        Server started at https://myproject.localhost

        [Uses browser MCP to navigate to https://myproject.localhost]
        I can now see your application running...
```

## JSON Output

The `--json` flag provides machine-readable output for scripting:

```bash
grove ls --json
```

**Port mode (default):**
```json
{
  "servers": [
    {
      "name": "feature-auth",
      "url": "http://localhost:3042",
      "port": 3042,
      "status": "running",
      "path": "/Users/you/projects/myapp-feature-auth",
      "uptime": "2h 15m",
      "log_file": "~/.config/grove/logs/feature-auth.log"
    }
  ],
  "proxy": null,
  "url_mode": "port"
}
```

**Subdomain mode:**
```json
{
  "servers": [
    {
      "name": "feature-auth",
      "url": "https://feature-auth.localhost",
      "subdomains": "https://*.feature-auth.localhost",
      "port": 3042,
      "status": "running",
      "path": "/Users/you/projects/myapp-feature-auth",
      "uptime": "2h 15m"
    }
  ],
  "proxy": {
    "status": "running",
    "http_port": 80,
    "https_port": 443
  },
  "url_mode": "subdomain"
}
```

## macOS Menubar App

A native macOS menubar app is available for quick server management without the terminal.

### Features

- See all running/stopped servers at a glance
- Start/stop servers with one click
- Open server URLs in browser
- View server logs
- Copy URLs to clipboard

### Building

```bash
cd menubar/GroveMenubar
make build   # Build the app
make run     # Build and run
```

The app bundle will be at `.build/GroveMenubar.app`. You can drag it to your Applications folder.

### Note

The menubar app communicates with the `wt` CLI. Make sure `wt` is installed and accessible. The app looks for `wt` in these locations:
- `~/development/go/bin/wt`
- `/usr/local/bin/wt`
- `/opt/homebrew/bin/wt`
- `~/go/bin/wt`

## Troubleshooting

### Docker Desktop Port Conflict

Docker Desktop on macOS binds to ports 80 and 443 by default, which conflicts with the grove proxy. You have several options:

**Option 1: Use alternate ports for grove (recommended)**

Edit `~/.config/grove/config.yaml`:

```yaml
proxy_http_port: 8080
proxy_https_port: 8443
```

Then access your servers at `https://myapp.localhost:8443` instead.

**Option 2: Disable Docker's port bindings**

1. Open Docker Desktop → Settings → Resources → Network
2. Uncheck "Use kernel networking for UDP" and related options
3. Or quit Docker Desktop when using grove

**Option 3: Stop Docker's internal proxy**

```bash
# Find and stop the Docker process using ports 80/443
lsof -i :443 | grep com.docker
# Note the PID and:
kill <PID>
```

### DNS Resolution for *.localhost

On most systems, `*.localhost` should resolve to `127.0.0.1` automatically. If not:

**macOS**: Add to `/etc/hosts`:
```
127.0.0.1 myapp.localhost
127.0.0.1 tenant1.myapp.localhost
```

Or use [dnsmasq](https://thekelleys.org.uk/dnsmasq/doc.html) for wildcard DNS.

## E2E Testing Guide

Here's how to test grove end-to-end:

### 1. Build and Install

```bash
cd cli
make build
sudo make install-local  # Installs to /usr/local/bin
```

### 2. Initial Setup

```bash
# Install Caddy
brew install caddy

# Trust the CA certificate (one-time)
grove setup
# Answer 'y' when prompted to trust the CA
```

### 3. Start the Proxy

```bash
grove proxy start
grove proxy status  # Verify it's running
```

### 4. Start a Dev Server

```bash
# Navigate to any git repo
cd ~/your-project

# Start the server
grove start bin/dev  # or: npm run dev, rails s, etc.

# Check it's running
grove ls
```

### 5. Test Access

```bash
# Direct HTTP (always works)
curl http://localhost:<port>

# Via proxy (requires Caddy trust + no Docker conflict)
curl -k https://your-project.localhost
```

### 6. Test MCP Integration

```bash
# Test MCP server directly
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"}}}' | grove mcp

# Should return: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05",...}}

# List tools
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | grove mcp
```

### 7. Cleanup

```bash
grove stop           # Stop current server
grove stop --all     # Stop all servers
grove proxy stop     # Stop the proxy
```

## Requirements

- Go 1.21+
- [Caddy](https://caddyserver.com/) (installed via `grove setup` or `brew install caddy`)
- macOS or Linux

## License

MIT
