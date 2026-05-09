---
type: system-intent
title: Multiplayer Snake Game
status:
  implementation:
    status: not-implemented
---

# Multiplayer Snake Game

A multiplayer web-based snake variant where many players share a dynamically
scaling set of playing fields. Crossing a field edge does not kill — the snake
re-emerges on the opposite edge of a different, randomly chosen field. Death
respawns the player into a random tile on a random field, so players are
continuously in play.

## Users

- Casual players who want to drop into a browser-based snake without an
  account.
- Returning players who authenticate with GitHub so their results are
  attributed to a persistent identity on the leaderboard.

## Slices

- [Client](./client.md) — what players see and how they join.
- [Auth](./auth.md) — naming, anonymous play, and GitHub authentication.
- [Gameplay](./gameplay.md) — snake movement, death, and respawn rules.
- [Fields](./fields.md) — field size, scaling, lifecycle, and cross-field
  teleport.
- [Leaderboard](./leaderboard.md) — what is tracked and shown.
- [Architecture](./architecture.md) — system boundaries and deployment.
