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

type LibraryInput struct {
	UserID string `json:"userID" jsonschema:"the steam user id"`
}
type LibraryOutput struct {
	Games []models.Game `json:"games"`
}

type ResolveVanityInput struct {
	VanityURL string `json:"vanityURL" jsonschema:"the steam vanity username or custom URL"`
}
type ResolveVanityOutput struct {
	SteamID string `json:"steamID"`
}

// SetupServer initializes the MCP server with tools.
func SetupServer(steam *adapter.SteamAdapter, scraper *scraper.TrendingScraper) *mcp_sdk.Server {
	server := mcp_sdk.NewServer(
		&mcp_sdk.Implementation{Name: "mcp-steam-scout", Version: "1.0.0"},
		nil,
	)

	// Register get_trending tool
	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "get_trending",
			Description: "Get currently trending games",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input TrendingInput) (*mcp_sdk.CallToolResult, TrendingOutput, error) {
			games, err := scraper.GetTrendingGames()
			return nil, TrendingOutput{Games: games}, err
		},
	)

	// Register resolve_steam_id tool
	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "resolve_steam_id",
			Description: "Resolve a Steam vanity username to a numeric Steam ID",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input ResolveVanityInput) (*mcp_sdk.CallToolResult, ResolveVanityOutput, error) {
			steamID, err := steam.ResolveVanityURL(input.VanityURL)
			if err != nil {
				return &mcp_sdk.CallToolResult{
					Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
					IsError: true,
				}, ResolveVanityOutput{}, nil
			}
			return nil, ResolveVanityOutput{SteamID: steamID}, nil
		},
	)

	// Register get_library tool
	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "get_library",
			Description: "Get games from your library",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input LibraryInput) (*mcp_sdk.CallToolResult, LibraryOutput, error) {
			userID := input.UserID
			if userID == "" {
				userID = steam.DefaultSteamID
			}
			games, err := steam.GetLibrary(userID)
			if err != nil {
				return &mcp_sdk.CallToolResult{
					Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
					IsError: true,
				}, LibraryOutput{}, nil
			}
			return nil, LibraryOutput{Games: games}, nil

		},
	)

	return server
}
