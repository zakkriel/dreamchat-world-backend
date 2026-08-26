#!/usr/bin/env bash
# ci/mutate.sh — the mechanical opening move, as a command.
#
# WHY THIS EXISTS
#
# `round-protocol.md` §7 says a review opens by reverting the change and checking that a test fails.
# That instruction sat in a document for a day and was obeyed exactly once — by hand, by the author,
# on their own diff. Then 17 of 17 subagent reviewers died on transport (failure-log #44) and the
# instruction had no runner at all.
#
# An instruction with no runner is advice. This is the runner. It is deliberately not an agent: it
# needs no model, no network, and no judgement, and it is the single highest-value part of a review —
# it answers "is this guard real?" mechanically, which is the one review question that never needs an
# opinion.
#
# A SURVIVING MUTANT IS A FAILING BUILD. If a test suite passes with the behaviour deleted, that suite
# does not defend that behaviour. That is not a style opinion; it is the definition.
#
# TWO CLASSES OF MUTANT, AND EVERYONE FORGETS THE SECOND
#
#   CODE-PATH mutants  — break the implementation. Delete a branch, defang a guard, invert a
#                        condition. These are the obvious ones and they are what people run.
#
#   INPUT-SHAPE mutants — feed the implementation something it did not expect. Absent, null, the
#                        wrong JSON type, an empty array, a duplicate, an id that does not exist.
#                        You cannot express these as a sed script on the source, so run them as
#                        assertions in the suite instead — but reason about them at the SAME moment,
#                        because they are the ones that get skipped.
#
# Measured, on this repo: seven code-path mutants across the SPEC-034 and SPEC-035 rounds were all
# CAUGHT, and the suite still shipped with a silent drop on malformed input — `witnesses: "<uuid>"`
# as a bare string committed with zero witnesses and no halt_reason. Every mutant asked "what if the
# code is wrong". None asked "what if the INPUT is wrong". A 100%-caught mutation report is not
# coverage of the second class; it is silence about it.
#
# So: for every field a change READS, name what happens when it is absent, null, the wrong type, and
# empty. Four questions, asked out loud, per field.
#
# USAGE
#
#   ci/mutate.sh --file <path> --test '<command>' \
#                --mutant '<label>::<sed-script>' [--mutant ...]
#   ci/mutate.sh --selftest
#
# The file is backed up, each mutant applied with `sed -i` in turn, the test command run, and the file
# restored — including on interrupt. Exit 0 only if every mutant was CAUGHT.
#
# HOW TO READ THE VERDICTS
#   CAUGHT   — the suite went red. The guard is real.
#   SURVIVED — the suite stayed green with the behaviour broken. The guard is theatre. Build fails.
#   NO-OP    — the mutant did not change the file. A typo in your sed script, not a result. Build fails,
#              because a mutation experiment that silently tested nothing is worse than none: it is a
#              green light you did not earn.

set -euo pipefail

MUT_BAK=""
MUT_FILE=""

FILE=""
TEST_CMD=""
MUTANTS=()
SELFTEST=no

while [ $# -gt 0 ]; do
  case "$1" in
    --file)     FILE="$2"; shift 2 ;;
    --test)     TEST_CMD="$2"; shift 2 ;;
    --mutant)   MUTANTS+=("$2"); shift 2 ;;
    --selftest) SELFTEST=yes; shift ;;
    -h|--help)  sed -n '2,36p' "$0"; exit 0 ;;
    *) echo "mutate: unknown argument '$1'" >&2; exit 2 ;;
  esac
done

