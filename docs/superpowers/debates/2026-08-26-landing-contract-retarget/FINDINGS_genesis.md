# Findings — world-genesis seat

**Scope:** Q2 (§3.2's retirement of §5), the genesis-side coverage-bug evidence, and provenance
checkability. Every finding below is `block`, `gate`, or `accept-with-reason`.

**Opening move, reported honestly:** the role brief's mutation opener ("revert the change, does a test
fail") has no target here — this round amends a PRD and a schema design record, ships no diff, and adds
no test. There is nothing to mutate. I do not fabricate a mutation table; I say so and move to the
document evidence, which is the only evidence this round produces.

---

## Q2 — does retiring §5 lose the cheap proof?

### Finding 1 — `block`: three of the four substitutes do have a cross-concept grounding problem; §3.2 is wrong as written

`prd_world_creation_depth.md:232` states the reason §5's original three were chosen: "none has a
cross-concept grounding problem, so they prove the contract cheaply." `PROPOSAL.md:75-79` substitutes
`integrity`, `latency_class`, `reliability_class`, `excluded[]` and asserts (`PROPOSAL.md:80`) "those four
are one declaration each if the contract holds." That assertion is false for three of the four, checked
against the contract's own text and against measured evidence, not against opinion:

- **`integrity`** — `SCHEMA-v3.md:107`, listed under "Matter" beside `bulk_class`/`capacity_class`
  (`:105-108`), a single facet key on one `matter` entity, self-contained exactly like the original §5
  three. **No cross-concept problem.** Confirmed by usage: `G_sueno_by_extraction.md:59,94` sets it as a
  bare per-entity value (`"integrity": "sound"`) with no reference to any other concept.

- **`latency_class`** — `SCHEMA-v3.md:118`, under "Knowing": "the delay before a fact is knowable **to a
  receiver**." Its own definition names a second party. Measured, not asserted: `R_score_grelda.md:96`
  scores it "same magnitude" but flags "**ladder unwritten**" — the class ladder is not fixed grammar,
  so three blind builders agreeing was calibration luck, not a converged contract. `R_score_grelda.md:424-459`
  documents the actual open gap this key sits inside: no reader obligation yet computes whether a
  receiver receives a channel's content at all (a `receipt(channel, emitter, receiver, t)` obligation,
  explicitly unbuilt), and that same passage (`:456-457`) states plainly that `path`/`indicator`/
  `history.knowledge`-family rows — the family `latency_class` and `reliability_class` both live in —
  "already presuppose that a receiver received." Landing `latency_class` alone, with no `channel`,
  `history`, and receiving-`entity` concepts landed first, means declaring a reader for a presupposition
  the contract has not yet built.

- **`reliability_class`** — `SCHEMA-v3.md:123`, under "Knowing," on `indicator`, and by `O9`
  (`SCHEMA-v3.md:62`) an indicator is obligated to "name a hidden state **some accumulator or property
  actually holds**" — the obligation's own text requires a second, already-landed concept to ground
  against. Measured, and worse than `latency_class`: `R_score_grelda.md:101` scores it **"contradictory,"
  "obligation ambiguous,"** with `poor` meaning "60% correct" in one builder's read and "30% wrong"
  (roughly the opposite number) in another's — `R_score_grelda.md:268-271` calls this a polarity sign
  error the aggregate score hid. This is the single worst-scoring row in the entire reader-obligation
  test. It is not a cheap first customer; it is the contract's most contested one.

