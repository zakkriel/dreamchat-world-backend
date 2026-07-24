#!/usr/bin/env python3
"""
Is "temperament decides it" a real variable, or an excuse for non-determinism?

Test: state the temperament EXPLICITLY, both ways, reasoning-first, N runs each.
Does the model TRACK the deciding trait?
  - crumbles  -> should be consistently EASIER (caves when afraid + lost protector)
  - hardens   -> should be consistently HARDER (defiant/reckless when cornered)

VERDICT:
  - Tracks cleanly (crumbles=EASIER, hardens=HARDER, both consistent)
      -> temperament is a real variable; build-plan §3c stands; the model reasons from complete state.
  - Doesn't track (either inconsistent, or doesn't flip with the trait)
      -> §3c was an excuse for noise; DELETE it. The model is just unreliable at this verdict.

Note: engine-owns-outcome stands either way on determinism/replay grounds — you don't put a
stochastic model in the canon path regardless. What's on trial here is the EXPLANATION.

Run:
  source venv/bin/activate
  export DEEPINFRA_API_KEY=...
  python validate-v3-temperament.py
"""
import os, sys
from collections import Counter
from openai import OpenAI

MODEL = "deepseek-ai/DeepSeek-V4-Flash"   # swap to deepseek-ai/DeepSeek-V3.1 if this 404s
N, TEMP = 6, 0.7

BASE = ("Facts:\n"
        "- Jonas is Mara's protector; his presence is the source of her courage.\n"
        "- {trait}\n"
        "- Jonas has just left, leaving Mara alone.\n"
        "First reason in ONE sentence. Then on a NEW final line output exactly one word: EASIER or HARDER.")

CRUMBLES = BASE.format(trait="Mara CRUMBLES under pressure: she caves and complies when afraid or threatened.")
HARDENS  = BASE.format(trait="Mara HARDENS under pressure: she turns defiant and reckless when cornered, with nothing to lose.")

def last_verdict(text: str) -> str:
    t = (text or "").upper()
    e, h = t.rfind("EASIER"), t.rfind("HARDER")
    if e == -1 and h == -1:
        return "?"
    return "EASIER" if e > h else "HARDER"

def run(label, prompt, client):
    print(f"\n=== {label}  (x{N}, temp={TEMP}) ===")
    tally = Counter()
    for i in range(N):
        try:
            r = client.chat.completions.create(
                model=MODEL,
                messages=[{"role": "user", "content": prompt}],
                temperature=TEMP, max_tokens=120,
            )
            out = r.choices[0].message.content
        except Exception as e:
            print(f"  {i+1}. ERROR: {e}"); tally["ERR"] += 1; continue
        v = last_verdict(out)
        tally[v] += 1
        print(f"  {i+1}. [{v:6}] {(out or '').strip()[:130]}")
    print(f"  -> {dict(tally)}")
    return tally

if __name__ == "__main__":
    if not os.environ.get("DEEPINFRA_API_KEY"):
        print("Set DEEPINFRA_API_KEY first."); sys.exit(1)
    client = OpenAI(base_url="https://api.deepinfra.com/v1/openai",
                    api_key=os.environ["DEEPINFRA_API_KEY"])

    c = run("CRUMBLES  (expect EASIER)", CRUMBLES, client)
    h = run("HARDENS   (expect HARDER)", HARDENS, client)

    crumbles_ok = c["EASIER"] == N
    hardens_ok  = h["HARDER"] == N
    tracks = crumbles_ok and hardens_ok

    print("\n=== VERDICT ===")
    print(f"  crumbles -> EASIER consistently: {crumbles_ok}  ({dict(c)})")
    print(f"  hardens  -> HARDER  consistently: {hardens_ok}  ({dict(h)})")
    print(f"  TRACKS THE TRAIT: {tracks}")
    if tracks:
        print("  => temperament is a real variable; §3c stands; model reasons from complete state.")
    else:
        print("  => §3c was an excuse for noise; DELETE it. Model is unreliable at the verdict.")
        print("     (engine-owns-outcome still stands on determinism grounds.)")
