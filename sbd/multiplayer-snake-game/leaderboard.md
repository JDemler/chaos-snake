---
type: sbd-slice
title: Leaderboard
realizes: ./intent.md
status:
  implementation:
    status: not-implemented
---

# Leaderboard

## Behavior: Leaderboard Tracks Highest Snake Length Per Player

For each player, the leaderboard records the highest length their snake has
ever reached, across all of their lives.

## Behavior: Leaderboard Surfaces the Currently Longest Living Snake

In addition to the persistent ranking, the leaderboard surfaces, in real
time, which currently alive snake is the longest, so players and spectators
can see the current king of the game.

## Technical Constraint: Persistent Leaderboard Data Lives in Supabase

Persistent leaderboard records are stored in Supabase. The game server reads
from and writes to Supabase rather than maintaining its own persistent store.
