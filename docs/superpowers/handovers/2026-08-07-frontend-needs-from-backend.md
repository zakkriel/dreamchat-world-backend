# NEEDS — what the frontend needs from the backend (2026-08-07)

**Author:** frontend review pass, `dreamchat-frontend` @ `main` (PR #5 merged: design system,
atmospheric chrome, play surface, runtime `schema_version` validation).
**Purpose:** one list of backend-side gaps that block frontend work, each with the code evidence and
the frontend consequence. Ordered by what blocks what, not by size.
**Not a design.** Nothing here proposes engine mechanism; every item is a contract surface the FE
consumes. The law lives in `00_strategy/06_rules_register.md` (D-6) and is only cited here.

---

## 0. Where the frontend is

**Unblocked and in progress (no backend dependency):** the named-slot app shell (SPEC-023), the
kind→component catalog (D-14), the configurable API base (SPEC-020), world id as runtime state
(SPEC-022, see the caveat in §1), and rebuilding the Compendium pages onto the design system —
including the required contract fields the pages currently drop (`known_artifacts`, `inline_links`,
`known_areas_inside`, `key_actors`).

**Blocked on this document:** everything in §1 (multi-world), §3 (the chunk-6 play surface and its
own gate), and §4 (the chunk-4 Carrying overlay).

---

## 1. World management — the one hard blocker (proposed SPEC-028)

**Evidence.** The router registers exactly eight handlers — three page, three index, one timeline, one
beat (`core/api/main.go:54-64`), all under the `/worlds/` prefix (`:66-67`). There is **no endpoint
that lists, creates, or describes a world**. Viewer identity is resolved server-side to the world's
single actor named `'Player'`, with `?viewer=<uuid>` honoured only in debug mode, and
`core/api/viewer.go:16` records "Auth/session out of scope this chunk".

**Frontend consequence.** SPEC-022 (`docs/open-spec-items.md:363-367`) says world id is runtime state,
multi-world from the start. The FE can deliver half of that today — the id stops being a compile-time
constant and flows through routing — but **a world cannot be chosen**, because nothing can answer
"which worlds are there, and what are they called". A picker would have to invent an endpoint, which
the anti-drift rule forbids (`implementation_playbook_superpowers.md:90` — document the gap, ship an
explicit stub, never an invented mechanism). So the FE ships a URL-supplied world id with a dev
default, and this entry is the stub's documentation.

**What is needed, minimally.**

1. **`GET /worlds`** — the worlds the caller may see. Per entry: `id`, a display label, and (folding in
   SPEC-019) its theme tokens. Perception-bound like every other payload: a world the caller has no
   path to is **absent**, not redacted (B-1, I-3). `schema_version` stamped like the rest (D-4).
2. **World-scoped viewer resolution** — today's "the actor named `Player`" default cannot survive more
   than one world. This pairs with the B1/auth stub (`MASTER_INDEX.md:124`); until then the FE keeps
   forwarding `?viewer=` verbatim and never interpreting it.
3. **A ruling on world creation** — either `POST /worlds` exists, or world creation is declared
   seed/tooling-only for now and the FE builds no create affordance. `MASTER_INDEX.md:125` lists
   "B2 — World creation" as a planned stub, so today the honest answer is probably "seed-only, say so
   out loud". **A decision here is worth more than an endpoint.**

**Not asked for:** any world *state* on this surface. A world list is a directory, not canon.

---

## 2. Cheap, already ledgered, needed for chunk 6 to look right

- **SPEC-021 — CORS for the FE origin** (`open-spec-items.md:356-361`). FE and BE are separate Railway
  services; without it the deployed FE cannot call the API at all. Owner: chunk 6, backend side.
- **SPEC-019 — world theme-token field** (`:341-347`). Accent, mood/treatment, ornament motif as plain
  data with `schema_version` + runtime validation. This is the "tokens are the floor" layer of D-15:
  the FE reads a world's tokens and recolors coherently with no code change, and the system never
  learns the word "fantasy" (GA-3). Fold it into `GET /worlds` (§1) rather than a separate call.

