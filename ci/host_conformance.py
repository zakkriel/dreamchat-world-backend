#!/usr/bin/env python3
"""SPEC-039: score every allowlisted host against one fixed corpus, one host at a time.

WHY THIS EXISTS. OpenRouter does not run the model; it forwards the request to one of several
companies that host it. Those hosts run different software, and a host whose constrained decoding is
broken returns a structurally VALID chain that says something the player never said. The gate cannot
catch it (the chain is valid), the seat log cannot show it (status ok), and the symptom reads as "the
model is occasionally wrong" — which sent this investigation after the model, the prompt, the scene
and a missing listener before the host was even suspected.

Measured 2026-08-28: byte-identical input, one host pinned per run — venice 6/24 wrong, deepinfra and
coreweave 24/24 correct. Swapping models never isolates this, because every model goes through the
same router.

HOW IT HOLDS VARIABLES STILL. The prompt and schema are the real assembled bytes dumped by
TestGenDecomposeHostProbe, not a reconstruction. Only the player's sentence changes. `allow_fallbacks`
is FORCED OFF so a pinned host cannot quietly hand the request to another one — without that, this
measures the router, not the host, which is exactly the mistake that cost four rounds.

WHAT IT IS NOT. Not a CI gate: it spends real money and needs a real key. Run it when the allowlist
changes, when a host is added, or when a seat starts giving answers that feel wrong.
"""

from __future__ import annotations

import argparse
import concurrent.futures as cf
import json
import os
import sys
import time
import urllib.error
import urllib.request

PLACEHOLDER = "__PLAYER_TEXT__"
ENDPOINT = "https://openrouter.ai/api/v1/chat/completions"


def load_corpus(path: str) -> list[tuple[str, str]]:
    rows: list[tuple[str, str]] = []
    with open(path, encoding="utf-8") as handle:
        for raw in handle:
            line = raw.rstrip("\n")
            if not line.strip() or line.lstrip().startswith("#"):
                continue
            sentence, _, expected = line.partition("\t")
            if not expected.strip():
                sys.exit(f"corpus line is not <sentence><TAB><type>: {line!r}")
            rows.append((sentence.strip(), expected.strip()))
    if not rows:
        sys.exit("corpus is empty")
    return rows


