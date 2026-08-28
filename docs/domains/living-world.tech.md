# living-world · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-12 · The living world ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the write path, validation, traps — and is
ruthless about what IS versus what is only designed. `living-world.product.md` holds what it means;
`living-world.seams.md` holds what crosses its boundary.

Line numbers are as of 2026-08-27; re-locate by grep before relying on one.

---

## Storage — what exists

- **`pending_event`** `(pending_id, world_id, fire_at_tick, magnitude, payload jsonb, status)` +
  `fn_due_pending` — the ledger. Payload shape is `{actor_id, attempt}` (`core/api/ledger.go:16`).
  `status ∈ pending|fired|cancelled`; no interval, no re-arm. `pending_id` has **no DB default** —
  test inserts must supply `gen_random_uuid()`.
- **`world_actor_config`** (per-tier `climb_rate`, `climb_chunk_ticks`, `cap`) and
  **`world_actor_setting`** (`enabled`, `intensity`) — per-world data, never hardcoded.
- **`world_eruption`** — append-only fire-log, `event_id NOT NULL REFERENCES canon_event`; the
  last-eruption source pressure derives from.
- **`fn_pressure_chance(world, tier, now)`** =
  `LEAST(cap, climb_rate × ((now − max(fired_tick)) / climb_chunk_ticks) × intensity)`, `0` when
  disabled or unconfigured. **`intensity` multiplies INSIDE the `LEAST`** so the cap stays the hard
  ceiling — both rationales are in the comment block (grep `fn_pressure_chance` in
  `core/db/schema.sql`).
- **`fn_world_slice(world, scene)`** — the World Actor's world-scope payload (ledger, presence,
  locations, recent world canon, scene).
- Migrations: `20260805100002`, `20260805100003`, `20260808100001`; the sockets predate them in
  `20260723100003_personality_world.sql` (co-owned — see the .map notes).

## The write path — the world's turn

`runWorldTurn` (`core/api/worldturn.go:47`) is the per-slot composer: **ledger first** (deterministic
pre-caused truth), then — only if nothing medium/large already fired — **the roll, biggest-first**
(`large→medium→small`). `livingWorldTiers` (`worldturn.go:184`) is the single source of truth for the
tier set, `magnitudeRank`, `eruptionCutsBeat` and the scan order — a new tier is added there plus the
SQL CHECKs, nowhere else. Biggest-first because small's higher chance would otherwise mask a rarer
medium/large fire and suppress the §5 cut. At most one eruption per turn.

- `fireDuePending` (`ledger.go:159`) fires rows in `(tickBefore, tickAfter]`, commits each payload
  through the ordinary pipeline, flips `status='fired'` inside the commit's transaction. A payload
  that does not land flips to terminal `cancelled`, never retried, with a log line.
- `commitWorldPayload` (`ledger.go:54`) is the ONE routing both callers share: passthrough types via
  `applyEvent`, everything else adjudicated. Perception is the commit path's own fan-out.
- **`postCommitFn`** (`ledger.go:30`): bookkeeping that MUST ride the same transaction as the commit
  it describes — the pending flip, the fire-log row. Rung 0 deferral A; without it a crash between
  statements re-fires an event or leaves a tier permanently undrained ("the lost drain").
- The roll: `rollTier` = `deterministicRoll(worldID|tick|lastEruption|tier)` (fnv64a → [0,1)) `<`
  `fn_pressure_chance` (`core/api/pressure.go`). Pure hash of committed state; replay reproduces it.
- `runWorldActor` (`core/api/worldactor.go`): world slice in, ONE intrusion out, sized to the drawn
  tier. The runtime gate (rung 0 deferral B) accepts two shapes only — an act by an entity already
  standing in the scene, or an `ActorMoved` pulling a non-present NPC INTO the scene (this seat's
  unique power). Anything else is `errIntrusionRejected`: a refused proposal, **not** a failed beat;
  the tier stays undrained and re-rolls differently next turn.

**Call sites** — the composer is *called, never modified* (rung-2 fence): `advanceWorldTurn` in
`orchestrator.go` at the two chain Stage-4 blocks and at the instant-floor crossing (deferral C —
`orchestrator.go:505` comment; `cutBeat` deliberately ignored there), and `journey.go:375` once per
leg, byte-for-byte the same unit.

## Designed, not driven

Be suspicious of any claim this subsystem "runs": much of the cluster is specification.

- **`pending_event` is fully built and read every crossing — and written by nothing but tests.**
  Exactly three inserts repo-wide: `ledger_test.go:28`, `orchestrator_worldtime_test.go:348`,
  `101_personality_world_test.sql:58` (verified 2026-08-27, `grep -rn 'INSERT INTO pending_event'`).
  No production writer exists; genesis as the first one is a proposal with two unresolved blockers
  (no magnitude class, actorless events inexpressible — S07a T13).
- **`origin='backstage'` is a reserved enum value with zero writers** (grep `'backstage'` in
  `core/api/*.go`: nothing outside the CHECK constraints and one test seed). Backstage Updates, Node
  Decay, review triggers/queue/radius, structural depth: **no code at all.**
- **`world_pressure` and `trait_pool` are vestigial.** Created `20260723100003`, superseded by the
  derived model, touched by no code; only `101_personality_world_test.sql` asserts their shape. The
  plan's own instruction: *"leave it, don't depend on it."*
