package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/iheanyi/grove/internal/discovery"
	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/styles"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run as an MCP server for Claude Code integration",
	Long: `Run grove as an MCP (Model Context Protocol) server.

This allows Claude Code to manage your dev servers directly. Configure
Claude Code to use this command as an MCP server:

  {
    "mcpServers": {
      "grove": {
        "command": "grove",
        "args": ["mcp"]
      }
    }
  }

Available tools:
  - grove_list: List all registered dev servers
  - grove_start: Start a dev server for a git worktree
  - grove_stop: Stop a running dev server
  - grove_url: Get the URL for a worktree's dev server
  - grove_status: Get detailed status of a dev server`,
	Run: func(cmd *cobra.Command, args []string) {
		runMCPServer()
	},
}

var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install grove as an MCP server in a supported provider",
	Long: `Configure a supported provider to use grove as an MCP server.

Supported providers:
  - claude-code (default): Installs to Claude Code (~/.claude/settings.json)
  - gemini: Installs to Gemini CLI (~/.gemini/settings.json)
  - opencode: Installs to OpenCode config (opencode.json)
  - cursor: Installs to Cursor IDE (~/.cursor/mcp.json)
  - codex: Installs to OpenAI Codex CLI (~/.codex/config.toml)

For OpenCode, Cursor, and Gemini, use --global to install to the global config
instead of the local project config.

Examples:
  grove mcp install                      # Install for Claude Code
  grove mcp install -p gemini            # Install for Gemini (local)
  grove mcp install -p gemini --global   # Install for Gemini (global)
  grove mcp install -p opencode          # Install for OpenCode (local)
  grove mcp install -p opencode --global # Install for OpenCode (global)
  grove mcp install -p cursor            # Install for Cursor (local)
  grove mcp install -p cursor --global   # Install for Cursor (global)
  grove mcp install -p codex             # Install for Codex (always global)

After installation, restart the provider to load the MCP server.`,
	RunE: runMCPInstall,
}

var (
	mcpInstallProvider string
	mcpInstallGlobal   bool
)

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpInstallCmd)

	mcpInstallCmd.Flags().StringVarP(&mcpInstallProvider, "provider", "p", "claude-code", "Provider to install for (claude-code, gemini, opencode, cursor, codex)")
	mcpInstallCmd.Flags().BoolVarP(&mcpInstallGlobal, "global", "g", false, "Install globally (for opencode, cursor, and gemini)")
}

func runMCPInstall(cmd *cobra.Command, args []string) error {
	// Find grove binary path
	grovePath, err := exec.LookPath("grove")
	if err != nil {
		// Fall back to current executable
		grovePath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find grove binary: %w", err)
		}
	}

	// Resolve symlinks to get actual path
	grovePath, err = filepath.EvalSymlinks(grovePath)
	if err != nil {
		return fmt.Errorf("failed to resolve grove path: %w", err)
	}

	switch mcpInstallProvider {
	case "claude-code":
		return installForClaudeCode(grovePath)
	case "gemini":
		return installForGemini(grovePath, mcpInstallGlobal)
	case "opencode":
		return installForOpenCode(grovePath, mcpInstallGlobal)
	case "cursor":
		return installForCursor(grovePath, mcpInstallGlobal)
	case "codex":
		return installForCodex(grovePath)
	default:
		return fmt.Errorf("unknown provider: %s (supported: claude-code, gemini, opencode, cursor, codex)", mcpInstallProvider)
	}
}

