# seats-and-bridge · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-13 · Seats, the LLM bridge, and cost ·
**Parent bounded context:** World Engine

This file holds how the domain is built — architecture, boot and call paths, validation, traps.
`seats-and-bridge.product.md` holds what it means; `seats-and-bridge.seams.md` holds what crosses
its boundary.

Line numbers are as of 2026-08-27; re-locate by grep before relying on one.

---

## Where the code lives

Verified against `git ls-files`, 2026-08-27 — all of it in `core/api/`, none in `core/db` (`D-13`):

- `bridge.go` — capabilities, `Seat`/`Driver`, `BindSeat`, the `timedDriver` decorator, `DefaultDriverFactory`
- `seatconfig.go` — environment → seat routing; `anthropic.go`, `openaicompat.go` — the two dialects
- `bridge_fakes.go` — every deterministic stand-in; `costsink.go` — the spend ledger
- `promptlatitude_test.go` + `prompts/` — the latitude gate and the fixed seat rulebooks
- Tests: `bridge_test.go`, `bridge_seats_test.go`, `bridge_intent_test.go`, `seatconfig_test.go`,
  `costsink_test.go`, `openaicompat_test.go`, `openaicompat_timeout_test.go`, `anthropic_test.go`

## The boot path

`main.go:124-133`: resolve seat config, build the bridge, **fail closed twice**:

1. **No default provider.** With no config and no `DREAMCHAT_BRIDGE=fake`, `seatConfigFromEnv`
   errors with the missing variable named and the server does not boot (`seatconfig.go:87-97`).
   A default is what quietly bills someone; `anthropic.go:45-49` records the hardcoded model it
   removed for the same reason. An unknown seat name in `DREAMCHAT_SEATS` is an error, not a shrug
   (`parseSeatOverrides`).
2. **Capability floor.** `BindSeat` (`bridge.go:136-145`) refuses any driver whose *reported*
   `Capabilities()` do not satisfy the seat's floor. Capability is a report, never a config label.

The environment scheme is `DREAMCHAT_SEAT_DEFAULT`, `DREAMCHAT_SEATS`,
`DREAMCHAT_PROVIDER_<ALIAS>_{BASE_URL,API_KEY,DIALECT,JSON_MODE}`, plus per-seat
`DREAMCHAT_SEAT_{ROUTING,MAX_TOKENS,TEMPERATURE,REASONING,JSON_MODE}_%s` (`seatconfig.go:46-67`).
`ADR-P024` makes setting these **part of the release**, before the merge that needs them.

## The call path

Every call goes `consumer → bridge.Driver(seat) → timedDriver → driver`. The decorator is applied
once in the binding path — an instrument added at call sites would be missing from the fourth one
(`bridge.go:212-214`) — and preserves `StreamingDriver` only when the underlying driver has it,
so the instrument cannot change what it measures (`bridge.go:244-272`).

**Structured requests (the leash).** `anthropic.go` forces a single tool whose `input_schema` is the
caller's schema (`tool_choice`, line 107). `openaicompat.go` has three modes (`jsonModeObject` /
`jsonModeSchema` / `jsonModeOff`, lines 199-209), each with a measured reason in its comment —
`off` exists because constrained decoding returned structurally perfect, textually empty narration.
Two refusals happen before sending: `json_object` with an array-rooted schema is a construction
error (`openaicompat.go:288-291`), and the system message names the array root explicitly
(`:268-271`) — both bought with live debugging sessions.

