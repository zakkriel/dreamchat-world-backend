# World creation — handover

**Written:** 2026-09-09 · **For:** the next agent taking over world creation.
**Scope:** everything about the new genesis pipeline — what it is, what shipped, what is open, and the
four things that will bite you in the first hour.

Read `AGENTS.md` first, then this. This file does not restate the law; it tells you where the work
stands and which of its claims you should not trust without checking.

---

## 0. STOP — four things before you touch anything

### 0.1 Six migrations are pending in production, and the next deploy refuses to boot

```
[ ] 20260907120000_strict_movement_modifiers.sql
[ ] 20260907120100_movement_feasibility.sql
[ ] 20260907120200_shared_movement_clock.sql
[ ] 20260907120300_active_movement_position.sql
[ ] 20260907140000_object_transfer_gate.sql
[ ] 20260909120000_key_opens.sql
```

Measured 2026-09-09: `Applied: 71, Pending: 6`. Production is **Online** only because the deployed
binary predates them and therefore does not require them. **`backend:ADR-P020` makes the service
refuse to boot on drift**, so the next deploy dies unless these are applied first. That exact mistake
took production down for **nine hours** on 2026-09-01 when PR #128 merged with one unapplied migration.

The order is not optional: **apply config → apply migrations → merge → watch it boot → exercise the
path.**

### 0.2 Another agent is mid-flight in this working tree

At handover the tree is on branch **`engine/authored-rules-in-play`** (identical to `origin/main`, 0
ahead / 0 behind) with **65 modified and 82 untracked files** that are **not mine and not committed**.
The six migrations above are among the untracked files — they exist on disk and in no commit.

**Never `git add -A` or `git add .` in this repo.** Stage explicit paths only. If you find work you did
not do, it is someone else's; leave it.

### 0.3 Two of this session's own diagnoses were wrong, and both are recorded

You will read confident prose in `docs/` that was corrected after being implemented and reverted. The
corrections are dated and in place, but the pattern is the warning: **claims in these documents were
derived from partial reads and shipped before being checked.** When a doc asserts what a function
does, follow it to the consumer before building on it. Details in §5.

### 0.4 `make schema-check` wipes the dev seed, and the Go suite pollutes it

`make schema-check` runs `db-down db-up migrate` with **no reseed step**, so every seed-dependent
pgTAP test fails afterwards until you `make reset`. Separately, running the Go suite mutates the dev
DB, so pgTAP run **after** Go will report ~3 spurious failures in the Drowned Lantern space tests.
**Always `make reset` immediately before `make test`.** Neither is a bug in your change.

---

## 1. What world creation IS now

Genesis is no longer one call. The old single-shot `world_genesis` seat is still registered and
**never called**; the live path is:

```
brief
  → (Custom lane only) in-fiction interview
  → world_understanding  mints world_identity/1        ← the world's condition, bargain, rules
  → world_fill           runs in CODE-SCHEDULED waves under that identity
  → world_fill_review    names breaches; tagged pieces are retracted
  → genesisDoc.validate()  (the belt)
  → commit (two transactions) + world.world_identity jsonb
  → kickstart → play
```

**The identity governs the fill.** That is the whole point of the redesign: the understanding pass
decides what kind of world this is, and every fill call is scoped by it.

**Fill wave order** (founder-ruled 2026-08-28, and it is the order in code):
`concepts → scaffold-1 → scaffold-2 → geography → canon → factions → people → arrival → closing`,
with canon **its own layer between the places and the lives** (`PR #175`), because a person authored
after their location exists can reference where they live, and canon written before the lives means a
person's history is not invented later to fit a personality.

**Seats in production:** `world_understanding`, `world_fill`, `world_fill_review` all resolve through
`DREAMCHAT_SEAT_DEFAULT` to `deepseek-v4-flash`, with `json_object` mode and an 8192 token ceiling set
explicitly (`ADR-P024` — seat config is part of the release and **nothing checks it**).

---

## 2. What shipped

