# FINAL — Resolve (SCAFFOLD — locked rulings + open questions)

**Job:** decide what actually happens for every attempt that survives the gate. The world's "no" lives
here. CONFIDENCE AT CHECKPOINT: ~35% — the call mechanics are the least-designed part of the system.

## Locked
- **Blocker-only deterministic side:** contract arithmetic runs at the GATE (see FINAL-action-contracts).
  Everything contract-less reaches the resolution LLM. No free passes; arithmetic never awards success.
- **Reality check first:** impossible-for-this-actor (no wings, novice-vs-golem) → BOUNCE, no canon.
  Distinct from failure, which IS an outcome and writes canon (she holds and lies).
- **Truth-side:** reads full committed canon of the involved parties (not the perceived scene).
  Involved actors' states must be in its input or it rules outcomes that ignore deafness/physics.
- **Tension review rides this seat:** at the start of resolving each act — recap in, tension updated,
  the act resolves under the new value (zero lag). Engine validates enum + provenance only.
- **Budget as input (Way A):** long actions fit themselves to the beat; progress accumulates as
  AttributeChanged until a threshold. The LLM never invents time the engine must swallow.
- **Mints only inside contract shapes;** the three nets (see contracts §7).
- **Output = typed events only** (the six types), treated as an untrusted proposal: engine verdicts
  (id-whitelist vs gathered slice, shapes, provenance), repair ×1, else bounce. Never re-resolved.
- **Ruled once, logged, replayed** — never re-run.
- **Two tiers of attributes (see contracts §5):** the Resolver freely invents Tier-2 attributes to track
  the status quo (engine never reads them), but may only write Tier-1 engine fields through committed,
  shape-validated events. **Discipline rule: if a fact should physically stop people, it lands in the
  Tier-1 field AND Tier-2 carries the meaning** — writing only the prose-state is incoherent state, a
  corrupted write, not a style choice.
- **Will is not adjudication:** whether an NPC *chooses* to comply is cognition (SPEC-012); resolve rules
  what reality permits.
- **Explain first, then match (2026-07-23):** the ruling writes its reasoning FIRST (from the gathered
  facts); the typed outcome must follow from that reasoning; outcome-contradicts-own-reasoning → repair ×1
  → bounce. Kills the recorded verdict-vs-reasoning failure. See RULINGS-2026-07-23 §1.
- **Colliding attempts = ONE combined ruling (2026-07-23):** held wind-up + reaction resolve as a single
  judgment over all coupled attempts + all involved parties' truth-side state → one set of typed events.
  No first-mover-by-call-order; the reaction is inside the judgment. Coupling detected mechanically;
  non-colliding chain steps stay sequential. See RULINGS-2026-07-23 §9.

## RULED since checkpoint (2026-07-23 grilling — see RULINGS-2026-07-23-grilling-session.md)
1. → **INPUT ruled:** involved entities + state/recent history + ONE hop of direct links; widen only on a
   real miss shown in play. Candidates/ids come from the graph; naming reach = the acting actor's OWN
   knowledge (§2–§3). Speech topics unknown to the speaker ride as words; the real entity enters via the
   LISTENER's knowledge — this also closes most of old #5 (a named/known entity's TRUE state is in the
   gather, so the ruling can't contradict it).
3. → **Contested exchange ruled:** one combined ruling (above).
6. → **Valence mitigation ruled:** explain-first-then-match (above). Model choice + prompt shape remain open.

## OPEN (remaining)
2. The OUTPUT schema, exactly (events + tension + visible/hidden split + minted rows — one shape;
   now must also carry the reasoning-first structure). **Semantic half ruled 2026-07-24** (see
   RULINGS-2026-07-24-reaction-beat.md §4, §6): each ruled event = TRUTH (canon, never lies) +
   optional default APPEARANCE + optional per-receiver variants (+ visible=false ⇒ no perceptions);
   attribute writes land as state_mutation attrs paths, Tier-1 strictly validated in code, Tier-2
   free. Remaining: the concrete JSON schema revision + prompt shape (Station D).
4. The repair loop concretely (what errors are attached, what the second pass may change).
5. (narrowed) Coherence beyond the gathered slice — cases where nothing in the action links the
   contradicted fact. The naming/gather rules close the known cases; watch play for residue.
6. (remaining half) Model choice / prompt shape per seat, re-validated against these mechanisms.
