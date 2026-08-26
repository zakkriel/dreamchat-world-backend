# Findings — play-loop seat · landing-contract retarget round

**Date:** 2026-08-26 · **Seat:** play-loop (`harness/roles/play-loop-expert.md`, dossier
`docs/areas/play-loop.md`) · **Assignment:** questions 4 and 5 of `PROPOSAL.md`, plus the ruling on
increment 6's dilution measurement.

**Verdict:** the retarget's *mechanism* survives my seat. Its **first-customer set does not reach a
table**, and its parallel plan is not five wide. 8 `block`, 4 `gate`, 1 `accept-with-reason`.

---

## The opening move, and what I did not run

**There is no diff.** `git status --porcelain` in `dreamchat-world-backend` shows this round's
artifacts as untracked docs — `docs/superpowers/debates/`, `docs/30_architecture/world_model/`,
`docs/10_prds/prd_world_creation_depth.md`. No code changed, so there is no guard to revert and
`ci/mutate.sh` has nothing to bite. **No mutation table is reported because none could be produced**,
not because everything survived.

The area gate (`cd core/api && go test ./... -count=1`; `go test -run Latitude`; `make reset && make
test`) was **not run**. It writes to a live Postgres and does not roll back (`go-tests.yml:4-6`, cited
in dossier §1), and this round changes no code it would exercise. I am not claiming it green.

Instead I ran the class a sed script cannot show — **absent · null · wrong type · empty** — against
the four proposed first customers. F0 and F1 are that output.

---

## F0 · `gate` — the wrong-type question has no answer today

`PROPOSAL.md:24`: v6 has "no schema file, no Go type" — *no machine representation at all*. So every
claim in this proposal about what `integrity`, `latency_class`, `reliability_class` or `excluded[]`
do on malformed input is currently unverifiable. This is the SPEC-035 defect class exactly:
`witnesses: "<uuid>"` as a bare string committed with zero witnesses and no `halt_reason`, found one
commit too late by asking the wrong-type question (role brief, "the class the table cannot show you").

**Check the round adds:** a wrong-type fixture per first-customer leaf must be *refused* by the
document validator (`plan:52-54`), not silently dropped. A validator that only implements v6's 13
semantic refusal rules leaves the silent-drop class open.

**Empty, separately:** an empty `excluded[]` is a legal world that excludes nothing. It produces no
observable of any kind — see F1.

---

# Q4 — does it reach the table?

`prd_world_creation_depth.md:203-204` is the rule I am applying: *"The only victory conditions here
that are not structural. A mechanism passing §4–§5 and failing these is deleted."*

## F1 · `block` — none of the four has a player observable inside the baseline five-beat window

`PROPOSAL.md:71-72` states the selection criterion outright: the first customers become the v6
sections "already used by every test world and **absent from the engine**." Absent from the engine is
the same thing as absent from the player.

The window: `prd_world_creation.md:22` — five beats. World time across it is bounded above by five
times the scene's tension budget (`tension.go:28-45`, read once per beat at `beatBudgetSeconds:47-63`).
The seeded world is `tense` = 30 s (`tension.go:32-33`), so **≤150 seconds of world time across the
whole window**, and less in practice because the budget is a ceiling, not a spend.

**`integrity` — none, for the rule as written.** Row 8 of the audit states the rule as "a degradation
level, **and how much remains before terminus**". Degradation needs state that changes with elapsed
time, derived at read time — which is increment 3 (`plan:137-139`), wave 2. Nothing degrades
observably inside 150 s. The only in-window observable is the *static* condition word, and only if the
player issues a QUERY, which routes a perceived fact sheet to the narrator
(`narrateprompt.go:257-265`). A static condition word is indistinguishable from `attrs.description`
prose, so it cannot meet the §6 bar of "differing … within five beats — **diffed, not asserted**"
(`prd_world_creation_depth.md:206-208`).

