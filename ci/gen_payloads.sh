#!/usr/bin/env bash
# =============================================================================
# SPEC-011 — generate REAL projection payloads from the seeded db by calling the
# actual SQL JSON functions for every seeded entity, as BOTH the Player and Jonas
# viewers. Non-null results are written to the payload dir (one JSON per file) for
# ci/schema_contract.py to validate. NULL results (unperceived entity -> 404 case)
# are skipped — there is no payload to validate.
#
# Requires a seeded db (run `make reset` first). Uses the repo's standard
# `docker compose exec -T db psql` access. Read-only.
# =============================================================================
set -euo pipefail
OUT="${1:-ci/.payloads}"
# NOTE: </dev/null is load-bearing — `docker compose exec` attaches stdin, which would
# otherwise EAT the `while read` loop's input below (only the first entity would be processed).
PSQL() { docker compose exec -T db psql -U postgres -d dreamchat -Atc "$1" </dev/null; }

rm -rf "$OUT"; mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"  # absolute — the scene/current generator below cds into core/api to run

# The payload contract targets the compendium FIXTURE world — the one that has a 'Player' actor.
# Anchor WORLD on 'Player' (never LIMIT-1-any-world): the dedicated Drowned Lantern play world
# (22222222-…) also seeds a 'Mara'/'Jonas' but uses 'Kade', not 'Player', so anchoring here pins the
# generator to the fixture world and keeps the name lookups from matching across worlds (a raw
# multi-row result would inject a control character into the payloads/manifest).
PLAYER_ROW=$(PSQL "SELECT world_id||'|'||entity_id FROM entity_registry WHERE canonical_name='Player' LIMIT 1")
WORLD=${PLAYER_ROW%%|*}
PLAYER=${PLAYER_ROW#*|}
JONAS=$(PSQL "SELECT entity_id FROM entity_registry WHERE canonical_name='Jonas' AND world_id='$WORLD'")
[ -n "$WORLD" ] && [ -n "$PLAYER" ] && [ -n "$JONAS" ] || { echo "could not read world/Player/Jonas — is the db seeded?" >&2; exit 1; }

VIEWERS="P:$PLAYER J:$JONAS"

save() { # save <name> <sql>  — writes only a non-null (non-empty) result
  local name="$1" sql="$2" val
  val=$(PSQL "$sql")
  # `if` (not `&&`) so a NULL/unperceived result returns 0 — otherwise `set -e` would
  # exit the script at the first entity a viewer cannot perceive (O1.., note-as-Jonas).
  if [ -n "$val" ]; then printf '%s\n' "$val" > "$OUT/$name.json"; fi
}

# --- pages: every entity of each kind, both viewers (NULL/unperceived skipped) ---
while IFS='|' read -r eid kind name; do
  case "$kind" in
    actor)    fn=fn_actor_page ;;
    location) fn=fn_location_page ;;
    artifact) fn=fn_artifact_page ;;
    *) continue ;;
  esac
  safe=${name// /_}
  for vp in $VIEWERS; do
    save "page_${kind}_${safe}_${vp%%:*}" "SELECT $fn('$WORLD','${vp#*:}','$eid')"
  done
done < <(PSQL "SELECT entity_id||'|'||entity_kind||'|'||canonical_name FROM entity_registry
               WHERE world_id='$WORLD' AND entity_kind IN ('actor','location','artifact')
               ORDER BY entity_kind, canonical_name")

# --- index: each kind, both viewers (always non-null envelope) ---
for kind in actor location artifact; do
  for vp in $VIEWERS; do
    save "index_${kind}_${vp%%:*}" "SELECT fn_compendium_index_json('$WORLD','${vp#*:}','$kind')"
  done
done

# --- timeline: both viewers ---
for vp in $VIEWERS; do
  save "timeline_${vp%%:*}" "SELECT fn_timeline('$WORLD','${vp#*:}')"
done

# --- scene/current: assembled in Go (buildScene, core/api/scenehandler.go), not a SQL function —
# there is no fn_* this generator can call directly the way it calls fn_actor_page/fn_timeline
# above. TestGenSceneCurrentPayloads (core/api/scenehandler_test.go) is the Go-side equivalent: gated
# on SCENE_PAYLOAD_DIR (skipped in the normal `go test ./...` suite), it drives the REAL ServeHTTP via
# httptest — the identical call net/http itself makes — against this same seeded db, for both
# viewers, and writes the real response bodies as payloads.
( cd "$(dirname "$0")/../core/api" && \
  SCENE_PAYLOAD_DIR="$OUT" \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
  go test -run '^TestGenSceneCurrentPayloads$' -count=1 -v . )

# --- world_actor/1 + place_author/1: SEAT-CONTRACT schemas (the structured-output LEASH a driver's raw
# response is validated against), not API response envelopes — their additionalProperties:false shape
# leaves no room for a "schema_version" field, so unlike every fn_*-backed payload above these cannot be
# identified by one. TestGenWorldActorPayload/TestGenPlaceAuthorPayload (core/api/schema_payloads_test.go)
# are the Go-side equivalent: gated on SEAT_PAYLOAD_DIR (skipped in the normal `go test ./...` suite),
# they drive the REAL runWorldActor/authorPlaceForLeg call paths — through the fake driver, the CI
# stand-in for an undelivered live model — against the seeded Drowned Lantern play world, and write the
# driver's raw output verbatim as world_actor_1.json/place_author_1.json. ci/schema_contract.py recovers
# the schema id these two carry from that filename (sid_from_filename) rather than a schema_version
# field, since a seat's raw output is never wrapped in one.
( cd "$(dirname "$0")/../core/api" && \
  SEAT_PAYLOAD_DIR="$OUT" \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
  go test -run '^(TestGenWorldActorPayload|TestGenPlaceAuthorPayload)$' -count=1 -v . )

# --- beat_frame/2: assembled by streaming POST /worlds/{w}/beats (core/api/beatsstream.go), one SSE
# frame at a time — no fn_* this generator can call directly, same reasoning as scene/current above.
# TestGenBeatFramePayloads (core/api/beatsstream_test.go) is the Go-side equivalent: gated on
# BEAT_STREAM_PAYLOAD_DIR (skipped in the normal `go test ./...` suite), it drives the REAL
# beatsStreamHandler.ServeHTTP — the identical call net/http itself makes — through a committed beat
# (Player tells Mara about the note, the same fixture beathandler_test.go's happy path uses) and
# writes each frame's RAW bytes (interpretation/narration/scene/journey/result/trace — trace only
# because the driven handler runs debug=true, rung3 Task 4) verbatim.
( cd "$(dirname "$0")/../core/api" && \
  BEAT_STREAM_PAYLOAD_DIR="$OUT" \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
  go test -run '^TestGenBeatFramePayloads$' -count=1 -v . )

# manifest: the viewers we generated for, so the validator can ENFORCE viewer coverage
# (both Player and Jonas must appear) rather than trust the generator.
printf '{"viewers":["%s","%s"]}\n' "$PLAYER" "$JONAS" > "$OUT/_manifest.json"

echo "generated $(find "$OUT" -name '*.json' -not -name '_*' | wc -l | tr -d ' ') payload(s) in $OUT"