### 2.1 The pipeline itself (PRs #128 → #183, all merged)

| PR | what it settled |
|---|---|
| #128 | the pipeline: identity → fill → commit, replacing one-shot genesis |
| #167–#170 | loading window, `location` as the one word, asset reuse, relevance-to-level, late-arrival settling |
| #171–#174 | cross-reference snapping, reconciliation of every reference, one ordered refusal surface, the compiled-mandate zero-value bugs |
| #175–#179 | **canon as its own layer**, measured; the critical path reported; two optimisations reverted because measurement rejected them; canon made visible and promoting |
| #180–#183 | `SPEC-048/049/050`, objects and concepts authored, the GA-2 vocabulary sweep |

**Seven live Andantes builds** drove this, each refusing on a different field before one committed.
The committed world (`52fae075`, and later `464550bd`) is playable, narrates in Spanish, and its
authored entities appear in play. **Measured: 26 calls, 15.6 min, $0.058, 71 entities.**

### 2.2 Concepts (PR #187, merged inside #188)

A world's ideas — *Auscultation: the craft of reading a beast's health through its deep pulse* — were
authored every build and **discarded at the commit seam** (`concepts=7` in the document, **0** rows in
`entity_registry`). They now register as `entity_kind='concept'`, with `entity_registry.descriptor`
carrying `what_it_is`: **the descriptor IS the truth** — one field, one meaning, authored identity,
never spoken to a character. No state row; a concept has no position and cannot act.

**Zero DDL** — `entity_registry.entity_kind` is free text with no CHECK.

### 2.3 Unwitnessed canon and `indirect` (`ADR-037`)

The genesis belt refused any event nobody witnessed. **`ADR-005` had always permitted it** — *"One
canon event fans out to **zero-to-N** perceptions"* — so the belt contradicted the founding ADR, and
`SPEC-040`'s claim that a superseding ADR was needed was itself wrong. Removing the refusal was a bug
fix. The player floor is untouched and still refuses.

