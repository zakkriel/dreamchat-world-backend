# Round 2 — seat: ux (Creation UX Expert)

Peer citations were spot-verified against the repo before use; every checked claim held:
extraction's "zero hits for `relationship_state` inside `core/api`" (re-run: zero matches),
simarch's `schema.sql:3080-3086`/`:3699-3710`/`:2584-2599` ranges, and gamedesign's
`narrateprompt.go:227-228` (the causeless "NOTHING RESOLVED" line, verbatim),
`fn_unearned_names` no-kind-filter (`schema.sql:2924-2939`), the kind-parameterised
`fn_compendium_index` (`schema.sql:1296-1304`), `arrival.why` never persisted
(`worldgenesishandler.go:534,661`; `world_character` insert at `worldgenesiscommit.go:200-204`),
and "`pending_event` is written by nothing but tests" (exactly three inserts repo-wide, all
tests: `ledger_test.go:28`, `orchestrator_worldtime_test.go:348`,
`101_personality_world_test.sql:58`). Note: gamedesign **re-filed a revised Round 1** after first
publication; all quotes and line numbers below cite the current revision. This was a debate of
unusually honest papers — the attacks below are structural, not citational.

## 1. Sharpest attack on each peer

### simarch — your correction seam is painted on the wall

You answered my anticipated attack with: *"the choice happens **before** commit, in the existing
`choice` frame shape (`world_genesis_frame.v3.schema.json:79-89`), or it does not happen. That is
ux's problem to solve; I will veto any post-commit mutation path"* (`round1_simarch.md:128-130`).

Factually wrong under the shipped flow you cite elsewhere in the same paper: the `choice` frame is
the terminal frame of a stream whose world has **already committed** — *"the world commits when
authoring ends (durable-worlds, 2026-08-21): the world already exists… only the player and the
arrival wait"* (`world_genesis_frame.v3.schema.json:4`; decisions 1–2,
`2026-08-21-durable-worlds-design.md:30-41`). By the time your seam renders, every order and norm
you propose is immutable canon. Your veto plus your timing error together entail exactly one
surviving correction surface: **pre-genesis**. You handed me the problem while pointing at a door
that does not open. I accept the hand-off; the list below makes the pre-genesis playback
veto-grade. Secondary, unanswered: your M3(b) renders membership *"as that viewer perceives it"*
(`round1_simarch.md:68-69`) but your `orders[]` shape has no authored input for whether membership
is *legible* — marked on sight versus concealed — so your implementation would pick that epistemic
policy silently. That bit drives roster lines, disguise and mobility (`BRIEFING.md:19`); it must
be authored and confirmable.

### extraction — your deferral is the `standing` failure with better paperwork

You concede your own storage target is *"prose wearing a column"* (`round1_extraction.md:92-93`)
and then defer the reader: *"a fact that can \[be joined on\], can be wired in later without a
genesis schema version bump"* (`:109-110`). "Committed but unread, wired later" is the exact
pattern all four papers indicted in `cast[].standing`: authored
(`world_genesis.v1.schema.json:130-134`), validated (`worldgenesis.go:310-311`), dead on arrival.
Your M3 reproduces it one table over — `relationship_state` is read by zero lines of `core/api`
(verified), and the context spec's promised `[RELATIONSHIPS]` block
(`06_context_assembly_spec.md:76,88`) is unrendered (gamedesign's independent finding,
`round1_gamedesign.md:144-149`). Meanwhile simarch's group-held-perception channel is **already
read** by `fn_visible_perceptions` (`schema.sql:3080-3086`) and reaches minds through the
cognition prompt. Under my gate — shown ⇒ committed ⇒ consumed — no confirmation statement may be
built over `relations[]` at launch, which means the user cannot see or correct the very facts your
schema exists to hold. Storage that cannot be surfaced loses to storage that can. Secondary,
unanswered: your M5's *"Don't touch this seat"* (`:67`) defends an interview that can silently not
run — seat errors collapse to `Done: true` indistinguishably (`worldinterview.go:71-84`) — so
"the existing instinct already covers it" is unfalsifiable by construction, and your M8 harness
measures genesis, never interview coverage.

### gamedesign — G4's "asks nothing" is its bug as much as its feature, and your revision made it bigger