# ── the engine ──────────────────────────────────────────────────────────────────────────────────────
run_experiment() {
  local file="$1" test_cmd="$2"; shift 2
  local mutants=("$@")
  local bak caught=0 survived=0 noop=0

  [ -f "$file" ] || { echo "mutate: no such file: $file" >&2; return 2; }
  [ ${#mutants[@]} -gt 0 ] || { echo "mutate: no --mutant given; refusing to pass vacuously" >&2; return 2; }

  # Restore on ANY exit, including Ctrl-C. A mutation tool that can leave a mutant in the tree is a
  # tool that can commit one. These are script-level globals on purpose: an EXIT trap set inside a
  # function still fires at SCRIPT exit, long after the function's locals are gone, and referencing
  # them there dies under `set -u`. Found by running --selftest.
  MUT_BAK="$(mktemp)"; MUT_FILE="$file"
  cp "$file" "$MUT_BAK"
  bak="$MUT_BAK"
  trap 'cp "$MUT_BAK" "$MUT_FILE" 2>/dev/null || true; rm -f "$MUT_BAK"' EXIT INT TERM

  echo "── baseline ──────────────────────────────────────────────────────────────────"
  if ! eval "$test_cmd" >/dev/null 2>&1; then
    echo "  BASELINE RED — the suite fails before any mutation. Fix that first; a mutation"
    echo "  experiment on a red baseline measures nothing."
    return 1
  fi
  echo "  green"
  echo

  local spec label script
  for spec in "${mutants[@]}"; do
    label="${spec%%::*}"
    script="${spec#*::}"
    if [ "$label" = "$spec" ]; then
      echo "mutate: malformed --mutant (need 'label::sed-script'): $spec" >&2
      return 2
    fi

    cp "$bak" "$file"
    sed -i '' "$script" "$file" 2>/dev/null || sed -i "$script" "$file"

    if cmp -s "$bak" "$file"; then
      printf '  %-12s %-52s\n' "NO-OP" "$label"
      echo "               the sed script changed nothing — this tested NOTHING"
      noop=$((noop + 1))
      continue
    fi

    if eval "$test_cmd" >/dev/null 2>&1; then
      printf '  %-12s %-52s\n' "SURVIVED" "$label"
      survived=$((survived + 1))
    else
      printf '  %-12s %-52s\n' "CAUGHT" "$label"
      caught=$((caught + 1))
    fi
  done

  cp "$bak" "$file"
  echo
  echo "── restored, re-checking baseline ────────────────────────────────────────────"
  if ! eval "$test_cmd" >/dev/null 2>&1; then
    echo "  BASELINE RED AFTER RESTORE — the file was not put back cleanly. Check git status."
    return 1
  fi
  echo "  green"
  echo
  printf '  %d caught, %d survived, %d no-op\n' "$caught" "$survived" "$noop"

  if [ "$survived" -gt 0 ]; then
    echo
    echo "  A SURVIVING MUTANT means the suite passes with that behaviour deleted. Either the"
    echo "  assertion that should defend it is missing, or the one that claims to is vacuous."
    return 1
  fi
  if [ "$noop" -gt 0 ]; then
    echo
    echo "  A NO-OP mutant tested nothing. Fix the sed script; do not read this run as a pass."
    return 1
  fi
  echo "  MUTATION OK — every named behaviour is defended by at least one assertion."
  return 0
}

# ── selftest: the tool must be able to fail, and must be able to notice a vacuous test ─────────────
if [ "$SELFTEST" = yes ]; then
  tmpd="$(mktemp -d)"
  trap 'rm -rf "$tmpd"' EXIT
  subject="$tmpd/subject.txt"
  probe="grep -q GUARDED '$subject'"
  fails=0

  printf 'GUARDED\nunrelated line\n' > "$subject"

  echo "=== probe 1: a mutation that breaks the asserted behaviour must be CAUGHT"
  if run_experiment "$subject" "$probe" 'breaks the guard::s/GUARDED/BROKEN/' >/dev/null 2>&1; then
    echo "  ok"
  else
    echo "  FAIL — a real break was not reported as caught"; fails=$((fails + 1))
  fi

  echo "=== probe 2: a mutation the test does not notice must SURVIVE and fail the run"
  if run_experiment "$subject" "$probe" 'untested line::s/unrelated line/gutted/' >/dev/null 2>&1; then
    echo "  FAIL — a surviving mutant passed the run"; fails=$((fails + 1))
  else
    echo "  ok"
  fi

  echo "=== probe 3: a sed script that changes nothing must be NO-OP, not a pass"
  if run_experiment "$subject" "$probe" 'matches nothing::s/NOT_PRESENT_ANYWHERE/x/' >/dev/null 2>&1; then
    echo "  FAIL — a no-op mutant passed the run"; fails=$((fails + 1))
  else
    echo "  ok"
  fi

  echo "=== probe 4: no --mutant at all must REFUSE, never pass vacuously"
  if run_experiment "$subject" "$probe" >/dev/null 2>&1; then
    echo "  FAIL — an empty experiment passed"; fails=$((fails + 1))
  else
    echo "  ok"
  fi

  echo "=== probe 4b: a mutant with no 'label::' prefix must REFUSE, not run unlabelled"
  if run_experiment "$subject" "$probe" 's/GUARDED/BROKEN/' >/dev/null 2>&1; then
    echo "  FAIL — a malformed mutant was accepted"; fails=$((fails + 1))
  else
    echo "  ok"
  fi

  echo "=== probe 5: a red baseline must be reported as such, not as a caught mutant"
  printf 'nothing here\n' > "$subject"
  if run_experiment "$subject" "$probe" 'on a red baseline::s/nothing/something/' >/dev/null 2>&1; then
    echo "  FAIL — a red baseline passed"; fails=$((fails + 1))
  else
    echo "  ok"
  fi

  echo "=== probe 6: the subject file must be restored after every run"
  printf 'GUARDED\nunrelated line\n' > "$subject"
  before="$(cat "$subject")"
  run_experiment "$subject" "$probe" 'breaks the guard::s/GUARDED/BROKEN/' >/dev/null 2>&1 || true
  if [ "$before" = "$(cat "$subject")" ]; then
    echo "  ok"
  else
    echo "  FAIL — the file was left mutated"; fails=$((fails + 1))
  fi

  echo
  if [ "$fails" -eq 0 ]; then
    echo "SELFTEST OK — the tool catches real breaks, and refuses vacuous, no-op and red-baseline runs."
    exit 0
  fi
  echo "SELFTEST FAIL — $fails probe(s) misbehaved. Do not trust a verdict from this script."
  exit 1
fi

[ -n "$FILE" ] && [ -n "$TEST_CMD" ] || { sed -n '2,36p' "$0"; exit 2; }
run_experiment "$FILE" "$TEST_CMD" "${MUTANTS[@]}"
