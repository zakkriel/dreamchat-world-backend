# FINAL — World & NPC Cognition (SCAFFOLD — locked rulings + open questions)

**Founder goal this file serves:** a world that does NOT revolve around the player; NPCs completely
autonomous, each with an always-respected, evolving personality. CONFIDENCE AT CHECKPOINT: ~50% —
the orchestration is locked; the seats themselves are SPEC-012, undesigned.

## Locked
- **World-first, every step:** before each player step resolves, the World Actor and present NPCs get
  their word. No sacred first action.
- **Telegraphed intent as input:** cognition seats receive the imminent step's wind-up ("he's turning to
  leave"), NOT committed history only. They react to what's coming.
- **No bypass:** an NPC/world act is an attempt into the SAME pipeline (floor → gate → resolve → commit).
  A "trusted NPC" path is a named consistency hole.
- **Reaction depth 1:** the player reacts to held outcomes; NPCs don't re-react to the reaction.
- **Tension review rides these seats** on their steps (no dedicated call).
- **Held outcome / telegraph:** a disruptive act commits its wind-up, the player gets a reaction beat,
  resolution runs with the reaction in it. No fait accompli.
- **One cognition call PER ACTION (2026-07-23):** not per text, not per NPC. Per action each present NPC
  sits in exactly ONE seat — the shared batch (public moment only) or an isolated call (her private
  knowledge rides alone). Split decided by id-intersection over about-ness links (mechanical, no
  judgment); secrets never enter a shared prompt — the wall holds by construction. Prefix caching makes
  it affordable (append-only canon is cache-native). See RULINGS-2026-07-23 §5–§6.
- **Intent is read per-NPC, at resolve (2026-07-23):** there is no single "the intent" — each listener
  reads the same words her own way. The decomposer never touches it. See RULINGS-2026-07-23 §4.

## RULED since checkpoint (2026-07-23 grilling — see RULINGS-2026-07-23-grilling-session.md)
1. → **INPUT ruled (shape):** cached prefix = personality core + backstory events; + recent perceived
   events; + old records retrieved by subject links (§10); batch seats get the public moment only,
   isolated seats add the NPC's own private records (§5). Exact payload assembly remains implementation.
3. → **Personality ruled: the Personality Module** (own module per D-2): authored core grounded in minted
   backstory events; malleability as a measurement; per-perceiver event magnitude judged by the
   already-open seat; engine composes magnitude vs malleability threshold to license core changes;
   sub-threshold experiences accumulate in slow per-trait pools (quiet arcs). See RULINGS §8.
4. → **Scheduling ruled:** the ROOM runs every step (locked above); the world BEYOND the room = pending-
   events ledger (known futures, fired on clock-crossing) + World Actor spontaneity on rising pressure
   accrued on WORLD-TIME, with independent small/medium/large magnitude pools. Intra-tick ordering
   candidate: the batch's fixed generation order. Presence caps stand (6 acting / 10 known). See RULINGS §7.
5. → **World Actor ruled:** world-scope context, intrusions may be unrelated to the scene (disruption is a
   feature, no mood filter), brings non-present NPCs in, same no-bypass pipeline, scene perceives only the
   perceivable edge. See RULINGS §7b.

## OPEN (remaining)
2. Cognition OUTPUT: the structured attempt shape; does it pass through decompose or arrive typed?
   (Same open as FINAL-decompose #3.)
- Payload assembly details; the batch prompt's exact shape; pool/pressure tuning values (data, from play).
