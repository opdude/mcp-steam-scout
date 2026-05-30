package adapter

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const xboxClientID = "0000000048183522"
const xboxRedirectURI = "https://login.live.com/oauth20_desktop.srf"

var (
	xblServerTimeMu    sync.Mutex
	xblServerTimeDelta time.Duration
)

func xblUpdateServerTime(headers http.Header) {
	date := headers.Get("Date")
	if date == "" {
		return
	}
	t, err := time.Parse(time.RFC1123, date)
	if err != nil || t.IsZero() {
		return
	}
	xblServerTimeMu.Lock()
	xblServerTimeDelta = time.Until(t)
	xblServerTimeMu.Unlock()
}

type xblDeviceToken struct {
	IssueInstant time.Time `json:"IssueInstant"`
	NotAfter     time.Time `json:"NotAfter"`
	Token        string

	proofKey *ecdsa.PrivateKey
}

func (d *xblDeviceToken) Valid() bool {
	return time.Now().Before(d.NotAfter.Add(-time.Minute))
}

func xblWindowsTimestamp(t time.Time) int64 {
	return (t.Unix() + 11644473600) * 10000000
}

func xblPadTo32(b *big.Int) []byte {
	out := make([]byte, 32)
	b.FillBytes(out)
	return out
}

func xblSign(req *http.Request, body []byte, key *ecdsa.PrivateKey) {
	xblServerTimeMu.Lock()
	delta := xblServerTimeDelta
	xblServerTimeMu.Unlock()

	var currentTime int64
	if delta != 0 {
		currentTime = xblWindowsTimestamp(time.Now().Add(delta))
	} else {
		currentTime = xblWindowsTimestamp(time.Now())
	}

	hash := sha256.New()
	buf := bytes.NewBuffer([]byte{0, 0, 0, 1, 0})
	_ = binary.Write(buf, binary.BigEndian, currentTime)
	buf.Write([]byte{0})
	hash.Write(buf.Bytes())

	hash.Write([]byte(req.Method))
	hash.Write([]byte{0})
	path := req.URL.Path
	if rq := req.URL.RawQuery; rq != "" {
		path += "?" + rq
	}
	hash.Write([]byte(path))
	hash.Write([]byte{0})
	hash.Write([]byte(req.Header.Get("Authorization")))
	hash.Write([]byte{0})
	hash.Write(body)
	hash.Write([]byte{0})

	r, s, _ := ecdsa.Sign(rand.Reader, key, hash.Sum(nil))
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	buf = bytes.NewBuffer([]byte{0, 0, 0, 1})
	_ = binary.Write(buf, binary.BigEndian, currentTime)
	sig := append(buf.Bytes(), signature...)
	req.Header.Set("Signature", base64.StdEncoding.EncodeToString(sig))
}

