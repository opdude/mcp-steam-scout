package adapter

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

type deviceAuthSession struct {
	DeviceCode string
	UserCode   string
	URL        string
}

var (
	authSessions   = map[string]*deviceAuthSession{}
	authSessionsMu sync.Mutex
)

// StartXboxDeviceAuth begins the Microsoft device code flow and returns
// (sessionID, url, userCode). The sessionID is used with PollXboxDeviceAuth.
func StartXboxDeviceAuth() (sessionID, authURL, userCode string, err error) {
	resp, err := http.PostForm("https://login.live.com/oauth20_connect.srf", url.Values{
		"client_id":     {xboxClientID},
		"scope":         {"service::user.auth.xboxlive.com::MBI_SSL"},
		"response_type": {"device_code"},
	})
	if err != nil {
		return "", "", "", fmt.Errorf("start device auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("start device auth: HTTP %d", resp.StatusCode)
	}

	var dc struct {
		UserCode        string `json:"user_code"`
		DeviceCode      string `json:"device_code"`
		VerificationURI string `json:"verification_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return "", "", "", fmt.Errorf("decode device auth: %w", err)
	}

	sess := &deviceAuthSession{
		DeviceCode: dc.DeviceCode,
		UserCode:   dc.UserCode,
		URL:        dc.VerificationURI,
	}

	authSessionsMu.Lock()
	authSessions[dc.DeviceCode] = sess
	authSessionsMu.Unlock()

	return dc.DeviceCode, dc.VerificationURI, dc.UserCode, nil
}

// PollXboxDeviceAuth polls a device auth session for completion.
// Returns (refreshToken, done, error). If still pending, done is false.
func PollXboxDeviceAuth(sessionID string) (refreshToken string, done bool, err error) {
	authSessionsMu.Lock()
	sess, ok := authSessions[sessionID]
	authSessionsMu.Unlock()

	if !ok {
		return "", false, fmt.Errorf("no such device auth session")
	}

	resp, err := http.PostForm("https://login.live.com/oauth20_token.srf", url.Values{
		"client_id":   {xboxClientID},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {sess.DeviceCode},
	})
	if err != nil {
		return "", false, fmt.Errorf("poll device auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var poll struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		RefreshToken     string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&poll); err != nil {
		return "", false, fmt.Errorf("decode poll: %w", err)
	}

	switch poll.Error {
	case "authorization_pending":
		return "", false, nil
	case "slow_down":
		return "", false, nil
	case "expired_token":
		authSessionsMu.Lock()
		delete(authSessions, sessionID)
		authSessionsMu.Unlock()
		return "", false, fmt.Errorf("device auth expired")
	case "authorization_declined":
		authSessionsMu.Lock()
		delete(authSessions, sessionID)
		authSessionsMu.Unlock()
		return "", false, fmt.Errorf("authorization declined")
	case "":
		authSessionsMu.Lock()
		delete(authSessions, sessionID)
		authSessionsMu.Unlock()
		return poll.RefreshToken, true, nil
	default:
		return "", false, fmt.Errorf("device auth error: %s", poll.Error)
	}
}

type XboxAdapter struct {
	refreshToken  string
	oauthToken    *oauthTokenHolder
	signerKey     *ecdsa.PrivateKey
	deviceToken   *xblDeviceToken
	deviceTokenMu sync.Mutex

	mu          sync.Mutex
	cacheGames  []models.Game
	cacheExpiry time.Time
}

type oauthTokenHolder struct {
	mu           sync.Mutex
	accessToken  string
	refreshToken string
	expiry       time.Time
}

func NewXboxAdapter(refreshToken string) (*XboxAdapter, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate signer key: %w", err)
	}
	a := &XboxAdapter{
		refreshToken: refreshToken,
		signerKey:    key,
	}
	if refreshToken != "" {
		a.oauthToken = &oauthTokenHolder{refreshToken: refreshToken}
	}
	return a, nil
}

// XboxAuthURL returns the OAuth URL for getting an Xbox auth code (alternative to device code flow).
func XboxAuthURL() string {
	return xboxLiveAuthURL()
}

func (x *XboxAdapter) SetAuthCode(code string) error {
	tok, err := xboxExchangeCode(code)
	if err != nil {
		return err
	}
	x.oauthToken.mu.Lock()
	x.oauthToken.accessToken = tok.AccessToken
	x.oauthToken.refreshToken = tok.RefreshToken
	x.oauthToken.expiry = tok.Expiry
	x.oauthToken.mu.Unlock()
	return nil
}

func (x *XboxAdapter) ensureAccessToken(ctx context.Context) (string, error) {
	if x.oauthToken == nil {
		return "", fmt.Errorf("oauth token not initialized; call NewXboxAdapter")
	}
	x.oauthToken.mu.Lock()
	tok := x.oauthToken.accessToken
	exp := x.oauthToken.expiry
	refresh := x.oauthToken.refreshToken
	x.oauthToken.mu.Unlock()

	if tok != "" && time.Now().Before(exp.Add(-5*time.Minute)) {
		return tok, nil
	}

	newTok, err := xboxRefreshToken(refresh)
	if err != nil {
		return "", fmt.Errorf("refresh oauth: %w", err)
	}

	x.oauthToken.mu.Lock()
	x.oauthToken.accessToken = newTok.AccessToken
	x.oauthToken.refreshToken = newTok.RefreshToken
	x.oauthToken.expiry = newTok.Expiry
	tok = newTok.AccessToken
	x.oauthToken.mu.Unlock()

	return tok, nil
}

func (x *XboxAdapter) ensureDeviceToken(ctx context.Context) (*xblDeviceToken, error) {
	x.deviceTokenMu.Lock()
	defer x.deviceTokenMu.Unlock()

	if x.deviceToken != nil && x.deviceToken.Valid() {
		return x.deviceToken, nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}

	dt, err := xblObtainDeviceToken(ctx, key)
	if err != nil {
		return nil, err
	}
	x.deviceToken = dt
	x.signerKey = key
	return dt, nil
}

func (x *XboxAdapter) GetLibrary(ctx context.Context) ([]models.Game, error) {
	x.mu.Lock()
	if x.cacheGames != nil && time.Now().Before(x.cacheExpiry) {
		games := x.cacheGames
		x.mu.Unlock()
		return games, nil
	}
	x.mu.Unlock()

	accessToken, err := x.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	device, err := x.ensureDeviceToken(ctx)
	if err != nil {
		return nil, err
	}

	sisu, err := xblObtainSISUToken(ctx, accessToken, device)
	if err != nil {
		return nil, fmt.Errorf("sisu auth: %w", err)
	}
	userToken := sisu.Token

	xsts, err := xblDoXSTS(ctx, userToken, x.signerKey)
	if err != nil {
		return nil, fmt.Errorf("xsts auth: %w", err)
	}

	if len(xsts.DisplayClaims.Xui) == 0 {
		return nil, fmt.Errorf("no user info in xsts response")
	}
	userhash := xsts.DisplayClaims.Xui[0].UHS
	xuid := xsts.DisplayClaims.Xui[0].XID

	games, err := x.fetchTitles(ctx, userhash, xsts.Token, xuid)
	if err != nil {
		return nil, fmt.Errorf("fetch titles: %w", err)
	}

	if err := x.fetchPlaytimeBatch(ctx, userhash, xsts.Token, xuid, games); err != nil {
		// Non-fatal — playtime is a nice-to-have
		_ = err
	}

	x.mu.Lock()
	x.cacheGames = games
	x.cacheExpiry = time.Now().Add(5 * time.Minute)
	x.mu.Unlock()

	return games, nil
}

type xboxTitleResponse struct {
	Titles []struct {
		TitleID      string   `json:"titleId"`
		Name         string   `json:"name"`
		Type         string   `json:"type"`
		Devices      []string `json:"devices"`
		TitleHistory struct {
			LastTimePlayed string `json:"lastTimePlayed"`
		} `json:"titleHistory"`
	} `json:"titles"`
}

func (x *XboxAdapter) fetchTitles(ctx context.Context, userhash, xstsToken, xuid string) ([]models.Game, error) {
	auth := fmt.Sprintf("XBL3.0 x=%s;%s", userhash, xstsToken)
	apiURL := fmt.Sprintf("https://titlehub.xboxlive.com/users/xuid(%s)/titles/titlehistory/decoration/achievement,image", xuid)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("x-xbl-contract-version", "2")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("Accept", "application/json")

	xblSign(req, nil, x.signerKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("titlehub API %d: %s", resp.StatusCode, string(body))
	}

	var result xboxTitleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode titles: %w", err)
	}

	var out []models.Game
	for _, g := range result.Titles {
		if g.Type != "Game" {
			continue
		}
		out = append(out, models.Game{
			ID:   g.TitleID,
			Name: g.Name,
		})
	}

	return out, nil
}

// fetchPlaytimeBatch tries to get MinutesPlayed from the userstats batch API.
// This is best-effort — some games may not return data.
func (x *XboxAdapter) fetchPlaytimeBatch(ctx context.Context, userhash, xstsToken, xuid string, games []models.Game) error {
	if len(games) == 0 {
		return nil
	}

	auth := fmt.Sprintf("XBL3.0 x=%s;%s", userhash, xstsToken)

	gameByID := make(map[string]*models.Game, len(games))
	for i := range games {
		gameByID[games[i].ID] = &games[i]
	}

	const batchSize = 10

	for start := 0; start < len(games); start += batchSize {
		end := start + batchSize
		if end > len(games) {
			end = len(games)
		}
		chunk := games[start:end]

		groups := make([]map[string]any, len(chunk))
		stats := make([]map[string]any, len(chunk))
		for i := range chunk {
			groups[i] = map[string]any{"name": fmt.Sprintf("g%d", start+i), "titleId": chunk[i].ID}
			stats[i] = map[string]any{"name": "MinutesPlayed", "titleId": chunk[i].ID}
		}
		body, _ := json.Marshal(map[string]any{
			"arrangebyfield": "xuid",
			"groups":         groups,
			"stats":          stats,
			"xuids":          []string{xuid},
		})

		req, err := http.NewRequestWithContext(ctx, "POST", "https://userstats.xboxlive.com/batch", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("batch request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", auth)
		req.Header.Set("x-xbl-contract-version", "2")
		xblSign(req, body, x.signerKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("batch http: %w", err)
		}

		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			continue
		}

		var batchResp struct {
			StatsListCollection []struct {
				Stats []struct {
					TitleID string `json:"titleid"`
					Name    string `json:"name"`
					Value   string `json:"value"`
				} `json:"stats"`
			} `json:"statlistscollection"`
		}
		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		if err := json.Unmarshal(respBody, &batchResp); err != nil {
			continue
		}

		for _, slc := range batchResp.StatsListCollection {
			for _, stat := range slc.Stats {
				if stat.Name == "MinutesPlayed" {
					if g, ok := gameByID[stat.TitleID]; ok {
						var v int
						if _, err := fmt.Sscanf(stat.Value, "%d", &v); err == nil {
							g.PlaytimeMinutes = v
						}
					}
				}
			}
		}
	}

	return nil
}
