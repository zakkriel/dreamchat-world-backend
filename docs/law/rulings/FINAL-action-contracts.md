# FINAL — Action Contracts

**What this document is.** The complete, finite definition of what the deterministic engine may compute.
An action contract = the fixed arithmetic ("grammar") the engine runs for one kind of action, plus the
typed data ("vocabulary") the LLM may mint inside it. If a computation is not in this document, **the
engine cannot do it** — it routes to the resolution LLM. Any code or doc that grants the engine powers
beyond this table is wrong.

**The one principle above everything (founder-locked):**

> Deterministic machinery — the gate and the contract arithmetic — exists to **BLOCK impossibilities**,
> never to award success. Nothing gets a free pass. An action "happens" only because nothing stopped it:
> physics didn't forbid it, and the world — which saw it coming — chose not to or failed to. When nothing
> blocks, the engine **records** the surviving action (writes the event, updates position, ticks the
> clock). Recording is clerk work, not judgment. Deciding and recording are different jobs.

Why this matters to you, the implementing agent: it is tempting to read "the contract computes the move"
as "the move succeeds." That reading is wrong and was explicitly corrected. The contract answers *"CAN
this happen (in this time, at this weight, at this distance)?"* — a possibility question. Success is only
ever the absence of a blocker. The world gets its word before every step (see the loop PRD: world-first,
telegraphed intent); opposition by another actor is *that actor's own act* and is **always** resolved by
the resolution LLM, never by arithmetic.

**Second core rule (founder-locked) — state stores MEASUREMENTS; verdicts are COMPUTED, never stored.**

> Every persisted property is a measurement: `max_room`, `occupied_room`, `max_load`, `carried_weight`,
> `base_speed`, `coordinates`. A verdict — "has room", "within load", "fits the time" — exists only as
> arithmetic the engine runs at the moment of asking. **No verdict is ever a stored column.**

Reasoning: a stored verdict is a cached judgment, and cached judgments rot (`has_room: true` written at
noon is a lie by evening) — every read of a stale verdict is the silent-corruption class this system
refuses. Naming convention that enforces it: **nouns for state, verbs for checks.** If a property name
reads as an answer (`has_`, `can_`, `is_allowed_`), it is a verdict pretending to be a fact — rename it or
delete it. Boundary so the rule doesn't overreach: **statuses are legal state** — `encumbered`, `tied`,
`limping` are *conditions* with provenance and a lifetime, written by a licensed cause; they are inputs to
arithmetic, never pre-cached answers about an action. This rule is also what keeps the engine modular:
future modules can compute new verdicts over the same measurements forever; stored decisions compose never.

---

## 1. The two-lane structure

Every attempt, no matter what the player typed, passes a **structural floor** — cheap checks on fields
that exist on everything: does the target exist, is the actor conscious (`can-act`), is the target in
reach, is it accessible. This is answerable for *anything* because these fields are universal.

Then the lanes split:

- **Contract lane** — only for the action types below that have a contract. The gate *additionally* runs
  the contract arithmetic (time, volume, weight). Deterministic, no LLM. **Blocker-only.**
