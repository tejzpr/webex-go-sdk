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
	"io"
	"net/http"
	"testing"
	"time"
)

func TestAPIError_ImplementsError(t *testing.T) {
	var err error = &APIError{
		StatusCode: 400,
		Status:     "400 Bad Request",
		Message:    "bad request",
	}

	if err.Error() == "" {
		t.Error("APIError.Error() returned empty string")
	}
}

func TestAPIError_ErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		contains []string
	}{
		{
			name: "With tracking ID",
			err: &APIError{
				StatusCode: 404,
				Status:     "404 Not Found",
				Message:    "resource not found",
				TrackingID: "ROUTER_abc123",
			},
			contains: []string{"404", "resource not found", "ROUTER_abc123"},
		},
		{
			name: "Without tracking ID",
			err: &APIError{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
				Message:    "internal error",
			},
			contains: []string{"500", "internal error"},
		},
		{
			name: "With RetryAfter",
			err: &APIError{
				StatusCode: 429,
				Status:     "429 Too Many Requests",
				Message:    "rate limited",
				RetryAfter: 60 * time.Second,
			},
			contains: []string{"429", "rate limited"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := tc.err.Error()
			for _, s := range tc.contains {
				if !containsStr(msg, s) {
					t.Errorf("Expected error message to contain %q, got %q", s, msg)
				}
			}
		})
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("network timeout")
	err := &APIError{
		StatusCode: 502,
		Message:    "bad gateway",
		Err:        inner,
	}

	if !errors.Is(err, inner) {
		t.Error("Expected APIError to unwrap to inner error")
	}
}

// --- Sub-type tests: each sub-type embeds *APIError ---

func TestRateLimitError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 429, Message: "rate limited", RetryAfter: 60 * time.Second}
	err := &RateLimitError{APIError: apiErr}

	// Should satisfy errors.As for both RateLimitError and APIError
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatal("Expected errors.As to match *RateLimitError")
	}
	if rle.RetryAfter != 60*time.Second {
		t.Errorf("Expected RetryAfter 60s, got %v", rle.RetryAfter)
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
	if ae.StatusCode != 429 {
		t.Errorf("Expected status 429, got %d", ae.StatusCode)
	}
}

func TestNotFoundError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 404, Message: "not found"}
	err := &NotFoundError{APIError: apiErr}

	var nfe *NotFoundError
	if !errors.As(err, &nfe) {
		t.Fatal("Expected errors.As to match *NotFoundError")
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
	if ae.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", ae.StatusCode)
	}
}

func TestAuthError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 401, Message: "unauthorized"}
	err := &AuthError{APIError: apiErr}

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatal("Expected errors.As to match *AuthError")
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
	if ae.StatusCode != 401 {
		t.Errorf("Expected status 401, got %d", ae.StatusCode)
	}
}

func TestForbiddenError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 403, Message: "forbidden"}
	err := &ForbiddenError{APIError: apiErr}

	var fe *ForbiddenError
	if !errors.As(err, &fe) {
		t.Fatal("Expected errors.As to match *ForbiddenError")
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
}

func TestConflictError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 409, Message: "conflict"}
	err := &ConflictError{APIError: apiErr}

	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatal("Expected errors.As to match *ConflictError")
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
}

func TestLockedError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 423, Message: "locked", RetryAfter: 30 * time.Second}
	err := &LockedError{APIError: apiErr}

	var le *LockedError
	if !errors.As(err, &le) {
		t.Fatal("Expected errors.As to match *LockedError")
	}
	if le.RetryAfter != 30*time.Second {
		t.Errorf("Expected RetryAfter 30s, got %v", le.RetryAfter)
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
}

func TestPreconditionRequiredError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 428, Message: "precondition required"}
	err := &PreconditionRequiredError{APIError: apiErr}

	var pre *PreconditionRequiredError
	if !errors.As(err, &pre) {
		t.Fatal("Expected errors.As to match *PreconditionRequiredError")
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
}

