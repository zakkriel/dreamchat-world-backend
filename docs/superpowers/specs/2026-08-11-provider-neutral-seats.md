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


---

# Amendment (2026-08-11): one aggregator, and a routing policy that is configuration

Founder ruling refining the above: a single live endpoint — an OpenAI-compatible aggregator — with a
**required routing policy on every request**: US/EU hosts only, `data_collection: deny`, and
DeepSeek-the-company excluded as an underlying host (open weights are fine; that company's pods are
not).

## The policy is per-seat config, never a constant

The aggregator takes a `provider` preferences object deciding WHICH host serves a request. It is
carried as `Params["routing"]`, resolved from
`DREAMCHAT_PROVIDER_<NAME>_ROUTING` with a per-seat override at
`DREAMCHAT_SEAT_ROUTING_<SEAT>`, and merged into every request body verbatim.

Three decisions worth stating:

- **Configuration, not a constant.** Jurisdiction and retention rules are commercial and legal
  judgements that change without this code changing. A policy compiled into a binary is one nobody
  can correct without a deploy.
- **Passed through as raw JSON, not modelled as a struct.** The aggregator owns that schema and adds
  fields regularly. A struct here would silently drop any field this repo had not heard of, turning
  *"policy not yet supported"* into *"policy silently not applied"* — the worst failure mode for a
  field whose entire job is compliance.
- **Validated at construction.** A malformed policy is a boot failure naming the seat, not a 400 in
  the middle of a playtest. Compliance config that fails late fails in front of the person it was
  meant to protect.

Every request carries it, structured or free-text alike: a narration is exactly as subject to
jurisdiction and retention rules as a schema'd call. An unrouted seat is not silently permissive —
the boot line prints `seat=alias:model(routed|unrouted)`.

## What "US/EU only" can and cannot mean here

**Verified against the aggregator's own API, not assumed.** Its provider records expose
`headquarters` — and that is **company HQ, not data-centre location**. There is no per-endpoint
region field in the standard preferences object; true in-region routing is an enterprise feature
behind a sales contact.

So "US/EU hosts only" is implemented as an explicit `only:[…]` allowlist derived from HQ, filtered
further to hosts whose published data policy is *no training and no prompt retention*, and
intersected with the endpoints that actually advertise structured output for the chosen models.
That is the strongest available enforcement short of the enterprise tier, and its limit should be
stated to the founder rather than papered over.

`require_parameters: true` is not decoration. One compliant endpoint for the strong model
(BaseTen, v4-pro) does **not** advertise structured output; without that flag the aggregator could
route a schema'd seat to it and the leash would fail at validation instead of at routing.

## Quantization is a quality lever nobody asked about yet

Compliant endpoints differ in quantization: the two cheapest for the mechanical model are `fp4`, the
cheapest for the narrative model is `fp8`. The preferences object takes a `quantizations` filter.
Left unset for now — the narrative seat happens to land on `fp8` anyway — but if narration reads
thin during the founder's week of play, that is the first knob to try before changing model.
