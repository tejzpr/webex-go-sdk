

# Webex Go SDK

A comprehensive, lightweight Go SDK for Cisco Webex API

[Go Reference](https://pkg.go.dev/github.com/WebexCommunity/webex-go-sdk/v2)
[License: MPL 2.0](https://opensource.org/licenses/MPL-2.0)
[Open Source](https://github.com/WebexCommunity/webex-go-sdk)
[Go Tests](https://github.com/WebexCommunity/webex-go-sdk/actions/workflows/go-test.yml)
[Lint](https://github.com/WebexCommunity/webex-go-sdk/actions/workflows/golangci-lint.yml)
[Codecov](https://codecov.io/gh/tejzpr/webex-go-sdk)
[Release](https://github.com/WebexCommunity/webex-go-sdk/releases/latest)
[Go Report Card](https://goreportcard.com/report/github.com/WebexCommunity/webex-go-sdk/v2)

## Implementation Status

- ✅ All REST APIs are fully implemented and working
- ✅ WebSocket APIs with end-to-end encrypted message decryption
- ✅ Real-time Webex Calling with WebRTC media (Mobius/BroadWorks)

## Installation

```bash
go get github.com/WebexCommunity/webex-go-sdk/v2
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/WebexCommunity/webex-go-sdk/v2"
)

func main() {
    // Get access token from environment
    accessToken := os.Getenv("WEBEX_ACCESS_TOKEN")
    if accessToken == "" {
        log.Fatal("WEBEX_ACCESS_TOKEN environment variable is required")
    }

    // Create client
    client, err := webex.NewClient(accessToken, nil)
    if err != nil {
        log.Fatalf("Error creating client: %v", err)
    }

    // Get my own details
    me, err := client.People().GetMe()
    if err != nil {
        log.Fatalf("Error getting my details: %v", err)
    }

    fmt.Printf("Hello, %s!\n", me.DisplayName)
}
```

## Supported APIs

### REST APIs (Fully Implemented)

- **People** - Manage users in your organization
- **Messages** - Send and receive messages in rooms
- **Rooms** - Create and manage Webex rooms
- **Teams** - Create and manage Webex teams
- **Team Memberships** - Add and remove people from teams
- **Memberships** - Add and remove people from rooms
- **Webhooks** - Register for notifications
- **Attachment Actions** - Handle interactive card submissions
- **Events** - Subscribe to Webex events
- **Room Tabs** - Manage tabs in Webex rooms
- **Meetings** - Create, list, update, and delete Webex meetings
- **Meeting Transcripts** - List, download, and manage meeting transcripts and snippets
- **Calling** - Call history, call settings (DND, call waiting, call forwarding, voicemail), contacts

### WebSocket APIs

- **Mercury** - Real-time WebSocket connection with automatic reconnection
- **Conversation Events** - Listen for messages, shares, and acknowledgements
- **End-to-End Encryption** - Full JWE decryption using KMS (ECDH key exchange + AES-256-GCM)

### Real-Time Call Control (Webex Calling)

- **CallingClient** - Line registration with Mobius, call lifecycle management, Mercury event routing
- **AudioBridge** - Browser-facing WebRTC PeerConnection with bidirectional RTP relay (PCMU/PCMA)
- **Call Control** - Dial, answer, hold, resume, transfer (blind/consult), DTMF, mute/unmute
- **SignalingTransport** - Transport-agnostic WebRTC signaling interface (WebSocket, gRPC, etc.)
- **Address Normalization** - Phone number sanitization and SIP/tel URI handling

## Examples

See the [examples](./examples) directory.

- [Calling Example](./examples/calling) - Web-based call control with browser audio bridge

### Sending a Message

```go
message := &messages.Message{
    RoomID: "ROOM_ID",
    Text:   "Hello, World!",
}

createdMessage, err := client.Messages().Create(message)
if err != nil {
    log.Fatalf("Error sending message: %v", err)
}
fmt.Printf("Message sent: ID=%s\n", createdMessage.ID)
```

## Documentation

For detailed documentation, examples, and API reference, see:

- [Go Reference Documentation](https://pkg.go.dev/github.com/WebexCommunity/webex-go-sdk/v2)
- [Examples Directory](./examples)
- [Cisco Webex API Documentation](https://developer.webex.com/docs/api/getting-started)

## Requirements

- Go 1.21 or later

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Add your name to [CONTRIBUTORS.md](./CONTRIBUTORS.md) if not already present
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

See [CONTRIBUTORS.md](./CONTRIBUTORS.md) for the list of contributors.

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](./LICENSE).