- **Everything else** — the infinite typed-anything space ("I cast a world-breaking ritual", "I read the
  whole novel", "I intimidate her") — has **no physics to check**. It goes to the resolution LLM. The
  engine contributes nothing but the structural floor and, later, the recording of whatever survives.

There is no third lane. There is no per-action special case.

---

## 2. Contract: `move` (ActorMoved)

**Grammar (fixed, engine-owned):**

```
duration = distance ÷ effective_speed
effective_speed = base_speed(movement_type) × Π(modifier factors)
blocker: duration must fit the beat budget (see §6 Tension) — else REJECT
```

**Vocabulary (LLM-minted, typed rows):**

```json
// a movement type — minted once, reused forever
{ "movementTypeId": "climb", "baseSpeed": 0.4 }        // meters per second

// a modifier — a percentage on specific movement types
{ "statusTypeId": "limping", "actionType": "move",
  "movementModifiers": [
    { "movementTypeId": "walk", "modifierPercent": -30 },
    { "movementTypeId": "run",  "modifierPercent": -70 } ] }
```

**Rules and their reasoning:**

- **Seeded defaults.** Every world is created with exactly one movement type — `walk`, 1.4 m/s — and one
  modifier row: `encumbered` (all movement −100%; written/cleared eagerly by the engine, see §4). These two
  are the only predefined rows; the encumbrance rule survives any world with gravity. Nothing else
  is predefined. Any other mode (run, swim, climb, fly, whatever a scene invents) is minted by the LLM on
  first need. Reasoning: movement types are a dynamic, open set — pre-authoring a list is the enumeration
  mistake this system exists to avoid; but the engine needs one default so a fresh world can compute at all.
- **Modifiers are percentages, stacking multiplicatively.** `-30%` means ×0.70. Example:
  `walk 1.4 × baby(-90%) × trained(+20%) × limping(-30%) = 1.4 × 0.10 × 1.20 × 0.70 = 0.1176 m/s`.
- **Floor: −100%. No cap.** −100% → speed 0 → duration infinite → never fits any budget → the action is
  prevented. This is how "tied → can't move" works: prevention is not a special rule, it *emerges from the
  arithmetic*. Below −100% is meaningless (negative speed) — mint validation rejects it. There is **no
  upper cap** (founder ruling): a +900% haste is legal data. What guards against a garbage mint ("sprint,
  400 m/s") is not a numeric bound but the three nets in §8.
- **If a movement type is not listed in a modifier, the modifier does not affect it.** `limping` says
  nothing about `swim`; swimming is unimpaired.
- **Over-budget = REJECT (v1).** The move does not happen; nothing is written; narrated as "not in this
  moment." Reasoning: the honest alternative — a partial move ("you get halfway") — requires representing
  a position *between places*, which the location model cannot do yet. When route topology exists
  (SPEC-018's deferred tail), the same arithmetic splits long moves into interruptible journey legs; the
  contract does not change, only the location model gains waypoints. Do NOT implement clock-jumping a long
  move through as one atomic step (except under `none` tension, §5): an atomic multi-minute move denies
  the world its turns mid-journey, and world-first is the loop's core rule.

**Worked example (blocker, not resolver):** Kade, limping, tries to cross a 20 m room while frantic
(budget 5 s). `1.4 × 0.70 = 0.98 m/s → 20 ÷ 0.98 ≈ 20.4 s > 5 s → REJECT`. Note what did *not* happen:
the engine did not decide Kade "failed dramatically" — it blocked an impossibility. If the budget were 60 s,
the move passes the gate — and still only *happens* if no one and nothing stops it first.

---

## 3. Space (the distance input for `move`)

**Model: nested coordinates on the existing location hierarchy.**

- Every location has a **coordinate within its parent** (the tavern has a coordinate inside the docks
  district; the district inside Vael; and so on up the breadcrumb).
- Things *inside* a location — the door, the bar, **and actors** — have coordinates within that location.
  (This adds one piece of state: an actor's position within their current scene. Today an actor's location
  is only the scene id; this contract requires the finer position, updated by each committed move.)
- **Distance between two things = distance between their coordinates at the nearest common parent.**
  - Door → bar: both inside the tavern → tavern-local coordinates. Say 12 m → walk ≈ 9 s.
  - Bar → alley: nearest common parent is the district → the tavern's and the alley's district-level
    coordinates. Say 800 m → walk ≈ 10 minutes.
  One formula, every scale, barroom to continent. No special in-scene case.
- **Accepted imprecision (do not "fix" this):** the bar→alley distance ignores the walk-to-the-tavern-door
  part. Correcting it would require inferring an exit point, which is reasoning, which is an LLM — and this
  path is deliberately LLM-free. The hierarchy makes the error marginal by construction: a child location
  is small inside its parent (15 seconds of barroom inside a 10-minute walk). State this in code comments;
  an agent who "improves" it with exit-point inference is breaking the design.
- **Coordinates are LLM-minted, never hand-authored.** A newly created place is minted *with* its
  coordinate-in-parent, validated like any typed data (a coordinate must exist and lie within the parent's
  extent). The hand-placed seed world is a test artifact, not the pattern.

---

## 4. Contract: `ObjectRelocated` (pick up, drop, give, put)

**Grammar (fixed, engine-owned) — two dimensions, exactly (founder-locked), with different roles:**

```
VOLUME — blocks. object size 1–10; volume(size) = 4^(size-1); container has max_room.
         room-check (computed at ask-time, never stored): occupied_room + 4^(size-1) ≤ max_room
         A size-5 crate cannot enter a size-2 pouch: geometric impossibility. True blocker.

WEIGHT — never blocks; it CONSEQUENCES. Grabbing two tons is not impossible — moving with it is.
         effective_weight(container) = (empty_weight + Σ effective_weight(contents)) × weight_modifier
         EAGER RULE (founder-locked): on any commit that changes a carry chain (grab, drop, hand over,
         item added to a held container), the engine recomputes carried_weight for every affected
         carrier — recursively up the chain — and writes/clears the seeded `encumbered` status in the
         SAME commit. carried_weight > max_load → encumbered (movement −100%, i.e. full stop).
         Provenance: the status traces to the exact event that caused it. The world sees the strain
         the moment it happens — cognition and perception read a true state, never a stale one.
```

**Rules and their reasoning:**

- **Two dimensions only.** No third axis, no modes (a future "drag vs carry" would attach to capacity —
  a deliberate extension, not assumed). The dimensions play different roles by principle: volume states a
  geometric impossibility (blocker); weight states a physical consequence (status). `within-load` as a
  blocker is **dead** — do not reimplement it; "REJECT: too heavy" was the system refusing, "you seize the
  crate; you cannot move an inch" is the world responding. The black hole: it *fits* the pouch (volume
  passes) — and the carrier is flattened, because the recursive formula passes the mass through to whoever
  holds the pouch and the eager `encumbered` write does the rest.
- **Capacity values are static measurements.** `max_load` (a person's) and `max_room` are plain attributes
  — **no status math touches them** (founder ruling: no stats system). Genuine weakening is an
  `AttributeChanged` event lowering `max_load` — existing machinery. A *container's* `max_load` is dormant
  data for now: a pouch tearing under the black hole is a future consequence, adjudicated when ruled —
  not v1 engine behavior.
- **The container formula (founder-specified).** A container's own weight is its *empty* weight. What the
  carrier feels = (empty weight + everything inside, recursively) × the container's weight modifier.
  Mundane container: modifier = 1. The modifier wraps *both* terms deliberately: a soaked backpack's own
  fabric is heavier too; a lightening enchantment lightens the whole thing.
  Worked, with eager: pack (2 kg empty, ×1.0) holds 4 waterlogged crates (25 kg × 1.6 = 40 each). Kade
  grabs the pack → commit updates his `carried_weight` to `(2 + 160) × 1.0 = 162` → 162 > his `max_load`
  80 → the same commit writes `encumbered`. The grab **happened** — canon says he holds it. His next move
  hits `walk × (−100%) = 0` → blocked by arithmetic. He can drop it (relocation needs no move — see
  below) → that commit clears `encumbered`.
- **Mundane containers: `max_room ≤ 4^(size-1)`** — a thing cannot hold more than its own volume —
  enforced at mint. A bag of holding is *not an exemption*: it is a container minted with a **volume
  modifier** on its contents, the same mechanism as the weight modifier. Magic is a modifier, not a rule
  change.
- **Relocation has NO time cost of its own (founder-locked).** Moving an object somewhere is always
  preceded by moving *yourself* there — and that move carries all the time. "Put the crate in the
  storeroom" only works if the player is (or gets themselves) there: relocation's structural floor
  requires `in-reach` — attempted from the bar, it fails with "you're not in the storeroom," and the
  player walks there as their own stated action (the Decomposer adds nothing — see FINAL-decompose). The
  time then lives in that stated move. A handover to someone beside
  you: no move needed → no time. (The old `fits-time` on the relocation row was a hallucination; it is
  dead. There is no formula for "handing a note takes N seconds" and none is needed.)
- **Encumbrance (weight slowing your move) is explicitly NOT built.** The move carries you and the object
  at normal speed. Known simplification; revisit only when ruled.
- **Throwing** (relocating without going there) breaks the in-reach precondition and has **no contract** —
  it routes to the resolution LLM until a contract is ruled.

---

## 5. Artifact types and the two tiers of attributes

### 5.1 Why types exist

Artifacts are generic entities. Without types, two failures are guaranteed:

1. **Vocabulary drift → decorative canon.** A ruling locks the door and writes `locked = true`. Five beats
   later another ruling — or the engine's own accessibility check — looks for... what? `locked`? `sealed`?
   `barred`? With free-form attributes, nothing pins the name, so state can be written once and never read
   again. The lock *exists in canon* and still doesn't work. This is the exact frustration case that drops
   users ("I locked it — the NPC just ran out through the open door"), and committed facts alone don't
   prevent it; **shared vocabulary does**.
2. **A stored-verdict violation.** Early sketches had `location.reachable = false` — an answer-shaped
   column, forbidden by the core rule (§0). With portals, `reachable` dies as a column: a location is
   reachable if a connecting portal's measurements permit passage — computed fresh at every ask.

The fix is the same move as the physics contract, applied to artifact semantics: **one TYPE, many
instances.** "Door" is not a type. **Portal** is a type; every door, window, gate, bar-hatch, and
shimmering rift is an instance of it. We never pre-create the world's in/out points — we define the
property set once.

### 5.2 The two tiers — and the wall between them (founder-locked)

Every artifact's attributes split into exactly two tiers:

**Tier 1 — engine-known attributes.** The small, fixed set that feeds deterministic checks. They exist so
*known checks* can be computed: a portal's `open`/`locked`/`connects`, a container's
`max_room`/`occupied_room`/`empty_weight`/modifiers, every object's `size`/`weight`, an actor's
`max_load`/`coordinates`. **Closed set.** It ships with the seeded types and grows only when *we* add a
check in code — never at runtime, never by mint.

**Tier 2 — LLM-invented attributes.** Everything else the Resolver mints to track the status quo of
things: `desecrated`, `bloodstained`, `humming_faintly`, `trusted_by_the_guild`, `barred_from_inside`.
Open and infinite. Written by rulings, carried in payloads, read by future rulings — so the world stays
coherent in judgment-space. **The engine never reads them.**

The wall, as two mirror rules for the implementing agent:

> **Rule A — the engine may only read attributes named in this document.** An engine check that reads any
> attribute the contracts don't define is a bug, full stop. "Just one more field" is the enumeration
> fantasy sneaking back in.
>
> **Rule B — the LLM may freely invent Tier-2 attributes, but may not write Tier-1 by invention.** `locked`
> changes through a committed, shape-validated ruling on a portal — never because a ruling's prose
> mentioned it. Tier 1 is exactly where validation is strict; Tier 2 is exactly where it is not.

This is the third face of one design: physics grammar vs minted vocabulary (§0), measurements vs verdicts
(§0), engine-known vs LLM-known state (here). Same split every time.

### 5.3 Seeded engine-bound types: Portal and Container — and only those

An **engine-bound type** is one whose Tier-1 properties feed a deterministic check. The check is code, and
the LLM cannot mint code (same line as A11: vocabulary yes, grammar never). So engine-bound types are
**seeded with every world**, like `walk` and `encumbered`. The seed set today is exactly two:

**Portal** — the traversal type. Instance properties (Tier 1):
```
connects: [location_id, location_id]   // the two places this portal joins — the adjacency graph
open:     bool                          // current position
locked:   bool                          // current locking state
```
The traversal check (computed at ask-time, never stored): a move from location A to location B passes the
accessibility floor iff a portal connecting A↔B permits passage (`open` and not `locked` in v1). This is
what makes the locked door real *for everyone* — player, NPC, world — five beats or five days later.

**Boundary (founder-locked): Portal is accessibility, NOT geometry.** A portal gates *whether* you can
move between places. It does **not** contribute exit points, doorway positions, or distance changes — the
cross-level distance imprecision stays accepted exactly as §3 states it. An agent who "improves" portals
into spatial geometry is reopening a decision that was deliberately closed.

**Container** — the holding type. Its Tier-1 properties and both checks (room, weight/eager-encumbrance)
are already fully specified in §4; the type system adds nothing new there — it *names* what §4 already
built. Container was the first typed artifact; Portal is the second.

**Single-type per artifact (v1).** An artifact is a Portal or a Container or an untyped thing — not both.
The walk-through wardrobe (portal AND container) is a real future case; multi-typing is deferred until an
actual instance demands it. Do not build the general mechanism speculatively.

### 5.4 Vocabulary types (mintable)

The LLM may mint **vocabulary types** — property schemas with no engine check behind them ("Shrine" with
`desecrated: bool`, "Ledger" with `entries_burned: int`). Their value is pinned naming: once minted, every
instance and every future ruling uses the same fields, killing drift inside judgment-space too.
Reuse-before-create applies (match against existing types before minting a new one — the same
identification discipline as entities). The engine treats every vocabulary-type property as Tier 2:
carried, never read.

### 5.5 Why Tier-2 does not create a hole: the layered nets

Tier-2 attributes are invisible to the engine's structural floor — and that is fine, because **there is no
LLM-free path through this system**, and the LLM seats all see Tier-2. Walk the worst case — a door barred
only as ad-hoc state — and count the nets:

1. **A barred door is closed — the floor blocks the move itself.** "I go through the door" reads the
   portal's Tier-1 `open = false` → blocked mechanically → "the door is closed." No LLM needed, nothing
   added by anyone (the Decomposer adds no steps — see FINAL-decompose).
2. **The player's follow-up "I open it" is always adjudicated** (§7). The resolution LLM reads the full
   state *including Tier-2* `barred_from_inside` → prevents it with meaning: "it won't budge — something
   is blocking it from the other side." This is the always-adjudicated rule doing exactly the job it was
   made for.
3. **Before and between every step:** world-first cognition saw "he's heading for the door" (telegraphed
   intent) and every present NPC could act on it.

The only way to ghost through is Tier-1 saying `open` while Tier-2 says `barred` — which is not a player
capability but a **corrupted write**: whoever ruled the barring wrote incoherent state. That is exactly
what the discipline rule forbids:

> **If a ruling intends a fact to physically stop people, it must ALSO land in the Tier-1 field**
> (`locked`, `open`, `connects`). Tier-2 carries the meaning ("barred from inside with the oak beam");
> Tier-1 carries the mechanics. Write both or the state is incoherent.

So the engine floor is the **last** net, not the only one. Do not read this section as "Tier-2 blocking is
optional" — read it as: the nets are layered, and the discipline rule keeps them agreeing.

---

## 6. Tension (the beat budget)

**What it is.** Tension is an attribute on the scene. The engine maps it to the beat budget — the time
window an action chain must fit (`fits-time` reads it):

```
frantic = 5 s · tense = 30 s · normal = 60 s · calm = 600 s · none = ∞
```

Values are data, retunable. The shape is the rule.

- **The budget is consumed cumulatively across the chain (founder-locked).** Each resolved action eats its
  duration from the beat's budget; the next action in the chain only runs if budget remains. Three
  12-second moves in a 30-second beat: the first two pass (12, 24), the third rejects (36 > 30) — the
  chain halts there, the first two stand. Per-action checking ("each move alone fits 30s") would make the
  tension tiers a lie.
- **Fitting the budget is JUST another blocker.** It means one specific impossibility — not enough time —
  didn't fire. It never *awards* the action: the world still gets its word before the step, and everything
  else can still stop it. Do not read "fits" as "happens."

- **`none` is the deliberate time-skip** (founder ruling): "we travel three days" is not blocked by the
  gate. Known limitation, stated so nobody "fixes" it: under `none` the skip is **atomic** — world events
  scheduled inside the skipped span fire as the clock crosses them, after the fact; nothing interrupts
  mid-journey until journey-split exists. Adjudicated actions under `none` are still judged — the budget
  input is simply unbounded; "might not resolve as the person wants" still holds.
- **Who writes tension (founder-locked flow):** the **LLM sets it at scene creation** (one more typed
  value in the scene mint). Then, at the start of resolving each new act, the LLM seat that is already
  open — the **resolution LLM** on adjudicated acts, the **world/NPC cognition seats** on their steps —
  receives the scene recap (old tension + all committed events), reviews tension, and the act resolves
  under the **new** value. Zero lag: tension applies to the act being resolved, not the next one. No
  dedicated review call exists; no seat watches mood freely (that would be an unlicensed writer). The
  **Decomposer never touches tension** — it decomposes and assigns ids, nothing else (founder ruling:
  any second job biases its reading of the player's words).
- **What the engine validates:** enum membership (one of the five values — else the standard
  untrusted-proposal treatment: repair ×1, then bounce) and provenance (the write stamps to the act that
  triggered the review). Nothing else — no "tension can't jump two tiers" physics; mood is not arithmetic.

**Long actions under a finite budget (founder ruling — "Way A"):** for adjudicated actions with no
computable duration ("I perform the six-hour ritual", "I read the novel"), the **budget is an input to the
ruling**, and the ruling fits itself to the beat: *"you begin — the first sigil is drawn."* Progress
accumulates as plain `AttributeChanged` writes (`ritual_progress: 1/6`) until a threshold completes it —
the same accumulate-until-threshold shape damage uses. The engine owns the clock; the LLM never invents a
duration the engine must swallow. The world can interrupt between beats — which is the point.

---

## 7. Declared contract-less (the engine has NO math here)

| Action / event type | Structural floor | Everything else |
|---|---|---|
| **Communicated** (say/show) | `can-act`, channel not blocked — production only; you can shout into an empty room | Reception/comprehension is perception's job; any *effect* of what was said is the resolution LLM's |
| **Ownership/AccessChanged** | `exists`, `has-authority` | No arithmetic exists or is permitted |
| **EntityCreated / Destroyed** | none — not a gated action | Adjudicated intent, always the resolution LLM; the create-write is provenance-guarded |
| **AttributeChanged** (actor-driven: "open the door", "drink the potion") | `can-act`, `exists`, `in-reach`, `accessible` | **Always adjudicated (founder-ruled, v1).** No trivial-sorting exists: the engine cannot tell "open the door" from "drink the potion" without reading meaning (forbidden), and the Decomposer does not sort (it decomposes and assigns ids, nothing else). Judging everything costs one LLM call on trivialities and can never be wrong. Revisit only with real volume data. The payoff is cohesion: every effect lands as a committed attribute with provenance — a locked door is locked for *everyone*, five beats or five days later |
| **Throw** | (see §4) | No contract → resolution LLM |

If you find yourself writing engine code that "resolves" any of these, stop — you are inventing powers.

---

## 8. Minting: validation and the three nets

**Units are fixed system-wide:** meters, seconds, kilograms (speeds in m/s). A mint in other units is
invalid data; conversion is the minting seat's problem, not the engine's.

**Mint ordering:** a modifier row may only reference movement types that already exist (seeded or
previously minted); a reference to an unknown type is a shape failure → the standard repair-then-bounce.

Mint-time validation is **shape + derivable bounds only**: base speed a positive number; modifier ≥ −100%;
size 1–10; mundane `max_room ≤ 4^(size-1)`; a coordinate within the parent's extent. There are
deliberately **no plausibility bounds** ("is 400 m/s too fast?") — plausibility is judgment, and judgment
was already exercised: what makes trusting the mint acceptable is *where it sits*, guarded by three nets:

1. **A mint only happens inside an adjudicated ruling that already passed the reality check** — "sprint at
   400 m/s" had to survive the same plausibility judgment as "a novice forges a legendary greatsword."
2. **Blast radius = one logged row.** Mint once, compute forever means a bad mint is a single correctable
   row with provenance — repaired by compensating event/merge, never deleted.
3. **Every mint is audit-trailed** to the ruling that produced it.

---

## 9. What this document does NOT cover (owned elsewhere)

The resolution LLM's call mechanics (inputs, output schema, contested resolution, repair) —
`FINAL-resolve.md`, OPEN. World/NPC cognition — `FINAL-world-npc-cognition.md`, OPEN (SPEC-012).
Commit and perception fan-out — `FINAL-commit-perception.md`. The loop itself — the FINAL loop PRD/diagram.
