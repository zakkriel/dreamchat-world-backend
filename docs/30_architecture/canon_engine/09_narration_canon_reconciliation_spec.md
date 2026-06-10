# 09 — Narration / Canon Reconciliation Spec

**Status:** Closes the highest-priority gap in the set. Defines what happens when the narrator's prose depicts a state change that the validation gate later rejects. Read alongside doc 04 (the pipeline that produces the rejection) and doc 02 (ADR-009, the gate; ADR-016, present-forward correction).

**Honesty note:** the central timing tension in §3 is **not fully solved** and is flagged as an open question in doc 00. This spec defines the policy; Phase 2 must validate that the policy reads as believable in practice.

---

## 1. The problem

The architecture narrates *first* and extracts/validates *after* (ADR-010 — narration must never block on extraction). This creates a window in which the player has already read prose asserting something the gate then rejects: the narration said "the guard hands you the key," but the gate rejects the transfer because the key isn't where the proposal claims. The player's *experienced fiction* now diverges from accepted canon.

The doctrine says canon is truth and narration is not (doc 01). That is correct for the database. But for the player, **the prose they read is their reality** — and silently discarding the rejected proposal (doc 04's default for rejections) means the world quietly forgets something the player watched happen. That is the exact drift the whole system exists to eliminate, reintroduced at the narration seam.

Core rule:

> A rejected state-changing narration may not vanish silently. If prose asserted a meaningful change, the pipeline must resolve it visibly — as accepted canon, a diegetic repair, a clarification, or an explicit unresolved state.

This rule protects the *player's* fiction; it does not make narration authoritative over canon. Both walls stand.

## 2. What counts as a narrated state change

Reconciliation is triggered only when rejected prose asserted a **durable** change: object/inventory transfer; movement; arrival/departure; injury, death, status change; relationship shift; promise/oath/commitment; knowledge transfer or secret disclosure; public announcement; time advance; faction stance shift; environmental change that affects play.

Pure flavor — mood, body language, atmosphere, sensory color — is **not** a state change and is never reconciled, *unless* a later beat references it as durable fact (at which point it becomes a proposal and goes through the gate like anything else).

## 3. The timing tension (the unsolved core)

The clean reconciliation route — silently rewrite "hands you the key" into "starts to, then refuses" — assumes the system can intercept the rejection **before the player reads the prose**. The asynchronous design means it usually cannot: the sentence has already streamed. Therefore reconciliation is, in the general case, **retrospective** — it can only follow the depicted change with an in-fiction reversal, which reads as a character changing course, not as the system erring.

Two architectural responses, both with costs; the choice is deferred to Phase 2 evidence:

- **Option A — synchronous pre-validation for state-changing beats only.** Before streaming prose that asserts a durable change, run a fast gate check on the *intended* change. Cost: reintroduces latency on exactly the beats the correction-window design wanted to keep fast; requires the narrator to declare intent before prose (a structural change to the generation call).
- **Option B — always-retrospective reconciliation.** Accept that prose streams first; reconciliation always takes the form of a *subsequent* in-fiction development (§4.1–§4.4). Cost: some reversals will read as NPC caprice rather than correction; relies on the narrator's skill at making the reversal feel motivated.

PoC stance (resolved): a **bounded hybrid**. Reversible claims use Option B (retrospective reconciliation, no latency cost). Irreversible claims — death, severe injury, irreversible transfer, major time jump, public reveal, faction stance shift, permanent relationship break, location destruction — use a **feasibility preflight** (Option A, scoped to this small set) *before* the narrator streams the committing prose, because these are exactly the cases where retrospective repair reads as cheap or insulting. Preflight is a feasibility check against optimistic state (accepted canon + same-scene pending claims), not full canonization: does the giver possess the item, is the target alive, is the location intact. Fail → the narrator never streams the impossible version and generates a feasible alternative instead. Preflight operates on the Narrative Claim Ledger (doc 12 §5), which detects and flags the irreversible claim types. This replaces the earlier "start with B, maybe add A later" hypothesis with a committed asymmetry: latency is paid only on rare high-stakes beats.

## 4. Reconciliation routes

Selected deterministically by the pipeline based on rejection type and reversibility:

**4.1 Convert to attempt** — the prose can plausibly become an attempted-but-incomplete action. Rejection: STATE_CONFLICT where the action was possible to try. Follow-prose: the guard *starts* to hand it over, then stops ("the captain still has it"). Use when the action is reversible and the reversal can be motivated.

**4.2 Diegetic contradiction repair** — the prose asserted something impossible. Rejection: the referenced entity/state doesn't exist. Follow-prose: the hand closes on empty table; there was no sword. Use when no plausible "attempt" framing exists.

**4.3 Clarification prompt** — repair would override player intent or confuse. Out-of-fiction-light prompt: "Did the guard actually give you the key, or refuse?" Use sparingly; it breaks flow and should be the exception.

**4.4 Pending-review pause** — the system cannot repair safely without deeper context. The scene avoids building further consequences on the unresolved change; the item surfaces in the creator inbox (doc 04 §6). Use for genuinely ambiguous high-stakes cases.

## 5. Lifecycle

```
narration streams → extraction → gate rejection
   → classify: did the rejected prose assert a durable change? (§2)
        no  → discard silently (it was flavor); done
        yes → select route (§4) by rejection type + reversibility
            → generate repair / prompt / pause
            → no later beat may depend on the rejected change until resolved
```

## 6. Hard constraints

Do not silently discard rejected state-changing narration. Do not accept invalid canon merely because narration asserted it. Do not let future beats depend on rejected canon. Do not hide a mismatch that affects the player's experienced reality. Do not expose database internals in the repair ("the validation gate rejected…" is forbidden player-facing text).

## 7. Success criterion

The player should feel the world *corrected, interrupted, or clarified itself naturally* — never that the previous message was ignored. This is a subjective bar; Phase 2 measures it by sampling reconciliation events and judging whether the follow-prose reads as motivated in-fiction. A high rate of route 4.3 (clarification prompts) is a signal that extraction or entity resolution upstream is too weak — reconciliation is the safety net, not the primary mechanism.
