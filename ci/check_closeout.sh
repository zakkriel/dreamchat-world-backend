#!/usr/bin/env bash
# check_closeout.sh — a round does not end until the docs it invalidated moved, and the reviewer
# who comes next knows what this round learned.
#
#   ci/check_closeout.sh <body-file> <base-ref>
#   ci/check_closeout.sh <body-file> --paths f1 f2 …
#   ci/check_closeout.sh --selftest
#
# WHY. `system_map.md` §7 has listed this exact gate as missing since the map was written:
#
#     Amend this map in the PR that changes its shape
#       | a CI check that a PR touching art*.go, worldgenesis*, image*, prompts/, schemaversion*
#         also touches system_map.md
#
# and `round-protocol.md` §6 says outright "it is on you, and it is listed as an open gate." This is
# that gate. A map amended in a later round is a map that was wrong in production, and the failure log
# has the receipts: AGENTS.md claimed all ten invariants ran in CI when six did (#26); it named a
# schema file that had been deleted (#27); a runbook told an operator to expect versions the code no
# longer emitted (#28); §4 of the map itself said 8 seats when there were 9 and 25 schemas when there
# were 28 (#30). Every one of those was a doc left behind by a round that shipped.
#
# TWO OBLIGATIONS, and the second is the one nothing anywhere gates:
#
#   1. SHAPE -> MAP. Touch a file whose shape a doc describes, and that doc moves in the same diff.
#   2. LEARNED -> DOSSIER. State what this round taught, and where it landed. A round that discovers a
#      trap and does not write it down leaves the NEXT reviewer exactly as ignorant as this one was —
#      which is how the same defect ships twice. The 2026-08-11 QA round found 40-of-70 mutation
#      survival and its findings became unversioned files at the workspace root; nobody was trained.
#
# WHAT IT CANNOT DO. It cannot judge whether the lesson is right or the amendment is good. It verifies
# that the claim was made and that the file it names actually changed. That converts silence into a
# statement someone can disagree with, which is the same trade as `Rules: none` and `Areas: none`.
set -uo pipefail

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# glob<TAB>doc-that-must-also-change<TAB>why. Backend-local only: CI checks out this repo alone, so a
# trigger pointing at a workspace doc could never be verified here. The cross-repo half is
# `harness/check.sh closeout`, which can see both trees.
TRIGGERS='core/api/art*.go	docs/30_architecture/system_map.md	§3 is the art pipeline
core/api/worldgenesis*	docs/30_architecture/system_map.md	§2 is the world-creation sequence
core/api/worldkickstart*	docs/30_architecture/system_map.md	§2 is the world-creation sequence
core/api/image*	docs/30_architecture/system_map.md	§3 and §6 describe the image seam
core/api/prompts/*	docs/30_architecture/system_map.md	§4 enumerates every prompt in the system
core/api/schemaversion*	docs/30_architecture/system_map.md	§5 is the deployment/boot-refusal reality
core/api/seatconfig.go	docs/30_architecture/system_map.md	§8 states what adding a seat costs
core/api/schema/*	docs/30_architecture/system_map.md	§4 and §6 describe the published contract
.github/workflows/*	docs/30_architecture/system_map.md	§7 is the enforced/not-enforced ledger — a new gate moves a row
ci/check_*	docs/30_architecture/system_map.md	§7 is the enforced/not-enforced ledger'

changed_has() { printf '%s\n' "$@" | grep -qxF "$DOC"; }

check_closeout() { # check_closeout <body> <path…>
  local body="$1"; shift
  local paths=("$@") bad=0 hit=0 g doc why p matched

  # ---- 1. shape -> map
  while IFS="$(printf '\t')" read -r g doc why; do
    [ -n "$g" ] || continue
    matched=""
    for p in "${paths[@]}"; do
      # shellcheck disable=SC2254
      case "$p" in $g) matched="$p"; break ;; esac
    done
    [ -n "$matched" ] || continue
    hit=1
    if printf '%s\n' "${paths[@]}" | grep -qxF "$doc"; then
      echo "OK    $matched → $doc changed too"
    else
      echo "FAIL  $matched changed but $doc did not"
      echo "        $why"
      bad=1
    fi
  done <<<"$TRIGGERS"
  [ "$hit" -eq 0 ] && echo "OK    no shape-describing file in this diff"

  # ---- 2. learned -> dossier
  # Read the WHOLE wrapped block, not just the first physical line. PR bodies and commit messages wrap
  # at 72-80 columns, so a lesson that names its file on the second line was being read as naming
  # nothing — found by running this gate against its own commit message. The block ends at a blank line
  # or the next `Key:` header.
  local learned
  # `IGNORECASE` is a GNU awk extension — macOS awk silently ignores it, so a capital `Learned:` never
  # matched and the block read empty. Lower-case the line explicitly instead.
  learned="$(printf '%s\n' "$body" | awk '
      { l = tolower($0) }
      on == 0 && l ~ /^[[:space:]]*learned[[:space:]]*:/ { on=1; sub(/^[^:]*:[[:space:]]*/,""); print; next }
      on && l ~ /^[[:space:]]*$/ { exit }
      on && l ~ /^[[:space:]]*[a-z][a-z-]*[[:space:]]*:/ { exit }
      on { print }' | tr '\n' ' ' | sed -E 's/[[:space:]]+/ /g; s/^ | $//g')"
  if [ -z "$learned" ]; then
    echo "FAIL  no 'Learned:' line. A round that teaches nobody anything leaves the next reviewer"
    echo "        exactly as ignorant as this one was."
    return 1
  fi

  if printf '%s' "$learned" | grep -qiE '^(nothing|none)\b'; then
    if printf '%s' "$learned" | grep -qiE '^(nothing|none)[^-—]*[-—][[:space:]]*\S'; then
      [ "${#learned}" -ge 30 ] || { echo "FAIL  'Learned: nothing' needs a real reason: $learned"; return 1; }
      echo "OK    nothing learned, reasoned: $learned"
    else
      echo "FAIL  'Learned: nothing' needs a reason after a dash — a sentence a reviewer can disagree with"
      return 1
    fi
  else
    # Any path it names must actually be in the diff. This is the teeth: you cannot claim to have
    # trained the reviewer without having touched the file that trains them.
    local named n=0
    named="$(printf '%s' "$learned" | grep -oE '(docs|harness)/[A-Za-z0-9_./-]+\.(md|tsv|map|sh)' | sort -u || true)"
    if [ -z "$named" ]; then
      echo "FAIL  'Learned:' states a lesson but names no file where it landed."
      echo "        Point at the dossier, the failure log, the map, or the checklist you updated."
      return 1
    fi
    while IFS= read -r f; do
      [ -n "$f" ] || continue
      n=$((n + 1))
      if printf '%s\n' "${paths[@]}" | grep -qxF "$f"; then
        echo "OK    lesson landed in $f (in this diff)"
      else
        case "$f" in
          docs/areas/*|docs/00_workspace/*|harness/*)
            # A workspace path cannot be verified from a runner that checks out only this repo.
            echo "OK    lesson claims $f — a WORKSPACE path, unverifiable here; ./harness/check.sh closeout checks it" ;;
          *) echo "FAIL  'Learned:' names $f but that file is not in this diff"; bad=1 ;;
        esac
      fi
    done <<<"$named"
  fi
  return "$bad"
}

# ------------------------------------------------------------------------------------- selftest
if [ "${1:-}" = "--selftest" ]; then
  fails=0
  probe() { local want=$1 label=$2 body=$3; shift 3
    local got; if check_closeout "$body" "$@" >/dev/null 2>&1; then got=pass; else got=fail; fi
    if [ "$got" = "$want" ]; then printf '  ok   %-52s (%s)\n' "$label" "$got"
    else printf '  FAIL %-52s wanted %s got %s\n' "$label" "$want" "$got"; fails=$((fails+1)); fi; }

  SM=docs/30_architecture/system_map.md
  L="Learned: portals are still not drawn → docs/areas/art-and-assets.md"
  probe fail "art change, map untouched"       "$L" core/api/artstyle.go
  probe pass "art change, map touched"         "$L" core/api/artstyle.go "$SM"
  probe fail "prompt change, map untouched"    "$L" core/api/prompts/narrate.txt
  probe fail "schema change, map untouched"    "$L" core/api/schema/beat_frame.v5.schema.json
  probe fail "new workflow, ledger untouched"  "$L" .github/workflows/x.yml
  probe pass "new workflow, ledger touched"    "$L" .github/workflows/x.yml "$SM"
  probe fail "new ci gate, ledger untouched"   "$L" ci/check_x.sh
  probe pass "unrelated file, no trigger"      "$L" core/api/journey.go
  probe fail "no Learned line at all"          "Fixes a thing." core/api/journey.go
  probe fail "Learned with no file named"      "Learned: portals are not drawn" core/api/journey.go
  probe fail "Learned names a backend file not in the diff" \
       "Learned: x → docs/open-spec-items.md" core/api/journey.go
  probe pass "Learned names a backend file in the diff" \
       "Learned: x → docs/open-spec-items.md" core/api/journey.go docs/open-spec-items.md
  probe pass "Learned names a workspace file"  "$L" core/api/journey.go
  probe fail "bare 'nothing'"                  "Learned: nothing" core/api/journey.go
  probe fail "'nothing' with a one-word reason" "Learned: nothing — typo" core/api/journey.go
  probe pass "'nothing' properly reasoned" \
       "Learned: nothing — a comment typo; no behaviour, no shape, no new trap" core/api/journey.go
  probe pass "wrapped Learned, file on line 2" \
       "Learned: a working-tree gate across repos with different cadences
is structurally doomed → docs/open-spec-items.md" core/api/journey.go docs/open-spec-items.md
  probe fail "wrapped Learned, named file not in diff" \
       "Learned: something long enough to wrap onto
a second line → docs/open-spec-items.md" core/api/journey.go
  probe pass "wrapped 'nothing' reason" \
       "Learned: nothing — this only reflows a comment
and changes no behaviour at all" core/api/journey.go

  echo
  [ "$fails" -eq 0 ] && { echo "SELFTEST OK — every assertion can fail, and the happy paths pass."; exit 0; }
  echo "SELFTEST FAIL — $fails probe(s) misbehaved."; exit 1
fi

# ----------------------------------------------------------------------------------------- main
[ "$#" -ge 2 ] || { echo "usage: $0 <body-file> <base-ref> | <body-file> --paths <f>…" >&2; exit 2; }
BODY_FILE="$1"; shift
case "$BODY_FILE" in /*) ;; *) BODY_FILE="$PWD/$BODY_FILE" ;; esac
body="$(cat "$BODY_FILE")"

if [ "$1" = "--paths" ]; then
  shift; paths=("$@")
  [ "${#paths[@]}" -gt 0 ] || { echo "FAIL  --paths given nothing — refusing to pass by seeing nothing" >&2; exit 1; }
else
  base="$1"; git fetch origin "$base" --quiet 2>/dev/null || true
  paths=()
  while IFS= read -r l; do [ -n "$l" ] && paths+=("$l"); done < <(
    git diff --name-only "origin/$base...HEAD" 2>/dev/null || git diff --name-only "$base...HEAD" 2>/dev/null || true )
  [ "${#paths[@]}" -gt 0 ] || { echo "FAIL  could not compute a diff against '$base' — refusing to pass by seeing nothing" >&2; exit 1; }
fi

echo "Close-out:"
if check_closeout "$body" "${paths[@]}"; then echo; echo "closeout OK"; exit 0; fi

cat >&2 <<'EOF'

::error::A round is not finished until the docs it invalidated moved and the next reviewer is trained.

  1. Touch a file whose shape a doc describes -> that doc moves in the SAME diff.
     system_map.md §7 has listed this gate as missing since the map was written.

  2. Say what this round taught, and where it landed:

       Learned: <the lesson> -> <path to the dossier / failure log / map you updated>
       Learned: nothing — <why, in a sentence>

     A round that finds a trap and does not write it down leaves the next reviewer as ignorant as
     this one was. That is how the same defect ships twice.

  ../harness/review.sh   prints the close-out obligations for your diff
  ../docs/00_workspace/round-protocol.md §6   the full amend-in-the-same-round table
EOF
exit 1
