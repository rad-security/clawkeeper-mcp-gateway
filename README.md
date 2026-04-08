# Clawkeeper MCP Gateway

Open-source MCP gateway with threat detection, warn mode, and fail-open design.

Sits between any MCP client (Cursor, Claude Code, Windsurf, Copilot) and your MCP servers. Inspects every tool call for threats, sensitive data, and policy violations. Deploys in 60 seconds.

## Quick Start

```bash
# Install
brew install rad-security/tap/clawkeeper-mcp-gateway
# or
curl -fsSL https://clawkeeper.dev/install-gateway.sh | bash

# Add your MCP servers
clawkeeper-mcp-gateway add github "npx -y @modelcontextprotocol/server-github"
clawkeeper-mcp-gateway add filesystem "npx -y @modelcontextprotocol/server-filesystem /path/to/allowed"

# Point your MCP client at the gateway
# In Cursor (~/.cursor/mcp.json) or Claude Code:
{
  "mcpServers": {
    "clawkeeper-mcp-gateway": {
      "command": "clawkeeper-mcp-gateway",
      "args": ["server"]
    }
  }
}
```

That's it. The gateway proxies all MCP traffic, detects threats in real-time, and logs everything locally.

## What It Detects

**36 threat detection patterns** running locally at sub-50ms:

| Category | Examples |
|---|---|
| Credential exfiltration | API keys piped to curl, SSH keys sent to external endpoints |
| Reverse shells | bash, netcat, python, perl, ruby, base64-encoded |
| Prompt injection | Override instructions, persona hijacking, jailbreak attempts |
| Security control bypass | Firewall disable, SELinux/AppArmor teardown, AV kill |
| Supply chain attacks | Suspicious package installs from raw URLs |
| Tool poisoning | Hidden instructions in MCP tool descriptions |
| Sensitive data | Stripe/AWS/GitHub keys, credit cards, SSNs, private keys, JWTs |

## Two Modes

**Audit (default):** Full proxy, full visibility, zero blocking. See every tool call, every threat, every server. Zero developer friction.

```bash
clawkeeper-mcp-gateway server
```

**Enforce:** Same proxy. Policies enforced — threats blocked or warned per configuration.

```bash
clawkeeper-mcp-gateway server --enforce
```

## Warn Mode

When a threat is detected in warn mode, the warning is returned to the AI client as context. The AI sees the threat and can self-correct — no developer interruption, no retry loops.

This is unique to Clawkeeper. Other gateways either block silently or pass through without feedback.

## Fail-Open Design

The gateway never breaks your tools:

- Detection error: tool call proceeds, event logged
- API timeout: falls back to local detection
- Gateway crash: watchdog spawns pass-through proxy instantly
- Network down: uses cached policy, queues events

## Connect to Dashboard

Optional. Get fleet-wide visibility, team policies, and identity-aware access controls.

```bash
clawkeeper-mcp-gateway auth login
```

Opens your browser for device authorization. Once connected, events stream to the dashboard and team policies sync every 60 seconds.

## CLI Reference

```bash
# Server management
clawkeeper-mcp-gateway add <name> <command>
clawkeeper-mcp-gateway remove <name>
clawkeeper-mcp-gateway list [--health] [--json]

# Gateway
clawkeeper-mcp-gateway server [--enforce]
clawkeeper-mcp-gateway logs [-f] [-l 50]
clawkeeper-mcp-gateway scan
clawkeeper-mcp-gateway export --format json|csv --since 2026-04-01

# Configuration
clawkeeper-mcp-gateway config show
clawkeeper-mcp-gateway auth login|status|logout
clawkeeper-mcp-gateway completion zsh|bash|fish
```

## Architecture

```
MCP Client (Cursor, Claude Code, Windsurf)
    |
    v
clawkeeper-mcp-gateway (local binary, 8MB)
    |
    +---> Detection Engine (36 patterns, <50ms)
    +---> Policy Engine (dashboard + local config)
    +---> Event Logger (JSONL + batch upload)
    +---> Watchdog (fail-open recovery)
    |
    v
MCP Servers (GitHub, filesystem, Slack, etc.)
```

## Works With the Claude Code Plugin

For complete coverage, deploy both:

- **MCP Gateway** — covers MCP tool calls across all IDEs
- **Claude Code Plugin** — covers native tools (Bash, Read, Write, Edit) that MCP can't see

Both report to the same dashboard. Single pane of glass.

## Compliance

| Framework | Controls |
|---|---|
| OWASP Agentic Top 10 | ASI01-ASI05 |
| OWASP LLM Top 10 | LLM01, LLM02, LLM03, LLM06 |
| SOC 2 | CC6.1, CC6.6, CC7.2, CC9.2 |
| EU AI Act | Art. 9, 12, 14, 15 |

## License

MIT

## Links

- [Dashboard](https://clawkeeper.dev)
- [Docs](https://clawkeeper.dev/docs)
- [Claude Code Plugin](https://github.com/rad-security/claude-code-plugin)
