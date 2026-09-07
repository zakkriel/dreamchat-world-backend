# naming-wall · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-4 · The naming wall ·
**Parent bounded context:** World Engine

A seam belongs to two domains: one side owns a fact, the other consumes it and must not re-derive or
re-decide it. `naming-wall.product.md` holds what the domain means; `naming-wall.tech.md` holds how
it is built.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Canon & Time / canon-spine** (WE-1) | source words and accepted `speech_perception` | Both commit doors preserve the source and call the same application function. A canonical word is not itself a recognition judgment (`ADR-038`). |
| consumes | **Perception** (WE-3) | holder-specific `spoken`, accepted associations, and source event | Explicit owner recognition grants sourced name knowledge and a subject link. Descriptions can coexist with recognition; related subject links never identify the owner. The naming wall never re-decides attention. |
| consumes | **World genesis** (WE-10) | canonical names and descriptors in `entity_registry` and `*_state` attrs | A missing descriptor makes `fn_display_name` fall through to the canonical name — *"a naming-wall breach by default"* (`core/api/worldgenesiscommit.go`, quoted in `digest/S13a` §14). Genesis owns seeding them; the wall does not invent placeholders (`schema.sql:2995-2997`). |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Perception** (WE-3) | `fn_viewer_text` for account prose | Apply the identity wall before adding the holder's accepted heard words. `fn_unheard_names` separately permits already-heard words at the output boundary (`ADR-038`). |
| provides | **Play loop** (WE-7) | the belt: `Violations` in narration validation, `Scrub`/`scrubAll` at emit (`tech.md` §The two application points has the call sites) | The play loop calls the belt and never re-implements the predicate — the belt itself reads the SQL definition or *"the check is theatre"* (`D-6`; migration `20260809090006`). Also the candidate whitelist's labels: `fn_display_names_distinct` disambiguates the whitelist itself, not just the display list, because it is the vocabulary the player's next sentence binds against (`beathandler.go:421`; `digest/S13a` §14). |
| provides | **NPC cognition** (WE-8) | per-seat viewer-relative labels | Batch prompts name a thing only when every mind agrees (`fn_batch_display_name`); each isolated NPC reads the room as SHE knows it; ids never change, only labels. Cognition never re-derives a label (agreed with WE-8's package, 2026-08-27). |
| provides | **Compendium surfaces** (UX-1) | walled perception content, plus `fn_perceived_name` / `fn_display_name` for entries and headings | Surfaces render what the seam already rewrote and the labels these functions return; they never re-resolve a name, and a NULL perceived name is normal and permanent. A surface reaching `entity_registry.canonical_name` directly is a `B-1` breach (agreed with UX-1's package, 2026-08-27). |
| provides | **Referee — deliberately nothing** | — | Resolve is truth-side and licensed to read canonical names (`core/api/wall_test.go:22-27`). "Walling the referee" is a bug, not a hardening. |

## The seams that do not exist

- **A group naming wall.** Nothing walls a collective; its canonical name is speakable at tick 0
  (`product.md` §deliberately-not-built is the one home). An agent asked to conceal a group's name is
  being asked to author a new mechanism, and must say so.
