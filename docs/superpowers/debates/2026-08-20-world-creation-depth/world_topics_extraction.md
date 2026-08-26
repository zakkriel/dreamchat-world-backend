# World topics — extraction seat

## How coverage happens without a spine (the mechanism, before the list)

The seat never sees a topic checklist — that checklist IS the ontology GA-2 forbids. Coverage
comes from two pressures that require no topic-awareness in the model at all:

**Referential pull.** `cast[].starts_in` must resolve to `places`; `history[].who` must resolve
to `cast`; a norm's `binds` must resolve to `cast`/`collectives`. This is the same mechanism
`validate()` already runs as a *negative* check (`worldgenesis.go:249-495`, dangling-reference
refusals) — used here as the *positive* elicitation force. A collective is authored only when
something else already needed to point at one. Nothing asks "does this world have collectives?"

**A small, closed, mechanism-adequacy floor** — not a content spine: ≥2 `places`, ≥1 `cast`,
exactly one `hiding` per person. These assert that the world is *walkable* and *has a private fact
worth the naming wall existing for* — properties of the engine's own machinery, true in a sci-fi
thriller, a workplace drama, and a horror story alike (GA-2's own test). Everything past the floor
is `minItems: 0`.

So "comprehensive" means: every cluster with an *accepted engine destination* is reachable by
reference, costs nothing unreached, and is never solicited by a checklist. Validation's job is
never "did you cover X" — it is "does everything you *did* author resolve, and does the floor
hold." Under-authoring is a dangling reference or a floor miss, never a missing topic.

## 1. Cluster list — destination, and what needs engine work

| Cluster | Destination | Status | Engine work needed |
|---|---|---|---|
| Place (region/places/ways) | `entity_registry`/`location_state`/`artifact_state` | live | none |
| Cast (descriptor/traits/malleability/speech_manner/hiding) | `entity_registry`(actor)/`personality_core`/`perception_record` | live | none |
| Objects | `entity_registry`(artifact)/`artifact_state` | live | none |
| History + per-holder knowledge | `canon_event`/`perception_record` | live | none |
| Collectives (optional) | `entity_registry` `entity_kind='group'` | legal, unused by genesis | **none — genesis-side only, no engine change** (ref-doc row 144) |
| Motion (movement + pace_class + hindrance) | `movement_type`/`status_modifier` | tables live, mintable | **yes, blocking:** `fn_move_duration_actor` hardcodes `'walk'` (`20260729100006_move_target.sql:63,66`); `base_speed` is a dead Tier-1 key (`tier1.go:15`); no actor↔movement-type binding exists. Do not mint this cluster before the binding ships — it repeats `cast[].standing` with an extra table. |
| Passage (`ways.impedes`) | portal Tier-1 `impedes` on movement type | **accepted shape** (`handover §4.2`) | `fn_portal_permits` gains a `movement_type` parameter; needs the founder ruling at §5.1 (four call sites cascade: `fn_portal_permits`, `premiseHolds`, `fn_actor_move_permitted`, `fn_fact_sheet.reachable`) |
| Speech constraint (obligatory/forbidden forms) | `speech_act_type`/`speech_constraint` | **accepted shape** (`handover §4.1`), **zero authoring path today** | decomposer classification — "within its existing job" per §4.1, the *cheapest* of the pending engine work |
| Norms — routing, not a cluster | whichever of {speech_constraint, passage.impedes, ordinary history+knowledge} the binding implies | not yet designed as a router | genesis authors ONE sentence + a binding; commit-time classification (grammar, closed, ours) picks the destination — see §4 |
| Near-future (optional) | `pending_event` | table live, read every clock crossing, written by tests only (`ledger.go:122-220`) | none — genesis becomes first writer |

## 2. Where the GA-2 line sits

Not "structure vs. no structure" — the schema already requires ≥2 places, ≥1 person, one hiding
each. The line is **topic vs. content, tested jointly by genre-invariance and reader-existence**.
A cluster earns a place only if (a) the *dimension itself* — not its values — survives being asked
in a sci-fi thriller, a workplace drama, and a horror story (GA-2's own test), **and** (b) it maps
onto engine machinery that is genre-blind by construction and already computes with *something*
there regardless of story (a duration, a trust decay, a beat budget). Movement, tension, epistemic
type all pass both halves. A topic that passes (a) alone but fails (b) — `role`/`wants`/`doing`,
genre-agnostic-sounding, zero reader — is not yet earning its place; it is authored-but-inert, the
*same defect* as a fixed spine, failing from the opposite direction: not "the service learned what
worlds usually have" but "the model was asked to spend tokens describing nothing anyone reads."
Reader-existence is not a nice-to-have on top of GA-2 compliance — it is *part of* the test.

## 3. What I would cut from the reference shape

- **`role`, `wants`, `doing`** as separate cast fields. No reader (ref-doc row 149). Their content
  is already covered once `belongs_to`/`hiding`/history's per-holder `knowledge` land properly —
  three more unread strings is `cast[].standing` times three, not more depth.
- **`regard[]`** in its current shape. Destination `relationship_state` has zero readers in
  `core/api`; the `[RELATIONSHIPS]` block is specced (`06_context_assembly_spec.md:76,88`) and
  never rendered (ref-doc row 150). Same class of trap as the group-held-perception leak: don't
  ship a cluster whose destination nothing reads.
- **`rules[]` as one undifferentiated bucket.** See §4 — it should not be a cluster with a fixed
  shape; it should be one authored sentence the *engine* routes.

## 4. The single biggest omission

The accepted speech-constraint mechanism (`handover §4.1`: obligation collapses to prohibition on
its negation, `speech_act_type` + `speech_constraint`, disposition `forbidden|conditional`) has
**no authoring path into genesis at all.** `rules[]` in the reference shape is generic prose
destined for "prompt-rendered rows (soft)" — ref-doc row 145, unread by anything mechanical —
when the engine already has a real, closed, gate-enforceable mechanism for exactly "who may say
what to whom," the founder's own headline example (`BRIEFING.md`: *"who may speak to whom, enter
where"*). The gap is not missing depth; it is an accepted mechanism with nothing wiring a brief
into it. A norm should be authored once — `{stated, binds}` — and classified at commit time
(closed, ours, the decomposer's existing job per §4.1) into `speech_constraint` when it governs
speech, `ways.impedes`-adjacent passage rules when it governs entry, or ordinary
`history[].knowledge` when it is reputational only. One authored fact, engine-chosen destination —
never a fourth bucket the seat has to pre-classify itself.

## 5. Fails a handover §1.2 test

**"Every authored leaf reaches a reader"** — fails, self-documented, in the reference file itself.
The worked example authors `role` (line 87), `doing`/`wants` (lines 89-90), and `regard` (lines
93-95) as ordinary cast fields sitting beside fully-live ones (`traits`, `hiding`). The *same
document's own reader-status table*, four rows later, marks all four ❌ "no reader anywhere" /
"zero readers" (rows 149-150). This is `cast[].standing` reproduced four times over inside the
file whose stated purpose is fixing that exact defect. Per §1.2's own instruction — "do not add
more of them" — these four fields should not appear in any worked example presented as current
shape; §3's cut list is the correction.
