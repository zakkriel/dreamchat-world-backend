# world_model/3 — the contract

v2 was a shape. v3 is a **contract**: it states what the author owes, what the builder owes, and what
makes a document invalid. Nothing here is more expressive than v2. Several things are less.

## The convergence commitment

**The facet list is frozen at eleven.** A twelfth facet may be added only by deleting an existing one.
If a future world needs a new facet that deletes nothing, this approach has failed and we say so rather
than widening. Same rule for top-level sections.

| | v1 | v2 | v3 |
|---|---|---|---|
| fixed entity kinds | 4 | 1 | 1 |
| facets | — | 11 | **11 (frozen)** |
| top-level sections | 19 | 17 | **16** |
| author obligations | 0 | 0 | **11** |
| refusal conditions | 0 | 0 | **12** |
| reader obligations | 0 | 0 | **21** |

---

## 1. Disambiguations — one legal reading, alternatives forbidden

Both encoders of two different worlds diverged here. v3 picks one and forbids the rest.

**D1 · Containment is one relation.** `within` holds **every** entity-in-entity relation — a person in a
room, a room in a district, a city on a creature's back. `holds[]` is **substances only**. `borne_by` is
**deleted**: if an entity has `motion`, everything `within` it moves with it, transitively. *Forbidden:*
using `holds` for an entity; using `within` for a substance.

**D2 · `layers[]` is deleted.** A concurrent reality is an **entity with `extent`** that other entities
are `within`. Sueño Común's shared dream is an entity whose `demand` for sleepers, unmet, ends it.
Law is scoped by `law[].within` naming an entity, so a rule that holds only inside the dream is a rule
scoped to that entity's interior. *Forbidden:* any construct meaning "a parallel rule-space" other than
an entity. This deletes a section rather than specifying one.

**D3 · Magnitude is decided by promotability.** An entity is `magnitude` **iff play may need to promote
an individual out of it**. Two hundred houses the player will only ever address collectively is one
`magnitude` entity; a house that can be knocked on is its own entity. *Forbidden:* referencing a
`magnitude` entity as an individual anywhere in the document — that contradiction is a refusal (R11).

**D4 · Facet keys are gated.** A key may appear only if its facet is declared. `bulk_class` without
`matter`, `pursuing` without `agency`, `connects` without `passage` are refusals (R7), not leniency.

---

## 2. Author obligations — what must be present

Each is justified by **what a builder needs to make a world live**, never by what worlds contain.

| # | Obligation | Why the builder needs it |
|---|---|---|
| O1 | ≥2 entities with `extent`, joined by ≥1 `passage` | with one place, movement has no target and nothing can be elsewhere |
| O2 | ≥1 entity with `agency` | nothing acts unless something decides |
| O3 | every `agency` entity has ≥1 `pursuing` | a mind with no goal can only react; the world then revolves around the player |
| O4 | ≥1 `agency` entity has `hiding` | with nothing withheld anywhere, every conversation is exhaustive and discovery is impossible |
| O5 | ≥1 `opposition` | with no authored incompatibility, a room's only motion is the player talking |
| O6 | every `matter` entity has `bulk_class` | without it nothing can be lifted, blocked, or contained |
| O7 | every `demand` names a substance suppliable somewhere in the document | an unsuppliable demand is a death sentence with no play in it |
| O8 | every `accumulator` has ≥1 `raised_by` **and** ≥1 threshold | otherwise it is a number that never moves or never matters |
| O9 | every `indicator` names a hidden state some accumulator or property actually holds | a sign of nothing reads as noise |
| O10 | ≥1 `arrivals` entry | a world nobody can enter is not playable |
| O11 | `excluded[]` is present, possibly empty, and explicitly so | silence about what a world excludes is what lets a builder import genre defaults |

---

## 3. Refusals — what makes a document invalid

