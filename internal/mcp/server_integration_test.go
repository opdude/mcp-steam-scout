//go:build integration

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/opdude/mcp-steam-scout/internal/adapter"
	"github.com/opdude/mcp-steam-scout/internal/scraper"
)

func loadEnv(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		_ = godotenv.Load(filepath.Join(dir, ".env"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

func connectToServer(t *testing.T, ctx context.Context, server *mcp_sdk.Server) *mcp_sdk.ClientSession {
	t.Helper()
	t1, t2 := mcp_sdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp_sdk.NewClient(
		&mcp_sdk.Implementation{Name: "test", Version: "0.0.0"}, nil,
	)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	return session
}

func callTool(t *testing.T, ctx context.Context, session *mcp_sdk.ClientSession, name string, args map[string]any) *mcp_sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp_sdk.CallToolParams{
		Name: name, Arguments: args,
	})
	if err != nil {
		t.Fatalf("tool %q: %v", name, err)
	}
	if result.IsError {
		text := ""
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(*mcp_sdk.TextContent); ok {
				text = tc.Text
			}
		}
		t.Fatalf("tool %q returned error: %s", name, text)
	}
	return result
}

func extractText(t *testing.T, result *mcp_sdk.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("no content in result")
	}
	tc, ok := result.Content[0].(*mcp_sdk.TextContent)
	if !ok {
		t.Fatal("content is not TextContent")
	}
	return tc.Text
}

func newCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestIntegration_Trending(t *testing.T) {
	loadEnv(t)
	ctx := newCtx(t)

	cfg := ServerConfig{SteamScraper: scraper.NewTrendingScraper()}
	srv := SetupServer(cfg)
	session := connectToServer(t, ctx, srv)
	defer session.Close()

	result := callTool(t, ctx, session, "get_trending", nil)
	var out TrendingOutput
	if err := json.Unmarshal([]byte(extractText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Games) == 0 {
		t.Fatal("expected at least 1 trending game")
	}
	t.Logf("trending: %d games, first: %s", len(out.Games), out.Games[0].Name)
}

func TestIntegration_SteamLibrary(t *testing.T) {
	loadEnv(t)
	apiKey := os.Getenv("STEAM_API_KEY")
	steamID := os.Getenv("STEAM_ID")
	if apiKey == "" || steamID == "" {
		t.Skip("STEAM_API_KEY and STEAM_ID must be set")
	}

	ctx := newCtx(t)
	steamAdapter := adapter.NewSteamAdapter(apiKey, steamID)
	cfg := ServerConfig{
		Steam:        steamAdapter,
		SteamScraper: scraper.NewTrendingScraper(),
	}
	srv := SetupServer(cfg)
	session := connectToServer(t, ctx, srv)
	defer session.Close()

	result := callTool(t, ctx, session, "get_library", nil)
	var out LibraryOutput
	if err := json.Unmarshal([]byte(extractText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Games) == 0 {
		t.Fatal("expected at least 1 game in steam library")
	}
	t.Logf("steam library: %d games, first: %s", len(out.Games), out.Games[0].Name)
}

func TestIntegration_ResolveSteamID(t *testing.T) {
	loadEnv(t)
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		t.Skip("STEAM_API_KEY must be set")
	}

	ctx := newCtx(t)
	steamAdapter := adapter.NewSteamAdapter(apiKey, "")
	cfg := ServerConfig{
		Steam:        steamAdapter,
		SteamScraper: scraper.NewTrendingScraper(),
	}
	srv := SetupServer(cfg)
	session := connectToServer(t, ctx, srv)
	defer session.Close()

	result := callTool(t, ctx, session, "resolve_steam_id", map[string]any{"vanityURL": "opdude"})
	var out ResolveVanityOutput
	if err := json.Unmarshal([]byte(extractText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.SteamID == "" {
		t.Fatal("expected non-empty steam ID")
	}
	t.Logf("resolved steam ID: %s", out.SteamID)
}

func TestIntegration_PSNLibrary(t *testing.T) {
	loadEnv(t)
	npsso := os.Getenv("PSN_NPSSO")
	if npsso == "" {
		t.Skip("PSN_NPSSO must be set")
	}

	ctx := newCtx(t)
	psnAdapter, err := adapter.NewPSNAdapter(npsso)
	if err != nil {
		t.Fatalf("create PSN adapter: %v", err)
	}

	cfg := ServerConfig{
		Steam:        adapter.NewSteamAdapter("", ""),
		SteamScraper: scraper.NewTrendingScraper(),
		PSN:          psnAdapter,
	}
	srv := SetupServer(cfg)
	session := connectToServer(t, ctx, srv)
	defer session.Close()

	result := callTool(t, ctx, session, "get_psn_library", nil)
	var out PSNLibraryOutput
	if err := json.Unmarshal([]byte(extractText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	t.Logf("psn library: %d games", len(out.Games))
	if len(out.Games) > 0 {
		t.Logf("first: %s", out.Games[0].Name)
	}
}

func TestIntegration_XboxLibrary(t *testing.T) {
	loadEnv(t)
	refreshToken := os.Getenv("XBOX_REFRESH_TOKEN")
	if refreshToken == "" {
		t.Skip("XBOX_REFRESH_TOKEN must be set")
	}

	ctx := newCtx(t)
	xboxAdapter, err := adapter.NewXboxAdapter(refreshToken)
	if err != nil {
		t.Fatalf("create Xbox adapter: %v", err)
	}

	cfg := ServerConfig{
		Steam:        adapter.NewSteamAdapter("", ""),
		SteamScraper: scraper.NewTrendingScraper(),
		Xbox:         xboxAdapter,
	}
	srv := SetupServer(cfg)
	session := connectToServer(t, ctx, srv)
	defer session.Close()

	result := callTool(t, ctx, session, "get_xbox_library", nil)
	var out XboxLibraryOutput
	if err := json.Unmarshal([]byte(extractText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	t.Logf("xbox library: %d games", len(out.Games))
	if len(out.Games) > 0 {
		t.Logf("first: %s", out.Games[0].Name)
	}
}

func TestIntegration_GOGLibrary(t *testing.T) {
	loadEnv(t)
	refreshToken := os.Getenv("GOG_REFRESH_TOKEN")
	if refreshToken == "" {
		t.Skip("GOG_REFRESH_TOKEN must be set")
	}

	ctx := newCtx(t)
	gogAdapter, err := adapter.NewGOGAdapter(refreshToken, os.Getenv("GOG_COOKIE"))
	if err != nil {
		t.Fatalf("create GOG adapter: %v", err)
	}

	cfg := ServerConfig{
		Steam:        adapter.NewSteamAdapter("", ""),
		SteamScraper: scraper.NewTrendingScraper(),
		GOG:          gogAdapter,
	}
	srv := SetupServer(cfg)
	session := connectToServer(t, ctx, srv)
	defer session.Close()

	result := callTool(t, ctx, session, "get_gog_library", nil)
	var out GOGLibraryOutput
	if err := json.Unmarshal([]byte(extractText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	t.Logf("gog library: %d games", len(out.Games))
	if len(out.Games) > 0 {
		t.Logf("first: %s", out.Games[0].Name)
	}
}

func TestIntegration_EpicLibrary(t *testing.T) {
	loadEnv(t)
	refreshToken := os.Getenv("EPIC_REFRESH_TOKEN")
	if refreshToken == "" {
		t.Skip("EPIC_REFRESH_TOKEN must be set")
	}

	ctx := newCtx(t)
	epicAdapter, err := adapter.NewEpicAdapter(refreshToken)
	if err != nil {
		t.Fatalf("create Epic adapter: %v", err)
	}

	cfg := ServerConfig{
		Steam:        adapter.NewSteamAdapter("", ""),
		SteamScraper: scraper.NewTrendingScraper(),
		Epic:         epicAdapter,
	}
	srv := SetupServer(cfg)
	session := connectToServer(t, ctx, srv)
	defer session.Close()

	result := callTool(t, ctx, session, "get_epic_library", nil)
	var out EpicLibraryOutput
	if err := json.Unmarshal([]byte(extractText(t, result)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	t.Logf("epic library: %d games", len(out.Games))
	if len(out.Games) > 0 {
		t.Logf("first: %s", out.Games[0].Name)
	}
}
