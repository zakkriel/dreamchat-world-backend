# ADR-P023: An art style is a module, and a style is a named profile

**Status:** Accepted (2026-08-20)
**Date:** 2026-08-20
**Series:** Platform / Images (ADR-P###, per D-5). Extends ADR-P021.
**Owner of decision:** founder — five presets or your own, chosen at world creation, built to be reused.
**Evidence (D-9):** `core/api/artstyle.go`, `core/api/worldartstyleshandler.go`,
`core/api/schema/art_styles.v1.schema.json`, migration `20260815150001_world_art_style.sql`,
`core/api/artstyle_test.go`, `core/api/worldartstyleshandler_test.go`.

## Context

Every world drew in one hardcoded house style — the literal `"dreamchat-default"`, twice. The `mood`
and `ornament` genesis authors for each world never reached the thing drawing it.

The catalogue is about to be reused hard: the same looks will front creation, per-character overrides,
redraw-in-another-style, and whatever comes next. A second copy of "anime means cel shaded" in a
frontend constant is how those surfaces start disagreeing about what a world already chose.

## Decision

1. **One module owns it.** `core/api/artstyle.go` holds the keys, labels, blurbs, the prose sent to a
   model, the latitude, and the resolution rules. Every other file asks it. Presets:
   `anime`, `realistic`, `manhwa`, `comic`, `3d`; plus `custom:<prose>`; plus the house fallback for a
   world that never chose.
2. **A style is a named profile, not a per-request prompt.** The platform keys its reuse cache on
   `style_profile_id`, so a style must be stable and addressable or nothing is ever reused. Presets
   share one profile each. A written style is named by the **hash of its prose**, so identical
   descriptions share one profile and one cache in any case or whitespace, and different ones do not.
3. **Stored on the world** as `world.art_style` — the raw choice, NULL when none was made. Resolved at
   fill time, so the day a style can be changed after creation, this path already picks it up.
4. **Validated before the seat call.** An unusable key is the one refusal that can cost nothing;
   authoring a whole world to then report a typo would spend a build to do it.
5. **The catalogue is served, never hardcoded.** `GET /worlds/art-styles` (`art_styles/1`) carries key,
   label and blurb — and deliberately NOT the prompt prose. `ArtStyle` keeps its look unexported, so a
   client picks by key and what that key means to a model stays tunable without shipping a frontend.

## Consequences

- A world with no choice keeps the existing `dreamchat-default` profile, so nothing already
  illustrated is re-keyed and no existing art is orphaned.
- Adding a look is one entry in `artStylePresets`. Do not add it anywhere else.
- The route is matched FIRST in `newRouter`: dispatch is first-match-wins and an id matcher would
  otherwise read `art-styles` as a world id.
