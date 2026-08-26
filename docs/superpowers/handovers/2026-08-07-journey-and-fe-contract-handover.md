# HANDOVER — The Journey + a proper BE↔FE contract (2026-08-07)

> **Historical.** This document references the `dreamchat-frontend-play` worktree, which **no
> longer exists**, and `:5173`, which is retired with the archived `dreamchat-frontend` repo
> (`workspace:ADR-W003`). The live frontend is `dream-weaver-visuals` on **5273**; the current
> bring-up is `../stack.sh start` at the workspace root. The steps below are kept as the record of
> what was executed at the time — do not follow them verbatim.

**For the agent resuming this.** This picks up a brainstorming session (superpowers:brainstorming) that
paused mid-scope-framing because the driving agent's context got heavy. You are designing **the Journey**
backend subsystem AND the **BE↔FE contract** it needs, per the founder's PRDs — not the throwaway test FE.
Read this whole doc, then resume at "WHERE WE PAUSED" below. Follow the founder's working style (see the
end). Do NOT start coding — you are in brainstorming; the terminal step is writing-plans after the founder
approves a written design.

---

## 1. THE ASK (the founder's words, this session)

> "start the journey! now an important aspect.. there will be a FE, there is a FE.. please make sure that
> the BE <> FE connection is relevant as we need to connect it to a proper FE and not the ugly 'test' FE
> we are using. if you have questions the full PRDS and docs show how it should be and what features,
> modules and UX should have."

