# Round 2 — seat: gamedesign (held by the extraction agent)

## 1. Sharpest attack on gamedesign's round-1 paper

Their own Tier A #5: *"Objects as leverage, not scenery. `carried_by` plus the fact sheet's
`reachable`/`weight`/`contents`. Table moment: beat 2 — take it, ask for it, or notice who won't
put it down. **Live.**"* (`world_topics_gamedesign.md:24-25`).

It is not live. Verified directly: the object-write loop in `commitWorldContent`
(`worldgenesiscommit.go:672-695`) sets `attrs.descriptor`, `attrs.kind`, and location-or-
`contained_by` — nothing else. No `size`, `weight`, `empty_weight`, `max_room`, `weight_modifier`
on any object, ever. Every actor gets the identical hardcoded `attrs.max_load = 80`
(`genesisPlayerMaxLoad`, `:62,666`). Per `fn_fact_sheet`'s gating (cited by simarch,
`schema.sql:1763-1778`), `weight_kg`/`contents`/`would_encumber` return **null** for every object
in every generated world. The exact beat-2 moment gamedesign names — "notice who won't put it
down" — depends on weight the world never authored. This is a role-blindness error, not a citation
error: gamedesign's charter is "does it reach the game," and this is the one item on their own list
that fails their own test, caught only because simarch's angle (starved mechanical consequence)
looks at what a *field* writes, not what a *scene* reads. A player-facing-play seat should have
run its own claim through `fn_fact_sheet`'s null path before calling it live.

Second, smaller miss: item 9 gestures at norms needing "one non-trait per-mind channel" for
enforcement (`:47-49`) without naming one. extraction's paper had already found that channel
sitting unused and *accepted* — `speech_act_type`/`speech_constraint`
(`handover §4.1`) — for the one sub-case gamedesign's own table-moment names ("an NPC refuses
something physically possible"). Gamedesign correctly diagnosed the trait-drift trap but, staying
inside "what produces story," never asked whether the engine had already solved it; it had.

## 2. Starved engine inputs, ranked by play impact

| Input | What genesis must author | Engine work |
|---|---|---|
| Bulk/capacity | a size class + carry class per object/actor (Tier-1 today, `tier1.go:5-21`) | **none** — stop omitting them (see §1) |
| Presence ("already running") | what each present person is doing, one line, into a reader that doesn't exist yet | **yes** — cognition prompt present-block |
| Opposition (incompatible wants) | at least two present people's wants stated as pulling against each other, same reader as above | **yes** — same reader, plus a comparison, not two free strings |
| Scheduled change | one authored near-future + a magnitude class (`pending_event.magnitude CHECK`, gamedesign's own catch, `:100-103`) | **yes** — genesis becomes first writer |
| Rule-as-refusal | one sentence + who it binds, routed at commit to `speech_constraint` when it's about speech | **yes, cheap** — decomposer classification, already scoped (`handover §4.1`) |
| Motion vocabulary | named ways of moving + pace class | **yes** — un-hardcode `'walk'`, pace-class table |
| Impediment | named conditions + what they hinder | **yes** — add `statuses` to the Tier-1 registry |
| Passage | which movers a barrier stops | **yes + a founder ruling** — `fn_portal_permits` signature (`handover §5.1`) |
| Recurrence | a repeat class on scheduled change | **its own program**, not this feature (simarch, `:91-94`) |

## 3. Adjudicating the two omissions — not one thing, different horizons

They are not the same gap. Opposition is felt **every beat a present mind decides** — cognition
runs before each player step by locked rule (`FINAL-world-npc-cognition.md:8-9`) — so once
authored it is renewable, present-tense, and lands inside the PRD's own 5-beat proof window
(`prd_world_creation.md:22`) by construction. Recurrence is a **single scheduled fire** whose
landing inside that same 5-beat window depends on the seat's `when_class` guess — it may not land
at all in the measured window, and its real cost is felt at session length far past beat 5, when a
world that fictionally repeats has already gone silent once. For *this* PRD's stated proof — depth
changes play, measured in five beats — opposition is strictly more load-bearing. Recurrence is
real, but it is a claim about long-session believability this debate's own acceptance criteria
never measure, which is why simarch is right to scope it as a separate program rather than urgent
to this one.

## 4. A round-1 fact-check catch

simarch's own row 5, "World temperament... Engine work: **none** — pure omission"
(`world_topics_simarch.md:31`), contradicts simarch's own standard two rows up: row 2 demands a
pace-class table for motion specifically because *"AC-7 forbids the seat emitting 1.4"*
(`:28`). `world_actor_setting.intensity` and `world_actor_config.climb_rate` are the same shape of
problem — raw `numeric` columns, no CHECK, no class-to-value table — so a genesis-authored
"temperament" is exactly as forbidden from emitting a raw number as pace is. Either both need a
class table (real engine work) or neither claim holds; simarch applied the rule to one cluster and
waived it for another in the same document.
