# DRAFT — The Drowned Lantern: souls, secrets, and seed content

> **Status: CLAUDE'S DRAFT — awaiting founder corrections.** This is proposal, not canon. Every trait,
> secret, and backstory event below is a starting point for Brian to cut, sharpen, or overrule — his
> corrections ARE the content. Shapes follow the Personality Module (RULINGS-2026-07-23 §8): authored
> core grounded in backstory events (each trait explained by a minted canon event), malleability as a
> measurement, secrets as private knowledge with subject links (about-ness hard rule, §6). Stations
> H/I convert the approved version into seed rows (`personality_core`, `trait_provenance`,
> `perception_record` + `perception_subject`, `entity_registry`).

## The scene (from the FINAL loop PRD's worked example, expanded)

The **Drowned Lantern**, a dockside tavern in the harbor quarter of **Vael**. Low beams, salt-rot,
one hearth, a bar with a hatch, a back door to the alley. Night. Rain coming.

**Scene tension at mint: `tense`** (30s beat budget) — the room reads calm, but two of the four
people in it are pretending.

## KADE — the player character (premise, not a mind)

A hunted courier. Three nights ago he watched the harbormaster's men burn a warehouse with people
inside; the note he carries proves who paid for it. He came to the Drowned Lantern because five
years ago Mara knew him under another name — though he doesn't know if she remembers, or what she'd
do about it. The player writes Kade's will; the world writes everything else.

- Carries: **art_note** (a sealed note, size 1, weight ~0) — the proof.
- Knows: Mara (from five years ago, as "Reyna's brother"), the tavern, the harbor quarter. Does NOT
  know Jonas beyond "the muscle," does not know the hooded woman at all.

## MARA — keeper of the Drowned Lantern

**Core traits** (each traces to a backstory event below):

| trait | value / manner | grounded by |
|---|---|---|
| guarded | answers questions with questions; volunteers nothing | M-E2 |
| dry-witted | deflects with humor before she deflects with silence | M-E1 |
| loyal_to_jonas | treats Jonas as family; will not see him harmed | M-E3 |
| distrusts_authority | harbormaster's men drink free — and learn nothing | M-E2 |
| steady_under_pressure | the last person in the room to raise her voice | M-E1 |

**Speech manner:** short sentences; harbor slang; calls strangers "sailor" regardless of trade;
never says a name she wasn't given.

**Malleability: 0.25** — stubborn. Sub-threshold experiences pool slowly; only something on the
scale of losing the tavern or losing Jonas rewrites her core.

**THE SECRET (private knowledge, subject-linked to Kade):**
> Mara recognizes Kade — he is "Reyna's brother," the boy who helped her sister's family flee Vael
> five years ago. She owes that family a life-debt she has never spoken aloud. She presents as a
> stranger to him: if the wrong people learn she knows him, the debt gets them both killed.

Mechanically: a `perception_record` private to Mara, `perception_subject → Kade`, plus core-adjacent
knowledge (`knows_kade_as: "Reyna's brother"`). The gap between this truth and her stranger's face is
the scene's deception engine — she IS afraid when he walks in; observers perceive a keeper wiping a
tankard.

**Backstory events** (minted canon at world creation, `trait_provenance` rows):
- **M-E1** (~20 years ago): Grew up behind this bar; her father ran it and taught her that a keeper
  who reacts is a keeper who's already lost. → dry-witted, steady_under_pressure.
- **M-E2** (~6 years ago): The harbormaster's predecessor shook the tavern for protection money and
  took her father's savings; the watch shrugged. Her father died the winter after. → guarded,
  distrusts_authority.
- **M-E3** (~4 years ago): A dock brawl left Jonas half-dead outside her door; she stitched him up
  and gave him work when no one else would. → loyal_to_jonas.
