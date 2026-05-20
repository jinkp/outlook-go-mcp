# outlook-mcp

A local Windows MCP server that exposes Microsoft Outlook Desktop to AI clients (OpenCode, Claude Code, etc.) via COM automation.

## Install

```powershell
irm https://raw.githubusercontent.com/jinkp/outlook-go-mcp/main/install.ps1 | iex
```

Installs to `%LOCALAPPDATA%\outlook-mcp\outlook-mcp.exe` and adds it to your user PATH.

## Prerequisites

- Windows 10/11
- Microsoft Outlook Desktop installed and running
- A configured MAPI profile in Outlook

## Setup

### 1. Generate config

```powershell
outlook-mcp setup config
```

Interactive TUI wizard that creates `config.yaml` with your attachment path and security settings.

### 2. Register in your AI client

```powershell
# OpenCode
outlook-mcp setup opencode

# Claude Code
outlook-mcp setup claude
```

Both launch a TUI wizard that writes the MCP server entry into your config file (global or local scope).

### 3. Or use the TUI menu for everything

```powershell
outlook-mcp tui
```

### Manual registration

**OpenCode** (`opencode.json`):
```json
{
  "mcp": {
    "outlook-mcp": {
      "type": "local",
      "command": "outlook-mcp",
      "args": ["mcp", "--config", "C:\\path\\to\\config.yaml"]
    }
  }
}
```

**Claude Code** (`~/.claude.json` or `.claude/settings.json`):
```json
{
  "mcpServers": {
    "outlook-mcp": {
      "command": "outlook-mcp",
      "args": ["mcp", "--config", "C:\\path\\to\\config.yaml"]
    }
  }
}
```

## Tools (15)

### Mail — Read

| Tool | Description | Required | Optional |
|---|---|---|---|
| `search_emails` | Search emails by query, folder, and time range | `query` | `folder`, `since`, `until`, `max_results` |
| `get_email` | Fetch one email with full body and attachment metadata | `id` | — |
| `list_attachments` | List attachment metadata without downloading | `email_id` | — |
| `list_folders` | List all mail folders (flat, with parent IDs) | — | — |

### Mail — Write (policy-gated)

| Tool | Description | Required | Optional | Security flag |
|---|---|---|---|---|
| `create_draft` | Create a new draft email | `to`, `subject`, `body` | — | `allow_create_draft` |
| `reply_draft` | Create a reply draft preserving thread | `email_id`, `body` | — | `allow_reply_draft` |
| `forward_draft` | Create a forward draft | `email_id`, `to`, `body` | — | `allow_forward_draft` |
| `mark_read` | Toggle read/unread state | `email_id`, `read` | — | `allow_mark_read` |
| `flag_email` | Toggle follow-up flag | `email_id`, `flagged` | — | `allow_flag_email` |
| `move_email` | Move email to a named folder | `email_id`, `folder` | — | `allow_move_email` |
| `delete_email` | Soft-delete (moves to Deleted Items) | `email_id` | — | `allow_delete_email` |
| `download_attachment` | Save attachment to disk (within `attachment_dir`) | `email_id`, `attachment_id`, `dest_dir` | — | `allow_save_attachments` |

### Calendar

| Tool | Description | Required | Optional |
|---|---|---|---|
| `list_events` | List calendar events in a time window | `start`, `end` | `max_results` |
| `get_event` | Fetch one event by ID | `id` | — |
| `create_event` | Create a simple non-meeting event | `title`, `start`, `end` | `location`, `body` |

## Daily Report

Generate a 5-section email intelligence report on demand or on a schedule:

```powershell
outlook-mcp report --config config.yaml
```

### Report sections

1. **Unanswered emails** — received in the last N hours with no reply found in Sent Items
2. **VIP emails** — from senders in your `vip_senders` list (exact or domain match)
3. **Emails with attachments** — received in the time window
4. **Active threads** — subjects with 3+ messages grouped together
5. **Today's calendar** — events for today with related-email hints

### Output modes

