#!/usr/bin/env bash
# check_round.sh — a PR must declare the areas it touches, and the declaration must match the diff.
#
#   ci/check_round.sh <body-file> <base-ref>   compute areas from the diff, check the body
#   ci/check_round.sh <body-file> --paths f1 f2 …
#   ci/check_round.sh --selftest
#
# THIS IS THE REGULATION. Everything else in the harness routes and advises; this is the one check that
# makes the routing unskippable. `harness/check.sh areas` proves every path HAS an owner. This proves
# the author KNEW who it was.
#
# Two required lines in the PR body:
#
#     Areas: perception, play-loop
#     Reviewed-by: perception-expert, play-loop-expert
#
# `Areas` is computed from `git diff --name-only <base>...HEAD`, mapped through
# docs/30_architecture/AREAS.map, and must match as a SET. You cannot quietly touch an area: the diff
# names it whether you do or not.
#
# `Reviewed-by` cannot be verified — no script knows whether a review happened. What it does is make
# the claim EXPLICIT and attributable. A false line is a lie somebody can catch later; a missing line
# was the previous status quo, where skipping the reviewer left no trace at all. That asymmetry is the
# whole value, and it is the same reasoning as `Rules: none — <reason>` in check_citations.sh: an
# escape hatch a reviewer can disagree with beats a silent exemption.
#
# Deliberately NOT asserted: that the review was good, or that the right person did it. Those are
# judgements. This gate only removes "I didn't know which area this was" as an available excuse.
set -uo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAP=docs/30_architecture/AREAS.map

# --- area lookup -----------------------------------------------------------------------------------