func TestGoneError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 410, Message: "gone"}
	err := &GoneError{APIError: apiErr}

	var ge *GoneError
	if !errors.As(err, &ge) {
		t.Fatal("Expected errors.As to match *GoneError")
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
}

func TestServerError_ErrorsAs(t *testing.T) {
	apiErr := &APIError{StatusCode: 502, Message: "bad gateway"}
	err := &ServerError{APIError: apiErr}

	var se *ServerError
	if !errors.As(err, &se) {
		t.Fatal("Expected errors.As to match *ServerError")
	}

	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatal("Expected errors.As to match *APIError")
	}
}

// --- NewAPIError factory tests ---

func TestNewAPIError_Returns_CorrectSubtype(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		body         string
		retryAfter   string
		expectType   string
		expectMsg    string
		expectTrkID  string
		expectRetry  time.Duration
	}{
		{
			name:        "400 Bad Request",
			statusCode:  400,
			body:        `{"message":"invalid field","trackingId":"TRK_400"}`,
			expectType:  "*webexsdk.APIError",
			expectMsg:   "invalid field",
			expectTrkID: "TRK_400",
		},
		{
			name:        "401 Unauthorized",
			statusCode:  401,
			body:        `{"message":"token expired","trackingId":"TRK_401"}`,
			expectType:  "*webexsdk.AuthError",
			expectMsg:   "token expired",
			expectTrkID: "TRK_401",
		},
		{
			name:        "403 Forbidden",
			statusCode:  403,
			body:        `{"message":"no access","trackingId":"TRK_403"}`,
			expectType:  "*webexsdk.ForbiddenError",
			expectMsg:   "no access",
			expectTrkID: "TRK_403",
		},
		{
			name:        "404 Not Found",
			statusCode:  404,
			body:        `{"message":"not found","trackingId":"TRK_404"}`,
			expectType:  "*webexsdk.NotFoundError",
			expectMsg:   "not found",
			expectTrkID: "TRK_404",
		},
		{
			name:        "409 Conflict",
			statusCode:  409,
			body:        `{"message":"duplicate","trackingId":"TRK_409"}`,
			expectType:  "*webexsdk.ConflictError",
			expectMsg:   "duplicate",
			expectTrkID: "TRK_409",
		},
		{
			name:        "410 Gone",
			statusCode:  410,
			body:        `{"message":"infected file removed"}`,
			expectType:  "*webexsdk.GoneError",
			expectMsg:   "infected file removed",
		},
		{
			name:        "423 Locked with Retry-After",
			statusCode:  423,
			body:        `{"message":"file being scanned"}`,
			retryAfter:  "60",
			expectType:  "*webexsdk.LockedError",
			expectMsg:   "file being scanned",
			expectRetry: 60 * time.Second,
		},
		{
			name:        "428 Precondition Required",
			statusCode:  428,
			body:        `{"message":"unscannable file"}`,
			expectType:  "*webexsdk.PreconditionRequiredError",
			expectMsg:   "unscannable file",
		},
		{
			name:        "429 Too Many Requests",
			statusCode:  429,
			body:        `{"message":"rate limited"}`,
			retryAfter:  "3600",
			expectType:  "*webexsdk.RateLimitError",
			expectMsg:   "rate limited",
			expectRetry: 3600 * time.Second,
		},
		{
			name:        "500 Internal Server Error",
			statusCode:  500,
			body:        `{"message":"internal error"}`,
			expectType:  "*webexsdk.ServerError",
			expectMsg:   "internal error",
		},
		{
			name:        "502 Bad Gateway",
			statusCode:  502,
			body:        `{"message":"bad gateway"}`,
			expectType:  "*webexsdk.ServerError",
			expectMsg:   "bad gateway",
		},
		{
			name:        "503 Service Unavailable",
			statusCode:  503,
			body:        `{"message":"unavailable"}`,
			expectType:  "*webexsdk.ServerError",
			expectMsg:   "unavailable",
		},
		{
			name:        "504 Gateway Timeout",
			statusCode:  504,
			body:        `{"message":"timeout"}`,
			expectType:  "*webexsdk.ServerError",
			expectMsg:   "timeout",
		},
		{
			name:        "415 Unsupported Media Type (generic)",
			statusCode:  415,
			body:        `{"message":"unsupported media type"}`,
			expectType:  "*webexsdk.APIError",
			expectMsg:   "unsupported media type",
		},
		{
			name:        "Non-JSON body",
			statusCode:  500,
			body:        `Internal Server Error`,
			expectType:  "*webexsdk.ServerError",
			expectMsg:   "", // Message field empty, RawBody has the text
		},
		{
			name:        "Empty body",
			statusCode:  400,
			body:        ``,
			expectType:  "*webexsdk.APIError",
			expectMsg:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Status:     fmt.Sprintf("%d %s", tc.statusCode, http.StatusText(tc.statusCode)),
				Header:     http.Header{},
			}
			if tc.retryAfter != "" {
				resp.Header.Set("Retry-After", tc.retryAfter)
			}

			err := NewAPIError(resp, []byte(tc.body))

			// Check it's an error
			if err == nil {
				t.Fatal("Expected non-nil error")
			}

			// Check type
			got := fmt.Sprintf("%T", err)
			if got != tc.expectType {
				t.Errorf("Expected type %s, got %s", tc.expectType, got)
			}

			// Check message via errors.As to APIError
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatal("Expected errors.As to match *APIError")
			}

			if tc.expectMsg != "" && apiErr.Message != tc.expectMsg {
				t.Errorf("Expected message %q, got %q", tc.expectMsg, apiErr.Message)
			}

			if tc.expectTrkID != "" && apiErr.TrackingID != tc.expectTrkID {
				t.Errorf("Expected trackingId %q, got %q", tc.expectTrkID, apiErr.TrackingID)
			}

			if tc.expectRetry > 0 && apiErr.RetryAfter != tc.expectRetry {
				t.Errorf("Expected RetryAfter %v, got %v", tc.expectRetry, apiErr.RetryAfter)
			}

			if apiErr.StatusCode != tc.statusCode {
				t.Errorf("Expected StatusCode %d, got %d", tc.statusCode, apiErr.StatusCode)
			}

			// RawBody should always be preserved
			if string(apiErr.RawBody) != tc.body {
				t.Errorf("Expected RawBody %q, got %q", tc.body, string(apiErr.RawBody))
			}
		})
	}
}

