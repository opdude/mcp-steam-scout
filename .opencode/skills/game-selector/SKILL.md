# Game Selector Skill

Suggests a game to play by analyzing your Steam library and current trending games.

## Usage

Ask me to suggest a game to play. I'll use the available MCP tools to:

1. Fetch your Steam library via `get_library`
2. Fetch trending games via `get_trending`
3. Cross-reference and make a recommendation

## Recommendation Priority

1. **Unplayed games you already own** — games in your library with 0 playtime
2. **Dabbled games** — games where you have some playtime but not a lot
3. **Trending purchase candidates** — trending games that match genres you play
4. **Revisit old favorites** — games you have significant playtime in but haven't played recently

## Limitations

- Xbox playtime is best-effort via userstats batch API; not all games return playtime data
- Epic Games Store has no playtime data at all
