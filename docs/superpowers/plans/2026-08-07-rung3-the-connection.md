# The connection — BE ⇄ FE (Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the backend say enough for a real play surface to exist. Today one endpoint answers a beat with narration and nothing else — no place, no participants, no journey. This builds the contract the architecture doc already specifies: where you are and who's here, the beat loop as a stream of frames, a continue step, and the journey state that makes a multi-beat trip playable.

**Architecture:** One perception-bound scene projection, assembled server-side from the payload the engine already builds and currently throws away. One frame protocol over server-sent events, emitted per **validated** narration line so the belts still run before any text reaches a player. Streaming granularity is a *driver capability*, not a contract term — a driver that cannot stream emits the identical frames at the end, so the frontend is written once.

**Tech Stack:** Go (`core/api`, package `main`), `net/http` SSE (no new dependency), published JSON Schemas under `core/api/schema/` (the frontend repo generates its types from these), plpgsql + dbmate. Tests: `make reset && make test`; `cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./...`; `make schema-check`.

## Global Constraints

- **Branch:** `rung3/the-connection`, off `rung2/the-journey`. One PR.
- **Design source:** `docs/superpowers/specs/2026-08-07-journey-design.md` §4.8, ruling R10, and `docs/30_architecture/mvp_slice_and_bridge.md` §4. Read all three.
- **No canon crosses the boundary (B-1, I-3, D-7).** Every payload is a perception-bound projection assembled server-side. If a field could only be filled from raw canon, it does not belong in the response.
- **The frontend renders, never decides (D-7).** Ship labels, not raw state for the client to interpret. No field exists so the FE can compute something the server should have.
- **Time crosses as tick + `display_label` (B-5).** Wall-clock never enters a UI payload.
- **Every payload carries `schema_version`**, and every new shape gets a published schema in `core/api/schema/` — that directory is what the frontend repo generates from.
- **Clean cutover (founder-approved 2026-08-07):** `POST /worlds/{w}/beat` is **deleted**, not aliased or deprecated. The only caller is the throwaway test page.
- **The belts survive streaming.** A narration line is emitted only after `DecodeAndValidateNarration` has accepted it. Nothing unchecked reaches a player, ever.
- No project-wide formatters or linters; `go vet ./...` only.

**Scope honesty — what this does NOT build, and why.** The architecture doc's §4.3 async channel (`image.ready`, `projection.updated`, `backstage.applied`, `correction.window_closed`) is **out**: none of those subsystems exist yet, and standing up a socket that can only ever be silent is scaffolding pretending to be a feature. The same goes for the correction-window frame — there is no correction machinery to report. Both get built when there is something real to send. Say so in the PR rather than quietly shipping an empty channel.

**Confirmed interfaces (consume by exact name):**

