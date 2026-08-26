# Findings — contracts-and-platform adversary

**Round:** 2026-08-26 landing-contract retarget. **Document under attack:** `PROPOSAL.md` in this
directory. **Assignment:** its questions 1 and 3, plus `prd_world_creation_depth.md` §9 Q1 and Q2.

**Area:** contracts-and-platform. Brief: `harness/roles/contracts-and-platform-expert.md`. Dossier:
`docs/areas/contracts-and-platform.md`. Dispositions per `docs/00_workspace/round-protocol.md` §7 —
`block`, `gate`, `accept-with-reason`, never "noted".

**What I ran, and what I did not.** Every citation below was opened and read this session. I ran no
mutation experiment: the round under review is a proposal document, there is no diff, and
`ci/mutate.sh` rewrites a source file — which a review of a design document may not do. §7 is
therefore a table of **predicted** verdicts with the reason for each prediction, and the implementing
round must run them and report the table it prints. I did not run `make schema-contract`, `make
schema-check` or `go test`: nothing in this round changes code, and citing a green suite over an
unchanged tree as evidence about a proposal would be exactly the "liveness, not correctness" error the
brief warns about.

---

## 1. Q1 — does retargeting break R2's static guarantee?

**Short answer: yes, and not for the reason the proposal asks about.** Facet gating is a false lead.
R2 dies on three other things, all of which are fixable and none of which the amendment names.

### C1 · `block` — R2 has no left operand, and the roadmap defers it into the increment that consumes it

R2 is "a static set-difference over schema × declarations" (`prd_world_creation_depth.md:119-123`). Its
left operand today is a draft-07 file with `additionalProperties:false` at every level
(`core/api/schema/world_genesis.v1.schema.json:7,12,39,54`), so enumerating its leaves is a mechanical
walk — `ci/schema_contract.py:62-69` (`walk_props`) already performs exactly that walk over every
published schema.

