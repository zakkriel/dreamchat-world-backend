#!/usr/bin/env bash
#
# fill_probe.sh — run THE FILL and nothing else, against the real seats, and report what it cost.
#
# ONE COMMAND. It runs the understanding pass, every fill wave, the reconciliation and the belt, in
# process. No database. No commit. No kickstart. No play. It writes the authored document where you can
# read it, and a ledger of where every output token went, grouped by work item.
#
#   ci/fill_probe.sh docs/design/2026-08-27-understanding-pass-probe/andantes-brief.md
#   FILL_PROBE_DEPTH=1 ci/fill_probe.sh /tmp/brief.md
#
# WHY NOT `POST /worlds/genesis`. That measures the whole pipeline to answer a question about the fill:
# it needs a database, it writes a half-world into production on the way, the stream is cut at a
# 900-second proxy edge, and Railway's log retention is shorter than a build so the per-call numbers are
# gone before you can read them. This keeps the fill in isolation, which is where the work is.
#
# NOT A CI GATE, on purpose: it spends real money and needs a real key.
#
# Seats come from the environment exactly as production resolves them, so what you measure here is what
# production runs. Override any of DREAMCHAT_SEAT_DEFAULT / DREAMCHAT_SEATS /
# DREAMCHAT_SEAT_ROUTING_WORLD_FILL / DREAMCHAT_SEAT_JSON_MODE_WORLD_FILL to compare configurations.
set -uo pipefail

BRIEF="${1:-}"
if [ -z "$BRIEF" ] || [ ! -f "$BRIEF" ]; then
  echo "usage: $0 <brief-file>" >&2
  exit 2
fi

if [ -z "${DREAMCHAT_PROVIDER_OPENROUTER_API_KEY:-}" ] && [ -z "${OPENROUTER_API_KEY:-}" ]; then
  cat >&2 <<'MSG'
No provider key in the environment. This probe calls a real model and costs real money.

  export DREAMCHAT_PROVIDER_OPENROUTER_API_KEY="$(railway variables --service world-api --kv \
      | grep '^DREAMCHAT_PROVIDER_OPENROUTER_API_KEY=' | cut -d= -f2-)"

Pull the rest of the live seat config the same way if you want to measure what production runs:

  for v in DREAMCHAT_SEAT_DEFAULT DREAMCHAT_SEATS DREAMCHAT_PROVIDER_OPENROUTER_BASE_URL \
           DREAMCHAT_PROVIDER_OPENROUTER_ROUTING DREAMCHAT_SEAT_ROUTING_WORLD_FILL \
           DREAMCHAT_SEAT_JSON_MODE_WORLD_FILL DREAMCHAT_SEAT_MAX_TOKENS_WORLD_FILL; do
    export "$v=$(railway variables --service world-api --kv | grep "^$v=" | cut -d= -f2-)"
  done
MSG
  exit 2
fi

DIR="${FILL_PROBE_DIR:-$(mktemp -d /tmp/fillprobe.XXXXXX)}"
mkdir -p "$DIR"

cd "$(dirname "${BASH_SOURCE[0]}")/../core/api" || exit 1

echo "=== fill probe ==="
echo "brief : $BRIEF"
echo "depth : ${FILL_PROBE_DEPTH:-1}"
echo "out   : $DIR"
echo

FILL_PROBE_DIR="$DIR" FILL_PROBE_BRIEF="$(cd "$(dirname "$BRIEF")" && pwd)/$(basename "$BRIEF")" \
  go test -count=1 -timeout 3600s -v -run '^TestFillProbe$' . 2>&1 |
  grep -vE '^(=== RUN|--- PASS|PASS$|ok )' |
  sed -E 's/^[[:space:]]*(hostconformance_test\.go:[0-9]+:[[:space:]]*)?//'

echo
echo "=== written ==="
ls -la "$DIR"
echo
echo "Read the document:  python3 -m json.tool $DIR/document.json | less"
