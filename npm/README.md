# mcp-steam-scout

[![MCP Server](https://img.shields.io/badge/MCP-Server-blue)](https://modelcontextprotocol.io)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/opdude/mcp-steam-scout/blob/main/LICENSE)

An [MCP](https://modelcontextprotocol.io) server that gives AI assistants like Claude access to your **Steam library and current gaming trends** to make personalised game recommendations.

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
        "STEAM_USERNAME": "your_steam_username_here"
      }
    }
  }
}
```

Get a free Steam API key at [steamcommunity.com/dev/apikey](https://steamcommunity.com/dev/apikey).

You **must** set `STEAM_API_KEY`, and **at least one** of `STEAM_ID` or `STEAM_USERNAME` for the server to function.

## Available tools

| Tool | Description |
|------|-------------|
| `resolve_steam_id` | Convert a Steam vanity username to a numeric Steam ID |
| `get_library` | Fetch your owned games with playtime data (requires STEAM_ID or STEAM_USERNAME) |
| `get_trending` | Get currently trending games from the Steam store |

## Example prompt

> "Look up my Steam ID for username opdude, fetch my library, check what's trending, and recommend me something new to play based on what I've played the most."

## Full documentation

See the [GitHub repository](https://github.com/opdude/mcp-steam-scout) for full setup instructions, configuration options, and development docs.
