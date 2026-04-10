# mcp-steam-scout

[![MCP Server](https://img.shields.io/badge/MCP-Server-blue)](https://modelcontextprotocol.io)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants like Claude access to your **Steam library and current gaming trends** to make personalised game recommendations weighted by actual playtime.

Built with the official [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk).

> **Steam API Terms of Service**: This project uses the Steam Web API. You must obtain your own API key from [steamcommunity.com/dev/apikey](https://steamcommunity.com/dev/apikey) and comply with the [Steam Web API Terms of Use](https://store.steampowered.com/developer/steam/api). Your Steam profile must be set to **public** for library lookups to work.

---

## Contents

- [Features](#features)
- [What you can do with it](#what-you-can-do-with-it)
- [Quick start](#quick-start)
- [Client configuration](#client-configuration)
- [Development](#development)

---

## Features

| Tool | What it does |
|------|-------------|
| `resolve_steam_id` | Converts a Steam vanity username to a numeric Steam ID |
| `get_library` | Fetches your owned games including playtime data |
| `get_trending` | Returns currently trending games from the Steam store |

---

## What you can do with it

Ask Claude things like:

> "Look up my Steam ID for username opdude, fetch my library, check what's trending, and recommend me something new to play based on what I've played the most."

> "What are the top trending games on Steam right now? Which ones match my playstyle based on my library?"

> "I mostly play strategy games — are any trending games in that genre worth trying?"

Claude chains the three tools together automatically, cross-references your playtime with current trends, and gives you personalised recommendations.

---

## Quick start

### Prerequisites

- A [Steam Web API key](https://steamcommunity.com/dev/apikey) — free and instant to register
- Node.js 18+ (for the `npx` install method) **or** Go 1.22+ (to build from source)

### Install

**via npx (recommended — no build step):**

```bash
npx @opdude/mcp-steam-scout
```

The binary is downloaded automatically on first run.

**via Go (build from source):**

```bash
git clone https://github.com/opdude/mcp-steam-scout
cd mcp-steam-scout
go tool task build
# binary is written to ./bin/mcp-server
```

### Find your Steam ID

Your Steam ID is a 17-digit number (e.g. `76561197962821445`). You can:

- Ask Claude to run `resolve_steam_id` with your vanity username
- Look it up manually at [steamid.io](https://steamid.io)

---

## Client configuration

### Claude Code / Claude Desktop (npx)

```json
{
  "mcpServers": {
    "mcp-steam-scout": {
      "command": "npx",
      "args": ["-y", "@opdude/mcp-steam-scout"],
      "env": {
        "STEAM_API_KEY": "your_steam_api_key_here",
        "STEAM_ID": "your_optional_steam_id_here"
      }
    }
  }
}
```

### Claude Code / Claude Desktop (local binary)

```json
{
  "mcpServers": {
    "mcp-steam-scout": {
      "command": "/path/to/bin/mcp-server",
      "env": {
        "STEAM_API_KEY": "your_steam_api_key_here",
        "STEAM_ID": "your_optional_steam_id_here"
      }
    }
  }
}
```

---

## Development

### Running tasks

All tasks are managed as Go tools — no separate installation required:

```bash
go tool task build   # build the binary to ./bin/mcp-server
go tool task test    # run all tests
go tool task lint    # run golangci-lint
```

### Project layout

```
cmd/mcp-server/     entry point
internal/adapter/   Steam Web API client
internal/scraper/   Steam store trending scraper
internal/mcp/       MCP tool definitions
pkg/models/         shared data structures
npm/                npx wrapper and binary installer
```

### Releasing

Binaries are built and distributed via [GoReleaser](https://goreleaser.com). The npm package at `npm/` downloads the appropriate platform binary from GitHub Releases on install.