// --- Convenience functions ---

func TestIsRateLimited(t *testing.T) {
	err := &RateLimitError{APIError: &APIError{StatusCode: 429}}
	if !IsRateLimited(err) {
		t.Error("Expected IsRateLimited to return true")
	}

	notRateErr := &APIError{StatusCode: 400}
	if IsRateLimited(notRateErr) {
		t.Error("Expected IsRateLimited to return false for 400")
	}

	if IsRateLimited(fmt.Errorf("plain error")) {
		t.Error("Expected IsRateLimited to return false for plain error")
	}
}

func TestIsNotFound(t *testing.T) {
	err := &NotFoundError{APIError: &APIError{StatusCode: 404}}
	if !IsNotFound(err) {
		t.Error("Expected IsNotFound to return true")
	}

	if IsNotFound(fmt.Errorf("plain error")) {
		t.Error("Expected IsNotFound to return false for plain error")
	}
}

func TestIsGone(t *testing.T) {
	err := &GoneError{APIError: &APIError{StatusCode: 410}}
	if !IsGone(err) {
		t.Error("Expected IsGone to return true")
	}

	if IsGone(fmt.Errorf("plain error")) {
		t.Error("Expected IsGone to return false for plain error")
	}
}

func TestIsLocked(t *testing.T) {
	err := &LockedError{APIError: &APIError{StatusCode: 423}}
	if !IsLocked(err) {
		t.Error("Expected IsLocked to return true")
	}

	if IsLocked(fmt.Errorf("plain error")) {
		t.Error("Expected IsLocked to return false for plain error")
	}
}

