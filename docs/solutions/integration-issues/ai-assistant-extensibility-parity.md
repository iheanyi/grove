---
title: MCP/Skills/Plugins Feature Parity Across AI Coding Tools
category: integration-issues
tags:
  - mcp
  - model-context-protocol
  - claude-code
  - cursor
  - copilot-cli
  - codex
  - opencode
  - gemini
  - extensibility
  - plugins
  - skills
  - interoperability
severity: medium
symptoms:
  - Different configuration file formats and locations across tools
  - Inconsistent command/args schema (string vs array vs object)
  - Varying local vs global config support
  - Tool naming and namespacing conflicts
  - Protocol version compatibility mismatches
  - Duplicated installation logic for each provider
  - No unified capability discovery mechanism
components:
  - mcp-server-implementation
  - config-file-parsers
  - tool-registration
  - cli-install-commands
  - provider-adapters
created: 2026-01-20
tools_analyzed:
  - name: Claude Code
    config_path: ~/.claude/settings.json
    install_method: claude mcp add CLI
    scope: user-only
    format: json
  - name: Gemini CLI
    config_path: ~/.gemini/settings.json or .gemini/settings.json
    install_method: direct JSON edit
    scope: global and local
    format: json
    schema_key: mcpServers
  - name: OpenCode
    config_path: ~/.config/opencode/opencode.json or opencode.json
    install_method: direct JSON edit
    scope: global and local
    format: json
    schema_key: mcp
    command_format: array
  - name: Cursor
    config_path: ~/.cursor/mcp.json or .cursor/mcp.json
    install_method: direct JSON edit
    scope: global and local
    format: json
    schema_key: mcpServers
  - name: Codex
    config_path: ~/.codex/config.toml
    install_method: TOML append
    scope: global-only
    format: toml
related_files:
  - cli/internal/cli/mcp.go
---

# MCP/Skills/Plugins Feature Parity Across AI Coding Tools

## Problem Statement

Ensuring consistent extensibility features (MCP servers, skills, plugins, hooks) across different AI coding assistants:
- **Claude Code** - Anthropic's CLI with MCP, skills, plugins, hooks
- **Cursor** - AI-powered IDE with rules and MCP support
- **Copilot CLI** - GitHub's CLI AI assistant
- **Codex** - OpenAI's coding assistant
- **OpenCode** - Open source AI coding tool
- **Gemini CLI** - Google's AI coding assistant

## Root Cause Analysis

### 1. Configuration Schema Divergence

Each tool developed its own configuration format independently:

| Tool | Config Path | Format | Schema Key |
|------|-------------|--------|------------|
| Claude Code | `~/.claude/settings.json` | JSON | `mcpServers` |
| Copilot CLI | `~/.copilot/mcp-config.json` | JSON | `mcpServers` |
| Cursor | `.cursor/mcp.json` | JSON | `mcpServers` |
| Gemini | `.gemini/settings.json` | JSON | `mcpServers` |
| OpenCode | `opencode.json` | JSON | `mcp` (different structure) |
| Codex | `~/.codex/config.toml` | TOML | `[mcp_servers]` |

### 2. Different Instruction File Systems

| Tool | Primary Instructions | Rules Location |
|------|---------------------|----------------|
| Claude Code | `CLAUDE.md` | `.claude/rules/` |
| Cursor | `.cursorrules` | `.cursor/rules/` |
| Copilot CLI | `.github/copilot-instructions.md` | `.github/instructions/` |
| Codex | `AGENTS.md` | N/A |
| OpenCode | `AGENTS.md` | `.opencode/agents/` |

### 3. Extensibility Architecture Differences

| Tool | Plugins | Skills | Hooks | Agents |
|------|---------|--------|-------|--------|
| Claude Code | Full system | 700+ skills | Event hooks | Subagents |
| Cursor | Limited | Via rules | Fast (10-20x) | Limited |
| Copilot CLI | Extensions | `.github/skills/` | Limited | Built-in |
| Codex | Limited | Limited | Limited | Agents SDK |
| OpenCode | JS/TS plugins | Limited | 6 mechanisms | Limited |

## Solution: Cross-Tool Compatibility Strategy

### Approach 1: AGENTS.md as Universal Instructions

