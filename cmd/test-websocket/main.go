// Temporary test script for WebSocket + decryption testing.
// Delete after use.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/conversation"
	"github.com/tejzpr/webex-go-sdk/v2/device"
	"github.com/tejzpr/webex-go-sdk/v2/mercury"
	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

func main() {
	token := os.Getenv("WEBEX_TOKEN")
	if token == "" {
		fmt.Println("WEBEX_TOKEN env var required")
		os.Exit(1)
	}

	fmt.Println("[1/4] Creating Webex client...")
	core, err := webexsdk.NewClient(token, nil)
	if err != nil {
		fmt.Printf("ERROR creating client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[2/4] Fetching user info + registering device...")
	userID, err := fetchUserID(token)
	if err != nil {
		fmt.Printf("ERROR fetching user ID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  User ID: %s\n", userID)

	deviceClient := device.New(core, nil)
	if err := deviceClient.Register(); err != nil {
		fmt.Printf("ERROR registering device: %v\n", err)
		os.Exit(1)
	}
	devURL, _ := deviceClient.GetDeviceURL()
	fmt.Printf("  Device URL: %s\n", truncate(devURL, 80))

	fmt.Println("[3/4] Setting up conversation + encryption client...")
	convClient := conversation.New(core, nil)
	convClient.SetEncryptionDeviceInfo(devURL, userID)
	encClient := convClient.EncryptionClient()
	encClient.SetDeviceInfo(devURL, userID)

	msgCount := 0

	convClient.On("post", func(activity *conversation.Activity) {
		msgCount++
		fmt.Printf("\n=== MESSAGE #%d ===\n", msgCount)
		fmt.Printf("  From: %s\n", activity.Actor.DisplayName)
		fmt.Printf("  EncryptionKeyURL: %s\n", truncate(activity.EncryptionKeyURL, 80))

		if activity.DecryptedObject != nil && activity.DecryptedObject.DisplayName != "" {
			raw := activity.DecryptedObject.DisplayName
			fmt.Printf("  Raw JWE (%d chars): %s\n", len(raw), truncate(raw, 80))
		}

		if activity.EncryptionKeyURL != "" && activity.DecryptedObject != nil && activity.DecryptedObject.DisplayName != "" {
			fmt.Printf("  Attempting decryption...\n")
			start := time.Now()
			plaintext, err := encClient.DecryptText(activity.EncryptionKeyURL, activity.DecryptedObject.DisplayName)
			elapsed := time.Since(start)
			if err != nil {
				fmt.Printf("  DECRYPT ERROR (%v): %v\n", elapsed, err)
			} else {
				fmt.Printf("  DECRYPTED (%v): %s\n", elapsed, plaintext)
			}
		}

		if activity.Target != nil {
			fmt.Printf("  Room: %s\n", truncate(activity.Target.ID, 60))
		}
		fmt.Println("==================")
	})

	fmt.Println("[4/4] Connecting to Mercury WebSocket...")
	mercuryClient := mercury.New(core, nil)
	mercuryClient.SetDeviceProvider(deviceClient)
	convClient.SetMercuryClient(mercuryClient)

	// Debug: log ALL Mercury events to see what arrives
	mercuryClient.On("*", func(event *mercury.Event) {
		if event.EventType == "mercury.buffer_state" || event.EventType == "mercury.registration_status" {
			return
		}
		fmt.Printf("  [DEBUG event] type=%s activity=%s\n", event.EventType, event.ActivityType)

		// For encryption events, show raw data keys
		if event.EventType == "encryption.kms_message" || event.EventType == "encryption" {
			if event.Data != nil {
				for k := range event.Data {
					fmt.Printf("    [DEBUG kms data key] %s\n", k)
				}
			}
		}
	})

	if err := mercuryClient.Connect(); err != nil {
		fmt.Printf("ERROR connecting to Mercury: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected! Listening for 120s.")
	fmt.Println(">>> Send a message in Webex to test. First msg may take ~30s.")
	fmt.Println()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\nStopping...")
	case <-time.After(120 * time.Second):
		fmt.Printf("\nTimeout. Received %d message(s).\n", msgCount)
	}

	mercuryClient.Disconnect()
	fmt.Println("Disconnected.")
}

func fetchUserID(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://webexapis.com/v1/people/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var info struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	}
	json.Unmarshal(body, &info)
	fmt.Printf("  Hello, %s!\n", info.DisplayName)
	return info.ID, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
