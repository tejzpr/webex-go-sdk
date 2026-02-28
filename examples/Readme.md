# Examples

Runnable examples demonstrating each module of the Webex Go SDK.

## Prerequisites

All examples require a `WEBEX_ACCESS_TOKEN` environment variable. Some examples require additional environment variables — see the source code for details.

```bash
export WEBEX_ACCESS_TOKEN="your-token-here"
```

## Available Examples

| Directory | Description |
|-----------|-------------|
| [attachmentactions](./attachmentactions) | Send an Adaptive Card and retrieve the attachment action submission |
| [calling](./calling) | Web-based call control demo with call history, settings, voicemail, contacts, and real-time calling (WebRTC) |
| [conversation-listen-internal](./conversation-listen-internal) | Listen for real-time conversation events over Mercury WebSocket with E2E encryption/decryption |
| [events](./events) | List and retrieve Webex compliance/audit events with filters |
| [meetings](./meetings) | List meeting series and past instances, get meeting details |
| [memberships](./memberships) | List, add, update (moderator), and remove room memberships |
| [messages](./messages) | Send, get, list, and delete messages in a room |
| [messages-listen](./messages-listen) | Real-time WebSocket message listener with graceful shutdown |
| [people](./people) | Get current user, list people by email, get by ID |
| [rooms](./rooms) | Create, get, list, update, and delete rooms |
| [roomtabs](./roomtabs) | Create, list, get, update, and delete room tabs (pinned URLs) |
| [teammemberships](./teammemberships) | List, add, update, and remove team memberships |
| [teams](./teams) | List, create, get, update, and delete teams |
| [transcripts](./transcripts) | List transcripts, get metadata and snippets, download full transcript |
| [webhooks](./webhooks) | Create, list, update, delete webhooks and receive callbacks via HTTP server |

## Running an Example

```bash
cd examples/rooms
go run main.go
```