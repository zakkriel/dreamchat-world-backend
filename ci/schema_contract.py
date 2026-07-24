#!/usr/bin/env python3
# =============================================================================
# SPEC-011 — published-schema contract check (two-sided). See
# docs/open-spec-items.md (SPEC-011) and docs/superpowers/plans/2026-06-15-spec-011-*.
#
#   Direction 1 (payload -> schema): every REAL payload (produced by the SQL JSON
#     functions, see ci/gen_payloads.sh) validates against its published schema,
#     matched by the payload's own `schema_version`. Catches over-tightening /
#     structural drift.
#   Direction 2 (DDL -> schema nullability): every field sourced from a NOT NULL
#     DDL column is typed non-nullable in the schema (e.g. `confidence` is "number",
#     never ["number","null"]); genuinely-nullable fields stay nullable. This is the
#     direction that catches the SPEC-010 over-permissive class — direction 1 cannot,
#     because a non-null payload validates fine against a nullable schema.
#
#   --selftest: flip `confidence` to nullable in each confidence-bearing schema and
#     assert direction 2 FAILS on the mutant (proves the check is not vacuous).
#
# Pure file-based: reads schemas + payloads from disk; no DB/network. Run it in a
# python container (the Makefile target does); only dependency is `jsonschema`.
# Usage:
#   python schema_contract.py <schema_dir> <payload_dir>   # full two-sided check
#   python schema_contract.py --selftest <schema_dir>      # teeth-test of direction 2
# =============================================================================
import sys, os, json, glob, copy
from jsonschema import Draft7Validator

# Fields whose engine DDL column is NOT NULL (Master DDL doc 03 §1.3,
# migration 20260610090004 perception_record): the schema MUST type them as the
# given scalar, never a list containing "null".
NOTNULL_FIELDS = {
    "confidence": "number",     # REAL NOT NULL DEFAULT 1.0  (the SPEC-010 field)
    "epistemic_type": "string", # TEXT NOT NULL
    "content": "string",        # TEXT NOT NULL
    "occurred_at_tick": "integer",  # <- valid_tick BIGINT NOT NULL
    "perception_id": "string",  # UUID PK NOT NULL
}
# Genuinely nullable in real payloads (withheld name / nullable column): MUST allow
# null, so a future over-tightening is caught here too.
NULLABLE_FIELDS = {"perceived_name", "group_label", "display_label"}

# Input/seat-contract schemas: published for codegen and as the structured-output leash for LLM
# seats, NOT output projection payloads. They have no SQL payload generator by design (beat_chain/*
# is the decompose vocabulary; ruling/1 is the resolve seat's explain-first output; npc_attempts/1
# is the cognition seats' output — the leash, ADR-009/D-1), so the Direction-1 payload-coverage
# check does not apply. They are still loaded (must be valid JSON Schema) and run through
# Direction 2 (vacuous unless they carry a NOT-NULL projection field). Projection schemas remain
# strictly coverage-required: a new projection schema added without a payload still fails.
INPUT_CONTRACT_SCHEMAS = {"beat_chain/1", "beat_chain/2", "ruling/1", "npc_attempts/1"}


def load_schemas(schema_dir):
    by_id = {}
    # All schema versions, not only .v1 — a .v2 file invisible to this check would be a
    # silent coverage hole (the exact class SPEC-011 exists to prevent).
    for f in sorted(glob.glob(os.path.join(schema_dir, "*.schema.json"))):
        s = json.load(open(f))
        by_id[s["$id"]] = (f, s)
    return by_id


def walk_props(node, path="$"):
    """Yield (path, field_name, subschema) for every declared property, recursively
    (descends into nested object `properties` and array `items`)."""
    if not isinstance(node, dict):
        return
    for name, sub in node.get("properties", {}).items():
        yield (f"{path}.{name}", name, sub)
        yield from walk_props(sub, f"{path}.{name}")
    items = node.get("items")
    if isinstance(items, dict):
        yield from walk_props(items, f"{path}[]")


def is_nullable(sub):
    t = sub.get("type")
    return isinstance(t, list) and "null" in t


def check_direction2(sid, schema):
    errs = []
    for path, name, sub in walk_props(schema):
        if name in NOTNULL_FIELDS:
            want = NOTNULL_FIELDS[name]
            if sub.get("type") != want:  # exactly the scalar; no list, no null
                errs.append(f"{sid} {path}: '{name}' is NOT NULL in the DDL and must be "
                            f'type "{want}", got {sub.get("type")!r}')
        if name in NULLABLE_FIELDS and not is_nullable(sub):
            errs.append(f"{sid} {path}: '{name}' is withheld/nullable in real payloads and "
                        f"must allow null, got {sub.get('type')!r}")
    return errs


