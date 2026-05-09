---
type: sbd-slice
title: Fields
realizes: ./intent.md
status:
  implementation:
    status: not-implemented
---

# Fields

## Technical Constraint: Each Field Is 30 by 30 Tiles

Every field has a fixed size of 30 by 30 tiles.

## Behavior: Field Count Scales With Concurrent Players

The server provisions roughly one field per five concurrent players. As
players join, additional fields are created so the player-to-field ratio
stays near five players per field.

## Behavior: Empty Fields Are Destroyed Unless They Are the Last One

A field is destroyed when it has no players left, except for the final
remaining field. At least one field is always preserved so that newly joining
players and respawning players have somewhere to spawn.

## Behavior: Cross-Field Teleport Targets a Random Other Field

When a snake exits an edge of its current field, the destination field is
chosen uniformly at random from the other active fields, and the snake
re-emerges on the edge opposite the one it exited.
