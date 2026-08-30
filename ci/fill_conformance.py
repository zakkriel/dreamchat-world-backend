#!/usr/bin/env python3
"""SPEC-039 companion: can each allowlisted host return a VALID world_fill/1 fragment?

The decompose probe (ci/host_conformance.py) asks whether a host picks the right branch of a small
schema from a short sentence. This asks a different question about the same hosts:

    given a ~16KB request and a deeply nested schema with additionalProperties:false,
    does the reply parse at all?

Three live fill calls failed on 2026-08-28 with `unknown field "canonical_name"`, `unknown field
"hiding"` and `unexpected EOF`. Those are structural failures — misnesting and truncation — and no
sentence-to-type corpus can catch them. Scored here instead:

    OK          parsed, no unknown fields, not truncated
    TRUNCATED   finish_reason=length, or JSON ended mid-document
    UNKNOWN     a key the schema does not name (which is what DisallowUnknownFields refuses in Go)
    INVALID     not JSON at all
    DEAD        the host would not serve the request

MODE MATTERS. Production runs world_fill in json_object, not json_schema strict, so that is what this
sends. Measuring the wrong mode would score a host the pipeline never exercises.
"""

import argparse
import concurrent.futures
import json
import os
import sys
import urllib.error
import urllib.request

URL = "https://openrouter.ai/api/v1/chat/completions"


def ask(key, model, prompt, host, max_tokens, effort, timeout):
    body = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "response_format": {"type": "json_object"},
        "temperature": 0.0,
        "max_tokens": max_tokens,
        "provider": {
            "only": [host],
            "allow_fallbacks": False,
            "data_collection": "deny",
            "require_parameters": True,
        },
    }
    if effort:
        body["reasoning"] = {"effort": effort}
    req = urllib.request.Request(
        URL,
        data=json.dumps(body).encode(),
        headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            payload = json.load(r)
    except urllib.error.HTTPError as e:
        return None, None, f"HTTP {e.code}"
    except Exception as e:  # noqa: BLE001 — a probe reports, it does not raise
        return None, None, type(e).__name__
    choices = payload.get("choices") or []
    if not choices:
        return None, None, "no choices"
    content = (choices[0].get("message") or {}).get("content") or ""
    finish = choices[0].get("finish_reason")
    usage = payload.get("usage") or {}
    return content, {"finish": finish, "out": usage.get("completion_tokens")}, None


def score(content, meta, allowed_top, allowed_cast):
    """Reproduce what the Go belt does: json.Decoder with DisallowUnknownFields."""
    if meta and meta.get("finish") == "length":
        return "TRUNCATED", "finish_reason=length"
    try:
        doc = json.loads(content)
    except json.JSONDecodeError as e:
        # An unterminated document is the JSON face of truncation — Go reports it as unexpected EOF.
        if "Expecting" in str(e) or "Unterminated" in str(e):
            return "TRUNCATED", f"unterminated: {e}"
        return "INVALID", str(e)[:60]
    if not isinstance(doc, dict):
        return "INVALID", f"root is {type(doc).__name__}, not an object"
    bad = [k for k in doc if k not in allowed_top]
    if bad:
        return "UNKNOWN", f'top-level "{bad[0]}"'
    for entry in doc.get("cast") or []:
        if isinstance(entry, dict):
            bad = [k for k in entry if k not in allowed_cast]
            if bad:
                return "UNKNOWN", f'cast[] "{bad[0]}"'
    return "OK", f"{meta.get('out')} tok" if meta else "ok"


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--material", required=True, help="dir holding prompt.txt and schema.json")
    ap.add_argument("--model", required=True)
    ap.add_argument("--hosts", required=True, help="comma-separated, probed one at a time")
    ap.add_argument("--runs", type=int, default=3, help=">1 catches nondeterminism, which is the point")
    ap.add_argument("--max-tokens", type=int, default=16384)
    ap.add_argument("--reasoning-effort", default="none")
    ap.add_argument("--timeout", type=int, default=420)
    ap.add_argument("--workers", type=int, default=3)
    args = ap.parse_args()

    key = os.environ.get("OPENROUTER_API_KEY")
    if not key:
        sys.exit("OPENROUTER_API_KEY is unset — this probe spends real money and needs a real key")

    prompt = open(os.path.join(args.material, "prompt.txt"), encoding="utf-8").read()
    schema = json.load(open(os.path.join(args.material, "schema.json"), encoding="utf-8"))
    allowed_top = set(schema.get("properties", {}))
    cast = schema.get("properties", {}).get("cast", {}).get("items", {})
    allowed_cast = set(cast.get("properties", {}))
    if not allowed_top or not allowed_cast:
        sys.exit("schema.json has no properties — re-run the dumper")

    hosts = [h.strip() for h in args.hosts.split(",") if h.strip()]
    effort = args.reasoning_effort if args.reasoning_effort not in ("", "omit") else None

    print(f"model={args.model}  json_object  max_tokens={args.max_tokens}  effort={effort}")
    print(f"prompt={len(prompt)} bytes  schema names {len(allowed_top)} top-level keys, "
          f"{len(allowed_cast)} on a cast entry")
    print(f"{len(hosts)} host(s) x {args.runs} run(s)\n")

    worst = 0
    for host in hosts:
        results = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as pool:
            futures = [
                pool.submit(ask, key, args.model, prompt, host, args.max_tokens, effort, args.timeout)
                for _ in range(args.runs)
            ]
            for f in concurrent.futures.as_completed(futures):
                content, meta, err = f.result()
                if err:
                    results.append(("DEAD", err))
                    continue
                results.append(score(content, meta, allowed_top, allowed_cast))
        ok = sum(1 for v, _ in results if v == "OK")
        verdicts = ", ".join(f"{v}({d})" for v, d in results)
        flag = "clean" if ok == len(results) else "SUSPECT"
        print(f"  {host:<12} {ok}/{len(results)} OK  [{flag}]  {verdicts}")
        if ok != len(results):
            worst = 1

    print("\nA host that cannot return a parseable fragment is not a prompt problem. Pin the seat to a"
          "\nhost that can (DREAMCHAT_SEAT_ROUTING_WORLD_FILL) and record the score in SPEC-039.")
    return worst


if __name__ == "__main__":
    sys.exit(main())
