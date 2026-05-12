---
type: mdd-slice
title: Fields
realizes: ./intent.md
status:
  implementation:
    status: implemented
    files:
      - internal/game/game.go
  evidence:
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestSingleFieldEdgeWraps
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestMultiFieldTeleportPreservesPerpendicularCoord
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestSnakeStraddlesTwoFieldsAfterCrossing
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestFieldSpawnsOnSixthJoin
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestFieldSpawnsScaleAtEleventhJoin
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestEmptyFieldDestroyedWhenNotLast
    - kind: unit-test
      path: internal/game/game_test.go
      name: TestLastFieldIsNotDestroyedWhenEmpty
---

# Fields

## Technical Constraint: Each Field Is 30 by 30 Tiles

Every field has a fixed size of 30 by 30 tiles.

## Behavior: Field Count Scales With Concurrent Players

The server provisions roughly one field per five concurrent players. As
players join, additional fields are created so the player-to-field ratio
stays near five players per field.

## Behavior: Empty Fields Are Destroyed Only When the Field Count Exceeds Capacity

A field with no remaining snake body cells is destroyed only when the active
field count is greater than the scaling target of one field per five
players, i.e. when `field_count > ceil(player_count / 5)`. Empty fields that
are still needed to satisfy that target are preserved, so an isolated player
leaving or teleporting out of a field does not by itself cause the field to
disappear and does not cause any other field to disappear in its place.

At least one field is always preserved (the target is treated as at least 1)
so that newly joining and respawning players always have somewhere to spawn,
even when no one is connected.

## Behavior: Cross-Field Teleport Targets a Random Other Field

When a snake exits an edge of its current field, the destination field is
chosen uniformly at random from the other active fields, and the snake
re-emerges on the edge opposite the one it exited. Its position along that
opposite edge preserves the perpendicular coordinate of the exit point: a
snake leaving the right edge at row y enters the left edge of the
destination at row y; a snake leaving the bottom edge at column x enters
the top edge at column x.

If only one field is active, there is no other field to choose, and the
snake re-emerges on the opposite edge of the same field with the
perpendicular coordinate preserved (i.e., classic wrap-around).