```yaml
report:
  output_file: "C:\\Reports\\daily.md"   # write markdown file
  draft_recipient: "you@company.com"      # create Outlook draft to yourself
  since_hours: 24                         # lookback window (1-168)
  vip_senders:
    - "ceo@company.com"
    - "@important-client.com"
  max_per_section: 20
```

At least one output (`output_file` or `draft_recipient`) is required.

### Schedule with Windows Task Scheduler

```
Program:   outlook-mcp.exe
Arguments: report --config C:\path\to\config.yaml
Trigger:   Daily at 08:00
```

Outlook must be running when the report executes.

## Configuration reference

Full example: [`configs/config.example.yaml`](configs/config.example.yaml)

### `outlook`

| Field | Default | Description |
|---|---|---|
| `outlook.profile` | `"default"` | MAPI profile name. Use `"default"` for the primary profile. |

### `security` — all write operations default to `false`

| Field | Enables tool |
|---|---|
| `allow_create_draft` | `create_draft` |
| `allow_reply_draft` | `reply_draft` |
| `allow_forward_draft` | `forward_draft` |
| `allow_mark_read` | `mark_read` |
| `allow_flag_email` | `flag_email` |
| `allow_move_email` | `move_email` |
| `allow_delete_email` | `delete_email` |
| `allow_save_attachments` | `download_attachment` |
| `allow_create_event` | `create_event` |

### `paths`

| Field | Description |
|---|---|
| `paths.attachment_dir` | Absolute path where `download_attachment` saves files. Required. |

### `limits`

| Field | Default | Range | Description |
|---|---|---|---|
| `limits.max_results` | `50` | `1–500` | Cap applied to search/list tools. |

### `report`

| Field | Default | Description |
|---|---|---|
| `report.output_file` | `""` | Absolute path for the markdown report file. |
| `report.draft_recipient` | `""` | Email address to send the report draft to. |
| `report.since_hours` | `24` | Lookback window in hours (1–168). |
| `report.vip_senders` | `[]` | List of VIP addresses. Supports exact (`boss@co.com`) and domain (`@co.com`) match. |
| `report.max_per_section` | `20` | Maximum items per report section (1–500). |

### `logging`

| Field | Default | Description |
|---|---|---|
| `logging.level` | `"info"` | Log level: `debug`, `info`, `warn`, `error`. Logs go to stderr. |

## CLI reference

```
outlook-mcp mcp [--config path]        Start the MCP stdio server
outlook-mcp report [flags]             Generate daily email report
  --config  path                       Config file (default: config.yaml)
  --output  path                       Override output_file
  --draft   email                      Override draft_recipient
  --since   hours                      Override since_hours
outlook-mcp setup opencode             Register in opencode.json (TUI wizard)
outlook-mcp setup claude               Register in Claude Code config (TUI wizard)
outlook-mcp setup config               Generate config.yaml (TUI wizard)
outlook-mcp tui                        Interactive menu
outlook-mcp --version                  Show version
```

## Architecture

Hexagonal structure with strict layer isolation:

```
cmd/outlook-mcp/     Cobra CLI entrypoint — subcommand wiring, bootstrap
internal/mcp/        MCP tool schemas, handlers, stdio server
internal/outlook/    Outlook COM adapters (Windows-only build tags)
internal/report/     Daily report engine — pure Go, no COM imports
internal/domain/     Shared DTOs and store interfaces
internal/security/   Policy gate — blocks write tools when not enabled
internal/config/     YAML config loading and validation
internal/logging/    Structured stderr logger
internal/opencode/   opencode.json read/write for self-registration
internal/claude/     Claude Code config read/write for self-registration
internal/tui/        Bubbletea TUI wizards and menu
internal/version/    Version string (injected at build via ldflags)
```

## Build from source

```powershell
# Current platform
make build

# Release binaries (windows/amd64 + windows/arm64)
make build-all

# Tests
make test
```

## Security notes

- All write-capable tools are denied by default. Enable them explicitly in config.
- `download_attachment` restricts paths to `attachment_dir` — path traversal is blocked.
- `delete_email` is a soft delete (moves to Deleted Items). No permanent deletion.
- COM automation runs on a dedicated OS-locked thread via `COMExecutor`.
- All logs go to stderr — stdout is reserved for the MCP stdio transport.