`world_model/6` has no such artifact. `PROPOSAL.md:23-24` states it outright: *"There is no machine
representation of v6 at all: no schema file, no Go type."* The roadmap ships it inside the very
increment that also ships the landing framework —
`docs/superpowers/plans/2026-08-26-world-model-eight-increments.md:97` ("the contract as a machine
artifact (schema + service type from one source, no drift)").

So the honest answer to the proposal's own Q1 is neither "still statically computable" nor "facet
gating makes it runtime": **R2 is uncomputable until the artifact exists.** The amendment must state
the artifact as a **precondition of R2**, ordered before the landing framework within increment 1, not
as a peer deliverable.

### C2 · `block` — four top-level sections have no keys to diff against

`opposition[]`, `traces[]`, `epochs[]` and `arrivals[]` are one prose line each in `SCHEMA-v2.md:55`,
`:60`, `:61`, `:63`. They appear in no key table anywhere and in no illustrative fragment — the
worked document at `SCHEMA-v2.md:86-190` omits all four. `SCHEMA-v3.md`, `v4`, `v5` and `v6` add
nothing for any of them.

Two of the four carry **obligations**: O5 requires ≥1 `opposition` (`SCHEMA-v3.md:58`) and O10 requires
≥1 `arrivals` entry (`:63`). The contract obliges an author to write sections whose leaves it never
defines. R2's right operand is a set of `LeafPath` (`prd_world_creation_depth.md:90`); its left operand
for these four sections is empty, so R2 either passes them vacuously or fails every document.

### C3 · `block` — and the consequence is already on disk: three encoders, two key names, one obligated section

Mechanical scan of the three v4 documents:

| Section | `G_grelda_by_simarch.md` | `G_marea_by_gamedesign.md` | `G_sueno_by_extraction.md` |
|---|---|---|---|
| `opposition[]` | `between`,`incompatible`,`stakes`,`source` (`:255-258`) | same (`:246-253`) | same (`:103-107`) |
| `traces[]` | `of`,`leaves`,`ages`,`source` (`:315-321`) | same (`:325-332`) | absent |
| `epochs[]` | `name`,`differed[{topic,subject,then}]`,`surviving_traces` (`:323-327`) | same (`:335-340`) | absent |
| `arrivals[]` | `premise`,`seen_as`,**`within`** (`:353`),`capability`,`source` | `premise`,`seen_as`,**`place`** (`:369`,`:374`,`:379`),`capability`,`senses` | `premise`,`seen_as`,**`place`** (`:125`),`capability` |

Three sections converged unaided. The fourth — the one O10 makes mandatory — did not: **`within` in one
document, `place` in two, for the same relation.** The leaf set of `arrivals[]` is therefore not a
property of the contract; it is a property of whichever document you happen to read. R2 over it is
empirical, not static.

Note also `senses` on a Marea arrival (`G_marea_by_gamedesign.md:370`): `SCHEMA-v5.md:88` already
records `senses` as one of six authored keys with **no reader obligation whatsoever**.

### C4 · `accept-with-reason` — facet gating is not the problem, and the round should stop looking there

D4 (`SCHEMA-v3.md:43-44`) constrains which keys may **appear in a document**; it does not constrain
which keys **exist in the contract**. The leaf universe is the union over the eleven frozen facets
(`SCHEMA-v2.md:26-38`), minus `borne_by` (deleted, `SCHEMA-v3.md:29`), minus `within` (removed from the
`extent` facet's key list and ungated, `SCHEMA-v6.md:69`). That union is enumerable with no document,
no items and no database — which is R2's stated requirement (`prd_world_creation_depth.md:87`,
`:121-123`).

Facet gating turns leaf **presence** into a per-document question. Leaf **existence** stays static, and
existence is the only thing R2 needs. **Reason for accept:** naming this so the round does not spend
itself defending a guarantee that was never threatened. The real defects are C1, C2, C3, C5 and C7, and
the amendment should replace its Q1 framing with them.

### C5 · `block` — the entity's non-facet keys are enumerated nowhere, and one of them is an open-keyed map

`SCHEMA-v2.md:26-38` tabulates **facet** keys only. The worked fragment puts keys on entities that
appear in no table: `name`, `facets`, `seen_as`, `capability{moves_by, carry_class}`, `senses{}`,
`supports[]` (`SCHEMA-v2.md:120-150`), and `confer` on offices (`:154`). `SCHEMA-v5.md:88` confirms it
independently — *"`medium`, `affords`, `resists`, `alters`, `senses` and `confer` — six authored keys
bearing on whether a receiver receives anything at all — have no reader obligation whatsoever."*

`senses` is worse than untabulated: it is an **open map keyed by a world-invented channel name**
(`SCHEMA-v2.md:136` `"senses": { "the bone": "acute" }`; `G_marea_by_gamedesign.md:370` `"senses":
{ "sight": "normal", "la voz de los objetos": "acute" }`). A `Consumes []LeafPath`
(`prd_world_creation_depth.md:90`) cannot name that path. The grammar/vocabulary split guarantees the
question recurs: `media`, `movements`, `channels`, `conditions` and `substances` are all open and
world-minted (`SCHEMA-v2.md:16-20`).

The amendment must decide whether an open-keyed map is **one** leaf or many, and say so in R2's own
words. Today R2 has no answer for it.

### C6 · `gate` — "non-numeric leaf" excludes nothing today and becomes an undecided exclusion under v6

R2 diffs against "every **non-numeric** leaf" (`prd_world_creation_depth.md:120-121`).
`world_genesis.v1.schema.json` declares **zero** `integer` or `number` leaves — verified by count, and
stated at `:4` (*"no number of any kind appears anywhere in this schema"*). So the qualifier currently
excludes nothing and has never been exercised.

Under v6 it starts excluding things. `SCHEMA-v2.md:68-70` permits a number a player reads, and
`SCHEMA-v4.md:96` makes `exemplar` fiction that *"may contain a number"* — there are 15 `exemplar`
occurrences across `G_grelda_by_simarch.md` and `G_marea_by_gamedesign.md`. A qualifier that has never
removed a leaf silently begins removing them, and nobody will notice because the failure mode is a
leaf that is **not** reported as unclaimed.

**The gate:** define "non-numeric" as a type predicate over the machine artifact, and make the
registration check **print the excluded set** rather than apply it silently. An exclusion nobody can
see is the schema-validator defect again (dossier §2 — `beat_frame.v3` used `"format":"uuid"` 35 times
and `minItems` twice and none of those constraints was ever checked, failure-log #20).

### C7 · `block` — one recursive `entities[]` weakens R2's guarantee, and §2's "survives untouched" claim is false

`world_genesis/1` partitions nouns into **disjoint typed arrays** — `places`, `ways`, `cast`, `objects`
(`world_genesis.v1.schema.json:6`). Concept and section coincide, so `⋃ Consumes` is a disjoint union
and its complement names both the unclaimed leaf **and** the concept that should have owned it.

`world_model/6` has one recursive `entities[]` discriminated by facet (`SCHEMA-v2.md:52`,
`:12` — *"one `entities[]`, kind expressed as composable facets"*). Every person-shaped, place-shaped
and object-shaped concept now claims overlapping paths inside one array. The union still covers, so R2
still fails on a wholly unclaimed leaf — but the guarantee degrades from *"some landing parses this
leaf"* to *"some landing wrote this path down"*, and nothing checks that `Parse`
(`prd_world_creation_depth.md:101`) ever reads a path its `Declare` claimed.

That degraded guarantee is the exact shape of the defect R2 exists to kill: `cast[].standing` is
schema-required (`world_genesis.v1.schema.json:117`), refused when blank (`worldgenesis.go:310-311`),
and written by no commit path (`prd_world_creation_depth.md:29-31`). A landing that lists
`entities[].standing` in `Consumes` and never parses it reproduces `standing` precisely, and passes R2.

`PROPOSAL.md:33-38` claims R1, R3–R6 survive untouched and lists R2 only as "becoming bidirectional"
(`:45-58`). **R2 does not survive untouched.** Either the amendment states the weakened guarantee, or
it adds the missing half: a landing's `Consumes` is checked against what its `Parse` actually reads.

### C8 · `block` — the amendment miscounts its own target, twice, and both are citation defects

- **"17 top-level sections"** (`PROPOSAL.md:18`, restated at `:111`). `SCHEMA-v3.md:16` says
  **16 (frozen)**; v3 D2 deleted `layers[]` (`:32-36`). Counted mechanically, the three v4 documents
  carry 16 content sections plus the `world_model` version marker. This is not a rounding error:
  `SCHEMA-v3.md:8-10` makes the section count a **freeze** — *"A twelfth facet may be added only by
  deleting an existing one… Same rule for top-level sections."* Writing 17 in an amendment is either
  an error or an unannounced freeze violation, and a reader cannot tell which.
- **"3 of 25 obligations working, 12 partial, 9 absent"** (`PROPOSAL.md:22-23`). 3 + 12 + 9 = 24, and
  the audit states its rule set is 24 in three places (`01_engine_capability_audit.md:11-12`, `:20`,
  `:29`). The 25th obligation is v6's own new `within`-tree derivation (`SCHEMA-v6.md:74-78`) and it
  has **no audit row**. The amendment attributes a 24-row result to a 25-row rule set and thereby
  assigns its own new obligation a status nobody measured.

Both are the failure class `ci/check_citations.sh` exists for — with the limit the dossier records
(`docs/areas/contracts-and-platform.md:162-166`): the gate asserts a cited **id** resolves, never that
a cited **number** does. That judgement is the reviewer's, and this is it.

---

## 2. Q3 — what the clean cutover strands

**The framing first.** `PROPOSAL.md:84-85` and roadmap closed decision 1
(`2026-08-26-world-model-eight-increments.md:64`) both say the old format and its bespoke commit path
are **removed**. `world_genesis/1` is a **published schema** in `core/api/schema/`, and this area's law
is unambiguous: *"A published version is superseded, never deleted"* (dossier `:152-155`). That rule
exists because `scene_current.v2`, `beat_frame.v2` and `world_directory.v1` were deleted and the
frontend's `verify:contract` caught it **only because the files vanished** (failure-log #7); had they
been kept, as normal versioning would, the gate would have printed OK against a 100% incompatible
client. Retiring a version is: publish `vN+1` beside `vN` → move every consumer → delete `vN` in a
round naming them all → record it in `contracts.md`'s retirement table (dossier `:90`). **The proposal
deletes `vN` with no `vN+1` published** — see C1, there is no artifact to publish.

### C9 · `accept-with-reason` — the cross-repo wire does *not* break

`world_genesis_frame/3` carries only `schema_version | kind | stated | question | options | world_id`
(`core/api/schema/world_genesis_frame.v3.schema.json:11-63`) and `world_kickstart_turn/2` only
`schema_version | done | question | …`. Both are target-agnostic, and both are vendored by
`dream-weaver-visuals/contracts/` — `world_kickstart_turn.v2.schema.json` verified byte-identical.
**Reason:** stating it so the round does not spend itself on a seam that is already safe. The
cross-repo published contract is not what the retarget threatens.

### C10 · `gate` — the replacement schema has exactly two legal endings in this area, and the amendment names neither

A v6 seat leash will have `additionalProperties:false` and no `schema_version` envelope, so it never
appears on the wire. `ci/schema_contract.py:147-149` fails any published schema no payload exercised.
The two correct endings (dossier `:89`, *"Those are the only two correct endings"*):

1. a captured payload wired into `ci/gen_payloads.sh` — the path `world_genesis/1` uses today
   (`ci/gen_payloads.sh:166-167`); or
2. a row in `INPUT_CONTRACT_SCHEMAS` (`ci/schema_contract.py:49`).

The round adds one of them in the same change that adds the schema file, or `make schema-contract` goes
red on merge.

### The stranded capabilities

Each below is a capability the current commit path provides, with **no `world_model/6` equivalent**.

### C11 · `block` — `tension` goes required → optional, and its absence silently disables the journey

- Required today: `world_genesis.v1.schema.json:53` (`places[].items.required` includes `tension`);
  refused out-of-set at `worldgenesis.go:284-285`.
- The region is additionally hardcoded to `calm`, with the reason written down:
  *"A region with no tension reads as 'none' ⇒ an infinite beat budget, which is the exact condition
  that made the Journey unreachable before SPEC-030. Every location gets stamped, parents included."*
  (`worldgenesiscommit.go:625-629`.)
- In v6, `tension` is an **optional** key on the `extent` facet (`SCHEMA-v2.md:28`), and **no
  obligation O1–O11 requires it** (`SCHEMA-v3.md:52-64`).

The failure is silent at every layer below it: `trg_validate_tension` fires only
`IF NEW.attrs ? 'tension'` (`core/db/schema.sql:3748`), so an unstamped location passes the trigger;
`beatBudgetSeconds` COALESCEs a missing row to `'none'` (`core/api/tension.go:58`); and
`tensionBudgetSeconds` maps `none` to `math.MaxInt64` (`:38-39`). Every act fits every beat, nothing
becomes a journey, and the suite stays green. This is the SPEC-030 regression reintroduced by a
contract change, and no existing gate can see it.

### C12 · `block` — the region, and with it the coordinate origin and the 0.6 ring

`world_genesis/1` requires a `region` that is the single parent of every place
(`world_genesis.v1.schema.json:6`, `:36-45`). `writeOpeningState` puts it at `(0,0)` with a footprint
from `fn_extent_class_metres` + `fn_area_around` (`worldgenesiscommit.go:606-621`), and
`genesisPlaceCoords` rings every room at **0.6 of that radius** *"specifically so leaving a room can
exceed a beat and become a journey"* (`worldgenesiscommit.go:884-892` — the same constant AC-3 names
as the canonical world-feel constant, `prd_world_creation_depth.md:154-158`).

`world_model/6` has no distinguished root. Containment is an arbitrary `within` tree
(`SCHEMA-v3.md:27-29`, ungated by `SCHEMA-v6.md:69`) with no rule electing one entity as the coordinate
origin and no guarantee that whatever is elected carries `extent_class`. Without a region radius there
is no ring, no distance, and `fn_distance`'s recursive climb to the nearest common parent
(`01_engine_capability_audit.md:33`) has nothing to climb to. No v6 key supplies it and no obligation
demands one.

### C13 · `block` — `world.tagline` and `world.ornament`, and with the first of them, world cover art

- **`tagline`**: required (`world_genesis.v1.schema.json:11`), refused blank
  (`worldgenesis.go:253-255`), written to `world.tagline` (`worldgenesiscommit.go:125-128`).
  `fillScenes` selects world covers `WHERE w.tagline IS NOT NULL` (`imagehandler.go:691-695`) under the
  rule at `:662-663`: *"It makes the founder's approval of a tagline STRUCTURAL rather than
  procedural: no tagline, no cover, because there is nothing to render from. The gate is the data, not
  a promise."* `artcommission.go:66` uses the same predicate, and the world-directory projection
  selects it (`core/db/schema.sql:3189`).
- **`ornament`**: required (`world_genesis.v1.schema.json:11`), refused blank
  (`worldgenesis.go:256-258`), written into `world.theme` (`worldgenesiscommit.go:88`); `PUT` on a
  world returns 400 without it (`worldshandler.go:142`).

`world_model/6`'s `world` section is `name, premise, mood` (`SCHEMA-v2.md:47`). Neither key exists.
The day the old path goes, every generated world loses its cover **silently** — the workspace standing
answer *"Art is automatic. Genesis kicks a reconciler and a ticker sweeps"* becomes false with nothing
red anywhere.

### C14 · `block` — `attrs.max_load`, and with it encumbrance, and the increments are ordered wrong for it

`genesisPlayerMaxLoad = 80` (`worldgenesiscommit.go:62`) is stamped on every actor at `:666`, `:786`
and `:804`. What happens without it is written in the function's own comment: *"v_cw > NULL (no
max_load) is NULL → the ELSE clears — an unset capacity can't be exceeded"*
(`core/db/schema.sql:937-938`). So `encumbered` never sets, and audit row 7's *"Refusal real and fires
at commit"* (`01_engine_capability_audit.md:39`) stops firing.