Earlier in the program the founder also set the sequence: **"the Journey, living world, and Over-budget
actions"** — and confirmed **over-budget == the Journey** (one feature, not two). The Living World (Station
G) was built FIRST (it's the seam the Journey rides); it is now COMPLETE, whole-branch-reviewed, and just
had a founder-requested code-health cleanup pass. The Journey is the agreed next subsystem.

## 2. WHERE WE PAUSED (resume here)

We are at the **scope-framing fork** of the brainstorming, awaiting the founder's answer. The driving agent
laid out that the request spans three layers and recommended a scope. **The founder has NOT yet answered the
scope question.** Do not proceed to the detailed design forks until they do.

**The three layers presented:**
1. **The Journey mechanism** (backend) — sustained-until-threshold actions unfolding over world-time,
   riding the world's-turn seam. *Definitely in.*
2. **The proper contract the Journey needs** — surfacing scene + journey state (where you are, the
   waypoint, progress, "interruptible," "arrived") and a real *continue/advance* step, shaped to the PRD's
   documented API. *This is where "make the connection relevant" lives.*
3. **The full FE product** — the actual Scene Canvas UI, all four Aux lenses, Workspace nav, streaming SSE,
   auth/sessions. A **large separate program**, far bigger than the Journey.

**Recommended scope (awaiting founder confirm):** this effort = **1 + 2 + a real, journey-capable play
surface** (a proper play page following the UX doctrine — scene, attributed dialogue, continue, a "you're
on your way" state — scoped to what the Journey needs and built to extend), with the **full FE product
(layer 3) as its own separate next program.**

The exact question put to the founder: *"Is that the right scope — the Journey, its proper contract, and a
real journey-scoped play surface, with the full FE product as a separate next program? Or do you want the FE
pushed further (or kept to just the contract) in this round?"*

## 3. REPO / BRANCH STATE

- **Main subject repo:** `/Users/pelao/REPOS/dreamchat/dreamchat-world-backend` (Go orchestrator `core/api`,
  plpgsql + dbmate migrations, pgTAP tests). Branch **`feat/living-world`**, HEAD **`e5e96ce`** (Living
  World + cleanup). The Journey should branch off this tip (it rides the world's-turn seam built here).
- **Durable ledger:** `.git/sdd/progress.md` (git-ignored scratch) — the full task-by-task record of the
  Living World build + cleanup + all deferred items. **Trust it + `git log` over recollection.**
- **Frontend repo:** `/Users/pelao/REPOS/dreamchat/dreamchat-frontend-play` (a worktree/branch
  `chunk-5.5-play-page` of the same repo as `dreamchat-frontend`). Its `src/pages/PlayPage.tsx` + `src/api.ts`
  are the **throwaway test FE** (self-labeled "dev scaffolding… dies with real sessions"). Canonical truth
  lives in the BACKEND repo docs (frontend AGENTS.md D-6).
- **Auto-memories** (loaded each session, `~/.claude/.../memory/MEMORY.md`): Brian's communication rules;
  provider-neutral stack (never default seats to Anthropic — owed work); the modular-architecture CORE
  mandate; the workspace map.

## 4. GROUNDING — WHAT THE JOURNEY IS (read these in full)

- **`docs/superpowers/specs/chunk-5.5-final/RULINGS-2026-07-30-space-and-journey.md` §2** — the founding
  ruling. Over-budget action is **NOT a REJECT — it is the JOURNEY**: a sequence of beats, each beat a
  world-action slot. Each slot the world acts first (telegraph / cut in / redirect, or nothing); nobody
  acts → the action **makes progress and carries to the next slot**; **context re-evaluates as the actor
  moves** (leave the tense tavern → next slot runs under lower tension, actor now "at a waypoint — the
  tavern door — looking onward"); across the slots the world had **multiple chances** to stop/redirect —
  if it never did, the actor **arrives**. Founder quote: *"it tried to get out and managed to, but there
  were multiple world-action slots in the middle that could have stopped or forced a change of plans — or
  not, and it just resolves."* This is its own body of work ("the Journey"), NOT a Station-F tweak.
- **`docs/superpowers/specs/2026-08-05-living-world-design.md`** (founder-approved, the most recent design;
  the Living World I just built) — **expands the Journey beyond travel** (Unit 1): *"Sustained-until-
  threshold acts are NOT a duration class — they are the Journey. 'Lay hidden until 2am', 'wait 100 years',
  'walk home' have no intrinsic length; they run until a condition is met… travel resolves on distance
  covered, a vigil resolves on the clock reaching a tick, a watch resolves on an event firing — one
  mechanism, spatial or temporal."* The decomposer recognizes the *"until/for <condition>"* form (a
  parse-shape, like QUERY) and binds the condition; **time conditions ride the clock, event conditions ride
  the pending-events ledger** — both built in the Living World. **Uniform test: span fits the beat's budget
  → resolves inline; exceeds it → becomes a Journey.**
- **The world's-turn seam (already built, ready to reuse per leg):** `core/api/worldturn.go` —
  `func (o *Orchestrator) runWorldTurn(ctx, worldID, scene string, tickBefore, tickAfter int64, seq int,
  outcome *BeatOutcome, trace *BeatTrace) (firedMag string, seqUsed int, err error)`. It fires due
  scheduled events (ledger), rolls the pressure tiers, and — if a tier fires — calls the World Actor, and
  reports the biggest magnitude fired (so the caller applies the §5 cut: small runs on, medium/large end the
  beat). It is a clean standalone unit (now has a direct-call test, `TestRunWorldTurn_Standalone_*`) built
  precisely so **"the Journey calls it once per leg — same unit, zero changes."** Supporting pieces:
  `fireDuePending` (ledger crossing), `rollTier`/`fn_pressure_chance` (pressure), `runWorldActor` (the seat),
  `lastEruptionTick`, the `world_eruption` fire-log. See `docs/superpowers/plans/2026-08-05-living-world.md`.
- **Current spatial mechanics (Station F, shipped, single-beat only):** `docs/runbooks/station-f-exit.md` —
  `fn_distance`, `fn_move_duration_actor` (distance ÷ speed), `fn_portal_permits` (locked/open doors),
  `fn_effective_weight`/encumbrance, `fn_target_position`. These work TODAY for in-scene single-beat moves
  (8 m to the bar = 6 s, fits one beat's tension budget). The Journey is what's needed when a move (or wait)
  **exceeds one beat's budget**. `docs/open-spec-items.md` SPEC-017/018 = deferred move-validity + the
  fuller spatial engine.
- **The reaction machinery the Journey reuses for interruptions:**
  `docs/superpowers/specs/chunk-5.5-final/RULINGS-2026-07-24-reaction-beat.md` — telegraph → held-outcome →
  reaction beat; §3 held-outcome is **loop state that lives in the world (a row), NOT session memory** —
  fires on the player's next input. §5 world eruptions: small never cuts, medium/large cut the beat. A
  Journey interruption should reuse this, not invent new machinery.

## 5. INVESTIGATION A — THE INTENDED FE (per the founder's PRDs)

Source of truth is the BACKEND repo docs. Key files (read for the FE design):
- `docs/20_design_ux/core_ux_loop_and_aux_sidebar.md` — the FE UX doctrine.
- `docs/30_architecture/mvp_slice_and_bridge.md` §2 + §4 — the FE⇄BE Bridge contract.
- `docs/20_design_ux/mockups/mock_gameplay_screen.png` — the concrete visual target.
- `docs/00_strategy/01_product_vision_and_promise.md`, `02_poc_scope_and_success_criteria.md`,
  `06_rules_register.md` (rule IDs are law — cite them).
- `docs/10_prds/compendium/*` — the four Compendium PRDs (Actors/Locations/Artifacts+Carrying/Timeline).

**Core principle:** *"feel like returning to an ongoing world, not opening a blank chatbot… I am back in the
world, not managing a dashboard."* Two layers: a Play-first default + an optional World Workspace.

**Main screen zones (`core_ux_loop_and_aux_sidebar.md` §2):**
1. **Main Scene Canvas** — the visual/emotional center ("Where are we?"): place, tone, atmosphere; compact
   visual + expandable prose.
2. **Scene Participants** — who is present/speaking/silent; strictly characters/NPCs/narrator (never objects
   or locations as avatars). *"The world decides who responds"* — targeting via NL or clicking an avatar
   must NOT guarantee obedience.
3. **Conversation / Narration Panel** — the text-first surface: user actions, dialogue, narrator, NPC
   dialogue, consequences, interruptions, cutaways. Loop: *user acts → world responds → world state updates
   → user continues.*
4. **Aux Context Sidebar** — four MVP lenses: **Current** (what matters now), **Inspect** (what am I looking
   at — on selection), **Intent** (how the system understood my action — the decomposed intent chain +
   confidence, correctable), **Known** (what do I know — user-perspective, preserves uncertainty: "Last
   known…", "unconfirmed"). Auto-switches by attention; MUST never leak hidden truth (knowledge boundary).
5. **World Workspace Navigation** (optional deeper layer): Timeline, Entities, Locations, Artifacts, Known
   World, Corrections, Settings, Creator Tools. ("Relationships" is EXCLUDED from MVP per the doc's own
   errata banner / rule B-3, even though the older mockup still shows it.)

**Interaction granularity (§3):** `Message → Beat → Scene Segment → Scene`. A **Beat** is the interactive
unit (~1–6 visible messages); canon commits are clustered, NOT 1:1 with messages. **Continue advances the
current moment by ONE beat — it does NOT fast-forward the world.** §3.4–3.5 anti-runaway rules the Journey
MUST honor: *"Time jumps require explicit or strongly implied in-world action"*; *"pause before major
consequences, transitions, or irreversible changes."* §4.9 Decay establishes the "Last known… / may need
review" language a Journey's arrival "Backstage Updates" would use.

**Movement/journey/time in the FE — the gap:** the PRDs say almost NOTHING yet about presenting a journey.
There is **no specified UI affordance for "you are mid-journey"** — no waypoint indicator, no progress, no
"traveling, interruptible" state. That is a **genuine open design surface** for this work. (No dedicated
"Journey" PRD/design file exists yet — the concept lives only in the rulings + the living-world design's
Unit 1 note. This handover's work fills that gap.)

**Intended Bridge contract (`mvp_slice_and_bridge.md` §4) — MOSTLY ASPIRATIONAL / NOT SHIPPED:**
- Read side: perception-bound, `schema_version`-stamped JSON the FE renders verbatim (never filters/sorts —
  rule D-7): `GET /worlds/{w}/compendium/{actors|locations|artifacts}[/{id}/page]`, `…/timeline`,
  `GET /worlds/{w}/carrying`, **`GET /worlds/{w}/scene/current` → scene canvas state + participants + aux
  Current payload** (SPECIFIED, NOT IMPLEMENTED).
- Write side (beat loop): **`POST /worlds/{w}/beats` (plural) returning a STREAMED SSE** — interpretation
  frame (Intent lens) → narration tokens → scene delta → aux delta → pushback frame → correction-window
  frame; plus **`POST /worlds/{w}/beats/continue`** ("advances the moment, never fast-forwards"). A beat
  yields zero/one/many canon changes — FE never assumes 1:1. (NONE of the streaming/continue is shipped.)
- Async channel: one SSE/WebSocket per world session (`image.ready`, `projection.updated`,
  `backstage.applied`, `correction.window_closed`). Module UI slots: `aux.panel`, `scene.overlay`,
  `action.bar` (FE renders unknown modules generically).
- Iron rule (D-7/D-1): FE never receives canon, never decides outcomes, never reconstructs hidden state.

## 6. INVESTIGATION B — THE CURRENT BE↔FE CONTRACT (what actually ships)

**One synchronous endpoint.** `POST /worlds/{w}/beat` (SINGULAR) — `core/api/beathandler.go:20` (route),
`:98-100` (POST only), registered `core/api/main.go:63`. Request: `{"text": "<free text>"}` (`:121-127`,
64 KB cap). Viewer identity is a **query param**, not body.

**Response** (`beathandler.go:264-286`):
```
{ "schema_version": "beat_result/3",
  "narration": <string>,               // legacy joined narrator blob
  "messages": [ {speaker_id, speaker_label, kind:"narration"|"speech"|"action", text} ],  // attributed segments
  "result": { "committed": []string, "halt_reason": string, "ticks_advanced": int64 /*DELTA*/,
              "unresolved_candidates": []string, "telegraphs": []string },
  "reasoning_log": <BeatTrace>          // ONLY when server DREAMCHAT_MODE=debug; ABSENT (not null) otherwise
}
```
`halt_reason` values today: `completed | telegraph | bounce | unresolved | premise_broken | turn_budget |
gate_reject | ruled_event_rejected`. `BeatOutcome` is `orchestrator.go:37-44`.

**What the response does NOT carry (the crux):** **no scene/world state at all** — no `location_id`, no
participant roster, no absolute tick (only a tick *delta*), no scene object, no map. There is **no
`GET /scene/current` endpoint** (specified in the arch doc, but the router `main.go:54-64` has exactly 8
handlers: 3 compendium page + 3 index + 1 timeline + 1 beat — no scene/carry). Server-side the engine DOES
build a `PerceptionPayload` per beat (`beatseats.go:10-20`: present actors + current location + perceived
artifacts + candidates + viewer aliases) but it is fed to the LLM seats and **discarded — never returned to
the FE**. The compendium page endpoints exist but require the FE to already know an entity id and are never
pushed after a beat (no beat→page correlation).

**Identity/session:** world = UUID path segment. Viewer resolved SERVER-side (`viewer.go:17-27`): production
default = the world's single actor named `'Player'`; a `?viewer=<uuid>` override honored ONLY in debug mode.
**No auth, no session, no multi-actor model** (`viewer.go:16`: "Auth/session out of scope this chunk").

**Movement/time surfaced today:** only `ticks_advanced` (a delta). No location/coords/absolute tick in the
beat response. `ActorMoved` + `duration_class` exist in the decompose vocabulary but don't cross into the
response. Note the intended rule `mvp_slice_and_bridge.md:48`: *"Time travels as `tick` (ordering) +
`display_label` (rendering)"* — the beat endpoint doesn't even do this yet.

**The test FE** (`dreamchat-frontend-play/src/`): `api.ts` `postBeat(world,text,viewer?)` → `BeatResult`
type mirrors the response (`api.ts:148-160`). `PlayPage.tsx` renders a flat scrolling transcript + textarea
+ a collapsed `<details>` debug dump of `reasoning_log`; `HALT_COPY` (`:49-57`) maps halt reasons to player
copy. World id + viewer id are **hardcoded compile-time UUIDs** (`router.tsx:14,20`, `PlayPage.tsx:9`),
explicitly "dev scaffolding… dies with real sessions." No scene, no map, no participants list, no session.
Play isn't even in the FE's documented surfaces (README lists only actors/locations/artifacts/timeline).

## 7. THE KEY INSIGHT (why FE and Journey are intertwined)

The test FE is thin **because the backend contract is thin.** The Journey is the first feature that CANNOT
work on the thin contract: a multi-turn trip is unplayable unless the FE can show *"you're on your way to
the house, at the tavern door, the world may still stop you — continue."* Today an over-budget move returns
`turn_budget` = *"the beat ran long, go again"* — the exact **"you can't even try to leave" dead-end
RULINGS-2026-07-30 rejects.** So the Journey **forces** the contract to grow up: it must introduce
(a) scene/waypoint state surfacing, (b) a journey-in-progress state, and (c) a real continue/advance step.
That is the concrete BE↔FE deliverable, and it should be shaped as a forward-compatible slice of the
documented Bridge contract (§4), not a test hack.

## 8. ANTICIPATED DESIGN FORKS (grill the founder on these, one at a time, each with a recommendation)

These are the driving agent's predicted forks — NOT yet decided. Refine as you go.

**Journey mechanism:**
1. **Journey state storage** — a persistent `journey` row (loop state that lives in the world, like
   `held_outcome` §3, but spanning MANY inputs): actor, goal/threshold, condition, accumulated progress,
   current waypoint/position, the tension context. Derived vs stored? Recommend a real table (a journey
   outlives a single beat; the next input/continue picks it up). NOT canon — loop state.
2. **The terminating condition (the threshold)** — travel = arrival at target position (distance covered);
   wait-until-time = clock reaches tick; wait-until-event = a pending-event/world-state predicate fires. The
   decomposer binds the "until/for <condition>" parse-shape. How is the condition represented + evaluated
   each leg? (Time → clock; event → the ledger.)
3. **Progress per leg + what a "leg" is** — a leg = one beat = one `runWorldTurn` slot. How much
   world-time/distance advances per leg? (A leg consumes ~one beat's tension budget of progress? Tension is
   re-read at each waypoint per §2.) Way-A accumulate-to-threshold.
4. **What the player DOES during a journey** — just **Continue** (advance one leg), or can they ACT each leg
   (change plans / interrupt themselves)? The PRD Continue advances one beat; §2 says each leg the world
   acts FIRST, then progress. Reconcile: is a journey leg a Continue press, and can the player instead type
   an action to alter/abort the journey?
5. **Interruption semantics** — when the world's-turn erupts (medium/large) or an NPC/pending event
   telegraphs mid-journey → does the journey **END** (you deal with it, then restate "continue home" — like
   §1 discard-and-restate) or **PAUSE** (auto-resume after you handle it)? Real fork; recommend END +
   trivially-restatable (matches §1, avoids suspended-state machinery), but confirm.
6. **Waypoints / path topology** — are waypoints real intermediate locations (spatial topology between
   start and goal, SPEC-018) or abstract "legs"? For v1, recommend abstract legs + the current
   location/position as the "waypoint," deferring a real path graph. Tension re-reads at each.
7. **Arrival** — threshold met → actor at goal; final narration; "Backstage Updates" (what changed while
   traveling). And what a non-spatial journey (a long vigil / over-budget monologue) looks like on arrival.

**BE↔FE contract (the "make it relevant" half):**
8. **The journey block in the beat response** — e.g. `{in_journey, goal, current_waypoint/scene, progress,
   interruptible, arrived}` + new `halt_reason`s (`journey_leg` = you advanced, continue; `journey_arrived`;
   `journey_interrupted`). Replaces the `turn_budget` dead-end.
9. **Scene state surfacing** — build `GET /worlds/{w}/scene/current` (specified but unbuilt) now, scoped, so
   the FE can render "where you are / the waypoint" and the participants? Recommend yes (the Journey needs
   it; do it as the real documented endpoint).
10. **Continue/advance step** — a `POST /worlds/{w}/beats/continue` (or a continue flag) that advances the
    journey one leg. Align with the documented `/beats/continue`.
11. **Streaming (SSE)** — recommend OUT of scope for the Journey (defer to the FE-product program); a
    journey works fine request/response (each leg = a continue call). Flag it.
12. **The real play surface** — scope: scene + attributed narration + participants + continue + a "you're on
    your way / interruptible / arrived" indicator, following the UX doctrine, extensible — NOT the full
    product (no 4 aux lenses / workspace / auth in this round). Confirm depth with the founder (this is the
    scope question in §2).

## 9. PROCESS / DISCIPLINE (how this program is run)

- **Brainstorming → written design (founder-approved) → writing-plans → subagent-driven-development**
  (fresh implementer subagent per task + a two-stage review gate each + a whole-branch review at the end).
  The review gates have caught real bugs every station — do not skip them.
- **Modular architecture is a CORE founder mandate** (memory `modular-architecture-mandate`): small focused
  single-purpose units, clean interfaces, NO cross-layer patches. The Journey must be its own clean unit(s)
  reusing the world's-turn seam — no jamming journey logic into `runChain`/`worldturn.go`.
- **Provider neutrality is OWED** (memory `provider-neutral-stack`): never default seats to Anthropic; the
  cheap stage-0 stack (Mistral/DeepSeek/euryale) is the intended direction; per-seat overrides owed for all
  six seats. Not the Journey's job but keep the contract provider-neutral.
- **Deferred Living World items, GATED before the real (non-fake) World Actor driver goes live at play**
  (in the ledger): (A) fire-log commit atomicity (world_eruption drain); (B) a runtime `location==scene`
  check in `runWorldActor`; (C) the empty/QUERY-only beat floor-window world's-turn gap. The Journey
  playthrough with a live driver should not happen until these are addressed. The station runs on the
  deterministic FAKE driver in CI (that is a test double, NOT a hardcoded world — the real driver is the LLM).
- **The founder personally play-tests each surface as the exit gate.** A Journey playthrough (walk home,
  the world interrupting) is the eventual gate. He deferred the Living World playthrough ("no need to test
  now") to build forward — confirm whether he wants to play before or after the Journey.
- **Founder communication style** (memory `brian-communication-rules`): plain language, NO index codes in
  what you show him, quote-or-don't-assert (cite the code/docs, don't hand-wave), challenge don't agree,
  ONE fork at a time. He moves fast ("lets goooo") and dislikes walls of text and ceremony.

## 10. IMMEDIATE NEXT ACTIONS FOR THE RESUMING AGENT

1. **Get the founder's scope answer** (§2) — do not design past it.
2. Read the load-bearing docs you haven't: `core_ux_loop_and_aux_sidebar.md`, `mvp_slice_and_bridge.md` §4,
   `RULINGS-2026-07-30-space-and-journey.md`, the living-world design's Unit 1 note, and skim
   `PlayPage.tsx`/`api.ts` for the honest baseline.
3. Grill the founder through the forks (§8), one at a time, each with a recommendation. Record rulings.
4. Write the design to `docs/superpowers/specs/2026-08-07-journey-design.md` (or similar), founder-review it,
   then invoke writing-plans. Branch the build off `feat/living-world`.
