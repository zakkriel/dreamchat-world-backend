# GRILLING-SESSION RULINGS — 2026-07-23

**Why this file exists.** A grill-with-docs session (2026-07-22/23) stress-tested the FINAL doc set
against the shipped code, all six transcripts, the post-compaction rulings, and the founder directly.
It closed most of the OPEN questions in the two lowest-confidence subsystems (resolution ~35%,
cognition ~50%). Every ruling below was made by the founder in that session. Style follows the
standing rule: each ruling ships with its example and its reasoning, so an implementing agent who
wasn't in the room cannot misread it.

---

## 1. The uncertain-outcome ruling: EXPLAIN FIRST, THEN MATCH

When the resolution LLM decides an uncertain outcome, it must write its **reasoning first** (from the
facts it was handed), and the typed outcome must **follow from that reasoning**. The engine bounces
the ruling (standard repair ×1 → bounce) if the outcome contradicts the reasoning the model itself
just wrote.

**Why.** The recorded test failure: the model reasoned "without Jonas she is more vulnerable → easier
to intimidate" and then emitted the verdict `HARDER`. A verdict token that contradicts its own
reasoning must never become canon. Explain-first-then-match kills exactly that bug, in one call, at no
extra cost.

**Boundary, stated honestly.** This guards against self-contradiction. It does NOT guard against
sound reasoning over wrong or missing facts — that guard is the grounding rule below. The two only
work as a pair.

## 2. What the ruling reads: INVOLVED + ONE HOP, widen from play

The facts handed to a ruling = the entities bound in the action (participants, targets, instruments)
+ their current state and recent history + **one hop of direct links** (relationships, owned/held
things). Widen only when real play shows a genuine miss — never speculatively.

**Why one hop.** "Jonas is Mara's protector" decides the leaning-on-Mara outcome, and Jonas is one
link from Mara. The recorded experiment proved the model *invents* the relationship (reversed, 6/6
runs) when it isn't handed the fact. One hop is a mechanical walk of real links — no seat ever judges
"relevance," which is not computable.

## 3. Naming things: the actor's OWN knowledge, one hop — and bluffing is legal

- The candidate whitelist for identification (decompose) = what the **acting actor** perceives or
  knows, one hop. An NPC knowing the harbormaster does NOT extend the player's reach. Nobody — player
  included — can bind an entity they have no perception/knowledge path to.
- Unknown reference as the **target of a physical action** ("hand the note to the harbormaster",
  no harbormaster known) → cannot resolve. Fail or clarify. No world lookup — searching canon on the
  strength of a player's guess is both a perception-wall leak and a fishing exploit (type names until
  one resolves).
- Unknown reference as a **topic of speech** ("I ask Mara about the harbormaster") → legal, never
  blocked. The word is NOT resolved to an id for the speaker. Whether anything real answers to it is
  decided truth-side through the **listener's** knowledge: Mara is a participant, so if she knows a
  harbormaster he is one hop from her and his true canon state (say: dead) enters the ruling. If she
  doesn't know one, she is as lost as the player.

**What this buys.** Bluffing and prying work as play ("fake it to see if she flinches") — the player
gets a *reaction*, never a *record*. And the "harbormaster is dead" coherence hole closes by
construction: the moment a known entity is talked about, its true state is in the ruling's facts, so
the outcome cannot contradict canon.

## 4. Reading intent: at RESOLVE, per NPC — never in the decomposer