func TestIsPreconditionRequired(t *testing.T) {
	err := &PreconditionRequiredError{APIError: &APIError{StatusCode: 428}}
	if !IsPreconditionRequired(err) {
		t.Error("Expected IsPreconditionRequired to return true")
	}

	if IsPreconditionRequired(fmt.Errorf("plain error")) {
		t.Error("Expected IsPreconditionRequired to return false for plain error")
	}
}

func TestIsAuthError(t *testing.T) {
	err := &AuthError{APIError: &APIError{StatusCode: 401}}
	if !IsAuthError(err) {
		t.Error("Expected IsAuthError to return true")
	}

	if IsAuthError(fmt.Errorf("plain error")) {
		t.Error("Expected IsAuthError to return false for plain error")
	}
}

func TestIsForbidden(t *testing.T) {
	err := &ForbiddenError{APIError: &APIError{StatusCode: 403}}
	if !IsForbidden(err) {
		t.Error("Expected IsForbidden to return true")
	}

	if IsForbidden(fmt.Errorf("plain error")) {
		t.Error("Expected IsForbidden to return false for plain error")
	}
}

func TestIsConflict(t *testing.T) {
	err := &ConflictError{APIError: &APIError{StatusCode: 409}}
	if !IsConflict(err) {
		t.Error("Expected IsConflict to return true")
	}

	if IsConflict(fmt.Errorf("plain error")) {
		t.Error("Expected IsConflict to return false for plain error")
	}
}

func TestIsServerError(t *testing.T) {
	err := &ServerError{APIError: &APIError{StatusCode: 502}}
	if !IsServerError(err) {
		t.Error("Expected IsServerError to return true")
	}

	if IsServerError(fmt.Errorf("plain error")) {
		t.Error("Expected IsServerError to return false for plain error")
	}
}

// --- ParseResponse integration with structured errors ---

func TestParseResponse_ReturnsStructuredErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		headers    http.Header
		checkFunc  func(err error) bool
		checkName  string
	}{
		{
			name:       "429 returns RateLimitError",
			statusCode: 429,
			body:       `{"message":"rate limited"}`,
			headers:    http.Header{"Retry-After": []string{"60"}},
			checkFunc:  IsRateLimited,
			checkName:  "IsRateLimited",
		},
		{
			name:       "404 returns NotFoundError",
			statusCode: 404,
			body:       `{"message":"not found"}`,
			headers:    http.Header{},
			checkFunc:  IsNotFound,
			checkName:  "IsNotFound",
		},
		{
			name:       "401 returns AuthError",
			statusCode: 401,
			body:       `{"message":"unauthorized"}`,
			headers:    http.Header{},
			checkFunc:  IsAuthError,
			checkName:  "IsAuthError",
		},
		{
			name:       "500 returns ServerError",
			statusCode: 500,
			body:       `{"message":"internal error"}`,
			headers:    http.Header{},
			checkFunc:  IsServerError,
			checkName:  "IsServerError",
		},
		{
			name:       "423 returns LockedError",
			statusCode: 423,
			body:       `{"message":"locked"}`,
			headers:    http.Header{"Retry-After": []string{"30"}},
			checkFunc:  IsLocked,
			checkName:  "IsLocked",
		},
		{
			name:       "410 returns GoneError",
			statusCode: 410,
			body:       `{"message":"gone"}`,
			headers:    http.Header{},
			checkFunc:  IsGone,
			checkName:  "IsGone",
		},
		{
			name:       "428 returns PreconditionRequiredError",
			statusCode: 428,
			body:       `{"message":"precondition required"}`,
			headers:    http.Header{},
			checkFunc:  IsPreconditionRequired,
			checkName:  "IsPreconditionRequired",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Status:     fmt.Sprintf("%d %s", tc.statusCode, http.StatusText(tc.statusCode)),
				Header:     tc.headers,
				Body:       io.NopCloser(newMockReadCloser(tc.body)),
			}

			var data map[string]string
			err := ParseResponse(resp, &data)

			if err == nil {
				t.Fatal("Expected error for status", tc.statusCode)
			}

			if !tc.checkFunc(err) {
				t.Errorf("Expected %s to return true, got false. Error type: %T", tc.checkName, err)
			}

			// Also check it satisfies APIError
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatal("Expected errors.As to match *APIError")
			}
			if apiErr.StatusCode != tc.statusCode {
				t.Errorf("Expected status %d, got %d", tc.statusCode, apiErr.StatusCode)
			}
		})
	}
}

