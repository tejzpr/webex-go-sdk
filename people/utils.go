/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package people

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// EncodeBase64 encodes a string to base64
func EncodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// DecodeBase64 decodes a base64 string
func DecodeBase64(s string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// InferPersonIDFromUUID converts a UUID to a Hydra ID without a network call
func InferPersonIDFromUUID(id string) string {
	// Check if already a Hydra ID
	decodedID, err := DecodeBase64(id)
	if err == nil && strings.Contains(decodedID, "ciscospark://") {
		return id
	}

	// Convert UUID to Hydra ID
	return EncodeBase64(fmt.Sprintf("ciscospark://us/PEOPLE/%s", id))
}