- **The diorama finding (S07a T14): every generated world is physically inert.** Genesis writes no
  `size`/`max_room`/weights, so the one state-depends-on-state chain (contained_by → weight →
  encumbrance → speed 0 → over-budget → Journey) is dead outside the hand-authored benchmark — and
  the ledger is empty at birth. The genesis-side design response is
  `docs/design/2026-08-26-world-identity-and-the-understanding-pass.md` (WE-10's; *design, not yet
  built*).

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `D-1` | Nothing mutates canon directly (`worldturn.go`/`worldactor.go` `Governed-by:` headers). The gate's "no" is an answer, not an error. | A world-truth shortcut past the doors is a second commit path. |
| `D-8` | Backstage-class work is async. | See `product.md`. |
| `SPEC-031` | **Landed.** Interruption frequency is arithmetic; the lever is `world_actor_config` data, founder-tuned by play. | Code-side "fixes" to rarity re-decide a felt-experience ruling. |
| `SPEC-032` | **Landed — and its own first diagnosis was wrong.** The waystation portal guard, not the accessibility floor, blocked road interruptions. *"Measure the thing before designing around it."* | Trusting the entry's upper half designs against a refuted cause. |

### What you may not decide alone

1. **Adding a tier** — `livingWorldTiers` + the SQL CHECKs; the cut threshold derives from it.
2. **Retuning climb/cap/intensity numbers** — founder's dial (`SPEC-031`).
3. **Giving `pending_event` a production writer** — blocked on the magnitude and actorless-payload
   rulings (S07a T13).
4. **Building recurrence** — ruled a separate engine program.
5. **Wiring anything into `applyNPCDecisions`' nil `postCommitFn`** — *"nothing may be wired into
   that hook"* (rung 0 execution corrections).

## Validation for this domain

Go (always `-count=1`; the suite is seed-dependent): `go test -run
'WorldTurn|Pressure|Ledger|WorldActor|Reaction' -count=1 ./core/api` — `worldturn_test.go`,
`pressure_test.go`, `ledger_test.go`, `worldactor_test.go`, `orchestrator_worldtime_test.go`,
`reaction_worldturn_test.go`. pgTAP: `101_personality_world*`, `103_world_pressure*`,
`104_world_slice*`. **`make reset` destroys the dev volume and must never be run.**

- **What counts as evidence here: this domain fails quiet, not loud.** A world that never interrupts
  looks identical to a broken one — SPEC-031's 0-for-14 legs was *the expected arithmetic*, and
  SPEC-032 hid behind it. Reproduce with the trace: the `world_turn` block on `reasoning_log`
  (debug-only) shows **all three rolls including the ones that did not fire** — you cannot tune, or
  diagnose, what you cannot see.
- **What counts as ceremony here:** `101_personality_world_test.sql`'s `world_pressure`/`trait_pool`
  blocks assert the shape of tables no code reads or writes — they pass with the whole subsystem
  deleted. Schema-shape assertions on sockets are this domain's vacuous test.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **A committed eruption needs its fire-log row in the same tx** or the tier never drains. | §The write path (`postCommitFn`) — the one home; whole-branch review's "lost drain". |
| **Refusal ≠ failure.** A refused intrusion once failed the beat → identical deterministic re-roll → livelock. | `errIntrusionRejected` (`worldactor.go:30`); PR #42 lesson, S08b T7. |
| **A small ledger fire does NOT skip the roll** — both firing in one turn is ordinary; the roll's commit offsets past the ledger's seq slots. | `worldturn.go` header ("task-9 review, Important #1"). |
| **The floor crossing gets a world's turn too** — skipping it silently loses pending fires in `(startTick, startTick+floor]`. | Deferral C, `orchestrator.go:505-509`. |
| **A tuning that lands green and changes nothing**: seeded defaults are `ON CONFLICT DO NOTHING`, so a config change needs the `UPDATE` for existing worlds as well. | SPEC-031's landed note (both halves). |
| **The Go suite poisons pgTAP.** `pressure_test.go:75-89` asserts an *empty* `world_eruption` precondition and drains state for what follows. | Two false regression reports in one day (exemplar tech.md trap row, re-verified against the file). |
| **At capped tiers the world erupts ~70% of runs** — quiet-world tests need `wtDisableWorldActor`; `defer` runs before `t.Cleanup`, so restore ordering bites. | S08b T10; `journey_acceptance_test.go:143`. |
| **The digest disagrees with the code on scan order** — S08b T8 says small→medium→large; code and S13a T25 say biggest-first. The code is right. | `worldturn.go:20-22,177-183`, verified 2026-08-27. |

## Open questions

1. **Where does Backstage build when it builds?** On the ledger, on the review-queue design, or its
   own machinery — unstated anywhere. And the world's turn runs synchronously inside the beat while
   `D-8` puts backstage async: the boundary between "the world's turn" and "backstage review" has no
   written answer.
2. **The Backstage/decay/structural-depth spec has no in-repo home.** Its source files
   (`02_poc_scope_and_success_criteria.md`, `04_parked_product_concepts.md`) were deleted by the
   `workspace:ADR-W006` consolidation (`88486c1`); the digest is the only surviving carrier.
   `ADR-P026` restored the world-identity doc from the same deletion class — does this spec get the
   same treatment?
3. **`world_pressure` / `trait_pool`: drop or keep?** "Leave it, don't depend on it" is the only
   instruction on record.
4. **Genesis as the ledger's first production writer** — proposed and conceded (S07a T13), blocked
   on two rulings. Whose round?
5. **Recurrence** — the separate engine program nobody has scheduled: class-resolved interval,
   re-arm rule, and whether a recurring fire re-enters the ruled path.
