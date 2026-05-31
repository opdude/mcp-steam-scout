---
name: game-selector
description: Helps choose what game to play by cross-referencing your Steam, PSN, Xbox, Epic, and GOG libraries with current trending games. Use this whenever the user asks for a game recommendation, says they're bored or can't decide what to play, wants to find something new that fits their taste, or wants to discover unplayed gems they already own across any platform. Also use when comparing libraries across platforms to find genre preferences, or when the user wants to know if a trending game is already in their library.
compatibility: opencode, claude
---

## Usage

Ask me to suggest a game to play. I'll use the available MCP tools to:

1. Fetch and merge your entire library via `get_merged_library` — this calls all configured platform adapters, normalizes titles (strips edition suffixes, trademark symbols, year/platform tags), and merges matching games into single entries with `totalPlaytimeMinutes` already summed across platforms. Sorted ascending — unplayed games appear first.
2. Fetch trending games via `get_trending`
3. Cross-reference and make a recommendation

## Detecting Available Platforms

The `get_merged_library` tool automatically detects which platforms are configured and includes their data. You don't need to call individual library tools. However, `get_trending` must be called separately.

- `get_merged_library` — always present; merges all configured platforms
- `get_trending` — always present
- `get_library`, `get_psn_library`, `get_xbox_library`, `get_epic_library`, `get_gog_library` — individual tools if you need raw per-platform data

## Recommendation Priority

1. **Unplayed games you already own** — games in your library with 0 playtime across all platforms
2. **Dabbled games** — games where you have some playtime but not a lot
3. **Trending purchase candidates** — trending games that match genres you play
4. **Revisit old favorites** — games you have significant playtime in but haven't played recently

## Key Analysis Tips

- **`totalPlaytimeMinutes` is already normalized across platforms** — `get_merged_library` handles this server-side. A game with `totalPlaytimeMinutes: 0` is truly unplayed across all platforms you own it on.
- **PlaytimeMinutes = minutes, not hours.** Convert for human-readable output (300 → "5 hours").
- **Trending games have 0 PlaytimeMinutes** — they're store data, not user data.
- **Xbox playtime** is best-effort via userstats batch API; not all games return MinutesPlayed.
- **Epic Games Store** has no playtime data at all. Epic-only titles in `get_merged_library` will show `totalPlaytimeMinutes: 0` but still count as "owned".
- **GOG playtime** requires the `GOG_COOKIE` env var; without it, GOG games have 0 playtime but still count as "owned".
- **Cross-platform patterns**: Use the `platforms` array in each merged entry to see which platforms you own a game on. Compare what you play on each platform to identify genre preferences.

## Example Flow

```
User: "What should I play tonight?"
Agent:
  1. Checks available tools — sees get_merged_library, get_trending
  2. Calls get_merged_library → 200 merged games across Steam/PSN/Xbox/Epic/GOG, sorted by playtime ascending
  3. Calls get_trending → 10 trending titles
  4. Reads merged results: unplayed games appear first with totalPlaytimeMinutes: 0
  5. Analysis:
     - Top playtime: Elden Ring (155h on PSN), Dark Souls III (116h PSN), Civ V (75h Steam) → likes RPGs & strategy
     - Unplayed (totalPlaytimeMinutes: 0): Baldur's Gate 3 (Xbox), Nioh Complete (Epic), Frostpunk (Xbox/Epic)
     - Trending overlap: user owns none
     - Trending games are mostly action-RPGs
  6. Recommends:
      - "Baldur's Gate 3 (Xbox) — unplayed. Deep CRPG combining your love of RPGs and tactical strategy."
      - "Nioh: The Complete Edition (Epic) — unplayed. You have 155h in Elden Ring — Nioh is a combat-focused souls-like with deep mechanics."
```