areas_for() { # areas_for <path> -> newline-separated area names
  awk -v target="$1" '
    /^[[:space:]]*#/ { next }
    /^[^ \t]/        { area=$0; next }
    /^[ \t]/ {
      if (area == "" || area ~ /^!/) next
      g=$0; sub(/^[ \t]+/,"",g); sub(/[ \t]*#.*$/,"",g)
      if (g=="") next
      # shell-style glob match, done in awk: turn the glob into a regex
      pat=g
      gsub(/\./ , "\\.", pat); gsub(/\*/, ".*", pat); gsub(/\?/, ".", pat)
      if (target ~ ("^" pat "$")) print area
    }' "$MAP" | sort -u
}

unowned_paths() {
  awk '/^!unowned$/{on=1;next} /^[^ \t!]/{on=0} on&&/^[ \t]/{sub(/^[ \t]+/,"");sub(/[ \t]*#.*$/,"");if($0!="")print}' "$MAP"
}

all_areas() { grep -vE '^[[:space:]]*(#|$)' "$MAP" | grep -E '^[^ 	!]' | sort -u; }

compute_areas() { # compute_areas <path>… -> sorted unique area list
  local p out=""
  for p in "$@"; do
    case "$p" in docs/*|.superpowers/*) continue ;; esac   # routed by dossiers, not by glob
    unowned_paths | grep -qxF "$p" && continue
    out="$out$(areas_for "$p")
"
  done
  printf '%s' "$out" | grep -v '^$' | sort -u
}

# --- the check -------------------------------------------------------------------------------------

check_round() { # check_round <body> <path…>  -> 0 ok / 1 fail
  local body="$1"; shift
  local want got bad=0 declared reviewers

  want="$(compute_areas "$@")"

  declared="$(printf '%s' "$body" \
    | grep -iE '^[[:space:]]*areas?[[:space:]]*:' | head -n1 \
    | sed -E 's/^[^:]*:[[:space:]]*//' )"
  reviewers="$(printf '%s' "$body" \
    | grep -iE '^[[:space:]]*reviewed[-_ ]?by[[:space:]]*:' | head -n1 \
    | sed -E 's/^[^:]*:[[:space:]]*//' )"

  # No backend code touched: an explicit, reasoned waiver, or nothing to check.
  if [ -z "$want" ]; then
    if printf '%s' "$declared" | grep -qiE '^none[[:space:]]*[-—:][[:space:]]*\S'; then
      [ "${#declared}" -ge 24 ] || { echo "FAIL  'Areas: none' needs a real reason: $declared"; return 1; }
      echo "OK    no backend code in this diff; waiver: $declared"
      return 0
    fi
    if [ -n "$declared" ]; then
      echo "FAIL  the diff touches no owned backend path, but the body declares: $declared"
      return 1
    fi
    echo "OK    no owned backend path in this diff, and nothing declared"
    return 0
  fi

  [ -n "$declared" ] || { echo "FAIL  no 'Areas:' line. This diff touches: $(echo $want)"; bad=1; }
  [ -n "$reviewers" ] || { echo "FAIL  no 'Reviewed-by:' line naming who adversaried this change"; bad=1; }

  if [ -n "$declared" ]; then
    got="$(printf '%s' "$declared" | tr ',' '\n' | sed -E 's/^[[:space:]]+|[[:space:]]+$//g' | grep -v '^$' | sort -u)"
    local missing extra
    missing="$(comm -23 <(printf '%s\n' "$want") <(printf '%s\n' "$got") | grep -v '^$' || true)"
    extra="$(comm -13   <(printf '%s\n' "$want") <(printf '%s\n' "$got") | grep -v '^$' || true)"
    [ -n "$missing" ] && { echo "FAIL  the diff touches areas the body does not declare: $(echo $missing)"; bad=1; }
    [ -n "$extra" ]   && { echo "FAIL  the body declares areas the diff does not touch: $(echo $extra)"; bad=1; }
    local a
    for a in $got; do
      all_areas | grep -qxF "$a" || { echo "FAIL  '$a' is not an area in $MAP"; bad=1; }
    done
  fi

  [ "$bad" -eq 0 ] && echo "OK    areas declared and matching: $(echo $want)  ·  reviewed-by: $reviewers"
  return "$bad"
}

# --- selftest --------------------------------------------------------------------------------------

if [ "${1:-}" = "--selftest" ]; then
  fails=0
  probe() { # probe <expect> <label> <body> <path…>
    local want=$1 label=$2 body=$3; shift 3
    local got; if check_round "$body" "$@" >/dev/null 2>&1; then got=pass; else got=fail; fi
    if [ "$got" = "$want" ]; then printf '  ok   %-50s (%s)\n' "$label" "$got"
    else printf '  FAIL %-50s wanted %s, got %s\n' "$label" "$want" "$got"; fails=$((fails+1)); fi
  }
  P=core/api/namingwall.go            # perception
  Q=core/api/orchestrator.go          # play-loop
  R=core/db/migrations/0001_x.sql     # canon-recording (glob), any name

  probe fail "no Areas line at all"            "Fixes a thing." "$P"
  probe fail "Areas but no Reviewed-by"        "Areas: perception" "$P"
  probe fail "wrong area declared"             "Areas: play-loop
Reviewed-by: x" "$P"
  probe fail "one of two areas declared"       "Areas: perception
Reviewed-by: x" "$P" "$Q"
  probe fail "declares an area not touched"    "Areas: perception, art-and-assets
Reviewed-by: x" "$P"
  probe fail "invented area name"              "Areas: telepathy
Reviewed-by: x" "$P"
  probe fail "bare waiver on a real diff"      "Areas: none — nothing to see
Reviewed-by: x" "$P"
  probe pass "single area, both lines"         "Areas: perception
Reviewed-by: perception-expert" "$P"
  probe pass "two areas, order irrelevant"     "Areas: play-loop, perception
Reviewed-by: both" "$P" "$Q"
  probe pass "migration -> canon-recording"    "Areas: canon-recording
Reviewed-by: canon-recording-expert" "$R"
  probe pass "docs-only, reasoned waiver"      "Areas: none — documentation only, no backend code changed
Reviewed-by: n/a" "docs/MASTER_INDEX.md"
  probe fail "docs-only, one-word waiver"      "Areas: none — docs
Reviewed-by: n/a" "docs/MASTER_INDEX.md"
  probe pass "docs-only, nothing declared"     "A doc fix." "docs/MASTER_INDEX.md"

  echo
  [ "$fails" -eq 0 ] && { echo "SELFTEST OK — every assertion can fail, and the happy paths pass."; exit 0; }
  echo "SELFTEST FAIL — $fails probe(s) misbehaved. The gate is not trustworthy."; exit 1
fi

# --- main ------------------------------------------------------------------------------------------

[ "$#" -ge 2 ] || { echo "usage: $0 <body-file> <base-ref> | <body-file> --paths <f>…" >&2; exit 2; }
BODY_FILE="$1"; shift
case "$BODY_FILE" in /*) ;; *) BODY_FILE="$PWD/$BODY_FILE" ;; esac   # same trap as check_citations.sh
body="$(cat "$BODY_FILE")"

if [ "$1" = "--paths" ]; then
  shift; paths=("$@")
  [ "${#paths[@]}" -gt 0 ] || { echo "FAIL  --paths was given no paths — refusing to pass by seeing nothing" >&2; exit 1; }
else
  base="$1"
  git fetch origin "$base" --quiet 2>/dev/null || true
  # No `mapfile`: macOS bash 3.2 lacks it and fails SILENTLY, leaving the array empty — which would
  # make this gate pass every PR by seeing no changed paths at all. The worst possible failure for a
  # gate: green because it looked at nothing.
  paths=()
  while IFS= read -r l; do [ -n "$l" ] && paths+=("$l"); done < <(
    git diff --name-only "origin/$base...HEAD" 2>/dev/null \
      || git diff --name-only "$base...HEAD" 2>/dev/null || true )
  [ "${#paths[@]}" -gt 0 ] || { echo "FAIL  could not compute a diff against '$base' — refusing to pass by seeing nothing" >&2; exit 1; }
fi

echo "Round declaration:"
if check_round "$body" "${paths[@]:-}"; then echo; echo "round OK"; exit 0; fi

cat >&2 <<EOF

::error::A PR must declare the areas it touches, and the declaration must match the diff.

  Areas: <the areas this diff touches, comma-separated>
  Reviewed-by: <who adversaried it>

Which areas a path belongs to is a LOOKUP, never a guess:
  docs/30_architecture/AREAS.map          the globs
  ../docs/00_workspace/areas.md           why the areas are what they are
  ../docs/areas/<area>.md                 the dossier each reviewer loads
  ../harness/review.sh <base>             prints all of it for your diff

No backend code in the diff? Say so where a reviewer can disagree:
  Areas: none — <why, in a sentence>
EOF
exit 1
