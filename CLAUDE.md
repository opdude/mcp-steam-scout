# Game Recommender MCP Server

This is a Go-based MCP (Model Context Protocol) server that provides tools to an LLM to recommend games based on your Steam library and current gaming trends.

## Development

### Prerequisites
- Go 1.25+
- Task (managed as a Go tool — no separate installation needed)

### Running the Server
Build the binary first with `go tool task build`, then run `./bin/mcp-server`.

### Testing
Run all tests using `task test`.

### Building
Build the local binary using `task build`.

## Architecture
- `cmd/mcp-server`: The entry point for the MCP server.
- `internal/adapter`: Implementations for different gaming platforms (e.g., Steam).
- `internal/scraper`: Logic for scraping trending game data.
- `internal/mcp`: MCP tool definitions and protocol handling.
- `pkg/models`: Shared data structures.

## Deployment
We use [GoReleaser](https://goreleaser.com/) to build and distribute binaries for various platforms.
