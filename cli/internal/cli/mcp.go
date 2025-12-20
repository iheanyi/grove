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

	"github.com/iheanyi/grove/internal/port"
	"github.com/iheanyi/grove/internal/registry"
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
	Short: "Install grove as an MCP server in Claude Code",
	Long: `Configure Claude Code to use grove as an MCP server.

This command adds the grove MCP server configuration to your Claude Code
settings file (~/.claude/settings.json).

After installation, restart Claude Code to load the MCP server.`,
	RunE: runMCPInstall,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpInstallCmd)
}

func runMCPInstall(cmd *cobra.Command, args []string) error {
	// Find wt binary path
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

	// Use claude mcp add command to properly register the MCP server
	claudeCmd := exec.Command("claude", "mcp", "add", "-s", "user", "-t", "stdio", "grove", grovePath, "mcp")
	output, err := claudeCmd.CombinedOutput()
	if err != nil {
		// Check if it's because it already exists
		if strings.Contains(string(output), "already exists") {
			fmt.Println("grove MCP server is already installed.")
			fmt.Println("\nTo reinstall, first remove it with: claude mcp remove wt")
			return nil
		}
		return fmt.Errorf("failed to install MCP server: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("✓ Installed grove MCP server in Claude Code\n\n")
	fmt.Printf("  Binary path: %s\n\n", grovePath)
	fmt.Println("The MCP server is now available. Run 'claude mcp list' to verify.")
	fmt.Println("\nAvailable tools:")
	fmt.Println("  - grove_list:   List all registered dev servers")
	fmt.Println("  - grove_start:  Start a dev server for a git worktree")
	fmt.Println("  - grove_stop:   Stop a running dev server")
	fmt.Println("  - grove_url:    Get the URL for a worktree's dev server")
	fmt.Println("  - grove_status: Get detailed status of a dev server")

	return nil
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
			Description: "List all registered dev servers and their URLs. Returns server names, URLs, ports, and status.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{},
			},
		},
		{
			Name:        "grove_start",
			Description: "Start a dev server for a git worktree. The server will be accessible at https://<worktree-name>.localhost with wildcard subdomain support.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"command": {
						Type:        "string",
						Description: "The command to run (e.g., 'bin/dev', 'rails s', 'npm run dev')",
					},
					"path": {
						Type:        "string",
						Description: "Path to the project directory (defaults to current directory)",
					},
				},
				Required: []string{"command"},
			},
		},
		{
			Name:        "grove_stop",
			Description: "Stop a running dev server by name.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name": {
						Type:        "string",
						Description: "Name of the server to stop (from grove_list)",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "grove_url",
			Description: "Get the URL for a worktree's dev server. Useful for browser automation.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name": {
						Type:        "string",
						Description: "Name of the server (optional, defaults to current worktree)",
					},
				},
			},
		},
		{
			Name:        "grove_status",
			Description: "Get detailed status of a dev server including health, uptime, and logs path.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name": {
						Type:        "string",
						Description: "Name of the server (optional, defaults to current worktree)",
					},
				},
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

	reg.Cleanup()
	servers := reg.List()

	if len(servers) == 0 {
		return mcpTextResult("No servers registered. Use grove_start to start a server.")
	}

	var sb strings.Builder
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
		cmd.Wait()
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
		process.Kill()
	}

	server.Status = registry.StatusStopped
	server.PID = 0
	server.StoppedAt = time.Now()
	reg.Set(server)

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
			return mcpTextResult(fmt.Sprintf("Server '%s' is not registered, but would be available at:\n\n- URL: https://%s.%s\n- Subdomains: %s\n\nUse grove_start to start the server.", name, name, cfg.TLD, cfg.SubdomainURL(name)))
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
