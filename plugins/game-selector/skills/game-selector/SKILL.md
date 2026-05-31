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

## Handling Large MCP Output

`get_merged_library` can return 10,000+ lines of JSON and will be truncated by the tool output limit. You cannot read it directly. Instead:

- The truncated output is saved to a file by the tool output handler (path shown in the tool result message).
- Use **subagents** (Task tool) with `grep` to extract meaningful slices from that saved file.
- Search by `totalPlaytimeMinutes` to find played vs unplayed games.

## Key Analysis Tips

- **`totalPlaytimeMinutes` is already normalized across platforms** — `get_merged_library` handles this server-side. A game with `totalPlaytimeMinutes: 0` is truly unplayed across all platforms you own it on.
- **PlaytimeMinutes = minutes, not hours.** Convert for human-readable output (300 → "5 hours").
- **Trending games have 0 PlaytimeMinutes** — they're store data, not user data.
- **Xbox playtime** is best-effort via userstats batch API; not all games return MinutesPlayed.
- **Epic Games Store** has no playtime data at all. Epic-only titles in `get_merged_library` will show `totalPlaytimeMinutes: 0` but still count as "owned".
- **GOG playtime** requires the `GOG_COOKIE` env var; without it, GOG games have 0 playtime but still count as "owned".
- **Cross-platform patterns**: Use the `platforms` array in each merged entry to see which platforms you own a game on. Compare what you play on each platform to identify genre preferences.

## Analysis Steps (after calling MCP tools)

After `get_merged_library` and `get_trending` return:

### Step 1: Identify genre preferences from top played games
Launch a subagent to parse the saved `get_merged_library` output. Have it grep for the top 25 games by `totalPlaytimeMinutes` (descending) and extract their `displayName`. Convert minutes to hours. This reveals the user's genre preferences.

### Step 2: Find unplayed gems that match those genres
Have the same or another subagent scan the saved output for games with `totalPlaytimeMinutes: 0` whose display names suggest they match genres identified in Step 1. Return the top 10 most interesting matches with their platforms.

### Step 3: Cross-reference with trending
Scan the trending game names against the unplayed library list. If a trending game is already owned (unplayed), flag it as a high-priority recommendation.

### Step 4: Make the recommendation
Follow the Recommendation Priority. Present 2-3 options with a clear top pick, explaining why each fits the user's taste based on their most-played genres.

## Example Flow

```
User: "What should I play tonight?"
Agent:
  1. Calls get_merged_library → 200+ merged games, output truncated (14K lines)
  2. Calls get_trending → trending titles
  3. Launches subagent to extract top 25 played games from the saved MCP output using grep
  4. Subagent returns: Elden Ring (261h), Dark Souls III (142h), Civ V (75h) → likes RPGs & strategy
  5. Launches subagent to find unplayed matches → Baldur's Gate 3 (Xbox), Nioh Complete (Epic), Frostpunk (Xbox/Epic)
  6. Cross-references trending → user owns none
  7. Recommends:
      - "Baldur's Gate 3 (Xbox) — unplayed. Deep CRPG combining your love of RPGs and tactical strategy."
      - "Nioh: The Complete Edition (Epic) — unplayed. You have 261h in Elden Ring — Nioh is a combat-focused souls-like."
```