func xblObtainDeviceToken(ctx context.Context, key *ecdsa.PrivateKey) (*xblDeviceToken, error) {
	data, err := json.Marshal(map[string]any{
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
		"Properties": map[string]any{
			"AuthMethod": "ProofOfPossession",
			"Id":         "{" + uuid.New().String() + "}",
			"DeviceType": "Win32",
			"Version":    "10.0.25398.4909",
			"ProofKey": map[string]any{
				"crv": "P-256",
				"alg": "ES256",
				"use": "sig",
				"kty": "EC",
				"x":   base64.RawURLEncoding.EncodeToString(xblPadTo32(key.X)),
				"y":   base64.RawURLEncoding.EncodeToString(xblPadTo32(key.Y)),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal device auth: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://device.auth.xboxlive.com/device/authenticate", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("device auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-xbl-contract-version", "1")
	xblSign(req, data, key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	xblUpdateServerTime(resp.Header)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("device auth: %s", resp.Status)
	}

	tok := &xblDeviceToken{proofKey: key}
	return tok, json.NewDecoder(resp.Body).Decode(tok)
}

type sisuUserToken struct {
	Token         string `json:"Token"`
	DisplayClaims struct {
		XUI []struct {
			UHS string `json:"uhs"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
}

// sisuResponse is the raw SISU response. On 401 (age gate), the body still contains UserToken.
type sisuResponse struct {
	UserToken *sisuUserToken `json:"UserToken"`
}

func xblObtainSISUToken(ctx context.Context, accessToken string, device *xblDeviceToken) (*sisuUserToken, error) {
	pk := device.proofKey.PublicKey
	data, err := json.Marshal(map[string]any{
		"AccessToken":       "t=" + accessToken,
		"AppId":             xboxClientID,
		"DeviceToken":       device.Token,
		"Sandbox":           "RETAIL",
		"UseModernGamertag": true,
		"SiteName":          "user.auth.xboxlive.com",
		"RelyingParty":      "http://xboxlive.com",
		"ProofKey": map[string]any{
			"crv": "P-256",
			"alg": "ES256",
			"use": "sig",
			"kty": "EC",
			"x":   base64.RawURLEncoding.EncodeToString(xblPadTo32(pk.X)),
			"y":   base64.RawURLEncoding.EncodeToString(xblPadTo32(pk.Y)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal sisu auth: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://sisu.xboxlive.com/authorize", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("sisu auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-xbl-contract-version", "1")
	xblSign(req, data, device.proofKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sisu auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	xblUpdateServerTime(resp.Header)

	respBody, _ := io.ReadAll(resp.Body)

	var sr sisuResponse
	if err := json.Unmarshal(respBody, &sr); err != nil {
		return nil, fmt.Errorf("parse sisu response: %w", err)
	}
	if sr.UserToken == nil {
		ec := resp.Header.Get("x-err")
		return nil, fmt.Errorf("sisu: HTTP %d x-err=%s body=%s", resp.StatusCode, ec, string(respBody))
	}
	return sr.UserToken, nil
}

type xblXSTSResponse struct {
	Token         string `json:"Token"`
	DisplayClaims struct {
		Xui []struct {
			UHS string `json:"uhs"`
			XID string `json:"xid"`
			GTG string `json:"gtg"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
}

func xblDoXSTS(ctx context.Context, userToken string, signKey *ecdsa.PrivateKey) (*xblXSTSResponse, error) {
	body, err := json.Marshal(map[string]any{
		"RelyingParty": "http://xboxlive.com",
		"TokenType":    "JWT",
		"Properties": map[string]any{
			"UserTokens": []string{userToken},
			"SandboxId":  "RETAIL",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal xsts: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://xsts.auth.xboxlive.com/xsts/authorize", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("xsts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-xbl-contract-version", "1")
	xblSign(req, body, signKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xsts auth: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("xsts auth: HTTP %d body=%s", resp.StatusCode, string(respBody))
	}

	tok := new(xblXSTSResponse)
	return tok, json.NewDecoder(resp.Body).Decode(tok)
}

func xboxLiveAuthURL() string {
	return fmt.Sprintf(
		"https://login.live.com/oauth20_authorize.srf?client_id=%s&response_type=code&approval_prompt=auto&scope=%s&redirect_uri=%s",
		xboxClientID,
		url.QueryEscape("service::user.auth.xboxlive.com::MBI_SSL"),
		url.QueryEscape(xboxRedirectURI),
	)
}

func xboxExchangeCode(code string) (*oauth2.Token, error) {
	data := url.Values{
		"client_id":    {xboxClientID},
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"scope":        {"service::user.auth.xboxlive.com::MBI_SSL"},
		"redirect_uri": {xboxRedirectURI},
	}

	resp, err := http.PostForm(microsoft.LiveConnectEndpoint.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("code exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("code exchange: %s", resp.Status)
	}

	var poll struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&poll); err != nil {
		return nil, fmt.Errorf("code exchange decode: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  poll.AccessToken,
		RefreshToken: poll.RefreshToken,
		TokenType:    poll.TokenType,
		Expiry:       time.Now().Add(time.Duration(poll.ExpiresIn) * time.Second),
	}, nil
}

func xboxRefreshToken(refreshToken string) (*oauth2.Token, error) {
	data := url.Values{
		"client_id":     {xboxClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {"service::user.auth.xboxlive.com::MBI_SSL"},
		"redirect_uri":  {xboxRedirectURI},
	}

	resp, err := http.PostForm(microsoft.LiveConnectEndpoint.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	xblUpdateServerTime(resp.Header)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token refresh: %s", resp.Status)
	}

	var poll struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&poll); err != nil {
		return nil, fmt.Errorf("refresh decode: %w", err)
	}

	rt := poll.RefreshToken
	if rt == "" {
		rt = refreshToken
	}

	return &oauth2.Token{
		AccessToken:  poll.AccessToken,
		RefreshToken: rt,
		TokenType:    poll.TokenType,
		Expiry:       time.Now().Add(time.Duration(poll.ExpiresIn) * time.Second),
	}, nil
}
