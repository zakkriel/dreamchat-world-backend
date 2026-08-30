#!/usr/bin/env bash
# SPEC-039: is every host in the routing allowlist actually able to parse a beat?
#
# ONE COMMAND. It dumps the real assembled decompose prompt + schema from the seeded play world, then
# replays those exact bytes against each allowlisted host separately and scores them.
#
#   OPENROUTER_API_KEY=... ci/host_conformance.sh
#
# It reads the binding from `railway variables` ITSELF — model, routing allowlist, token ceiling and
# reasoning effort — because a score means nothing without the binding it was measured under, and a doc
# or a comment is not the authority (AGENTS.md rule 0c). Override any of it with HOST_PROBE_MODEL /
# HOST_PROBE_HOSTS, or by exporting the DREAMCHAT_* names directly.
#
# It deliberately does NOT hand you an `eval` recipe. The obvious one is wrong:
#   eval "$(railway variables --kv | sed 's/^/export /')"
# turns {"effort":"none"} into {effort:none} — bash strips the quotes — and the probe then sends an
# invalid effort to every host and scores all four DEAD. That cost a run here on 2026-08-30.
#
# NOT A CI GATE, on purpose: it spends real money and needs a real key. Run it when the allowlist
# changes, when a host is added, or when a seat starts answering in a way that feels wrong.
#
# Needs a database with the seeded Drowned Lantern world.
set -uo pipefail
cd "$(dirname "$0")/.."

# One python read of the authority, so no value ever passes through shell quoting.
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

def pick(name, default=""):
    # An explicitly exported value wins over the deployed one: that is how you test a candidate
    # binding before you set it.
    return os.environ.get(name) or live.get(name) or default

model = os.environ.get("HOST_PROBE_MODEL", "")
if not model:
    # DREAMCHAT_SEAT_DEFAULT covers every seat NOT named in DREAMCHAT_SEATS, and decompose is not
    # named there — which is exactly how an earlier round tested the wrong model and called it verified.
    seats = pick("DREAMCHAT_SEATS")
    named = dict(p.split("=", 1) for p in seats.split(",") if "=" in p) if seats else {}
    model = named.get("decompose") or pick("DREAMCHAT_SEAT_DEFAULT")
model = model.split(":", 1)[1] if model.startswith("openrouter:") else model

hosts = os.environ.get("HOST_PROBE_HOSTS", "")
if not hosts:
    routing = pick("DREAMCHAT_PROVIDER_OPENROUTER_ROUTING")
    try:
        hosts = ",".join(json.loads(routing).get("only") or []) if routing else ""
    except json.JSONDecodeError:
        hosts = ""

effort = ""
reasoning = pick("DREAMCHAT_PROVIDER_OPENROUTER_REASONING")
if reasoning:
    try:
        effort = json.loads(reasoning).get("effort") or ""
    except json.JSONDecodeError:
        effort = ""

print(model)
print(hosts)
print(pick("DREAMCHAT_PROVIDER_OPENROUTER_MAX_TOKENS", "2048"))
print(effort)
print("railway" if live else "environment")
'
)
[[ -n "$BINDING" ]] || { echo "could not resolve the seat binding" >&2; exit 2; }
{ read -r MODEL; read -r HOSTS; read -r MAXTOK; read -r EFFORT; read -r SOURCE; } <<< "$BINDING"

[[ -n "$MODEL" ]] || { echo "no model: set HOST_PROBE_MODEL, or DREAMCHAT_SEAT_DEFAULT=openrouter:<model>" >&2; exit 2; }
[[ -n "$HOSTS" ]] || { echo "no hosts: set HOST_PROBE_HOSTS=a,b,c or DREAMCHAT_PROVIDER_OPENROUTER_ROUTING" >&2; exit 2; }
echo "binding read from: $SOURCE"

MATERIAL="$(mktemp -d)"
trap 'rm -rf "$MATERIAL"' EXIT

echo "== dumping the real request material from the seeded world"
( cd core/api && HOST_PROBE_DIR="$MATERIAL" go test -run '^TestGenDecomposeHostProbe$' -count=1 . ) \
  || { echo "the dumper failed — is the seeded database up?" >&2; exit 1; }

echo "== scoring each host on identical bytes"
exec ci/host_conformance.py \
  --material "$MATERIAL" \
  --corpus ci/host_conformance_corpus.tsv \
  --model "$MODEL" \
  --hosts "$HOSTS" \
  --max-tokens "$MAXTOK" \
  --reasoning-effort "$EFFORT" \
  "$@"
