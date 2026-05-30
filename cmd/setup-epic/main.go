// setup-epic guides the user through Epic Games Store OAuth setup and prints
// an EPIC_REFRESH_TOKEN they can add to their opencode.json config.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	clientID     = "34a02cf8f4414e29b15921876da36f9a"
	clientSecret = "daafbccc737745039dffe53d94fc76cf"
	tokenURL     = "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/token"
	libraryURL   = "https://library-service.live.use1a.on.epicgames.com/library/api/public/items"
)

type tokenResponse struct {
	AccessToken    string `json:"access_token"`
	ExpiresIn      int    `json:"expires_in"`
	RefreshToken   string `json:"refresh_token"`
	RefreshExpires int    `json:"refresh_expires"`
	AccountID      string `json:"account_id"`
	DisplayName    string `json:"displayName"`
	Error          string `json:"error"`
	ErrorMessage   string `json:"errorMessage"`
}

func main() {
	fmt.Println("=== Epic Games Store Refresh Token Setup ===")
	fmt.Println()
	fmt.Println("1. Open this URL in your browser and sign in:")
	fmt.Println()
	fmt.Println("   https://www.epicgames.com/id/login?redirectUrl=https%3A%2F%2Fwww.epicgames.com%2Fid%2Fapi%2Fredirect%3FclientId%3D34a02cf8f4414e29b15921876da36f9a%26responseType%3Dcode")
	fmt.Println()
	fmt.Println("2. After logging in, you'll be redirected to a page.")
	fmt.Println("   Copy the entire URL from the address bar.")
	fmt.Println("   It will contain '?code=...' or 'authorizationCode=...'.")
	fmt.Println()
	fmt.Print("3. Paste the URL (or just the code) here: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
	input = strings.TrimSpace(input)

	code := extractCode(input)
	if code == "" {
		fmt.Fprintln(os.Stderr, "Error: could not find authorization code in input")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Print("Exchanging code for tokens... ")

	tokenResp, err := exchangeCode(code)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAILED")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OK")
	fmt.Printf("Logged in as: %s\n", tokenResp.DisplayName)
	fmt.Println()

	fmt.Print("Verifying library access... ")

	games, err := fetchLibrary(tokenResp.AccessToken)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAILED")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OK")
	fmt.Printf("Found %d games in your Epic library.\n", games)
	fmt.Println()
	fmt.Println("Add this to your opencode.json environment section:")
	fmt.Printf("  \"EPIC_REFRESH_TOKEN\": \"%s\"\n", tokenResp.RefreshToken)
}

func extractCode(input string) string {
	// Try parsing as a URL with query parameter
	if u, err := url.Parse(input); err == nil {
		if c := u.Query().Get("code"); c != "" {
			return c
		}
		if c := u.Query().Get("authorizationCode"); c != "" {
			return c
		}
	}
	// If there's a fragment with authorizationCode
	if strings.Contains(input, "authorizationCode=") {
		parts := strings.Split(input, "authorizationCode=")
		if len(parts) == 2 {
			return strings.Split(parts[1], "&")[0]
		}
	}
	return ""
}

func exchangeCode(code string) (*tokenResponse, error) {
	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
		"token_type": {"eg1"},
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result tokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("API error: %s - %s", result.Error, result.ErrorMessage)
	}
	if result.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}

	return &result, nil
}

func fetchLibrary(accessToken string) (int, error) {
	req, err := http.NewRequest("GET", libraryURL+"?includeMetadata=true", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Records []interface{} `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return len(result.Records), nil
}
