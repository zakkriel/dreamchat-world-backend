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

WORLD=$(PSQL "SELECT world_id FROM entity_registry LIMIT 1")
PLAYER=$(PSQL "SELECT entity_id FROM entity_registry WHERE canonical_name='Player'")
JONAS=$(PSQL "SELECT entity_id FROM entity_registry WHERE canonical_name='Jonas'")
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

echo "generated $(find "$OUT" -name '*.json' | wc -l | tr -d ' ') payload(s) in $OUT"
