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

	adapter := adapter.NewSteamAdapter(apiKey, steamID)
	if username != "" && steamID == "" {
		id, err := adapter.ResolveVanityURL(username)
		if err != nil {
			log.Printf("Warning: failed to resolve username: %v", err)
		} else {
			adapter.DefaultSteamID = id
		}
	}

	server := mcp.SetupServer(
		adapter,
		scraper.NewTrendingScraper(),
	)

	log.Println("Server setup, running...")
	if err := server.Run(context.Background(), &mcp_sdk.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
