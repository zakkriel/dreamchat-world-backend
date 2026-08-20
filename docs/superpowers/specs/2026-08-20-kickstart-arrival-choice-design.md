# Kickstart: the arrival is chosen, not assigned (2026-08-20)

**Status:** Approved design, pre-implementation
**Amends:** `10_prds/prd_world_creation.md` AC-6 (see "The amendment", below)
**Depends on:** the shipped genesis pipeline (`core/api/worldgenesishandler.go`, `worldgenesis.go`, `worldgenesiscommit.go`), the genesis document contract (`schema/world_genesis.v1.schema.json`), the interview turn contract (`schema/world_interview_turn.v1.schema.json`), and the commit ladder in the PRD's Spec Body §4.

## The problem

A build ends in a teleport. The genesis seat authors exactly one `arrival` — descriptor,
canonical name, place, stated line, why — and the commit ladder stamps it in the same
transaction as everything else. The user watched their world get built and then found
themselves standing in it as whoever the seat decided they were, wherever the seat decided
they stood. The one place the user's own preference could matter most — *who am I here, and
how does this start?* — is the one place they were never asked.

The Custom lane's interview may already ask "who are you?" against the *brief*. But the
richest moment to ask is **after the world's content exists**: candidates grounded in the
authored cast and history ("the debt collector Vesna has been dreading") are fiction; the
same question asked before the cast exists can only produce archetypes, and archetypes are
the template poison GA-2/GA-3 forbids.

## The shape (decisions, all made 2026-08-20)

1. **Two sequential questions, interview grammar.** Not a 3×3 grid. After the world is
   authored: "Who are you here?" — 3 grounded character candidates + "write your own" + a
   recommended default. Then "How does it start?" — 3 scenarios *for the chosen character* +
   "write your own" + a recommended default. Same grammar the interview already has
   (options + free text + marked default, PRD Spec Body §5), so the frontend learns nothing
   new and the user is never trapped.
2. **Brief wins, skip to scenarios.** If the brief or interview answers already state the
   player's identity, the character question is not asked — the user's own words outrank the
   seat completely, exactly as `prompts/world_genesis.txt:2` already rules. Only the
   scenario question is offered.
3. **Both lanes, always skippable.** Fast lane and Custom lane both pass through the
   kickstart. Every question arrives with its recommended option pre-selected; one act
   ("Start here") accepts the defaults and commits. Choice is offered, never imposed.

## The structural change: author, then choose, then commit

Today `POST /worlds/genesis` is one SSE stream spanning author → commit, one transaction
(`worldgenesishandler.go:17-21`). A user choice now sits between authoring and commit, so
the build splits into phases. The transaction discipline does not change — it moves.

- **Phase 1 — author.** `POST /worlds/genesis` streams narration frames exactly as today
  while the doc is authored, but ends with a **choice frame** instead of a committed world:
  the character candidates (or the scenario question directly, when identity was stated), a
  recommended default, and an opaque draft handle. No transaction has begun; authoring is
  in-memory LLM work. When identity was stated, the `world_kickstart` seat is called once
  at the end of this phase (against the stated identity) so the stream can end in the
  scenario question without an extra client round-trip.
- **Phase 2 — kickstart turns.** Plain JSON, one question per response, the interview's
  request shape: client sends `{handle, choice-or-custom-text}`, gets the next question or
  "nothing left to choose". The scenario options are authored here by the new
  `world_kickstart` seat, grounded in the full doc plus the chosen character.
- **Phase 3 — commit.** The final answer triggers the whole existing ladder — `world_genesis`
  event, backstory, scene genesis, the **chosen** arrival — in one transaction, SQL only, no
  model call. Playable or nothing, unchanged (AC-2).

**The draft lives in server memory with a TTL (~15 minutes), keyed by the handle.** Expiry
or abandonment loses the build and its spend — precisely the posture the PRD already took
for watched builds ("close the tab and the build is lost", Non-goals). No jobs table, no
resumption.

**The doc never crosses the wire.** This kills the stateless alternative (client carries the
doc back, interview-style): the genesis document contains every `hiding`, every backstory
event, every knowledge path. A curious user with devtools must not read the world's answers
before playing it. The client sees candidate *premises* only.

### Rejected alternatives

- **Client carries the doc** — leaks all secrets; tamperable. Dead on arrival.
- **Commit world first, arrival later** — a `playable:false` world listed until the user
  chooses violates AC-2 and leaves debris on abandonment.
- **3×3 authored upfront** — nine authored arrivals, most discarded; a wall of choice; ~3×
  the authoring cost for no grounding gain over sequential questions.

## What each piece is

- **Character candidate** — a premise, never a mind: `{descriptor, canonical_name, why}`,
  drawn from *this world's* authored cast and history. No traits, no core, no disposition;
  B-4 holds exactly as it does for today's single `arrival`
  (`world_genesis.v1.schema.json:271-303`).
