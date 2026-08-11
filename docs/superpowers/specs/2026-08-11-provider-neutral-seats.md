# Provider-neutral seats (2026-08-11)

Discharges the provider-neutrality mandate the founder recorded on 2026-08-07:
*"never default seats to Anthropic; the cheap stage-0 stack (Mistral/DeepSeek/euryale) is the
intended direction; per-seat overrides owed for all six seats"*
(`docs/superpowers/handovers/2026-08-07-journey-and-fe-contract-handover.md:270`).

## The gate that came with it, and its status

The same handover records a founder gate one line later: *"Deferred Living World items, GATED before
the real (non-fake) World Actor driver goes live at play … The Journey playthrough with a live driver
should not happen until these are addressed."* — (A) fire-log commit atomicity, (B) a runtime
`location==scene` check in `runWorldActor`, (C) the floor-window world's turn.

**All three are closed** (`660a81d`, `e1d0d17`, `e40cef4`, `adc12dd`; `postCommitFn` in `ledger.go`,
the two `errIntrusionRejected` checks in `worldactor.go`). The gate is satisfied, so live seats may
go to play. Recorded here because the gate was easy to miss and the answer was not obvious.

## What was actually wrong

Routing was an all-Anthropic map plus two hand-written override blocks — one for `resolve`, one for
both cognition seats — each with its own `DREAMCHAT_<SEAT>_{PROVIDER,MODEL,BASE_URL,API_KEY}`
family. Four of seven seats had no override at all. The honest summary was *"Anthropic unless you
are one of three specific seats"*, which is not a neutral contract however swappable it claims to
be. `newAnthropicDriver` also carried a hardcoded `claude-opus-4-8` default, so a config naming no
model still got a working Claude — the exact mechanism by which a default quietly bills someone.

## The scheme

```
DREAMCHAT_SEAT_DEFAULT   provider:model                              # default for every seat
DREAMCHAT_SEATS          seat=provider:model,seat=provider:model     # per-seat overrides

DREAMCHAT_PROVIDER_<NAME>_BASE_URL     where that provider lives
DREAMCHAT_PROVIDER_<NAME>_API_KEY      its bearer token
DREAMCHAT_PROVIDER_<NAME>_DIALECT      wire dialect; default openai-compat
DREAMCHAT_PROVIDER_<NAME>_JSON_MODE    json_object (default) | json_schema
```

`<NAME>` is an operator-chosen alias, uppercased. **No vendor name appears anywhere in the code.**
The code knows wire *dialects* — a technical fact — and the environment knows vendors, a commercial
choice. Adding a provider is two variables and no diff. The seat→provider→dialect indirection is the
mandate made structural rather than promised.

**Fails closed.** No configuration and no `DREAMCHAT_BRIDGE=fake` means the server does not boot,
with the missing variable named. Keeping a default would have been easy and is precisely what the
ruling forbids. A missing config is an operator mistake, and the honest moment to say so is boot,
not the first beat with a 401 from a provider nobody meant to call.

## Structured output

`json_object` is the default and universal: the schema travels in the system message, the model is
told to answer with one JSON document, and the Go validator decides. `json_schema` is opt-in per
provider for those implementing strict decoding — strictly better where available, a 400 where not,
hence opt-in. The system-message leash rides along in **both** modes, because whether a provider's
strict mode is enforced or advisory is not visible from outside.

**Tool-call forcing is deliberately absent from this dialect.** It is the Anthropic driver's leash
because that dialect has no JSON mode; here it would reach the same place by a longer road, and an
unexercised third branch on the seat path is the kind of surface that hides a defect until a live
beat finds it. Retry-on-invalid is unchanged and lives where it always has — the caller's two-attempt
loop (`beatsstream.go:386`), driver-agnostic by design.

**Streaming is unchanged and absent.** `StreamingDriver` is optional and *no* driver implements it,
including the Anthropic one, so narrate has always taken the non-streaming fallback. Moving narrate
to another provider loses nothing.

## Measured token volumes

Per seat, from the real prompt files, the real published schemas, and a measured `fn_world_slice`
payload (7 641 chars ≈ 1 910 tokens). Tokens ≈ chars/4.

| seat | in | out |
|---|---|---|
| decompose | 4 910 | 75 |
| narrate | 4 340 | 125 |
| resolve | 2 962 | 62 |
| cognition_batch | 3 057 | 87 |
| cognition_isolated | 2 102 | 62 |
| world_actor | 3 238 | 50 |
| place_author | 1 208 | 37 |

A **typical beat** fires decompose + narrate + cognition_batch. `resolve` fires only on adjudicated
attempts, `world_actor` only when a pressure tier fires, `place_author` only when a journey mints a
waystation.

## Test coverage

`core/api/seatconfig_test.go` — routing is a pure function of an injected lookup, so every case is a
recorded environment: seven seats across four aliases; overrides land, unnamed seats take the
default; **no config fails closed and names the knob**; a default is optional only when every seat is
named; a typo'd seat name is rejected rather than silently falling back (the invisible-until-the-bill
bug); a missing endpoint names the exact variable; alias normalisation; models containing colons; the
fake bridge needing no configuration. Plus the driver's recorded wire shapes for both structured
modes, free text sending no `response_format`, an unknown `json_mode` rejected at construction, and
every structured seat binding on the driver.
