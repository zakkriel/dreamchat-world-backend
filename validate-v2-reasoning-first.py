#!/usr/bin/env python3
"""
Confirm: was the grounded "HARDER" label an answer-first artifact, or a real reasoning error?

Re-run GROUNDED with REASONING-FIRST (reason, THEN commit one word). Parse the FINAL word.
- Flips to EASIER  -> the earlier HARDER was the format's fault, not the model's logic.
- Still HARDER     -> the model's logic itself is unreliable; the engine must own the outcome outright.

Either way the design conclusion holds (engine owns the verdict); this just tells us how
unreliable the model is, which is worth knowing.

Run:
  source venv/bin/activate
  export DEEPINFRA_API_KEY=...
  python validate-v2-reasoning-first.py
"""
import os, sys
from collections import Counter
from openai import OpenAI

MODEL = "deepseek-ai/DeepSeek-V4-Flash"   # swap to deepseek-ai/DeepSeek-V3.1 if this 404s
N, TEMP = 6, 0.7

GROUNDED_REASON_FIRST = (
    "Facts:\n"
    "- Jonas is Mara's protector; his presence is the source of her courage.\n"
    "- Jonas has just left, leaving Mara alone.\n"
    "First reason in ONE sentence. Then on a NEW final line output exactly one word: EASIER or HARDER."
)

def last_verdict(text: str) -> str:
    t = (text or "").upper()
    e, h = t.rfind("EASIER"), t.rfind("HARDER")   # LAST occurrence = the committed word
    if e == -1 and h == -1:
        return "?"
    return "EASIER" if e > h else "HARDER"

if __name__ == "__main__":
    if not os.environ.get("DEEPINFRA_API_KEY"):
        print("Set DEEPINFRA_API_KEY first.")
        sys.exit(1)
    client = OpenAI(base_url="https://api.deepinfra.com/v1/openai",
                    api_key=os.environ["DEEPINFRA_API_KEY"])

    print(f"=== GROUNDED, REASONING-FIRST  (x{N}, temp={TEMP}) ===")
    tally = Counter()
    for i in range(N):
        try:
            r = client.chat.completions.create(
                model=MODEL,
                messages=[{"role": "user", "content": GROUNDED_REASON_FIRST}],
                temperature=TEMP, max_tokens=120,
            )
            out = r.choices[0].message.content
        except Exception as e:
            print(f"  {i+1}. ERROR: {e}")
            tally["ERR"] += 1
            continue
        v = last_verdict(out)
        tally[v] += 1
        print(f"  {i+1}. [{v:6}] {(out or '').strip()[:140]}")
    print(f"  -> {dict(tally)}")
    print("\nExpected if answer-first was the artifact: clean EASIER.")
    print("If it STILL says HARDER: the model's logic is unreliable even reasoning-first ->")
    print("  the engine must own the outcome entirely (which we're doing anyway).")
