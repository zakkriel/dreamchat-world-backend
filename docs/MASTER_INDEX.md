# Backend docs — the map

**Consolidated 2026-08-27** (`workspace:ADR-W006`). Everything that used to sit beside these files —
debates, superseded plans, handovers, old PRDs and architecture drafts — was replaced by the
workspace's `digest/` and quarantined in `TODELETE_IGNORE/` pending deletion. If a document is not
in this tree, read the digests, not the quarantine.

Start with `./harness2/domain.sh <file>` from the workspace root — it names the domain package that
governs your change.

| Path | Holds | Cited as |
|---|---|---|
| `law/06_rules_register.md` | the numbered rules, seven parts. The status column is load-bearing. | `B-* C-* D-* E-* F-* GA-* I-*` |
| `law/05_glossary_ubiquitous_language.md` | the signed-off ubiquitous language — deviations are bugs | terms |
| `law/02_world_state_adrs.md` | the frozen engine ADRs | `ADR-###` |
| `law/adr/` | platform ADRs + the governance-pack schemas | `ADR-P###` |
| `law/rulings/` | founder-locked doctrine: `FINAL-*` and dated `RULINGS-*` | quoted by code and migrations |
| `open-spec-items.md` | the spec ledger — open, landed, and designed work | `SPEC-###` |
| `maps/AREAS.map` | path → area ownership (gated by `harness/check.sh areas`) | — |
| `maps/system_map.md` | the backend-internal seams. Read before touching the backend. | — |
| `domains/` | the domain packages (`<domain>.product.md` / `.tech.md` / `.seams.md`) | routed by `harness2/DOMAINS.map` |
| `design/` | design docs still cited by the ledger, code, or CI | by path |
| `runbooks/` | live operations (`full-stack-from-zero.md` is what `stack.sh` mechanises) | by path |

Resolvers that depend on these paths: `ci/check_citations.sh` (the citation gate),
`harness/brief.sh`, `harness/check.sh`, `harness2/domain.sh`. Moving a file here means fixing those
in the same round — the map amended later is a map that was wrong in production.