def run(schema_dir, payload_dir):
    by_id = load_schemas(schema_dir)
    if not by_id:
        return [f"no *.v1.schema.json found in {schema_dir}"], 0, 0, 0, 0
    errs, n_payload = [], 0
    seen_versions, seen_viewers = set(), set()

    # Expected viewers for the COVERAGE assertion, from the generator's manifest.
    expected_viewers = None
    mf = os.path.join(payload_dir, "_manifest.json")
    if os.path.exists(mf):
        expected_viewers = set(json.load(open(mf)).get("viewers", []))

    # Direction 1: real payloads validate against their published schema.
    # (Files starting with "_" are manifests/metadata, not payloads.)
    files = sorted(f for f in glob.glob(os.path.join(payload_dir, "*.json"))
                   if not os.path.basename(f).startswith("_"))
    if not files:
        errs.append(f"no payloads found in {payload_dir} (did ci/gen_payloads.sh run against a seeded db?)")
    for f in files:
        payload = json.load(open(f))
        sid = payload.get("schema_version")
        if sid not in by_id:
            errs.append(f"[dir1] {os.path.basename(f)}: unknown schema_version {sid!r}")
            continue
        validator = Draft7Validator(by_id[sid][1])
        for e in sorted(validator.iter_errors(payload), key=str):
            errs.append(f"[dir1] {os.path.basename(f)} ({sid}): {e.message} at {list(e.path)}")
        seen_versions.add(sid)
        if isinstance(payload.get("viewer_id"), str):
            seen_viewers.add(payload["viewer_id"])
        n_payload += 1

    # COVERAGE (enforced — a subset or single-viewer run MUST fail, not pass vacuously).
    # This is the only thing standing between a regressed generator (it has bitten twice:
    # stdin-eaten loop; set -e on a NULL payload) and a false green.
    for sid in sorted(set(by_id) - seen_versions - INPUT_CONTRACT_SCHEMAS):
        errs.append(f"[coverage] no real payload exercised schema {sid} — the generator must "
                    f"produce at least one payload for every published schema")
    if expected_viewers is not None:
        for v in sorted(expected_viewers - seen_viewers):
            errs.append(f"[coverage] viewer {v} produced no payloads — both Player and Jonas required")
    elif len(seen_viewers) < 2:
        errs.append(f"[coverage] payloads cover only {len(seen_viewers)} distinct viewer(s) and there "
                    f"is no _manifest.json — both Player and Jonas required")

    # Direction 2: schema nullability matches the DDL, for every published schema.
    for sid, (_f, schema) in by_id.items():
        for e in check_direction2(sid, schema):
            errs.append(f"[dir2] {e}")

    return errs, n_payload, len(by_id), len(seen_versions), len(seen_viewers)


def selftest(schema_dir):
    """Teeth: flipping confidence to nullable MUST make direction 2 fail."""
    by_id = load_schemas(schema_dir)
    total = bitten = 0
    for sid, (_f, schema) in by_id.items():
        if not any(name == "confidence" for _p, name, _s in walk_props(schema)):
            continue
        total += 1
        mutant = copy.deepcopy(schema)

        def flip(node):
            if not isinstance(node, dict):
                return
            for k, v in node.get("properties", {}).items():
                if k == "confidence":
                    v["type"] = ["number", "null"]
                flip(v)
            if isinstance(node.get("items"), dict):
                flip(node["items"])

        flip(mutant)
        if any("confidence" in e for e in check_direction2(sid, mutant)):
            bitten += 1
            print(f"  selftest: nullable confidence in {sid} -> direction 2 FAILS (good)")
        else:
            print(f"  SELFTEST FAIL: nullable confidence in {sid} was NOT caught by direction 2")
    if total == 0:
        print("SELFTEST FAIL: no confidence-bearing schema to mutate")
        return 1
    if bitten == total:
        print(f"SELFTEST OK: direction 2 bites on all {total} confidence-bearing schema(s)")
        return 0
    return 1


if __name__ == "__main__":
    args = sys.argv[1:]
    if args and args[0] == "--selftest":
        sys.exit(selftest(args[1] if len(args) > 1 else "core/api/schema"))
    schema_dir = args[0] if len(args) > 0 else "core/api/schema"
    payload_dir = args[1] if len(args) > 1 else "ci/.payloads"
    errs, n_payload, n_schema, n_seen, n_viewers = run(schema_dir, payload_dir)
    print(f"SPEC-011 schema contract: {n_payload} payload(s); "
          f"coverage {n_seen}/{n_schema} published schemas, {n_viewers} viewer(s)")
    if errs:
        print(f"FAIL ({len(errs)} violation(s)):")
        for e in errs:
            print("  -", e)
        sys.exit(1)
    print("PASS: every published schema exercised by real payloads (both viewers); all payloads "
          "validate; NOT NULL-sourced fields non-nullable; nullable fields stay nullable.")
    sys.exit(0)