**Cost.** The request asks for the bill (`body["usage"]`, `openaicompat.go:260`). The invoice is
recorded **before the content checks**: *"a reply that fails validation was still billed, and a
spend report that only counts successful calls is the one that under-reports a repair storm —
exactly the pathology the number exists to catch"* (`openaicompat.go:354-358`). `costsink.go`
attributes spend to seats by before/after snapshot and warns past `DREAMCHAT_BEAT_COST_WARN_USD`;
the genesis ceiling lives at its consumer (`worldgenesishandler.go:67`).

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `D-13`, `ADR-P018` | Per-seat routing behind `Driver`; capability floor validated against the reported set; no provider SDK in `core/db`; adding a provider is one factory case, adding a seat one `Seat` entry. | A shared client or hardcoded model couples the engine to a vendor and hides the decompose seat's constrained-generation requirement. |
| `ADR-P022` | The latitude block: five paragraphs, byte-identical, immediately after every role line; gated by `promptlatitude_test.go`. | One censoring seat breaks the world; failure-log row 5 is the receipt. |
| `ADR-P024` | A seat's required config is recorded next to its `Seat` definition and set in every environment before the merge that needs it. | `world_genesis` truncated at a 2048 provider default and reported a config error as a content error. |
| `ADR-P020` (sibling, by `ADR-P024`'s own framing) | Merged is not released — the same ordering discipline, on the config axis. | Correct, tested, merged code runs against parameters that cannot serve it, silently. |

### What you may not decide alone

1. **Adding a capability to the set or a seat to `allSeatNames`.** A new seat needs its floor, its
   config recorded (`ADR-P024`), its latitude block, and its fake — all four, not some.
2. **Any latitude wording change.** Founder-locked text (`ADR-P022`); the gate enforces byte-identity,
   not meaning.
3. **A default provider, anywhere.** Founder ruling on the record since 2026-08-07
   (`seatconfig.go:12-14`).
4. **Weakening a fake.** A fake laxer than the real driver is how three bugs shipped green
   (`ADR-P018`, incident note; failure-log row 37).

## Validation for this domain

`cd core/api && go test -run 'Latitude|Bridge|Seat|CostSink|OpenAICompat|Anthropic' -count=1 .` —
plus the governance gate (`go test -run 'Govern' -count=1 .`). Always `-count=1`; the suite is not
idempotent.

- **What counts as evidence here:** boot failures are loud by design, but the domain's
  characteristic failure is a **green server committing nothing** — `DREAMCHAT_BRIDGE=fake` with a
  wrong-shaped stand-in streamed correct frames while canon stayed at seed rows
  (`bridge_fakes.go:389-395`). Hand-drive a beat; do not trust the frame sequence.
- **What counts as ceremony here:** asserting on a fake's output shape. The live anthropic driver is
  *compiled and vetted, never invoked by a test* (`anthropic.go:12-14`) — the CI guarantee is the
  capability floor plus the constrained fake, so a test of the fake proves nothing about the driver
  unless the fake's strictness is itself pinned (which `bridge_test.go:21` does for narrate's open
  floor).

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The latitude block edited in less than all files.** An agent edited two prompt files and stopped; a founder caught it. | failure-log row 5; the gate is `promptlatitude_test.go`, which reads files AND embedded prompts — an earlier version listed seven seats and missed the two embed.FS ones (`promptlatitude_test.go:107-113`). |
| **A fake laxer than its seat.** Four recorded instances in `bridge_fakes.go` alone; SPEC-034 names the pattern's fourth workspace instance. | failure-log row 37; `bridge_fakes.go` headers (nil table, wrong shape, no-capability narrate, hardcoded seeded ids). |
| **Evidence paths without the `core/api/` prefix are silently skipped** by `governance_test.go`'s regex — the gate once checked one of four named files while appearing to check all. | failure-log row 31; `ADR-P018`'s own incident note. |
| **A streamed reply's invoice arrives in a final chunk with no `choices`.** The skip-guard threw it away; narrate — the one streaming seat — under-reported every beat. | `openaicompat.go:530-534`. |
| **A reasoning model spends the whole budget thinking** and returns null content, `finish_reason` "length" — surfacing two retries later as a schema failure blaming the model. | `openaicompat.go:362-368` names it instead. |
| **Cost delta attribution assumes seats run sequentially.** Parallelise seats and misattribution is silent — never a wrong beat total, always the wrong seat. | `costsink.go:78-81`, recorded in the snapshot's own comment. |

## Open questions

1. **`ADR-P022`'s evidence line says `prompts/*.txt` "(all nine)"; there are ten** (verified,
   `git ls-files 'core/api/prompts/*.txt'`). The same understated-count shape failure-log row 4
   warns about. Editing an ADR is the founder's; recorded, not resolved.
2. **When is the `ADR-P024` boot gate built?** The ADR says it is worth building and not built;
   nothing schedules it.
3. **The cognition seats' comments cite `SPEC-?`** (`bridge.go:108-113`) — placeholders where a spec
   id should resolve. Whoever knows the ids should pin them.
