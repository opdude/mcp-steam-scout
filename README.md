# mcp-steam-scout

[![MCP Server](https://img.shields.io/badge/MCP-Server-blue)](https://modelcontextprotocol.io)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants like Claude access to your **Steam and PlayStation libraries and current gaming trends** to make personalised game recommendations weighted by actual playtime.

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

### Steam tools

| Tool | What it does |
|------|-------------|
| `resolve_steam_id` | Converts a Steam vanity username to a numeric Steam ID |
| `get_library` | Fetches your owned Steam games including playtime data |
| `get_trending` | Returns currently trending games from the Steam store |

### PlayStation tools (optional)

Enabled automatically when `PSN_NPSSO` is set.

| Tool | What it does |
|------|-------------|
| `get_psn_library` | Fetches your PS5 and PS4 games including playtime data |

---

## What you can do with it

Ask Claude things like:

> "Look up my Steam ID for username opdude, fetch my library, check what's trending, and recommend me something new to play based on what I've played the most."

> "What are the top trending games on Steam right now? Which ones match my playstyle based on my library?"

> "Compare my Steam and PlayStation libraries — what genres do I play most across both platforms?"

> "I mostly play strategy games — are any trending games in that genre worth trying?"

Claude chains the tools together automatically, cross-references your playtime with current trends, and gives you personalised recommendations.

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

### Find your Steam Username or ID

You can configure the server using your Steam vanity username or your 17-digit Steam ID. You can:

- Look up your Steam ID manually at [steamid.io](https://steamid.io)
- Ask Claude to run `resolve_steam_id` with your vanity username to find your ID.

### Get your PSN NPSSO token (optional)

The NPSSO token is a session token issued by Sony after you log in to PlayStation. Sony does not provide an official API key — this is the standard method used by PSN tools and libraries.

1. Log in to [playstation.com](https://www.playstation.com) in your browser and make sure you are fully signed in.
2. Visit [https://ca.account.sony.com/api/v1/ssocookie](https://ca.account.sony.com/api/v1/ssocookie) — while logged in, this returns a JSON response containing your `npsso` value.
3. Copy the `npsso` value from the response and set it as `PSN_NPSSO` in your MCP client config.

> **Token expiry**: The NPSSO token expires after a period of inactivity. If PSN tools return authentication errors, repeat the steps above to get a fresh token.

---

## Client configuration

You **must** set `STEAM_API_KEY`, and **at least one** of `STEAM_ID` or `STEAM_USERNAME`. `PSN_NPSSO` is optional and enables PlayStation tools when set.

### Claude Code / Claude Desktop (npx)

```json
{
  "mcpServers": {
    "mcp-steam-scout": {
      "command": "npx",
      "args": ["-y", "@opdude/mcp-steam-scout"],
      "env": {
        "STEAM_API_KEY": "your_steam_api_key_here",
        "STEAM_USERNAME": "your_steam_username_here",
        "PSN_NPSSO": "your_npsso_token_here"
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
        "STEAM_USERNAME": "your_steam_username_here",
        "PSN_NPSSO": "your_npsso_token_here"
      }
    }
  }
}
```

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `STEAM_API_KEY` | Yes | Steam Web API key from [steamcommunity.com/dev/apikey](https://steamcommunity.com/dev/apikey) |
| `STEAM_ID` | One of these | Your 17-digit numeric Steam ID |
| `STEAM_USERNAME` | One of these | Your Steam vanity username |
| `PSN_NPSSO` | No | NPSSO token from the `npsso` cookie on playstation.com. Enables PSN tools when set. |

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
internal/adapter/   Steam and PSN API clients
internal/scraper/   Steam and PlayStation Store trending scrapers
internal/mcp/       MCP tool definitions
pkg/models/         shared data structures
npm/                npx wrapper and binary installer
```

### Releasing

Binaries are built and distributed via [GoReleaser](https://goreleaser.com). The npm package at `npm/` downloads the appropriate platform binary from GitHub Releases on install.
