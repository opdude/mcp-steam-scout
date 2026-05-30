package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/opdude/mcp-steam-scout/pkg/models"
)

const (
	// epicClientID and epicClientSecret are the public credentials for the
	// Epic Games Store launcher OAuth client, used by the community for API access.
	epicClientID     = "34a02cf8f4414e29b15921876da36f9a"
	epicClientSecret = "daafbccc737745039dffe53d94fc76cf"
	epicTokenURL     = "https://account-public-service-prod03.ol.epicgames.com/account/api/oauth/token"
	epicLibraryURL   = "https://library-service.live.use1a.on.epicgames.com/library/api/public/items"
)

type epicRecord struct {
	AppName       string
	CatalogItemID string
	Namespace     string
}

// EpicAdapter implements Epic Games Store library access via the launcher's internal API.
type EpicAdapter struct {
	refreshToken string
	accessToken  string
	accountID    string
	displayName  string
	client       *http.Client

	mu          sync.Mutex
	cacheGames  []models.Game
	cacheExpiry time.Time
}

// NewEpicAdapter creates an EpicAdapter and exchanges the refresh token for an access token.
func NewEpicAdapter(refreshToken string) (*EpicAdapter, error) {
	a := &EpicAdapter{
		refreshToken: refreshToken,
		client:       &http.Client{},
	}
	if err := a.refreshAccessToken(); err != nil {
		return nil, fmt.Errorf("epic authentication failed: %w", err)
	}
	return a, nil
}

type epicTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	ExpiresAt        string `json:"expires_at"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpires   int    `json:"refresh_expires"`
	RefreshExpiresAt string `json:"refresh_expires_at"`
	AccountID        string `json:"account_id"`
	DisplayName      string `json:"displayName"`
	Error            string `json:"error"`
	ErrorCode        string `json:"errorCode"`
	ErrorMessage     string `json:"errorMessage"`
}

// refreshAccessToken exchanges the refresh token for a new access token.
func (e *EpicAdapter) refreshAccessToken() error {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {e.refreshToken},
		"token_type":    {"eg1"},
	}

	req, err := http.NewRequest("POST", epicTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(epicClientID, epicClientSecret)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}

	var tokenResp epicTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return fmt.Errorf("epic token error: %s (%s)", tokenResp.Error, tokenResp.ErrorMessage)
	}
	if tokenResp.AccessToken == "" {
		return fmt.Errorf("epic token response missing access_token")
	}

	e.accessToken = tokenResp.AccessToken
	e.accountID = tokenResp.AccountID
	e.displayName = tokenResp.DisplayName
	return nil
}

type epicLibraryResponse struct {
	Records []struct {
		AppName       string `json:"appName"`
		CatalogItemID string `json:"catalogItemId"`
		Namespace     string `json:"namespace"`
		ProductID     string `json:"productId"`
		SandboxName   string `json:"sandboxName"`
	} `json:"records"`
	ResponseMetadata *struct {
		NextCursor string `json:"nextCursor"`
	} `json:"responseMetadata"`
}

// GetLibrary fetches the user's Epic Games Store library.
// Results are cached for 5 minutes.
func (e *EpicAdapter) GetLibrary() ([]models.Game, error) {
	if e.accessToken == "" {
		return nil, fmt.Errorf("access token not set; use NewEpicAdapter")
	}
	if e.client == nil {
		e.client = &http.Client{}
	}

	e.mu.Lock()
	if e.cacheGames != nil && time.Now().Before(e.cacheExpiry) {
		games := e.cacheGames
		e.mu.Unlock()
		return games, nil
	}
	e.mu.Unlock()

	var rawRecords []epicRecord
	seen := make(map[string]bool)
	cursor := ""

	for {
		u := epicLibraryURL + "?includeMetadata=true"
		if cursor != "" {
			u += "&cursor=" + url.QueryEscape(cursor)
		}

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create library request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+e.accessToken)

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("library request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read library response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusUnauthorized {
				if err := e.refreshAccessToken(); err != nil {
					return nil, fmt.Errorf("epic re-auth failed: %w", err)
				}
				continue
			}
			return nil, fmt.Errorf("epic library API returned %s: %s", resp.Status, string(body))
		}

		var libResp epicLibraryResponse
		if err := json.Unmarshal(body, &libResp); err != nil {
			return nil, fmt.Errorf("failed to decode library response: %w", err)
		}

		for _, r := range libResp.Records {
			id := r.Namespace + ":" + r.CatalogItemID
			if seen[id] {
				continue
			}
			seen[id] = true
			rawRecords = append(rawRecords, epicRecord{
				AppName:       r.AppName,
				CatalogItemID: r.CatalogItemID,
				Namespace:     r.Namespace,
			})
		}

		if libResp.ResponseMetadata == nil || libResp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = libResp.ResponseMetadata.NextCursor
	}

	// Resolve display names from the catalog API
	nameMap := e.resolveCatalogNames(rawRecords)

	games := make([]models.Game, 0, len(rawRecords))
	for _, r := range rawRecords {
		displayName := r.AppName
		if n, ok := nameMap[r.Namespace+":"+r.CatalogItemID]; ok && n != "" {
			displayName = n
		}
		games = append(games, models.Game{
			ID:       r.Namespace + ":" + r.CatalogItemID,
			Name:     displayName,
			Platform: "epic",
		})
	}

	e.mu.Lock()
	e.cacheGames = games
	e.cacheExpiry = time.Now().Add(5 * time.Minute)
	e.mu.Unlock()

	return games, nil
}

type catalogItemInfo struct {
	Title string `json:"title"`
}

// resolveCatalogNames fetches display titles from the catalog API for the given records.
func (e *EpicAdapter) resolveCatalogNames(records []epicRecord) map[string]string {
	result := make(map[string]string)

	type nsKey struct {
		namespace string
		itemID    string
	}
	seen := make(map[nsKey]bool)

	// Group unique item IDs by namespace
	nsItems := make(map[string][]string)
	for _, r := range records {
		key := nsKey{r.Namespace, r.CatalogItemID}
		if seen[key] {
			continue
		}
		seen[key] = true
		nsItems[r.Namespace] = append(nsItems[r.Namespace], r.CatalogItemID)
	}

	for namespace, itemIDs := range nsItems {
		u := "https://catalog-public-service-prod06.ol.epicgames.com/catalog/api/shared/namespace/" +
			url.PathEscape(namespace) + "/bulk/items?includeDLCDetails=false&country=us&locale=en"
		for _, id := range itemIDs {
			u += "&id=" + url.QueryEscape(id)
		}

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+e.accessToken)

		resp, err := e.client.Do(req)
		if err != nil {
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		var data map[string]catalogItemInfo
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			continue
		}

		for _, id := range itemIDs {
			if info, ok := data[id]; ok && info.Title != "" {
				result[namespace+":"+id] = info.Title
			}
		}
	}

	return result
}
