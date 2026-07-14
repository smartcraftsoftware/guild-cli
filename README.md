# Guild CLI

Terminal interface to SmartCraft's [Guild](https://github.com/smartcraftsoftware/guild) platform. Manage issues, log time, and pull context — all from the terminal.

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

## Claude Code AI Cost Tracking

AI token cost tracking now happens entirely on the Guild server side, via Claude Code's native OpenTelemetry export — there is nothing to install or configure in this CLI. See your Guild instance's admin documentation for the one-time, org-wide Managed Settings configuration.

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