def ask(key: str, model: str, prompt: str, schema: dict, host: str, sentence: str,
        temperature: float, max_tokens: int, reasoning: dict | None, timeout: int) -> tuple[str, str]:
    """Return (verdict, detail). Verdict is the first attempt's type, or ERR:<reason>."""
    body: dict = {
        "model": model,
        "max_tokens": max_tokens,
        "temperature": temperature,
        "messages": [{"role": "user", "content": prompt.replace(PLACEHOLDER, sentence)}],
        "response_format": {
            "type": "json_schema",
            "json_schema": {"name": "seat_output", "schema": schema, "strict": True},
        },
        # One host, and no escape hatch. allow_fallbacks:true would let the router serve this from a
        # different host and report a score for the wrong machine.
        "provider": {"only": [host], "allow_fallbacks": False, "require_parameters": True,
                     "data_collection": "deny"},
    }
    if reasoning is not None:
        body["reasoning"] = reasoning

    payload = None
    last = ("ERR:EXC", "not attempted")
    # 429 is capacity, not conformance: a pinned host under load says "not now", which is a different
    # fact from "wrong answer" and must not be scored as either. Retry it a few times with backoff —
    # without this, the two hosts that serve this model best both read DEAD (measured 2026-08-30).
    for attempt in range(4):
        req = urllib.request.Request(
            ENDPOINT,
            data=json.dumps(body).encode(),
            headers={"Content-Type": "application/json", "Authorization": f"Bearer {key}"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=timeout) as res:
                payload = json.load(res)
            break
        except urllib.error.HTTPError as err:
            detail = err.read().decode(errors="replace")[:160]
            last = (f"ERR:HTTP{err.code}", detail)
            if err.code == 429 and attempt < 3:
                time.sleep(2 * (attempt + 1))
                continue
            return last
        except Exception as err:  # noqa: BLE001 — a probe reports every failure rather than dying
            return "ERR:EXC", f"{type(err).__name__}: {err}"
    if payload is None:
        return last

    choices = payload.get("choices") or []
    if not choices:
        return "ERR:NOCHOICE", json.dumps(payload)[:160]
    content = (choices[0].get("message") or {}).get("content") or ""
    if not content.strip():
        finish = choices[0].get("finish_reason")
        return "ERR:EMPTY", f"finish_reason={finish}"
    try:
        chain = json.loads(content)
    except json.JSONDecodeError as err:
        return "ERR:BADJSON", f"{err}: {content[:120]}"
    if not isinstance(chain, list):
        return "ERR:NOTARRAY", json.dumps(chain)[:160]
    if not chain:
        return "EMPTY", "the host returned no attempts"
    first = chain[0]
    if not isinstance(first, dict) or "type" not in first:
        return "ERR:NOTYPE", json.dumps(first)[:160]
    return str(first["type"]), json.dumps(first)[:160]


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--material", required=True, help="dir holding prompt.txt and schema.json")
    ap.add_argument("--corpus", required=True)
    ap.add_argument("--model", required=True, help="e.g. deepseek/deepseek-v4-flash")
    ap.add_argument("--hosts", required=True, help="comma-separated allowlist, one probed at a time")
    ap.add_argument("--runs", type=int, default=2, help="repeats per sentence; >1 catches nondeterminism")
    ap.add_argument("--temperature", type=float, default=0.0)
    ap.add_argument("--max-tokens", type=int, default=2048)
    ap.add_argument("--reasoning-effort", default="none", help='"none" or "" to omit the field')
    ap.add_argument("--timeout", type=int, default=120)
    ap.add_argument("--workers", type=int, default=6)
    args = ap.parse_args()

    key = os.environ.get("OPENROUTER_API_KEY", "").strip()
    if not key:
        sys.exit("OPENROUTER_API_KEY is not set")

    prompt = open(os.path.join(args.material, "prompt.txt"), encoding="utf-8").read()
    if PLACEHOLDER not in prompt:
        sys.exit(f"prompt.txt has no {PLACEHOLDER} — re-run the dumper")
    schema = json.load(open(os.path.join(args.material, "schema.json"), encoding="utf-8"))
    corpus = load_corpus(args.corpus)
    hosts = [h.strip() for h in args.hosts.split(",") if h.strip()]
    reasoning = {"effort": args.reasoning_effort} if args.reasoning_effort else None

    # The binding is printed because a score is meaningless without it: model, decoding mode,
    # temperature, effort and ceiling all change the answer (AGENTS.md rule 0c).
    print(f"model={args.model}  json_schema strict  temp={args.temperature}  "
          f"effort={args.reasoning_effort or 'omitted'}  max_tokens={args.max_tokens}")
    print(f"corpus={len(corpus)} sentences x {args.runs} run(s)  hosts={len(hosts)}  "
          f"prompt={len(prompt)} bytes\n")

    # THREE numbers, never one. "Could not answer" and "answered wrongly" are different facts, and
    # collapsing them is the exact confusion that sent the original investigation after the model, the
    # prompt and the scene for four rounds: a rate-limited host and a mis-parsing host both look like
    # "this host scored badly" until you separate them. Only `wrong` is a conformance failure.
    wrong_anywhere = False
    for host in hosts:
        jobs = [(s, e) for (s, e) in corpus for _ in range(args.runs)]
        started = time.time()
        with cf.ThreadPoolExecutor(max_workers=args.workers) as pool:
            got = list(pool.map(
                lambda job: ask(key, args.model, prompt, schema, host, job[0],
                                args.temperature, args.max_tokens, reasoning, args.timeout),
                jobs))

        correct, wrong, unavailable = [], [], []
        for (sentence, expected), (verdict, detail) in zip(jobs, got):
            if verdict.startswith("ERR:"):
                unavailable.append((sentence, expected, verdict, detail))
            elif verdict == expected:
                correct.append(sentence)
            else:
                wrong.append((sentence, expected, verdict, detail))

        if wrong:
            label = "BAD "     # it answered, and answered wrongly — the only conformance failure
        elif unavailable and not correct:
            codes = {u[2] for u in unavailable}
            label = "BUSY" if codes == {"ERR:HTTP429"} else "DEAD"
        elif unavailable:
            label = "PART"     # answered some, could not be reached for the rest
        else:
            label = "OK  "

        print(f"{label} {host:<14} correct {len(correct):>3}   wrong {len(wrong):>3}   "
              f"unavailable {len(unavailable):>3}   {time.time() - started:5.1f}s")
        for sentence, expected, verdict, detail in wrong:
            print(f"       WRONG  {sentence[:40]:<40} want {expected:<16} got {verdict}")
        if unavailable:
            codes = sorted({u[2] for u in unavailable})
            print(f"       unavailable: {', '.join(codes)} — capacity or catalogue, NOT a wrong answer")
        if wrong:
            wrong_anywhere = True

    if wrong_anywhere:
        print("\nA host returned the wrong ATTEMPT for input other hosts parse correctly. That is a "
              "player's sentence becoming an act they did not choose, and the gate cannot catch it "
              "because the chain is structurally valid. Decide with the table above, not with one "
              "sample: remove a consistently wrong host from "
              "DREAMCHAT_PROVIDER_OPENROUTER_ROUTING.only, and record the numbers in SPEC-039.")
        return 1
    print("\nNo host returned a wrong attempt. Hosts marked BUSY/DEAD/PART were not fully measured — "
          "that is capacity or catalogue, not conformance, and says nothing either way about them.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
