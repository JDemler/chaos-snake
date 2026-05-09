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

## Behavior: Player Controls Via Arrow Keys, WASD, and On-Screen Touch Buttons

Players can change their snake's direction using either the arrow keys or
WASD on a keyboard, or by tapping on-screen directional buttons that the
client renders for touch devices. All three input methods are equivalent.

## Technical Constraint: Client Is Plain HTML, JavaScript, and CSS

The client is delivered as static HTML, JavaScript, and CSS. No client-side
framework or build pipeline is required to run it.
