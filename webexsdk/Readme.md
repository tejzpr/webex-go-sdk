# webexsdk

The `webexsdk` package is the core foundation of the Webex Go SDK. It provides the HTTP client, configuration, authentication, automatic retry with exponential backoff, pagination, structured error types, and the plugin system that all API modules build on.

> **Note:** Most users should import the top-level `webex` package, which lazily initialises all API modules. The `webexsdk` package is documented here for advanced use-cases, custom configuration, and error handling.

## Overview

This package provides:

1. **Client** — Authenticated HTTP client for the Webex REST API
2. **Config** — Timeout, retry, and base URL configuration
3. **Automatic Retry** — Exponential backoff for 429, 423, 502, 503, 504
4. **Pagination** — RFC 5988 Link header parsing via `Page`
5. **Structured Errors** — Type-safe error hierarchy with convenience checkers
6. **Multipart Upload** — `RequestMultipart` for file uploads
7. **Plugin System** — Interface for extending the client

## Installation

```go
import "github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
```

## Configuration

```go
cfg := &webexsdk.Config{
    BaseURL:        "https://webexapis.com/v1", // Default
    Timeout:        30 * time.Second,           // Default
    MaxRetries:     3,                          // Default (0 disables retries)
    RetryBaseDelay: 1 * time.Second,            // Default (exponential: delay * 2^attempt)
    Logger:         log.Default(),              // Any webexsdk.Logger (Printf method)
    HttpClient:     nil,                        // Custom *http.Client (optional)
    DefaultHeaders: map[string]string{          // Extra headers on every request
        "X-Custom": "value",
    },
}

client, err := webexsdk.NewClient("YOUR_ACCESS_TOKEN", cfg)
```

### Config Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `BaseURL` | `string` | `https://webexapis.com/v1` | Webex API base URL |
| `Timeout` | `time.Duration` | `30s` | HTTP client timeout |
| `MaxRetries` | `int` | `3` | Max retry attempts (0 = no retries) |
| `RetryBaseDelay` | `time.Duration` | `1s` | Initial retry delay (exponential backoff) |
| `Logger` | `Logger` | `log.Default()` | Logger with `Printf(format, v...)` |
| `HttpClient` | `*http.Client` | auto-created | Custom HTTP client |
| `DefaultHeaders` | `map[string]string` | empty | Headers added to every request |

## Automatic Retry

The SDK automatically retries requests that receive transient error responses:

| Status Code | Meaning | Retry Behaviour |
|-------------|---------|-----------------|
| **429** | Too Many Requests | Wait `Retry-After` header duration, then retry |
| **423** | Locked (malware scanning) | Wait `Retry-After` header duration, then retry |
| **502** | Bad Gateway | Exponential backoff |
| **503** | Service Unavailable | Exponential backoff |
| **504** | Gateway Timeout | Exponential backoff |

Backoff formula: `RetryBaseDelay * 2^attempt` (e.g., 1s → 2s → 4s → 8s).

When `Retry-After` is present (429 and 423), the header value overrides the calculated backoff.

All request methods support retry: `Request`, `RequestURL`, `RequestMultipart`.

## Pagination

List endpoints return paginated results. The `Page` type parses RFC 5988 `Link` headers automatically:

```go
// First page
resp, err := client.Request(http.MethodGet, "rooms", params, nil)
page, err := webexsdk.NewPage(resp, client, "rooms")

// Iterate through items
for _, raw := range page.Items {
    var room rooms.Room
    json.Unmarshal(raw, &room)
}

// Next page
if page.HasNext {
    nextPage, err := page.Next()
}

// Previous page
if page.HasPrev {
    prevPage, err := page.Prev()
}
```

### Page Fields

| Field | Type | Description |
|-------|------|-------------|
| `Items` | `[]json.RawMessage` | Raw JSON items (unmarshal into your type) |
| `HasNext` | `bool` | Whether a next page exists |
| `HasPrev` | `bool` | Whether a previous page exists |
| `NextPage` | `string` | Absolute URL for the next page |
| `PrevPage` | `string` | Absolute URL for the previous page |

### Direct Cursor Navigation

Save a cursor URL and jump directly to any page later — no sequential traversal needed:

```go
// Session 1: paginate and save a cursor
page, _ := webhooksClient.List(&webhooks.ListOptions{Max: 10})
cursor := page.NextPage  // save this (e.g., to a database or cache)

// Session 2 (later): jump directly to the saved page
resumedPage, _ := client.PageFromCursor(cursor)
for _, raw := range resumedPage.Items {
    // process items
}

// Continue pagination from the resumed position
if resumedPage.HasNext {
    nextPage, _ := resumedPage.Next()
    // ...
}
```

This is ideal for:
- **Bookmarking** a position in a large result set
- **Resuming** pagination after a process restart
- **Skipping** pages you've already processed

#### Supported Modules

All modules with list/pagination endpoints embed `*Page` and support `PageFromCursor`:

| Module | Page Type | List Method |
|--------|-----------|-------------|
| `rooms` | `RoomsPage` | `List(&ListOptions{})` |
| `messages` | `MessagesPage` | `List(&ListOptions{})` |
| `teams` | `TeamsPage` | `List(&ListOptions{})` |
| `webhooks` | `WebhooksPage` | `List(&ListOptions{})` |
| `meetings` | `MeetingsPage` | `List(&ListOptions{})` |
| `events` | `EventsPage` | `List(&ListOptions{})` |
| `memberships` | `MembershipsPage` | `List(&ListOptions{})` |
| `teammemberships` | `TeamMembershipsPage` | `List(&ListOptions{})` |
| `recordings` | `RecordingsPage` | `List(&ListOptions{})` |
| `transcripts` | `TranscriptsPage` | `List(&ListOptions{})` |
| `roomtabs` | `RoomTabsPage` | `List(&ListOptions{})` |
| `people` | `PeoplePage` | `List(&ListOptions{})` |

## Structured Errors

All API errors are returned as typed structs. The base type is `APIError`, with specific sub-types for common HTTP status codes:

| Error Type | HTTP Status | Description |
|------------|-------------|-------------|
| `*AuthError` | 401 | Invalid or expired access token |
| `*ForbiddenError` | 403 | Insufficient permissions |
| `*NotFoundError` | 404 | Resource not found |
| `*ConflictError` | 409 | Resource conflict |
| `*GoneError` | 410 | Resource permanently removed (e.g., infected file) |
| `*LockedError` | 423 | Resource locked (e.g., file being scanned) |
| `*PreconditionRequiredError` | 428 | Precondition required (e.g., unscannable file) |
| `*RateLimitError` | 429 | Rate limited (`RetryAfter` field set) |
| `*ServerError` | 5xx | Server-side error |
| `*APIError` | other | Generic API error |

### APIError Fields

| Field | Type | Description |
|-------|------|-------------|
| `StatusCode` | `int` | HTTP status code |
| `Status` | `string` | HTTP status line |
| `Message` | `string` | Error message from API response body |
| `TrackingID` | `string` | Webex tracking ID for support |
| `RetryAfter` | `time.Duration` | Retry wait time (429, 423) |
| `RawBody` | `[]byte` | Raw response body |

### Convenience Functions

Use `errors.As` or these helpers to check error types:

```go
resp, err := client.Request(http.MethodGet, "rooms/invalid", nil, nil)
if err != nil {
    switch {
    case webexsdk.IsNotFound(err):
        // Handle 404
    case webexsdk.IsAuthError(err):
        // Handle 401
    case webexsdk.IsForbidden(err):
        // Handle 403
    case webexsdk.IsRateLimited(err):
        // Handle 429 — check RetryAfter
        var rl *webexsdk.RateLimitError
        if errors.As(err, &rl) {
            time.Sleep(rl.RetryAfter)
        }
    case webexsdk.IsGone(err):
        // Handle 410
    case webexsdk.IsLocked(err):
        // Handle 423
    case webexsdk.IsPreconditionRequired(err):
        // Handle 428
    case webexsdk.IsConflict(err):
        // Handle 409
    case webexsdk.IsServerError(err):
        // Handle 5xx
    default:
        // Generic API error
        var apiErr *webexsdk.APIError
        if errors.As(err, &apiErr) {
            log.Printf("API error %d: %s (trackingId: %s)",
                apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
        }
    }
}
```

## Request Methods

| Method | Description |
|--------|-------------|
| `Request(method, path, params, body)` | Relative path request with retry |
| `RequestWithContext(ctx, method, path, params, body)` | Single request with context (no retry) |
| `RequestWithRetry(ctx, method, path, params, body)` | Relative path request with context + retry |
| `RequestURL(method, fullURL, body)` | Absolute URL request with retry |
| `RequestURLWithRetry(ctx, method, fullURL, body)` | Absolute URL request with context + retry |
| `RequestMultipart(path, fields, files)` | Multipart form-data POST with retry |
| `RequestMultipartWithRetry(ctx, path, fields, files)` | Multipart POST with context + retry |
| `PageFromCursor(cursorURL)` | Direct navigation to a page via saved cursor URL |

## Plugin System

API modules implement the `Plugin` interface and can be registered with the client:

```go
type Plugin interface {
    Name() string
}

client.RegisterPlugin(myPlugin)
plugin, ok := client.GetPlugin("myPlugin")
```

## Logger Interface

Any type implementing `Printf` can be used as the SDK logger:

```go
type Logger interface {
    Printf(format string, v ...any)
}
```

The standard library's `*log.Logger` satisfies this interface.

## Related Resources

- [Go Reference Documentation](https://pkg.go.dev/github.com/WebexCommunity/webex-go-sdk/v2/webexsdk)
- [Webex API Documentation](https://developer.webex.com/docs/api/getting-started)