func installForClaudeCode(grovePath string) error {
	// Use claude mcp add command to properly register the MCP server
	claudeCmd := exec.Command("claude", "mcp", "add", "-s", "user", "-t", "stdio", "grove", grovePath, "mcp")
	output, err := claudeCmd.CombinedOutput()
	if err != nil {
		// Check if it's because it already exists
		if strings.Contains(string(output), "already exists") {
			fmt.Println("grove MCP server is already installed.")
			fmt.Println("\nTo reinstall, first remove it with: claude mcp remove grove")
			return nil
		}
		return fmt.Errorf("failed to install MCP server: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("✓ Installed grove MCP server in Claude Code\n\n")
	fmt.Printf("  Binary path: %s\n\n", grovePath)
	fmt.Println("The MCP server is now available. Run 'claude mcp list' to verify.")
	printMCPTools()

	return nil
}

func installForGemini(grovePath string, global bool) error {
	var configPath string

	if global {
		// Global config: ~/.gemini/settings.json
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ".gemini")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "settings.json")
	} else {
		// Local config: .gemini/settings.json in current directory
		configDir := ".gemini"
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "settings.json")
	}

	// Read existing config or create new one
	config := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config at %s: %w", configPath, err)
		}
	}

	// Get or create the mcpServers section
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Check if grove already exists
	if _, exists := mcpServers["grove"]; exists {
		fmt.Printf("grove MCP server is already configured in %s\n", configPath)
		fmt.Println("\nTo reinstall, remove the 'grove' entry from the 'mcpServers' section and run this command again.")
		return nil
	}

	// Add grove MCP server configuration (Gemini/Cursor format)
	mcpServers["grove"] = map[string]interface{}{
		"command": grovePath,
		"args":    []string{"mcp"},
	}
	config["mcpServers"] = mcpServers

	// Write updated config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	location := "local"
	if global {
		location = "global"
	}

	fmt.Printf("✓ Installed grove MCP server in Gemini (%s)\n\n", location)
	fmt.Printf("  Config file: %s\n", configPath)
	fmt.Printf("  Binary path: %s\n\n", grovePath)
	fmt.Println("Restart Gemini CLI to load the MCP server.")
	printMCPTools()

	return nil
}

func installForOpenCode(grovePath string, global bool) error {
	var configPath string

	if global {
		// Global config: ~/.config/opencode/opencode.json
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ".config", "opencode")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "opencode.json")
	} else {
		// Local config: opencode.json in current directory
		configPath = "opencode.json"
	}

	// Read existing config or create new one
	config := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config at %s: %w", configPath, err)
		}
	}

	// Get or create the mcp section
	mcpSection, ok := config["mcp"].(map[string]interface{})
	if !ok {
		mcpSection = make(map[string]interface{})
	}

	// Check if grove already exists
	if _, exists := mcpSection["grove"]; exists {
		fmt.Printf("grove MCP server is already configured in %s\n", configPath)
		fmt.Println("\nTo reinstall, remove the 'grove' entry from the 'mcp' section and run this command again.")
		return nil
	}

	// Add grove MCP server configuration
	mcpSection["grove"] = map[string]interface{}{
		"type":    "local",
		"command": []string{grovePath, "mcp"},
		"enabled": true,
	}
	config["mcp"] = mcpSection

	// Write updated config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	location := "local"
	if global {
		location = "global"
	}

	fmt.Printf("✓ Installed grove MCP server in OpenCode (%s)\n\n", location)
	fmt.Printf("  Config file: %s\n", configPath)
	fmt.Printf("  Binary path: %s\n\n", grovePath)
	fmt.Println("Restart OpenCode to load the MCP server.")
	printMCPTools()

	return nil
}

func installForCursor(grovePath string, global bool) error {
	var configPath string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	if global {
		// Global config: ~/.cursor/mcp.json
		configDir := filepath.Join(homeDir, ".cursor")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "mcp.json")
	} else {
		// Local config: .cursor/mcp.json in current directory
		configDir := ".cursor"
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "mcp.json")
	}

	// Read existing config or create new one
	config := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config at %s: %w", configPath, err)
		}
	}

	// Get or create the mcpServers section
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Check if grove already exists
	if _, exists := mcpServers["grove"]; exists {
		fmt.Printf("grove MCP server is already configured in %s\n", configPath)
		fmt.Println("\nTo reinstall, remove the 'grove' entry from the 'mcpServers' section and run this command again.")
		return nil
	}

	// Add grove MCP server configuration (Cursor format)
	mcpServers["grove"] = map[string]interface{}{
		"command": grovePath,
		"args":    []string{"mcp"},
	}
	config["mcpServers"] = mcpServers

	// Write updated config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	location := "local"
	if global {
		location = "global"
	}

	fmt.Printf("✓ Installed grove MCP server in Cursor (%s)\n\n", location)
	fmt.Printf("  Config file: %s\n", configPath)
	fmt.Printf("  Binary path: %s\n\n", grovePath)
	fmt.Println("Restart Cursor to load the MCP server.")
	printMCPTools()

	return nil
}

