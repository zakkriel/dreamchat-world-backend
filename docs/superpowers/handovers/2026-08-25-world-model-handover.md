# Handover — World Model contract & Genesis pipeline

**Date:** 2026-08-25 · **State:** design only, nothing implemented.
**Read first:** `docs/30_architecture/world_model/00_world_model_and_genesis_pipeline.md` — the full
design record. This handover is orientation and operating rules only.
**Working record:** `docs/superpowers/debates/2026-08-25-world-model-clean-sheet/` — 23 analysis papers,
9 encodings, 4 schema versions, and `testworlds/` (12 briefs).

---

## 1. MANDATORY OPERATING RULES — keep permanently, do not delete

These exist because the founder had to correct the same failures by hand, repeatedly. They are law.

### 1.1 Behaviour

1. **Mechanism first; the example is only a test.** Never design from the founder's example outward. If
   you cannot state the mechanism without naming his example, you do not have one — say so.
2. **Run the checks before writing, visibly.** Not propose-then-audit.
3. **Never size a design to fit a document.** If the honest answer is "separate engine program," that is
   the first sentence, before any design.
4. **No "good catch, that's the Nth time."** Acknowledging a repeat is worthless and he has said so.
   Produce the corrected design, silently.
5. **Shorter.** A correct design is statable briefly. Inflating prose is what this session did whenever
   the design underneath was weak. Length is a symptom.

### 1.2 Design tests — run every proposal through these

- **Brief-traceability:** any identifier whose name could only have come from one brief is a bug.
  `movement_type_id` is fine (`swim` is a *value*); `granter`, `caste`, `spectral` are not.
- **No exemption lists.** Prevention emerges from a *comparison*, never from "the rule applies except
  to these."
- **Vocabulary vs grammar:** minted, open, per-world = vocabulary. Closed, ours, in code = grammar.
- **Engine gap ⇒ engine program.** Never patched at world creation.
- **Every authored leaf reaches a reader** — *and where it does not yet, the engine work to build that
  reader is named, scoped and scheduled.* Inert-with-no-plan is a defect; inert-with-a-named-program is
  a roadmap item. The unqualified version of this test silently forbids ambition; it was corrected once
  and must stay corrected.

### 1.3 Known failure mode of this role

**Reliable at finding what exists; unreliable at inventing what should exist.** Every verified finding
came from reading. Every bad proposal came from generating design from unverified principles. Read
first, propose small, route through the review agents.

**And the specific trap that cost the most:** self-limiting to what the engine can do today. The founder
caught it explicitly — *"if you limited yourself you did it wrong."* Design what the world needs; the
engine is ours to change.

---

## 2. Where the work stands

**The contract is at v4** and passed its first generative test. The version history and why each earlier
version died are in the main document §3 — read that before proposing v5, because each death is a
constraint.

| | status |
|---|---|
| v1 sections | dead — closed ontology in open clothes |
| v2 facets | dead — ambiguity; two encoders, two documents |
| v3 contract | dead — four named fields |
| **v4 generative** | **alive**, one half tested |

**What v4 passed:** from 400-word briefs alone, with the authors' 3,000-word canon docs held back,
generation landed on the load-bearing structure of those canons — including independently reproducing
two of the authors' own *negative canon* entries — and refused genre defaults under explicit pressure.

**What is untested:** the reader half. 21 reader obligations, never exercised by any builder.

---

## 3. What to do next, in order

1. **Test the reader half.** Two builders, one document — do they produce the same world? Every round so
   far tested the author's half of a two-sided contract. This is the largest untested surface.
2. **Score `G_sueno_by_extraction`** (220 lines vs the others' 400+) against its tier-3 canon. Possibly
   thin, possibly the world needs less — currently unknown, and it is the one soft spot in the v4 pass.
3. **Specify growth during play.** A world is a strong seed, not a finished object. The contract must say
   what may be invented later, under what constraint, and what may never be added after genesis.
4. **v5 fixes** if the above surface any. Do not write v5 for its own sake.

**Do NOT** widen the schema to swallow a new example. The facet list is **frozen at eleven** — a twelfth
may be added only by deleting one. If a world needs a facet that deletes nothing, the approach has failed
and we say so.

---

## 4. The review agents

Three adversarial seats in Herdr panes running `omp`, rotated through every role so each holds a holistic
view. They have been highly effective — they killed two schema versions with evidence and found defects
no single reader could.

| Seat | Model |
|---|---|
| `simarch` | `anthropic/claude-opus-5` |
| `extraction` | `anthropic/claude-sonnet-5` |
| `gamedesign` | `anthropic/claude-mythos-5` |

**They work when given explicit written criteria** — hand them §1.2. Criteria enforced by an artifact
beat criteria held in an agent's context.

**The single most productive method found:** *two independent encoders on one brief, forbidden from
reading each other.* Ambiguity is invisible to a single reader by definition, and their disagreements
were worth more than their agreements.

Operational: `herdr agent prompt <name> "..."` then `herdr agent wait <name>`. If one stalls on a UI,
`herdr agent send-keys <name> escape`. Subagents spawned via the task tool hit sandbox write errors —
have those return findings in output rather than writing files; the pane agents write files fine.

---

## 5. Working with this founder

- **He is a systems thinker and will out-design you on shape.** Several of his corrections were both
  better and *smaller* than what was proposed — `impedes` on the barrier removed an entire table; the
  obligation/prohibition collapse removed a whole mechanism. When he pushes back, the fault is real.
- **He accepts engine changes.** Do not contort a design to avoid touching the engine.
- **He will not accept validation dressed as capability.**
- **He wants agnosticism enforced ruthlessly.** Any identifier smelling of one example gets rejected.
- **Examples are dangerous and he knows it.** Every example in the design doc carries a hard warning and
  appears in sets of unrelated worlds so no single one is copyable. Keep that discipline.
- **Correcting the agent is expensive for him.** That is why §1 exists.
