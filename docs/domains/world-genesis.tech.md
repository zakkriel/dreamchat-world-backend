# world-genesis · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-10 · World genesis and world creation ·
**Parent bounded context:** World Engine

This file holds how the domain is built — routes, the build's transaction shape, validation, traps.
`world-genesis.product.md` holds what it means; `world-genesis.seams.md` holds what crosses its
boundary.

---

## Routes

```
POST /worlds                    directory row + operating defaults, authors NO entities → playable:false
POST /worlds/interview          one JSON turn: the next question, or nothing left to ask
POST /worlds/genesis            SSE stream of world_genesis_frame/3, ending in a `choice` frame
                                carrying the id of the world it ALREADY committed
POST /worlds/genesis/kickstart  one JSON turn per answer; the LAST answer is the arrival transaction
GET  /worlds                    the directory, via fn_world_directory()
POST /worlds/{id}/refresh       append-only successor world
```

None of the create routes hang off `/worlds/{id}` — there is no world yet, which is the point.

## Where the code lives

| Path | Role |
|---|---|
| `core/api/worldgenesishandler.go` | The three collection-level routes. **Carries `Governed-by: ADR-P021`.** Its header is the best single explanation of the flow in the repo — read it first. |
| `core/api/worldgenesis.go`, `worldgenesiscommit.go` | The build and the commit ladder |
| `core/api/worldinterview.go` | The Custom lane's interview turn |
| `core/api/worldkickstart.go`, `kickstartstate.go` | The character turn and the scenario/arrival turn, resumable |
| `core/api/worldrefresh.go` | Refresh: mints a **new** world from a template and archives the source |
| `core/api/worldshandler.go` | `GET /worlds` (directory), `POST /worlds` |
| `core/api/worldactor.go`, `worldactorprompt.go`, `placeauthor.go` | The seats that author a cast member and a place |
| `core/api/worldturn.go` | Turn plumbing shared with the play loop |
| `core/db/seeds/` | The two seeded worlds (Mara 0A, The Drowned Lantern) |

Prompts: `world_genesis.txt`, `world_interview.txt`, `world_kickstart.txt`, `world_actor.txt`,
`place_author.txt` — five of the nine seats, every one carrying the byte-identical latitude block
(`ADR-P022`).

Contracts: seat leashes `world_genesis.v1`, `world_interview.v1`, `world_kickstart.v1`,
`world_actor.v1`, `place_author.v1`; wire responses `world_genesis_frame.v3`,
`world_interview_turn.v1`, `world_kickstart_turn.v2`, `world_created.v1`, `world_refreshed.v1`,
`world_directory.v2`. Five are vendored by the frontend — see `seams.md`.

## The build is two transactions, not one — and that was a reversal

`build()` commits everything the world *is* before the stream ends; `kickstart()` runs the arrival
transaction later. The earlier design deliberately chose all-or-nothing and production overruled it:
two builds on 2026-08-20/21 both survived 33 s + 63 s of **paid** authoring and then lost the world
because a refused kickstart answer discarded an in-memory draft on a 15-minute TTL. The rule that
came out of it: *"the world is the expensive, correct part; the kickstart is the cheap, flaky part;
the cheap part must not be able to destroy the expensive part"*
(`docs/design/2026-08-21-durable-worlds-design.md`, the amending authority over the PRD).

An empty kickstart answer is the **resume path**, not an error: it re-serves the pending question. A
refused turn is a `422` with the stated reason and the world untouched.

## The understanding pass, in the pipeline

