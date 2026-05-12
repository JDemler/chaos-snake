---
type: mdd-slice
title: Authentication and Identity
realizes: ./intent.md
status:
  implementation:
    status: not-implemented
---

# Authentication and Identity

## Behavior: Players Can Play Anonymously With a Chosen Name

A player may join the game by entering a display name. GitHub authentication
is not required to play.

## Behavior: GitHub Authentication Provides Persistent Identity

A player may sign in with GitHub. An authenticated player's results are
attributed to their GitHub identity for purposes such as the leaderboard.

## Technical Constraint: Auth Is Handled Through Supabase

Authentication, including the GitHub OAuth flow, is handled through Supabase.
The game server does not implement its own auth.