`capacity_class` is v6's replacement (`SCHEMA-v2.md:31`) and it has **no ladder in the engine** — audit
row 7 again. The ladder lands in **increment 2**
(`2026-08-26-world-model-eight-increments.md:115-124`), i.e. **after** increment 1 removes the writer
(`:88-102`). Either the ordering changes, or the amendment records a one-increment regression with a
SPEC id and a written reason — `accept` is available here, silence is not.

### C15 · `block` — the arrival-choice flow, entire

- `arrival_candidates` — the array (`world_genesis.v1.schema.json:304-319`) and its coherence rule:
  exactly three, every field filled, names distinct, exactly one **is** the arrival and is the
  recommended default (`worldgenesis.go:399-424`).
- `arrival.why` — carried onto the chosen arrival (`worldgenesishandler.go:534`) and rendered as the
  option's `implication` with `recommended` (`:658-663`).
- `newCast` — people the chosen identity references into existence, minted at arrival
  (`worldgenesiscommit.go:152-159`), given minds grounded in the arrival event (`:169-174`), with
  mutual name knowledge sourced from the one `world_genesis` event *because `fn_perceived_name` reads
  only perceptions sourced there as names* (`:176-197`); merged across turns by `mergeNewCast`
  (`kickstartstate.go:144-159`).