[AGENTS.md](https://agents.md/) is the emerging vendor-neutral standard, adopted by 60,000+ open source projects.

**Project structure for maximum compatibility:**

```
project/
├── AGENTS.md                    # Universal (Codex, Cursor, Copilot, OpenCode)
├── CLAUDE.md                    # Claude Code (can @import AGENTS.md)
├── .github/
│   └── copilot-instructions.md  # Copilot fallback
├── .cursor/
│   └── rules/                   # Cursor rules
└── .cursorrules                 # Cursor fallback
```

**CLAUDE.md that imports AGENTS.md:**

```markdown
# Claude Code Instructions

@import ./AGENTS.md

## Claude-Specific
- Use `trash` instead of `rm -rf`
- Prefer editing existing files over creating new ones
```

### Approach 2: Unified MCP Server Configuration

Create a single source of truth for MCP servers:

**Master MCP config (`mcp-servers.json`):**

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-filesystem-server", "/path/to/project"],
      "env": {}
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@anthropic/mcp-github-server"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
```

**Tool-specific adapters:**

```bash
# Claude Code and Cursor: Direct symlink
ln -s ../mcp-servers.json .claude/mcp.json
ln -s ../mcp-servers.json .cursor/mcp.json

# Codex: Requires TOML conversion
# See scripts/sync-mcp-to-toml.sh
```

### Approach 3: Grove's Multi-Provider Installation

Grove demonstrates this pattern in `cli/internal/cli/mcp.go`:

```go
// grove mcp install --all
// Installs to: claude-code, gemini, opencode, cursor, codex

func installForClaudeCode() error {
    // Uses: claude mcp add grove grove mcp
    return runClaudeMCPAdd()
}

func installForGemini(global bool) error {
    // Uses: ~/.gemini/settings.json or .gemini/settings.json
    return updateJSONConfig(geminiConfigPath(global), "mcpServers")
}

func installForOpenCode(global bool) error {
    // Uses: different schema with "type", "command" as array, "enabled"
    return updateOpenCodeConfig(global)
}

func installForCursor(global bool) error {
    // Uses: ~/.cursor/mcp.json or .cursor/mcp.json
    return updateJSONConfig(cursorConfigPath(global), "mcpServers")
}

func installForCodex() error {
    // Uses: ~/.codex/config.toml (TOML format)
    return appendTOMLConfig()
}
```

## Tool-Specific Features Without Parity

### Claude Code Exclusive Features

1. **Plugin System**: Full directory structure with commands, agents, skills, hooks
2. **Skill Marketplace**: 700+ skills with progressive disclosure
3. **Subagent Architecture**: Parallel agent execution
4. **Hooks**: Pre/post event handlers for tool calls

```
.claude-plugin/
├── plugin.json
├── commands/          # Slash commands
├── agents/           # Specialized agents
├── skills/           # SKILL.md files
└── hooks/            # Event handlers
```

### Cursor Exclusive Features

1. **Rules MDX Format**: Metadata headers with glob auto-attachment
2. **40-Tool Limit**: Requires careful MCP server selection
3. **Integrated IDE**: Deep editor integration

### Copilot CLI Exclusive Features

1. **Built-in Specialized Agents**: Explore, Task
2. **GitHub Actions Integration**: Native CI/CD
3. **Org Policy Inheritance**: Enterprise governance

### Codex Exclusive Features

1. **Agents SDK Integration**: Orchestrate multiple agents
2. **MCP Server Mode**: Run Codex as MCP server
3. **TOML Configuration**: Different format

## Prevention Strategies

### 1. Parity Matrix Tracking

```yaml
# .grove/parity-matrix.yaml
capabilities:
  mcp_tools:
    context7:
      claude_code: true
      cursor: true
      copilot: false
      codex: false
      opencode: true
    sentry:
      claude_code: true
      cursor: partial
      copilot: false
      codex: false
      opencode: true
```

### 2. CI/CD Parity Checks

```yaml
# .github/workflows/parity-check.yml
name: AI Tool Parity Check

on:
  push:
    paths:
      - 'CLAUDE.md'
      - 'AGENTS.md'
      - '.claude/**'
      - '.cursor/**'
      - '.github/copilot-instructions.md'

jobs:
  check-parity:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Check configuration files exist
        run: ./scripts/check-parity.sh
      - name: Verify MCP server sync
        run: ./scripts/sync-mcp-check.sh
```

### 3. Shared Rules Generation

```bash
#!/bin/bash
# scripts/sync-rules.sh

SHARED_RULES=".grove/shared-rules.md"

# Generate CLAUDE.md
cat "$SHARED_RULES" > CLAUDE.md
echo -e "\n## Claude-Specific\n- Use \`trash\` instead of \`rm -rf\`" >> CLAUDE.md

# Generate .cursor/rules
mkdir -p .cursor/rules
cp "$SHARED_RULES" .cursor/rules/project.md

# Generate AGENTS.md
cp "$SHARED_RULES" AGENTS.md

# Generate Copilot instructions
mkdir -p .github
cp "$SHARED_RULES" .github/copilot-instructions.md
```

## Test Cases

```go
func TestConfigurationFilesExist(t *testing.T) {
    requiredFiles := map[string]string{
        "CLAUDE.md":                       "Claude Code",
        ".cursor/rules":                   "Cursor",
        ".github/copilot-instructions.md": "Copilot",
        "AGENTS.md":                       "Codex/OpenCode",
    }

    for path, tool := range requiredFiles {
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("Missing %s config: %s", tool, path)
        }
    }
}

