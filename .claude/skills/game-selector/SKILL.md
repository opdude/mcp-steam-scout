---
name: game-selector
description: Helps choose what game to play by cross-referencing your Steam, PSN, Xbox, Epic, and GOG libraries with current trending games. Use this whenever the user asks for a game recommendation, says they're bored or can't decide what to play, wants to find something new that fits their taste, or wants to discover unplayed gems they already own across any platform. Also use when comparing libraries across platforms to find genre preferences, or when the user wants to know if a trending game is already in their library.
compatibility: opencode, claude
---

## Usage

Ask me to suggest a game to play. I'll use the available MCP tools to:

1. Fetch your Steam library via `get_library`
2. Fetch your PlayStation library via `get_psn_library` (if configured)
3. Fetch your Xbox library via `get_xbox_library` (if configured)
4. Fetch your Epic Games Store library via `get_epic_library` (if configured)
5. Fetch your GOG library via `get_gog_library` (if configured)
6. Fetch trending games via `get_trending`
7. Cross-reference across all platforms and make a recommendation

## Detecting Available Platforms

Check your available MCP tools at the start of each session:

- `get_library` — always present (Steam)
- `get_psn_library` — present when `PSN_NPSSO` is configured
- `get_xbox_library` — present when `XBOX_REFRESH_TOKEN` is configured
- `get_epic_library` — present when `EPIC_REFRESH_TOKEN` is configured
- `get_gog_library` — present when `GOG_REFRESH_TOKEN` is configured
- `get_trending` — always present

Call every tool that exists. Skip gracefully if one errors.

## Recommendation Priority

1. **Unplayed games you already own** — games in your library with 0 playtime across all platforms
2. **Dabbled games** — games where you have some playtime but not a lot
3. **Trending purchase candidates** — trending games that match genres you play
4. **Revisit old favorites** — games you have significant playtime in but haven't played recently

## Key Analysis Tips

- **Normalize game identities across platforms and editions**: Many games appear as multiple entries (e.g. "Skyrim" on Steam + "Skyrim SE" on PS, or "Metro 2033" + "Metro 2033 Redux" on the same platform). Fuzzy-match by name, strip edition suffixes ("Special Edition", "Remastered", "Redux", "GOTY", "Enhanced Edition"), and **aggregate total playtime across all platforms and all editions** before categorizing. A game with 0 min on one edition but 670 min on a remaster is **not unplayed** — treat it as played.
- **PlaytimeMinutes = minutes, not hours.** Convert for human-readable output (300 → "5 hours").
- **Trending games have 0 PlaytimeMinutes** — they're store data, not user data.
- **Xbox playtime** is best-effort via userstats batch API; not all games return MinutesPlayed.
- **Epic Games Store** has no playtime data at all — only names. Epic-only titles still count as "owned".
- **GOG playtime** requires the `GOG_COOKIE` env var; without it, GOG games have 0 playtime but still count as "owned".
- **Cross-platform patterns**: Compare what you play on each platform to identify genre preferences.

## Example Flow

```
User: "What should I play tonight?"
Agent:
  1. Checks available tools — sees get_library, get_psn_library, get_xbox_library, get_epic_library, get_trending
  2. Calls resolve_steam_id("opdude") → "76561197960287930"
  3. Calls get_library → 45 Steam games
  4. Calls get_psn_library → 12 PSN games
  5. Calls get_xbox_library → 149 Xbox games
  6. Calls get_epic_library → 15 Epic games
  7. Calls get_trending → 10 trending titles
  8. Normalizes: Skyrim (Steam 90h, Xbox 2h, PS 50min) → ~93h total. Not unplayed.
  9. Analysis:
     - Top playtime: Elden Ring (340h Steam), RimWorld (200h Steam) → likes RPGs & strategy
     - Unplayed (0 min across all platforms): Baldur's Gate 3, Disco Elysium
     - Epic-only owned: Sid Meier's Civilization VI (no playtime data)
     - Trending overlap: user owns none
     - Trending games are mostly action-RPGs
  10. Recommends:
      - "Disco Elysium (Steam) — owned with only 12 minutes played. Deep narrative RPG in your favorite genre."
      - "Baldur's Gate 3 (Steam) — trending #2, you already own it but haven't started."
```