`world_model/6`'s `arrivals[]` is *"plural premises; no opening state"* (`SCHEMA-v2.md:63`) — one line,
no key list (C2), no recommended default, no `why`, and no notion of a document amended between the two
transactions R6 requires (`prd_world_creation_depth.md:139-141`). This is a shipped feature with its own
plan (`docs/superpowers/plans/2026-08-20-kickstart-arrival-choice.md`) whose vendored wire contract
(C9) would keep validating while the thing behind it is gone.

### C16 · `block` — the Ironmoor guard

`identifierShapedName` (`worldgenesis.go:541-565`) refuses a machine-shaped person name at three sites
(`:306-307`, `:347-348`, `:410-411`). Its comment records the live incident: *"the Ironmoor breach,
live play 2026-08-20: genesis emitted slug join-keys as people's canonical names, the registry stored
them, and the naming wall guarded strings no model ever writes — the seats humanised the slugs to
'Silas' and 'Emmett' and the player read both."*

`world_model/6`'s thirteen refusals (`SCHEMA-v3.md:70-84` plus R13 at `SCHEMA-v4.md:118`) contain
nothing about the **shape** of a name; R1 requires only that a reference resolve. Deleting `validate()`
deletes the guard, and the defect it stops is a logged production incident, not a hypothetical.

### C17 · `block` — two floor refusals become sufficiency prose with no checker

