# world_model/7 — capabilities the old engine already enforced, reclaimed as obligations

Delta over v6. Everything in v3, v4, v5 and v6 holds unless changed here.

Written because `AMENDMENT.md` §4 (`docs/superpowers/debates/2026-08-26-landing-contract-retarget/AMENDMENT.md:115-131`)
decided seven capabilities `world_genesis/1` already enforced and `world_model/2–6` silently dropped —
found by the retarget round's contracts seat (C11–C18) and ruled on by the product owner. **One concern:**
close every one of the seven before increment 1 lands, so `world_model` stops being a step backward from
the schema it replaces. Five are obligated on the author; two are derived by the runner. All seven
re-verified against the current source this session, not carried on the round's word alone.

---

## 1. `tension` — required on every `extent` entity, never optional

`world_genesis/1` requires it (`world_genesis.v1.schema.json:53`, `places[].items.required`) and refuses
it out of its five-value set (`worldgenesis.go:284-285`). `world_model/2` made it an optional key on the
`extent` facet (`SCHEMA-v2.md:28`) and no `SCHEMA-v3.md` obligation (O1–O11, `:52-64`) requires it back.

**The absence is not neutral.** `tensionBudgetSeconds` maps `"none"` to `math.MaxInt64` (`tension.go:38-39`),
and a missing tension row COALESCEs to `"none"` defensively (`tension.go:24-27,49-50`, `:58`) — "the
budget is a BLOCKER, never an award... fitting means one specific impossibility didn't fire" becomes an
impossibility that can never fire, because the budget is infinite. Every act fits every beat and nothing
becomes a journey — the SPEC-030 regression, reintroduced silently by the v2 rewrite and invisible to a
suite that never authors an entity without the key.

**Verified independently**, not on the round's count alone: of `G_grelda_by_simarch.md`'s eight `extent`
entities (`Grelda`, `la plaza mayor`, `Cuesta Menor`, `la subida`, `la Ochenta y Tres`, `la Cuarenta`,
`la casa de Ordo`, `el granero de la Junta`; `:88-145`), six carry `tension`. Two do not: `Grelda`
(`:88-89`, the root) and `el granero de la Junta` (`:142-145`, holds grain but declares no
`tension`) — matching the round's finding exactly.

### The change

New author obligation:

| # | Obligation | Why the builder needs it |
|---|---|---|
| O12 | every `extent` entity has `tension` | its absence gives an unbounded (`"none"`) beat budget and nothing ever becomes a journey — the SPEC-030 regression, now silent because the key is optional |

**No silent default.** Absence of `tension` on an `extent` entity is a **refusal** at the document
validator, never a fallback to `"none"`. `tension.go`'s COALESCE-to-`"none"` stays in the runner as a
defence against a corrupted read reaching it after registration — it is not licence for a valid document
to omit the key.

---

## 2. Exactly one root `extent` per world

R8 already makes `within` a tree (`SCHEMA-v3.md:79`): a `within` cycle is refused because "containment is
a tree." A tree does not need a distinguished root to be acyclic — and nothing in O1–O11 or R1–R13
obligates one. The old schema did: a single `region` is every place's parent
(`world_genesis.v1.schema.json:36-45`), placed at the coordinate origin, and the 0.6 ring factor measures
distance out from it, deliberately, so leaving a room can exceed a beat budget and become a journey
(`prd_world_creation_depth.md:154-158`).

**Verified, not assumed.** `G_grelda_by_simarch.md` already has exactly one root: `Grelda` (`:88-89`)
carries no `within` key; every other `extent` entity is `within` it, directly (`la plaza mayor`,
`Cuesta Menor`, `:91-97`) or transitively (`la subida` and everything under it, `:99-145`). The document
already obeys the rule that does not yet exist in the contract.

### The change

| # | Obligation | Why the builder needs it |
|---|---|---|
| O13 | exactly one entity with `extent` has no `within` | the coordinate origin and the 0.6 ring factor have nothing to measure from without a single distinguished root, and R8's tree admits more than one without this |

---

## 3. `arrivals[]` — three candidates, one `chosen`, a `why` on every entry

