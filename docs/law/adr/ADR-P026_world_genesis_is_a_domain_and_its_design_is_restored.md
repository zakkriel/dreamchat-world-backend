# ADR-P026: World genesis is a domain with its own package, and the understanding-pass design is restored as live law

**Status:** Accepted (2026-08-27)
**Date:** 2026-08-27
**Series:** Platform / Engine (`ADR-P###`, per `D-5`)
**Governing rules:** `D-6` (git docs are the source of truth; the copy is the one that goes stale),
`GA-2`/`GA-3` (the design's own foundation), `workspace:ADR-W006` (consolidation and its lookup rule).
**Owner of decision:** founder ruling in conversation, 2026-08-27 — *"world genesis is a domain on its
own"* — following the understanding-pass probe (PR dreamchat-world-backend#126).

---

## Context

Two facts collided, and the probe made the collision visible.

**1. The stage-1 genesis design had no home.** The consolidation (`workspace:ADR-W006`, commit
`88486c1`) deleted the settled understanding-pass design
(`docs/30_architecture/world_model/03_world_identity_and_the_understanding_pass.md`, settled
2026-08-26, §3.11 corrected in `70840b9` the same day), the twelve testworld briefs, and the depth
PRD, with no successor file. "Understanding pass" appeared nowhere in any working tree. The same PR
(#124) merged the design as settled and deleted it — a sweep, not a retirement ruling.

**2. The domain package that would hold it was unwritten.** `harness2` routes genesis paths by
fall-through to the area dossier; the WE-10 cluster (`digest/01_TOPIC_MAP.md`) named the domain but no
package existed, so nothing owned the design's decisions and nothing could cite them.

The probe (PR #126) had to recover its inputs from git history as bannered evidence and reported the
gap as a decision only the founder could make. `ADR-W006`'s lookup rule — *"cited by something live,
or it stays out"* — meant the design stayed out until something live cited it.

## Decision

1. **World genesis is a domain of its own.** Its package is
   `docs/domains/world-genesis.{product,tech,seams}.md` (cluster WE-10 · World genesis and world
   creation, parent bounded context World Engine), registered in `harness2/DOMAINS.map`. Genesis
   work loads the package first; where the package and the area dossier disagree, the package wins
   and the dossier line is deleted in the same round (`harness2/START_HERE.md` §4, `D-6`).
2. **The understanding-pass design is live law again.** Restored verbatim from `70840b9` to
   `docs/design/2026-08-26-world-identity-and-the-understanding-pass.md` (one provenance line added,
   nothing else), cited by the package's decisions tables. This closes `ADR-W006`'s loop: the
   package and this ADR are the live citers.
3. **The twelve testworld briefs stay in history.** They are test fixtures, not law. A round that
   needs one recovers it as bannered evidence, exactly as the probe did (`ADR-W006`'s amendment,
   2026-08-27). Nothing live cites them standing, so they stay out.

## Evidence

- The probe: `docs/design/2026-08-27-understanding-pass-probe/` (PR #126) — the design's
  understanding pass, hand-run on a never-encoded brief, derived 10 of 13 held-back negative-canon
  entries, produced an identity whose twenty answers were judged world-specific in 17 of 20 cases by
  an isolated judge, and emitted 11 rules (3 generative), replacing the "roughly nine" estimate in
  the design's own Q1.
- The founder's companion ruling, same conversation: the fill-mechanism probe runs next **on the
  probe's identity as-is, misses included** — measuring what a missed protective rule costs before
  any design amendment. Recorded in the package's open questions.

## Consequences

- Genesis design decisions are citable: the package cites the design; the design resolves at a live
  path. The probe-style recovery dance is no longer needed for the design itself.
- `docs/areas/world-genesis.md` remains the area-level view (review routing, boundaries), corrected
  where stale; the package is the governing document for domain work.
- The design's appendix (Q1–Q13) remains open questions, now homed in the package's §7 — an agent
  hitting one is deciding something new and must say so.
- Superseding any part of the restored design is a new ADR, not an edit to the restored file.
