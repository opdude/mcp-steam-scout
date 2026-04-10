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
	if apiKey == "" {
		log.Println("Warning: STEAM_API_KEY not set")
	}

	server := mcp.SetupServer(
		adapter.NewSteamAdapter(apiKey),
		scraper.NewTrendingScraper(),
	)

	log.Println("Server setup, running...")
	if err := server.Run(context.Background(), &mcp_sdk.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
