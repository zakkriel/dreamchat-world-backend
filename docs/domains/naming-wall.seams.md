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
| consumes | **Canon & Time / canon-spine** (WE-1) | `payload.spoken` — the words themselves, distinct from the referee's account | Both commit doors write it (`schema.sql:275-279`, `:569-572`, Communicated only, else `{}`). Spoken words are the ONLY teacher and one of the two corpus prose sources (`tech.md` §Teaching, §The definition). The wall never reads the summary to teach — that correction is `SPEC-033`'s sharpest rule. |
| consumes | **Perception** (WE-3) | the perception write, as the moment of application | WE-3 decides who gets a row; "the viewer could hear it" IS "the fan-out wrote this holder a row" — the wall never re-decides audibility, and when the overhearer rule lands, hearing-teaches follows it for free (`docs/open-spec-items.md` §SPEC-033). |
| consumes | **World genesis** (WE-10) | canonical names and descriptors in `entity_registry` and `*_state` attrs | A missing descriptor makes `fn_display_name` fall through to the canonical name — *"a naming-wall breach by default"* (`core/api/worldgenesiscommit.go`, quoted in `digest/S13a` §14). Genesis owns seeding them; the wall does not invent placeholders (`schema.sql:2995-2997`). |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Perception** (WE-3) | `fn_viewer_text`, applied on every perception write | Perception applies it; the wall owns the predicate and the substitution. **Perception never performs name substitution; the wall never decides who perceives** (matching `perception-and-knowledge.seams.md`, provides→WE-4). WE-3's map entry currently also claims `fn_viewer_text` in `@functions` — overlap recorded for the moderator, not resolved here. |
| provides | **Play loop** (WE-7) | the belt: `Violations` in narration validation, `Scrub`/`scrubAll` at emit (`tech.md` §The two application points has the call sites) | The play loop calls the belt and never re-implements the predicate — the belt itself reads the SQL definition or *"the check is theatre"* (`D-6`; migration `20260809090006`). Also the candidate whitelist's labels: `fn_display_names_distinct` disambiguates the whitelist itself, not just the display list, because it is the vocabulary the player's next sentence binds against (`beathandler.go:421`; `digest/S13a` §14). |
| provides | **NPC cognition** (WE-8) | per-seat viewer-relative labels | Batch prompts name a thing only when every mind agrees (`fn_batch_display_name`); each isolated NPC reads the room as SHE knows it; ids never change, only labels. Cognition never re-derives a label (agreed with WE-8's package, 2026-08-27). |
| provides | **Compendium surfaces** (UX-1) | walled perception content, plus `fn_perceived_name` / `fn_display_name` for entries and headings | Surfaces render what the seam already rewrote and the labels these functions return; they never re-resolve a name, and a NULL perceived name is normal and permanent. A surface reaching `entity_registry.canonical_name` directly is a `B-1` breach (agreed with UX-1's package, 2026-08-27). |
| provides | **Referee — deliberately nothing** | — | Resolve is truth-side and licensed to read canonical names (`core/api/wall_test.go:22-27`). "Walling the referee" is a bug, not a hardening. |

## The seams that do not exist

- **A group naming wall.** Nothing walls a collective; its canonical name is speakable at tick 0
  (`product.md` §deliberately-not-built is the one home). An agent asked to conceal a group's name is
  being asked to author a new mechanism, and must say so.
- **Teaching on the ruled path.** `apply_ruled_event` renders through `fn_viewer_text` but never
  writes `name_knowledge` (`tech.md` §Traps). Until the SPEC-038-shaped fix, names spoken through
  that door are unlearnable.
- **The overhearer rule.** Co-present overhearers are the documented §3 deferral; hearing-teaches is
  bolted to the perception row, so it follows automatically when WE-3 lands that rule — do not build
  a wall-side audibility test in the meantime.