*"G4 asks nothing — the near future is inferred in the same call"*
(`round1_gamedesign.md:153-154`). You present that as the virtue; it is the vulnerability. Your
revision widened G4 from a norm demonstration to a universal: *"**one authored thing about to
happen, in every world**"* with the acceptance *"in every world… the player perceives one
authored future event they did not cause"* (`:105-112`). So now **every** build plants an
append-only near-future the user never sees before spend — the single highest-consequence
inference in your mechanism set, since it decides what the player watches happen in their first
beats. A seat that plants "the deference demanded, publicly, in the square" against a user who
meant a quieter cruelty has authored the opening spectacle of a world the user did not mean,
unfixably. That is precisely the silent-inference class my Round 1 exists to kill — and you
conceded the principle **in your own answer to me**: *"a user who cannot see the inference cannot
correct it. A review screen, not another question"* (`:157-158`). Your concession indicts your
mechanism: the review screen you grant me must show the future you author, or it reviews a
constitution while the seat schedules an event behind it. The fix costs one line — the authored
future renders in the playback as a strikeable statement naming the *practice*, never the scene
("Soon: the tithe falls due"). Secondary, unanswered: your G5 makes the collective's descriptor
*"load-bearing as a first sight"* (`:122`) — true only for **marked** membership; for a concealed
order an honest first-sight group descriptor is a contradiction, leaking exactly what the fiction
says strangers cannot see. Nothing in your revision distinguishes grey sashes from secret
societies. And your ordering overclaim — *"Without G2, nothing else here is worth building"*
(`:88-89`) — is defeated by your own G3: depth delivered through cognition produces narratable
NPC acts, not gate rejections; the unexplained-refusal failure fires only on the physical-access
path. G2 is real; it gates the access surface, not the PRD.

## 2. Concessions

1. **To simarch — the storage and consumption design, entire.** Orders as `entity_kind='group'`
   registry rows (zero DDL, `schema.sql:3699-3710`), norms as group-held `public` perceptions
   grounded in backstory events (never `world_genesis` — `fn_perceived_name` would read them as
   names, `schema.sql:2584-2599`), cognition rendering on the stable cache prefix, and the M6
   invariant keeping the player's arrival perception count at exactly 1. This satisfies my gate
   end-to-end; my playback statements trace to `orders[]`/`norms[]`, not `relations[]`.
2. **To extraction — the interview's shape and the harness.** No dedicated "social structure"
   question type: conceded fully; my "enforcement question" is what "ask what changes the world
   most" (`world_interview.txt:3`) already selects when a norm is implied and open. Your line
   *"it needs somewhere to put the answer"* (`round1_extraction.md:66`) is the correct ordering
   principle for the whole feature. Your M8 plant-and-measure design is the right quality
   instrument — I want playback traceability added to it, not a rival harness.
3. **To gamedesign — §0, and it catches me too.** Your revision's disqualifier — *"an acceptance
   criterion only a hierarchical world can satisfy is itself a GA-2/GA-3 violation"*
   (`round1_gamedesign.md:8-11`) — is correct, you applied it to your own v1 criteria (`:166-169`),
   and it applies to mine: my earlier list said "the two lowest-standing cast members" and "in a
   caste world," which teaches hierarchy through the test suite exactly as you describe. Every
   criterion in §3 below is rephrased over "the authored norm, whatever it is," with vacuous-when-
   unimplied semantics and G6's negative control adopted. I also concede G0 wholesale
   (wire-or-delete generalized across `standing`, `arrival.why`, `places[].kind` — your findings,
   verified), G6's unlike-shapes fixture rule ("if the fixtures are all hierarchies, the harness
   is the ontology," `:130-131`), and the felt-surface frame: consumption means a surface the
   player can feel, and your `tension` worked example is the standard.
