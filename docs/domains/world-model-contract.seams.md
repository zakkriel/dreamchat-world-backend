# world-model-contract · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-11 · The world-model contract and depth ·
**Parent bounded context:** World Engine

A seam belongs to two domains, so it gets its own file. Each row declares an expectation — one side
owns a fact, the other consumes it and must not re-derive or re-decide it.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **World genesis** (WE-10) | the filled document | Genesis fills the document; this domain owns what the document IS (mirrors `world-genesis.seams.md`'s own row). The pipeline, the lanes, the kickstart and the commit are WE-10's (`ADR-P026`); the schema's machine representation exists for `world_genesis/1` only — the `world_model` artifact is unbuilt (`tech.md` §What is live). |
| consumes | **The trials** (git-history, not a domain) | evidence | Reasoning, never state. A trial killed a version or it did not; nothing else in one binds anybody (`tech.md` §Traps). |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **World genesis** (WE-10) | the author's half — obligations and refusals | Genesis obeys them, never redefines them; a document is refused whole with the reason named. Validity and sufficiency are two bars, both checked. |
| provides | **Play loop** (WE-7) / **Time** (WE-6) | `tension` ⇒ a beat budget | The one fully WORKING reader obligation on the play side (audit row 10; `tension.go:28-43`). Play consumes the budget; it never defines the contract's vocabulary. Absence of `tension` is a refusal under v7 O12, never a silent `none` (`SPEC-030`'s lesson, `tech.md` decisions table). |
| provides | **Space & Journey** (WE-5) | `extent_class` ⇒ metres | The author picks the class; the engine owns the number (`fn_extent_class_metres`). The resolver's fail-open default is this domain's top trap (`tech.md`, the one home); a generic fail-closed resolver is increment 2's, not a local patch. |
| provides | **Perception** (WE-3) | reader obligation row 19 — a channel never discloses `hiding` | Holds **by construction** in the shipped engine (audit row 19), and the closed decision reads: build on it, do not disturb it. |
| provides | **Seats & the LLM bridge** (WE-13) | `excluded[]` — negative canon binding every authoring seat, for the life of the world | **Nothing enforces it yet** (audit row 24: no key in any seat schema, no column, no check — only a global content floor in prompt prose, identical in nine files, that nothing checks output against). A seat treating that floor as the per-world `excluded[]` is mistaking a service rule for world law. |
| provides | **the narrator, at beat time** | `WorldStatement` — the world's global statement | Owned here (`core/api/worldstatement.go`; confirmed unclaimed by play-loop and presentation, 2026-08-27). Sourced from the COMMITTED document's own content, **never `world.brief`** — the founder-gate rule, tested two ways (`worldstatement_test.go`). `world_model/6` replaces its five fields with the richer statement; the struct is the seam that absorbs it. |

## The seams that do not exist yet

- **The machine artifact and the validator.** No `world_model` document has ever been validated; the
  landing framework (Declaration/Landing, leaf coverage, the runner owning ids/ticks/classes/
  provenance) is designed in the deleted depth PRD and increment-1 plan, and none of it is code. An
  agent writing "a quick validator" is starting increment 1 — say so and load the plan from git
  history first (`tech.md` §Where the corpus lives).
- **`excluded[]` enforcement.** See the seats row above — the seam is named, the surface is absent.
- **Growth during play.** The contract must say what may be invented after genesis and what may
  never be; it does not, deliberately unscoped (product.md §8). An agent hitting it is deciding
  something new.
