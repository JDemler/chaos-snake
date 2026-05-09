---
type: sbd-slice
title: Architecture
realizes: ./intent.md
status:
  implementation:
    status: not-implemented
---

# Architecture

## Technical Constraint: Game Server Is a Go Binary on a VPS

The authoritative game server is a single Go binary deployed to a VPS. All
gameplay state and tick logic live on the server.

## Technical Constraint: Client Is Static Web Assets

The client is plain HTML, JavaScript, and CSS, served as static assets. The
client renders state and forwards inputs but does not simulate the game.

## Technical Constraint: Auth and Persistence Are Handled by Supabase

Identity (GitHub OAuth) and persistent data such as the leaderboard live in
Supabase. The Go server reads from and writes to Supabase rather than
running its own auth or persistent storage.
