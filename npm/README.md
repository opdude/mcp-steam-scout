# mcp-steam-scout

[![MCP Server](https://img.shields.io/badge/MCP-Server-blue)](https://modelcontextprotocol.io)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/opdude/mcp-steam-scout/blob/main/LICENSE)

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants like Claude access to your **Steam, PlayStation, Xbox, Epic Games Store, and GOG libraries and current gaming trends** to make personalised game recommendations.

## Quick start

Add to your MCP client config (e.g. Claude Code or Claude Desktop):

```json
{
  "mcpServers": {
    "mcp-steam-scout": {
      "command": "npx",
      "args": ["-y", "@opdude/mcp-steam-scout"],
      "env": {
        "STEAM_API_KEY": "your_steam_api_key_here",
        "STEAM_USERNAME": "your_steam_username_here",
        "PSN_NPSSO": "your_npsso_token_here",
        "XBOX_REFRESH_TOKEN": "your_xbox_refresh_token_here",
        "EPIC_REFRESH_TOKEN": "your_epic_refresh_token_here",
        "GOG_REFRESH_TOKEN": "your_gog_refresh_token_here",
        "GOG_COOKIE": "your_gog_al_cookie_value_here"
      }
    }
  }
}
```

Get a free Steam API key at [steamcommunity.com/dev/apikey](https://steamcommunity.com/dev/apikey).

You **must** set `STEAM_API_KEY`, and **at least one** of `STEAM_ID` or `STEAM_USERNAME`. All other tokens are optional — omit them if you don't need that platform's tools.

## Available tools

### Steam

| Tool | Description |
|------|-------------|
| `resolve_steam_id` | Convert a Steam vanity username to a numeric Steam ID |
| `get_library` | Fetch your owned Steam games with playtime data |

### Trending (no setup required)

| Tool | Description |
|------|-------------|
| `get_trending` | Get currently trending games from Steam and GOG |

### PlayStation (requires `PSN_NPSSO`)

| Tool | Description |
|------|-------------|
| `get_psn_library` | Fetch your PS5 and PS4 games with playtime data |

### Xbox (requires `XBOX_REFRESH_TOKEN`)

| Tool | Description |
|------|-------------|
| `get_xbox_library` | Fetch your Xbox game library via Xbox Live |

### Epic Games Store (requires `EPIC_REFRESH_TOKEN`)

| Tool | Description |
|------|-------------|
| `get_epic_library` | Fetch your Epic Games Store library (playtime data is not available from Epic's API) |

### GOG (requires `GOG_REFRESH_TOKEN`)

| Tool | Description |
|------|-------------|
| `get_gog_library` | Fetch your GOG library with playtime data |

## Getting your PSN NPSSO token

The NPSSO token is a session token issued by Sony after you log in to PlayStation. Sony does not provide an official API key — this is the standard method used by PSN tools and libraries.

1. Log in to [playstation.com](https://www.playstation.com) and make sure you are fully signed in.
2. Visit [https://ca.account.sony.com/api/v1/ssocookie](https://ca.account.sony.com/api/v1/ssocookie) — while logged in, this returns a JSON response containing your `npsso` value.
3. Copy the `npsso` value from the response and set it as `PSN_NPSSO` in your MCP client config.

> **Token expiry**: The NPSSO token expires after a period of inactivity. If PSN tools return authentication errors, repeat the steps above to get a fresh token.

## Getting your Xbox refresh token

The Xbox refresh token is obtained via Microsoft's device code flow. Run the setup tool directly (no clone needed):

```bash
go run github.com/opdude/mcp-steam-scout/cmd/setup-xbox@latest
```

The tool will guide you through authentication and print the `XBOX_REFRESH_TOKEN` value to add to your config.

## Getting your Epic Games Store refresh token

The Epic refresh token is obtained via an OAuth flow. Run the setup tool directly (no clone needed):

```bash
go run github.com/opdude/mcp-steam-scout/cmd/setup-epic@latest
```

The tool will guide you through authentication and print the `EPIC_REFRESH_TOKEN` value to add to your config.

> **Privacy policy / EULA acceptance**: If authentication fails with `corrective_action_required`, first visit [store.epicgames.com](https://store.epicgames.com) in your browser, log in, and accept any pending privacy policy or terms of service prompts. Then try the setup tool again.

## Getting your GOG refresh token

The GOG refresh token is obtained via an OAuth flow. Run the setup tool directly (no clone needed):

```bash
go run github.com/opdude/mcp-steam-scout/cmd/setup-gog@latest
```

The tool will guide you through authentication and print the `GOG_REFRESH_TOKEN` value to add to your config.

> **Playtime data**: GOG playtime requires an additional web session cookie. Set `GOG_COOKIE` to the value of the `gog-al` cookie from your browser after logging into gog.com.

## Example prompts

> "Fetch my Steam library, check what's trending, and recommend me something new to play based on what I've played the most."

> "Compare my Steam, PlayStation, Xbox, Epic, and GOG libraries — what genres do I play most across all platforms?"

## Installing the skills

This repo includes AI agent skills for cross-platform game recommendations and setup guidance.

```bash
npx skills add https://github.com/opdude/mcp-steam-scout --skill game-selector
npx skills add https://github.com/opdude/mcp-steam-scout --skill mcp-steam-scout-setup
```

For Claude Code, use the plugin marketplace instead:

```bash
/plugin marketplace add opdude/mcp-steam-scout
/plugin install game-selector@mcp-steam-scout
/plugin install mcp-steam-scout-setup@mcp-steam-scout
```

## Full documentation

See the [GitHub repository](https://github.com/opdude/mcp-steam-scout) for full setup instructions, configuration options, and development docs.
