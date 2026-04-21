# Guild CLI

Terminal interface to SmartCraft's [Guild](https://github.com/smartcraftsoftware/guild) platform. Manage issues, log time, report token costs, and pull context — all from the terminal.

## Install

### Homebrew (macOS)

```bash
brew install smartcraftsoftware/tap/guild-cli
```

### From source

```bash
go install github.com/smartcraftsoftware/guild-cli@latest
```

### Binary releases

Download from [GitHub Releases](https://github.com/smartcraftsoftware/guild-cli/releases).

## Quick Start

```bash
# Authenticate with your Guild instance
guild auth login --server http://your-guild-server.com

# Set your default team
guild config set team <team-id>

# List your team's issues
guild issues list

# Create an issue
guild issues create --title "Fix login bug" --type bug --priority high

# Log time
guild time log --project 1 --duration 2h --description "Feature work"

# Report PR token cost
guild pr cost --repo org/repo --pr 42 --tokens 50000 --cost 0.75

# Get context for GSD
guild context --json
```

## Claude Code Integration

Guild can automatically track AI token costs per commit and per PR across all Claude Code sessions — including vanilla Claude, GSD, Superpowers subagents, and any other wrapper.

### How it works

Three Claude Code hooks are installed into your settings:

- **SessionStart** — checks login status at the start of each session and prompts you to authenticate if needed
- **PostToolUse** — detects every `git commit` Claude Code performs and records the commit SHA and repo
- **Stop** — when the session ends, reports the session's total token cost to Guild, split across the commits made in that session

Costs are stored against the commit SHA immediately. If the commit hasn't reached GitHub yet, Guild holds the record and links it to the PR automatically once the push arrives via webhook.

### Setup (individual developer)

```bash
# 1. Log in
guild auth login --server http://your-guild-server.com

# 2. Install the Claude Code hooks (writes to ~/.claude/settings.json)
guild setup claude

# That's it — all future Claude Code sessions report automatically
```

Use `--local` to write hooks to `.claude/settings.json` in the current project instead of globally:

```bash
guild setup claude --local
```

### Setup (Claude Enterprise — org-wide, zero per-user config)

Add the following to your organisation's **Managed Settings** in the Claude.ai admin console (`Admin Settings > Claude Code > Managed settings`). This pushes the hooks to all developers automatically when they authenticate.

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [{ "type": "command", "command": "guild hook session-start" }]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{ "type": "command", "command": "guild hook post-tool-use" }]
      }
    ],
    "Stop": [
      {
        "hooks": [{ "type": "command", "command": "guild hook stop" }]
      }
    ]
  }
}
```

Developers still need the `guild` binary on their PATH and must run `guild auth login` once. The `SessionStart` hook will prompt them to do so at the start of each Claude Code session until they log in.

To prevent users from overriding org hooks, add `"allowManagedHooksOnly": true` to your managed settings.

### Manual commit cost reporting

```bash
# Report cost for HEAD commit (auto-detects repo from git remote)
guild commit cost --tokens 12000 --cost 0.18

# Report for a specific commit
guild commit cost --sha abc1234 --repo org/repo --tokens 12000 --cost 0.18

# Include a session ID for deduplication
guild commit cost --sha abc1234 --repo org/repo --tokens 12000 --cost 0.18 --session-id <id>
```

## Commands

| Command | Description |
|---------|-------------|
| `guild auth login` | Authenticate via browser |
| `guild auth status` | Show current auth status |
| `guild auth logout` | Remove stored token |
| `guild config set <key> <value>` | Set config (server, team) |
| `guild config get <key>` | Get config value |
| `guild issues list` | List team issues |
| `guild issues create` | Create an issue |
| `guild issues view <id>` | View issue details |
| `guild time log` | Log a time entry |
| `guild time list` | List recent time entries |
| `guild time delete <id>` | Delete a time entry |
| `guild pr cost` | Report token cost for a PR |
| `guild commit cost` | Report token cost for a commit |
| `guild setup claude` | Install Claude Code hooks into settings.json |
| `guild hook session-start` | Hook: prompt login if not authenticated |
| `guild hook post-tool-use` | Hook: capture git commits during a session |
| `guild hook stop` | Hook: report session cost to Guild on exit |
| `guild context` | Get assigned issues for GSD |
| `guild version` | Print version info |
| `guild completion bash\|zsh\|fish` | Generate shell completions |

## Configuration

Config is stored at `~/.guild/config.yaml`:

```yaml
server_url: http://localhost:3000
token: guild_...
team_id: "1"
```

Override at runtime with flags:
- `--server <url>` — override server URL
- `--config <path>` — use alternate config file

Session data captured by hooks is stored temporarily in `~/.guild/sessions/` and cleaned up after each session ends.

## Shell Completions

```bash
# Bash
guild completion bash > /etc/bash_completion.d/guild

# Zsh
guild completion zsh > "${fpath[1]}/_guild"

# Fish
guild completion fish > ~/.config/fish/completions/guild.fish
```

## Development

```bash
go build -o guild .
go test ./...
```

## License

MIT
