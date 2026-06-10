# API Addendum — Asset Packs and Sprite Sheets

## 1. Purpose

This document extends the existing DreamChat Image Platform API with explicit sprite sheet and asset-pack behavior.

The existing API already supports character/place visual identities and pack generation. This addendum clarifies the request/response shape needed for sprite sheets, scene-state matching, and runtime expression retrieval.

## 2. Generate Character Expression Sheet

```txt
POST /v1/characters/{character_id}/generate-expression-sheet
```

### Auth

`images:write`

### Request

```json
{
  "visual_identity_id": "vid_char_123",
  "style_profile_id": "style_cinematic_storyworld",
  "quality_tier": "standard",
  "sheet_contract": {
    "rows": 2,
    "columns": 5,
    "expression_order": [
      "neutral",
      "warm",
      "amused",
      "suspicious",
      "angry",
      "afraid",
      "sad",
      "surprised",
      "focused",
      "exhausted"
    ]
  },
  "generation_mode": "sheet_then_slice",
  "fallback_mode": "separate_expressions_if_sheet_fails"
}
```

### Response

```json
{
  "job_id": "job_123",
  "status": "queued",
  "job_type": "character_expression_sheet",
  "visual_identity_id": "vid_char_123"
}
```

## 3. Get Best Character Expression Asset

```txt
GET /v1/characters/{character_id}/expression-asset?expression=suspicious&angle=front&fallback=true
```

### Auth

`images:read`

### Response

```json
{
  "asset": {
    "asset_id": "asset_expr_004",
    "asset_type": "character_expression",
    "variant_key": "expression_suspicious_front",
    "url": "https://cdn.example.com/assets/asset_expr_004.webp",
    "metadata": {
      "expression": "suspicious",
      "angle": "front",
      "fallback_used": false
    }
  }
}
```

If fallback is used:

```json
{
  "asset": {
    "asset_id": "asset_expr_001",
    "asset_type": "character_expression",
    "variant_key": "expression_neutral_front",
    "url": "https://cdn.example.com/assets/asset_expr_001.webp",
    "metadata": {
      "expression": "neutral",
      "requested_expression": "skeptical",
      "fallback_used": true,
      "fallback_reason": "requested_expression_not_available"
    }
  }
}
```

## 4. Generate Location Variant Pack

```txt
POST /v1/places/{place_id}/generate-variant-pack
```

### Auth

`images:write`

### Request

```json
{
  "visual_identity_id": "vid_place_123",
  "style_profile_id": "style_cinematic_storyworld",
  "variants": [
    {
      "variant_key": "day_busy",
      "scene_state": {
        "time_of_day": "day",
        "crowd_level": "busy",
        "mood": "normal"
      }
    },
    {
      "variant_key": "night_quiet",
      "scene_state": {
        "time_of_day": "night",
        "crowd_level": "quiet",
        "mood": "calm"
      }
    }
  ]
}
```

## 5. Resolve Best Location Scene Asset

```txt
POST /v1/places/{place_id}/resolve-scene-asset
```

### Auth

`images:read`

### Request

```json
{
  "scene_state": {
    "time_of_day": "night",
    "weather": "clear",
    "crowd_level": "quiet",
    "mood": "tense",
    "place_state": "normal"
  },
  "allow_background_generation": true,
  "allow_blocking_generation": false
}
```

### Response

```json
{
  "asset": {
    "asset_id": "asset_place_456",
    "variant_key": "night_quiet",
    "match_score": 0.84,
    "url": "https://cdn.example.com/assets/asset_place_456.webp"
  },
  "background_job": {
    "job_id": "job_789",
    "status": "queued",
    "reason": "better_tense_night_variant_missing"
  }
}
```

## 6. Generate Artifact Visual

Existing artifact generation can remain, but request metadata should indicate whether the asset is for inventory, Aux Context Sidebar, or detailed inspection.

```json
{
  "artifact_id": "artifact_red_gem_necklace",
  "artifact_role": "inventory_item",
  "display_context": "aux_current_sidebar",
  "visual_priority": "scene_relevant",
  "known_to_user": true
}
```

## 7. UI Inspection Flag

For UI-driven requests, the core app should be able to flag that a request is non-canon inspection.

```json
{
  "interaction_context": {
    "interaction_type": "ui_inspection",
    "canon_write_expected": false
  }
}
```

The Image Platform does not write canon, but this metadata helps logs, analytics, and downstream integration avoid treating visual browsing as world-changing activity.
