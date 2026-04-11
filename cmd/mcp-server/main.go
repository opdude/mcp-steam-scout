package main

import (
	"context"
	"log"
	"os"

	mcp_sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/opdude/mcp-steam-scout/internal/adapter"
	"github.com/opdude/mcp-steam-scout/internal/mcp"
	"github.com/opdude/mcp-steam-scout/internal/scraper"
)

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting server...")

	apiKey := os.Getenv("STEAM_API_KEY")
	steamID := os.Getenv("STEAM_ID")
	username := os.Getenv("STEAM_USERNAME")

	if apiKey == "" {
		log.Fatal("STEAM_API_KEY must be set")
	}
	if steamID == "" && username == "" {
		log.Fatal("Either STEAM_ID or STEAM_USERNAME must be set")
	}

	steamAdapter := adapter.NewSteamAdapter(apiKey, steamID)
	if username != "" && steamID == "" {
		id, err := steamAdapter.ResolveVanityURL(username)
		if err != nil {
			log.Printf("Warning: failed to resolve username: %v", err)
		} else {
			steamAdapter.DefaultSteamID = id
		}
	}

	cfg := mcp.ServerConfig{
		Steam:        steamAdapter,
		SteamScraper: scraper.NewTrendingScraper(),
	}

	// PSN support is optional. When PSN_NPSSO is set, the PSN adapter authenticates
	// at startup and the get_psn_library and get_psn_trending tools are registered.
	if npsso := os.Getenv("PSN_NPSSO"); npsso != "" {
		psnAdapter, err := adapter.NewPSNAdapter(npsso)
		if err != nil {
			log.Printf("Warning: failed to initialize PSN adapter: %v", err)
		} else {
			cfg.PSN = psnAdapter
			log.Println("PSN adapter initialized")
		}
	}

	server := mcp.SetupServer(cfg)

	log.Println("Server setup, running...")
	if err := server.Run(context.Background(), &mcp_sdk.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
