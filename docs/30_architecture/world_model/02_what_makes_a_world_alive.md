# What makes a world alive — the intention, and the distance to it

**Date:** 2026-08-26 · **Status:** synthesis, not a decision record. Written after reading the whole
genesis/creation corpus: the vision and PoC criteria, the world-model contract v2→v7, both creation PRDs,
the canon-engine set, the 39KB UX spec and six compendium PRDs, the twelve test-world briefs, the three
genesis encodings, and the engine code.

**Why it exists.** The contract work had drifted into mechanics. This puts the product intention back on
top, and states the distance to it in product terms rather than as a capability ledger.

---

## 1. The intention, in the founder's own words

**Philosophy** (`00_strategy/01_product_vision_and_promise.md:9-15`): *"A world that lives with you, not
for you."* The player is part of the world and **not its centre by default**.

**The PoC's central question** (`02_poc_scope_and_success_criteria.md:13`): *"Can one small world remember
important past events and use that memory to evolve coherently when in-world time passes?"* Two layers —
memory works, then living-world continuity works on top of it.

**Success is emotional** (`:751-759`): *"This world remembers what matters. The people inside it have
their own context. Time changed things. The world did not become confused. I can return to it and still
believe in it."*

**The seven pillars** (`01_product_vision:71-141`): persistent world · dynamic NPCs whose importance grows
· memory weighted by relevance · time that passes only in the fiction · **backstage updates** · a narrator
that represents the world · correction that feels like world direction.

**The contract's own test for aliveness** (`A_world_by_simarch.md:3-5`), and the best single sentence in
the corpus:

> **"If every character froze, what still happens, and how does a player find out?"**

**The bar for a world being open** (`SCHEMA-v4.md:62-63`, also principle P7):

> **"Every boundary a player reaches must be a fictional boundary, never an authoring boundary."**

That one is load-bearing for a reason worth restating: **the player cannot tell the two apart.** A locked
door and an unauthored edge both read as *"I can't go there."* So a single authoring boundary
retroactively makes every real refusal suspect — after the papier-mâché, every "no" reads as the machine
running out rather than the world holding a line.

---

## 2. What "alive" actually decomposes into

Sixteen concrete beats are specified across the UX corpus. They divide cleanly, and the division is the
whole finding.

**Built, and real today:** you come back and the sidebar knows what you were mid-sentence in · you address
someone and they don't obey (*"Targeting should not guarantee obedience"*) · the world decides who speaks ·
your six-step plan gets three steps in before something pushes back · the world refuses to tell you
something and simply doesn't have it in the payload.

**Specified and not built:** something you knew becomes *unreliable* rather than vanishing · you watch your
own past understanding be wrong, v1→v2→v3 · three sources disagree and nothing decides for you · news takes
time to reach you · you cross an in-world gap and the world has moved *for reasons* · a place remembers
being different · things wear out and run down.

**The one-line diagnosis:** **space is built, the wall against omniscience is built, time and doubt are
not.**

---

## 3. Where the corpus already agrees with itself, twice, from opposite ends

Two independent readings — one from the engine code, one from the world briefs — reached the same missing
mechanism without contact.

**From the engine:** `pending_event` is the ledger of pre-caused world truth. `fireDuePending`
(`ledger.go:159-223`) fires it correctly inside a clock crossing, atomically. `fn_world_slice` surfaces it
to the World Actor. It is tested. **Every `INSERT INTO pending_event` in the repository is a test.** And
`world_actor.v1` has no field that could schedule anything.

> **The world can interrupt. It cannot plan.**

**From the world briefs:** every tier-2 and tier-3 brief carries five to seven *semillas* — armed
situations with no trigger. *"La Ochenta y Tres deja de golpear una noche, de golpe, y el silencio es
peor."* *"Vira Cor llega a la noche catorce."* **No encoding carries a single one.** And without them,
*"what changed while time passed"* has nothing to draw on but the last thing the player did.

**A world with no future cannot have a past that moved.** That is pillar 5, and it is one producer away
from existing — not a rewrite.

---

## 4. Five mechanisms shipped as tables instead of mechanisms

The engine's shape is further along than a capability count suggests. Each of these exists in the schema,
is correct in shape, and is driven by nothing:

| Mechanism | Where | What it would give the product |
|---|---|---|
| `world_pressure(accrued, threshold)` | `schema.sql:4307-4313` | pressure that **builds and latches** instead of a timer that re-arms |
| `trait_pool(accrued, threshold)` | `:4121-4127` | a character who **changes through play** — pillar 2's only substrate |
| `perception_record.importance` (default 5.0) | `:3124` | memory weighted by relevance — pillar 3, and the depth-by-relevance that pillar 5 sorts by |
| `confidence` (default 1.0) · `distortion_level` (default 0) | `:3115-3116` | doubt. Every belief in the world is currently certainty |
| `invalid_tick` · `expired_at` | `:3119-3121` | *"Last known… you have not confirmed this recently"* — the most-repeated phrase in the UX corpus |