Step 2 of the five (`docs/design/2026-08-26-world-identity-and-the-understanding-pass.md`, restored
by `ADR-P026`). Its emissions and their slots are the design's §3; the fill mechanism it governs is
§7 (rules are the work plan; the code schedules, the model interprets; tagging survives for scoped
retraction — reviews, not gates, §7.3). **Not yet built**; the hand-run probe is the only execution
to date (`docs/design/2026-08-27-understanding-pass-probe/`, PR #126).

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-P020` | The deploy does not run migrations; the service refuses to boot on drift, on purpose. | Releasing without apply-config → apply-migrations → merge → watch-boot causes the outage it caused. |
| `ADR-P021` | Art is reconciled, never commissioned; the kick is detached — the stream has already ended. | A provider outage destroys an authored world instead of delaying pictures. |
| `ADR-P022` | Every seat prompt carries the byte-identical latitude block. | `go test -run Latitude` fails; or worse, it passes and one seat quietly has different latitude. |
| `ADR-P023` | A style's look lives in `artstyle.go` and nowhere else; clients pick by key. | A second copy of a style's look, and the copies drift. |
| `ADR-P024` | Seat config is part of the release — set in the environment BEFORE the merge that needs it. Nothing checks this. | A seat needing `json_object` or a token ceiling silently fails in production. |
| `ADR-P026` | The package + restored design are the governing documents for this domain. | Genesis work falls back to re-deriving settled design. |
| `D-1` + the `fast_path` exception | Canon is written through `apply_event`/`apply_ruled_event`; genesis' `origin='fast_path'` is the ONE documented exception, existing because the actors an event would reference do not exist yet. | A second exception corrupts replay (`I-1`) and provenance (`I-2`). |
| `I-2`, `I-9` | Provenance on every row; invariants are the permanent regression suite. | A red invariant blocks merge, always. |
| `SPEC-028` | World management API landed; `POST /worlds` authors no entities, `playable:false` is honest. | "Fixing" the empty world builds the wrong feature. |

### What you may not decide alone

1. **Adding a seat.** Four files (`system_map.md` §8) — prompt with the latitude block,
   `allSeatNames`, the embedded map in `promptlatitude_test.go`, its `Seat` + capability floor + a
   deterministic fake — and its config in the environment before the merge (`ADR-P024`).
2. **Changing the commit ladder.** The durable-worlds spec is the amending authority, not the PRD.
3. **Moving a published `world_*` schema.** Cross-repo round; the frontend's five artifacts move in
   the same round (`seams.md`).
4. **The fill mechanism and its call budget.** Design appendix Q1, explicitly *"the largest risk to
   this document"*; the multiplier is now measured (11 rules, 3 generative — probe PR #126) but
   per-call cost is not. Founder's decision.
5. **Re-adding any Fast-lane pre-build step.** Founder-ruled out (design §8.1).
6. **The twenty functions list** — whether it is exactly twenty and whether it evolves (design §10).
7. **What genesis does with a genre-reference brief** ("like Dune but underwater") — design Q3,
   unresolved, arrives on day one.

## Validation for this domain

From the area dossier, unconditional lines first:

```bash
make reset && make schema-check
cd core/api && go test ./... -count=1 -run 'Genesis|World|Kickstart|Interview|Latitude'
make schema-contract                    # if a published shape moved
../harness/check.sh contract-drift      # if the frontend consumes it
```

pgTAP: `120_world_template*`, `27_world_directory*`, `28_world_taglines*`,
`101_personality_world*`, `100_spine_seeds*`, `109_drowned_lantern_seed*`.
`core/api/genesisart_test.go` (a built world commissions its own art) is owned by
**art-and-assets**; it runs here because the kick is ours.

**A recorded contradiction, not resolved here:** `perception-and-knowledge.tech.md` states *"`make
reset` destroys the dev volume holding twelve worlds and must never be run"*; this area's gate block
names `make reset` as its first command. Both are quoted faithfully from their sources. Until a
ruling reconciles them, run `make reset` only against a disposable database, never a dev volume you
care about.

**Known red on main (2026-08-27):** `TestRunWorldTurn_Standalone_CallableWithoutBeatLoop` fails
(`firedMag = "", want small (forced)`) on `origin/main` — found by the probe round's gate run, on a
world-genesis-owned path (`worldturn*`). Unfixed as of this package's writing; do not treat it as
your regression, and do fix or report it rather than re-discovering it.

**What counts as evidence here:** the expensive failures are *paid* — a lost world is 30–90 s of
seat spend, and a wrong world is coherent and confident (design §8.1's stated cost). So: reproduce a
creation-path defect with a real build before fixing, and check every refusal that can happen before
the paid seat call actually happens before it (`ResolveArtStyle` runs early for exactly this
reason).

**What counts as ceremony here:** `./stack.sh smoke` is liveness, not correctness — an empty beat
streams the same frame count and passes identically (workspace `AGENTS.md`). And a suite asserting
on the two seeded worlds validates seeds, not genesis: the seeds were authored by hand, so no
genesis logic is exercised by reading them.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The cheap transaction destroying the expensive one.** | §The build is two transactions above — the one home for this fact; two production deaths behind it. |
| **A commissioning call added to a creation path.** Art is a detached kick. | `ADR-P021`; a world shipped with no images. |
| **`fast_path` used anywhere but genesis.** | `D-1`; corrupts replay (`I-1`) and provenance (`I-2`). Not a precedent — the actors-do-not-exist-yet reason applies nowhere else. |
| **A bad art-style key refused late.** `ResolveArtStyle` runs early: omitted is legal, a bad key refuses BEFORE the expensive seat call. | Area dossier §2; moving the check later pays for a build that cannot finish. |
| **Making a model re-derive what the server can compute.** | The kickstart seat had to join `cast[].starts_in` against `places[]` inside a serialized document; the routed flash model failed the join **twice in a row**. The legal set is computable server-side. |
| **An identity that references people into existence, dropped.** "Joe, son of Dalma and Harry" used to silently drop the kin because the kickstart prompt declares the world immutable; the character turn now authors them. | Area dossier §2; if you touch that prompt, know which of the two things it is claiming. |
| **The pass reads a dry register as emotional flatness.** The probe's identity called Octavo's death "no trauma fundacional"; the richer tiers state that death founded the entire institutional order — the dryness IS the trauma response. Single-world evidence. | Probe REPORT Judgement 4 (PR #126). |
| **The pass derives premise-object exclusions and misses world-wide ones.** It ruled the Andantes out as gods and never ruled out magic in the world at large; it also missed "predictions are estimates, never certainties". Single-world evidence; whether it costs anything is the next probe's question (founder ruling 2026-08-27: fill runs on the flawed identity as-is). | Probe REPORT Judgement 1, refusal rows 3/9/13 (PR #126). |

## Open questions

1. **The design's appendix Q1–Q13** — thirteen, homed there, still open
   (`docs/design/2026-08-26-world-identity-and-the-understanding-pass.md` Appendix). The five that
   could change the design: Q1 call budget (multiplier now measured: 11 rules, 3 generative — probe
   PR #126; per-call cost still unmeasured), Q2 Fast-lane identity parity, Q3 genre-reference
   briefs, Q4 when filling stops, Q5 identity inside or beside the document. An agent hitting any of
   them is deciding something new.
2. **`SPEC-036`** — enforcement of a world's own rules. Deferred deliberately; the exist-kind cost
   scales with hours played, so it returns when world creation during gameplay is designed.
3. **The fill probe** — does rule-governed filling yield entailed content? Untested (the probe
   stopped at the identity). Founder ruling 2026-08-27: run it on the probe's identity **as-is,
   misses included**, measuring what a missed protective rule costs before amending the pass.
4. **Two known blockers, recorded at the design's close:** the world-model schema has no machine
   representation (gates step 5 entirely — WE-11's, see `seams.md`), and `prd_world_creation.md`
   still describes `places / cast / objects / ways`, structurally the world-model version that died
   — reconcile before building against either.
5. **B1 — identity/auth.** `POST /worlds` is unauthenticated and SPEC-028 itself says it should be
   the first endpoint auth goes in front of; per-user isolation does not exist (area dossier §3).