func installForCodex(grovePath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Codex config is always at ~/.codex/config.toml
	configDir := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	configPath := filepath.Join(configDir, "config.toml")

	// Read existing config
	existingContent := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existingContent = string(data)
	}

	// Check if grove is already configured
	if strings.Contains(existingContent, "[mcp_servers.grove]") {
		fmt.Printf("grove MCP server is already configured in %s\n", configPath)
		fmt.Println("\nTo reinstall, remove the '[mcp_servers.grove]' section and run this command again.")
		return nil
	}

	// Build the grove MCP server TOML config
	groveConfig := fmt.Sprintf(`
[mcp_servers.grove]
command = %q
args = ["mcp"]
`, grovePath)

	// Append to existing config
	newContent := existingContent + groveConfig

	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	fmt.Printf("✓ Installed grove MCP server in Codex\n\n")
	fmt.Printf("  Config file: %s\n", configPath)
	fmt.Printf("  Binary path: %s\n\n", grovePath)
	fmt.Println("Restart Codex to load the MCP server.")
	printMCPTools()

	return nil
}

func printMCPTools() {
	fmt.Println("\nAvailable tools:")
	fmt.Println("  - grove_list:    List all registered dev servers")
	fmt.Println("  - grove_start:   Start a dev server for a git worktree")
	fmt.Println("  - grove_stop:    Stop a running dev server")
	fmt.Println("  - grove_url:     Get the URL for a worktree's dev server")
	fmt.Println("  - grove_status:  Get detailed status of a dev server")
	fmt.Println("  - grove_restart: Restart a running dev server")
	fmt.Println("  - grove_new:     Create a new git worktree")
	fmt.Println("\nNote: For task management, use a dedicated task MCP server:")
	fmt.Println("  - Tasuku: tk_list, tk_start, tk_done, tk_learn, etc.")
	fmt.Println("  - Beads:  bd_list, bd_start, bd_done, etc.")
}

