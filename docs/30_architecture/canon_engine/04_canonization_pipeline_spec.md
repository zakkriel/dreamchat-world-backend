# 04 — Canonization Pipeline Specification

**Status:** Defines the central service: how raw interaction becomes (or doesn't become) canon. Covers the dual pipeline, the correction-window state machine, the template library format, the validation-gate API contract (proposal schema + structured rejection schema), the repair loop, and acceptance. Entity resolution internals are in doc 05; what happens after acceptance (projections, propagation) is in doc 03.

---

## 1. Purpose and position

The canonization pipeline sits between the conversation and the ledger. Its contract:

> **Input:** a beat (a bounded block of interaction: user action(s) + narrator output).
> **Output:** zero or more *accepted* canon events with their participant rows, mutations, perceptions, threshold entries, and (selectively) causal bundles — or structured rejections, or nothing at all.

"Nothing at all" is a first-class outcome: most chatter produces no canon (ADR-001/009). The canonization threshold is: an interaction is canon-worthy iff it changes state, creates or moves knowledge, or commits the future (promise/threat/plan). Examples of canon-worthy: entity learned something; promise made; item changed owner; relationship shifted; secret became public; rumor started; location/scene changed; conflict escalated; entity entered/left; in-world time advanced; a correction changed canon.

## 2. The dual pipeline

### 2.1 Fast path (deterministic, synchronous, no LLM)

Mechanical actions issued as recognized commands or unambiguous structured intents write **fully-formed accepted events** directly (origin `fast_path`), with their R1 mutations and perceptions, validated by construction.

**Fast-path action catalog (PoC):**

| Action | event_type | Mutations | Perceptions |
|---|---|---|---|
| move / enter / leave | `move` | actor.location_id; registry.current_scene_id | actor (direct); present witnesses (direct) |
| take / drop | `transfer` | artifact.holder/location | actor; witnesses |
| give / trade | `trade` | both inventories / artifact holder | both parties; witnesses |
| tell-as-command (explicit) | `disclosure` | — | listener (`told`), speaker (`shared`) |
| mark-public (command) | `publicize` | — | public-knowledge fan-out per scope rules |
| advance time | `time_advance` | world clock | — (feeds backstage pressure) |

The fast path is also the Phase 1 substrate and the scripted driver for the Mara slice (doc 07).

### 2.2 Slow path (LLM, asynchronous, inside the window)

Ambiguous narrative beats — promise, betrayal, threat, deception, rumor, emotional/relationship shift, implicit disclosure — route to extraction. **There are two defaults depending on whether the beat carries an irreversible change** (doc 12 §5, ADR-031):

- **Reversible beats (the common default): narrate first.** Prose streams immediately; extraction runs after, async.
- **Irreversible-intent beats (death, severe injury, irreversible transfer, major time jump, public reveal, faction stance shift, permanent relationship break, location destruction): preflight first.** The narrator emits its *intended* durable claims as a plan pre-pass, a feasibility preflight runs against optimistic state, and only on pass does the committing prose stream. On fail, the narrator streams a feasible alternative instead. This is the *only* case where anything precedes narration — and it exists so the player never reads an impossible irreversible event that then has to be awkwardly walked back.

Sequence (with step 0 applying only to irreversible-intent beats):

0. **(Irreversible only) Plan intent → feasibility preflight → gate the stream.** Fail → re-plan a feasible alternative before streaming. (doc 12 §5)
1. **Narrate.** The narrator's response streams to the user. For reversible beats this is the first step; extraction never blocks narration (ADR-010). Claim detection runs on the prose via narrator hints + an independent pass (doc 12 §2.1).
2. **Extract.** The extraction call receives: the beat transcript, the scene's entity whitelist (doc 05), the template registry index, and the proposal JSON schema. Priority: classify against a template → slot-fill; else free-form proposal (escape hatch → `pending_review`).
3. **Resolve entities** (doc 05). Unresolvable/ambiguous references become structured errors, not guesses.
4. **Gate.** The validation gate verdicts each proposal (§5).
5. **Repair (×1).** Rejected proposals with `repair_allowed=true` are returned to the extractor once, errors attached. Second failure → discard (logged) or park as `pending_review` if partially valid. Rejected *durable* claims must reconcile (doc 09), never silently vanish (I-10).
6. **Hold as `proposed`** until the window closes. Proposed events are invisible to projections, perceptions, thresholds, and replay.
7. **Accept on window close.** Surviving proposals transition to `accepted`; doc 03's triggers fire.

Every extraction round-trip is written to `extraction_log` (inputs, outputs, verdicts, repairs) — the audit trail and the future SLM corpus (ADR-012).

## 3. The correction window (state machine)

```
            user acts
                │
        ┌───────▼────────┐   narration streams immediately
        │  WINDOW_OPEN   │◄──────── extraction running async
        └───────┬────────┘
   user edits / rejects moment        user continues (explicit lock)      idle timeout / scene change
        │                                   │                                   │
        ▼                                   ▼                                   ▼
  proposals amended/discarded      ┌────────────────┐                  ┌────────────────┐
  (re-gate as needed)              │  WINDOW_LOCKED │  ───────────────►│   ACCEPTING    │
        │                          └────────────────┘                  └───────┬────────┘
        └──────────► back to WINDOW_OPEN                                       ▼
                                                              surviving proposals → accepted
                                                              gate re-checks state conflicts
                                                              (world may have moved)
```

- **Primary closure: explicit user lock (ADR-011).** The user reads the narration and continues — the continue interaction (or submitting the next action) locks the block.
- **Fallback closure:** idle timeout (default 120 s, configurable) and scene-change heuristic, so windows always close.
- **Correction semantics inside the window:** the user can rewrite or reject the moment; affected proposals are amended or discarded and re-gated. This is the *only* free-rewrite zone. After acceptance, corrections are present-forward compensating events (ADR-016).
- **Acceptance re-validation:** because the world may have changed between proposal and acceptance (fast-path events in the interim), the gate re-runs `STATE_CONFLICT` checks at acceptance time. Conflicts at this stage → `pending_review`, never silent acceptance.

## 4. The template library (ADR-012)

Templates are designer-authored, versioned JSON documents in a registry table (or files loaded at boot for PoC). A template defines the event shape, required participant roles, mutation recipes, perception recipes, and an optional bundle skeleton.

```json
{
  "template_id": "DISCLOSURE_BUNDLE",
  "version": 2,
  "event_type": "private_disclosure",
  "summary_pattern": "{speaker} privately tells {listener}: {gist}",
  "slots": {
    "speaker":  {"entity_kind": "actor", "required": true},
    "listener": {"entity_kind": "actor", "required": true},
    "gist":     {"type": "short_text", "required": true}
  },
  "visibility_scope": "private",
  "mutations": [],
  "perceptions": [
    {"holder": "{listener}", "epistemic_type": "told",
     "content_pattern": "{speaker} told me: {gist}", "confidence": 0.9},
    {"holder": "{speaker}",  "epistemic_type": "shared",
     "content_pattern": "I told {listener}: {gist}", "confidence": 1.0}
  ],
  "bundle": null
}
```

```json
{
  "template_id": "THEFT_BUNDLE",
  "version": 1,
  "event_type": "theft",
  "summary_pattern": "{thief} steals {item} from {place}",
  "slots": {
    "thief": {"entity_kind": "actor", "required": true},
    "item":  {"entity_kind": "artifact", "required": true},
    "place": {"entity_kind": "location", "required": true},
    "witnesses": {"entity_kind": "actor", "required": false, "many": true}
  },
  "visibility_scope": "secret",
  "mutations": [
    {"entity": "{item}", "attribute_path": "attrs.holder_id", "new_value": "{thief}"},
    {"entity": "{place}", "attribute_path": "attrs.status", "new_value": "compromised"}
  ],
  "perceptions": [
    {"holder": "{thief}", "epistemic_type": "direct", "content_pattern": "I stole {item} from {place}."},
    {"holder": "{witnesses}", "epistemic_type": "overheard", "confidence": 0.5,
     "content_pattern": "Something happened at {place} — I caught only part of it."}
  ],
  "bundle": {
    "semantics": "conjunctive",
    "inputs": [
      {"role": "enabler", "slot": "tool_event",        "input_kind": "event",      "necessity": true},
      {"role": "enabler", "slot": "opportunity_state",  "input_kind": "mutation",   "necessity": true},
      {"role": "trigger", "slot": "intent_perception",  "input_kind": "perception", "necessity": true}
    ]
  }
}
```

Rules: template firing is the **only** path that auto-creates bundles (ADR-008/012); bundle slot inputs must resolve to durable record IDs (doc 03 §1.4 hard rule) or the bundle is dropped (the event itself can still pass); the LLM never sees or invents the mutation/perception recipes — it supplies the slot fills, the system instantiates the rest deterministically. Initial PoC library: MOVE-class fast paths (not templates), DISCLOSURE, PUBLICIZE, PROMISE, THREAT, RUMOR_START, RELATIONSHIP_SHIFT, THEFT, BETRAYAL. Coverage gaps observed in `extraction_log` drive library growth.

## 5. Validation-gate API contract

### 5.1 Proposal envelope (extractor → gate)

```json
{
  "beat_id": "uuid",
  "world_id": "uuid",
  "proposals": [
    {
      "proposal_index": 0,
      "origin": "template",
      "template_id": "DISCLOSURE_BUNDLE",
      "event_type": "private_disclosure",
      "summary": "Player privately tells Mara about the hidden ledger",
      "in_world_time": "1402-06-10T21:30:00Z",
      "visibility_scope": "private",
      "confidence": 0.93,
      "participants": [
        {"entity_id": "uuid-player", "entity_kind": "actor", "role_qualifier": "speaker"},
        {"entity_id": "uuid-mara",   "entity_kind": "actor", "role_qualifier": "listener"}
      ],
      "slot_fills": {"speaker": "uuid-player", "listener": "uuid-mara",
                     "gist": "the mayor keeps a hidden ledger"},
      "mutations": [],
      "perceptions": [
        {"holder_id": "uuid-mara", "epistemic_type": "told",
         "content": "The player told me the mayor keeps a hidden ledger.",
         "confidence": 0.9, "acquired_at": "1402-06-10T21:30:00Z"}
      ],
      "bundle": null
    }
  ]
}
```

Free-form proposals use `"origin":"freeform"`, `"template_id": null`, and must carry explicit `mutations`/`perceptions`; they can be at best parked `pending_review`.

### 5.2 Verdict envelope (gate → pipeline / extractor)

```json
{
  "beat_id": "uuid",
  "verdicts": [
    {
      "proposal_index": 0,
      "verdict": "accepted_pending_window",
      "event_id": "uuid-assigned",
      "errors": []
    },
    {
      "proposal_index": 1,
      "verdict": "rejected",
      "repair_allowed": true,
      "errors": [
        {"code": "ENTITY_AMBIGUOUS", "path": "participants[1].entity_id",
         "detail": "'the guard' matches 2 active candidates",
         "candidates": ["uuid-guard-042 (night-shift museum guard)",
                        "uuid-guard-077 (gate guard)"]},
        {"code": "STATE_CONFLICT", "path": "mutations[0]",
         "detail": "actor uuid-seren does not hold artifact uuid-vase-09"}
      ]
    }
  ]
}
```

### 5.3 Error code registry

| Code | Meaning | Repairable |
|---|---|---|
| SCHEMA_INVALID | Output violates the proposal JSON schema | yes (×1) |
| TEMPLATE_SLOT_MISSING | Required slot unfilled | yes |
| ENTITY_UNRESOLVED | Mention matched no candidate | yes |
| ENTITY_AMBIGUOUS | Mention matched >1 candidate (candidates returned) | yes |
| ENTITY_OUT_OF_SCENE | Resolved entity not present/eligible in scene | yes |
| STATE_CONFLICT | Contradicts current projections (possession, location…) | yes |
| KNOWLEDGE_VIOLATION | Proposal requires a holder to know something with no valid path | yes |
| SCOPE_VIOLATION | Perception fan-out violates visibility scope | yes |
| TEMPORAL_VIOLATION | in_world_time out of order / before referenced records | yes |
| DUPLICATE_EVENT | Same beat already produced an equivalent accepted event | no |
| LOCKED_CANON_CONFLICT | Contradicts accepted canon (post-window) | no → pending_review |
| BUNDLE_INPUT_UNRESOLVED | Bundle slot lacks a durable record | no → drop bundle, keep event |

The structured `candidates` array in entity errors is what makes the repair loop converge instead of thrash.

### 5.4 Gate check order

Schema → entity resolution verification → scene/scope eligibility → state conflicts against projections → knowledge-path check (does each perception's holder have a valid acquisition path?) → temporal sanity → duplicate check → bundle input verification. First-failure-fast per check class, but **all** errors for a proposal are collected and returned together (one repair round must see everything).

## 6. UI contract (minimal but load-bearing)

The UI must: render narration immediately; show window state (open/locked) unobtrusively; expose the correction affordance during the window (rewrite/reject the moment); treat "continue" as the explicit lock (ADR-011); and surface `pending_review` items in a non-blocking inbox (creator-facing for PoC). The UI never sees canon rows — its timeline and sidebars come from perception/knowledge projections only (ADR-005).

## 7. Failure posture

Extraction service down → narration continues; beats queue for extraction; windows fall back to timeout closure with extraction completing late (events accept with late `recorded_at`, in-world time unaffected — the three-axis model absorbs this by design). Gate down → fast path continues (validated by construction); slow path holds proposals. Nothing about narrative availability ever depends on the extraction stack.