How words are read (sincere / prying / a threat) is decided at resolve, by **each listener's own
cognition, separately**. There is no single "the intent": Mara reads the probe wary, Jonas reads it
amused. The decomposer records the speech and the listener ids and nothing else — a parser that just
judged the room "prying" mis-parses the next sentence (single-job rule, founder-locked: "the
decomposer doesn't do shit besides decompose and assign IDs").

## 5. Cognition seats: ONE CALL PER ACTION; batch the public, ISOLATE the secret-holders

- Cost unit: one cognition call **per action** (a typed text may contain several actions) — not per
  text, not per NPC.
- Per action, each present NPC sits in **exactly one** seat:
  - **The shared batch** — one call, produces the reactions of every NPC whose read of the moment
    needs nothing beyond what everyone perceived. The batch prompt carries ONLY the public moment.
  - **An isolated call** — any NPC whose **private** knowledge is relevant to this action is pulled
    out of the batch and gets her own call, where her secret rides alone.
- **No double-acting.** The second flow touchpoint (a target's will consulted inside a contested
  resolve) is the same mind at a second moment of the same action, capped by depth-1 — not a second act.
- **The wall holds by construction, not by prompt discipline.** A secret never enters a shared
  prompt, so it cannot bleed into another NPC's reasoning. Instructing a model to ignore a secret it
  can see is the pink-elephant bet the recorded experiments already broke; and "was this reaction
  influenced?" is undetectable after the fact. You cannot validate a leak away — you can only not
  create it.
- **Batch-vs-isolated is a mechanical lookup, not a judgment:** intersect the action's bound ids with
  the subject links of the NPC's private records (one hop). `perception_subject` (ADR-035) already
  ships this machinery. Failure asymmetry: a missed flag → that NPC reacts flat this action (dull,
  never a leak); an over-flag → one extra call. The dangerous failure is structurally impossible.
- **Cost lever: prefix caching.** Providers give ~90% off input tokens repeating a previous prompt's
  prefix. Append-only canon (B-5) makes payloads cache-native: stable prefix (rules, profile, scene) +
  append-only committed events + mutable tail. Mutable summaries live at the TAIL, never the head.
- The batch's fixed generation order is the candidate answer to deterministic intra-tick ordering.

## 6. About-ness links: HARD RULE, validated at write

Every write of private knowledge MUST carry subject entity links (who/what it is about), validated
like any typed output: missing links → repair ×1 → bounce. Same discipline class as "if a fact should
physically stop people, write the Tier-1 field." An unlinked secret is invisible to every mechanical
lookup in this file — the isolation lookup (§5), the ruling gather (§2), and memory retrieval (§10).

## 7. The world beyond the room: TWO composing mechanisms

**7a. The pending-events ledger — known futures.** Consequences already caused get a fire-time row
(the fireshield's decay trigger, generalized): the dusk patrol, the contact arriving, the spell
expiring at tick 340. The advancing clock fires them into the beat when crossed. Zero model calls
until their moment. The world's agenda is DATA, independent of the player — "you're the clock, not
the cause," made mechanical.

**7b. The World Actor — spontaneity.** A mind that erupts with intrusions **nobody scheduled**, from
WORLD-scope context (region state, factions, presence, the ledger), explicitly allowed to be
**unrelated to the scene** — the town alarm and the gnoll wave cutting off whatever the player was
about to do. Disruption at the worst moment is a feature ("puts life into the world"), so there is NO
appropriateness/mood filter. It can bring non-present NPCs into scenes (it is the presence-boundary
mover). Its output enters the SAME pipeline as everyone else (no bypass), and the scene perceives
only the perceivable edge: the tavern hears the alarm and sees the patrol bolt; it does not see the
gnolls.

**Trigger: rising pressure on WORLD-TIME.** The chance of the World Actor acting grows with in-world
time since it last acted (founder's mechanism). Engine owns WHEN (mechanical accrual + a logged roll —
committed, replayed, never re-rolled); the LLM owns WHAT. Pressure rides clock ticks, not player
steps (a per-step basis would key the world's rhythm to player behavior — the player-centricity this
system refuses). Pressure carries across scenes — a tense moment can still get cut by the alarm if
pressure built during the quiet hour before. `none`-tension time skips near-guarantee eruptions at
crossing: three days of travel is three days of world.

**Magnitude: TIERED POOLS, independent.** Frequency and magnitude are different dials — a
world-changing event every five minutes kills the mood. Three size classes, each with its own pool
filling at its own speed on world-time: **small** (minutes — the drunk, the rain, the cart),
**medium** (hours — a fight breaks out, a messenger bursts in), **large** (days/weeks — the town
alarm, the gnoll wave). Pools are independent: texture events cannot reset or starve the big ones;
a pool drains when it fires. The engine hands the World Actor the drawn size as an input constraint
("author a SMALL intrusion"); size labels are a validated enum, like the tension tiers. Data,
retunable.

## 8. The Personality Module (its own module, per D-2)

Personality is core to the life of this system and gets its own module. Five parts:

1. **Authored core, grounded in backstory events.** Traits, values, fears, loyalties, speech manner —
   minted with the NPC, each trait explained by **minted canon events** ("her brother drowned at the
   harbor when she was twelve" → distrusts sailors). D-11 applied to character: traits have provenance
   from birth. The core + backstory ARE the cached prefix of her isolated calls.
2. **Malleability — a measurement on the character.** How strong-willed vs impressionable she is.
   Stored as a plain measurement (nouns for state), minted at creation.
3. **Personal magnitude of a lived event — judged per-perceiver.** A stolen muffin is nothing to Mara
   and shattering to the starving orphan; "partner's death because of YOUR mistake" — the guilt is the
   payload. So magnitude cannot be an objective stamp on the event: her already-open cognition seat
   (B-11 — updates only on perceived triggers) proposes it as a typed tier, reasoning first (§1),
   engine validates enum + provenance. No new seat, no mood-watcher.
4. **The engine composes the dials.** magnitude ≥ her malleability threshold → a core change is
   LICENSED (typed, provenance to the event). Below threshold → the experience lands in the
   belief/memory layer only — recorded forever, but it does not rewrite who she is. Drift is blocked
   by construction: the muffin structurally cannot rewrite her core. No rate-limiter needed.
5. **Sub-threshold experiences ACCUMULATE (slow per-trait pools).** A hundred small kindnesses CAN
   soften a rigid character: consistent sub-threshold impacts accrue in a slow pool per trait (scaled
   by her malleability); a filled pool licenses a gradual core change with provenance to the
   accumulated events. Same pool shape as §7 — reuse, not new machinery. One-off noise still cannot
   move her. (Malleability itself eroding under repeated hardship: possible phase-2 refinement,
   deliberately deferred.)

**What "always respected, evolving" now means mechanically:** respected = the core is in every prompt
and no seat may rewrite it by whim (no self-modification, no idle reflection loop); evolving = every
change has a licensed cause you can point to — a big event that cleared her threshold, or months of
consistent small ones that filled a pool.

## 9. Colliding attempts: ONE COMBINED RULING

When attempts genuinely collide in the same instant — a held wind-up plus a reaction (Jonas cuts in
as Kade presses on; Mara caught between) — ALL coupled attempts and all involved parties' truth-side
state go into a **single judgment**, and out comes **one set of typed events** covering the whole
exchange ("Jonas wedges in; Mara shuts her mouth; Kade's question dies half-spoken").

**Why.** Sequential rulings let call order decide the contest: whoever is judged first wins, and the
first ruling never weighed the counter-effort. And the held-outcome principle demands the reaction be
*inside* the resolution — sequential structurally cannot do that (the outcome is canon before the
reaction exists). Coupling is detected mechanically (held wind-up + reaction on the same
moment/participants); ordinary non-colliding chain steps stay sequential exactly as today.

**Wall note.** One combined call sees several actors' private state — and that is fine HERE: the
resolver is the referee, truth-side, licensed by design to read the involved parties' full canon. The
perception walls protect the character-mind seats (cognition, decompose, narrate), never the referee.
This closes FINAL-resolve OPEN #3.

## 10. Long-horizon memory: SUBJECT-TRIGGERED RETRIEVAL

At beat 500 an NPC's call cannot carry thousands of perceived events, and past some size more history
makes the model worse. What fills her mind per call:

- **Always:** personality core + backstory (the cached prefix).
- **Recent:** the last stretch of what she perceived (the append-only tail).
- **The old:** retrieved by **subject links** — records ABOUT the entities in the current action/scene
  are pulled in mechanically (the same id-intersection as §5/§6). The betrayal surfaces the moment
  Kade walks back into her tavern. Which is how memory feels: you see the face, you remember.

"Nothing is forgotten" therefore means: **nothing is ever unreachable — not everything is always
present.** The database keeps every record forever; presence in a mind is triggered by who and what is
actually in play. Accepted v1 loss, named so nobody pretends otherwise: free association ("that
smell — like the night of the fire") is not id-linked and will not surface mechanically.
This closes FINAL-commit-perception OPEN #4.

---

## Honest remainders (NOT resolved by this session)

- Decompose output JSON schema; UNRESOLVED clarify mechanics; the NPC-attempts path through decompose.
- Perception fan-out fidelity rules (distance/senses/language degradation); narrator payload
  composition; how hidden truths later surface.
- The ruling's repair-loop details; prompt shapes; per-seat model choices re-validated against these
  mechanisms.
- The €10/user + 30s/beat arithmetic against the final seat count (batch + isolated + combined
  rulings + narrate, with prefix caching) — must be sanity-checked at implementation.
- Tier-1 measurements: engine DDL columns vs validated JSONB (D-4 tension) — not grilled.
- No play surface exists (frontend is Compendium-only) — sequencing not grilled.
- Deferred by choice: malleability erosion; associative memory; journey-split; encumbrance gradient.
