# Outlook MCP Server (Go)

A local MCP server that exposes Microsoft Outlook Desktop capabilities to AI tools.

## Prerequisites
- Windows 10/11
- Microsoft Outlook Desktop installed and configured with a MAPI profile
- Go 1.21+

## Installation
```bash
go build -o outlook-mcp.exe ./cmd/outlook-mcp
```

## Configuration
Copy `configs/config.example.yaml` to `config.yaml` and edit it for your machine.

| Field | Type | Default | Description |
|---|---|---:|---|
| `outlook.profile` | `string` | `"default"` | Outlook profile name used when connecting to Desktop Outlook. |
| `security.allow_send_email` | `bool` | `false` | Reserved for future work. Sending email is not supported in the MVP. |
| `security.allow_create_draft` | `bool` | `false` | Enables `create_draft`. When `false`, the tool fails before any Outlook write. |
| `security.allow_create_event` | `bool` | `false` | Enables `create_event`. When `false`, the tool fails before any Outlook write. |
| `security.allow_save_attachments` | `bool` | `false` | Reserved for future attachment export support. |
| `storage.attachments_dir` | `string` | none | Absolute Windows path reserved for attachment export workflows. |
| `logging.level` | `string` | `"info"` | Structured stderr log level: `debug`, `info`, `warn`, or `error`. |
| `limits.max_results` | `int` | `20` | Maximum result cap applied to search/list tools. Allowed range: `1..100`. |

## Available Tools
| Tool | Description | Required params | Optional params |
|---|---|---|---|
| `search_emails` | Search Outlook emails by query and filters. | `query` | `folder`, `since`, `until`, `max_results` |
| `get_email` | Fetch one email by Outlook id. | `id` | none |
| `list_attachments` | List attachment metadata without downloading content. | `email_id` | none |
| `create_draft` | Save a draft email in Outlook. | `to`, `subject`, `body` | none |
| `list_events` | List calendar events in a time window. | none | `since`, `until`, `max_results` |
| `get_event` | Fetch one calendar event by Outlook id. | `id` | none |
| `create_event` | Save a simple non-meeting calendar event. | `title`, `start`, `end` | `location`, `body` |

## Usage with OpenCode
Add the server to your `opencode.json` MCP configuration:

```json
{
  "mcp": {
    "outlook": {
      "type": "stdio",
      "command": "C:\\tools\\outlook-mcp.exe",
      "args": ["--config", "C:\\tools\\outlook-mcp\\config.yaml"]
    }
  }
}
```

## Usage with Claude Desktop
Add the server to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "outlook": {
      "command": "C:\\tools\\outlook-mcp.exe",
      "args": ["--config", "C:\\tools\\outlook-mcp\\config.yaml"]
    }
  }
}
```

## Security
- `create_draft` and `create_event` require explicit allow flags in config.
- `send_email` is not supported in the MVP.
- All writes are policy-gated before any Outlook interaction.

## Testing
```bash
go test ./...
go test -tags integration ./...
```

Integration tests require Outlook Desktop running and a configured MAPI profile.

## Architecture
The project follows a small hexagonal structure:
- `cmd/outlook-mcp` contains the executable entrypoint and runtime wiring.
- `internal/mcp` owns MCP tool schemas, request parsing, and server transport.
- `internal/outlook` isolates all Outlook COM access behind store interfaces and a serialized executor.
- `internal/security`, `internal/config`, and `internal/logging` provide policy, runtime config, and structured logging services.
