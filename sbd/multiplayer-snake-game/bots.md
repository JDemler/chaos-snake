---
type: sbd-slice
title: Bots and Admin
realizes: ./intent.md
status:
  implementation:
    status: not-implemented
---


# Bots and Admin

# Not Implemented

## Behavior: Admin Interface Lives at /admin

The server exposes an admin interface at the path `/admin`. It is served by
the same Go binary as the game, alongside the static client. There is no
authentication on `/admin` in this iteration; access is gated only by
knowledge of the URL.

## Behavior: Admin Can Add a Single Bot

From the admin interface, the operator can add one bot to the game. The new
bot enters on a random tile of a random active field, exactly like a
respawning player.

## Behavior: Admin Can Remove a Single Bot

The operator can remove a specific bot from the game. The bot's snake is
taken out of play immediately. Admin removal is not a death: it does not
trigger respawn and does not produce a death-related leaderboard event.

## Behavior: Admin Can Remove All Bots

A single admin action removes every bot currently in the game. Human
players are unaffected.

## Behavior: Admin Can Set a Target Total Snake Count

The operator can configure a target total snake count `N`. While a target
is set, the server keeps `humans + bots == N` by:

- Spawning new bots when `humans + bots < N`.
- Removing bots when `humans + bots > N`, e.g. as humans join. Human
  players are never auto-removed.

When no target is set, the bot population is whatever the operator has
manually configured via add and remove.

## Behavior: Bots Are First-Class Participants in the Game

Bots are treated as players by the rest of the system:

- **Field scaling.** Bots contribute to `player_count` as defined in
  [fields.md](./fields.md). Adding bots can trigger new fields and
  removing them can let empty fields be destroyed.
- **Header counter.** Bots are included in the live "players connected"
  counter described in [client.md](./client.md).
- **Leaderboard.** Bots compete on both views in
  [leaderboard.md](./leaderboard.md): their current length contributes
  to the "currently longest living snake", and their lifetime maximum
  is tracked under a stable bot identity for the persistent ranking.
- **Death and respawn.** When a bot dies it follows the same automatic
  respawn rule as players in [gameplay.md](./gameplay.md). Bots only
  leave the game when an admin action removes them.

Each bot has a stable display name for the time it exists in the game.
The name is shown in the same places as a human player's name and is
used to attribute leaderboard records.

## Behavior: Bots Use One-Step Lookahead AI

On each server tick, before snake movement is applied, every alive bot
picks its next direction using only the current state. The choice is
constrained the same way a player's input is, including the no-reverse
rule that already applies to long-enough snakes.

1. **Evasion.** Discard any direction whose immediate next tile on the
   bot's current field is occupied by a snake body cell — the bot's own
   body or any other snake's body.
2. **Eating.** Among the remaining directions, pick the one whose next
   tile has the smallest Manhattan distance to that field's food
   pellet.

If every direction is filtered out by evasion, the bot keeps its current
direction and dies on the next step under the existing collision rules,
then respawns.

Bots do not plan further than one tile ahead, do not coordinate with
other bots, and do not predict other snakes' next moves.

## Technical Constraint: Bots Run Inside the Game Server Tick

Bot decisions are computed inside the Go server's tick loop and are
applied at the same tick boundary as player input. Bots are not separate
WebSocket clients and do not consume any external connection.