- *"nothing leads out of `<arrival place>`"* — `worldgenesis.go:376-382`, explicitly the SPEC-030 floor
  (`:357`).
- *"nobody is in `<arrival place>` when the player walks in"* — `worldgenesis.go:386-395`.

v6's O1 requires ≥2 extents joined by ≥1 passage, and O10 requires ≥1 `arrivals` entry
(`SCHEMA-v3.md:54`, `:63`) — **neither ties the passage or the population to the arrival.**
`SCHEMA-v4.md:73` (S1) and `:76` (S4) say exactly what is needed, but `SCHEMA-v4.md:60` classifies
sufficiency as *"a second bar, distinct from validity"* and the roadmap's standing gate 1 is the **13
refusal rules**, not S1–S6 (`2026-08-26-world-model-eight-increments.md:53-54`). So both floors become
unenforced by construction, and a world can be authored that the player cannot leave.

### C18 · `gate` — the tick ladder loses its bound, and per-build cost loses its ceiling

`validate` asserts `genesisBackstoryBaseTick + len(History) ≤ genesisSceneTick`
(`worldgenesis.go:491-492`) and its comment says the schema's ceiling is what guarantees it.
`world_genesis/1` bounds every array explicitly, and says why: *"Array ceilings bound COST, not shape"*
(`world_genesis.v1.schema.json:4`; e.g. `places` `minItems: 2, maxItems: 8` at `:48-49`).

**No version of `world_model` bounds any array.** The tick-ladder assertion becomes reachable, and
per-build token cost becomes unbounded — which is this area's (boot, cost, the release), not the
domain's. **The gate:** the machine artifact carries `maxItems` on every array, or the runner derives
the tick ladder from `len(history)` instead of asserting it against a constant. Pick one in the
amendment; both are one line of policy and neither is a design question.

### C19 · `accept-with-reason` — `world.brief` reaching the narrator is independent of the retarget

