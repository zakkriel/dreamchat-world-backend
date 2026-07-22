# FINAL — Decompose

**Job (founder-locked, absolute):** turn the player's raw text into the chain of attempts the player
**actually stated**, with real entity ids. Nothing else. It decides no outcomes, writes nothing, judges
nothing, and adds nothing.

---

## The one rule: the Decomposer ADDS NOTHING. Ever.

If the player's words state three actions, the chain has three attempts. If the stated action is
impossible from where things stand — wrong room, closed door, target a continent away — **that is not the
Decomposer's problem to fix.** It emits the attempt as stated; the engine's checks fail it; the player
hears the truth and decides what to do about it.

**Why (the reasoning an implementing agent needs):**

Any rule that lets the Decomposer "helpfully complete" an action cannot tell a small fix from an absurd
one without judging scale — and judging is exactly what this seat must never do. The proof case:

- *"Put the crate in the storeroom"* (player is at the bar) — a helpful Decomposer adds the walk. Seems
  harmless.
- *"I get the potions from my wardrobe"* (wardrobe is two months away by horse) — the **same** helpful
  rule adds a two-month journey, ignoring the scene, everyone in it, and the tension system. A frantic
  5-second beat now contains a horse trek.

Both cases are identical to a "fix the impossibility" rule: add the move that makes it possible. Any line
drawn between them ("only add *small* steps") requires the Decomposer to judge what's small — a smart
Decomposer, which guesses intent, which corrupts silently. So the line is drawn at zero: **a failure
message beats a guessed plan, every time.** "Nope, you're not in the room" is correct behavior.

**Failure is an answer, not a problem.** The system pushes back step by step and the player responds step
by step — that is the game working:

> Player: "I go through the door."
> → chain: `[move through door]` as stated → the floor reads the portal: `open = false` → blocked →
> *"The door is closed."*
> Player: "I open it and go through."
> → chain: `[open door, move through]` as stated → the open is always-adjudicated → the resolution LLM
> reads the full state, including ad-hoc `barred_from_inside` → *"It won't budge — something is blocking
> it from the other side."*

Nobody added steps. Nobody guessed. The world answered, the player chose.

---

## What the Decomposer DOES (positive examples)

- **Stated chains, faithfully.** *"I cross to the bar, slip her the note, and ask about the
  harbormaster"* → `[move → bar, relocate note → her, communicate → her]`. Three stated actions, three
  attempts, in order.
- **Identification.** Every reference becomes a real id, chosen ONLY from the scene payload's candidate
  whitelist. *"the note"* → `art_note` (the carried item with the active thread).
- **UNRESOLVED on genuine ties.** *"I give the note to her"* with Mara (just spoken to) and a hooded
  woman (a plausible contact) both live → do not pick. Emit UNRESOLVED → the player is asked *"Mara, or
  the woman in the corner?"* A rejected proposal beats a guessed canon.
- **Attempts, not outcomes.** *"I attack him"* → the attack attempt. What happens is resolve's job.

## What the Decomposer does NOT do (negative examples)

- **No missing-step completion.** *"Put the crate in the storeroom"* from the bar → emits the relocation
  as stated → in-reach fails → *"you're not in the storeroom."* It does NOT emit the walk.
- **No plan-building.** *"Burn the ledger"* with the only candle across the room → emits the burn attempt
  → it fails for lack of fire → the player is told. It does NOT build `[walk to candle, take candle, walk
  back, burn]` — that is choosing *how*, which is the player's right.
- **No scene-breaking journeys.** The wardrobe case above. The attempt fails with the truth ("your
  wardrobe is two months away"); the player decides if that trip is happening.
- **No judgment calls of any kind.** It does not rate tension, does not predict outcomes, does not decide
  whether an action is trivial, does not soften or interpret. Words → stated attempts + ids. Done.

---

## OPEN (to be driven in the point-by-point pass)

1. The output JSON schema, exactly (fields per attempt; how a target vs an instrument is marked).
2. UNRESOLVED mechanics: exactly what the clarify question contains, and what happens to the rest of the
   chain behind the unresolved step.
3. Whether NPC/world attempts (from cognition) pass through decompose at all or arrive pre-structured.
