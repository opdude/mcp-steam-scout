package mcp

import (
	"context"
	"log"

	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/opdude/mcp-steam-scout/internal/adapter"
	"github.com/opdude/mcp-steam-scout/internal/recommender"
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

type XboxLibraryInput struct{}
type XboxLibraryOutput struct {
	Games []models.Game `json:"games"`
}

type XboxAuthURLInput struct{}
type XboxAuthURLOutput struct {
	SessionID string `json:"sessionID"`
	URL       string `json:"url"`
	UserCode  string `json:"userCode"`
}

type XboxAuthPollInput struct {
	SessionID string `json:"sessionID"`
}
type XboxAuthPollOutput struct {
	RefreshToken string `json:"refreshToken,omitempty"`
	Done         bool   `json:"done"`
}

type EpicLibraryInput struct{}
type EpicLibraryOutput struct {
	Games []models.Game `json:"games"`
}

type GOGLibraryInput struct{}
type GOGLibraryOutput struct {
	Games []models.Game `json:"games"`
}

type RecommendGameInput struct{}
type RecommendGameOutput struct {
	recommender.Recommendation
}

// ServerConfig holds the adapters and scrapers to register as MCP tools.
// PSN, Xbox, Epic, and GOG are optional — set to nil to disable their tools.
type ServerConfig struct {
	Steam        *adapter.SteamAdapter
	SteamScraper *scraper.TrendingScraper
	PSN          *adapter.PSNAdapter
	Xbox         *adapter.XboxAdapter
	Epic         *adapter.EpicAdapter
	GOG          *adapter.GOGAdapter
	Recommender  *recommender.Recommender
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
			Description: "Get currently trending games from the Steam store, GOG, and Epic Games Store. The playtimeMinutes field in each game represents playtime in minutes, not hours.",
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

	// Xbox setup tools — always registered, no token needed.
	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "get_xbox_auth_url",
			Description: "Start the Xbox device code auth flow. Returns a URL and user code. The user must visit the URL and enter the code. Then call complete_xbox_auth with the sessionID.",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input XboxAuthURLInput) (*mcp_sdk.CallToolResult, XboxAuthURLOutput, error) {
			sessionID, url, userCode, err := adapter.StartXboxDeviceAuth()
			if err != nil {
				return &mcp_sdk.CallToolResult{
					Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
					IsError: true,
				}, XboxAuthURLOutput{}, nil
			}
			return nil, XboxAuthURLOutput{SessionID: sessionID, URL: url, UserCode: userCode}, nil
		},
	)

	mcp_sdk.AddTool(
		server,
		&mcp_sdk.Tool{
			Name:        "complete_xbox_auth",
			Description: "Poll for completion of an Xbox device code auth flow. Call this after the user has authenticated at the URL. If done is false, the user hasn't finished yet. If done is true, the refreshToken is returned.",
		},
		func(ctx context.Context, req *mcp_sdk.CallToolRequest, input XboxAuthPollInput) (*mcp_sdk.CallToolResult, XboxAuthPollOutput, error) {
			token, done, err := adapter.PollXboxDeviceAuth(input.SessionID)
			if err != nil {
				return &mcp_sdk.CallToolResult{
					Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
					IsError: true,
				}, XboxAuthPollOutput{}, nil
			}
			if done {
				return nil, XboxAuthPollOutput{RefreshToken: token, Done: true}, nil
			}
			return nil, XboxAuthPollOutput{Done: false}, nil
		},
	)

	// Xbox tools — registered only when an Xbox adapter is provided.
	if cfg.Xbox != nil {
		mcp_sdk.AddTool(
			server,
			&mcp_sdk.Tool{
				Name:        "get_xbox_library",
				Description: "Get games from your Xbox library. The playtimeMinutes field in each game represents playtime in minutes, not hours.",
			},
			func(ctx context.Context, req *mcp_sdk.CallToolRequest, input XboxLibraryInput) (*mcp_sdk.CallToolResult, XboxLibraryOutput, error) {
				games, err := cfg.Xbox.GetLibrary(ctx)
				if err != nil {
					return &mcp_sdk.CallToolResult{
						Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
						IsError: true,
					}, XboxLibraryOutput{}, nil
				}
				return nil, XboxLibraryOutput{Games: games}, nil
			},
		)
	}

	// Epic tools — registered only when an Epic adapter is provided.
	if cfg.Epic != nil {
		mcp_sdk.AddTool(
			server,
			&mcp_sdk.Tool{
				Name:        "get_epic_library",
				Description: "Get games from your Epic Games Store library. Note: playtime data is not available from Epic's API.",
			},
			func(ctx context.Context, req *mcp_sdk.CallToolRequest, input EpicLibraryInput) (*mcp_sdk.CallToolResult, EpicLibraryOutput, error) {
				games, err := cfg.Epic.GetLibrary()
				if err != nil {
					return &mcp_sdk.CallToolResult{
						Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
						IsError: true,
					}, EpicLibraryOutput{}, nil
				}
				return nil, EpicLibraryOutput{Games: games}, nil
			},
		)
	}

	// GOG tools — registered only when a GOG adapter is provided.
	if cfg.GOG != nil {
		mcp_sdk.AddTool(
			server,
			&mcp_sdk.Tool{
				Name:        "get_gog_library",
				Description: "Get games from your GOG library. The playtimeMinutes field in each game represents playtime in minutes, not hours.",
			},
			func(ctx context.Context, req *mcp_sdk.CallToolRequest, input GOGLibraryInput) (*mcp_sdk.CallToolResult, GOGLibraryOutput, error) {
				games, err := cfg.GOG.GetLibrary()
				if err != nil {
					return &mcp_sdk.CallToolResult{
						Content: []mcp_sdk.Content{&mcp_sdk.TextContent{Text: "Error: " + err.Error()}},
						IsError: true,
					}, GOGLibraryOutput{}, nil
				}
				return nil, GOGLibraryOutput{Games: games}, nil
			},
		)
	}

	// recommend_game tool — registered when a recommender is configured.
	if cfg.Recommender != nil {
		mcp_sdk.AddTool(
			server,
			&mcp_sdk.Tool{
				Name:        "recommend_game",
				Description: "Get personalized game recommendations by analyzing your libraries across all configured platforms (Steam, PSN, Xbox, Epic) and current trending games. Returns unplayed gems, dabbled games, top played, trending overlap, and trending purchase candidates.",
			},
			func(ctx context.Context, req *mcp_sdk.CallToolRequest, input RecommendGameInput) (*mcp_sdk.CallToolResult, RecommendGameOutput, error) {
				var steamGames, psnGames, xboxGames, epicGames, gogGames []models.Game
				var err error

				steamGames, err = cfg.Steam.GetLibrary()
				if err != nil {
					log.Printf("recommend_game: steam library error: %v", err)
				}

				if cfg.PSN != nil {
					psnGames, err = cfg.PSN.GetLibrary()
					if err != nil {
						log.Printf("recommend_game: psn library error: %v", err)
					}
				}

				if cfg.Xbox != nil {
					xboxGames, err = cfg.Xbox.GetLibrary(ctx)
					if err != nil {
						log.Printf("recommend_game: xbox library error: %v", err)
					}
				}

				if cfg.Epic != nil {
					epicGames, err = cfg.Epic.GetLibrary()
					if err != nil {
						log.Printf("recommend_game: epic library error: %v", err)
					}
				}

				if cfg.GOG != nil {
					gogGames, err = cfg.GOG.GetLibrary()
					if err != nil {
						log.Printf("recommend_game: gog library error: %v", err)
					}
				}

				trending, err := cfg.SteamScraper.GetTrendingGames()
				if err != nil {
					log.Printf("recommend_game: trending error: %v", err)
				}

				rec := cfg.Recommender.Recommend(steamGames, psnGames, xboxGames, epicGames, gogGames, trending)
				return nil, RecommendGameOutput{rec}, nil
			},
		)
	}

	return server
}
