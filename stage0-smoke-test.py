#!/usr/bin/env python3
"""
Stage-0 smoke test for DreamChat (Chunk 5.5).

Checks the two things that block the build:
  1) DECOMPOSE host (DeepInfra): strict structured output  AND  tolerates mature INPUT.
  2) NARRATE route (OpenRouter): GENERATES mature content without refusing.
Plus a quick reasoning-access check (DeepSeek on DeepInfra).

Run:
  pip install openai
  export DEEPINFRA_API_KEY=...        # do NOT hardcode keys
  export OPENROUTER_API_KEY=...
  python stage0-smoke-test.py

Content note: the prompts below are mature/tense but not explicit. To test your REAL
boundary, swap in a representative sample of your actual content where marked. The one
hard line never moves, on any host or model: no minors.
"""

import os, sys, json
from openai import OpenAI

# --- config: VERIFY these model ids against each provider's current catalog ---
DEEPINFRA_BASE  = "https://api.deepinfra.com/v1/openai"
OPENROUTER_BASE = "https://openrouter.ai/api/v1"

DECOMPOSE_MODEL = "mistralai/Mistral-Small-24B-Instruct-2501"
REASONING_MODEL = "deepseek-ai/DeepSeek-V3.1"  # adjust to a current DeepInfra DeepSeek id
NARRATE_MODEL   = "sao10k/l3.3-euryale-70b"    # a permissive RP route; adjust to a current OpenRouter id

REFUSAL_MARKERS = ["i can't", "i cannot", "i'm not able", "i am not able", "i won't",
                   "i'm unable", "as an ai", "i must decline", "i'm sorry, but i can"]

def looks_like_refusal(text: str) -> bool:
    t = (text or "").lower()
    return (not t.strip()) or any(m in t for m in REFUSAL_MARKERS)

# minimal beat_chain-ish schema, just to confirm structured output works on your account
BEAT_SCHEMA = {
    "type": "object",
    "properties": {
        "events": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "type": {"type": "string", "enum": ["move", "say", "intimidate"]},
                    "target": {"type": "string"},
                    "content": {"type": "string"},
                },
                "required": ["type"],
                "additionalProperties": False,
            },
        }
    },
    "required": ["events"],
    "additionalProperties": False,
}

def test_decompose() -> bool:
    print("\n[1] DECOMPOSE — DeepInfra: strict structured output + mature-input tolerance")
    client = OpenAI(base_url=DEEPINFRA_BASE, api_key=os.environ["DEEPINFRA_API_KEY"])
    # mature/tense input (the slice's intimidation beat). >>> swap in a sample of YOUR content to test the real edge.
    player_text = 'I grab Mara by the collar and snarl, "Drop the act. I know you know me. Talk."'
    try:
        r = client.chat.completions.create(
            model=DECOMPOSE_MODEL,
            messages=[
                {"role": "system", "content": "Convert the player's input into structured events. Output JSON only."},
                {"role": "user", "content": player_text},
            ],
            response_format={"type": "json_schema",
                             "json_schema": {"name": "beat_chain", "strict": True, "schema": BEAT_SCHEMA}},
            temperature=0, max_tokens=300,
        )
        out = r.choices[0].message.content
        json.loads(out)  # must be valid JSON or this raises
        ok = not looks_like_refusal(out)
        print("   structured JSON:", "OK" if ok else "REFUSED/empty")
        print("   ->", out[:300])
        return ok
    except Exception as e:
        print("   ERROR:", e)
        return False

def test_reasoning() -> bool:
    print("\n[2] REASONING — DeepInfra: DeepSeek access")
    client = OpenAI(base_url=DEEPINFRA_BASE, api_key=os.environ["DEEPINFRA_API_KEY"])
    try:
        r = client.chat.completions.create(
            model=REASONING_MODEL,
            messages=[{"role": "user", "content":
                       "One sentence: if Jonas just left and Mara is now alone, is she easier or harder to intimidate, and why?"}],
            temperature=0, max_tokens=150,
        )
        out = r.choices[0].message.content
        print("   ->", (out or "")[:250])
        return bool((out or "").strip())
    except Exception as e:
        print("   ERROR:", e)
        return False

def test_narrate() -> bool:
    print("\n[3] NARRATE — OpenRouter permissive route: generates mature content")
    client = OpenAI(base_url=OPENROUTER_BASE, api_key=os.environ["OPENROUTER_API_KEY"])
    sys_p = "You are the narrator of a dark-fantasy adult RPG. Render vivid, mature prose for adults. Stay in scene."
    # mature/tense scene. >>> swap in a representative sample of YOUR most explicit content to test the real edge.
    scene = ("The player has just grabbed Mara and threatened her. She is secretly terrified but feigns icy "
             "calm. Narrate what the player perceives, in 4-5 sentences.")
    try:
        r = client.chat.completions.create(
            model=NARRATE_MODEL,
            messages=[{"role": "system", "content": sys_p}, {"role": "user", "content": scene}],
            temperature=0.8, max_tokens=300,
        )
        out = r.choices[0].message.content
        ok = not looks_like_refusal(out)
        print("   generated:", "OK" if ok else "REFUSED/empty")
        print("   ->", (out or "")[:400])
        return ok
    except Exception as e:
        print("   ERROR:", e)
        return False

if __name__ == "__main__":
    for k in ("DEEPINFRA_API_KEY", "OPENROUTER_API_KEY"):
        if not os.environ.get(k):
            print(f"Set {k} in your environment first (do not hardcode keys).")
            sys.exit(1)

    results = {
        "decompose (structured + input tolerance)": test_decompose(),
        "reasoning access": test_reasoning(),
        "narrate (mature generation)": test_narrate(),
    }

    print("\n=== SUMMARY ===")
    for name, ok in results.items():
        print(f"   {'PASS ' if ok else 'CHECK'}  {name}")
    print("\nNotes:")
    print(" - 'CHECK' = read the output above; the refusal heuristic is rough, trust your eyes.")
    print(" - Swap the two marked prompts for a representative sample of YOUR content to test the real edge.")
    print(" - Verify the model ids at the top against each provider's current catalog.")
    print(" - Hard line, always, every host and model: no minors. Non-negotiable.")
