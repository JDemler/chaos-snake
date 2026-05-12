---
type: mdd-slice
title: Leaderboard
realizes: ./intent.md
status:
  implementation:
    status: partially-implemented
    files:
      - internal/game/game.go
      - internal/transport/message.go
      - web/app.js
      - web/index.html
      - web/style.css
  evidence:
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestLeaderboardRecordsPeakLengthAcrossDeaths
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestLeaderboardOrdersByPeakThenName
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestLeaderboardIncludesNewlyJoinedPlayer
---

# Leaderboard

## Behavior: Leaderboard Tracks Highest Snake Length Per Player

For each player, the leaderboard records the highest length their snake has
ever reached, across all of their lives.

Until GitHub authentication is wired up, "player" is keyed by display name,
so two anonymous players sharing a name share a record.

## Behavior: Leaderboard Surfaces the Currently Longest Living Snake

In addition to the persistent ranking, the leaderboard surfaces, in real
time, which currently alive snake is the longest, so players and spectators
can see the current king of the game.

## Technical Constraint: Persistent Leaderboard Data Lives in Supabase

Persistent leaderboard records are stored in Supabase. The game server reads
from and writes to Supabase rather than maintaining its own persistent store.

Not yet realized: peak lengths currently live in process memory on the game
server and are lost on restart. Supabase wiring is deferred until auth is in
place.
