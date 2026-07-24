# Resolver live-smoke runbook

Runs the live-model resolver (resolve seat only) against a locally running API.
Use this to verify that a real model rules an adjudicated attempt and writes a canon event
with a perceived appearance that differs from the canon truth (the deception split).

---

## Environment variables

| Variable | Required | Example (DeepInfra) | Example (OpenRouter) |
|---|---|---|---|
| `DREAMCHAT_RESOLVE_API_KEY` | Yes | `di_...` | `sk-or-...` |
| `DREAMCHAT_RESOLVE_BASE_URL` | Yes | `https://api.deepinfra.com/v1/openai` | `https://openrouter.ai/api/v1` |
| `DREAMCHAT_RESOLVE_MODEL` | Yes | `meta-llama/Meta-Llama-3.1-70B-Instruct` | `meta-llama/llama-3.1-70b-instruct` |
| `DREAMCHAT_RESOLVE_PROVIDER` | No (defaults to `openai-compat`) | `openai-compat` | `openai-compat` |

> The fourth variable `DREAMCHAT_RESOLVE_PROVIDER` is only needed when running the server
> (it controls `defaultSeatConfig` in `main.go`). The go test live smoke wires the driver
> directly from the first three.

---

## Boot sequence

```bash
# 1. Start Postgres and apply all migrations + seed data
make db-up && make migrate && make seed

# 2. Run the API with fake decompose/narrate and live resolve seat
export DREAMCHAT_RESOLVE_PROVIDER=openai-compat
export DREAMCHAT_RESOLVE_BASE_URL=https://api.deepinfra.com/v1/openai
export DREAMCHAT_RESOLVE_MODEL=meta-llama/Meta-Llama-3.1-70B-Instruct
export DREAMCHAT_RESOLVE_API_KEY=di_YOUR_KEY_HERE
export DREAMCHAT_MODE=debug

cd core/api
go run .
```

The server starts on `:8080`. Debug mode (`DREAMCHAT_MODE=debug`) enables the `?viewer=` query
override so you can inspect any actor's perception wall.

---

## curl example

Post a beat that decomposes to an `AttributeChanged` attempt targeting Mara (bbbb…bbbb).
Player ID is `aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`; world ID is `11111111-1111-1111-1111-111111111111`.

```bash
curl -s -X POST \
  'http://localhost:8080/worlds/11111111-1111-1111-1111-111111111111/beat?viewer=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' \
  -H 'Content-Type: application/json' \
  -d '{"text":"I lean on her to talk about the harbormaster"}' | jq .
```

---

## What a good response looks like

```json
{
  "schema_version": "beat_result/2",
  "narration": "You press Mara for information...",
  "result": {
    "committed": ["<event-uuid>"],
    "halt_reason": "completed",
    "ticks_advanced": 0,
    "unresolved_candidates": [],
    "telegraphs": []
  }
}
```

- `halt_reason: "completed"` — the model issued a ruling and a canon event was committed.
- `halt_reason: "bounce"` — the model ruled "impossible"; nothing written to canon. This is
  legitimate (a live model may genuinely decide the attempt fails impossibly).
- The `narration` is built from the **player's perception wall** (appearance text), not from
  canon truth. This is the deception split: the timeline shows what the player perceived, while
  canon carries the unvarnished truth.

### Inspect the deception split

With debug mode on, query the timeline endpoint for each participant to see what they perceived:

```bash
# Player's perceived timeline (appearance text)
curl -s 'http://localhost:8080/worlds/11111111-1111-1111-1111-111111111111/timeline?viewer=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' | jq '.[-1]'

# Canon truth lives in the database; query it directly:
# (substitute the event UUID from the beat response)
psql "$DATABASE_URL" -c "SELECT summary FROM canon_event WHERE event_id='<event-uuid>';"
```

The `summary` column in `canon_event` is the truth. The `content` column in `perception_record`
(joined via `fn_visible_perceptions`) is the appearance the player receives.

---

## Cost note

A single beat through the resolve seat is typically one completion request: short system prompt
(~200 tokens) + slice context (~300–500 tokens) + ruling output (~100–200 tokens).
At mid-2026 open-model prices on DeepInfra or OpenRouter this is fractions of a cent per beat
for a 70B-class model. **Provider pricing changes frequently — check the provider's pricing page
before assuming these numbers.**

---

## Run the gated live smoke test

```bash
export DREAMCHAT_RESOLVE_API_KEY=di_YOUR_KEY_HERE
export DREAMCHAT_RESOLVE_BASE_URL=https://api.deepinfra.com/v1/openai
export DREAMCHAT_RESOLVE_MODEL=meta-llama/Meta-Llama-3.1-70B-Instruct

cd core/api
go test -count=1 -run TestStationD_LiveSmoke -v ./...
```

The test skips automatically when `DREAMCHAT_RESOLVE_API_KEY` is unset (all other CI runs).
It asserts only structure: one committed adjudicated event exists and `halt_reason ∈ {completed, bounce}`.
It never asserts model prose content.