`epistemic_type` gained **`indirect`**: knowledge perceived **through a medium** — a recording, a
spell, a dream. Distinct from `told` (a person's *mind* stands between you and the event) and
`inference` (your own *reasoning* does). Reliability stays out of the vocabulary: `confidence` and
`distortion_level` already exist, so a dream is `indirect` at low confidence and a tape at high.

### 2.4 The art outage, fixed the same session

Every picture in the product was blank while **544 assets sat `ready` in storage**. Not a generation
failure — a **starvation** failure. BFL's account hit `402`, the adapter could not tell an unpaid
invoice from a `429`, and the art reconciler re-commissioned the doomed owners every two minutes:
**875 failed jobs in 24h**, draining the shared **1000 requests/hour** token budget that the asset
**read** path spends. One provider's unpaid invoice hid every picture the other, paid provider had
rendered.

Fixed across five PRs (backend #185/#186, platform #58/#59/#60, workspace #20/#21): `402` is terminal
but **walkable**; `BFL_SAFETY_TOLERANCE` is sent explicitly; `fal_t2i` added as a **second** scene
route at priority 150; resolver availability derives from the **registry** (a hand-written list had
made `fal_dev`'s and `fal_t2i`'s routes reconcile `valid` and be silently unselectable); and the
reconciler skips terminal refusals. Token limits raised 60/1000 → **300/20000**. Verified: 11/11 world
covers and 6/6 sprites download real bytes.

---

## 3. The knowledge design — settled, and mostly NOT built

This is the live design thread. Founder-settled over a sparring session; the document is
`docs/design/2026-09-02-concepts-as-knowledge.md` and the deferred system is **`SPEC-051`**.

### 3.1 Three quantities and one law

| layer | what it is | changes? |
|---|---|---|
| **the truth** | what the idea actually is — authored **identity**, not canon. Never spoken to any character. | no |
| **a position** | one written account of it, **shared** — written once, pointed at by many characters (1:n:n) | no |
| **a grade** | how **fluently** a given character holds it | constantly |

**Grade is not closeness to truth.** That was considered and discarded: truth is permanently
obscured, so a character can be right and never know it. Grade is *investment* — how likely you are
to **fail at referencing** what you hold.

**One law**, reusing `core/api/pressure.go`'s existing machinery:

```
chance = f(grade)
roll   = hash(world, tick, holder, position)   -- pure; never math/rand (I-1)
fired  = roll < chance
```

**Two outcomes by degree:** recall thins first, application fails at the bottom. Reaching for a rival
branch's answer was considered and **rejected**.

### 3.2 Two axes that must never be conflated

| axis | measures | who sees it | affects |
|---|---|---|---|
| position ↔ truth | how **complete** the account is | nobody, ever | what is *possible* |
| character ↔ position | how **fluently** it is held | rendered as prose | how *reliably* it is managed |

The founder's worked example settles it. Pyromancy's truth is *elemental power, focused*. One branch
says emotion-driven, another rune-driven. **Both cast fire. Both work.** So a position is not false,
it is **partial**. A shaky apprentice holding a perfectly good branch still fizzles — different axis.

**Consequence: nothing stores "is this wrong."** No flag. A stored wrongness flag would eventually
reach a prompt and the irony would collapse.

### 3.3 Divergence, and what the models are told

A fork is a **canon event** — someone realised something, and that is an action. The old position
survives for whoever still holds it. Holders are a **plain list, not a `group` entity**: everyone who
happens to believe something is not an organisation and must not act like one.

**The seat gets prose, never numbers.** Not `depth: 4`, but *"your knowledge of the topic tells you
silence is death."* **The seat is never told its character is wrong** — the truth surfaces only when
it clashes with the position *and* the character infers a new understanding, which is a fork, which is
an event.

### 3.4 Membership does NOT grant knowledge (founder ruling, `SPEC-051` item 8)

Founder: *"Not sure that if belonging to a faction or group makes you automatically have that
knowledge… it sounds more like a character creator / validator to check what position does the
character have in that faction and assign knowledge accordingly."*

**`B-2` backs it:** the valid knowledge paths are observation, told, record, broadcast, inference,
propagation, common knowledge. ***Belonging* is not among them; being *told* is.** Knowledge is
assigned by the character's `standing` at creation, joining or promotion, using `told` / `taught` /
`granted`.

Four reasons, all of which the rejected model failed:
- **Rank** — a novice and the Auscultadora Mayor are both members; live visibility gives them identical knowledge.
- **Leaving** — live visibility makes expulsion instant amnesia. A disgraced ex-member who still knows the secret is a story; one who forgets it is a bug.
- **Provenance (`I-2`)** — one row with many readers cannot answer who knew what, when, how.
- **The leak stops being a leak** — see §5.1.

**`publishes` vs `buries`**, settled by asking how a real faction behaves: a catechism is *public* (a
heretic knows the official line perfectly, which is often why they are a heretic), while what a closed
synod decided is known *by rank*. Both fields are already authored per faction, and the split is
already drawn — the Auscultator College `publishes` "The Monthly Report" and `buries` "inconvenient
readings, especially from junior Auscultators", while The Apprentice's `standing` reads *"trusted to
take readings but not to interpret them."* **The assignment rule is written in prose before any code
exists.**

### 3.5 The reveal path (from `SPEC-040`, settled shape, unbuilt)

Founder: *"NOTHING ever links to cannon but perceptions. if a recording shows something, it shows the
perception."* **The schema already enforces it** —
`perception_record_source_event_id_fkey` is the only reference into canon anywhere and it is
`NOT NULL`.

So the medium **holds a perception**, which is already legal (`holder_id` has no FK and no kind check,
the same trick the `"Common Knowledge"` faction pseudo-entity uses), and whoever reaches it acquires
theirs `indirect`:

```
E        car burns in an empty garage               (canon, no perceptions)
P_cam    holder = the camera artifact, of E         (invisible to everyone)
P_her    holder = her, of E, INDIRECT               (acquired now, valid at E's tick)
         provenance_edge: P_her ← P_cam  (derived_from, source_kind='perception')
```

`fn_visible_perceptions` returns only perceptions held by the viewer or by a faction/group, so an
artifact's perception stays invisible until something grants you your own — exactly right for a sealed
tape.

---

## 4. What is MISSING — ordered

### 4.1 The knowledge table (the next real round)

Positions shared 1:n:n, membership carrying a grade, append-only with validity windows (`B-5`,
`ADR-006`), forks as canon events, and `granted` + `taught` added to `epistemic_type`.

**Nothing reads a `concept` row today.** It is an inert row waiting for this. Needs **one engine ADR**
(a new core table, the route `ADR-035` took, per `D-5`).

### 4.2 Factions are still discarded, and are now unblocked

Six factions per world, authored with **seven fields of substance** — `kind`, `goal`, `sacrifice`,
`seat`, `controls`, `publishes`, `buries` — and **zero rows** reach the engine. `belongs_to` is parsed
into `genesisActor.BelongsTo` and **never persisted**; there is no membership table anywhere.

The founder's ruling (§3.4) removed the blocker that was stopping this. What remains: a membership
representation, and whether it carries a **coarse rank** beside the prose `standing` (nothing can
currently query *"who is senior in the Weight Guild"*).

### 4.3 The reveal, and why it must not be built first

`generate_perceptions` derives holders from `event_participant` and hardcodes
`acquired_tick = valid_tick = the event's tick`, so it **cannot grant a late perception to someone who
was not there**. `provenance_edge` is declared and **written by nothing** — though its `source_kind`
already admits `'perception'` and its `how_type` already carries `witnessed_by`, `reported_by`,
`inferred_from`. And **no action exists** for watching a tape or casting a divination.

**Build the action first.** A minting path with no caller is dead code — the same trap that made us
defer the positions table.

### 4.4 The other open specs

| spec | state | note |
|---|---|---|
| `SPEC-041` | OPEN | perceptions are **replaced, never mutated**. Late perceptions make this live: if you overwrite what someone believed, you cannot explain why they acted. |
| `SPEC-042` | OPEN | traits (trauma, belief) not linked to the perceptions that formed them. Depends on 041. |
| `SPEC-050` | OPEN, measured | **canon does not scale with depth**: 35 locations and 76 people at depth 3, still **12 events**. Same history for 3.5× the world. Removing the holder requirement (§2.3) should make events *cheaper* to author, so re-measure before redesigning. |
| `SPEC-048` | OPEN | the fill's shape must be predictable while its prose must not be. |
| `SPEC-036` | OPEN, deferred | a world's own rules have no enforcement path. |
| `SPEC-051` | OPEN, BIG | the concept/knowledge system — **eight** questions, and the standard to hold it to. |

### 4.5 Never done, deliberately

- **No live Andantes fill-quality probe** against the unchanged paper identity (`docs/design/2026-08-27-understanding-pass-probe/U_andantes_identity.md`). The founder asked for it repeatedly and it kept being displaced. Fill quality has only ever been judged from committed worlds, not from that probe.
- **Custom §6 in-fiction question quality** was never revisited.
- The old `world_genesis` seat is still registered and unbound-removal is a later PR.

---

## 5. Claims in the docs that were WRONG, and how they were caught

Read this section before trusting any diagnosis in `docs/`.

### 5.1 "The faction visibility branch is a leak" — false

A task was written, implemented, and reverted. The `faction`/`group` branch in
`fn_visible_perceptions` is **the common-knowledge implementation**: `B-2` names common knowledge a
valid path, the glossary defines it, and `core/db/seeds/seed_mara_0A.sql:24` registers a `faction`
pseudo-entity literally named **"Common Knowledge"** as the holder of public facts. Removing it broke
**11 assertions across 7 pgTAP files**.

Caught because the implementer **reported the failures instead of rewriting the tests to pass.**

### 5.2 "The history weaver only gets concept names" — false, and the fix inverted the goal

A task replaced `scope.Concepts` with joined `"name — meaning"` lines. `scope.Concepts` is a
**selector**, matched by exact string equality in `buildWorldFillPrompt`, and the renderer beside it
**already emits** `is: <what_it_is>` and `contested: <contested>`. The joined lines matched nothing, so
the change **silently dropped every concept from the canon prompt** — with no log and no failing test.

It also recreated a format the repo had **already deleted after a live failure**: the prompt marker at
`worldidentity.go:78` reads *"Cross-reference these by the EXACT string inside the quotes and nothing
else — never the descriptor, never the quotes, never the two joined."*

### 5.3 "`SPEC-043`'s blocker is `entity_kind`'s closed set" — false

That set is on `event_participant`. `entity_registry.entity_kind` is free text with **no CHECK**, and
`perception_subject` has no kind constraint and **no FK on `entity_id`**. The real blocker is one
column: **`perception_record.source_event_id` is `NOT NULL`** — a perception cannot exist without an
event, and knowing a doctrine after twenty years has no event. That is the fact that shapes the whole
knowledge system, and the spec had it wrong.

### 5.4 "Allowing unperceived canon needs a superseding ADR" — false

`ADR-005` already said *"zero-to-N perceptions."* See §2.3.

### 5.5 Guards that did not guard

Four guards this session **survived deletion of the code they claimed to protect**. Two examples worth
internalising: a mutation script that printed `MUTANT APPLIED` **unconditionally** so a "surviving"
mutant had never been applied; and an assertion that compared a value against **the constant it was
testing** — a tautology that passes on exactly the regression it existed to catch.

**Revert the fix, watch the test fail, restore it.** Every time. `AGENTS.md` says 40 of 70 probes
survived a fully green run; that is not history, it recurred this week.

---

## 6. Where things live

| what | where |
|---|---|
| The design of record for knowledge | `docs/design/2026-09-02-concepts-as-knowledge.md` |
| The identity/understanding-pass design | `docs/design/2026-08-26-world-identity-and-the-understanding-pass.md` |
| The fill stage, with live-run corrections | `docs/design/2026-08-28-the-filling-stage.md` |
| Every open seam | `docs/open-spec-items.md` |
| Engine decisions | `docs/law/02_world_state_adrs.md` (`ADR-037` is knowledge-through-a-medium) |
| The law | `docs/law/06_rules_register.md` |
| The map — amend it in the same round | `docs/maps/system_map.md` |
| The genesis pipeline | `core/api/worldidentity.go` (schedule + fill), `core/api/worldgenesiscommit.go` (transcription), `core/api/worldgenesis.go` (the belt) |
| The deterministic roll to reuse | `core/api/pressure.go` |

**The domain package wins over the dossier (`D-6`):** `docs/domains/world-genesis.{product,tech,seams}.md`.

---

## 7. How to work here

- **`../harness/brief.sh <file>`** before you edit it, and **`--ask "<your question>"`**. It is the index; everything else fires at PR time, which is too late.
- Cite a rule ID, an ADR, or a line of code — or say plainly you have a preference, not a constraint. A backend PR citing an id that does not resolve **fails the build**.
- **Never `git add -A`.** Parallel writers, always.
- Amend the map in the **same** round. The areas gate will also refuse a new file that belongs to no area — add a glob to `docs/maps/AREAS.map`.
- The founder's standing constraint on mechanics, verbatim: *"do not go creating crazy gates and crazy test and validations in the code and call them a system. we want it to keep it clean as (speed, velocity and the physics engine)."* Three quantities and one law is the standard.
- Ask **one thing at a time**, in the founder's own words, with the consequence of each option and a recommendation. A batch of eleven questions was rejected outright; so was a multiple-choice dialog where a spar was wanted.
- **`./stack.sh smoke` is a liveness check, not a correctness one.** An empty beat passes it identically to a real one.
