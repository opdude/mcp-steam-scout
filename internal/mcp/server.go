package mcp

import (
	"context"

	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/opdude/mcp-steam-scout/internal/adapter"
	"github.com/opdude/mcp-steam-scout/internal/scraper"
	"github.com/opdude/mcp-steam-scout/pkg/models"
)

type TrendingInput struct{}
type TrendingOutput struct {
	Games []models.Game `json:"games"`
}

type LibraryInput struct{}
type LibraryOutput struct {
	Games []models.Game `json:"games"`
}

type ResolveVanityInput struct {
	VanityURL string `json:"vanityURL" jsonschema:"the steam vanity username or custom URL"`
}
type ResolveVanityOutput struct {
	SteamID string `json:"steamID"`
}

type PSNLibraryInput struct{}
type PSNLibraryOutput struct {
	Games []models.Game `json:"games"`
}

// ServerConfig holds the adapters and scrapers to register as MCP tools.
// PSN is optional — set to nil to disable PSN tools.
type ServerConfig struct {
	Steam        *adapter.SteamAdapter
	SteamScraper *scraper.TrendingScraper
	PSN          *adapter.PSNAdapter
}

// SetupServer initializes the MCP server with tools based on the provided config.
func SetupServer(cfg ServerConfig) *mcp_sdk.Server {
	server := mcp_sdk.NewServer(
		&mcp_sdk.Implementation{Name: "mcp-steam-scout", Version: "1.0.0"},
		nil,
	)

	// Steam tools — always registered.
	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "get_trending",
			Description: "Get currently trending games from the Steam store. The playtimeMinutes field in each game represents playtime in minutes, not hours.",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input TrendingInput) (*mcp_sdk.CallToolResult, TrendingOutput, error) {
			games, err := cfg.SteamScraper.GetTrendingGames()
			return nil, TrendingOutput{Games: games}, err
		},
	)

	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "resolve_steam_id",
			Description: "Resolve a Steam vanity username to a numeric Steam ID",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input ResolveVanityInput) (*mcp_sdk.CallToolResult, ResolveVanityOutput, error) {
			steamID, err := cfg.Steam.ResolveVanityURL(input.VanityURL)
			if err != nil {
				return &mcp_sdk.CallToolResult{
					Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
					IsError: true,
				}, ResolveVanityOutput{}, nil
			}
			return nil, ResolveVanityOutput{SteamID: steamID}, nil
		},
	)

	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "get_library",
			Description: "Get games from your Steam library. The playtimeMinutes field in each game represents playtime in minutes, not hours.",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input LibraryInput) (*mcp_sdk.CallToolResult, LibraryOutput, error) {
			games, err := cfg.Steam.GetLibrary()
			if err != nil {
				return &mcp_sdk.CallToolResult{
					Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
					IsError: true,
				}, LibraryOutput{}, nil
			}
			return nil, LibraryOutput{Games: games}, nil
		},
	)

	// PSN tools — registered only when a PSN adapter is provided.
	if cfg.PSN != nil {
		mcp_sdk.AddTool(
			server,
			&mcp_sdk.Tool{
				Name:        "get_psn_library",
				Description: "Get games from your PlayStation library. The playtimeMinutes field in each game represents playtime in minutes, not hours.",
			},
			func(ctx context.Context, req *mcp_sdk.CallToolRequest, input PSNLibraryInput) (*mcp_sdk.CallToolResult, PSNLibraryOutput, error) {
				games, err := cfg.PSN.GetLibrary()
				if err != nil {
					return &mcp_sdk.CallToolResult{
						Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
						IsError: true,
					}, PSNLibraryOutput{}, nil
				}
				return nil, PSNLibraryOutput{Games: games}, nil
			},
		)

	}

	return server
}
