---
name: mcp-steam-scout-setup
description: Helps users set up and configure the mcp-steam-scout MCP server. Use this when the user asks how to install, configure tokens, set up platforms (Steam, PSN, Xbox, Epic, GOG), troubleshoot authentication errors, or verify their setup is working. Also use when the user wants to add or refresh any platform token.
compatibility: opencode, claude
---

# mcp-steam-scout Setup Guide

Guide the user through setting up the mcp-steam-scout MCP server step by step. Start by checking what they already have configured and only walk through what's missing.

## Required Tokens Overview

| Platform | Env Variable | How to get it |
|---|---|---|
| Steam | `STEAM_API_KEY` + `STEAM_ID` or `STEAM_USERNAME` | Free API key from steamcommunity.com/dev/apikey. STEAM_ID is their 17-digit numeric ID; STEAM_USERNAME is their vanity name. |
| PlayStation | `PSN_NPSSO` | Cookie from playstation.com after login |
| Xbox | `XBOX_REFRESH_TOKEN` | Device code flow via setup-xbox CLI |
| Epic | `EPIC_REFRESH_TOKEN` | OAuth flow via setup-epic CLI |
| GOG | `GOG_REFRESH_TOKEN` + `GOG_COOKIE` (optional) | OAuth flow via setup-gog CLI |

## Step-by-step Setup

### 1. Steam API Key

Tell the user to:
1. Visit https://steamcommunity.com/dev/apikey
2. Log in with their Steam account
3. Enter any domain (e.g. "localhost") and click Register
4. Copy the API key

They need to either provide their **Steam ID** (17 digits from https://steamid.io) or their **Steam vanity username** (the custom URL part of their profile).

### 2. PSN NPSSO (optional)

Tell the user to:
1. Log in at https://www.playstation.com
2. Visit https://ca.account.sony.com/api/v1/ssocookie
3. Copy the `npsso` value from the JSON response

> The NPSSO token expires after inactivity. If PSN tools return auth errors, get a fresh one.

### 3. Xbox Refresh Token (optional)

Tell the user to run:
```bash
go run github.com/opdude/mcp-steam-scout/cmd/setup-xbox@latest
```

The tool will display a URL and code. They visit the URL, enter the code, authenticate, and the tool prints the refresh token.

### 4. Epic Refresh Token (optional)

Tell the user to run:
```bash
go run github.com/opdude/mcp-steam-scout/cmd/setup-epic@latest
```

The tool shows a login URL. They log in, get redirected to a URL with a code, paste it back, and get the refresh token.

> If they get `corrective_action_required`, they need to visit store.epicgames.com, log in, and accept any pending privacy/EULA prompts first.

### 5. GOG Refresh Token (optional)

Tell the user to run:
```bash
go run github.com/opdude/mcp-steam-scout/cmd/setup-gog@latest
```

Same OAuth flow — visit URL, log in, paste code, get token.

> **Playtime data**: For GOG playtime, they also need `GOG_COOKIE` set to the `gog-al` cookie value from gog.com browser cookies.

## Client Configuration

After the user has collected all their tokens, help them configure their MCP client.

### Opencode config (~/.config/opencode/opencode.json or opencode.json)

```json
{
  "mcp": {
    "mcp-steam-scout": {
      "type": "local",
      "command": ["npx", "-y", "@opdude/mcp-steam-scout"],
      "enabled": true,
      "environment": {
        "STEAM_API_KEY": "...",
        "STEAM_USERNAME": "...",
        "PSN_NPSSO": "...",
        "XBOX_REFRESH_TOKEN": "...",
        "EPIC_REFRESH_TOKEN": "...",
        "GOG_REFRESH_TOKEN": "...",
        "GOG_COOKIE": "..."
      }
    }
  }
}
```

### Claude Desktop / Claude Code

```json
{
  "mcpServers": {
    "mcp-steam-scout": {
      "command": "npx",
      "args": ["-y", "@opdude/mcp-steam-scout"],
      "env": {
        "STEAM_API_KEY": "...",
        "STEAM_USERNAME": "...",
        "PSN_NPSSO": "...",
        "XBOX_REFRESH_TOKEN": "...",
        "EPIC_REFRESH_TOKEN": "...",
        "GOG_REFRESH_TOKEN": "...",
        "GOG_COOKIE": "..."
      }
    }
  }
}
```

If the user built from source instead of npx, use the binary path:
```
"command": "/path/to/bin/mcp-server"
```

## Verification

After configuration, help the user verify by:
1. Restarting their MCP client
2. Calling each `get_*_library` tool to confirm platforms respond
3. Calling `get_trending` to confirm trending data works
4. Asking for a recommendation to test full cross-platform analysis

If any platform returns an auth error, walk through getting a fresh token for that platform and updating the config.
