# Gate evidence — the four plan-review gates, discharged with output shown

## Gate 3 — verbatim restoration (D-6)

`diff <(git show 70840b9:docs/30_architecture/world_model/03_world_identity_and_the_understanding_pass.md) docs/design/2026-08-26-world-identity-and-the-understanding-pass.md`:

```
5a6
> **Restored** 2026-08-27 verbatim from git `70840b9` by `ADR-P026`, after the `workspace:ADR-W006` consolidation (`88486c1`) deleted it with no successor.
```

Exactly the one restoration line; nothing else differs.

## Gate 4 — every DOMAINS.map glob matches (per-glob counts vs `git ls-files`)

| Glob | Matches |
|---|---|
| `core/api/kickstartstate*` | 1 |
| `core/api/placeauthor*` | 2 |
| `core/api/prompts/place_author.txt` | 1 |
| `core/api/prompts/world_actor.txt` | 1 |
| `core/api/prompts/world_genesis.txt` | 1 |
| `core/api/prompts/world_interview.txt` | 1 |
| `core/api/prompts/world_kickstart.txt` | 1 |
| `core/api/schema/place_author*` | 1 |
| `core/api/schema/world_actor*` | 1 |
| `core/api/schema/world_created*` | 1 |
| `core/api/schema/world_directory*` | 1 |
| `core/api/schema/world_genesis*` | 2 |
| `core/api/schema/world_interview*` | 2 |
| `core/api/schema/world_kickstart*` | 2 |
| `core/api/schema/world_refreshed*` | 1 |
| `core/api/worldactor*` | 3 |
| `core/api/worldgenesis*` | 6 |
| `core/api/worldinterview*` | 1 |
| `core/api/worldkickstart*` | 2 |
| `core/api/worldrefresh*` | 2 |
| `core/api/worldshandler*` | 2 |
| `core/api/worldturn*` | 2 |
| `core/db/seeds/*` | 2 |
| `core/db/tests/100_spine_seeds*` | 1 |
| `core/db/tests/101_personality_world*` | 1 |
| `core/db/tests/109_drowned_lantern_seed*` | 1 |
| `core/db/tests/120_world_template*` | 1 |
| `core/db/tests/27_world_directory*` | 1 |
| `core/db/tests/28_world_taglines*` | 1 |
| `docs/design/prd_world_creation.md` | 1 |
| `docs/design/2026-08-20-kickstart-arrival-choice-design.md` | 1 |
| `docs/design/2026-08-21-durable-worlds-design.md` | 1 |
| `docs/design/2026-08-26-world-identity-and-the-understanding-pass.md` | 1 |

Zero-match globs: 0. (The depth-PRD glob the dossier implied was checked and found deleted by
`88486c1`; it was never added — the dossier row was corrected instead.)

## Gate 5 — the routing guard, watched going red then green

BEFORE the DOMAINS.map entry (recorded at round open):

```
 No domain package owns this path yet. Naming the domain is part of your plan — do not
 guess one. Falling through to the area-level brief:
 [...]
 AREA  world-genesis
```

AFTER the entry:

```
╔══════════════════════════════════════════════════════════════════════════╗
║  DOMAIN: world-genesis                                                   ║
║  CONTEXT: World Engine                                                   ║
╚══════════════════════════════════════════════════════════════════════════╝
   ✓ dreamchat-world-backend/docs/domains/world-genesis.product.md
   ✓ dreamchat-world-backend/docs/domains/world-genesis.tech.md
   ✓ dreamchat-world-backend/docs/domains/world-genesis.seams.md

 Read the files your task touches, before you write a line. A technical change reads
 tech and
```

## Gate 6 — docs-only diff confirmation

Backend paths touched (from `git status` at commit time): `docs/design/`, `docs/domains/`,
`docs/law/adr/` only — no `core/` path, so the dossier's unconditional gate block re-proves an
unchanged baseline and is skipped per the gate's own condition. Workspace paths:
`harness2/DOMAINS.map`, `docs/areas/world-genesis.md`, the friction journal. Verified again at the
diff checkpoint against the actual diff.