**`latency_class` — none, and the proposal removes the mechanism that would have given it one.**
Observing a delay requires an event the player did not witness whose news arrives late. Genesis
authors no `pending_event`: it is "read every clock crossing (`ledger.go:122-220`) and today written by
nothing but three test inserts" (`prd_world_creation_depth.md:189`). The §5 concept that guaranteed an
in-window authored off-screen event is `near_future[]`, whose AC **is** this window — *"within the
baseline's 5-beat window the player perceives one authored event they did not cause, in every world"*
(`:191-192`) — and `PROPOSAL.md:71` retires it. Second, independent reason: the read path has **no
clock term at all**. `fn_visible_perceptions` (`schema.sql:3133-3148`) filters holder, `invalid_tick`
and `expired_at` and nothing else; a post-dated `acquired_tick` is visible immediately. Knowledge is
not merely instantaneous by construction (audit row 15) — the wall is timeless.

**`reliability_class` — none, and none is available in principle inside the window.** The rule forbids
exposing the value behind the sign (audit row 21). A sign that misreports is indistinguishable from
one that does not until the player holds a second, contradicting acquisition of the same fact. The
player arrives holding **exactly one perception** (`prd_world_creation.md:24`). Two independent
acquisitions of one fact inside five beats cannot be guaranteed in any world, let alone "in every
world" the way the retired AC-7 was.

**`excluded[]` — none.** Enforcement is on the seats, "for the life of the world" (audit row 24). Its
observable is a non-event: the absence of a thing the player was never told was excluded and cannot
be told, because they arrive knowing nothing. Increment 6's own stated proof — *"an attempt to
introduce an excluded thing is refused, and the refusal names the exclusion"* (`plan:218`) — is a
**harness** observable driven by a scripted attempt, not a player observable. Add F0's empty case: a
world that excludes nothing produces literally zero.

**Disposition `block`.** §3.2 routes four mechanisms through §4–§5 that §6 then deletes. Under the
PRD's own rule this must be settled here, not discovered after the landing framework has been built
around them.

## F2 · `block` — retiring §5 orphans §6, the only non-structural section

`PROPOSAL.md:71` retires §5 as written. Every AC in §6 is worded in §5's concepts:

- AC-9 — "built with and without the **`norms` landing**" (`prd_world_creation_depth.md:206-208`)
- AC-10 — "an NPC act that contravenes a **bound norm**" (`:209-211`)
- AC-11 — "N≥20 planted briefs whose **implied norms** share no shape" (`:213-216`)
- AC-12 — "Entailed **collectives, norms and near-future** render as strikeable statements" (`:218-222`)

The proposal amends §5 and leaves §6 standing, so the amendment's first customers inherit **zero**
victory conditions in the section the PRD calls "the only victory conditions here that are not
structural" (`:203`). A proposal may not delete the customers a section's ACs name and leave the
section unamended. §6 moves in this round or the retarget does not.

## F3 · `block` — §3.2 manufactures the defect class §3.1 adds an index to catch

`PROPOSAL.md:57` names three defect classes, the third being **"reader-with-no-consumer"**.
`PROPOSAL.md:71-72` then selects the first customers *because* they are absent from the engine. R1
requires every landing to declare a non-empty `Reader` (`prd_world_creation_depth.md:117`), and the
plan's fence requires that "every authored leaf reaches a reader" (`plan:45`).

The consumers for these four are scheduled **after** increment 1: `integrity` in increment 3
(`plan:137-153`), `latency_class` and `reliability_class` in increment 5 (`plan:180-201`), `excluded[]`
in increment 6 (`plan:202-220`). Landing them as increment 1's first customers writes four
reader-with-no-consumer rows into the same round that ships the index built to find them. Either the
first customers change, or the round states which of the four declares a reader that a live code path
actually reads on the day increment 1 merges — with the file:line.

## F4 · `gate` — "the largest single playability lever" is not established, and the brief is the wrong payload

**What the narration path reads today, every beat** — `narrateSceneBody`, `narrateprompt.go:157-267`:

| Block | Source | Line |
|---|---|---|
| the bounding header | `prompts/narrate.txt`, embedded | `:88-89` |
| `PLACE:` name + Tier-2 description | the location candidate matched by id against `payload.Here` | `:169-192` |
| `PRESENT:` `label [id]` per actor, viewer dropped | payload candidates | `:170-197` |
| `YOU ARE:` viewer aliases | `payload.ViewerAliases` | `:203-206` |
| `NOTHING RESOLVED` | empty delta + halt reason | `:227-229` |
| `WHAT JUST HAPPENED` / `RECENT BACKGROUND` | perception lines, split on `preIDs` | `:211-249` |
| `QUESTIONS THE PLAYER ASKED` + perceived fact sheets | `queryAnswers`, geometry banded at `:270-315` | `:257-266` |

Three of those — the place description, the actor labels, the perception lines — **are world-authored
text produced at genesis**. The narrator is already handed this world's own prose every beat. Confirmed
separately: `world.brief` is read only by `kickstartstate.go:96-99` and the genesis/interview handlers,
and its column comment says so — *"Operational provenance, never rendered: no projection selects it"*
(`schema.sql`, `COMMENT ON COLUMN public.world.brief`). So the audit's "world identity during play:
there is none" (audit:91-107) is right about the brief and overstated about the world: what is missing
is the **global, per-world** statement, not world content per beat.

Against that, inside the same increment, sits the brief-to-document coverage check (`plan:102`), which
addresses the audit's own worked failure: `mundo-08-sueno-comun-1-basico.md` states a numbered
threshold, a daily cycle with opening hours and a record that is true about a dream and false as an
accusation; `G_sueno_by_extraction.md` **encoded none of it and still passed its own validity and
sufficiency checks** (audit:154-166). Every per-beat thing the narrator reads is *document*-derived.
A document missing two-thirds of the brief starves all five beats, every beat, in every world. Rank
the coverage check above the brief-to-narrator block, or defend the ordering with something other than
cheapness. Cheap and largest are different claims and the proposal argues only the first.

**The gate, and it is mine:** handing the narrator `world.brief` while the document is short of it
hands it material the state cannot back. That is precisely what `narrate.txt`'s second wall forbids —
`NEVER CONTRADICT OR EXTEND THE STATE` (`narrateprompt.go:28`) — and precisely the founder-gate bug
that wall exists for: the driver dropped the payload, the narrator was left holding the one-line
instruction only, and *"with nothing to render, it invented an entire scene and broke frame"*
(`narrateprompt.go:14-17`). **Check the round adds:** what reaches the narrator is the committed
document's own content and minted vocabulary, never the raw brief, and a test asserts the brief string
does not appear in the assembled narrate prompt.

## F5 · `gate` — placement trap: the plain fallback would narrate with no world

`narrateBaseRules()` slices the header at `narrateSegmentContractMarker` = `"OUTPUT — STRUCTURED
NARRATION SEGMENTS"` (`narrateprompt.go:71`, `:93-98`) and the plain-prose fallback
`buildNarratePlainPrompt` (`:143-151`) ships only what precedes it. A world block appended to the end
of `narrate.txt` is therefore **dropped on the fallback path** — the path taken only after two
structured attempts have already failed, which is exactly when invention risk is highest.

`narrateSceneBody` is shared verbatim by the structured, repair and plain prompts *"so the scene the
model renders never changes between attempts"* (`narrateprompt.go:155-156`). **Check the round adds:**
a test asserting the world block is present in all three of `buildNarratePrompt`,
`buildNarrateRepairPrompt` and `buildNarratePlainPrompt`. Also note any `narrate.txt` edit fires
`go test -run Latitude` (`ADR-P022`, dossier §"The gate for this area"); the latitude block stays
byte-identical.

---

# Q5 — is the parallel plan sound?

## F6 · `block` — the dependency the graph missed is R2, and it is a cycle

R2 computes `⋃ Consumes` across **all** declarations and diffs it against **every** non-numeric leaf
of the target schema; an unclaimed leaf is a registration failure naming the leaf
(`prd_world_creation_depth.md:119-123`). It is a whole-schema, all-or-nothing check — that is its
entire value, and it is what killed `standing`.

Retarget the diff at `world_model/6` (`PROPOSAL.md:10`) and it now runs against 17 sections and 11
frozen facets of which the engine has **3 WORKING, 12 PARTIAL, 9 ABSENT** (audit:20). Two consequences
the graph does not model:

1. **Increment 1's registration cannot go green until every v6 leaf has a landing**, and every landing
   needs a non-empty reader (R1, `:117`) — i.e. until increments 2–8 have delivered the readers. The
   platform increment blocks on the features that declare into it. That is a cycle, and it is the
   opposite of `plan:268`'s "everything declares into the framework."
2. **The five wave-1 increments are not "mutually independent"** (`plan:269`). Each changes
   `⋃ Consumes`; any leaf one of them introduces is a named registration failure for **all** of them
   until claimed. Their build outcomes are coupled through a global set difference, regardless of
   which files they touch.

Either the target schema is staged — in which case the round states the staging key, and the retarget
is nominal until the last stage — or increment 1 is not a wave-0 increment at all.

## F7 · `block` — increment 6 depends on increment 4; the graph has no such edge

Increment 6's proof: *"an attempt to introduce an excluded thing is refused, **and the refusal names
the exclusion**"* (`plan:218`). Naming the condition that refused is increment 4's second defect and
its ship: *"the refusal names the condition that stopped it"* (`plan:156`), against today's generic
`premise_broken`/`journey_barred` naming nothing (`plan:161`; audit row 4, with
`station_f_exit_test.go:285-287`, `placeauthor_test.go:355`).

`plan:261-264` has `I1 --> I2 & I4 & I6 & I7 & I8` and no `I4 --> I6`; `plan:269` runs 4 and 6
concurrently in wave 1. Increment 6 cannot meet its own stated proof before 4 lands.

**Second collision the file-overlap note misses:** 4 and 6 both land in the refusal path —
`fn_portal_permits` (`schema.sql:2686-2697`), `fn_actor_move_permitted` (`:792-816`), the
`apply_event`/`apply_ruled_event` enforcement, mirrored in Go at `orchestrator.go:1156-1160,1241-1256`
(audit row 4). The note names only `fn_move_duration_actor`, `fn_portal_permits`, `orchestrator.go`
for 2 and 4 (`plan:275-277`).

## F8 · `block` — increment 7 depends on increment 2, and is missing from the overlap note

A moving place needs a speed of its own. The only speed source in the engine is
`movement_type.base_speed_mps` (`schema.sql:4007`, seeded `walk = 1.4` at `:3704-3706`), and
`fn_move_duration_actor` (`:2529-2543`) **hardcodes `'walk'`** with no actor↔movement-type binding
(audit row 2). Removing that hardcode is increment 2's ship (`plan:123-124`). Building 7 concurrently
means either waiting for that binding or writing a second hardcode — and `plan:39-40` names
`fn_move_duration_actor`'s existing hardcode as the live instance of the fence violation.

**Collision surface for 7, none of it in the note:** `fn_distance` (`schema.sql:1557`, recursive climb
to nearest common parent — which a moving container invalidates), `fn_move_duration_actor`
(`:2529-2543`), `orchestrator.go:308-312`, `journey.go:468-513` and `journeyScene:534-568` (audit rows
1 and 3). The note (`plan:275-277`) claims movement/portal is a 2-and-4 collision only. It is 2, 4
**and 7**.

## F9 · `block` — wave 2's two increments are not concurrent until the delay mechanism is named

Increment 5 ships *"delay before a fact is knowable"* (`plan:193`). `fn_visible_perceptions`
(`schema.sql:3133-3148`) has no clock term, so there are exactly two ways to build it:

- a read-time comparison against `fn_world_now` (`schema.sql:3212`) — which **is** increment 3's
  "derived at read time rather than by an event per tick" mechanism (`plan:139`), making `3 --> 5` an
  edge and serialising wave 2 entirely; or
- a discrete scheduled write through `pending_event` / `fireDuePending` (`ledger.go:122-220`), which
  needs nothing from 3.

`plan:270` asserts both run concurrently without deciding which. Same shape for 5's decay half:
`invalid_tick`/`expired_at` are read on every knowledge path and written by nothing (audit row 17),
and expiry is time-derived. The plan calls wave 2 "the only real ordering constraints among the
features" — that claim is unproven until this is decided.

## F10 · `accept-with-reason` — increment 6's comparison corpus is partial until increment 2