| # | Refused | Reason |
|---|---|---|
| R1 | any unresolved name reference | the document is the only namespace |
| R2 | `passage.connects` ≠ exactly 2 extents | a way joins two sides |
| R3 | a number in an engine-computed field | the builder owns quantity; classes are the interface |
| R4 | a class outside its ladder | ladders are grammar |
| R5 | `magnitude` entity referenced individually | D3 contradiction |
| R6 | `excluded[]` entry contradicted by an authored entity | the document disagrees with itself |
| R7 | a facet key without its facet | D4 |
| R8 | `within` cycle | containment is a tree |
| R9 | `demand` with no supplier | O7 |
| R10 | `agency` with no `pursuing` | O3 |
| R11 | an `accumulator` threshold ladder out of order, or with a repeated `at` | crossings must be totally ordered |
| R12 | `history` entry with `standing: "disputed"` whose knowledge holders all agree | a dispute needs disagreement |

A document violating any refusal is **rejected whole**, with the reason named. There is no partial build.

---

## 4. Reader obligations — what the builder owes

The half that did not exist. For each authored class, what a builder **must** derive. Two builders
honouring these produce the same world; without them, "same document, different world" is as fatal as
"same brief, different document."

### Space and motion
| Author writes | Builder must derive |
|---|---|
| `extent_class` | a footprint; distances between entities `within` it |
| `pace_class` | a speed; with distance ⇒ a duration for every move |
| `motion.trajectory` | a position over time; everything `within` moves with it, transitively (D1) |
| `passage.admits` / `obstructs` | permit or refuse traversal per predicate; refusal states the failed predicate |
| `passage.hazard_class` | a cost or condition on those who cross, distinct from refusal |

### Matter
| `bulk_class` | a volume and a mass |
| `capacity_class` | how much it holds; exceeding it refuses |
| `integrity` | a degradation level, and how much remains before terminus |
| `holds[].abundance` | a quantity; drawing reduces it; exhaustion refuses the draw |

### Time and change
| `tension` | a beat budget; acts exceeding it become extended rather than refused |
| `process.rate_class` | how fast state moves, without an event per tick |
| `cycle.period_class` + `phases` | when each phase flips; the flip is an event that leaves a trace |
| `accumulator.thresholds` | fire **once**, in order, at each crossing; `irreversible` never un-fires |
| `demand.unmet` | apply the effect after `onset_class`, and go on applying it |

### Knowing
| `channel.latency_class` | the delay before a fact is knowable to a receiver |
| `channel.reach` | who can receive at all |
| `channel.decay` | when a belief acquired through it expires |
| `channel.conceals` | what identity is withheld; `none` renders present entities' `pursuing` and `hiding` to all receivers |
| `path` | confidence, and what kind of later event can correct it |
| `indicator.reliability_class` | how often the sign misreports; the hidden value is **never** exposed |
| `history.standing: disputed` | hold both accounts; **never** resolve without a later event doing so |
| `record.asserts[].accurate: false` | the claim is readable and wrong; reading it does not correct the reader |

### Everything
| `excluded[]` | refuse to author anything matching, in every seat, for the life of the world |

---

## 5. What changed from v2, in full

- **deleted:** `layers[]` section; `borne_by` key; `holds` for entities
- **disambiguated:** `within` is the sole containment relation; `magnitude` decided by promotability
- **added:** 11 obligations, 12 refusals, 21 reader obligations, the facet freeze
- **unchanged:** all 11 facets, the vocabulary/grammar split, class-not-number, numbers permitted in
  player-read fiction, `excluded[]`, `offices[]`, `standing[]`, `indicators[]`, disputed events

## 6. Expected consequence

**Some of the six existing encodings become invalid.** That is the point — until now nothing could be.
The Sueño encodings both used `layers[]`, which no longer exists; several entities have `agency` with
no `pursuing` (R10); no encoding declared `excluded[]` (O11).

The next test is therefore two-part: re-encode under v3 and check (a) whether the same two encoders now
agree, and (b) whether the refusals catch real errors rather than good documents.
