# compendium-surfaces · product

**Repo:** `dreamchat-world-backend` · **Cluster:** UX-1 · The compendium surfaces ·
**Parent bounded context:** Compendium & Play UX

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `compendium-surfaces.tech.md` holds how it is built;
`compendium-surfaces.seams.md` holds what crosses its boundary.

---

## What this domain is for

**One job: the reference surfaces — everything a player consults between beats, rendered from what
this viewer knows and nothing else.**

Seven surfaces: the three compendium indexes (actors, locations, artifacts), the three dossiers
behind them, the timeline, plus the transcript and the Carrying overlay. Every one is
perception-bound (`B-1`): an unperceived thing is absent from the payload, never redacted or
placeholdered. The failure this prevents, per the Actors PRD: users *"either lose the thread… or
are given an omniscient database view (which destroys secrets, rumor, and discovery)"*
(`digest/S05_the_prds.md` Topic 5).

## Ubiquitous language

The definitions live in `docs/law/05_glossary_ubiquitous_language.md`; this table carries only what
each term settles for these surfaces.

| Term | Means, precisely |
|---|---|
| **Actor** | Any world participant or force with continuity — **including groups, institutions, governments** (Glossary §2). A sword, a market, a rumor, a murder are not Actors. |
| **Artifact** | Any meaningful object or asset with continuity. **Not limited to what the player carries** — a heard-about vase in an embassy qualifies. |
| **Carrying** | The possession overlay: what the user-controlled character has on them now. **Artifacts and Carrying must never merge** (Glossary §3.2): Compendium = known; Carrying = held; you can hold what you know nothing about. |
| **Collected Knowledge** | The dossier's body. The label is deliberate — knowledge, not "perceptions" — and it is **grouped by subject, never by event or moment** (see `tech.md` §Grouping for the receipt). |
| **Timeline Record** | *"A chronological display entry pointing to a Perception Version — never directly to canon"* (Glossary §2), even when perception and canon are identical. |
| **Transcript** | The one read surface that is a **RECORD, not a projection** — delivered prose, never recomputed, never retro-labelled (`tech.md` §Transcript). |
| **Decay** | Stale knowledge is **worded, never hidden** — "last known…" language, never a visibility filter (register Mechanics row; `fn_compendium_decay`). |

Genre words — inventory, loot, quest, possession — are violations here, not style choices
(`GA-2`, `F-1`). Raw engine tokens (`epistemic_type` values, ticks, UUIDs, confidence numbers) never
reach UI copy (`F-2`, `B-5`).

## What this domain is not

- **Not who knows what.** Perception & Knowledge (WE-3) decides that; this domain only renders it
  (`seams.md`).
- **Not naming.** The wall (WE-4) owns every name a surface prints; a null name is normal and
  permanent, not a loading state.
- **Not the play loop.** Beats, seats and narration are WE-7's; this domain stores and serves the
  transcript rows the loop hands it.
- **Not presentation.** Layout, styling, story-language mapping of epistemic kinds — frontend
  (`D-7`).
- **Not the correction UX.** Errors are jumped to from these pages, fixed elsewhere (`C-11`).

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `B-1` | Every page renders from the holder's perception; hidden truth is absent from the payload. | A UI-level filter is a violation even when it looks correct; `I-3` is the CI teeth. |
| `B-9` | Syntheses derive deterministically from stored versions — no regeneration on read. | Reload producing different text is drift, and it is a bug. |
| `B-3` | No relationship UI, at all — no panel, field, synthesis or label. Relationship-flavored information arrives only as ordinary sourced knowledge records. | Rebuilding the struck "Relationship to you" card reopens a founder ruling (rev 2). |
| `B-4` | The system never authors the player's interiority — no trust meters, no "you feel". | A surface stating what *you* believe overwrites the one reading that is the player's. |
| `B-5` | Ticks order; authored labels display; wall-clock never crosses the boundary. | "Seen 1h ago" was struck from the mockups for exactly this. |
| `C-4` | Play mode shows the perceived world; only creator/debug may show authoritative state. | The `?viewer=` override outside debug is a `B-1` breach with extra steps. |
| `C-12` | One hierarchy expression per Location page. | Breadcrumb + tree + panel at once — the mockup shows both; the law says pick one. |

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No relationship surface.** `B-3`/`B-4`, two independent reasons (unearned knowledge; authored
  interiority). The modeling exists; the surfacing is removed on purpose.
- **No "What is Known" summary panel.** Artifacts PRD: *"becomes redundant… Collected Knowledge is
  the source-of-truth UX for what is known"* (`digest/S05_the_prds.md` Topic 4).
- **No nearby objects in Carrying.** Needs a scene-object affordance mechanic that does not exist;
  nearby things appear as lens prose, never as possession (`digest/S05` Topic 8).
- **No Carrying for NPCs.** Unreachable by construction — no carrier argument exists (`seams.md`,
  provides-row for the play surface).
- **No `contextual_actions`, no `open_full_artifact_link` in `carrying/1`.** Presentation is the
  frontend's (`D-7`); the entry's `id` IS the link — a shipped route would hardcode the frontend's
  URL space (migration `20260809090001`).
- **No day structure on the timeline.** Parsing "Day 3" out of a free-text label is the client
  deriving world truth (`D-7`); grouping by *runs* of identical labels is the legal alternative
  (`digest/S11_frontend.md` Topic 18).
- **No timeline gap-filler.** "No records yet" renders as unknown, never invented content
  (Timeline PRD AC#5, `digest/S05` Topic 9).
