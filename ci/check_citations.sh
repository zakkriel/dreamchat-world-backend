#!/usr/bin/env bash
# check_citations.sh — every rule id a PR body cites must exist.
#
#   ci/check_citations.sh <file>     read the body from a file
#   ci/check_citations.sh -          read the body from stdin
#   ci/check_citations.sh --selftest prove the check can fail
#
# ONE assertion: EVERY identifier cited RESOLVES. `D-99` fails. `ADR-P007` fails. `SPEC-999` fails.
# A body that cites nothing PASSES — there is nothing to resolve.
#
# WHY THIS IS THE ONLY MECHANICAL GATE LEFT (2026-08-27 founder ruling; receipts in
# docs/00_workspace/review-test-suite-2026-08-26.md §Q3). The register's standing rule is "Do not
# invent constraints. If you cannot cite a rule ID, an ADR, or a line of code, you do not have a
# constraint — you have a preference." An invented id is an invented constraint, and that is a fact a
# script can establish in 20ms. Whether the cited rule is the RIGHT one is semantic and belongs to the
# domain-trained reviewer (harness/roles/area-expert.md) — you can prove nobody invented a rule id,
# you cannot prove they obeyed one.
#
# The MANDATORY-CITE half was cut the same day, with its waiver. It required at least one id, but the
# id universe is 100+ ids and none is correlated with the diff, so `Rules: B-1` passed a change to the
# Makefile. A tax that any four characters satisfy teaches pasting, not reading.
#
# KNOWN FALSE-POSITIVE CLASS, stated rather than hidden: an id QUOTED in order to say it does not (or
# no longer) exist is indistinguishable from an id CLAIMED. `docs/adr/ADR-W002` names `ADR-P002` to
# record that the backend never allocated it; a doc-wide sweep flags that as unresolved. This gate is
# scoped to PR BODIES, where the distinction does not arise — nobody cites a retired id as authority
# for a change. Do not "fix" it by loosening the resolver; that would make every invented id pass.
set -uo pipefail