Increment 6 measures drift against *"the world's own minted vocabulary"* (`plan:209`), and the largest
body of minted vocabulary — the class ladders — is increment 2's ship (`plan:123`, fence `plan:41`).
**Accepted, not blocked:** the document's own nouns, descriptors and place language exist at increment
1 and are a sufficient corpus for a first measurement.

**Reason for the PR body:** the scale vocabulary is outside the drift corpus until increment 2 lands,
so 6's first number is a partial baseline and must be labelled as one.

---

# Increment 6 — can it detect dilution?

## F11 · `block` — it measures a different quantity than the one it names; "speculative" understates it

Closed decision 4 defines the distinction as **cause**, not appearance: *"earned change is legitimate
and must not be blocked … The distinction is **cause** — earned change has a recorded event behind
it; dilution has none"* (`plan:69-71`). It is a closed decision; it is not mine to reopen and I am not
reopening it — I am checking the measurement against it.

Increment 6's measurement is *"drift measured over a long run as a **comparison** between the
narration and the world's own minted vocabulary"* (`plan:209`). A vocabulary comparison over prose
reads **words**. It cannot read cause. It therefore scores the exact case closed decision 4 exists to
protect — the world under a permanent drought whose hero destroys the god that cursed it — identically
to genuine dilution, because both present as the world's minted words thinning out of the narration.
**A measurement that cannot implement the definition it is measured against is not "measurable but
unproven"; it is measuring something else.** The `plan:211-212` flag ("the only speculative piece in
these eight — measurable, unproven") is the wrong flag.

**Second, and it is the play-loop half:** narration cannot dilute the world, by construction.
`NEVER CONTRADICT OR EXTEND THE STATE` (`narrateprompt.go:28`) is a wall with a belt behind it, and
the audit confirms the per-beat belts are epistemic, not stylistic (audit:102-105). What narration can
dilute is the **player's experience** of the world — real, worth measuring, and not what "the world
stays itself" says. Facts change only through recorded events, which is 6's *other* ship
(`plan:207-208`), and that is where cause is legible exactly, at zero speculation. The increment has
one half that is enforceable and cause-aware and one half pointed at the surface where cause does not
exist, and it is the second half carrying the increment's name.

**Stated in 6's favour so it is not re-derived:** the corpus exists. `transcript_entry`
(`schema.sql:4146`) stores *"the viewer's lived story as DELIVERED: rendered prose, post-belt, never
retro-labelled"* (`COMMENT ON TABLE`, `:4163`), viewer-scoped and ordered by `entry_no`, served by
`fn_transcript` (`:2929-2975`); `world.genesis_doc` holds the document. The comparison is mechanically
possible. **The corpus is not the problem. The definition is.**

## F12 · `gate` — there is no before-number, and after increment 1 there cannot be one

Increment 2's proof carries *"Before-numbers are on file"* (`plan:135`). Increment 6 has none, and its
treatment — the narrator receiving the world — ships in **increment 1** (`plan:101-102`, restated at
`:212-213`). Once 1 lands, no untreated run exists to measure against, and "measurable, unproven"
becomes permanently unresolvable: a number can be computed and nothing can be said about whether it
improved.

**Check the round adds:** capture the drift score over the three existing test worlds **before**
increment 1 changes the narrator's inputs, and file it the way increment 2's before-numbers are filed.

---

# Dispositions

