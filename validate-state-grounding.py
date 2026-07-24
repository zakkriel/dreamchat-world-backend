#!/usr/bin/env python3
"""
Validate (or break) the state-grounding assumption — build-plan §3.

CLAIM: the reasoning seat flip-flopped on "is Mara easier/harder to intimidate after
Jonas leaves" because the prompt didn't give it the relationship, NOT because the model
is broken. If we STATE the relationship, the answer should become consistent and
directionally correct.

This tests at the PROMPT level, where the claim lives. (The DB/seed version is build-time
and only matters once the engine threads state into the resolve prompt — which presupposes
exactly what we're testing here, so it can't be the validation.)

Run:
  source venv/bin/activate
  export DEEPINFRA_API_KEY=...
  python validate-state-grounding.py
"""
import os, sys
from collections import Counter
from openai import OpenAI

MODEL = "deepseek-ai/DeepSeek-V4-Flash"   # the actual resolve/cognition seat
# if this 404s, swap to "deepseek-ai/DeepSeek-V3.1" (the smoke test confirmed it live)
N = 6                                      # runs per condition
TEMP = 0.7                                 # >0 on PURPOSE — to expose instability

UNGROUNDED = (
    "Jonas has just left and Mara is now alone. "
    "Is she EASIER or HARDER to intimidate? "
    "Answer with one word (EASIER or HARDER), then one sentence why."
)

GROUNDED = (
    "Facts about the world:\n"
    "- Jonas is Mara's protector; his presence is the source of her courage.\n"
    "- Jonas has just left, leaving Mara alone.\n"
    "Given ONLY these facts: is Mara EASIER or HARDER to intimidate? "
    "Answer with one word (EASIER or HARDER), then one sentence why."
)

def verdict(text: str) -> str:
    t = (text or "").upper()
    e, h = t.find("EASIER"), t.find("HARDER")
    if e == -1 and h == -1:
        return "?"
    if h == -1 or (e != -1 and e < h):
        return "EASIER"
    return "HARDER"

def run(label: str, prompt: str, client: OpenAI) -> Counter:
    print(f"\n=== {label}  (x{N}, temp={TEMP}) ===")
    tally = Counter()
    for i in range(N):
        try:
            r = client.chat.completions.create(
                model=MODEL,
                messages=[{"role": "user", "content": prompt}],
                temperature=TEMP, max_tokens=80,
            )
            out = r.choices[0].message.content
        except Exception as e:
            print(f"  {i+1}. ERROR: {e}")
            tally["ERR"] += 1
            continue
        v = verdict(out)
        tally[v] += 1
        print(f"  {i+1}. [{v:6}] {(out or '').strip()[:120]}")
    print(f"  -> {dict(tally)}")
    return tally

if __name__ == "__main__":
    if not os.environ.get("DEEPINFRA_API_KEY"):
        print("Set DEEPINFRA_API_KEY first.")
        sys.exit(1)
    client = OpenAI(base_url="https://api.deepinfra.com/v1/openai",
                    api_key=os.environ["DEEPINFRA_API_KEY"])

    u = run("UNGROUNDED (no relationship given)", UNGROUNDED, client)
    g = run("GROUNDED (Jonas = protector)", GROUNDED, client)

    print("\n=== HOW TO READ THIS ===")
    print(" ASSUMPTION HOLDS if: UNGROUNDED is split/unstable  AND  GROUNDED is consistent + EASIER")
    print("   (protector leaves -> less protected -> easier to intimidate).")
    print("   => resolve must SUPPLY state; the model reasons from it. Build-plan §3 stands.")
    print()
    print(" ASSUMPTION BREAKS if: GROUNDED still splits.")
    print("   => the model won't reason reliably even from given state, so resolve must COMPUTE the")
    print("      direction deterministically in the engine and use the model only for flavor.")
    print("      That's the more important finding — it changes the resolve design.")
    print()
    ug_split = len([k for k in (u["EASIER"], u["HARDER"]) if k > 0]) > 1
    g_consistent = g["EASIER"] == N or g["HARDER"] == N
    print(f" Observed: ungrounded_split={ug_split}, grounded_consistent={g_consistent}, "
          f"grounded_direction={'EASIER' if g['EASIER']>=g['HARDER'] else 'HARDER'}")
