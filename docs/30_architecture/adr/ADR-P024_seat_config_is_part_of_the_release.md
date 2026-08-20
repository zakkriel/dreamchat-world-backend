# ADR-P024: Production seat configuration is part of the release, not folklore in `.env.example`

**Status:** Accepted (2026-08-20)
**Date:** 2026-08-20
**Series:** Platform / Operations (ADR-P###, per D-5). Sibling to ADR-P020 — same failure, different
axis. Attaches to D-13 / ADR-P018 (per-seat routing).
**Owner of decision:** raised by the harness review (Challenger §5), on evidence from the
world-creation bring-up.
**Evidence (D-9):** `core/api/seatconfig.go` (`DREAMCHAT_SEAT_MAX_TOKENS_%s`,
`DREAMCHAT_SEAT_JSON_MODE_%s`), `.env.example` (the measured genesis figures), and the production
incident recorded below.

## Context

ADR-P020 says a merged schema change is not a released one, and makes the binary refuse a database
that disagrees with it. **Configuration is the same failure on a different axis, and it is not
guarded at all.**

Seat behaviour is environment-resolved by design (D-13: no default provider, per-seat overrides). So
correct, tested, merged code can run in production against seat parameters that cannot serve it, and
nothing anywhere says so:

- `world_genesis` produced measured completions of **3034 and 3384 tokens**. Production's provider
  default was **2048**. Deployed as-is, every build truncates and is reported to the user as a
  malformed-document refusal — a content error for what is a config error.
- `world_genesis` and `world_interview` need `json_object`, not `json_schema`: the contract uses
  optional fields and `oneOf`, which strict schema mode rejects.

That knowledge lived in a commented block in `.env.example`. CI never sees Railway's environment, so
nothing checks it, and the only reason it was applied at all is that a human read the PR body.

## Decision

**A named seat's required configuration is part of its release, and is recorded where the seat is
defined — not only in `.env.example`.**

1. A seat whose contract requires more than the provider default (token ceiling, JSON mode, capability
   floor) states that requirement next to its `Seat` definition, and `.env.example` remains the copy
   the operator pastes.
2. Shipping a seat, or changing a seat's contract, includes setting its variables in every environment
   **before** the merge that makes them load-bearing — the same ordering ADR-P020 imposes on
   migrations, and for the same reason.
3. A release is not "merged". It is: **config applied → merged → deploy observed → path exercised.**
   Railway reporting SUCCESS means a container started, not that a feature works.

## Consequences

- The world-creation bring-up needed exactly this sequence and got it by hand:
  `DREAMCHAT_SEAT_MAX_TOKENS_WORLD_GENESIS=8192`,
  `DREAMCHAT_SEAT_JSON_MODE_WORLD_GENESIS=json_object`,
  `DREAMCHAT_SEAT_JSON_MODE_WORLD_INTERVIEW=json_object`.
- **Still unguarded (open, deliberately named rather than pretended away):** there is no boot check
  that a bound seat's resolved parameters can serve its contract, and no `make doctor` against a
  committed required-env manifest. ADR-P020 earned its refusal after a day of production damage; this
  one has burned a bring-up. The gate is worth building and is not built. Do not read this ADR as
  saying the problem is solved — it says the problem is *known*, and names the sequence that avoids it
  until a gate exists.