| # | Disposition | One line |
|---|---|---|
| F0 | `gate` | v6 has no machine artifact, so the wrong-type question is unanswerable; validator must refuse a wrong-type fixture per first-customer leaf |
| F1 | `block` | none of `integrity`, `latency_class`, `reliability_class`, `excluded[]` has a player observable inside five beats |
| F2 | `block` | retiring §5 leaves §6's four ACs naming deleted concepts; the first customers have no victory conditions |
| F3 | `block` | first customers selected *because* absent from the engine = the reader-with-no-consumer class §3.1 adds an index to catch |
| F4 | `gate` | "largest playability lever" unproven — the narrator already gets world prose every beat; world context must be the committed document, never the raw brief |
| F5 | `gate` | a world block after the segment-contract marker is sliced off the plain fallback; assert it in all three prompt builders |
| F6 | `block` | R2 is whole-schema, so v6-as-target makes increment 1 block on increments 2–8 and couples the five "independent" wave-1 increments |
| F7 | `block` | 6 needs 4's named refusal; no edge, and they share the refusal path |
| F8 | `block` | 7 needs 2's movement-type binding; 7 is absent from the file-overlap note and collides across `fn_distance`/`journey.go` |
| F9 | `block` | wave 2 is concurrent only if 5's delay uses `pending_event`; if it uses read-time derivation it needs 3. Undecided |
| F10 | `accept-with-reason` | 6's drift corpus lacks the class ladders until 2 lands; document's own nouns suffice for a first, partial baseline |
| F11 | `block` | 6's drift measure reads words, not cause, so it cannot implement closed decision 4; narration cannot dilute the world by construction |
| F12 | `gate` | no before-number for drift, and increment 1 destroys the chance to take one |

---

# Friction

**A journal exists** — `docs/00_workspace/friction/2026-08-26-live-friction-journal.md`, nine entries,
timestamps genuinely spread across 02:04–02:09Z. That is the good case and worth saying out loud. Two
rulings and one omission.

**WASTE — `ci/check_round.sh`'s Areas refusal, as currently shaped.** Entry 02:09:12Z: *"check_round
required an Areas: declaration for a docs-only commit and I guessed the wrong one; the launcher had
the right answer but nothing told me to ask it before writing the body."* A gate that refuses a blank
declaration and then **accepts a wrong guess** produces false confidence, and a guessed `Areas:` line
is the invented-constraint failure the harness forbids. Cost paid, nothing caught. **Fix named, not a
deletion:** `ci/check_round.sh`'s refusal message must print `./harness/review.sh --pr-body`. The
routing exists in `AGENTS.md` pre-flight and `round-protocol.md` §7; it is absent at the one moment
the author is looking at the error.

**WASTE — the spine check's shared pattern.** Entry 02:07:15Z: *"the spine check silently passed after
I added a whole new brief section — its friction pattern matched the journal text too, so one pattern
now covers two different sections and a missing one would go unnoticed."* A check that cannot
distinguish two sections is `AGENTS.md`'s own failure mode: the map claiming coverage it does not have
(failure-log #26/#30). **Fix named:** one pattern per section, or the check reports which sections it
actually matched.

**Friction not recorded, and it is the cheapest finding here.** Nobody logged the cost of establishing
**what the narrator is handed per beat** — three documents in this round argue about it (`PROPOSAL.md`
§3.4, audit §"World identity during play", `plan:101-102`) and none states it. I had to read
`narrateprompt.go` end to end to answer F4. **Doc named:** `docs/areas/play-loop.md` §1, the
`narrateprompt.go`/`narration.go` row, which today says only "the resolve and narrate seats". It
should carry the per-beat input list — header · PLACE + description · PRESENT · YOU ARE ·
NOTHING RESOLVED · WHAT JUST HAPPENED / RECENT BACKGROUND · QUESTIONS + fact sheets. "Does the
narrator know X" is now a recurring question and the dossier cannot answer it.

---

# Close-out obligations if this round proceeds

Named because a change that ships without moving the docs it invalidated hands the next reviewer a
lie:

- `prd_world_creation_depth.md` **§6** — F2. It cannot stay worded in `norms`/`collectives`/
  `near_future` while §3.2 retires them.
- `docs/superpowers/plans/2026-08-26-world-model-eight-increments.md` **§Dependencies and parallel
  waves** — F7, F8, F9: the `4 --> 6` edge, increment 7 in the file-overlap note, and the wave-2
  decision.
- Same plan, **increment 6** — F11: the drift measurement's flag is wrong, not merely optimistic.
- `docs/areas/play-loop.md` **§1** — the narrator's per-beat inputs (friction), and **§2** as a trap
  if F5's fallback-slicing hazard is real when the world block lands.

**Learned:** the five-beat window is a harder gate than the structural ACs, and this round's four
first customers were chosen by a criterion — *absent from the engine* — that is negatively correlated
with passing it.
