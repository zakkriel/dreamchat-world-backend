#!/usr/bin/env bash
# OBS-1: how often does gameplay alter an entity while a world build would still be running?
#
#   ci/obs_alteration_window.sh            # against the local dev database
#   DATABASE_URL=... ci/obs_alteration_window.sh
#
# Read docs/observability/product-observability.md before quoting the output. In particular this number
# OVER-COUNTS: it counts any canon event naming an entity, including someone merely learning a name,
# which is irrelevant to whether a later tier can safely author. It is an upper bound on a risk whose
# real size needs the events split by whether they changed PRESENT state.
#
# No migration and no instrumentation: genesis writes canon with origin 'fast_path' ('template' for
# templated worlds) and everything else is gameplay, so this is derivable and retroactive.
set -uo pipefail
cd "$(dirname "$0")/.."

SQL=$(cat <<'ENDSQL'
WITH play AS (
  SELECT w.world_id,
         left(w.display_name, 24) AS name,
         ep.entity_id,
         extract(epoch FROM (min(ce.accepted_at) - w.created_at)) AS delay_s
  FROM world w
  JOIN canon_event ce
    ON ce.world_id = w.world_id
   AND ce.origin NOT IN ('fast_path', 'template')
  JOIN event_participant ep ON ep.event_id = ce.event_id
  GROUP BY w.world_id, w.display_name, ep.entity_id
)
SELECT name                                              AS world,
       count(*) FILTER (WHERE delay_s <=   60)           AS within_1min,
       count(*) FILTER (WHERE delay_s <=  300)           AS within_5min,
       count(*) FILTER (WHERE delay_s <= 1140)           AS within_19min,
       count(*)                                          AS ever,
       round(min(delay_s)::numeric)                      AS first_touch_s
FROM play
GROUP BY name
ORDER BY within_19min DESC, ever DESC;
ENDSQL
)

echo "== OBS-1 gameplay alteration inside the build window"
echo "   upper bound only — see docs/observability/product-observability.md"
echo

if [ -n "${DATABASE_URL:-}" ]; then
  # A remote database, reached through whatever psql is available.
  if command -v psql >/dev/null 2>&1; then
    printf '%s\n' "$SQL" | psql "$DATABASE_URL" -v ON_ERROR_STOP=1
  else
    printf '%s\n' "$SQL" | docker exec -i dreamchat-world-backend-db-1 psql "$DATABASE_URL" -v ON_ERROR_STOP=1
  fi
else
  printf '%s\n' "$SQL" | docker compose exec -T db psql -U postgres -d dreamchat -v ON_ERROR_STOP=1
fi

echo
echo "Watch for alterations appearing inside within_1min. That would mean the safe window has closed"
echo "and the freshness rule for tiered creation has to be strict rather than best-effort."
