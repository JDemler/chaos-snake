---
type: sbd-slice
title: Gameplay
realizes: ./intent.md
status:
  implementation:
    status: not-implemented
---

# Gameplay

## Behavior: Each Player Controls One Snake While Alive

A player controls a single snake on whichever field they currently occupy.

## Behavior: Field Edges Teleport, They Do Not Kill

When a snake moves past the edge of its current field, it does not die. It
re-emerges on the opposite edge of a different active field, chosen at random
from the other active fields. If only one field is active, the snake
re-emerges on the opposite edge of the same field.

## Behavior: Snakes Die From Collisions Only

The only way a snake dies is by collision. Field edges never kill.

### Behavior: Self-Collision Kills the Snake

If a snake's head enters a tile occupied by its own body, the snake dies.

### Behavior: Hitting Another Snake's Body Kills the Moving Snake

If a snake's head enters a tile occupied by another snake's body, the moving
snake dies. The other snake survives.

### Behavior: Head-on-Head Collision Kills Both Snakes

If two snake heads enter the same tile in the same tick, both snakes die.

## Behavior: Dead Players Respawn Automatically

When a snake dies, the player is automatically respawned at a random tile on a
randomly chosen active field. Players are not removed from the game on death.
