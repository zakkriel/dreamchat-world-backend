# Veto check — gamedesign

**Vetoes honored.** V1 (player must not arrive knowing the law): §3:99-107 + AC-3:160-161 + signal
:68. V2 (no hierarchy/social-structure identifier): §3:86-92, `standing_over` cut, :248 "never
'under/above'". V3 (wire-or-delete first): AC-1:130-135, blocking. Unanimous cuts all present.
AC-2's correction of my naming-wall claim is verified correct (`fn_display_name` falls through to
`canonical_name`, `schema.sql:1445-1453`; `:2937` therefore excludes groups) — my G5 was wrong.

**Defects (3):**

1. **Misattributed citation (mine, round 2).** AC-3:157-158 — *"Those holders are exactly what
   `fn_private_records` already feeds cognition (`schema.sql:2734-2740`)"*. That range is inside
   `fn_public_moment` (begins `:2732`); `fn_private_records` is `:2677-2731`. §3:101-102 pairs both
   ranges correctly, so this is a copy error in the load-bearing AC.

2. **The delivery claim fails for batch-seated minds.** AC-3:158-159 — *"the law reaches minds with
   zero new read paths"* — and AC-6:177 — *"bound norms arrive as the private records they now are"*.
   Private records render only for the **isolated** seat (`cognitionprompt.go:153-160`); a batch seat
   sees only `fn_public_moment`, whose `HAVING count(DISTINCT holder_id) = cardinality(p_present)`
   (`schema.sql:2693`) requires *every* present NPC to hold the row (`npcs` passed at
   `orchestrator.go:853`). A norm bound to a **subset** — the asymmetric case this PRD exists for —
   reaches no batch mind. Either the prompt renders norms explicitly, or AC-6's A/B kill-switch fires
   for the wrong reason.

3. **Structure-free failure: AC-4 has no reachable vacuity.** AC-10:213-214 mandates *"at least one
   `history[]` entry's `knowledge` records each norm enforced or broken"*, so AC-4:166's *"Vacuous
   when no breach is authored"* is unreachable whenever a norm exists. AC-4 then requires *"≥2
   holders"* while `cast` `minItems` is 1 (`world_genesis.v1.schema.json:112`): a legal one-actor
   world with a norm cannot satisfy it.
