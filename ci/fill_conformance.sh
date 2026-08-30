#!/usr/bin/env bash
# SPEC-039 companion: can each allowlisted host return a valid world_fill/1 fragment?
#
#   OPENROUTER_API_KEY=... ci/fill_conformance.sh
#
# ONE COMMAND. It dumps the real assembled world_fill request — identity, the descent's authored
# document, and the per-person ascent work item — then replays those exact bytes against each host
# separately and scores whether the reply PARSES.
#
# WHY THIS EXISTS BESIDE ci/host_conformance.sh, RATHER THAN INSIDE IT. That probe measures whether a
# host picks the right branch of a small schema from a short sentence, against a corpus whose own
# header says it is a fixed ruler and must not change between measurements. Fill asks a different
# question of the same hosts: given ~16KB in and up to 16384 tokens out against a deeply nested schema
# with additionalProperties:false, does the answer parse at all? On 2026-08-28 three live fill calls
# failed with `unknown field "canonical_name"`, `unknown field "hiding"` and `unexpected EOF`. Nothing
# in a sentence-to-type corpus could have caught any of them.
#
# It reads the binding from `railway variables` ITSELF — model, allowlist, token ceiling, effort —
# because a score means nothing without the binding it was measured under. Override with
# FILL_PROBE_MODEL / FILL_PROBE_HOSTS.
#
# It sends json_object, because that is what the world_fill seat runs in production. Measuring
# json_schema strict here would score a mode the pipeline never exercises.
#
# NOT A CI GATE, on purpose: it spends real money and needs a real key. Run it when the allowlist
# changes, when a host is added, or when the fill seat starts returning malformed fragments.
set -uo pipefail
cd "$(dirname "$0")/.."

BINDING=$(
  {
    if command -v railway >/dev/null 2>&1; then railway variables --json --service world-api 2>/dev/null; fi
  } | python3 -c '
import json, os, sys
raw = sys.stdin.read().strip()
live = {}
if raw:
    try:
        live = json.loads(raw)
    except json.JSONDecodeError:
        live = {}

def var(name, default=""):
    return str(live.get(name, os.environ.get(name, default)) or default)

# The fill seat may carry its own routing; that override is the whole point of pinning it.
routing = var("DREAMCHAT_SEAT_ROUTING_WORLD_FILL") or var("DREAMCHAT_PROVIDER_OPENROUTER_ROUTING")
hosts = []
if routing:
    try:
        hosts = json.loads(routing).get("only") or []
    except json.JSONDecodeError:
        hosts = []

seats = var("DREAMCHAT_SEATS")
model = ""
for pair in seats.split(","):
    if pair.strip().startswith("world_fill="):
        model = pair.split("=", 1)[1].split(":", 1)[-1]
if not model:
    model = var("DREAMCHAT_SEAT_DEFAULT").split(":", 1)[-1]

effort = "none"
reasoning = var("DREAMCHAT_SEAT_REASONING_WORLD_FILL") or var("DREAMCHAT_PROVIDER_OPENROUTER_REASONING")
if reasoning:
    try:
        effort = json.loads(reasoning).get("effort", "none")
    except json.JSONDecodeError:
        pass

print(os.environ.get("FILL_PROBE_MODEL") or model)
print(os.environ.get("FILL_PROBE_HOSTS") or ",".join(hosts))
print(var("DREAMCHAT_SEAT_MAX_TOKENS_WORLD_FILL", "16384"))
print(effort)
'
)
MODEL=$(printf '%s\n' "$BINDING" | sed -n 1p)
HOSTS=$(printf '%s\n' "$BINDING" | sed -n 2p)
MAXTOK=$(printf '%s\n' "$BINDING" | sed -n 3p)
EFFORT=$(printf '%s\n' "$BINDING" | sed -n 4p)

if [ -z "$MODEL" ] || [ -z "$HOSTS" ]; then
  echo "could not read the binding. Set FILL_PROBE_MODEL and FILL_PROBE_HOSTS, or log in to railway." >&2
  exit 1
fi

echo "== binding read from the live service"
echo "   model      $MODEL"
echo "   hosts      $HOSTS"
echo "   max_tokens $MAXTOK"
echo "   effort     $EFFORT"
echo

MATERIAL=$(mktemp -d)
trap 'rm -rf "$MATERIAL"' EXIT

echo "== dumping the real world_fill request material"
( cd core/api && HOST_PROBE_DIR="$MATERIAL" go test -run '^TestGenFillHostProbe$' -count=1 . ) \
  || { echo "the dumper failed" >&2; exit 1; }

echo
exec python3 ci/fill_conformance.py \
  --material "$MATERIAL" \
  --model "$MODEL" \
  --hosts "$HOSTS" \
  --max-tokens "$MAXTOK" \
  --reasoning-effort "$EFFORT" \
  "$@"
