# 12 — Narrative Claim Ledger Spec

**Status:** New first-class primitive. Closes the structural gap between generated prose and accepted canon. This is the bridge that makes under-canonization measurable, reconciliation trackable, and chained-action dependencies inspectable — replacing audit-sampling estimates with a queryable lifecycle. Read alongside doc 04 (pipeline), doc 09 (reconciliation), doc 10 (clock/`beat_seq`), doc 07 (the invariant this enables).

**Origin:** proposed in adversarial review as the missing intermediate artifact. Adopted because it is the common structural fix for four previously-separate soft spots (O-2, O-4, O-7, O-8 in doc 11).

---

## 1. The gap it closes

The pipeline went straight from prose to proposed canon:

```
narration → extraction → proposed canon → validation
```

The problem: nothing recorded *what the prose asserted*. A durable claim the extractor missed left **no trace** — so "did we canonize everything meaningful?" could only be answered by sampling transcripts against the event log (the weak side of I-6). A rejected claim could vanish silently (the gap doc 09 addresses but couldn't *track*). Chained actions had no way to see a prior beat's not-yet-accepted assertions.

The fix inserts one lightweight artifact:

```
narration → narrative claims → proposed canon → validation / reconciliation
```

A **Narrative Claim** is not canon. It is a record that the generated prose asserted a durable thing, with a mandatory terminal status. Every durable assertion now has a lifecycle instead of relying on audit luck.

## 2. What a claim is

A claim is detected from narrator (or user) prose whenever it asserts a **durable state change** — the same catalog as doc 09 §2: transfer, movement, arrival/departure, injury/death/status, relationship shift, promise/oath, knowledge transfer, disclosure, public announcement, time advance, faction shift, play-affecting environmental change. Pure flavor produces no claim.

### 2.1 Detection: narrator hints + independent pass (the circularity caveat)

Claims are detected two ways, and **both** run — this is deliberate, not redundant:

1. **Narrator-emitted hints (cheap first signal).** The narration call returns lightweight durable-claim hints alongside prose: `{"durable_claims":[{"type":"item_transfer","span":"the guard hands you the key"}]}`. Not canon, not schema-heavy — just the narrator flagging what it believes it asserted. This reduces missed detection at near-zero cost.
2. **Independent detection pass (the backstop).** A separate detection step scans the prose for durable assertions *without* trusting the hints.

**Why both (the caveat that matters):** the narrator is the least trustworthy component in the system. A narrator that hallucinates a key transfer will also hallucinate its hint; a narrator that omits a transfer from prose will omit its hint too. Self-reported claims are *correlated with the prose's own errors, not independent of them* — so hints alone cannot break the circularity ("the ledger only works if detection works, and detection here is the same unreliable model"). The independent pass is what provides the uncorrelated signal. Disagreement between the two (hint without detection, or detection without hint) is itself a high-value flag routed to review. Hints make detection *cheaper and better*; they do not make it *trustworthy on their own*. Do not collapse to hints-only — that would relocate under-canonization one layer earlier and hide it behind a name.

```json
{
  "claim_id": "claim_123",
  "world_id": "uuid",
  "beat_id": "beat_77",
  "beat_seq": 2,
  "source_span": "The guard hands you the iron key.",
  "claim_type": "item_transfer",
  "impact": "reversible",                     // reversible | irreversible (drives preflight, doc 09)
  "entities": {
    "giver": "npc_guard_04",
    "receiver": "player_01",
    "item": "item_iron_key"
  },
  "requires_canonization": true,
  "status": "detected"
}
```

`entities` here are *resolved* references (doc 05) where possible; unresolved entities are carried as mention strings and become an entity-resolution error downstream, not a silent drop.

## 3. Terminal status (the invariant)

Every claim with `requires_canonization=true` must eventually reach exactly one terminal status:

| Status | Meaning |
|---|---|
| `canonized` | Became an accepted canon event (with mutations/perceptions) |
| `non_canon_flavor` | On inspection, not actually durable — downgraded, no canon |
| `converted_to_attempt` | Reconciled as an attempted-but-incomplete action (doc 09 §4.1) |
| `repaired` | Reconciled via diegetic contradiction or clarification (doc 09 §4.2–4.3) |
| `pending_review` | Cannot resolve safely yet; parked — a **legal beat-close state** (see below) |
| `missed` / `error` | Detected but neither canonized, reconciled, nor parked — a **defect** |

**Invariant I-10 (corrected — closure must not fight async flow).** The naive form ("all claims resolve before the beat closes") would reintroduce exactly the latency ADR-010 removed: if "Continue" closes the window while extraction is still running, a hard resolve-before-close rule becomes a blocking join. The correct invariant is therefore about *non-abandonment*, not immediacy:

> **No unresolved claim may be ignored, and no dependent action may bypass an unresolved claim it depends on.**

Concretely: a beat may close with a claim in `detected`/`proposed`/`pending_review` — extraction can still be in flight, and that's fine. What is forbidden is (a) a claim *disappearing* without reaching a terminal state (that's `missed`, a defect), and (b) a later action that *depends on* an unresolved claim proceeding as if it were resolved (the chained-action guard, ADR-022). `pending_review` is a legal, durable resting state; `missed`/`error` is never legal. The instrument is unchanged: `count(*) WHERE status IN ('missed','error')` is the under-canonization metric, and a non-empty result is release-blocking; long-lived `pending_review` is a backlog signal, not a failure.

## 4. What it fixes, concretely

