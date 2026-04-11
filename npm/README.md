# mcp-steam-scout

[![MCP Server](https://img.shields.io/badge/MCP-Server-blue)](https://modelcontextprotocol.io)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/opdude/mcp-steam-scout/blob/main/LICENSE)

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants like Claude access to your **Steam and PlayStation libraries and current gaming trends** to make personalised game recommendations.

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
        "PSN_NPSSO": "your_npsso_token_here"
      }
    }
  }
}
```

Get a free Steam API key at [steamcommunity.com/dev/apikey](https://steamcommunity.com/dev/apikey).

You **must** set `STEAM_API_KEY`, and **at least one** of `STEAM_ID` or `STEAM_USERNAME`. `PSN_NPSSO` is optional — omit it if you don't need PlayStation tools.

## Available tools

### Steam

| Tool | Description |
|------|-------------|
| `resolve_steam_id` | Convert a Steam vanity username to a numeric Steam ID |
| `get_library` | Fetch your owned Steam games with playtime data |
| `get_trending` | Get currently trending games from the Steam store |

### PlayStation (requires `PSN_NPSSO`)

| Tool | Description |
|------|-------------|
| `get_psn_library` | Fetch your PS5 and PS4 games with playtime data |

## Getting your PSN NPSSO token

The NPSSO token is a session token issued by Sony after you log in to PlayStation. Sony does not provide an official API key — this is the standard method used by PSN tools and libraries.

1. Log in to [playstation.com](https://www.playstation.com) and make sure you are fully signed in.
2. Visit [https://ca.account.sony.com/api/v1/ssocookie](https://ca.account.sony.com/api/v1/ssocookie) — while logged in, this returns a JSON response containing your `npsso` value.
3. Copy the `npsso` value from the response and set it as `PSN_NPSSO` in your MCP client config.

> **Token expiry**: The NPSSO token expires after a period of inactivity. If PSN tools return authentication errors, repeat the steps above to get a fresh token.

## Example prompts

> "Fetch my Steam library, check what's trending, and recommend me something new to play based on what I've played the most."

> "Compare my Steam and PlayStation libraries — what genres do I play most across both platforms?"

## Full documentation

See the [GitHub repository](https://github.com/opdude/mcp-steam-scout) for full setup instructions, configuration options, and development docs.