- **Scenario** — the arrival made specific: `{place, why, stated}` plus which
  already-in-motion moment greets the player. The chosen scenario becomes the arrival rung
  of the ladder verbatim; the "at least one cast member starts in the arrival place" check
  (`prompts/world_genesis.txt:9`) applies to every scenario candidate, not just the default.
- **Genesis schema change** — `world_genesis/1` gains `arrival_candidates` (exactly 3
  items, premise-shaped) beside the existing `arrival`, which becomes the recommended
  default. Emitted only when the brief left identity open; the schema leaves candidates
  optional and the prompt carries the brief-wins rule; the Go validator's checks are
  mechanical — exactly three, distinct names, non-blank fields, exactly one candidate
  matching the arrival.
- **New seat `world_kickstart`** — authors 3 scenario options for one chosen character
  against the full doc. Schema-leashed like every seat: `additionalProperties:false`, no
  number of any kind anywhere (PRD AC-7). Prompt carries the same GA-2/GA-3 discipline:
  scenarios come from this world's content, no archetype vocabulary, no genre branch.
- **Frame contract** — the build stream grows a frame kind, so `world_genesis_frame/1`
  moves to `world_genesis_frame/2`; the frontend pins the version and fails the load on
  mismatch — the version moving IS the notification (`worldgenesishandler.go:38-41`). The
  kickstart turn reuses the interview turn shape under its own version for the same reason.
- **Fakes and CI** — `DREAMCHAT_BRIDGE=fake` binds a deterministic `world_kickstart` fake;
  its output is captured as a CI-validated payload exactly as `place_author_1.json` is
  (PRD AC-13). The full journey — brief, build, choose, commit, arrival — runs green with
  no API key.
- **Cost** — the genesis cost sink and ceiling (`DREAMCHAT_GENESIS_COST_WARN_USD`) cover
  the kickstart call; still one `world genesis timing:` line per build, now including the
  kickstart call's tokens.

## The amendment

PRD AC-6 states: *"There is no roster to choose from, no 'view as', no perspective
switcher."* This design offers a roster — so the sentence is amended, not worked around,
by the same reasoning that amended frontend law 12 for the login gate: **the reason the
rule exists does not reach this case.**

What the rule protects is the D-7 perception boundary: choosing *whose eyes you look
through* during play, which would hand the player perceptions they never earned. A
pre-play choice among authored *premises* touches no perception: whichever candidate is
chosen, the player still arrives holding exactly one `direct` perception (their own stated
arrival line, AC-4), still has no traits and no core (B-4), still earns every name from
zero. The frontend law's own regex (`laws.test.ts:216`: `view as|switch
(user|character|viewer)`) does not match a creation-flow choice screen and needs no change.

What stays absolute: no mid-play switching, no "view as", no roster *during* play, no
traits or inner state for the player under any candidate.

## What this must never learn (inherited, sharpened)

The kickstart prompt and schema contain no archetype taxonomy — no "the insider / the
outsider / the professional" spine, no genre-conditional anything. Candidates exist only
because this world's cast and history imply them. A recurring candidate shape across
unrelated briefs is a defect, not a convenience (GA-2/GA-3, PRD §8).

## Acceptance criteria

1. A Fast-lane build with an identity-silent brief ends its stream in a choice frame
   carrying exactly 3 character candidates, each `{descriptor, canonical_name, why}` with
   no numeric field, plus a marked recommendation.
2. A brief that states the player's identity produces a stream ending in the scenario
   question directly; the character question never renders and no `arrival_candidates`
   are emitted.
3. Accepting both recommended defaults ("Start here") commits a world identical in shape
   to today's: one transaction, one `ActorMoved`, one `direct` perception, `playable:true`.
4. A custom free-text answer at either question flows into the arrival exactly as an
   interview answer flows into the brief — the user's words outrank the candidates.
5. Every scenario candidate names an arrival place where at least one cast member starts.
6. An expired or abandoned draft handle answers with a stated refusal and leaves no
   directory row, no world, no debris.
7. The genesis doc (secrets, history, knowledge) is never present in any response body at
   any phase; candidates and scenario options are the only authored content that reaches
   the client before arrival.
8. The full journey runs green under `DREAMCHAT_BRIDGE=fake` with captured payloads for
   `world_kickstart` and the new frame version validated in CI.
9. Invariants I-1…I-10 hold on a kickstarted world; the player's epistemic state at
   arrival is exactly one perception regardless of which candidate or custom text was
   chosen.
10. One `world genesis timing:` line per build includes the kickstart seat's calls,
    tokens and cost; the configured ceiling logs a COST WARNING when a build spends past
    it. (The refusing ceiling of PRD AC-11 is pre-existing debt, deliberately out of this
    feature's scope — decision 2026-08-20.)