4. **Against myself.** (a) My interview honesty state (distinguish "nothing to ask" from "could
   not ask") is real but minor — below the line. (b) My Round 1 flirtation with showing
   enforcement history in playback was wrong: enforcement beats are plot and stay behind the AC-7
   secrecy wall; the playback shows the constitution — collectives, norms, the authored practice
   about to recur — never scenes, secrets or history. (c) My Round 1 assumed the playback replaces
   interview questions; extraction is right that it complements them: statements are entailments
   to strike, questions are genuine opens to answer.

## 3. Final converged recommendations (ranked; vetoes marked)

Ranked by leverage; each with a one-line acceptance criterion, phrased per gamedesign §0:
structure-free, vacuous — not failed — when the brief implies no norm or collective. **[VETO]**
marks a condition I consider disqualifying for any proposal that violates it.

1. **Wire-or-delete (gamedesign G0). [VETO on any new authored field without a consumer.]**
   Every non-numeric `world_genesis/1` leaf reaches a table read by a prompt or payload, or
   leaves the schema — `standing`, `arrival.why`, `places[].kind` included.
   *AC: a CI test asserts the property for every leaf of `world_genesis/1`.*
2. **Pre-genesis playback (ux M1; conceded by all three peers). [VETO on shipping inferred depth
   without it — post-commit correction is impossible by simarch's own veto plus durable-worlds
   timing.]** Every high-consequence inference — collectives, norms, the authored near future —
   renders as strikeable world-language statements; amendments travel as `InterviewAnswer` rows
   into the ANSWERS block; "Build now" always live; constitution only, never plot.
   *AC: amendments reach the genesis prompt verbatim; struck content appears in no group, norm or
   pending event; fast lane with defaults accepted remains one tap; a brief implying nothing
   renders no statements and skips the screen.*
3. **Optional `orders[]`-style collectives + `cast[].belongs_to` + authored `legibility`,
   committed as `entity_kind='group'` registry rows** (simarch M1 ⊇ extraction M1/M2, amended per
   my legibility attack); `minItems: 0`; zero collectives is the normal case and costs nothing.
   *AC: a brief implying a collective yields ≥1 group registry row carrying a legibility value; a
   brief implying none yields zero and a byte-identical pipeline.*
4. **Norms as group-held `public` perceptions off a backstory event, plus per-holder variance in
   `history[].knowledge`** (simarch M2 + gamedesign G1, structure-free form). **[VETO on
   `relationship_state` writes until the `[RELATIONSHIPS]` block is rendered — no unread
   storage.]**
   *AC: at arrival `fn_visible_perceptions(world, player)` returns every authored norm while the
   player's own perception count is exactly 1; where a norm is authored, two holders' beliefs
   about it differ in content and `epistemic_type`, no holder holds `direct` for what they were
   only told, and the check is vacuous when no norm is implied.*
5. **Cognition renders each decided-for mind's collective and bearing norms, with the A/B
   kill-switch** (simarch M3, his own falsifiability clause).
   *AC: one norm-implying brief built with and without the mechanism produces differing NPC
   decisions within five beats, or the mechanism is cut.*
6. **Legibility gates both surfaces** (my attack on simarch M3(b) and gamedesign G5): roster
   lines render membership only as the viewer perceives it, and a concealed collective's
   descriptor never renders as a first sight.
   *AC: marked-membership and concealed-membership variants of one brief produce differing roster
   lines, differing playback statements, and no first-sight group descriptor in the concealed
   case.*
7. **Belt refusals for authored law** (extraction M4 + simarch M5): references resolve,
   `standing_over` acyclic, every norm binds a collective with a member reachable from
   `arrival.place`, ≥1 `history[]` beat records the norm enforced or broken — all vacuous when
   nothing is authored.
   *AC: each malformed-document class refuses with a stated reason and has a captured
   fake-driver payload validated in CI.*
8. **One authored near future per world** (gamedesign G4 as revised — an ongoing-world mechanism,
   not a norm mechanism — amended per my attack): genesis becomes `pending_event`'s first
   production writer, `when` as a class, payload by canonical name, **and the authored future is
   a strikeable playback statement naming the practice, never the scene**.
   *AC: in every world the player perceives one authored future event they did not cause within
   the 5-beat window, and that future appeared in the playback before spend.*
9. **Refusal states its cause** (gamedesign G2, re-scoped per my attack): the NOTHING RESOLVED
   render carries the deterministic obstacle already computed, in the fact vocabulary the
   narrator answers questions from (`narrate.txt:33`); a dependency of the access surface (locked
   ways whose keys authored people carry), not of the PRD.
   *AC: a move at a locked way produces narration naming the obstruction; zero new seats.*
10. **Two-layer eval with unlike shapes and a negative control** (extraction M8 + gamedesign G6
    as revised): N=20 plant-and-measure across briefs whose implied norms share no shape — an
    order, a debt, a forbidden place, a duty rota, a non-human protocol — plus a control implying
    no norm (0% invented collectives, <10% missed structure, every playback statement traces to
    brief or amendment, control authors nothing extra); plus a scripted 5-beat run auditing one
    NPC act consistent with the authored norm, one stated-cause refusal, one compendium item, one
    authored future the player did not cause — sampled and human-audited per I-6's methodology,
    behaviour targets never CI equalities.
    *AC: both layers run (fake + sampled live) and their targets are met before depth is declared
    shipped; no fixture set is single-shape.*

**Below the line, deliberately ranked out (not forgotten):** the collective's compendium page
(gamedesign G5 — one handler registration beside `main.go:45-50`, valuable, sequencing-safe to
defer; its naming-wall consequence arrives free with rec 3 since `fn_unearned_names` has no kind
filter); kickstart candidates as positions relative to authored collectives (my M5 — prompt
emphasis, folded into rec 10's audit); the interview error-state distinction (my M4 — one boolean
of honesty, do it when the seat is next touched). **Agreed cuts, four seats:** no relation-kind or
group-kind enums, no social-structure identifier in any code/prompt/schema/fixture name, no
numeric authority or Tier-1 growth, no engine-side norm enforcement or viewer-aware
`fn_portal_permits`, no threshold-accumulation mechanisms, no new tables, no second authoring
seat, no rules codex panel, no post-commit mutation of any kind.