// JSON-RPC types
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP types
type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      serverInfo   `json:"serverInfo"`
	Capabilities    capabilities `json:"capabilities"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type capabilities struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

type toolsCapability struct{}

type tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolsListResult struct {
	Tools []tool `json:"tools"`
}

type callToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type callToolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCP Server
type mcpServer struct{}

func runMCPServer() {
	server := &mcpServer{}
	server.run()
}

func (s *mcpServer) run() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(&req)
	}
}

func (s *mcpServer) handleRequest(req *jsonRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// No response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *mcpServer) handleInitialize(req *jsonRPCRequest) {
	result := initializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: serverInfo{
			Name:    "grove",
			Version: Version,
		},
		Capabilities: capabilities{
			Tools: &toolsCapability{},
		},
	}
	s.sendResult(req.ID, result)
}

func (s *mcpServer) handleToolsList(req *jsonRPCRequest) {
	tools := []tool{
		{
			Name:        "grove_list",
			Description: "List all registered dev servers and their URLs. Shows server names, URLs, ports, and running status. Use to see what development servers are running or available.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{},
			},
		},
		{
			Name:        "grove_start",
			Description: "Start a dev server for a git worktree. Run development server commands like bin/dev, rails s, npm run dev. Server accessible via URL based on port or subdomain mode.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"command": {
						Type:        "string",
						Description: "The dev server command to run (e.g., 'bin/dev', 'rails s', 'npm run dev', 'yarn dev')",
					},
					"path": {
						Type:        "string",
						Description: "Path to the project directory or git worktree (defaults to current directory)",
					},
				},
				Required: []string{"command"},
			},
		},
		{
			Name:        "grove_stop",
			Description: "Stop a running dev server by name. Kills the server process and marks it as stopped.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name": {
						Type:        "string",
						Description: "Name of the dev server to stop (use grove_list to see available servers)",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "grove_url",
			Description: "Get the URL for a worktree's dev server. Returns the HTTP localhost address for accessing the server. Useful for browser automation and testing.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name": {
						Type:        "string",
						Description: "Name of the dev server (optional, defaults to current worktree)",
					},
				},
			},
		},
		{
			Name:        "grove_status",
			Description: "Check detailed status of a dev server including running state, health, uptime, port, process ID, and logs file path.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name": {
						Type:        "string",
						Description: "Name of the dev server to check (optional, defaults to current worktree)",
					},
				},
			},
		},
		{
			Name:        "grove_restart",
			Description: "Restart a running dev server by name. Stops and starts the server process with the same command.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name": {
						Type:        "string",
						Description: "Name of the dev server to restart (use grove_list to see available servers)",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "grove_new",
			Description: "Create a new git worktree and checkout a branch. Creates an isolated working directory for feature branches. Registers with grove for server management.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"branch": {
						Type:        "string",
						Description: "Name of the new branch to create (e.g., 'feature-auth', 'fix-login-bug')",
					},
					"base": {
						Type:        "string",
						Description: "Base branch to create from (optional, defaults to main or master)",
					},
					"path": {
						Type:        "string",
						Description: "Path to the git repository (optional, defaults to current directory)",
					},
				},
				Required: []string{"branch"},
			},
		},
	}

	s.sendResult(req.ID, toolsListResult{Tools: tools})
}

func (s *mcpServer) handleToolsCall(req *jsonRPCRequest) {
	var params callToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	var result callToolResult

	switch params.Name {
	case "grove_list":
		result = s.toolList()
	case "grove_start":
		result = s.toolStart(params.Arguments)
	case "grove_stop":
		result = s.toolStop(params.Arguments)
	case "grove_url":
		result = s.toolURL(params.Arguments)
	case "grove_status":
		result = s.toolStatus(params.Arguments)
	case "grove_restart":
		result = s.toolRestart(params.Arguments)
	case "grove_new":
		result = s.toolNew(params.Arguments)
	default:
		result = callToolResult{
			Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Unknown tool: %s", params.Name)}},
			IsError: true,
		}
	}

	s.sendResult(req.ID, result)
}

// Tool implementations

func (s *mcpServer) toolList() callToolResult {
	reg, err := registry.Load()
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to load registry: %v", err))
	}

	// Cleanup is best-effort for listing - ignore errors as we can still list servers
	_, _ = reg.Cleanup()
	servers := reg.List()

	var sb strings.Builder

	// Add task nudge if there's an active task
	cwd, _ := os.Getwd()
	if taskID, taskDesc := discovery.GetActiveTask(cwd); taskID != "" {
		sb.WriteString(fmt.Sprintf("📋 **Current Task:** %s\n", taskID))
		if taskDesc != "" {
			desc := ansi.Truncate(taskDesc, styles.TruncateDefault, styles.TruncateTail)
			sb.WriteString(fmt.Sprintf("   %s\n", desc))
		}
		sb.WriteString("\n")
	}

	if len(servers) == 0 {
		sb.WriteString("No servers registered. Use grove_start to start a server.")
		return mcpTextResult(sb.String())
	}

	sb.WriteString("Registered servers:\n\n")

	for _, server := range servers {
		status := "stopped"
		if server.IsRunning() {
			status = "running"
		}

		// Use URL based on configured mode
		url := cfg.ServerURL(server.Name, server.Port)

		sb.WriteString(fmt.Sprintf("- **%s** (%s)\n", server.Name, status))
		sb.WriteString(fmt.Sprintf("  URL: %s\n", url))
		if cfg.IsSubdomainMode() {
			sb.WriteString(fmt.Sprintf("  Subdomains: %s\n", cfg.SubdomainURL(server.Name)))
		}
		sb.WriteString(fmt.Sprintf("  Port: %d\n", server.Port))
		if server.IsRunning() {
			sb.WriteString(fmt.Sprintf("  PID: %d\n", server.PID))
		}
		sb.WriteString("\n")
	}

	return mcpTextResult(sb.String())
}

func (s *mcpServer) toolStart(args map[string]interface{}) callToolResult {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return mcpErrorResult("command is required")
	}

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	// Make path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Invalid path: %v", err))
	}

	// Detect worktree
	wt, err := worktree.DetectAt(absPath)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to detect worktree: %v", err))
	}

	// Load registry
	reg, err := registry.Load()
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to load registry: %v", err))
	}

	// Check if already running
	if existing, ok := reg.Get(wt.Name); ok && existing.IsRunning() {
		return mcpTextResult(fmt.Sprintf("Server '%s' is already running at %s (port %d)", wt.Name, existing.URL, existing.Port))
	}

	// Allocate port
	allocator := port.NewAllocator(cfg.PortMin, cfg.PortMax)
	serverPort, err := allocator.AllocateWithFallback(wt.Name, reg.GetUsedPorts())
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to allocate port: %v", err))
	}

	// Build URL based on configured mode
	url := cfg.ServerURL(wt.Name, serverPort)

	// Create log file
	logDir := cfg.LogDir
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to create log directory: %v", err))
	}
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", wt.Name))

	// Open log file
	logFH, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to open log file: %v", err))
	}

	// Start the process via shell with stdin kept open
	cmdParts := strings.Fields(command)
	shellCmd := fmt.Sprintf("tail -f /dev/null | PORT=%d exec %s", serverPort, mcpShellQuoteArgs(cmdParts))
	cmd := exec.Command("/bin/sh", "-c", shellCmd)
	cmd.Dir = absPath
	cmd.Stdout = logFH
	cmd.Stderr = logFH
	cmd.Env = os.Environ()

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		logFH.Close()
		return mcpErrorResult(fmt.Sprintf("Failed to start server: %v", err))
	}

	pid := cmd.Process.Pid

	go func() {
		// Wait for process to exit, close log file regardless of outcome
		cmd.Wait() //nolint:errcheck // Process cleanup, error doesn't affect outcome
		logFH.Close()
	}()

	time.Sleep(100 * time.Millisecond)

	if !mcpIsProcessRunning(pid) {
		return mcpErrorResult(fmt.Sprintf("Server process exited immediately. Check logs at: %s", logFile))
	}

	// Save to registry
	server := &registry.Server{
		Name:      wt.Name,
		Port:      serverPort,
		PID:       pid,
		Command:   cmdParts,
		Path:      absPath,
		URL:       url,
		Status:    registry.StatusRunning,
		StartedAt: time.Now(),
		Branch:    wt.Branch,
		LogFile:   logFile,
	}

	if err := reg.Set(server); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to save to registry: %v", err))
	}

	var result string
	if cfg.IsSubdomainMode() {
		result = fmt.Sprintf("Server started successfully!\n\n- Name: %s\n- URL: %s\n- Subdomains: %s\n- Port: %d\n- PID: %d\n- Logs: %s",
			wt.Name, url, cfg.SubdomainURL(wt.Name), serverPort, pid, logFile)
	} else {
		result = fmt.Sprintf("Server started successfully!\n\n- Name: %s\n- URL: %s\n- Port: %d\n- PID: %d\n- Logs: %s",
			wt.Name, url, serverPort, pid, logFile)
	}
	return mcpTextResult(result)
}

func (s *mcpServer) toolStop(args map[string]interface{}) callToolResult {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcpErrorResult("name is required")
	}

	reg, err := registry.Load()
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to load registry: %v", err))
	}

	server, ok := reg.Get(name)
	if !ok {
		return mcpErrorResult(fmt.Sprintf("No server registered for '%s'", name))
	}

	if !server.IsRunning() {
		return mcpTextResult(fmt.Sprintf("Server '%s' is not running", name))
	}

	process, err := os.FindProcess(server.PID)
	if err == nil {
		// Best effort kill - process may already be dead
		process.Kill() //nolint:errcheck // Best effort during shutdown
	}

	server.Status = registry.StatusStopped
	server.PID = 0
	server.StoppedAt = time.Now()
	if err := reg.Set(server); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to update registry: %v", err))
	}

	return mcpTextResult(fmt.Sprintf("Server '%s' stopped", name))
}

func (s *mcpServer) toolURL(args map[string]interface{}) callToolResult {
	var name string

	if n, ok := args["name"].(string); ok && n != "" {
		name = n
	} else {
		wt, err := worktree.Detect()
		if err != nil {
			return mcpErrorResult(fmt.Sprintf("Failed to detect worktree: %v. Please provide a name.", err))
		}
		name = wt.Name
	}

	reg, err := registry.Load()
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to load registry: %v", err))
	}

	server, ok := reg.Get(name)
	if !ok {
		// Server not registered - show what URL would be
		if cfg.IsSubdomainMode() {
			return mcpTextResult(fmt.Sprintf("Server '%s' is not registered, but would be available at:\n\n- URL: %s\n- Subdomains: %s\n\nUse grove_start to start the server.", name, cfg.ServerURL(name, 0), cfg.SubdomainURL(name)))
		}
		return mcpTextResult(fmt.Sprintf("Server '%s' is not registered.\n\nUse grove_start to start the server. It will be available at http://localhost:PORT", name))
	}

	status := "stopped"
	if server.IsRunning() {
		status = "running"
	}

	// Use URL based on configured mode
	url := cfg.ServerURL(server.Name, server.Port)

	if cfg.IsSubdomainMode() {
		return mcpTextResult(fmt.Sprintf("Server: %s (%s)\n\n- URL: %s\n- Subdomains: %s\n- Port: %d",
			server.Name, status, url, cfg.SubdomainURL(server.Name), server.Port))
	}
	return mcpTextResult(fmt.Sprintf("Server: %s (%s)\n\n- URL: %s\n- Port: %d",
		server.Name, status, url, server.Port))
}

func (s *mcpServer) toolStatus(args map[string]interface{}) callToolResult {
	var name string

	if n, ok := args["name"].(string); ok && n != "" {
		name = n
	} else {
		wt, err := worktree.Detect()
		if err != nil {
			return mcpErrorResult(fmt.Sprintf("Failed to detect worktree: %v. Please provide a name.", err))
		}
		name = wt.Name
	}

	reg, err := registry.Load()
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to load registry: %v", err))
	}

	server, ok := reg.Get(name)
	if !ok {
		return mcpTextResult(fmt.Sprintf("Server '%s' is not registered. Use grove_start to start a server.", name))
	}

	// Use URL based on configured mode
	url := cfg.ServerURL(server.Name, server.Port)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Server: %s\n\n", server.Name))
	sb.WriteString(fmt.Sprintf("- Status: %s\n", server.Status))
	sb.WriteString(fmt.Sprintf("- URL: %s\n", url))
	sb.WriteString(fmt.Sprintf("- Port: %d\n", server.Port))
	sb.WriteString(fmt.Sprintf("- Path: %s\n", server.Path))

	if server.Branch != "" {
		sb.WriteString(fmt.Sprintf("- Branch: %s\n", server.Branch))
	}

	if server.IsRunning() {
		sb.WriteString(fmt.Sprintf("- PID: %d\n", server.PID))
		sb.WriteString(fmt.Sprintf("- Uptime: %s\n", server.UptimeString()))

		if port.IsListening(server.Port) {
			sb.WriteString("- Port Status: listening\n")
		} else {
			sb.WriteString("- Port Status: not listening (server may still be starting)\n")
		}
	}

	if server.LogFile != "" {
		sb.WriteString(fmt.Sprintf("- Log File: %s\n", server.LogFile))
	}

	return mcpTextResult(sb.String())
}

func (s *mcpServer) toolRestart(args map[string]interface{}) callToolResult {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcpErrorResult("name is required")
	}

	reg, err := registry.Load()
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to load registry: %v", err))
	}

	server, ok := reg.Get(name)
	if !ok {
		return mcpErrorResult(fmt.Sprintf("No server registered for '%s'", name))
	}

	// Stop if running
	if server.IsRunning() {
		process, err := os.FindProcess(server.PID)
		if err == nil {
			process.Kill() //nolint:errcheck // Best effort during restart
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Restart using the same command
	if len(server.Command) == 0 {
		return mcpErrorResult(fmt.Sprintf("Server '%s' has no command recorded", name))
	}

	// Re-use the start logic
	startArgs := map[string]interface{}{
		"command": strings.Join(server.Command, " "),
		"path":    server.Path,
	}

	return s.toolStart(startArgs)
}

func (s *mcpServer) toolNew(args map[string]interface{}) callToolResult {
	branch, ok := args["branch"].(string)
	if !ok || branch == "" {
		return mcpErrorResult("branch is required")
	}

	baseBranch := ""
	if b, ok := args["base"].(string); ok {
		baseBranch = b
	}

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	// Make path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Invalid path: %v", err))
	}

	// Detect worktree/repo info
	wt, err := worktree.DetectAt(absPath)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to detect git repository: %v", err))
	}

	// Get the main repo path
	mainRepoPath := absPath
	if wt.IsWorktree && wt.MainWorktreePath != "" {
		mainRepoPath = wt.MainWorktreePath
	}

	// Determine base branch
	if baseBranch == "" {
		// Try to detect default branch
		cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
		cmd.Dir = mainRepoPath
		output, err := cmd.Output()
		if err == nil {
			ref := strings.TrimSpace(string(output))
			baseBranch = strings.TrimPrefix(ref, "refs/remotes/origin/")
		} else {
			// Fallback to main or master
			for _, candidate := range []string{"main", "master"} {
				checkCmd := exec.Command("git", "rev-parse", "--verify", candidate)
				checkCmd.Dir = mainRepoPath
				if err := checkCmd.Run(); err == nil {
					baseBranch = candidate
					break
				}
			}
		}
		if baseBranch == "" {
			return mcpErrorResult("Could not determine base branch. Please specify with 'base' parameter.")
		}
	}

	// Get repo name for worktree naming
	repoName := filepath.Base(mainRepoPath)
	if repoName == ".bare" {
		repoName = filepath.Base(filepath.Dir(mainRepoPath))
	}

	// Sanitize branch name for directory
	sanitizedBranch := worktree.Sanitize(branch)
	worktreeName := fmt.Sprintf("%s-%s", repoName, sanitizedBranch)

	// Determine worktree path
	var worktreePath string
	if cfg.WorktreesDir != "" {
		expandedDir := expandPath(cfg.WorktreesDir)
		worktreePath = filepath.Join(expandedDir, repoName, sanitizedBranch)
	} else {
		// Create as sibling to main repo
		parentDir := filepath.Dir(mainRepoPath)
		worktreePath = filepath.Join(parentDir, worktreeName)
	}

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to create parent directory: %v", err))
	}

	// Create the worktree
	cmd := exec.Command("git", "worktree", "add", "-b", branch, worktreePath, baseBranch)
	cmd.Dir = mainRepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to create worktree: %v\nOutput: %s", err, string(output)))
	}

	var sb strings.Builder
	sb.WriteString("Worktree created successfully!\n\n")
	sb.WriteString(fmt.Sprintf("- Name: %s\n", worktreeName))
	sb.WriteString(fmt.Sprintf("- Branch: %s\n", branch))
	sb.WriteString(fmt.Sprintf("- Path: %s\n", worktreePath))
	sb.WriteString(fmt.Sprintf("- Based on: %s\n", baseBranch))
	sb.WriteString("\nTo start working:\n")
	sb.WriteString(fmt.Sprintf("  cd %s\n", worktreePath))
	sb.WriteString("  grove start <command>\n")

	return mcpTextResult(sb.String())
}

// Helpers

func mcpTextResult(text string) callToolResult {
	return callToolResult{
		Content: []toolContent{{Type: "text", Text: text}},
	}
}

func mcpErrorResult(text string) callToolResult {
	return callToolResult{
		Content: []toolContent{{Type: "text", Text: text}},
		IsError: true,
	}
}

func mcpShellQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		escaped := strings.ReplaceAll(arg, "'", "'\\''")
		quoted[i] = "'" + escaped + "'"
	}
	return strings.Join(quoted, " ")
}

func mcpIsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (s *mcpServer) sendResult(id interface{}, result interface{}) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *mcpServer) sendError(id interface{}, code int, message string, data interface{}) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.send(resp)
}

func (s *mcpServer) send(resp jsonRPCResponse) {
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}