- **M-E4** (5 years ago, PRIVATE — the secret's provenance): She hid Reyna's family in the cellar
  for nine days while the harbor was watched; Reyna's teenage brother — Kade — ran the messages
  that got them out. → the life-debt, knows_kade.

## JONAS — the muscle by the bar

**Core traits:**

| trait | value / manner | grounded by |
|---|---|---|
| protective_of_mara | reads every stranger as a threat to her first, himself second | J-E1 |
| slow_to_speak | acts before he explains; three words where others use ten | J-E2 |
| brawler_not_killer | ends fights; doesn't start them; hates blades | J-E2 |
| debt_of_gratitude | the tavern is the only place that ever took him back | J-E1 |

**Speech manner:** monosyllables; states facts, not opinions ("Bar's closed." "You're leaving.");
uses names only for Mara.

**Malleability: 0.45** — steadier than he looks, but a man rebuilt once already; the right person
could reach him.

**Private knowledge (subject-linked to Mara):** he knows Mara keeps a knife under the till and a
debt she never explains — he's seen her go pale twice in four years at faces from the harbor, and
he's learned not to ask, just to stand closer. (He does NOT know who Kade is.)

**Backstory events:**
- **J-E1** (~4 years ago): Beaten near to death over a fixed fight and left in the alley; Mara took
  him in. → protective_of_mara, debt_of_gratitude.
- **J-E2** (~10 years ago): Prizefighter until he killed a man in the ring with one unlucky blow;
  never threw a clean punch for money again. → slow_to_speak, brawler_not_killer.

## THE HOODED WOMAN — corner table (deliberately thin)

**Core traits:** watchful; unhurried; pays in coin too clean for this district.
**Malleability: 0.6.**
**Private knowledge (subject-linked to Kade):** she is an agent of the harbormaster's paymaster,
carrying a description of a courier — young, dark-haired, moves like a dock rat — and a purse for
whoever confirms him. She is not sure Kade is the man. Yet.
**Backstory:** ONE minted event (H-E1, three days ago): took the contract in a counting-house above
the silk quay. Everything else about her stays unauthored — she is Station G/E fuel (telegraphs,
held outcomes), and thinness is the point at this stage.

## Artifacts + places (seed rows)

- **art_note** — sealed note (size 1). Tier-2 flavor: `sealed_with_gray_wax`. What it proves lives
  in its Tier-2 description; no engine meaning.
- **The tavern room** (scene) — coordinates within the harbor quarter; contains: the bar, the
  hearth, four tables, **portal_front_door** (Portal: connects [tavern, dock_street], open,
  unlocked), **portal_back_door** (Portal: connects [tavern, alley], closed, unlocked),
  **portal_cellar_hatch** (Portal: connects [tavern, cellar], closed, LOCKED — Mara has the key;
  the cellar is where M-E4 happened).
- **dock_street**, **alley**, **cellar** — minimal location stubs so movement has somewhere to go.

## Why this shape (for the founder's review, per the example+reasoning rule)

- Every trait has a **provenance event** — nothing is a floating adjective (D-11 applied to
  character; the Personality Module reads these as the core's cached prefix).
- The **secret is knowledge with subject links**, not a flag — so the isolation lookup
  (RULINGS-2026-07-23 §5) pulls Mara into her own cognition call the moment an action touches Kade,
  and the batch never sees it.
- **Jonas knows OF a secret without knowing IT** — so his cognition can act protective without the
  wall leaking Mara's truth: two minds, two different incomplete views of the same room.
- The **hooded woman's uncertainty** ("not sure. Yet.") is the scene's fuse: her confirming Kade is
  a natural held-outcome / telegraph moment, and the player's behavior is what feeds it.
- The **locked cellar hatch** puts a Portal with a real Tier-1 `locked=true` in the first playable
  room — the two-tier wall gets exercised by the very first "I go down the hatch."

## Open for the founder (answer inline, or overrule wholesale)

1. Names/register: keep Vael + these names, or rename to taste?
2. Mara's malleability 0.25 vs Jonas 0.45 — does that ratio match your feel of them?
3. The hooded woman: keep her paymaster-side, or would you rather she be hunting the harbormaster
   too (a possible ally misread as a threat)?
4. Kade's note: proof of the warehouse burning, or would you rather leave WHAT it proves unauthored
   until play demands it?
5. Anything here that reads as MY voice instead of your world's — mark it and I'll cut it.
