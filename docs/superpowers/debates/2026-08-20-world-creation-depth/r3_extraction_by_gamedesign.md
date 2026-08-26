# R3 — extraction seat (final synthesis)

**Checks (§1.2).** Every identifier below names an engine column, a capability, or a closed family —
none traceable to a brief. The floor is a comparison against the engine's read set, not an exemption
list. Class→number stays grammar. Engine gaps named as programs.

## 0. What replaces referential pull

Referential pull is dead as elicitation (edges point downward; a minimal world is a global optimum), and
nothing schema-side can *solicit* without becoming a spine. Coverage therefore splits three ways, and
only the first two are the schema's:

1. **Completeness per thing, not coverage per topic** — whatever you author must be fully measurable.
2. **A mechanism-liveness floor** — the few things without which a named mechanism is dead code.
3. **Brief-relative solicitation, which is the interview's job** — it already holds the only legitimate
   rule (`prompts/world_interview.txt:3`) and is brief-relative by construction, so it cannot ossify
   into an ontology.

And one new artifact I own: **a bidirectional key index as a standing CI check.** Author→reader catches
class A (authored leaf, no commit path). Reader→author catches class B (engine input, no author).
Reader-with-no-consumer catches class C. Three defect classes, one index, one check — and the reason
class B could grow silently is that nobody ever computed the second direction.

## 1. Verdicts

**(a) `collectives` — illegal as a cluster; cut it.** Nothing computes with a group: the group branch of
`fn_visible_perceptions` has no viewer filter (leaks to the player), `fn_public_moment`/
`fn_private_records` filter `holder_id = ANY(p_present)` so a group is never present (reaches no mind),
and nothing walls a group's name (`schema.sql:1445-1453`, `:2937`). Its one real reader would be the
kind-parameterised compendium index (`schema.sql:1296-1304`), which requires a route to be registered —
so collectives are legal **only** if that route ships in the same change; otherwise they are class A by
construction. And they are unnecessary: a norm's `binds` can name cast members directly, exactly as
`history[].who` does, and "who belongs to what" is a *perception about a person* — already live. Cut the
entity, keep the fact. Re-entry: a registered compendium route, or a mechanism that computes with
membership.

**(b) The floor holds, but its cited precedent is wrong and proves the opposite.**
`trg_validate_tension` is `IF NEW.attrs ? 'tension' AND NOT (…)`
(`20260723100002_six_type_spine.sql:44-46`) — it validates the value **only when the key is present**.
A missing tension passes silently and reads as `none` ⇒ an infinite beat budget, which the benchmark
world documents as the bug it had to fix (`20260813142100_world_templates.sql:480-483`). So the engine's
tension guard is exactly the silent-absence hole the floor must close, not its precedent. The correct
precedent is presence-mandating at the gate: `EntityCreated` with no descriptor is `gate_reject`
(`schema.sql:212-214`).

So: **every entity a landing mints must carry every engine-read key its kind has.** What it refuses: an
object with no bulk class; a body with no carry class; a place with no tension; a container with no
capacity. What it does *not* do — and this is why it is not a spine — is say how many of anything exists,
or that any kind exists at all. It is a join against the engine's read set. It also gives the eval a
real contract: the fake driver must emit a fully-keyed world or CI fails.

