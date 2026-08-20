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

# --- carrying (carrying/1): both fixture viewers, PLUS the play world's two carriers.
# The fixture world's Player and Jonas carry nothing, so on their own they would only ever exercise
# the empty envelope and the item shape would ship unvalidated — a coverage hole that reads green.
# The Drowned Lantern's Kade and Mara each hold exactly one thing, so those two payloads are what
# actually put `carried[*]` in front of the validator. Anchored on canonical_name, same as $WORLD
# above, never a LIMIT-1-any-world.
for vp in $VIEWERS; do
  save "carrying_${vp%%:*}" "SELECT fn_carrying('$WORLD','${vp#*:}')"
done
DL=$(PSQL "SELECT world_id FROM world WHERE display_name='The Drowned Lantern'")
if [ -n "$DL" ]; then
  for who in Kade Mara; do
    eid=$(PSQL "SELECT entity_id FROM entity_registry WHERE world_id='$DL' AND canonical_name='$who'")
    [ -n "$eid" ] && save "carrying_dl_$who" "SELECT fn_carrying('$DL','$eid')"
  done
fi

# --- transcript (transcript/2): the ONE surface with no seeded rows, because history starts when
# the feature ships and old beats are never fabricated. Left alone it would only ever produce the
# empty envelope and every `entries[*]` field — the whole delivered-narration shape, including the
# narration/3 speech split — would ship unvalidated: a coverage hole that reads green. So the
# generator writes two entries of its own first (a narration+speech beat, and a journey beat with no
# player input), then pages them: page 1 with limit=1 exercises `next_before`, page 2 exercises the
# cursor, and the other viewer exercises the empty envelope AND the viewer-scoping.
PSQL "INSERT INTO transcript_entry (world_id, viewer_id, in_world_tick, stated, segments, halt_reason, journey)
      VALUES
      ('$WORLD','$PLAYER',10,'I ask her about the note',
       '[{\"speaker_id\":null,\"speaker_label\":\"\",\"kind\":\"narration\",\"text\":\"The common room is low and dim.\",\"quote\":null},
         {\"speaker_id\":\"$JONAS\",\"speaker_label\":\"the muscle by the bar\",\"kind\":\"speech\",\"text\":\"he looks up from the tap\",\"quote\":\"The tide turns at dusk.\",\"emotion\":\"happy\"},
         {\"speaker_id\":\"$JONAS\",\"speaker_label\":\"the muscle by the bar\",\"kind\":\"action\",\"text\":\"he sets a tankard down.\",\"quote\":null}]'::jsonb,
       'completed', NULL),
      ('$WORLD','$PLAYER',12,NULL,
       '[{\"speaker_id\":null,\"speaker_label\":\"\",\"kind\":\"narration\",\"text\":\"You walk on.\",\"quote\":null}]'::jsonb,
       'journey', '{\"arrived\":false}'::jsonb)" >/dev/null
save "transcript_P_page1" "SELECT fn_transcript('$WORLD','$PLAYER',NULL,1)"
NEXT=$(PSQL "SELECT (fn_transcript('$WORLD','$PLAYER',NULL,1)->>'next_before')")
[ -n "$NEXT" ] && save "transcript_P_page2" "SELECT fn_transcript('$WORLD','$PLAYER',$NEXT,50)"
save "transcript_J_empty" "SELECT fn_transcript('$WORLD','$JONAS')"

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

# --- image_regenerate/1: the regenerate button's response envelope, captured through the composed
# router exactly as a browser receives it (the handler alone could be perfect and unrouted).
( cd "$(dirname "$0")/../core/api" && \
  IMAGE_PAYLOAD_DIR="$OUT" \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
  go test -run '^TestGenImageRegeneratePayload$' -count=1 -v . )

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

# --- world_directory/1 + world_created/1 (SPEC-028): GET /worlds is fn-backed, but POST /worlds is a
# Go response envelope, and both should be exercised through the REAL handler rather than half here
# and half in SQL. TestGenWorldPayloads (core/api/worldshandler_test.go) drives both via httptest,
# gated on WORLD_PAYLOAD_DIR, and deletes the world it creates so the directory payload keeps listing
# exactly the seeded worlds instead of accumulating one fixture per CI run.
( cd "$(dirname "$0")/../core/api" && \
  WORLD_PAYLOAD_DIR="$OUT" \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
  go test -run '^TestGenWorldPayloads$' -count=1 -v . )

# --- world creation (PRD: prd_world_creation.md) — four schemas, two kinds of contract.
# world_genesis/1 and world_interview/1 are SEAT leashes captured byte-identically from what the driver
# returned (filename-keyed, like world_actor_1.json). world_genesis_frame/1 and world_interview_turn/1 are
# API contracts captured by driving the REAL handlers through httptest, so what CI validates is what a
# browser receives — SSE framing and field-omission included. Unlike TestGenWorldPayloads this one CANNOT
# clean up after itself: canon tables carry forbid_delete triggers, so the world it builds stays in the
# throwaway CI database.
( cd "$(dirname "$0")/../core/api" && \
  GENESIS_PAYLOAD_DIR="$OUT" \
  DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
  go test -run '^TestGenSeatContractPayloads$|^TestGenGenesisAPIPayloads$' -count=1 -v . )

# manifest: the viewers we generated for, so the validator can ENFORCE viewer coverage
# (both Player and Jonas must appear) rather than trust the generator.
printf '{"viewers":["%s","%s"]}\n' "$PLAYER" "$JONAS" > "$OUT/_manifest.json"

echo "generated $(find "$OUT" -name '*.json' -not -name '_*' | wc -l | tr -d ' ') payload(s) in $OUT"