O10 requires "≥1 `arrivals` entry" (`SCHEMA-v3.md:63`) and no more. The shape it replaces was two things:
a mandatory `arrival` carrying `why` (`world_genesis.v1.schema.json:271-302`), and an optional
`arrival_candidates` — present only when the player's identity is left open — that had to number exactly
three, every field filled, and exactly one matching the arrival as the recommended default
(`world_genesis.v1.schema.json:304-319`; enforced at `worldgenesis.go:397-424`). `world_model` folds both
into the one `arrivals[]` section (`SCHEMA-v2.md:63`, "plural premises; there is no opening state") and
obligates neither the count nor the mark.

### The change

New author obligation, refining O10 rather than replacing it:

| # | Obligation | Why the builder needs it |
|---|---|---|
| O14 | when `arrivals[]` leaves the player's identity open, it carries exactly three entries, exactly one marked `chosen`, and a `why` on every entry. When identity is fixed by input, O10's single entry is unchanged | a choice with no marked recommendation, or an offer of two or four, is not a choice a builder can resolve into one opening |

`arrivals[]` gains two entry-level keys, `chosen` and `why`. No new top-level section, no new facet — the
section `arrivals[]` already exists (`SCHEMA-v2.md:63`).

---

## 4. A machine-shaped person name is refused — the Ironmoor guard

`identifierShapedName` (`worldgenesis.go:551-565`) refuses a `canonical_name` containing `_`, or fully
cased with no capital letter anywhere — at three sites: a cast member (`:306-307`), the player's own
arrival name (`:347-348`), and an arrival candidate (`:410-411`). Its doc comment records why it exists,
not a hypothetical: **the Ironmoor breach, live play 2026-08-20** — genesis emitted slug join-keys as
people's canonical names, the registry stored them, and two seats humanised the same slug into two
different names, both of which the player read. `world_model/2–6` has no refusal doing this job. **A
guard with a logged incident behind it is not dropped in a rewrite.**

### The change

| # | Refused | Reason |
|---|---|---|
| R14 | a `canonical_name`, on any entity carrying `agency`, that is machine-shaped — contains `_`, or is fully cased with no capital anywhere | the Ironmoor breach: a join-key emitted as a person's name is read by a player as their name |

Scoped to `agency`, matching the guard's own exemption: places and objects are never machine-shaped in
practice because "places and objects are exempt — their names are made of ordinary English and were never
the leak" (`worldgenesis.go:549-550`). A script with no case of its own — CJK and the like — carries no
capitals and passes untouched, same as today (`:555-564`).

---

## 5. The two arrival-floor refusals become executable, not sufficiency prose

Two refusals live in the engine today and nowhere in the contract:

- *"nothing leads out of `<arrival place>`"* — the SPEC-030 floor (`worldgenesis.go:357,376-382`).
- *"nobody is in `<arrival place>` when the player walks in"* — `worldgenesis.go:386-395`.

`SCHEMA-v4.md` names the same two floors as sufficiency conditions — S1, "every extent reachable from an
arrival has content" (`:73`), and S4, "every `passage` leads to an authored extent, or is `obstructed`"
(`:76`) — and then classifies sufficiency as "a second bar, distinct from validity" (`:60`), checkable but
not gated by the 13 refusal rules that are the standing coverage-check surface
(`2026-08-26-world-model-eight-increments.md:53-54`). A document can satisfy every R1–R13 and still leave
the player unable to leave the room they started in.

### The change

Two new refusals, promoting the arrival-specific instances of S1 and S4 from prose to the validator. This
delta mechanizes the floor only — S1 and S4 remain otherwise unmechanized prose everywhere else in the
document; mechanizing all of sufficiency is a larger job and is not decided here.

| # | Refused | Reason |
|---|---|---|
| R15 | the entity named by the `chosen` (or sole) `arrivals[]` entry has no `passage` to another authored `extent` | the SPEC-030 floor: nothing leads out of where the player starts |
| R16 | no `agency` entity's authored position is inside that same entity | nobody is there when the player walks in |

---

## 6. Every array carries a maximum length

