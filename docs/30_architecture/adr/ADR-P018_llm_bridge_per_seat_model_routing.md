# ADR-P018: LLM bridge — model-agnostic per-seat routing

**Status:** Accepted (chunk-5 Leg-2 gate, 2026-06-18)
**Date:** 2026-06-18
**Series:** Platform / Bridge (ADR-P###, per D-5) — does NOT touch frozen engine canon. ADR-036
(action-driven clock) remains the only *engine* ADR for Chunk 5; this is a bridge-layer ADR.
**Owner of decision:** Chunk 5 Leg 2 (first LLM in the loop)
**Governing rule:** D-13 (model-agnostic per-seat LLM routing — added to the register by the founder).
**Evidence (D-9):** the running bridge (`core/api/bridge.go`, `bridge_fakes.go`, `anthropic.go`,
`beathandler.go`) + `bridge_test.go` green in this PR. Filed *with* code, not ahead of it.

## Context

Chunk 5 puts the LLM in the play loop. There are multiple distinct LLM *jobs* (decompose prose→events;
narrate; later resolve/adjudicate and NPC cognition), each with different cost, risk, and capability
needs. A single hardcoded model, or a single client the whole loop shares, would (a) couple the engine
to a provider, (b) make the cheap high-volume seat (narrate) and the high-risk seat (decompose) share
a model they shouldn't, and (c) hide the decompose seat's hard requirement — schema-constrained
generation — behind prose-asking that can still emit out-of-vocab.

## Decision

The bridge routes **per seat**. Concretely (all in the bridge layer, `core/api` — never `core/db`):

1. **Seat taxonomy.** A `Seat` is one LLM job with a name and a capability floor. Thin slice wires two:
   `decompose` (requires structured output) and `narrate` (requires nothing). `resolve` (SPEC-013) and
   NPC cognition (SPEC-012) drop in later as **new seat entries** — no redesign.
2. **Driver interface.** A `Driver` is one bound model: `Name()`, `Capabilities() CapabilitySet`,
   `Generate(ctx, GenRequest) (string, error)`. `GenRequest` carries the perception-bound payload
   (B-1/I-3) + prompt + an optional `Schema` (non-nil ⇒ structured/constrained generation).
3. **Capability floor, validated against the REPORTED set — never a config label.** `BindSeat` fails
   **closed** if the driver does not report the seat's required capabilities. This is the
   Image-Platform latent-risk lesson: a config label can claim anything; only the driver's reported
   capability is trusted.
4. **Structured output IS the decompose leash, at generation time.** The decompose seat requires
   `CapStructuredOutput`; its driver constrains generation to the `beat_chain/1` schema, so output is
   schema-valid **by construction** — not prompt-asking, not tool-use-as-hint (both can still emit
   out-of-vocab). `DecodeAndValidateChain` remains as a defense-in-depth belt. Therefore an out-of-vocab
   event is *structurally* prevented: a non-constrained driver cannot even be **bound** to decompose.
5. **Per-seat config.** `SeatConfig` maps seat name → `{provider, model, params}`; a `DriverFactory`
   resolves it. Re-pointing one seat's entry changes **only** that seat.
6. **Default driver = Claude via the Anthropic API** (structured output via forced tool-use as the
   decompose leash), swappable behind the interface. Deterministic fakes are used in CI; the live model
   is wired separately and kept **out of CI** (needs `ANTHROPIC_API_KEY`).
7. **Quarantine holds per seat regardless of bound model:** decompose proposes only (D-1/SPEC-015);
   narrate is perception-bound (ADR-020); the gate (`apply_beat`) is the only canonization point.

## Consequences

- No provider SDK touches the canon engine (`core/db` is pure SQL). Drivers live only in the bridge.
- Adding a provider is one `case` in `DefaultDriverFactory`; adding a seat is one `Seat` entry.
- `beat_chain/1` is a published **input-contract** schema (frontend codegens requests from it); SPEC-011
  exempts it from output-payload coverage (it has no projection payload by design).
- The live Anthropic driver is compiled + vetted but exercised only outside CI; the capability floor +
  the constrained fake carry the CI guarantee.

## Reversibility

Swapping providers or models is a config change behind the `Driver` interface; no engine or handler
code changes. A future seat (resolve/cognition) is additive. Not a one-way door.
