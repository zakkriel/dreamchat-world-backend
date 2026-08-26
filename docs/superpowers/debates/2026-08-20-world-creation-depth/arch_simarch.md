# Architecture round — simarch

The diagnosis is right: accretion, not composition. The contract as printed is wrong, and AC-3's own
gate fails at design time, before any migration is attempted.

## 1. HARD VETO — P1 kills two of the eight concepts on day one

Only two functions in the whole commit path insert `state_mutation`: `writeOpeningState`
(`worldgenesiscommit.go:596`) and `writeArrival` (`:768`). `writeNamingEvent` (`:357`) and
`writeHistory` (`:414`) write canon_event and perceptions only — deliberately: "The events are
AttributeChanged with no mutations… every belief is written explicitly" (`:404-407`).

So `history` and the naming event have no engine-read state target and **cannot be registered** under
P1. AC-3: "If any of them requires a special case, the contract is wrong." Two do. Forcing `history`
to invent a `stateWrite` manufactures mutations the replay property (`:570-572`) does not want.
`world` fails the other end — no entity, no event, no distribution: a prerequisite (`:76-78`), not a
landing.

**P1 is not a property of authored concepts — only of entity-bearing ones.**

## 2. HARD VETO — norms → `personality_core.traits` cannot carry a law

- **Name leakage (I-3).** `buildCognitionPrompt` dumps the trait blob raw: `sb.Write(m.Traits)`
  (`cognitionprompt.go:143-146`). Relabelling covers scene and imminent lines only
  (`batchDisplayLabels:269-294`, `relabelScene:298-308`); trait JSON is never scrubbed, and the Go
  `NamingWall` is an API-boundary belt, not prompt-side. A norm's `stated` is a free sentence that
  will routinely name people — "no one speaks to Kessa before she speaks first" — pushing an
  unearned name into every bound mind's prompt.
- **A fabricated magnitude.** `traits/1` entries are `{value: float, manner}` (`insertMind:516-525`),
  `value` from `strengthValue` (`worldgenesis.go:497-511`). A norm has no strength class, so the
  runner invents the number: P2 inverted, the engine authoring fiction.
- **Law inherits drift.** `malleability` governs how easily traits change; `trait_pool`
  (`schema.sql:3950-3956`) accrues them to a threshold. A law is not a disposition that erodes.

AC-6's selling point — "No cognition-prompt change, because the surface already exists" — is the
tell: the channel was chosen because it is already rendered.

**The hard thing:** no existing read path delivers a shared standing fact to a batch mind. Group-held
perceptions reach no mind and leak to the player (§6, correct). Per-holder perceptions are excluded by
`fn_private_records`' shared-CTE (`schema.sql:2707`) and reach isolated seats only
(`cognitionprompt.go:100-102,395-412`). Traits are unsafe. A channel must be **built** — one stable-
prefix section, one query. v2 rejected that as a smell while shipping a worse version of it.

## 3. `Operate` non-empty is validation in a costume

`Operate(item) []stateWrite` returns a slice per item at runtime; you cannot know at registration
whether it is ever non-empty without calling it — that is a test. AC-2 claims both "a program that
does not build" (line 81) and "a startup-time failure" (line 106); Go gives neither for a slice return.

Worse, it misses its namesake bug. `standing` is a **leaf**, not a concept: a `cast` landing writing
`location_id`, `coordinates`, `descriptor` registers happily while ignoring it. AC-2's "This replaces
v1's CI leaf-test entirely" is **false** — P1 is per-landing, the defect is per-leaf.

## 4. The interface cannot express five of eight concepts

Only `Refuse` takes a second argument. But `ways.connects` needs place uuids (`wayKey:344-346` derives
identity from two other concepts' names), `objects.where.carried_by` needs cast ids, `cast.starts_in`,
`history.who`, `arrival.place` likewise — all via `genesisIDs` (`:64-73`). Passing the resolver to
every member fixes it, but then `Operate` is a function of the whole document and P3's "one file"
erodes.

`Ground(item) eventSpec` also conflates two things. `cast` **creates no event**: `writeMinds` selects
the first history event the person took part in (`:471-491`), falling back to the opening moment
(`:496-499`). That is reference, not specification — and it is exactly the archivist site (`:550-555`)
the PRD claims becomes structurally unavailable.

## 5. "One transaction" contradicts durable-worlds

`commitArrival` is a second transaction, possibly another process, rebuilding ids from the database
(`loadGenesisIDs:224-256`) and re-running the cast concept for `newCast` grounded in the arrival event
instead of history (`:134-146`; `insertMind` shared by both, `:507-510`). "One runner inside the
existing single transaction" (line 62) is not the shipped shape — a third escape hatch AC-3 does not
anticipate.

## 6. What gets harder

- **Ordering coupling relocates, it does not vanish.** All state lands under *one* event with one
  monotonic `seq` (`:589-604`). Per-landing `Operate` forces the runner to interleave four landings
  into one seq space — the old function body's order, re-encoded as runner data.
- **AC-3's byte-identical diff is a second project, not a test.** The benchmark is plpgsql with fixed
  uuids and hand-placed coordinates (`20260813142100:100-101`); it never passes through a
  `genesisDoc`. Round-tripping requires reverse-authoring a document first.
- **`stateWrite` is three types.** `state_mutation` needs event_id + tick + seq; `personality_core`
  has no event column (provenance is `trait_provenance`); `pending_event` needs `fire_at_tick`. "The
  runner stamps provenance on every write" cannot be uniform.

## 7. What I would do instead

1. Split static from dynamic: `Declare() spec` — entity kinds, event kind, **engine-read paths**,
   dependencies — checked at registration; `Apply(ctx, world, ids, items)` does the work. Static
   declaration is what makes P1 real.
2. Bind the requirement to **leaves**: registration fails if any leaf of the concept's schema fragment
   is unmapped to a declared target. That kills `standing`; P1 does not.
3. Make grounding a sum type: `creates(eventSpec)` | `references(landing, selector)`, and forbid
   `references` selecting `world_genesis` unless the perception is a name. The archivist rule, enforced.
4. Replace "Operate non-empty" with "declare at least one reader": a state path, a perception holder
   set, **or** an event other landings reference. `history` then passes honestly.
5. Build the norm delivery channel explicitly. Do not launder it through `traits`.
