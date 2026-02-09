# Conversation Listen (Internal API)

This example demonstrates how to listen for real-time conversation events using
the Webex Go SDK's internal conversation API with automatic end-to-end
encryption/decryption via the KMS service.

## When to Use

Use the **conversation client** when you need:

- Real-time WebSocket message events with decryption
- Fine-grained control over activity types (`post`, `share`, `acknowledge`)
- Access to raw conversation activity data (actor, target, encryption metadata)

For a simpler message-only listener, see the
[messages-listen](../messages-listen/) example which uses `messages.Listen()`.

## Usage

```bash
export WEBEX_ACCESS_TOKEN="your-access-token"
go run main.go
```

## How It Works

The `client.Conversation()` convenience method handles all the internal wiring:

1. **Device Registration** -- Registers an SDK device with Webex (WDM service)
2. **Mercury WebSocket** -- Sets up the real-time WebSocket connection
3. **Encryption/KMS** -- Configures ECDH key exchange for message decryption

```go
client, _ := webex.NewClient(token, nil)

// One call replaces ~25 lines of manual device/mercury/encryption setup
conv, err := client.Conversation()

conv.On("post", func(activity *conversation.Activity) {
    content, _ := conv.GetMessageContent(activity)
    fmt.Println(content) 

conv.Connect()
defer conv.Disconnect()
```

## Advanced Usage

For full control over Device, Mercury, or Encryption configuration, you can
wire the components manually:

```go
client, _ := webex.NewClient(token, nil)

// Custom device configuration
deviceClient := device.New(client.Core(), &device.Config{
    DeviceType: "WEB",
})
deviceClient.Register()
deviceURL, _ := deviceClient.GetDeviceURL()
deviceInfo := deviceClient.GetDevice()

// Custom Mercury configuration
mercuryClient := mercury.New(client.Core(), &mercury.Config{
    PingInterval: 15 * time.Second,
    MaxRetries:   10,
})
mercuryClient.SetDeviceProvider(deviceClient)

// Manual conversation wiring
convClient := conversation.New(client.Core(), nil)
convClient.SetMercuryClient(mercuryClient)
convClient.SetEncryptionDeviceInfo(deviceURL, deviceInfo.UserID)

convClient.On("post", handler)
mercuryClient.Connect()
```
