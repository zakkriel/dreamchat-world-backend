# Runbook — Sprite Sheet Quality Failure

## Symptoms

- Sprite sheet generated but sliced expressions look wrong.
- Some cells contain no face/portrait.
- Character identity changes across cells.
- Provider added text labels or frames inside cells.
- One expression is unusable.

## Immediate Checks

1. Open original sheet asset.
2. Check whether the grid matches the active contract.
3. Verify row/column count.
4. Inspect generated slices.
5. Check provider/model used.
6. Check prompt version and style profile.
7. Check whether reference image was provided.

## Mitigation

If most cells are usable:

- keep valid slices
- mark invalid slices as `failed_quality_check`
- map missing expressions to neutral or closest available expression
- optionally queue separate generation for missing expressions

If sheet is unusable:

- mark parent sheet as failed
- retry once with stricter prompt
- if retry fails, fallback to separate expression generation

## Follow-Up

- Add failure reason to provider/model quality stats.
- Improve prompt compiler or grid contract.
- Disable sprite-sheet route for provider if failure rate is high.
- Add sample to benchmark corpus.