- **Under-canonization (O-4) becomes a query, not a sample.** `SELECT count(*) FROM narrative_claim WHERE status='missed'` is the metric. The asymmetric-audit posture of ADR-023 stops depending on N=20 sampling; missed canon is detected per-claim, every beat.
- **Reconciliation (O-2) becomes trackable.** Every rejected durable claim must land in `converted_to_attempt` / `repaired` / `pending_review` (or `canonized`/`non_canon_flavor`) — never silently vanish. Doc 09's "no silent discard" rule is now *enforced by the ledger's non-abandonment invariant* (I-10), not merely asserted. A beat may close before a claim resolves; the claim may not be *lost*.
- **Chained actions (O-8) can inspect claims.** The gate's pending-proposal check (ADR-022) reads same-scene claims, not just proposed events — a follow-on action ("unlock the door") can see the prior beat's `item_transfer` claim for the key even before it accepts.
- **Template coverage (O-7) becomes visible.** Claims that no template matched and that fell to the free-form escape hatch are flagged on the claim; the unmatched-claim-type distribution is the coverage-gap signal that drives template authoring.
- **High-impact preflight (doc 09 O-2 resolution) targets known claim types.** Preflight runs on claims with `impact=irreversible` *before* their prose commits (see §5).

## 5. Relationship to high-impact preflight

Doc 09 resolves the reconciliation-timing fork (O-2) by preflighting only irreversible categories. **The sequencing is the whole point and must not be inverted:** preflight operates on *planned* claims (narrative intent) **before** the committing prose streams — never on a detected claim after the fact, which could only repair, not prevent.

```
WRONG:  stream prose → detect claim → preflight   (can only apologize)
RIGHT:  plan intended durable claims → preflight high-impact ones → stream prose
```

Mechanically, for beats the narrator intends to carry an irreversible change (death, severe injury, irreversible transfer, major time jump, public reveal, faction stance shift, permanent relationship break, location destruction), the generation step first emits its *intended* durable claims as a plan (a cheap structured pre-pass, distinct from the post-prose hints in §2.1), the **feasibility preflight** runs against optimistic state (accepted canon + same-scene pending claims):

- Does the giver possess the item? Is the target alive? Is the door here? Can the faction shift this way?
- **Pass** → the narrator streams the committing prose; the claim proceeds normally to canonization.
- **Fail** → the narrator never streams the impossible version; it streams a feasible alternative instead. No retrospective repair for the cases where repair feels cheapest.

This is *feasibility*, not full canonization — cheap, bounded to rare high-stakes beats, and the only place the architecture pays pre-narration latency. Reversible claims skip the plan-and-preflight step entirely: they stream freely and rely on §2.1 detection + retrospective reconciliation (doc 09 Option B). The irreversible-category list is the trigger for the intent pre-pass.

## 6. Lifecycle

```
beat begins
  → if narrator intends an irreversible change:
        emit intended durable claims (plan pre-pass)
        → feasibility preflight against optimistic state (§5)
              fail → narrator streams a feasible alternative instead
  → prose streams
  → detect durable claims: narrator hints + independent pass (§2.1)   [status: detected]
  → resolve entities (doc 05)
  → extraction maps claims → proposals                                [status: proposed]
  → validation gate (doc 04)
        accept → [canonized]
        reject + durable → reconcile (doc 09)   [converted_to_attempt | repaired | pending_review]
        downgrade        → [non_canon_flavor]
  → beat close: assert I-10 (no ignored/bypassed unresolved claims; see §3)
```

## 7. Phase scope

- **Phase 0A:** not present (no narration, no extraction).
- **Phase 2:** the ledger arrives *with* the slow path — claim detection, terminal-status enforcement, the under-canonization query. This is the phase where under-canonization is the primary risk (ADR-023), and the ledger is its instrument. **Treat Phase 2 as an instrumented experiment, not just a feature phase:** the ledger adds real cost (narrator hints + independent detection + extraction + validation + possible repair + audit), and the load-bearing metric is `durable claims detected / durable claims later judged real`. If detection is weak, canon quality is weak — so this ratio, not feature completeness, is the Phase 2 gate.
- **Phase 3+:** preflight on irreversible claims; chained-action inspection.

The ledger is lightweight (one table, one detection pass, one closing assertion) but it is the difference between *hoping* the world doesn't forget and *knowing* it didn't.

## 8. Minimal schema

```sql
CREATE TABLE narrative_claim (
  claim_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id        UUID NOT NULL,
  beat_id         UUID NOT NULL,
  beat_seq        INT  NOT NULL,           -- intra-tick ordering (doc 10 §4); ordering lives here, not on a timestamp
  source_span     TEXT NOT NULL,
  claim_type      TEXT NOT NULL,
  impact          TEXT NOT NULL CHECK (impact IN ('reversible','irreversible')),
  entities        JSONB NOT NULL DEFAULT '{}',
  requires_canon  BOOLEAN NOT NULL DEFAULT true,
  status          TEXT NOT NULL DEFAULT 'detected'
                  CHECK (status IN ('detected','proposed','canonized','non_canon_flavor',
                                    'converted_to_attempt','repaired','pending_review','missed','error')),
  resolved_event_id UUID REFERENCES canon_event(event_id),  -- set when canonized
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_nc_beat   ON narrative_claim (beat_id);
CREATE INDEX idx_nc_missed ON narrative_claim (world_id) WHERE status IN ('missed','error');
CREATE INDEX idx_nc_open   ON narrative_claim (beat_id) WHERE status IN ('detected','proposed');
```
