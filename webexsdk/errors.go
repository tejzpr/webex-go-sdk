/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webexsdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// APIError is the base error type for all Webex API errors.
// It provides structured access to the HTTP status code, error message,
// tracking ID, and raw response body. All specific error sub-types embed
// this struct, so consumers can use errors.As(err, &apiErr) to access
// common fields regardless of the specific error type.
type APIError struct {
	// StatusCode is the HTTP status code from the response.
	StatusCode int

	// Status is the HTTP status line (e.g., "404 Not Found").
	Status string

	// Message is the error message from the Webex API response body.
	Message string

	// TrackingID is the Webex tracking identifier for support debugging.
	TrackingID string

	// RetryAfter is the duration to wait before retrying, parsed from
	// the Retry-After header. Zero if not applicable.
	RetryAfter time.Duration

	// RawBody is the raw response body bytes, preserved for debugging.
	RawBody []byte

	// Err is an optional wrapped error for errors.Unwrap support.
	Err error
}

// Error implements the error interface.
func (e *APIError) Error() string {
	msg := fmt.Sprintf("API error: %d", e.StatusCode)
	if e.Message != "" {
		msg += " - " + e.Message
	}
	if e.TrackingID != "" {
		msg += " (trackingId: " + e.TrackingID + ")"
	}
	return msg
}

// Unwrap returns the wrapped error, if any.
func (e *APIError) Unwrap() error {
	return e.Err
}

// --- Specific error sub-types ---

// RateLimitError is returned for HTTP 429 Too Many Requests responses.
// The RetryAfter field (inherited from APIError) indicates how long to wait.
type RateLimitError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *RateLimitError) Unwrap() error { return e.APIError }

// AuthError is returned for HTTP 401 Unauthorized responses.
type AuthError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *AuthError) Unwrap() error { return e.APIError }

// ForbiddenError is returned for HTTP 403 Forbidden responses.
type ForbiddenError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *ForbiddenError) Unwrap() error { return e.APIError }

// NotFoundError is returned for HTTP 404 Not Found responses.
type NotFoundError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *NotFoundError) Unwrap() error { return e.APIError }

// ConflictError is returned for HTTP 409 Conflict responses.
type ConflictError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *ConflictError) Unwrap() error { return e.APIError }

// GoneError is returned for HTTP 410 Gone responses.
// For the Contents API, this indicates an infected file that was removed.
type GoneError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *GoneError) Unwrap() error { return e.APIError }

// LockedError is returned for HTTP 423 Locked responses.
// For the Contents API, this indicates a file is being scanned for malware.
// The RetryAfter field (inherited from APIError) indicates when to retry.
type LockedError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *LockedError) Unwrap() error { return e.APIError }

// PreconditionRequiredError is returned for HTTP 428 Precondition Required responses.
// For the Contents API, this indicates an unscannable file. Adding the query
// parameter allow=unscannable will enable the download.
type PreconditionRequiredError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *PreconditionRequiredError) Unwrap() error { return e.APIError }

// ServerError is returned for HTTP 5xx responses (500, 502, 503, 504).
type ServerError struct {
	*APIError
}

// Unwrap returns the underlying APIError for errors.As traversal.
func (e *ServerError) Unwrap() error { return e.APIError }

// --- Factory ---

// apiErrorBody is used to parse the Webex API error response JSON.
type apiErrorBody struct {
	Message    string `json:"message"`
	TrackingID string `json:"trackingId"`
}

// NewAPIError creates a structured error from an HTTP response and its body.
// It parses the JSON body for message and trackingId fields, reads the
// Retry-After header, and returns the appropriate error sub-type based
// on the HTTP status code.
func NewAPIError(resp *http.Response, body []byte) error {
	base := &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		RawBody:    body,
	}

	// Parse JSON body for message and trackingId
	var parsed apiErrorBody
	if len(body) > 0 {
		if err := json.Unmarshal(body, &parsed); err == nil {
			base.Message = parsed.Message
			base.TrackingID = parsed.TrackingID
		}
		// If JSON parsing fails, leave Message empty — RawBody preserves the original
	}

	// Parse Retry-After header
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if seconds, err := strconv.Atoi(ra); err == nil && seconds > 0 {
			base.RetryAfter = time.Duration(seconds) * time.Second
		}
	}

	// Return the appropriate sub-type
	switch resp.StatusCode {
	case http.StatusUnauthorized: // 401
		return &AuthError{APIError: base}
	case http.StatusForbidden: // 403
		return &ForbiddenError{APIError: base}
	case http.StatusNotFound: // 404
		return &NotFoundError{APIError: base}
	case http.StatusConflict: // 409
		return &ConflictError{APIError: base}
	case http.StatusGone: // 410
		return &GoneError{APIError: base}
	case 423: // Locked
		return &LockedError{APIError: base}
	case 428: // Precondition Required
		return &PreconditionRequiredError{APIError: base}
	case http.StatusTooManyRequests: // 429
		return &RateLimitError{APIError: base}
	case http.StatusInternalServerError, // 500
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return &ServerError{APIError: base}
	default:
		return base
	}
}

// --- Convenience functions ---

// IsRateLimited reports whether err is a rate limit error (HTTP 429).
func IsRateLimited(err error) bool {
	var e *RateLimitError
	return errors.As(err, &e)
}

// IsNotFound reports whether err is a not found error (HTTP 404).
func IsNotFound(err error) bool {
	var e *NotFoundError
	return errors.As(err, &e)
}

// IsAuthError reports whether err is an authentication error (HTTP 401).
func IsAuthError(err error) bool {
	var e *AuthError
	return errors.As(err, &e)
}

// IsForbidden reports whether err is a forbidden error (HTTP 403).
func IsForbidden(err error) bool {
	var e *ForbiddenError
	return errors.As(err, &e)
}

// IsConflict reports whether err is a conflict error (HTTP 409).
func IsConflict(err error) bool {
	var e *ConflictError
	return errors.As(err, &e)
}

// IsGone reports whether err is a gone error (HTTP 410).
func IsGone(err error) bool {
	var e *GoneError
	return errors.As(err, &e)
}

// IsLocked reports whether err is a locked error (HTTP 423).
func IsLocked(err error) bool {
	var e *LockedError
	return errors.As(err, &e)
}

// IsPreconditionRequired reports whether err is a precondition required error (HTTP 428).
func IsPreconditionRequired(err error) bool {
	var e *PreconditionRequiredError
	return errors.As(err, &e)
}

// IsServerError reports whether err is a server error (HTTP 5xx).
func IsServerError(err error) bool {
	var e *ServerError
	return errors.As(err, &e)
}

// FieldError represents an error on a specific field of a resource.
// When the Webex API encounters a partial failure retrieving a resource
// in a list response, individual fields may contain errors instead of
// their normal values.
type FieldError struct {
	// Code is the error code (e.g., "kms_failure").
	Code string `json:"code"`
	// Reason describes why the field could not be retrieved.
	Reason string `json:"reason"`
}

// ResourceErrors maps field names to their errors within a single resource.
// Present in list response items when partial failures occur (HTTP 200 with
// some items having field-level errors).
//
// Example JSON:
//
//	"errors": {
//	  "title": {
//	    "code": "kms_failure",
//	    "reason": "Key management server failed to respond appropriately."
//	  }
//	}
type ResourceErrors map[string]FieldError

// HasErrors returns true if this ResourceErrors map contains any errors.
func (re ResourceErrors) HasErrors() bool {
	return len(re) > 0
}

// HasFieldError returns true if the specified field has an error.
func (re ResourceErrors) HasFieldError(field string) bool {
	_, ok := re[field]
	return ok
}
