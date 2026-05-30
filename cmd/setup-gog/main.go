// setup-gog guides the user through GOG OAuth setup and prints
// a GOG_REFRESH_TOKEN they can add to their opencode.json config.
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
	clientID     = "46899977096215655"
	clientSecret = "9d85c43b1482497dbbce61f6e4aa173a433796eeae2ca8c5f6129f2dc4de46d9"
	tokenURL     = "https://auth.gog.com/token"
	authURL      = "https://auth.gog.com/auth?client_id=46899977096215655&redirect_uri=https://embed.gog.com/on_login_success?origin=client&response_type=code&layout=client2"
	libraryURL   = "https://embed.gog.com/account/getFilteredProducts?mediaType=1"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	SessionID    string `json:"session_id"`
}

func main() {
	fmt.Println("=== GOG Refresh Token Setup ===")
	fmt.Println()
	fmt.Println("1. Open this URL in your browser and sign in:")
	fmt.Println()
	fmt.Println("   " + authURL)
	fmt.Println()
	fmt.Println("2. After logging in, you'll be redirected to a page.")
	fmt.Println("   Copy the entire URL from the address bar.")
	fmt.Println("   It will contain '?code=...'.")
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
	fmt.Println()
	fmt.Print("Verifying library access... ")

	count, err := fetchLibrary(tokenResp.AccessToken)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAILED")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OK")
	fmt.Printf("Found %d games in your GOG library.\n", count)
	fmt.Println()
	fmt.Println("Add this to your opencode.json environment section:")
	fmt.Printf("  \"GOG_REFRESH_TOKEN\": \"%s\"\n", tokenResp.RefreshToken)
}

func extractCode(input string) string {
	if u, err := url.Parse(input); err == nil {
		if c := u.Query().Get("code"); c != "" {
			return c
		}
	}
	return ""
}

func exchangeCode(code string) (*tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"https://embed.gog.com/on_login_success?origin=client"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	req, err := http.NewRequest("GET", tokenURL+"?"+form.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

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

	if result.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}

	return &result, nil
}

func fetchLibrary(accessToken string) (int, error) {
	req, err := http.NewRequest("GET", libraryURL, nil)
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
		Products []interface{} `json:"products"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return len(result.Products), nil
}