`world_genesis/1` bounds every array it declares — `places` alone at `minItems: 2, maxItems: 8`
(`world_genesis.v1.schema.json:48-49`) — and says why in its own title: *"Array ceilings bound COST, not
shape — they are not a statement about what a world usually contains"* (`:4`, GA-2/GA-3). `validate`
separately asserts the tick ladder fits under the arrival tick
(`genesisBackstoryBaseTick+int64(len(d.History)) > genesisSceneTick`, `worldgenesis.go:489-492`) on the
strength of that ceiling — the schema's ceiling is what guarantees the assertion never fires, and the
assertion is what keeps the two in step if the ceiling ever moves. **No version of `world_model` bounds
any array.** The tick-ladder assertion is reachable and per-build token cost is unbounded.

### The change

| # | Refused | Reason |
|---|---|---|
| R17 | an array with more entries than its section's declared maximum | restores the tick-ladder assertion's bound, and caps per-build token cost the way `world_genesis/1` always did |

Each section's ceiling is set where that section is defined, not invented here — this delta obligates
that every array *has* one, not what any particular number is.

---

## 7. `world.tagline` and `world.ornament` are derived, never authored

`world_model`'s `world` section has always been `name, premise, mood` (`SCHEMA-v2.md:47`) — neither
`tagline` nor `ornament` exists there, authored or derived, in any version through v6. `world_genesis/1`
required both, refused blank (`world_genesis.v1.schema.json:11`, `worldgenesis.go:253-257`), and used
`tagline` as a **structural** gate on cover art: `fillScenes` selects world covers `WHERE w.tagline IS
NOT NULL` (`imagehandler.go:691-695`), under a rule stated in the code's own words —

> "It makes the founder's approval of a tagline STRUCTURAL rather than procedural: no tagline, no cover,
> because there is nothing to render from. The gate is the data, not a promise." (`imagehandler.go:661-663`)

`artcommission.go:66` uses the same predicate. Removing the old commit path without replacing this loses
world cover art silently, the day the old path goes.

**The cost, stated rather than hidden:** deriving `tagline` from `premise` and `ornament` from `mood`
means a line the founder never approved gates a purchase the founder used to structurally control. That
is a real loss of the "the gate is the data, not a promise" property — the gate becomes data derived from
data the founder *did* approve, one hop removed. It is accepted because the alternative is losing cover
art on every world the day the old path is removed, and because it is reviewable: increment 8 ships the
surface that shows invented content and re-derives dependents on amendment, leaving stated content
untouched (`2026-08-26-world-model-eight-increments.md:265-267,270-271`) — a derived tagline is exactly
the kind of invented content that surface exists to show back to the founder.

### The change

Two new reader obligations. No new authored key — `premise` and `mood` are unchanged.

| Author writes | Builder must derive |
|---|---|
| — | `world.tagline`: one line derived from `world.premise`, gating cover-art commissioning exactly as the authored tagline did |
| — | `world.ornament`: derived from `world.mood` |

Reader-obligation count: **25 → 27.**

---

## What this delta does not change

The facet list stays frozen at eleven; no twelfth facet, no exemption list. Top-level sections stay at
sixteen; nothing here adds one — `arrivals[]` gains two entry-level keys and `world` gains two derived
keys, both inside sections that already exist. No refusal or obligation not named above is touched: S2,
S3, S5, S6 remain sufficiency prose, unmechanized, exactly as `SCHEMA-v4.md` left them; `AMENDMENT.md`'s
still-open items (§7) — error legibility under centralised class resolution, the mass-aggregation
question, the friction journal — are untouched by this delta and not decided here.

**Counts.** Author obligations: 11 → **14** (O12–O14). Refusals: 13 → **17** (R14–R17). Reader
obligations: 25 → **27**.

---

## Why this is a delta and not an edit

Same reason as v5 and v6. `SCHEMA-v2.md`'s section table and `SCHEMA-v3.md`'s obligation table are the
record of what `world_model` was believed to need when the old engine's coverage was never checked
against it. Editing those tables in place would destroy the evidence that seven enforced behaviours were
dropped across two contract versions and caught only when a round went and read the engine that already
shipped them (`FINDINGS_contracts.md` C11–C18) rather than trusting that a smaller, later contract must be
a superset of a larger, earlier one.
