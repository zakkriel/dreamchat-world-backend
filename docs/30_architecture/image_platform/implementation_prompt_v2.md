# Implementation Prompt — Image Platform V2 Addendum

Implement the V2 addendum for DreamChat Image Platform without replacing the existing service architecture.

Keep existing principles:

- standalone service
- Go API + workers
- OpenAPI-first
- async generation jobs
- bearer token auth
- Postgres source of truth
- Redis queue MVP
- S3 object storage
- provider adapters
- retrieval before generation

Add support for:

1. NPC expression sprite sheet generation
2. deterministic sprite sheet slicing
3. parent/child visual asset relationships
4. expression metadata and fallback mapping
5. location scene-state variant matching
6. artifact display-context metadata
7. UI inspection vs world action metadata

Build incrementally:

## Phase A — Data Model

- Add parent_asset_id, crop_index, crop_box, expression_key, scene_state, match_tags to visual_asset.
- Add sprite_sheet_contract table.
- Add sprite_sheet_slice table.
- Seed the 2x5 character expression contract.

## Phase B — Slicing Service

Create an internal package:

```txt
/internal/spritesheets
```

It should:

- load image
- validate dimensions
- slice by rows/columns
- apply safe margin
- write outputs to storage
- return crop metadata
- record slice assets

## Phase C — API

Add endpoints or equivalent handlers:

```txt
POST /v1/characters/{character_id}/generate-expression-sheet
GET /v1/characters/{character_id}/expression-asset
POST /v1/places/{place_id}/resolve-scene-asset
POST /v1/places/{place_id}/generate-variant-pack
```

## Phase D — Worker

Add job type:

```txt
character_expression_sheet
```

Worker flow:

1. compile prompt
2. call provider
3. store parent sheet
4. slice sheet
5. create child assets
6. run basic quality checks
7. mark job preview_ready/ready

## Phase E — Fallbacks

Runtime expression retrieval must fallback:

1. requested expression
2. closest mapped expression
3. neutral
4. base portrait

Do not trigger runtime generation by default.

## Done When

- A mock provider can generate a deterministic 2x5 placeholder sheet.
- The service slices it into 10 assets.
- Each sliced asset is retrievable by expression.
- Location scene-state matching returns an existing best match.
- Tests cover slicing, fallback, and parent/child asset metadata.