- **`excluded[]`** — top-level section (`00_world_model_and_genesis_pipeline.md:274`), and its *reader*
  side scores clean (`R_score_grelda.md:103`, "identical," all three builders refused the same way). But
  its *refusal* side, `R6` (`SCHEMA-v3.md:77`, "`excluded[]` entry contradicted by an authored entity —
  the document disagrees with itself"), requires checking every excluded entry against **every other
  authored entity in the document** — the most cross-concept obligation of the four by construction, since
  it must see the output of every other landing to be checked at all. Landing it as "one declaration"
  either defers `R6` (leaving the refusal unimplemented, silently) or requires the `Refuse` resolver to
  read across every other concept's minted state — which is exactly the risk `prd_world_creation_depth.md:269-272`
  already names and leaves open as Open Question 1 ("does the resolver become the new god object").
  Landing `excluded[]` first is choosing to answer that open question by accident, under cover of a
  "cheap" first customer.

**Verdict:** the substitution does not prove what §5 proved. §5's three were chosen precisely to
isolate the landing-contract *mechanism* (Declare/Parse/Apply/Refuse, R1 reader-sum-type, R2 leaf
coverage) from grounding difficulty, so that "the contract holds" could be tested without also testing
whether the resolver can reach across concepts. Landing `latency_class`, `reliability_class`, and
`excluded[]` first tests the mechanism *entangled with* three separate, already-documented open problems:
the receipt/composition gap (`R_score_grelda.md:424-459`), a measured-contradictory reader obligation
(`R_score_grelda.md:101`), and the resolver god-object risk (`prd_world_creation_depth.md:269-272`). A
failure under that combined load will not say which of the four problems failed. That is a materially
different, harder test than §5 ran, dressed in §5's language ("one declaration each," "prove the contract
cheaply").

**Disposition:** `block`. Overriding needs a written reason in the PR body addressing why the resolver
god-object question (§9 Q1) is being answered implicitly by this ordering choice.

### Finding 2 — the right first customer, on the proposal's own criterion

Only `integrity` clears the bar §5 actually used (no cross-concept grounding problem) **and** the bar
`PROPOSAL.md:71-73` adds (absent from the engine, used by every test world). From the audit's world-usage
scan, row 8 is ABSENT (`01_engine_capability_audit.md:40`) and scored `y / y / y` across all three test
documents (`01_engine_capability_audit.md:127`) — the same universality claim the proposal makes for all
four, but true for only this one without qualification. **Recommendation: retarget the depth-as-first-
customer set to `integrity` alone.** Land `latency_class`, `reliability_class`, and `excluded[]` after
their prerequisites exist: the receipt/composition mechanism (roadmap increment 4,
`2026-08-26-world-model-eight-increments.md:154-171`) for the first two, and a resolver capable of
reading every landed concept (or the document validator absorbing `R6` before landing time) for the
third.

**Disposition:** `gate` — mechanizable: change `PROPOSAL.md §3.2`'s substitute list to `integrity` only,
and note the other three as sequenced, not retired, first customers.

---

## Genesis coverage bug — verified independently

`01_engine_capability_audit.md:154-166` claims `mundo-08-sueno-comun-1-basico.md` states content that
`G_sueno_by_extraction.md` never authored, one layer earlier than "authored but inert." Checked directly
against both files, not taken on the audit's word:

- **Rule 6** (`mundo-08-sueno-comun-1-basico.md:28`): "El que no duerme cuatro noches seguidas empieza a
  soñar despierto, en público, y lo ve el barrio" — a numbered threshold (four nights) with a public
  consequence. `G_sueno_by_extraction.md`'s `law[]` array (lines 33-47) has five entries covering rules 1,
  2, 4, 10, and an inferred marriage rule from 4+8 — **rule 6 is not among them**, and there is no
  `accumulators[]` key anywhere in the emitted JSON (lines 8-127). The extraction's own §5
  (`G_sueno_by_extraction.md:189-192`) confirms this was not an oversight but a considered cut: "I
  considered one [accumulator for nights-without-sleep → threshold] and cut it... I left it out." Vira
  Cor, named in the brief at eleven nights specifically to demonstrate rule 6
  (`mundo-08-sueno-comun-1-basico.md:47`), is carried in the extraction only as prose color
  (`G_sueno_by_extraction.md:74-76`, "an insomniac eleven nights in, and it shows" / "make it through one
  more day without projecting in public") with no mechanism behind it. **Confirmed: stated, demonstrated
  by name in the brief, never authored.**
- **Daily cycle / opening hours**: the brief states "todas las noches" (every night, rule 1, captured),
  "de nueve a cuatro" for the archive (`mundo-08-sueno-comun-1-basico.md:41`) and "pública desde las
  nueve" for the transcription (`:64`). The extraction captures only "since nine" as one record's
  `access.cadence` (`G_sueno_by_extraction.md:91`) — the closing hour ("cuatro") is dropped entirely, and
  there is no `cycles[]` key anywhere in the document. **Confirmed: partially stated, partially authored,
  the numbered half silently missing.**
- **The record that is true about a dream and false as an accusation**: `record.asserts[].accurate:
  false` is a named reader obligation (`SCHEMA-v3.md:125`) for exactly this shape. The extraction's one
  record entity, `last night's volume, the Twelve` (`G_sueno_by_extraction.md:89-91`), has `asserts` as a
  bare string array with no `accurate` field at all — the mechanism built for the document's central fact
  (the transcription is accurate about what was dreamed, false as an accusation of murder) is never
  invoked. `R_score_grelda.md:103` independently confirms this obligation went unexercised across the
  whole reader-test round too ("unexercised: the one record asserts `accurate: true`").

All three misses coexist with a self-check (`G_sueno_by_extraction.md:129-152`) that reports every S1-S6,
O1-O11, and R1-R13 check passing. That is expected, not a contradiction: those checks test internal
document consistency (names resolve, obligations are present where facets are declared, refusal
conditions are absent) — none of them compares the document against the brief's prose. **The audit's
characterization is verified: this is a real defect, one layer earlier than "authored but inert," and
undetected by every check currently defined.**

### Does it belong in this increment?

`docs/superpowers/plans/2026-08-26-world-model-eight-increments.md:102` — increment 1's own "Ships" line
already names "the brief-to-document coverage check" as a deliverable of this increment, and its "Proof"
line (`:112`) already requires "a pipeline-created world passes the validator and the coverage check
first try." This is not a new scope proposal — it is an already-closed roadmap commitment, in a file
`PROPOSAL.md`'s own reading list requires reading (`PROPOSAL.md:130`). It is unlike the three items
`prd_world_creation_depth.md §10` expels (`:283-292`): those are defects in *other* surfaces (narrator
rendering, interview error handling, a dormant SQL filter) explicitly outside the landing contract's own
deliverable. The coverage bug is a defect in genesis's own core deliverable — the document the landing
contract consumes — and the roadmap has already placed its fix inside increment 1, the increment this
proposal is amending.

`PROPOSAL.md §3.4` ("Two additions the audit forces") names the document validator and the
narrator-reaches-the-world item as exhaustive, and does not mention the coverage check, despite citing
the same audit file whose closing section (`01_engine_capability_audit.md:154-166`) is exactly this
finding, and despite the increments roadmap (also in the reading list) already committing to it. This is
an omission in the amendment text, not a disagreement about scope.

**Disposition:** `gate` — real, mechanizable ("mechanical and cheap," `01_engine_capability_audit.md:166`),
already scoped to increment 1 by prior decision. Add it to `PROPOSAL.md §3.4` as a named third addition
(or an explicit cross-reference to `2026-08-26-world-model-eight-increments.md:102,112`) so the round does
not silently drop a commitment its own reading list already made.

---

## Provenance — checkable at registration, at emit, or only by a human?

`world_model/6`'s provenance requirement (`SCHEMA-v4.md:34-39`) obliges `"source": "stated"` or
`"source": { "inferred_from": [...] }` on every element, and `R13` (`SCHEMA-v4.md:47,118`) refuses a
chain that does not terminate in stated content. Two genuinely different things are bundled under
"provenance," and they are checkable at different times by different means:

1. **Structural termination — mechanizable, at emit (the document validator), not at landing
   registration.** Walking every element's `inferred_from` list to confirm it resolves to another named
   element in the same document, recursively, until a `"stated"` leaf is reached with no cycle and no
   dangling reference, is a static graph-termination check — the same *shape* as `R1`'s unresolved-name
   check (`SCHEMA-v3.md:72`). It needs the document only, not the target engine schema, so it belongs to
   the per-document validator `PROPOSAL.md:91-93` already calls out as "different gate, different time"
   from `R2` — it does **not** belong to landing's registration-time check, since registration
   (`prd_world_creation_depth.md:119-123`) is a set-difference over schema × declarations and has no
   notion of one document element citing another.

2. **Semantic honesty — only a human, and demonstrably not caught otherwise.** Whether a `"stated"` tag
   actually matches something in the brief, and whether an `inferred_from` citation is a *sound*
   inference rather than a plausible-looking pointer, cannot be checked structurally. This is not a
   theoretical gap: `G_sueno_by_extraction.md`'s own self-check (`:150-152`) reports "R13: every
   `inferred_from` chain bottoms in a quoted line from the brief... satisfying R13 by construction" —
   and the same document simultaneously drops rule 6, the archive's closing hour, and the
   `accurate`-field mechanism entirely (see the coverage-bug section above). A structural R13 pass proves
   only that the citations present are well-formed; it says nothing about content that was never given a
   `source` tag because it was never authored at all, and nothing about whether a `"stated"` claim is
   honest. Only the human audit that produced `01_engine_capability_audit.md` caught that gap, and it
   caught it by reading the brief against the document directly, not by re-running R13.

**Disposition:** `accept-with-reason`. The structural half is real and mechanizable — `gate`: fold it into
the document validator's 13 refusal rules (it is one of them, `R13`, already counted in `PROPOSAL.md:91`
"v6 has 13 refusal rules"). The semantic half is a standing, human-only check with no proposed
mechanization in this round or the roadmap, and no evidence anyone believes it is mechanizable — that
limitation should be stated in the PRD rather than implied to be covered by "the document validator," since
the coverage-bug evidence above shows a document can pass every mechanical check R13 offers while still
failing provenance in the sense that matters (the tag is honest, the content behind it exists).