`PROPOSAL.md:94-96` bundles it into the amendment. It does not belong there: `world.brief` is written
from the request, not from the document (`worldgenesiscommit.go:125-128`), and the finding that no beat
path reads it (`core/db/schema.sql:4234`'s `COMMENT`, `01_engine_capability_audit.md:95-98`) holds under
either contract. **Reason:** it is the one item in §3.4 that needs no contract at all, and holding it
hostage to increment 1's ordering delays *"the largest single playability lever in the increment"*
(`PROPOSAL.md:96`) for no reason the round can name.

---

## 3. Inherited — `prd_world_creation_depth.md` §9 Q1: does `Refuse`'s resolver become the god object?

### C20 · `gate` — yes, it gets worse; and the amendment already contains the bound it declines to claim

**Today.** The cross-concept rules are joins over one flat name namespace with five maps
(`worldgenesis.go:270`, `:296`, `:430`, and the lookups at `:318`, `:353`, `:447-450`, `:463-468`). The
example §9 Q1 names — *"somebody is already in the arrival room"* — is `worldgenesis.go:386-395`, a scan
of one array against one field.

**Under `world_model/6`, nine rules quantify over the whole document rather than over an item:**

| Rule | Where | Why it is not per-item |
|---|---|---|
| R1 any unresolved name reference | `SCHEMA-v3.md:72` | the document is the namespace |
| R5 `magnitude` referenced individually | `:76` | needs every reference in the document |
| R6 `excluded[]` contradicted by an authored entity | `:77` | whole-document match |
| R8 `within` cycle | `:79` | graph over all entities |
| R9 / O7 `demand` with no supplier | `:80`, `:60` | cross-section: entities × substances |
| O8 `accumulator` needs `raised_by` **and** a threshold | `:61` | cross-element |
| O9 `indicator` names a state some accumulator or property holds | `:62` | cross-section |
| R11 threshold ladder ordering | `:82` | within-element (this one *is* per-item) |
| R13 `inferred_from` chain terminating in stated content | `SCHEMA-v4.md:118` | transitive closure over every element |

None of those is expressible as `Refuse(item, resolver)` (`prd_world_creation_depth.md:103`) without the
resolver acquiring a view of the entire document — which is §9 Q1's failure mode stated exactly:
*"a resolver rich enough for eight concepts is the old accretion renamed"* (`:270-271`).

**The bound, and it holds.** `PROPOSAL.md:91-93` already draws the line: *"R2 is a registration-time
check over schema × declarations; this is a per-document check. Different gate, different time, both
needed."* Make it a **type boundary** rather than a sentence:

> The document validator takes `*genesisDoc` and **no resolver**. `Refuse` takes `resolver` and **no
> document**. Neither type is in the other's scope.

Every rule in the table above then has exactly one legal home and **cannot** migrate, because the type
it would need to migrate is not reachable from where it would move. That is mechanizable, it is
checkable at compile time, and it is what `prd_world_creation_depth.md:271-272` ordered — *"Bound it
deliberately in step 1, or the contract rots from this seam."*

`PROPOSAL.md:104-105` files this as *"unanswered, and v6's 17 sections raise the pressure on that
seam."* It is not unanswered; the answer is four lines earlier in the same document, and the section
count in that sentence is wrong (C8). **The round states the bound and stops calling it open.**

---

## 4. Inherited — §9 Q2: centralised class→number as a single point of failure, and error legibility

### C21 · `block` — the premise is wrong in both directions, and the correction moves the risk

**`prd_world_creation_depth.md:273` says the trade is "~40 named `refuse()` messages" for one path.
Counted:**

- `validate()` contains **67** `return refuse(` sites (`worldgenesis.go:249-495`); 82 across the four
  genesis files.
- Of those, the ones a class→number resolver would actually replace number **seven** — every *"outside
  the closed set"* message: `worldgenesis.go:263`, `:285`, `:287`, `:321`, `:328`, `:374`, `:484`.

So the SPOF trade is **7-for-1, not 40-for-1** — the resolver is a smaller risk than §9 Q2 assumes.
The number that matters is the other one: **the clean cutover deletes 67 named refusals**, against
`world_model/6`'s 13 refusals plus 11 obligations = **24** document rules
(`SCHEMA-v3.md:70-84`, `:52-64`, `SCHEMA-v4.md:118`). The error-legibility loss is in the cutover, not
in the resolver, and the amendment attributes it to the wrong place.

**`PROPOSAL.md:36-37` says "the whole engine has three hand-built conversions"** (from
`01_engine_capability_audit.md:71-77`; repeated at
`2026-08-26-world-model-eight-increments.md:123`). **There are five in the live path:**

| Conversion | Where | In the audit's three? |
|---|---|---|
| `extent_class` → metres | `extent_class_metres:3901` / `fn_extent_class_metres:1777` | yes |
| duration class → seconds | `duration_class_seconds:3857` / `fn_duration_class_seconds:1647` | yes |
| `tension` → seconds | `core/api/tension.go:28-45` | yes |
| **strength class → `personality_core.traits[].value`** | `core/api/worldgenesis.go:500-511` | **no** |
| **strength class → `personality_core.malleability`** | `core/api/worldgenesis.go:515-526` | **no** |

The last two survive the retarget — `world_model/6` still authors `disposition[].strength`
(`SCHEMA-v2.md:137-138`) and `personality_core.malleability` still carries
`CHECK (malleability > 0 AND malleability <= 1)` (`core/db/schema.sql:4058`). So increment 2's *"one
generic resolver replacing the three ad-hoc conversions"* under-scopes by two, and the audit's own
count is scoped to play-time conversions without saying so.

**And both uncounted conversions carry a silent default.** `strengthValue` returns `0.5` and
`malleabilityValue` returns `0.35` on `default:` (`worldgenesis.go:508-509`, `:523-524`) — an
unrecognised class word becomes a **number**, not a refusal. That is safe today only because
`genesisStrengths` refuses it upstream (`:320-321`, `:327-328`), and those two are hand-maintained
duplicates of one ladder **by deliberate policy**: *"Duplicated from the schema on purpose: the schema
constrains a cooperative provider, this constrains reality"* (`:233-234`).

`world_model/6` has **thirteen** ladders (`01_engine_capability_audit.md:66-69`) and the cutover deletes
the belt. A centralised resolver inherits thirteen chances to do quietly what these two already do
quietly, with nothing upstream to make the default unreachable.

**What the amendment must state, as a decision rather than an open question:**

1. The resolver **has no `default:` arm.** An unknown rung is a refusal that names the rung, the
   ladder, and the authored path — the shape `worldgenesis.go:328` already uses
   (`"%q's trait %q has strength %q, outside the closed set"`), preserved as the resolver's own message
   format so the 7 messages it replaces lose no information.
2. Every "empty means moderate" policy that exists today becomes **explicit ladder configuration**.
   There is one live instance: an empty `malleability` is permitted (`:320`) and mapped to 0.35
   (`:523-524`). Under one resolver that per-field policy either becomes one global default — a
   behaviour change nobody chose — or it is written down per ladder.

### C22 · `block` — the retarget removes the only thing that catches a hallucinated key, and re-creates the SPEC-035 silent drop

The brief's four input questions — **absent · null · wrong type · empty** — are not covered by any
mutation table, and this is where they land.

`world_genesis.v1.schema.json` sets `additionalProperties:false` at every level (`:7`, `:12`, `:39`,
`:54`), so a key the seat invents is refused by the structured-output leash, and `genesisDoc` is
field-for-field with it (*"Field-for-field with the schema; no field carries a number, which is the
whole point"*, `worldgenesis.go:72-73`).

`world_model/6` has no schema and no Go type (`PROPOSAL.md:23-24`), and Go's `encoding/json` **drops an
unknown key silently**. C3 makes this concrete rather than hypothetical: two of three encoders write
`arrivals[].place`, one writes `arrivals[].within`. Whichever the service type lacks is dropped, the
world arrives with no arrival extent, and nothing is said — which is precisely the defect the role
brief records for SPEC-035: `witnesses: "<uuid>"` as a bare string, committed with zero witnesses and
no `halt_reason`, found by asking the wrong-type question one commit too late
(`harness/roles/contracts-and-platform-expert.md:58-62`).

**Requirement:** the machine artifact is generated with `additionalProperties:false` throughout **and**
the service type decodes with `DisallowUnknownFields`. Otherwise R2 checks the schema while the
document goes unchecked, and C11/C13/C14 all become silent-drop bugs instead of loud ones.

---

## 5. Friction

### C23 · `block` — this round has no friction journal

`docs/00_workspace/friction/` holds exactly one file, `2026-08-26-live-friction-journal.md`, and all
nine of its entries (02:04:52Z–02:09:12Z) are about building the harness itself — the spine check,
`AREAS.map` indentation, `check_round`'s `Areas:` line. **None is about writing this proposal.**
Per `harness/roles/contracts-and-platform-expert.md:123-124`, a round with no journal is itself a
finding: either it met no friction, or nobody was watching what it cost. `ci/check_closeout.sh`
requires a `Friction:` line and this amendment carries none.

**I do not accept `Friction: none`,** and here is the entry that should have been written, with the doc
named against it (`gap`):

> **gap** — the section, facet, obligation and refusal counts for `world_model/6` are not stated in one
> place, so an author writing an amendment reconstructs them from `SCHEMA-v3.md`'s summary table — a
> table that is **already known to be wrong** in at least one row (`SCHEMA-v5.md:65-66`: *"`SCHEMA-v3.md`'s
> own summary table at line 19 says 21; that count was already wrong… Use the rows."*). A table wrong
> in one row and authoritative in the others is exactly how C8 happened, twice, in one document.
>
> **Doc to fix:** `SCHEMA-v6.md` already carries *"Reader-obligation count: 24 → 25"* (`:78`). It
> should carry the other three counts the same way — sections, facets, refusals — and
> `SCHEMA-v3.md:12-19`'s summary table should be struck or annotated per row, because it is currently
> the most citable and least reliable source in the set.

### C24 · Ruling on the friction ledger — row 6 is `WASTE`, upheld, and still live

`docs/00_workspace/friction-log.md` closes with: *"The next round that has a reviewer should rule on
row 6 first: it is the only clean WASTE, its fix is one check, and it has already cost two agents
time."* This review has a reviewer. **Ruling: WASTE upheld.**

Row 6 says `AREAS.map` membership is by indentation and an unindented line silently becomes an area
name. Verified still true, in both readers:

- `harness/check.sh:406` — `areas="$(printf '%s\n' "$map" | grep -E '^[^ \t!]' …)"`. Every unindented
  line is taken as an area name with no validation of any kind.
- `harness/check.sh:586` (the close-out reader) — `[ -n "$globs" ] || continue`. A phantom area with
  zero globs is **silently skipped**, so the second reader cannot catch what the first invented.

**Rule to fix, named:** `check_areas` must refuse an area name for which neither
`docs/areas/<name>.md` nor `harness/roles/<name>-expert.md` exists. That is one condition inside the
loop at `harness/check.sh:416-426`, and it converts a silent phantom into a named failure — which is
what the row asked for. The ledger's own control condition is satisfied: this is a reviewer ruling, not
a quiet fix.

### C25 · One rule EARNED, said out loud

*"If you cannot cite a rule ID, an ADR, a line of code, or a logged incident, you do not have a
constraint"* (`docs/00_workspace/round-protocol.md:341-346`). **EARNED by this review:** C8 and C21 are
both citations that do not resolve to what they claim, and both were found only by opening the cited
file and counting rows. Neither would have been caught by `ci/check_citations.sh`, which asserts an id
resolves and deliberately does not assert the citation is apposite (dossier `:162-166`). That limit is
already documented and needs no change — but it is the reason this finding exists, and it belongs in
the ledger as a catch rather than a cost.

---

## 6. Answers, stated plainly

**Q1 — does retargeting break a static guarantee?** Yes. Not through facet gating (C4 — that is a
false lead; leaf **existence** stays static and that is all R2 needs), but through four other things:
R2 has no schema artifact to diff against and the roadmap defers it into the increment that consumes it
(C1); four top-level sections define no keys at all, two of them under obligations (C2); the one
obligated section among them has already diverged across the three real documents, `within` vs `place`
(C3); and the entity's non-facet keys — including an open map keyed by world-invented channel names —
are enumerated nowhere (C5). Separately, R2's guarantee **weakens** at v6 even once computable, because
one recursive `entities[]` replaces disjoint typed arrays and nothing checks that a claimed leaf is a
parsed leaf (C7). **R2 does not die, but it does not survive untouched either, and `PROPOSAL.md` §2's
claim that it does is false.**

**Q3 — does the clean cutover strand anything?** Yes: nine capabilities, seven of them silent.
`tension` required→optional, disabling the journey (C11); the region, the coordinate origin and the 0.6
ring (C12); `world.tagline` and world cover art, plus `world.ornament` (C13); `attrs.max_load` and
encumbrance, with the increments ordered wrong for it (C14); the whole arrival-choice flow including
`newCast` (C15); the Ironmoor identifier-name guard (C16); and two arrival floor refusals that become
uncheckable sufficiency prose (C17). Plus the array ceilings that bound per-build cost (C18). The
cross-repo wire is **not** among them (C9).

**§9 Q1 — does `Refuse`'s resolver become the god object?** It gets worse: nine v6 rules quantify over
the whole document. **But there is a bound that holds, and the amendment already contains it** —
document validator takes the document and no resolver; `Refuse` takes the resolver and no document.
Make it a type boundary, not a sentence, and the rules cannot migrate (C20).

**§9 Q2 — SPOF and error legibility?** The question is aimed at the wrong target. The resolver replaces
**7** named refusals, not ~40; the cutover deletes **67**. The real risk is that two of the five
existing class→number conversions are uncounted and both carry a silent numeric default, and v6 has
thirteen ladders with no belt upstream (C21). The resolver refuses unknown rungs with no `default:`
arm, or the SPEC-035 defect class ships again (C22).

---

## 7. Mutation experiments — PREDICTED, not run

**Not run, and why:** there is no diff in this round, and `ci/mutate.sh` rewrites a source file, which
a review of a design document may not do. **A prediction is not a result.** The implementing round must
run these and report the table `ci/mutate.sh` prints; `ci/mutate.sh --selftest` (7 probes) first, so a
verdict from it is trustworthy.

| # | File:line | Mutant | Predicted | Why |
|---|---|---|---|---|
| 1 | `worldgenesiscommit.go:627` | delete the region `attrs.tension` write | **SURVIVED** | `trg_validate_tension` fires only when the key is present (`schema.sql:3748`); `beatBudgetSeconds` COALESCEs to `'none'` (`tension.go:58`) ⇒ ∞ budget, no test asserts a finite one |
| 2 | `worldgenesiscommit.go:666` | delete the cast `attrs.max_load` write | **SURVIVED** | `fn_apply_carry_change` clears rather than sets on NULL max_load, by design and by its own comment (`schema.sql:937-938`) |
| 3 | `worldgenesiscommit.go:889-892` | change the ring factor `0.6` to `0.05` | **SURVIVED** | AC-3 exists precisely because this world-feel constant has no owner and no gate (`prd_world_creation_depth.md:154-158`) |
| 4 | `worldgenesis.go:253-255` | delete the tagline refusal | **SURVIVED** | the consequence is a missing cover in `fillScenes` (`imagehandler.go:691-695`), which no genesis test exercises |

If any of the four is **CAUGHT**, the corresponding finding (C11, C14, C12, C13) is weaker than stated
and should be re-dispositioned in the implementing round. Report the table either way.

---

## 8. Dispositions

| # | Disposition | Finding |
|---|---|---|
| C1 | `block` | R2 has no schema artifact; the roadmap defers it into the increment that consumes it |
| C2 | `block` | `opposition[]`, `traces[]`, `epochs[]`, `arrivals[]` define no keys; O5 and O10 oblige two of them |
| C3 | `block` | the obligated section already diverged: `within` vs `place` across three documents |
| C4 | `accept` | facet gating is a false lead — leaf existence stays static; replace Q1's framing |
| C5 | `block` | entity non-facet keys enumerated nowhere; `senses` is an open map `LeafPath` cannot name |
| C6 | `gate` | "non-numeric leaf" is a no-op today and an undecided exclusion under v6; print the excluded set |
| C7 | `block` | one recursive `entities[]` weakens R2 from "parsed" to "written down"; §2's claim is false |
| C8 | `block` | 17 sections (16 frozen); 25 obligations against a 24-row audit — both citations misresolve |
| C9 | `accept` | the cross-repo wire is target-agnostic and does not break |
| C10 | `gate` | the new schema needs a `gen_payloads.sh` payload or an `INPUT_CONTRACT_SCHEMAS` row |
| C11 | `block` | `tension` required→optional; absence silently gives an ∞ beat budget (SPEC-030 regression) |
| C12 | `block` | no region ⇒ no coordinate origin, no radius, no 0.6 ring, nothing for `fn_distance` to climb to |
| C13 | `block` | `world.tagline` (⇒ no cover art) and `world.ornament` have no v6 source |
| C14 | `block` | `attrs.max_load` unwritten ⇒ encumbrance silently dead; increment 2 lands the ladder too late |
| C15 | `block` | `arrival_candidates`, `arrival.why` and `newCast` have no v6 equivalent |
| C16 | `block` | `identifierShapedName` — the Ironmoor guard — has no v6 refusal |
| C17 | `block` | the two arrival floor refusals become sufficiency prose with no checker |
| C18 | `gate` | no v6 array carries a ceiling; the tick-ladder assertion and per-build cost lose their bound |
| C19 | `accept` | `world.brief` → narrator is contract-independent and should not wait on increment 1 |
| C20 | `gate` | bound the resolver by type: validator takes the document and no resolver, `Refuse` the reverse |
| C21 | `block` | 7 not ~40 refusals replaced; 67 deleted; 5 not 3 conversions; two carry a silent numeric default |
| C22 | `block` | no `additionalProperties:false` and no `DisallowUnknownFields` ⇒ the SPEC-035 silent drop returns |
| C23 | `block` | this round has no friction journal; the `gap` entry and the doc to fix are named |
| C24 | `WASTE upheld` | `friction-log.md` row 6 is live at `harness/check.sh:406` and `:586`; fix named |
| C25 | `EARNED` | "cite or it is not a constraint" caught C8 and C21; no gate could have |

**Blocking count: 15.** The two that cannot be overridden with a sentence are **C1** (R2 is not
computable without the artifact, so the amendment's headline mechanism has no operand) and **C11 + C14
together** (two silent regressions of shipped, working behaviour, neither visible to any gate).

**Standing rule:** overriding any `block` requires a written reason in the PR body — a sentence someone
can later disagree with, not a conversation (`docs/00_workspace/round-protocol.md:329`).
