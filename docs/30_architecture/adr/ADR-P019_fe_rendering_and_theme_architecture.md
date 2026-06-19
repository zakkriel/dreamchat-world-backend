# ADR-P019: Frontend rendering model + theme architecture

**Status:** Accepted (FE Architecture working session, 2026-06-19 — chunk-6 pre-brainstorm)
**Date:** 2026-06-19
**Series:** Platform / Frontend (ADR-P###, per D-5) — does NOT touch frozen engine canon.
**Owner of decision:** Chunk 6 (play surface) — the FE seams land here; the module render engine
and the deep theme catalog land later (S4 / theme #2+).
**Governing rules:** D-14 (FE rendering model — one engine, a catalog of kinds) and D-15 (theme
architecture — neutral skeleton + swappable skins), both added to the register in this PR.
**Cites:** D-2 (module UI into named slots), D-7 (FE presentation only), GA-3 (no hardcoded genre
sections), D-4 (`schema_version` on evolving data), D-8 (images async), D-3 (Image Platform never
owns world truth), B-1 / I-3 (perception boundary).
**Evidence:** the chunk-6 mockups in `docs/20_design_ux/mockups/` — scene + Aux Current/Known/Inspect/
Intent lens surfaces (`mock_aux_lens_current.png`, `mock_aux_lens_known_actor.png`,
`mock_aux_lens_inspect_artifact.png`, `mock_aux_lens_intent.png`), the actor hero pages
(`mock_compendium_actor_seren_v2.png`), the location page (`mock_compendium_location_dawnfall_market.png`),
and the timeline spine (`mock_compendium_timeline.png`). These are the Fantasy theme #1 made concrete;
they are the bespoke shells and repeating content blocks D-14 names.

## Context

Chunk 6 builds the play surface. Two architectural questions had to be settled before the build so
the cheap seams land now and the expensive machinery (module render engine, deep theme catalog) can
defer without rework:

1. **How does the FE render backend data?** The risk is carrying two code paths for one job — a
   hardcoded path for "the surfaces we know" plus a generic path for "anything else" — which rots into
   divergence. The backend already tags data by *kind* (`epistemic_type`, group keys, participants,
   carrying) and sends no layout or look (D-7, B-1/I-3). The FE needs one disciplined way to turn a
   kind into a crafted component.
2. **How does the product look like many genres without the system knowing any genre?** GA-3 forbids
   hardcoded genre taxonomies. The mockups are unmistakably "fantasy," yet the system must never encode
   "fantasy" — it must read a world's accent/art/tokens as plain data and let a *theme* supply the look.

## Decision

### A. Rendering model (D-14) — one engine, a catalog of kinds

- **One rendering path per job.** Never a hardcoded path *and* a generic path for the same data.
- **Backend tags by kind; the FE owns the catalog.** The catalog maps each kind → a richly-crafted
  component and all presentation. The backend sends data + a kind tag only — never layout or look.
- **Known-structure vs unknown-structure is the dividing line.** UI whose structure the FE author
  knows ahead of time — every chunk-6 surface (scene, narration, input, participants, Aux
  Current/Known, carrying) — uses **native catalog components**. UI whose structure the FE *cannot*
  know ahead of time — 3rd-party **module** widgets (Stats/Battle) — uses a **generic fragment
  renderer, deferred to S4**, built when the first module needs it.
- **Two FE-owned layers** (both visible in the mockups):
  - *Bespoke, art-directed shells/layouts* — scene canvas + participant ring, actor hero page,
    timeline spine. Hand-built, singular, full craft.
  - *Repeating content blocks* — knowledge items, portrait chips, lens sections, thread rows. From the
    catalog: one kind, crafted once, reused.
- **Interactive catalog components.** Some kinds carry declared **actions** (suggested affordances,
  e.g. "Look closer"). Clicking one **submits a beat** — identical to typing the action. Rules:
  accelerators, not a cage (never replace the free-text input); never decide outcomes (submit intent
  only — D-7); suggested actions are **backend-generated data** tagged by kind — the FE renders and
  wires them.
- **Aux is push + pull.** It reacts to user focus (clicks) *and* to world events via the beat stream's
  `aux delta` frames (e.g. picking up an item surfaces its card).
- **Modules (S4).** Compose from the catalog into **named slots** (D-2); the FE never hosts module
  *code*; the render contract stays **declarative-only** so 3rd-party code is never required in the
  renderer. If/when 3rd-party *code* arrives, it runs server-side or in a WASM sandbox — decided in
  the module-architecture doc, not here.

### B. Theme architecture (D-15) — neutral skeleton + swappable skins

- **Neutral skeleton + named slots; themes change visuals only** — never structure, behavior,
  features, or requirements.
- **Themes may go deep.** Beyond recolor, a theme can swap a component's visual *form* (knowledge-item
  = gold card in Fantasy, torn typewriter strip in Horror, HUD readout in Sci-fi). Same data, slot,
  behavior.
- **Cost discipline.** *Tokens are the floor* — a theme supplying only colors/fonts gets a coherent
  recolored look for free. *Per-component art direction is opt-in* — premium themes go deep where they
  choose. Do not promise infinite cheap deep themes.
- **Two independent atmosphere layers.**
  - *World art* (scene/portrait/item imagery) = **content**, bound to the world (Image Platform; D-8 /
    D-3 — the Image Platform never owns world truth).
  - *Chrome theme* (palette, type, ornament, frames, component variants) = **swappable skin**; the
    world supplies a default, the **user can override**.
- **Genre-agnostic preserved (GA-3).** The system never knows "fantasy"; it reads a world's
  accent/art/tokens as plain data (`schema_version` + runtime validation — D-4). No hardcoded genre
  modes.
- **Fantasy is theme #1** — built first because the mockups exist and to prove the theming
  architecture. It is **not** the baseline/default; it is one theme among many.

## Consequences

- The FE has exactly one rendering discipline. The module fragment renderer is an *additional* engine
  for the unknown-structure case only, and it is deferred to S4 with no impact on chunk 6.
- The interactive-component *pattern* (action-carrying catalog components that submit beats) lands in
  chunk 6 as a seam, so Inspect's affordances (chunk 7) drop in with no rework.
- Theming is data, not code branches: a new genre look is a new theme (tokens, optionally deep
  component variants), never a system change — GA-3 stays intact.
- The render contract being declarative-only keeps 3rd-party *code* out of the renderer; the trust
  model (declarative now, WASM/server sandbox if code arrives) is owed to the module-architecture doc.
- Several small seams fall out as SPEC items (theme-token field, configurable API base + BE CORS,
  dynamic multi-world id, app shell with named slots + Aux docked↔full-screen, Electron-wrappable
  delivery) — filed in `docs/open-spec-items.md`.

## Reversibility

The catalog is additive (new kinds = new entries) and themes are data, so both grow without redesign.
The one genuinely load-bearing commitment is *one path per job* (D-14): walking it back would
re-introduce the two-code-paths divergence this ADR exists to prevent. The module fragment renderer
and any 3rd-party-code trust model remain open and are decided later (module-architecture doc, S4).
