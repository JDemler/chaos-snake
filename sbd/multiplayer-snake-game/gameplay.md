---
type: sbd-slice
title: Gameplay
realizes: ./intent.md
status:
  implementation:
    status: implemented
    files:
      - internal/game/game.go
  evidence:
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestSnakeMovesForwardOneTilePerTick
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestSnakeWrapsAroundEdges
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestPelletGrowsSnakeAndRespawns
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestSelfCollisionRespawns
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestFollowingOwnTailDoesNotKill
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestHeadOnHeadKillsBoth
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestRunningIntoOtherBodyKillsMover
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestReverseDirectionIsIgnoredForLongerSnake
---


# Gameplay

## Behavior: Each Player Controls One Snake While Alive

A player controls a single snake on whichever field they currently occupy.

## Behavior: Food Pellets Spawn on Each Field and Grow the Eater

Every active field has one food pellet at a random unoccupied tile. When a
snake's head enters the pellet's tile, the snake grows by one tile and a new
pellet spawns on a random unoccupied tile of the same field.

## Technical Constraint: Server Ticks at 10 Hz

The authoritative game loop advances state at ten ticks per second
(every 100 milliseconds). One tick equals one tile of snake movement.

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
