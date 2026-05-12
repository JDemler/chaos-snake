---
type: mdd-slice
title: Client
realizes: ./intent.md
status:
  implementation:
    status: implemented
    files:
      - web/index.html
      - web/app.js
      - web/style.css
      - cmd/server/main.go
      - internal/transport/hub.go
  evidence:
    - kind: api-test
      path: internal/transport/hub_test.go
      name: TestJoinAndInputFlow
---

# Client

## Behavior: Player Lands on the Site and Sees a Join Flow

When a player visits the site, they see a control to enter a display name and
a control to join the game.

## Behavior: Joining Places the Player Into a Live Field

After joining, the player is placed into one of the active fields and begins
controlling a snake immediately.

## Behavior: Header Shows Live Player and Field Counts

The page header reports, in real time, how many players are currently
connected to the game and how many fields are active. These counters update
whenever a player joins or leaves and whenever a field is created or
destroyed, so a viewer can always see the overall scale of the game.

## Behavior: Player Sees Their Current Field Large, Other Fields as Thumbnails

While playing, the client renders the field that the player's snake head
currently occupies as the main, large view, and renders every other active
field as a smaller live thumbnail beside or below the main view. When the
player teleports across an edge, the main view switches to follow them to
their new field; the field they left becomes one of the thumbnails.

## Behavior: Player Controls Via Arrow Keys, WASD, and On-Screen Touch Buttons

Players can change their snake's direction using either the arrow keys or
WASD on a keyboard, or by tapping on-screen directional buttons that the
client renders for touch devices. All three input methods are equivalent.

## Technical Constraint: Client Is Plain HTML, JavaScript, and CSS

The client is delivered as static HTML, JavaScript, and CSS. No client-side
framework or build pipeline is required to run it.

# Not Implemented

## Behavior: Site Shows a GitHub Sign-In Option

In addition to the basic name + join controls, the site shows a button to
authenticate with GitHub. Tapping it initiates the Supabase-managed OAuth
flow described in [auth.md](./auth.md). Until this is built, players can
only play anonymously by entering a name.
