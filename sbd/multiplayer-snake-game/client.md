---
type: sbd-slice
title: Client
realizes: ./intent.md
status:
  implementation:
    status: not-implemented
---

# Client

## Behavior: Player Lands on the Site and Sees a Join Flow

When a player visits the site, they see a control to enter a display name, an
option to authenticate with GitHub, and a control to join the game.

## Behavior: Joining Places the Player Into a Live Field

After joining, the player is placed into one of the active fields and begins
controlling a snake immediately.

## Technical Constraint: Client Is Plain HTML, JavaScript, and CSS

The client is delivered as static HTML, JavaScript, and CSS. No client-side
framework or build pipeline is required to run it.
