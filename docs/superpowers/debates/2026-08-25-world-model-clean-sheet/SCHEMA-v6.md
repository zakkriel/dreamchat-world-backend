# world_model/6 — containment is one authored key, and the engine derives the tree

Delta over v5. Everything in v3, v4 and v5 holds unless changed here. **One change.**

Written because increment 1 of
`docs/superpowers/plans/2026-08-26-world-model-eight-increments.md` blocks on it: the document validator
cannot be written while the contract refuses every document ever written to it.

---

## 1. The contradiction

Three statements, all in force, mutually impossible:

| Where | Says |
|---|---|
| `SCHEMA-v2.md:28` | the `extent` facet's keys are `extent_class`, `medium`, `tension`, **`within`** |
| `SCHEMA-v3.md` D4 | "A key may appear only if its facet is declared." A facet key without its facet is **R7**, "not leniency" |
| `SCHEMA-v3.md` D1 | "`within` holds **every** entity-in-entity relation — **a person in a room**, a room in a district, a city on a creature's back" |

**D1's own worked example is a D4 violation.** A person is `matter + agency`; a person is never `extent`.
So the sentence that establishes the rule breaks the rule, and read strictly R7 rejects the document.

### How wide it actually is — counted, not estimated

Every `within` on an entity lacking the `extent` facet, across all three v4 documents:

| Document | Violations | What they are |
|---|---|---|
| `G_grelda_by_simarch.md` | **9** | 4 people, 2 groups, 2 objects, 1 record |
| `G_marea_by_gamedesign.md` | **10** | 4 people, 2 groups, 1 passage, 3 objects/records |
| `G_sueno_by_extraction.md` | **9** | 4 people, 1 group, 1 object, 3 records |

**28 violations across three documents, and not one of them is an error.** Every single one is ordinary
containment — a person in a house, a rod in a house, a ledger in a granary, a crowd in a square.

`G_grelda_by_simarch.md` §3 reported this as "fourteen entries" and that figure was repeated into
`R_score_grelda.md` and the design record. **The real count is 9 in that document.** Both are corrected.

Three independent blind readers used `within` in the forbidden way throughout and **none noticed** — which
is itself the finding that refusals are unreachable from the reader's side (`R_score_grelda.md` §7).

**A rule that every document breaks, ten times each, in the same way, is not a rule.**

---

## 2. Why the gate exists anyway — the distinction is real

The naive fix is to declare D4 overreaching and move on. That would lose something true.

The engine keeps **three** containment edges, and it keeps them separate because each feeds different
arithmetic (`core/api/tier1.go:16-20`):

| Edge | Meaning | Read by |
|---|---|---|
| `parent_location_id` | a place's parent in the nested-coordinate hierarchy | `fn_distance`, `fn_location_depth` |
| `contained_by` | the carry edge — contents of X | `fn_effective_weight`, `fn_occupied_room`, the encumbrance rule |
| `location_id` | an entity's place | placement and co-location |

So *place inside place* and *thing inside thing* are genuinely not the same relation: one carries geometry,
the other carries mass and volume. The gate was reaching for that distinction. **It just expressed it as a
restriction on the author instead of a derivation by the engine** — and an author writing "the rod is in
the house" should not have to know which tree the engine will file it under.

---

## 3. The change

**`within` is removed from the `extent` facet's key list and is ungated.** It is a general key, legal on
any entity, exactly as D1 says. D4 and R7 continue to hold for every other facet key.

And the distinction §2 identifies becomes a **reader obligation**, where it belongs:

| Author writes | Builder must derive |
|---|---|
| `within` | **which containment tree this edge belongs to, from the container's facets.** A container with `extent` places the contained entity inside it — the edge carries geometry, and distance is measured through it. A container with `matter` and no `extent` bears the contained entity — the edge carries mass and volume. Every `within` edge belongs to exactly one tree, and the tree is derived, never authored |

Reader-obligation count: **24 → 25**.

**`holds[]` is untouched** — substances only, per D1. This change is about entities.

### What is deliberately left open

**A container declaring both `extent` and `matter`** — a living house, a train, a floating island — places
what is inside it *and* is itself a physical body. Whether its contents aggregate into its own mass is not
decided here, because the answer needs the mechanism increment 2 builds (`capacity_class` and
`bulk_class` resolving to real quantities) and the composition rules increment 4 builds. Placement is
settled now: **`extent` wins for placement** — you are *in* the house, never *carried by* it.

Filed as increment 2's first design question. Naming it here so it is not silently answered by whichever
function is written first.

---

## 4. Options rejected, so they are not re-proposed

**Keep the gate; require containers to declare `extent`.** A person carrying a rod would need "has an
interior; things are within it." That is a category error, and the facet list is **frozen at eleven** —
redefining a facet to dodge a gate spends the freeze on a bug. It also needs an exemption list, which
the standing rules forbid.

**Resurrect `borne_by` as a second authored key.** v3 D1 deleted it deliberately, and the reason was
*motion*: "if an entity has `motion`, everything `within` it moves with it, transitively." That reason is
still good and increment 7 depends on it. Bringing the key back to solve a *weight* problem would reopen a
closed decision to fix an unrelated one, and would make the author carry a distinction the engine can
derive.

**Leave it and let the validator special-case `within`.** A validator with a hole in it where the contract
contradicts itself is a validator nobody can trust, and the hole would be invisible to every future reader
— which is precisely how this defect survived four contract versions.

---

## 5. Why this is a delta and not an edit

Same reason as v5. v2's key table is the record of what was believed when the facets were first written;
editing it in place destroys the evidence that a key was assigned to a facet on the assumption that only
places contain things, and that three independent readers then broke that assumption thirty times without
noticing.
