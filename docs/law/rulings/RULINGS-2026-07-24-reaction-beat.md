# REACTION-BEAT RULINGS — 2026-07-24

**Why this file exists.** The held-outcome / reaction-beat flow was the last undesigned load-bearing
piece of the loop (the FINAL docs locked the principle — "the enemy moves to strike, never the enemy
killed you" — but not the machinery). This grilling session ruled it, plus the two seams the
foundations code fail-louds on (rich-ruling commits; the deception split). Every ruling below was
made by the founder. Style per the standing rule: example + reasoning, so an implementing agent who
wasn't in the room cannot misread it.

---

## 1. A telegraph ends the beat; the rest of the chain is DISCARDED

When a cognition seat telegraphs a disruptive act mid-chain (Jonas pushes off the bar as you start
step 1 of "cross to the bar, slip her the note, ask about the harbormaster"):

- The **wind-up commits** as a perceivable canon event (replay rule: telegraph → reaction →
  resolution are committed events, in order).
- A **held-outcome record** is written (see §3).
- The **beat ends right there.** The narrator delivers the moment ("Jonas pushes off the bar,
  moving to cut in"). The un-run steps (the note, the question) are **dropped**.
- The player's next message is **free text** — it can restate the plan ("I press on and slip her
  the note anyway"), which makes the restatement the reaction.

**Why discard, not suspend/resume:** the world just changed; the old plan is stale by definition.
A resume prompt needs suspended-chain storage plus rules for when the collision invalidated the
stored steps — machinery that guesses the player still means it. "The world pushes back, the player
decides" — a fresh statement IS the decision.

**Interruption is the exception, not the rhythm (founder-stressed).** Per step, the world's response
has three tiers, sorted by the cognition seat's own judgment (already expressed as
`none | commit | telegraph` in npc_attempts/1):
- **none** — most steps, most of the time. Chain runs.
- **commit** — a non-disruptive act lands inline (Jonas shifts his weight). Perceivable; chain
  continues unless it broke the next step's premise.
- **telegraph** — disruptive/contested only. Held outcome; beat ends.
The engine never decides what is dramatic; it obeys the mind's choice.

## 2. The reaction meets the held act(s): FIRST action + held into ONE judgment

The reaction is free text → decompose runs as usual (ids from the whitelist) and may yield several
attempts. Rule: the **first stated action** joins the held act(s) in the one combined ruling (the
collision rule locked 2026-07-23 §9). The **remaining actions run as a normal chain** against the
post-collision world — world-first, premise checks, all the usual rules.

**Example.** Held: Jonas's cut-in. Reaction: "I shove him back, and still slip her the note, and
wink at her." → combined ruling over {Jonas's cut-in, the shove}; then the note and the wink run as
an ordinary two-step chain in the world the ruling produced. The referee never adjudicates the wink
inside the collision.

**Non-resisting reactions are still reactions.** "I back off" — or something unrelated ("I ask the
barkeep for ale") — still enters the combined ruling as the player's answer to the moment; the
referee reads it as not-contesting and Jonas's act likely completes. No special case.

## 3. Held-outcome storage: a dedicated record; the STATE lives in the world

- A held outcome is its own row: the NPC's full intended act (typed attempt) + a link to the
  committed telegraph event + status `pending | resolved`.
- It does **NOT** reuse the pending-events ledger: the ledger fires on the **clock**; a held outcome
  fires on the **player's next input**. Different trigger, different row.
- **No session state machine.** The next input, before decompose routing, checks "any pending held
  outcome in this scene?" — yes → the input is a reaction (§2); no → a normal beat. Server memory
  holds nothing; replay and crash-recovery come free.
- **No timeout exists (v1).** The player is the clock; the world cannot move until they answer, so
  the very next input always resolves the hold — whatever it says.
- **Multiple simultaneous telegraphs** (Jonas cuts in AND the hooded woman rises): both wind-ups
  commit, the beat ends once, and the reaction's first action meets **all** pending held acts in the
  one combined judgment. Depth-1 (no NPC re-reactions) keeps it from cascading.

## 4. The deception split: TRUTH + default APPEARANCE + per-receiver variants

A ruled event carries:

- **truth** — what really happened. **Always what canon records.** Canon never lies.
- **appearance** (optional) — the default face an ordinary observer receives ("Mara seems unmoved
  and shrugs it off" while the truth is "Mara, secretly shaken, masks it and deflects").
- **per-receiver variants** (optional) — the referee, who judged the moment truth-side and knows
  every participant, may grant a specific perceiver a sharper or duller read ("to Jonas: you catch
  the tremor in her hand").
- **visible=false** — a fully hidden act: no perceptions generated at all.

Fan-out writes each receiver's perception from: their variant, else the default appearance, else
the truth. **The appearance is NOT a second canon (founder-stressed):** different perceivers may
legitimately read the same moment differently; every divergence is licensed by the ruling and
logged. Later fan-out fidelity (distance, senses, language — commit/perception OPEN #1) rides this
same per-receiver hook; no new machinery.

**Rejected alternatives, for the record:** appearance-as-canon + hidden note (makes canon carry the
lie — breaks bedrock); two separate events per masked moment (doubles rows, halves can drift).

## 5. World eruptions: MEDIUM/LARGE cut the beat; SMALL never does

A World Actor eruption is not aimed at the player and is not contestable — there is no held outcome
and no reaction beat. But it can seize the scene:

- **small** (the rain, the drunk) — commits, is perceivable, the chain runs on.
- **medium / large** (the brawl, the town alarm) — commits, the narrator delivers it, and the beat
  **ends**: remaining steps discarded (same rule as §1), next input free.

**Why keyed to the magnitude tier (mechanical), not judgment:** the tier already exists as a
validated enum with independent pressure pools; one rule, zero new dials. Strict premise-logic alone
would have you finish interrogating Mara while the alarm screams over the docks — premise-legal and
dramatically dead. Disruption at the wrong moment is the feature.

## 6. Rich-ruling commits (the apply_ruled_event seam) — mechanics, stated and confirmed

- **Attribute writes** land as `state_mutation` rows on `attrs.*` paths (the existing machinery —
  trigger projects them), with provenance to the ruling's committed event.
- **Tier-1 names** (the engine-known closed set that ships with the seeded artifact types: a
  portal's `open/locked/connects`, a container's measurements, …) are **strictly validated** —
  name in the closed set, value type-checked — because engine checks read them. The closed set
  lives in code, per the contracts doc ("grows only when we add a check in code — never at
  runtime, never by mint").
- **Tier-2 names** (everything else) are written as-is; the engine never reads them; LLM seats do.
- The Tier-1/Tier-2 discipline rule stands: a fact meant to physically stop people lands in the
  Tier-1 field AND Tier-2 carries the meaning.
- Ruled events of non-target shapes (a ruled move, a ruled utterance) become committable when
  Station D replaces the foundations' fail-loud `ruled_event_rejected` guard with the full
  `apply_ruled_event` — carrying §4's truth/appearance/variants and this section's writes.

## 7. The empty reaction: the player's WORDS enter the ruling even when no act does
   (ruled 2026-07-24, Station E execution session)

Decompose can legitimately emit ZERO attempts from a reaction ("I do nothing", "I just watch") —
the schema has no minimum, and stillness is a real answer. Ruling: the player's raw text still
enters the combined ruling as their stated answer, marked as words-not-an-act. No typed attempt is
invented for it, and it commits no canon event of its own.

**Example.** Jonas's cut-in is held. The player types "I just watch." Decompose emits `[]`. The
combined ruling's prompt carries the held act plus: `THE PLAYER'S ANSWER (stated, not an act): "I
just watch"`. The referee reads it as not-contesting; Jonas's act likely completes.

**Why not held-acts-alone (what the code briefly did):** it drops the player's words entirely —
the referee never sees the answer, violating §2's "still enters the combined ruling as the
player's answer to the moment."

**Why not bounce-and-ask ("say what you do"):** stillness, silence, and watching are legitimate
answers; forcing a typed act punishes exactly the non-resisting case §2 protects. The clarify
bounce stays reserved for UNRESOLVED reference ties, where the engine genuinely cannot proceed.

---

## What this closes / what stays open

**Closes:** the reaction-beat state machine (FINAL loop PRD §2 "held outcome", previously
machinery-less); FINAL-resolve OPEN #2's semantic half (the output's truth/appearance/variant
shape — the concrete JSON schema update lands with Station D); the eruption-interrupt boundary.

**Still open (owned by their stations):** the ruling's prompt shape + repair-error details
(Station D); fan-out fidelity rules and hidden-truth surfacing (Station E+); UNRESOLVED clarify-loop
UX; the beat response contract's exact fields for "the beat ended early — here's why"
(Station D/E implementation detail; the halt taxonomy already carries the reasons).