func TestMCPServerParity(t *testing.T) {
    claudeServers := loadMCPConfig(".claude/settings.json")
    cursorServers := loadMCPConfig(".cursor/mcp.json")

    for server := range claudeServers.Servers {
        if _, ok := cursorServers.Servers[server]; !ok {
            t.Logf("WARNING: %q in Claude but not Cursor", server)
        }
    }
}
```

## Recommended Implementation

### Phase 1: Establish Universal Foundation

```bash
# 1. Create AGENTS.md as source of truth
touch AGENTS.md

# 2. Create CLAUDE.md that references it
echo "@import ./AGENTS.md" > CLAUDE.md

# 3. Create unified MCP config
touch mcp-servers.json

# 4. Symlink to tool-specific locations
mkdir -p .claude .cursor .github
```

### Phase 2: Add Tool-Specific Enhancements

```bash
# Claude Code: Add skills and hooks (exclusive features)
mkdir -p .claude/skills .claude/hooks

# Cursor: Add rules with glob patterns
mkdir -p .cursor/rules
```

### Phase 3: Build Custom MCP Servers

For cross-tool functionality, build MCP servers that work everywhere:

```typescript
// Universal project context MCP server
import { Server } from "@modelcontextprotocol/sdk/server/index.js";

const server = new Server(
  { name: "project-context", version: "1.0.0" },
  { capabilities: { tools: {}, resources: {} } }
);

// Works with: Claude Code, Cursor, Copilot, Codex, OpenCode, Gemini
```

## Tradeoffs

### Using AGENTS.md Everywhere

**Pros:**
- Single source of truth
- 60k+ project adoption
- Linux Foundation governance

**Cons:**
- Claude Code still prefers CLAUDE.md (requires import)
- Less granular than tool-specific systems

### MCP-Only Strategy

**Pros:**
- Universal protocol support
- 10,000+ existing servers

**Cons:**
- Cursor's 40-tool limit
- No skill progressive disclosure
- Hooks not standardized

### Tool-Specific Optimization

**Pros:**
- Access to exclusive features
- Best performance per tool

**Cons:**
- Maintenance burden
- Team fragmentation

## Related Documentation

- Grove MCP implementation: `cli/internal/cli/mcp.go`
- Grove agent detection: `cli/internal/discovery/discovery.go`
- Multi-agent dashboard plans: `.claude/plans/worktree-management-improvements.md`

## Conclusion

**Pragmatic approach for 2026:**

1. Use **AGENTS.md** as universal instructions
2. Maintain **single mcp-servers.json** with symlinks
3. Accept **CLAUDE.md required** for Claude Code (with @import)
4. Build **MCP servers** for cross-tool functionality
5. Use **tool-specific features sparingly** and document when used

MCP convergence provides ~80% compatibility. The remaining 20% (skills, plugins, hooks) requires tool-specific configuration.
