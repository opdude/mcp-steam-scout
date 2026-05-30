// setup-xbox guides the user through Microsoft's device code flow and prints
// an XBOX_REFRESH_TOKEN they can add to their opencode.json config.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const clientID = "0000000048183522"

func main() {
	fmt.Println("=== Xbox Refresh Token Setup ===")
	fmt.Println()

	resp, err := http.PostForm("https://login.live.com/oauth20_connect.srf", url.Values{
		"client_id":     {clientID},
		"scope":         {"service::user.auth.xboxlive.com::MBI_SSL"},
		"response_type": {"device_code"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start device auth: %v\n", err)
		os.Exit(1)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var dc struct {
		UserCode        string `json:"user_code"`
		DeviceCode      string `json:"device_code"`
		VerificationURI string `json:"verification_uri"`
		Interval        int    `json:"interval"`
	}
	if err := json.Unmarshal(body, &dc); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("1. Open this URL in your browser:")
	fmt.Printf("   %s\n", dc.VerificationURI)
	fmt.Println()
	fmt.Println("2. Enter this code:")
	fmt.Printf("   %s\n", dc.UserCode)
	fmt.Println()
	fmt.Println("3. Sign in with your Microsoft/Xbox account.")
	fmt.Println()
	fmt.Print("Waiting for authentication...")

	interval := dc.Interval
	if interval < 5 {
		interval = 5
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		data := url.Values{
			"client_id":   {clientID},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {dc.DeviceCode},
		}
		resp, err := http.PostForm("https://login.live.com/oauth20_token.srf", data)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		var poll struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			RefreshToken     string `json:"refresh_token"`
		}
		if err := json.Unmarshal(body, &poll); err != nil {
			continue
		}

		switch poll.Error {
		case "authorization_pending":
			fmt.Print(".")
			continue
		case "slow_down":
			time.Sleep(5 * time.Second)
			continue
		case "":
			fmt.Println()
			fmt.Println()
			fmt.Println("Authentication successful!")
			fmt.Println()
			fmt.Println("Add this to your opencode.json environment section:")
			fmt.Printf("  \"XBOX_REFRESH_TOKEN\": \"%s\"\n", poll.RefreshToken)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "\nError: %s - %s\n", poll.Error, poll.ErrorDescription)
			os.Exit(1)
		}
	}
}