**(c) One generic class table, not one per quantity.** `{world_id, family, class} → numeric`, with
`family` closed **in code** (grammar, never minted — handover's refusal of minted grammar holds) and one
thin typed accessor per family so callers stay unit-safe. The founder's growth constraint decides it:
quantity N+1 must be one declaration, not one migration. Per-quantity tables are why bulk, capacity,
carry and pace look like four migrations today. Grandfather the four existing tables
(`extent_class_metres`, `duration_class_seconds`, `journey_legs_band`, `world_actor_config`) — churning
live physics for symmetry buys nothing; fold each in when it next needs a change. Stated cost: two
mechanisms coexist until then.

**(d) The commit-time norm router as proposed is a keyword table — reject.** Classifying free prose into
speech/passage/reputation needs either a second seat call against a one-call ceiling
(`worldgenesis.go:172-174`) or a lexical rule set, which is the exemption list §1.2 forbids. The
legitimate version inverts it: **the seat points instead of describing.** A norm names the authored
capability it constrains — a speech-act type, a movement type — so routing is a type dispatch on which
reference is populated, which is grammar. A norm that points at nothing is reputational and lands as
ordinary per-holder knowledge. This is referential pull used for what it is actually good for: pruning
and dispatch, never solicitation.

## 2. THE ANSWER — the topic list

Live today, keep unchanged: **places + ways** (beat 1 — the only nameable things are this room, its
portals, the far side), **objects' identity and location**, **per-holder history and one private fact
each** (beats 3-5 — two accounts of one night contradict, the deepest thing the engine already does),
**traits/speech manner**, **tension per place**.

Starved — this is "what a full world must contain":

| Topic | Genesis authors | Destination | Engine work | Beat felt |
|---|---|---|---|---|
| Bulk & capacity | one bulk class per thing; capacity class on holders | `artifact_state.attrs` `size`/`empty_weight`/`max_room` | generic class table + resolution | 2 — the thing you cannot simply take; the container that fills |
| Carry | one carry class per body | `attrs.max_load` (today one literal 80, `worldgenesiscommit.go:62,666`) | same table | 2-3 — two people, different capability; the entry to encumbrance→speed→Journey |
| Present intention & **opposition** | what each present mind is doing, and one stated incompatibility between two of them | cognition prompt (absent section) | one section + one query | 1 — you interrupt something; 2-5 — minds act unbidden |
| Norms | one sentence + who it binds + the capability it points at | `speech_constraint`, else knowledge | decomposer classification (scoped, §4.1) | 2+ — an NPC refuses what is physically possible |
| Scheduled change | one imminent event, a magnitude class, attributed to an entity that can act | `pending_event` (`ledger.go:122-220`) | writer only | 3-5 — the world moves without you |
| Conditions | named conditions per body + what each hinders | `attrs.statuses`, `status_modifier` | register `statuses` in `tier1.go:4-22`; needs motion | 3 — the same stair is impossible for one person |
| Motion | named ways of moving, each a pace class | `movement_type` + actor binding | blocking program: `'walk'` hardcoded (`20260729100006_move_target.sql:63,66`) | 1 — distance stops being one number |
| Passage | what a barrier stops | portal `impedes` | founder ruling (§5.1) | 2 — a way that is shut only to you |

Cut: `collectives` (a), `regard`, `role`, `near_future.sets`, and `wants` as free per-mind prose —
opposition replaces it as one checkable relation. Restlessness-as-a-class is legal but low value: the
pressure roll is unsituated (`schema.sql:2661-2665`), so it tunes noise, not life.

## 3. Build order — cheapest real gain first

1. **Generic class table + bulk/capacity/carry keys.** No new reader needed; it revives four fact-sheet
   fields and the engine's only worked state-depends-on-state chain. Highest gain per unit work.
2. **The cognition section: present intention + opposition.** One section, one query; it is also the
   destination the five cut fields were competing for.
3. **`pending_event` writer** with a magnitude class and an actorful attribution (`{actor_id, attempt}`,
   `ledger.go:16-19`).
4. **Motion program**: un-hardcode `'walk'`, actor↔type binding, `statuses` registered.
5. **Norm dispatch** onto `speech_constraint` (needs 4's vocabulary for the passage cases).
6. **Passage `impedes`** — after the ruling. 7. **Recurrence** — its own program.
Standing: the bidirectional key index in CI from step 1 onward.

## 4. What I did not believe in round 1

- I marked **objects as leverage "live"**; it is not — with no `size`/`max_room` the fact sheet returns
  null for weight, volume, encumbrance and contents. A play-facing claim must be traced to the *field
  write*, not the scene read. Hence the larger shift: coverage is a property of the engine's read set,
  not of the fiction. Ranking topics by "the beat the player feels it" made class B invisible to me,
  because a starved input has no authored leaf to trace from.
- I proposed **`wants` as a per-mind string**. Wearing the schema hat: two free strings are
  uncomparable, so a validator cannot tell an opposed pair from two coexisting moods. Opposition must be
  authored as a relation between two named minds — the one place I now accept a structural requirement.
- I cited **trait drift**; `trait_pool` is unreferenced and `malleability` is prompt text
  (`cognitionprompt.go:146,216`). My conclusion survives on a different ground: a trait key renders as
  *character*, so a law stored there makes defiance read as out-of-character.
- I assumed the **naming wall covered groups**. It cannot; that is half of verdict (a).
