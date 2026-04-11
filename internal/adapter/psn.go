package adapter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const (
	// psnClientID and psnClientSecret are the credentials for Sony's official
	// PlayStation Android app, widely used by the community for PSN API access.
	psnClientID     = "09515159-7237-4370-9b40-3806e67c0891"
	psnClientSecret = "ucPjka5tntB2KqsP"
	psnRedirectURI  = "com.scee.psxandroid.scecompcall://redirect"

	psnAuthorizeURL = "https://ca.account.sony.com/api/authz/v3/oauth/authorize"
	psnTokenURL     = "https://ca.account.sony.com/api/authz/v3/oauth/token"
	psnGameListURL  = "https://m.np.playstation.com/api/gamelist/v2/users/me/titles"
)

// PSNAdapter implements PlayStation Network library access via Sony's unofficial OAuth API.
type PSNAdapter struct {
	NPSSO       string
	AccessToken string
	Client      *http.Client
}

// NewPSNAdapter creates a PSNAdapter and immediately exchanges the NPSSO token for an
// OAuth access token. Returns an error if authentication fails.
//
// The NPSSO token can be obtained from the `npsso` cookie on playstation.com after
// logging in via a web browser.
func NewPSNAdapter(npsso string) (*PSNAdapter, error) {
	a := &PSNAdapter{
		NPSSO:  npsso,
		Client: &http.Client{},
	}
	if err := a.authenticate(); err != nil {
		return nil, fmt.Errorf("PSN authentication failed: %w", err)
	}
	return a, nil
}

// authenticate exchanges the NPSSO cookie for an OAuth access token.
func (p *PSNAdapter) authenticate() error {
	code, err := p.fetchAuthCode()
	if err != nil {
		return err
	}
	return p.fetchAccessToken(code)
}

// fetchAuthCode sends the NPSSO cookie to Sony's OAuth authorize endpoint and extracts
// the authorization code from the redirect Location header.
func (p *PSNAdapter) fetchAuthCode() (string, error) {
	params := url.Values{
		"access_type":   {"offline"},
		"client_id":     {psnClientID},
		"redirect_uri":  {psnRedirectURI},
		"response_type": {"code"},
		"scope":         {"psn:mobile.v2.core psn:clientapp"},
	}

	req, err := http.NewRequest("GET", psnAuthorizeURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Cookie", "npsso="+p.NPSSO)

	// Don't follow the redirect — the code is in the Location header, not at the destination.
	noRedirectClient := *p.Client
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PSN auth code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently {
		return "", fmt.Errorf("unexpected PSN auth response status: %s", resp.Status)
	}

	location := resp.Header.Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("failed to parse PSN auth redirect location: %w", err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("no authorization code in PSN auth redirect")
	}
	return code, nil
}

// fetchAccessToken exchanges an authorization code for an OAuth access token.
func (p *PSNAdapter) fetchAccessToken(code string) error {
	body := url.Values{
		"code":         {code},
		"grant_type":   {"authorization_code"},
		"redirect_uri": {psnRedirectURI},
		"token_format": {"jwt"},
	}

	req, err := http.NewRequest("POST", psnTokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	creds := base64.StdEncoding.EncodeToString([]byte(psnClientID + ":" + psnClientSecret))
	req.Header.Set("Authorization", "Basic "+creds)

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch PSN access token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PSN token endpoint returned error: %s", resp.Status)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode PSN token response: %w", err)
	}
	if result.AccessToken == "" {
		return fmt.Errorf("PSN token response missing access_token")
	}

	p.AccessToken = result.AccessToken
	return nil
}

// GetLibrary fetches the authenticated user's PS5 and PS4 game library from the PSN API.
func (p *PSNAdapter) GetLibrary() ([]models.Game, error) {
	if p.AccessToken == "" {
		return nil, fmt.Errorf("PSN access token is not configured")
	}

	params := url.Values{
		"categories": {"ps5_native_game,ps4_game"},
		"limit":      {"200"},
		"offset":     {"0"},
	}

	req, err := http.NewRequest("GET", psnGameListURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create PSN library request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.AccessToken)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PSN library: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PSN API returned error: %s", resp.Status)
	}

	var result struct {
		Titles []struct {
			TitleID      string `json:"titleId"`
			Name         string `json:"name"`
			PlayDuration string `json:"playDuration"`
		} `json:"titles"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode PSN library response: %w", err)
	}

	games := make([]models.Game, 0, len(result.Titles))
	for _, t := range result.Titles {
		games = append(games, models.Game{
			ID:              t.TitleID,
			Name:            t.Name,
			PlaytimeMinutes: parseISO8601Duration(t.PlayDuration),
		})
	}

	return games, nil
}

// parseISO8601Duration converts an ISO 8601 duration string (e.g. "PT5H30M") to minutes.
func parseISO8601Duration(duration string) int {
	if duration == "" {
		return 0
	}
	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:\d+S)?`)
	matches := re.FindStringSubmatch(duration)
	if matches == nil {
		return 0
	}
	hours, _ := strconv.Atoi(matches[1])
	mins, _ := strconv.Atoi(matches[2])
	return hours*60 + mins
}