- Route pattern: `regexp.MustCompile(`^/worlds/([0-9a-fA-F-]{36})/beat$`)` — `beathandler.go:20`; handlers implement `Match(r) bool` + `ServeHTTP`, registered in the `router{handlers: []matcher{…}}` literal at `main.go:54-64`.
- `func (h *beatHandler) payload(ctx, worldID, viewerID string) (PerceptionPayload, error)` — `beathandler.go:386`; `type PerceptionPayload` — `beatseats.go:10` (`Lines`, `LineIDs`, `Candidates []Candidate`, `ViewerAliases`); `type Candidate` — `beathandler.go:23`.
- `func narrateRoster(payload PerceptionPayload, viewerID string) ([]string, map[string]string)` — `beathandler.go:303` (present ids + the viewer's own display name per id).
- `func ResolveViewer(ctx, pool, worldID, debugOverride string, debug bool) (string, error)` — `viewer.go:17`.
- `func DecodeAndValidateNarration(raw string, presentIDs []string, speechTexts map[string][]string) ([]NarrationSegment, error)` — `narration.go:48`; `func narrateMessages(segments []NarrationSegment, labelFor map[string]string) ([]beatMessage, string)` — `beathandler.go:319`.
- `type Driver interface` and `GenRequest` — `bridge.go`; seats bound via `NewBridge(...)` — `main.go:48`.
- Journey state: `func (o *Orchestrator) activeJourney(ctx, worldID, actorID string) (*Journey, error)` and `type Journey` — `journey.go` (rung 2, commit `19fbcb2`); halt reasons `journey_leg | journey_arrived | journey_interrupted | journey_unresolved | journey_barred`.
- `func (o *Orchestrator) RunBeat(ctx, worldID, actorID string, chain []Attempt, startTick int64, trace *BeatTrace) (BeatOutcome, error)` — `orchestrator.go:83`. An **empty chain with an active journey is the continue press** (rung 2, commit `9ec9d7e`).
- Existing response shape to replace — `beathandler.go:264-286`: `schema_version: "beat_result/3"`, `narration`, `messages[]`, `result{committed, halt_reason, ticks_advanced, unresolved_candidates, telegraphs}`, `reasoning_log` (debug only, absent otherwise).

---

### Task 0: The watch horizon gets its own dial (correction to the previous step)

**Why first.** A watch ("wait until the ship is in") needs a horizon so nothing waits forever. It currently borrows `fn_duration_class_seconds('extremely_long')` — two hours — because the journey plan never said where that dial lived. It works, but a watch horizon is not a duration class, and the conflation will mislead the next reader. Twenty lines, and it lands before anyone plays.

**Files:** Create `core/db/migrations/20260807100005_watch_horizon.sql`, `core/db/tests/116_watch_horizon_test.sql`; modify `core/api/journey.go` (the horizon lookup only), `core/db/schema.sql`.

- [ ] **Step 1: Write the failing pgTAP test** — 3 assertions: an unseeded world falls back to a positive horizon (never NULL); a per-world row overrides it exactly; a freshly seeded world has the row.
- [ ] **Step 2: Run it, confirm it fails** (`function fn_watch_horizon_seconds does not exist`).
- [ ] **Step 3: Migration**, following `20260805100001_duration_class_config.sql`'s structure exactly: table `watch_horizon(world_id uuid PRIMARY KEY, horizon_seconds bigint NOT NULL CHECK (horizon_seconds > 0))`; `fn_watch_horizon_seconds(p_world_id uuid) RETURNS bigint` = `COALESCE(row, 86400)` — a day, which is a horizon rather than a speech length; `seed_world_defaults` extended by copying its **current** body verbatim (it now also carries the extent classes and the journey leg band) and appending one row. Down-migration restores `seed_world_defaults` **first**, then drops. Exercise `dbmate down` then up for real.
- [ ] **Step 4: Point `journey.go` at it** — replace the `durationClassSeconds(..., "extremely_long")` horizon lookup with `fn_watch_horizon_seconds`. Update the comment that explains the choice. The existing watch tests must pass unchanged except where they assert the literal 7200; those may move to the new default, and say so in your report.
- [ ] **Step 5:** `make reset && make test && make schema-check`, full Go suite, `go vet`. Commit: `fix(journey): a watch horizon is its own dial, not a duration class`.

---

### Task 1: Where am I, and who's here

**Files:** Create `core/api/scenehandler.go`, `core/api/schema/scene_current.v1.schema.json`, `core/api/scenehandler_test.go`; modify `core/api/main.go` (register the handler).

**Interfaces:**
- Produces: `GET /worlds/{w}/scene/current` → the scene projection; `func buildScene(ctx context.Context, pool *pgxpool.Pool, worldID, viewerID string, dbg bool) (sceneView, error)` reusable by Task 3's scene frame.

The payload, `schema_version: "scene_current/1"`:

```
{
  "schema_version": "scene_current/1",
  "place":        {"id": <uuid>, "label": <string>, "description": <string|null>, "tone": <string|null>},
  "participants": [{"id": <uuid>, "label": <string>, "kind": "actor"}],
  "now":          {"tick": <int>, "display_label": <string|null>},
  "journey":      <journey block | null>,
  "current":      [<string>]
}
```

- `place.label` and every `participants[].label` come from the viewer's own naming (`fn_display_name`, already applied to `payload.Candidates`) — never the canonical registry name the viewer may not know.
- `participants` is **characters only** (UX doctrine §2.2: never objects, locations, or factions as participants), excludes the viewer themself, and is exactly what `narrateRoster` already computes.
- `current` is the "what matters now" lines the viewer perceives — `payload.Lines`, which is already perception-bound and recency-windowed. It is prose the FE renders verbatim (D-7), not structured state to interpret.
- `journey` is the block defined in Task 2, or `null` when the viewer is not on one.
- `tone` is the place's authored tension where one exists — a word, not a number.

- [ ] **Step 1: Write the failing test** — `TestSceneCurrent_ShowsWhereYouAreAndWhoIsPresent`: against the seeded world, assert the place is the tavern with the viewer's label for it, that the present cast appears, that the **viewer is not in their own participants list**, and that `schema_version` is stamped. Add `TestSceneCurrent_LeaksNoCanon`: assert the response contains no `event_id`, no `canon`, and nobody the viewer cannot perceive — the wall test this endpoint must pass (B-1/I-3).
- [ ] **Step 2:** Run, confirm 404 (no such route).
- [ ] **Step 3:** Implement. `buildScene` calls the existing perception payload and `narrateRoster` rather than issuing new omniscient queries — the projection already exists, it was simply never returned.
- [ ] **Step 4:** Publish `scene_current.v1.schema.json` and validate the real payload against it (`ci/schema_contract.py` is the existing two-sided check — see `make schema-contract`).
- [ ] **Step 5:** Register in `main.go`, verify, commit: `feat(bridge): GET scene/current — where you are, who is here, what matters now`.

---

### Task 2: The journey block

**Files:** Modify `core/api/journey.go` (a projection function), `core/api/scenehandler.go`; create `core/api/schema/journey_block.v1.schema.json`; test in `core/api/scenehandler_test.go`.

**Interfaces:**
- Produces: `func (o *Orchestrator) journeyBlock(ctx context.Context, worldID, viewerID string) (*journeyBlock, error)`

```
{
  "active":        <bool>,
  "kind":          "travel" | "wait" | "watch",
  "goal_label":    <string|null>,     // the viewer's own name for the destination
  "where_label":   <string|null>,     // the place they are passing through, or null for open road
  "progress":      <number 0..1>,
  "legs_done":     <int>,
  "legs_total":    <int>,
  "interruptible": <bool>,
  "status":        "active" | "arrived" | "ended"
}
```

**Labels, not ids-plus-homework.** `goal_label` is what *this viewer* calls the destination (`fn_display_name`), because the frontend renders and never resolves. `progress` is derived server-side. `interruptible` is true while the journey is active — it is what lets the page honestly say "the world may still stop you" instead of implying safety.

- [ ] **Step 1: Write the failing test** — `TestJourneyBlock_ReportsWhereYouAreHeadedAndHowFar`: start a journey, assert the block's `progress` rises across legs, `legs_done`/`legs_total` match the row, and `goal_label` is the viewer's name for the target, not a raw uuid. Add `TestJourneyBlock_NullWhenNotTravelling`.
- [ ] **Step 2-4:** Implement, publish the schema, verify, commit: `feat(bridge): the journey block — where you're headed, how far, whether you can still be stopped`.

---

### Task 3: The beat as a stream of frames

**Files:** Create `core/api/sse.go`, `core/api/beatsstream.go`, `core/api/schema/beat_frame.v1.schema.json`, `core/api/beatsstream_test.go`; modify `core/api/main.go`.

**Interfaces:**
- Produces: `POST /worlds/{w}/beats` → `text/event-stream`; `type frameWriter` with `emit(kind string, payload any) error`.

Frames, in order, each an SSE `data:` line carrying `{"schema_version":"beat_frame/1","kind":…,"…":…}`:

| kind | when | carries |
|---|---|---|
| `interpretation` | as soon as the chain decodes | how the input was understood — the intent chain, for the Intent lens later |
| `narration` | one per **validated** line | `speaker_id`, `speaker_label`, `kind`, `text` — the existing `beatMessage` shape |
| `scene` | after the beat resolves | the Task 1 scene projection |
| `journey` | after the beat resolves | the Task 2 block |
| `result` | last | `committed`, `halt_reason`, `ticks_advanced`, `unresolved_candidates`, `telegraphs` |
| `error` | instead of the rest | a player-safe message; never a stack trace, never engine internals |

**The belts run before emission, without exception.** A narration frame is written only for a segment `DecodeAndValidateNarration` accepted. That is the whole reason narration streams per line rather than per token: a line is the smallest unit the ghost-speaker and verbatim-speech checks can judge.

**Failure mid-stream is a defined state, not an accident.** Once frames are on the wire the status code is already 200, so a later failure emits an `error` frame and closes. Every failure path in the handler must choose a frame; "it just stops" is a bug.

Flush after every frame (`http.Flusher`), or the whole exercise is theatre — buffered frames arrive together.

- [ ] **Step 1: Write the failing tests** — `TestBeats_EmitsFramesInOrder` (drive with `httptest`, parse the event stream, assert the sequence and that every frame validates against the published schema); `TestBeats_GhostSpeakerNeverReachesTheWire` (a driver authoring a speaker who is not present must produce no narration frame for it — the belt still bites mid-stream); `TestBeats_NarrateFailureEmitsAnErrorFrame`.
- [ ] **Step 2-5:** Implement, publish the schema, verify, commit: `feat(bridge): POST /beats streams validated frames`.

---

### Task 4: Real line-by-line, where the driver can

**Files:** Modify `core/api/bridge.go` (an optional streaming capability), `core/api/beatsstream.go`; test in `core/api/beatsstream_test.go`.

**Interfaces:**
- Produces: `type StreamingDriver interface { GenerateStream(ctx context.Context, req GenRequest, onDelta func(string)) (string, error) }` — an **optional** interface a driver may also implement.

**Why optional, and why the contract does not change.** The founder chose line-at-a-time delivery knowing the belts need whole lines. A driver that can stream lets the handler validate and emit each line the moment it completes; a driver that cannot produces the identical frames at the end. Same protocol, same frontend, better feel where the stack supports it. Granularity is a driver capability, not a contract term — this task is what makes that sentence true rather than aspirational.

- [ ] **Step 1: Write the failing test** — a fake driver that emits a structured narration array in chunks, asserting the first narration frame is written **before** the driver finishes. That ordering is the entire feature; a test that only checks the final output would pass without streaming existing.
- [ ] **Step 2-4:** Implement (type-assert the seat driver to `StreamingDriver`; fall back cleanly), verify both paths, commit: `feat(bridge): stream narration lines as they validate, where the driver supports it`.

---

### Task 5: Continue, and the old door closes

**Files:** Modify `core/api/beatsstream.go` (the continue route), `core/api/main.go`; delete `core/api/beathandler.go`'s route and handler registration; update `docs/30_architecture/mvp_slice_and_bridge.md` if it now misdescribes reality; tests in `core/api/beatsstream_test.go`.

- `POST /worlds/{w}/beats/continue` — same frame protocol, no body. It advances the moment by one beat and never fast-forwards (C-6). Mechanically it is `RunBeat` with an **empty chain**, which rung 2 already defined as the continue press.
- `POST /worlds/{w}/beat` (singular) is **deleted** — founder-approved clean cutover, no alias, no deprecation shim. Remove the route, the handler, and any test that exercised the old path *as a path* (tests covering the beat pipeline itself must be repointed, not deleted — if you find yourself deleting engine coverage, stop and report).

- [ ] **Step 1: Write the failing tests** — `TestBeatsContinue_AdvancesAJourneyOneLeg` (start a journey, POST continue, assert exactly one leg and a `journey` frame showing progress); `TestOldBeatEndpointIsGone` (`POST /worlds/{w}/beat` → 404).
- [ ] **Step 2-4:** Implement, verify, commit: `feat(bridge): POST /beats/continue; the singular /beat endpoint is gone`.

---

### Task 6: Gate

- [ ] **Step 1:** `make reset && make test && cd core/api && go test -count=1 ./... && go vet ./... && cd ../.. && make schema-check`
- [ ] **Step 2:** Twice with no reset between.
- [ ] **Step 3:** `make schema-contract` — the two-sided check that real payloads match the published schemas. Every new schema must be exercised by a real payload, not just declared.
- [ ] **Step 4: Drive it by hand.** Start the server, POST a beat that begins a journey, and watch the frames arrive with `curl -N`. Paste the real transcript into the report. A streaming endpoint nobody has watched stream is not verified.
- [ ] **Step 5:** Ledger entry, then the PR — quoting the founder's original ask (*"make sure the BE ↔ FE connection is relevant"*), listing what was deliberately left out (the async channel, the correction frame) and why, and citing B-1/I-3/D-7/C-6.