---

## 3. Needed for chunk 6's own gate — rung 3 of the journey design

Chunk 6 is *"Play surface polish — scene canvas, participants strip, Aux Current+Known lenses,
return-to-world flow"* and its human test is *"leave mid-scene, come back tomorrow: do you land
oriented?"* (`implementation_playbook_superpowers.md:72`). **That test cannot be run against the
current contract**, regardless of frontend effort.

- **`GET /worlds/{w}/scene/current`** — specified in `mvp_slice_and_bridge.md` §4.1 and defined in
  `docs/superpowers/specs/2026-08-07-journey-design.md` §4.8 (where you are, who is present, what
  matters now, plus the journey block). The engine **already assembles** exactly this per beat —
  present actors, current location, perceived artifacts, candidates, viewer aliases
  (`core/api/beatseats.go:10-20`) — and then **discards it**. Nothing else in the FE can render a
  scene canvas or a participants strip; the beat response carries no location, no roster, no absolute
  tick, only a `ticks_advanced` delta.
- **`POST /worlds/{w}/beats` + `POST /worlds/{w}/beats/continue`** with the SSE frame order and the
  label-only journey block (journey-design §4.8; R6, R7, R10). **Continue is a product rule the FE
  cannot fake** — C-6 says Continue advances the current moment and never fast-forwards, and there is
  no endpoint that advances one leg. Note the design removes singular `/beat` outright rather than
  aliasing it; the FE has exactly one call site plus three tests to change, so a clean cut is fine —
  just say when.
- Because streaming granularity is declared a driver capability rather than a contract term
  (journey-design §4.8), the FE will be written once and render identically whether frames stream or
  arrive in one burst. **No FE work is wasted if streaming lands later than the endpoints.**

---

## 4. Needed to close chunk 4's gate

- **`GET /worlds/{w}/carrying`** — listed in `mvp_slice_and_bridge.md` §4.1, not implemented. Chunk 4
  is gated on *"all four PRDs' read-side ACs"* (`implementation_playbook_superpowers.md:70`), and the
  Artifacts+Carrying PRD requires a Carrying overlay that is explicitly **not** the Artifact
  Compendium (AC#1) and whose Carry States render decay language when stale (AC#3). Everything else in
  that gate is frontend work already scheduled; this endpoint is the only missing input.

---

## 5. Contract hygiene — cheap, protects both sides

**Publish a `beat_result` JSON schema.** The five projection payloads are published in
`core/api/schema/` and vendored byte-identical into `dreamchat-frontend/contracts/`, so drift is
CI-gated in both directions. The beat envelope is **code-defined only**
(`core/api/beathandler.go:265-286`) — there is no `beat_result` schema, so it is invisible to
`ci/schema_contract.py` (which matches payloads to schemas by their own `schema_version`) and to the
frontend's `verify:contract`. The single most load-bearing payload in the product is the one nothing
checks.

The FE now validates what it can: read surfaces are checked against their pinned `schema_version`, and
the beat envelope is checked by **family** (`beat_result/*`) precisely because it has no published
schema to pin. A published schema would let both sides gate it exactly.

Related, and still open upstream: `mvp_slice_and_bridge.md:104` item 5 — *"payload `schema_version`
negotiation between FE and backend releases"* — remains undecided. The FE's current policy is: pinned
id per read surface, mismatch fails the load and renders the load-error surface, never invented data.
If the intended policy is different, that is a one-line ruling.

---

## 6. What the frontend will never ask for

Stated so it need not be re-litigated: canon, hidden truth, authoritative world state in play mode,
module code, or any judgement about outcomes, knowledge or correction validity (D-7, D-1, B-1, C-4).
If a payload appears to contain hidden truth, the FE treats that as a backend bug to report, never as
data to use (I-3).