The last three are the **cheapest unlock in the corpus**: the read paths, the indexes and the filters are
already built and already correct. Only the writes are missing.

And `06_context_assembly_spec.md:32-40` already **decides** the memory scoring model —
`w_r·0.995^hours + w_v·relevance + w_i·importance/10`, mandatory inclusions, epistemic framing per record.
None of it is code. `perception_record.importance` is the fossil of a decision that was made and never
executed.

**Standing warning:** this codebase has shipped a table instead of a mechanism five times. A sixth is not
progress.

---

## 5. Three tuning bugs with large product consequences

Not architecture. Numbers, already in the seeds, each producing a wrong world.

1. **The world's own life is effectively invisible.** `fn_pressure_chance` seeds
   (`schema.sql:3711-3714`) put the cap at **small ≈ 70 minutes** of fiction, **medium ≈ 5.8 hours**,
   **large ≈ 70 days**. In a normal session the world offers small interruptions and nothing else.
2. **Everything reads stale after a minute.** `fn_compendium_decay`'s horizon is 72 **ticks**, and a tick
   is a second (`:1346-1350`). So every compendium fact older than ~72 seconds of fiction renders as
   stale. That is not memory decay; it is a mislabelled constant — and it actively poisons the one
   uncertainty surface that does ship.
3. **The class resolvers fail open.** `fn_extent_class_metres` returns **50 m** and
   `fn_duration_class_seconds` returns **2 s** on an unknown class word, under a comment reading *"never
   fails closed."* In a product whose promise is a coherent remembered world, a silent wrong number is
   worse than a refusal: it yields a world confidently wrong about distance and time.

---

## 6. The constraint that is law, not backlog

`06_rules_register.md:35`, **B-11**, ratified 2026-06-17: *"an NPC's beliefs and appraisals update only
when the actor perceives something, never on a free-running idle loop."*

So "NPCs live their own lives" is not an unbuilt feature — it is a **written prohibition**. Any autonomy
work must supersede a rule first. Related: **B-3** bars relationship UI in the MVP, which is why the dead
relationship projection (`schema.sql:452-454`, an explicit `NULL;`) buys nothing a player can feel yet.

Both are product questions before they are engineering ones.

And one enforced rule inverts the promise directly: the World Actor is **refused any intrusion that does
not manifest at the player's current scene** (`worldactor.go:122-142`). The world's own life is, by
construction, only ever visible in the player's line of sight — the precise inverse of *"the world feels
bigger than me."*

---

## 7. What genesis must invent, measured

The twelve briefs answer this precisely, because each world exists at three densities. **The 400-word brief
is not a weaker world — it is the same world with six specific things removed**, and those six are the
genesis workload:

1. **The quantity under a stated rule.** Tier 1: *"El que no duerme cuatro noches seguidas empieza a soñar
   despierto."* Tier 3: a five-rung ladder. Genesis mints the ladder beneath the rung.
2. **Institutional vulnerability.** Absent at tier 1 in all four worlds. *"Without this a faction is
   furniture; with it, a faction is something a player can push over."*
3. **The secret — and at tier 3, its cost to the keeper.** *"Esconde que perdió el oído del suelo"* → *"y
   que sus resultados no bajaron, lo cual lo tiene aterrado."*
4. **A record with a hole in it.** Tier 3 only, all four worlds. *"un hueco de dieciocho meses, hace
   sesenta años, sin ninguna entrada."*
5. **Negative canon.** No tier-1 brief has a *"Lo que no existe"* section; every tier-2 and tier-3 brief
   does, and they are long. These are the refusals that stop a narrator solving the premise.
6. **Armed situations with no trigger** — the *semillas* of §3.

---

## 8. Los Andantes has never been encoded

The world that most stresses the model — nine living creatures 44–90 km long walking a shallow ocean with
cities on their backs, where *"the political geography of this world is a calendar"* and the ground itself
can sicken and sink in six days — **has never been put through genesis at any tier.**

Its primitives were adopted into the contract anyway: `motion.trajectory` (`SCHEMA-v3.md:100`) and
`passage.hazard_class` (`:102`). Both remain unexercised — the reader round's own sheet records
`motion.trajectory` as *"unexercised — no entity in this document has motion."*

**The contract carries features designed for a world nobody has built.** That is the cheapest available
test of whether the contract is real.

---

## 9. What an amazing genesis would do that none of the three encodings did

From the world-brief reading, and marked as judgement rather than record:

- **Derive the negative canon instead of waiting for tier 3 to hand it over.** One encoder refused to
  infer any, and the miss is specific: the highest-value inference available in that brief was *"the dream
  has never predicted anything"* — which both later tiers state as a hard rule. Without it the disputed
  event has no norm to violate, and a narrator makes the dream prophetic by the third scene. **A genesis
  that cannot derive a world's refusals from its premise cannot protect that premise for one session.**
- **Emit three arrivals at different distances from the detonante**, and make at least one of them someone
  the detonante happens *to* rather than *through*. This is pillar 1 in mechanical form: if the player is
  not the world's centre, the world must be authored as something a protagonist is drawn *from*.
- **Emit the scheduled future** — armed situations, triggers in the same class vocabulary as accumulators,
  and the channel by which the player would learn it fired. §3's convergence, from the input side.
- **Ship the inference dependency graph, not a flat load-bearing list.** Pillar 7 needs the edges:
  correcting *"the house is starving of people, not grain"* must retract the accumulator, the indicator
  that reads it, the condition, and the opposition, in one gesture. As shipped, correction leaves orphaned
  derivations and a world that argues with itself.
- **Cross the premises the brief states separately.** Grelda states, in adjacent paragraphs, that houses
  eat *people-noise* and that reputation travels through the floor and cannot be cleaned. Together those
  make **two competing reputation systems with opposite latencies** — the Junta's ledger, instant and
  written and correctable; the floor, weeks-slow and indelible — so a person can be clean in one and
  finished in the other. No encoding has an entity holding opposite standings in the two systems.
- **Classify the world's indifference.** All four briefs say, in their own register, that the world is not
  aimed at the player: *"la mayor parte del daño que causan es indiferencia, no crueldad"* · *"no tienen
  intención dirigida hacia los habitantes"* · *"No hubo diluvio ni castigo divino: hubo una estadística"*
  · *"todo el daño es exposición."* Emit whether the antagonist is a **process, an institution, or a
  crowd** — it is the single field that stops a narrator inventing a villain.

---

## 10. Contradictions to settle, because both sides are live

1. **Two incompatible genesis documents are both current.** `prd_world_creation.md` commits to
   `places / cast / objects / ways` — which is `world_genesis/1`, structurally **v1 of the world model,
   the version that died**. `world_model/2-7` replaced it with one recursive `entities[]` and eleven
   facets. Nothing in the PRD acknowledges the replacement.
2. **`tagline`:** authored fiction the service never composes (PRD) vs derived from `premise`
   (`SCHEMA-v7.md` §7). Direct opposition.
3. **Numbers:** the seat emits *"no number of any kind"* (PRD AC-7) vs `exemplar` is *"fiction and may
   contain a number"* (`SCHEMA-v4.md` F1). One must move.
4. **Correction vs append-only canon:** confirmation is *"the only correction window that exists"* and
   amending re-infers dependents, against *"Canon is append-only"*. Reconcilable only if confirmation is
   strictly pre-commit — which the pipeline diagram does not say.

---

## 11. The order that buys the most felt aliveness

Product-first, with the engine evidence behind each. Items 1–3 are small; each is visible to a player.

1. **Give the world a future.** A production writer for `pending_event`, and `fire_at_tick` on the World
   Actor's schema. The reader, fire window, atomic flip and ledger view are built and tested. This is the
   first time the world will contain something that has not happened yet, and everything about *"time
   mattered"* starts here.
2. **Fix the three numbers of §5.** The pressure tiers, the 72-second decay horizon, and fail-closed
   resolvers with a named cause. Hours of work, and today they produce a world that is inert, amnesiac and
   confidently wrong about distance.
3. **Write the three dead columns** — `confidence`, `distortion_level`, `invalid_tick`. The Known lens's
   four registers currently have one register of backing data. Until these are written, the honest render
   is a wall of certainty, and any richer prose is the narrator inventing epistemic status the store cannot
   defend.
4. **Write `importance` at perception creation and score retrieval by it.** The scoring model is already
   specified. Until this lands, an NPC cannot remember a promise about someone who is not in the room, and
   pillar 3 is unimplementable.
5. **Let the world act where the player is not, and show it on return.** Lift the current-scene
   enforcement to a *reach* rule, fan those events to whoever was there, and the `'backstage'` origin
   finally has a producer. This is the item that buys *"the world feels bigger than me"* — and it must
   follow item 1, because a world that can only act *now* has nothing to have done while you were away.
6. **One elapsed-time state resolver, failing closed**, replacing the two hand-welded instances and giving
   decay, degradation, stocks, cycles and unmet needs a single home.

**Not on this list, deliberately:** relationship projection. It is a real hole under pillar 2, but B-3
bars relationship UI in the MVP, so it buys nothing a player can feel until that rule changes. A product
question, not engineering work.

**And the cheapest test of the contract:** encode Los Andantes. It exercises the two primitives nothing has
ever run on.
