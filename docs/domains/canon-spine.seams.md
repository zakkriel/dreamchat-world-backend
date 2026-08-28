# canon-spine · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-1 · The canon spine ·
**Parent bounded context:** World Engine

A seam belongs to two domains, so it gets its own file. Each row declares an expectation — one
side owns a fact, the other consumes it and must not re-derive or re-decide it.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Play loop** (WE-7) | attempts and adjudicated rulings | Attempts arrive pre-adjudicated; the doors enforce a **structural floor only** and never re-referee. `apply_beat` delegates to `apply_event` with `p_legacy_types=true` (`schema.sql:108`); the Go layer re-validates the closed vocabulary before it reaches the doors (leash then belt — WE-7's fact). The doors' return contract is fixed: `{"event_id", "halt_reason"}`, nothing written on reject. |
| consumes | **Time and the clock** (WE-6) | `p_tick` / `p_seq`, the `in_world_label` carry | The clock owns what tick it is; the spine owns that `(world_id, in_world_tick, beat_seq)` is unique among accepted rows (`SPEC-002`, `B-5`). The spine never advances time — `apply_beat`'s `ADR-036` advance is WE-7/WE-6 machinery around the doors, not in them. |
| consumes | **World genesis** (WE-10) | pre-accepted seed and template rows, `origin='fast_path'` | The one non-door writer: `fn_instantiate_drowned_lantern` (`schema.sql:1889`) inserts backstory canon directly. Genesis must respect the spine's shape (CHECKs, append-only); the spine does not decide what a world starts with. WE-10's package is `world-genesis.{product,tech,seams}.md`. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **perception-and-knowledge** (WE-3) | a committed event | `generate_perceptions(event_id)` runs once per accepted event, inside the commit (`schema.sql:345`). Canon decides *whether it happened*; perception decides *who noticed* — the spine never widens who perceives (`SPEC-035` precedent). Known defect: the second door writes perceptions itself and disagrees; the one home is `perception-and-knowledge.tech.md` §"The second door". The doors are a co-owned seam surface — WE-3's `DOMAINS.map` entry already claims them in `@functions`, and so does this domain's. |
| provides | **Projections and the read model** (WE-2) | `state_mutation` rows, provenance-mandatory | The spine writes WHAT CHANGED with `event_id NOT NULL` (`I-2`); WE-2 derives current state and asserts replay (`I-1`, `ADR-026`). Projections are written only by the maintainer (`I-7`) — nothing on the spine side ever touches a `*_state` table except through `apply_mutation` and the door helpers. `apply_mutation` itself is claimed by both sides in `@functions`; its file glob is WE-2's to take. |
| provides | **The naming wall** (WE-4) | `payload.spoken` (`SPEC-033`) | Canon stores what was actually said, verbatim, only for speech and only when non-empty (`schema.sql:275-278`). The wall reads it as its vocabulary source (`fn_unearned_names`); the spine never applies name substitution — a summary is the referee's account, not a viewer text. |
| provides | **Every read surface** (WE-2 consumers, UX-1, WE-12's ledger) | committed rows, read-only | Consumers read `canon_event`/`state_mutation`; **nobody inserts except the doors** (`D-1`). Documented deliberate reads exist (e.g. Carrying's containment-ledger read, migration `20260809090001`) and are the reader's decision to defend, not the spine's. Trap for every reader: live rows carry legacy `event_type` labels (`canon-spine.tech.md` §Traps). |

## The seams that do not exist

Name them, because this is the section an agent will otherwise improvise into.

- **No repair seam.** `origin='compensation'` and the `accepted→retconned|superseded` transitions
  have zero writers (`canon-spine.product.md` §deliberately not built). An agent asked to "fix"
  a wrong event has no mechanism to reach for; building one is reopening `ADR-016`'s
  present-forward ruling, and must say so.
- **No proposed-lifecycle seam.** There is no validation-gate pipeline feeding `status='proposed'`
  rows and no acceptance-transition trigger (`SPEC-003`'s deferred half). The doors commit
  accepted rows directly; a consumer that waits for a `proposed→accepted` signal waits forever.
- **No bundle seam.** The causal tables accept manual test data only; no runtime path writes them
  before Phase 4 (`ADR-029`, frozen). A domain wanting "why did this happen" answers today gets
  provenance edges and mutation lineage, not bundles.
