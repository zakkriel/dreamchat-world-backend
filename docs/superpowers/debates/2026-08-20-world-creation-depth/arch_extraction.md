# Architecture round — extraction

## Ranked findings

**1. [HARD VETO on AC-3 as written] `history` cannot satisfy P1 — its `Operate` is correctly
empty today, by design.** Verified against the PRD's own round-trip fixture: in
`20260813142100_world_templates.sql`, every backstory event (M-E1…H-E1, lines 141-167) gets a
`canon_event` row and `perception_record` rows (240-257) and **zero** `state_mutation` rows —
only the scene-genesis event writes state (274-300). `worldgenesiscommit.go:404-407` confirms
this is deliberate: "AttributeChanged with no mutations... nothing fans out automatically."
History's whole job is provenance + knowledge, no engine-read state, and that's *correct*, not a
bug. P1 says a landing with empty `Operate` "cannot be registered — a startup-time failure." Migrate
`history` honestly and AC-3's own proof (byte-identical round-trip) fails on day one; give it a
fake `state_mutation` to type-check and you've built exactly the "authored-but-inert" P1 exists to
forbid, just laundered through a dummy write instead of a dropped field. **This is the escape
hatch** (answers Q1), and it directly answers Q2: P1 is not structural, it's a heuristic
("concepts usually mutate state") wearing a build-time-failure costume. The actual invariant this
codebase enforces is provenance-is-mandatory (`Ground`), not operate-is-mandatory.

**2. `cast`'s `Ground` is not a function of a cast item — it's a function of (item, all of
`history`).** Verified: `writeMinds` (`worldgenesiscommit.go:479-491`) resolves each actor's
grounding event by scanning **every** `doc.History[].Who` for the first entry naming them,
falling back to `historyEventIDs[0]`. `Landing.Ground(item) eventSpec` has no parameter through
which `cast`'s landing can see `history`'s parsed items. Either every landing gets implicit read
access to the whole document (cast's declaration now silently depends on history's shape — the
"one file" locality claim is false) or the runner special-cases this join (exactly the hidden
coupling Q3 asks about, and worse than mere *ordering*: today's `writeMinds` does this in one
visible function; under the contract it has nowhere honest to live).

**3. `world` doesn't fit any of the six members — it's the bootstrap, not a peer.** Verified:
`commitWorldContent` calls `createWorldTx` first (`:94`) to mint `worldID` itself — the scope
every other landing's `Mint` needs to exist before it can run — then a bare `UPDATE world SET
tagline=...` (`:124-130`): no `entity_registry` row, no `canon_event`, no perception. AC-3 lists
`world` among "the eight... no escape hatches." It needs one, or a fictional Mint/Ground/Distribute
that do nothing, which is P1's own forbidden shape turned on the contract itself.

**Q3, directly: yes, both forms.** New SPOF: `fn_extent_class_metres`/`fn_duration_class_seconds`/
tick assignment collapse into one runner path (P2) — sound for correctness, but a bug there now
breaks every concept atomically, with no per-field `refuse()` message (`worldgenesis.go`'s ~40
named checks) pointing at cause. New hidden coupling: finding #2, which the "dependency-ordered
execution" language (§2) only describes as *write* sequencing, not the *read* dependency the
current code actually has.

**Q4 — `norms[]` → `personality_core.traits`: unsound channel reuse.** `traits[].manner` is
authored and rendered (`cognitionprompt.go:136-151`, "THE MINDS YOU SPEAK FOR") as first-person
temperament — "how a disposition shows in behaviour" (`world_genesis.v1.schema.json:155-159`). A
norm's `stated` is a third-person world rule. Both would render in the same undifferentiated
key→{value,manner} list with no marker distinguishing "who I am" from "what binds me" — a rule
disguised as a personality quirk the model may weigh and discard like any trait. `norms[]`'s
proposed shape (`{canonical_name, stated, binds[]}`) carries no `strength` class, so the runner
must invent `traits.value` from nothing to satisfy the required field. And every entry in `traits`
is subject to the same malleability/quiet-arc drift machinery (`FINAL-world-npc-cognition.md:36-39`)
— a caste law smuggled through this channel inherits the *personal-disposition-softens-with-
experience* mechanic nobody decided laws should have.

**Q5 — what gets harder.** Today a broken concept fails loudly and locally: one function
(`worldgenesis.go:249-495`) with ~40 named `refuse()` messages, one bug traceable in one file.
Under the contract, cross-concept bugs like #2 become *harder to see*, not easier — the
architecture's selling point is that concepts don't know about each other, but the one place that
currently does this join in full view (`writeMinds`) has to either vanish into runner internals or
break the no-escape-hatch promise. Debugging trades "read one function" for "read the scheduler
plus N declarations to find which one silently produced a wrong `eventSpec`."

## What I'd do instead

Don't force all eight legacy concepts onto one interface to prove N+1 uniformity before it's
earned. Keep P2 (runner owns ids/ticks/class-resolution/provenance-stamping) — that part is real
and valuable regardless of the interface question. Make `Operate` **declared-optional** ("Operate:
none, because X") instead of registration-rejecting, so it catches `standing` (declared, wrote
nothing) without outlawing `history` (never should write anything). Let `cast`'s naming/trait/
secret provenance stay three explicit sub-facts instead of pretending one `Ground` call covers
them. Land `collectives[]`/`norms[]`/`near_future[]` on the tightened contract first — none of
them have the cross-reference problem — and defer forcing `world`/`cast`/`history` onto it until
migration is proven, not promised.