# Resolve a file argument BEFORE cd'ing, or a relative path silently resolves against the repo root
# instead of the caller's cwd — `cat` fails, the body is empty, and the check reports "cites nothing".
# A gate that reads nothing and blames you for it is worse than one that crashes.
if [ "${1:-}" != "" ] && [ "$1" != "-" ] && [ "$1" != "--selftest" ]; then
  case "$1" in /*) ;; *) set -- "$PWD/$1" ;; esac
fi
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

REGISTER=docs/00_strategy/06_rules_register.md
ADR_DIR=docs/30_architecture/adr
ENGINE_ADRS=docs/30_architecture/canon_engine/02_world_state_adrs.md
SPECS=docs/open-spec-items.md

# An id is real only where the doc DEFINES it, never where it is merely mentioned. Each of the four
# series is defined in its own shape, and each resolver matches exactly that shape — so citing a rule
# that some other rule's prose happens to name does not count as citing the register.
#
#   B/C/D/E/F/GA   a table row whose FIRST cell is the id      | D-8 | Synchronous path stays … |
#   I-1…I-10       the one enumeration line in Part A          **Invariants (CI-enforced):** I-1 … · I-2 …
#   ADR-P###       a filename in the platform ADR directory    ADR-P021_art_is_reconciled_….md
#   ADR-###        a heading in the frozen engine ADR doc       ## ADR-029 — Phase 0 splits into …
#   SPEC-###       a heading in the open-spec ledger            ## SPEC-011 — standing payload↔schema …
rule_exists() { grep -qE "^\|[[:space:]]*$1[[:space:]]*\|" "$REGISTER"; }
inv_exists()  { grep -E '^\*\*Invariants' "$REGISTER" | grep -qE "\b$1\b"; }
padr_exists()  { ls "$ADR_DIR" 2>/dev/null | grep -q "^${1}[_-]"; }
eadr_exists()  { grep -qE "^#+[[:space:]]*$1\b" "$ENGINE_ADRS" 2>/dev/null; }
spec_exists()  { grep -qE "^#+[[:space:]]*$1\b" "$SPECS" 2>/dev/null; }

check_body() { # check_body <body-text> ; echoes findings, returns 0 ok / 1 fail
  local body="$1" id kind bad=0 found=0 seen=""

  for id in $(printf '%s' "$body" | grep -oE '\b(ADR-P[0-9]{3}|ADR-[0-9]{3}|SPEC-[0-9]{3}|GA-[0-9]{1,2}|[BCDEFI]-[0-9]{1,2})\b' | sort -u); do
    case " $seen " in *" $id "*) continue ;; esac
    seen="$seen $id"
    found=$((found + 1))
    case "$id" in
      ADR-P*) kind="platform ADR"; padr_exists "$id" || { echo "FAIL  $id — no such file in $ADR_DIR/"; bad=1; continue; } ;;
      ADR-*)  kind="engine ADR";   eadr_exists "$id" || { echo "FAIL  $id — not defined in $ENGINE_ADRS"; bad=1; continue; } ;;
      SPEC-*) kind="spec item";    spec_exists "$id" || { echo "FAIL  $id — not defined in $SPECS"; bad=1; continue; } ;;
      I-*)    kind="invariant";    inv_exists  "$id" || { echo "FAIL  $id — not one of the invariants named in $REGISTER"; bad=1; continue; } ;;
      *)      kind="rule";         rule_exists "$id" || { echo "FAIL  $id — not defined in $REGISTER"; bad=1; continue; } ;;
    esac
    echo "OK    $id ($kind)"
  done

  # Citing nothing is not a finding. The gate resolves what is there; it does not levy a keyword tax.
  if [ "$found" -eq 0 ]; then
    echo "OK    no identifiers cited — nothing to resolve"
  fi
  return "$bad"
}

# ------------------------------------------------------------------------------------ selftest
#
# A gate you have not tried to fool is not a gate. The first version of branch-currency.yml reported
# SUCCESS on a branch cut twelve commits behind main. So: prove each assertion can fail, and prove
# the happy path passes.
if [ "${1:-}" = "--selftest" ]; then
  fails=0
  probe() { # probe <expect pass|fail> <label> <body>
    local want=$1 label=$2 body=$3 got
    if check_body "$body" >/dev/null 2>&1; then got=pass; else got=fail; fi
    if [ "$got" = "$want" ]; then
      printf '  ok   %-46s (%s)\n' "$label" "$got"
    else
      printf '  FAIL %-46s wanted %s, got %s\n' "$label" "$want" "$got"; fails=$((fails + 1))
    fi
  }

  probe pass "empty body"                    ""
  probe pass "prose with no citation"        "Fixes the thing that was broken. Tested locally."
  probe fail "invented rule id"              "Per D-99 the payload must be flat."
  probe fail "invented platform ADR"         "Follows ADR-P007."
  probe fail "invented spec"                 "Closes SPEC-999."
  probe fail "real id shape, wrong series"   "Per GA-99 terms must be generic."
  probe pass "one real rule"                 "Perception-bound per B-1; no canon rows cross the API."
  probe pass "real rule + real platform ADR" "Art is reconciled (ADR-P021), not commissioned. Async per D-8."
  probe pass "real spec item"                "Closes SPEC-011 by capturing a payload for the new schema."
  probe fail "one real id, one invented"     "Per B-1 and D-77, the page renders from perception."
  probe fail "invented invariant"            "Guaranteed by I-77."
  probe fail "invented engine ADR"           "Decided in ADR-888."
  probe pass "real invariant + engine ADR"   "Replay stays invariant (I-1); phase split per ADR-029."
  probe pass "all four series at once"       "B-1 + I-3 at the API boundary, ADR-P020 on boot, closes SPEC-011."

  echo
  if [ "$fails" -eq 0 ]; then echo "SELFTEST OK — every assertion can fail, and the happy paths pass."; exit 0; fi
  echo "SELFTEST FAIL — $fails probe(s) did not behave as specified. The gate is not trustworthy."
  exit 1
fi

# ---------------------------------------------------------------------------------------- main
[ "$#" -eq 1 ] || { echo "usage: $0 <file>|-|--selftest" >&2; exit 2; }
if [ "$1" = "-" ]; then body="$(cat)"; else body="$(cat "$1")"; fi

echo "Citations found in the PR body:"
if check_body "$body"; then
  echo
  echo "citations OK"
  exit 0
fi

cat >&2 <<'EOF'

::error::A rule id cited in this PR body does not exist. An invented id is an invented constraint.

  rules    docs/00_strategy/06_rules_register.md          B-*, C-*, D-*, E-*, F-*, GA-*, I-*
  ADRs     docs/30_architecture/adr/                      ADR-P###
           docs/30_architecture/canon_engine/02_...md     ADR-### (engine)
  specs    docs/open-spec-items.md                        SPEC-###

Cite what you actually read, or cite nothing — a body that cites nothing passes this gate.

Run it locally before you push:  ci/check_citations.sh - <<'BODY' ... BODY
EOF
exit 1
