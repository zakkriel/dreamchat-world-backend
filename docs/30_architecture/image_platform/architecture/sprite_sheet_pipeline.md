# Architecture Addendum — NPC Sprite Sheet Generation and Slicing Pipeline

## 1. Purpose

This document defines the technical pipeline for generating one NPC expression sprite sheet and slicing it into reusable expression assets.

The goal is to reduce image-generation calls while improving visual consistency across NPC expressions.

## 2. Pipeline Overview

```txt
DreamChat Core
  -> request NPC expression pack
Image API
  -> validate visual identity
  -> compile sprite sheet prompt
  -> create generation job
Worker
  -> provider generates one sprite sheet
  -> store original sheet
  -> slice sheet by grid contract
  -> store each expression asset
  -> create metadata and fallback map
  -> mark job preview_ready / ready
UI
  -> request best expression asset at runtime
```

## 3. Sprite Sheet Contract

Recommended contract:

```json
{
  "layout": "grid",
  "rows": 2,
  "columns": 5,
  "cell_count": 10,
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
  ],
  "safe_margin_px": 24,
  "allow_text_labels": false,
  "allow_internal_frames": false
}
```

## 4. Slicing Algorithm

Given image dimensions `W x H`:

```txt
cell_width = W / columns
cell_height = H / rows

for row in rows:
  for column in columns:
    x = column * cell_width
    y = row * cell_height
    crop = image[x:x+cell_width, y:y+cell_height]
    apply optional safe margin crop
    write expression asset
```

The algorithm should fail or request manual review if:

- dimensions are not divisible enough for the contract
- face detection finds no face in a cell
- face detection finds multiple faces in a cell
- blank/low-detail cells are detected
- content violates provider or platform safety policy

## 5. Quality Checks

Minimum automated checks:

- file exists and is readable
- expected number of cells created
- each cell has sufficient visual entropy/detail
- each cell likely contains one portrait/face, if face detection is available
- output dimensions meet UI minimum
- no cell is empty or near-duplicate unless expected

Optional later checks:

- face embedding similarity to base portrait
- CLIP similarity to expression label
- style consistency score
- manual review queue for failed sheets

## 6. Asset Records

The original sprite sheet should be stored as a `visual_asset` with:

```json
{
  "asset_type": "character_expression_sheet",
  "variant_key": "expression_sheet_v1",
  "metadata": {
    "sheet_layout": "2x5",
    "expression_order": ["neutral", "warm", "amused", "suspicious", "angry", "afraid", "sad", "surprised", "focused", "exhausted"]
  }
}
```

Each slice should be stored as a normal reusable asset:

```json
{
  "asset_type": "character_expression",
  "variant_key": "expression_suspicious_front",
  "parent_asset_id": "asset_sheet_123",
  "metadata": {
    "expression": "suspicious",
    "angle": "front",
    "crop_index": 3,
    "crop_box": {"x": 1536, "y": 0, "width": 512, "height": 512}
  }
}
```

## 7. Runtime Selection

The DreamChat Core should request an expression asset with a desired expression and fallback policy.

The Image Platform should return:

1. exact expression match
2. closest expression from fallback map
3. neutral expression
4. base portrait

Runtime selection should not trigger generation by default.

## 8. Provider Considerations

Not every provider/model will follow a grid contract reliably.

The provider adapter should expose capability metadata:

```json
{
  "supports_sprite_sheet": true,
  "supports_identity_reference": true,
  "supports_image_to_image": true,
  "max_output_width": 2048,
  "max_output_height": 1024
}
```

If a provider cannot produce reliable sprite sheets, use separate expression generation as fallback.

## 9. Failure Modes

| Failure | Behavior |
|---|---|
| Provider does not follow grid | Mark sheet as `failed_quality_check`; retry or fallback to separate expressions |
| One or more cells invalid | Store valid cells; mark missing expressions; use fallback map |
| Character drift too high | Reject sheet or require manual review |
| Sheet generation too slow | Return base portrait first; continue sheet generation in background |
| Cost spike | throttle pack generation and prefer smaller PoC pack |

## 10. Implementation Recommendation

PoC:

- support one fixed 2x5 contract
- deterministic slicing only
- simple quality checks
- manual review if needed

V1:

- support capability-aware provider routing
- store parent/child asset relationships
- use fallback expression mapping
- add visual similarity checks