func TestParseResponse_Success_StillWorks(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(newMockReadCloser(`{"key":"value"}`)),
	}

	var data map[string]string
	err := ParseResponse(resp, &data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if data["key"] != "value" {
		t.Errorf("Expected key=value, got %v", data)
	}
}

// containsStr is a helper to check substring.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- Partial failure tests ---

func TestResourceErrors_HasErrors(t *testing.T) {
	re := ResourceErrors{
		"title": FieldError{Code: "kms_failure", Reason: "KMS failed"},
	}
	if !re.HasErrors() {
		t.Error("Expected HasErrors to be true")
	}

	empty := ResourceErrors{}
	if empty.HasErrors() {
		t.Error("Expected HasErrors to be false for empty map")
	}

	var nilRE ResourceErrors
	if nilRE.HasErrors() {
		t.Error("Expected HasErrors to be false for nil map")
	}
}

func TestResourceErrors_HasFieldError(t *testing.T) {
	re := ResourceErrors{
		"title": FieldError{Code: "kms_failure", Reason: "KMS failed"},
	}
	if !re.HasFieldError("title") {
		t.Error("Expected HasFieldError('title') to be true")
	}
	if re.HasFieldError("id") {
		t.Error("Expected HasFieldError('id') to be false")
	}
}

func TestResourceErrors_UnmarshalJSON(t *testing.T) {
	// Test that ResourceErrors correctly unmarshals from Webex API JSON
	jsonData := `{
		"id": "Y2lzY29zcGFyazovL3VzL1JPT00vNTIxN0EyMzY",
		"title": "eyJhbGciOiIiLCJraWQiOiIiLCJlbmMiOiIifQ....",
		"errors": {
			"title": {
				"code": "kms_failure",
				"reason": "Key management server failed to respond appropriately."
			}
		}
	}`

	var item struct {
		ID     string         `json:"id"`
		Title  string         `json:"title"`
		Errors ResourceErrors `json:"errors,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &item); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if item.ID != "Y2lzY29zcGFyazovL3VzL1JPT00vNTIxN0EyMzY" {
		t.Errorf("Unexpected ID: %s", item.ID)
	}

	if !item.Errors.HasErrors() {
		t.Fatal("Expected errors to be present")
	}

	if !item.Errors.HasFieldError("title") {
		t.Error("Expected title field error")
	}

	titleErr := item.Errors["title"]
	if titleErr.Code != "kms_failure" {
		t.Errorf("Expected code 'kms_failure', got %q", titleErr.Code)
	}
	if titleErr.Reason != "Key management server failed to respond appropriately." {
		t.Errorf("Unexpected reason: %s", titleErr.Reason)
	}
}

func TestResourceErrors_OmitEmpty(t *testing.T) {
	// Items without errors should not have an errors field in JSON
	jsonData := `{
		"id": "room1",
		"title": "Normal Room"
	}`

	var item struct {
		ID     string         `json:"id"`
		Title  string         `json:"title"`
		Errors ResourceErrors `json:"errors,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &item); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if item.Errors.HasErrors() {
		t.Error("Expected no errors for normal item")
	}
}
