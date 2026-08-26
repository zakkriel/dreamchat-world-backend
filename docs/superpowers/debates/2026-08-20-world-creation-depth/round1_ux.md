# Round 1 — seat: ux (Creation UX Expert)

## 1. Thesis

The founder's ask is not an interview problem. It is a **confirmation problem**, and today the product has no confirmation surface at all. Walk the shipped journey: the user writes a brief, optionally answers one question at a time (`worldinterview.go:5-8`), then watches `working` frames whose only payload is a verbatim `stated` line (`world_genesis_frame.v3.schema.json:24-28`), then picks an arrival at the `choice` frame, then stands in the world. At **no point** does the user see the *system* the seat inferred from "heavily ruled by a social caste called Alphas" — the genesis document is deliberately never served (AC-7, re-affirmed at `2026-08-21-durable-worlds-design.md:111-112`), the build is one call with no repair loop (`worldgenesis.go:171-175`), and the world is immutable afterward ("Editing a world after it exists" is a named non-goal, `prd_world_creation.md:38`). So the correction window for the *most consequential* inference the product makes — what the world's law is — is **zero seconds wide**. If the seat reads "Alphas" as flavor, the user discovers it three beats in, and their only remedy is to pay for a whole new world.

My position: depth extraction is worthless UX-wise unless the user can **see the inferred law before spend, correct it in their own words, and then feel it at arrival**. And one grounded fact makes this urgent: the pipeline already *pretends* to extract social order and then throws it away. `cast[].standing` — "What they do and where they sit in this world's order" (`world_genesis.v1.schema.json:130-134`) — is required by the schema (`:117`), refused when empty (`worldgenesis.go:310-311`), and **never written by any commit path**: `worldgenesiscommit.go` contains no reference to `Standing`; the traits map it builds carries only `speech_manner` and traits (`worldgenesiscommit.go:514-525`). Today the one field where "Alphas rule" would live is validated, then dropped on the floor. Any confirmation surface built on fields like that would be confirming a lie.

## 2. Concrete mechanisms

### M1 — The playback screen: strikeable statements, before genesis

After the brief (both lanes), before the expensive genesis call, one cheap seat turn renders the brief's **entailments as world-language statements** the user can strike or rewrite:

> — Alphas rule here. ✓
> — At least one caste sits beneath them, unnamed so far. ✓
> — Rank is visible on sight. [change: "only insiders can tell"]
> — Breaking rank is punished, and everyone knows it. ✓

This is not a wizard and not a form. It is the interview's exact machinery **inverted** — statements instead of questions — and it reuses the existing stateless carriage: amended statements travel as `InterviewAnswer` rows, which `buildWorldGenesisPrompt` already threads into genesis where "ANSWERS … are the user's own words and outrank your judgement completely" (`world_genesis.txt:2`, `worldgenesis.go:164-169`). No new pipeline, no session, no table — the same "client carries everything" discipline (`prd_world_creation.md:158`).

GA-2/GA-3 compliance is inherited, not invented: statements derive from *this* brief's content exactly as interview questions must ("Every question you ask must be answerable only about THIS world", `world_interview.txt:4`). A statement that would be true of any brief is malformed for the same reason a slot-filling question is.

Friction contract: **one screen, all statements pre-accepted, "Build now" always live** — the interview's "user is never trapped" rule (`prd_world_creation.md:157`) applied verbatim. Fast lane stays one tap for a user who trusts the defaults; the screen costs interaction only when the user disagrees, which is precisely the moment interaction is worth its price.

### M2 — Shown ⇒ committed ⇒ consumed (the anti-`standing` rule)

A playback statement may only be rendered if it traces to a field that survives commit into a location a play-time seat reads. Two real precedents prove the pattern exists: `hiding` becomes a private `direct` perception (`worldgenesiscommit.go:556-557`), `speech_manner` lands in the traits map "because the cognition seats read it" (`worldgenesiscommit.go:514-518`). `standing` fails the rule today. I place this as a **requirement on simarch and extraction**: whatever representation they converge on (rules-as-events, rules-as-perceptions, whatever survives the gate), the UX contract is that the confirmation surface reads from that representation's inputs — never from decorative fields. Confirming prose that evaporates at commit is worse than no confirmation: it teaches the user their corrections are ceremony, the exact failure `world_interview.v1.schema.json:21` warns about for padded questions.

### M3 — The interview asks about *violation*, not taxonomy

The interview prompt already ranks questions by "what changes the world most … the pressure everyone is under" (`world_interview.txt:3`). For a brief that implies a rule system, the highest-yield single question is the **enforcement question**: "What happens to someone who talks back to an Alpha?" — because its answer simultaneously generates history events, at least one `hiding`, and per-caste `knowledge` entries, the three channels that actually reach canon (`world_genesis.v1.schema.json:214-270`). What the interview must *not* do is taxonomy-farming ("name the lower castes", "how many ranks are there") — where the brief is silent the seat decides (`world_genesis.txt:2`), and counting is the engine's business anyway (`prd_world_creation.md:70`). One new question *class*, not a longer interview; M1 replaces the broad questions the interview would otherwise burn turns on.

The `implication` field the schema already carries per option (`world_interview.v1.schema.json:48-52`) is the ownership handle: "what choosing this would make true" is the user's contract line. Depth work should treat it as load-bearing, not optional garnish.

### M4 — Honest failure must be distinguishable from honest completion

Today a seat error or malformed turn silently collapses to `Done: true` (`worldinterview.go:71-84`, comment at `:54-57`). Right instinct — never punish the user for infrastructure — wrong signal: a Custom-lane user who wanted grilling gets silently downgraded to fast lane with no indication anything was skipped. That is *under-asking that ships a world the user didn't mean*, my seat's own named failure mode. The turn needs to distinguish "nothing genuinely open" from "could not author the next question", so the surface can say "I couldn't find the next question — build with what you have, or ask me to try again." One boolean of honesty, same posture the build stream already adopted when a silent minute read as a hang (AC-9 amendment, `prd_world_creation.md:72`).

### M5 — The system must be *felt* at the two readbacks that already exist

The build stream and the kickstart are the only mirrors the user gets, so depth must show in both. `working` frames name what was committed, verbatim, in the world's language (`prd_world_creation.md:165`; `world_genesis_frame.v3.schema.json:27`) — a caste world's frames should say so: "the law of the quarter: no unranked speaks first." Honest by construction, since it names authored content (law 2 holds). And the three `arrival_candidates` — already required to be "implied by the cast, the places or the history" (`world_genesis.txt:27`, schema `:304-319`) — are the natural systemic probe: in a caste world the three doors should be three *positions in the system* (marked low-caste, passing as Alpha, exempt outsider), because choosing a door is the first moment the user *feels* that the system exists. The kickstart's grounded immutability machinery (`worldkickstart.go:22-31`, populated-places belt `:42-61`) needs no change for this; it is a prompt-emphasis change on an existing surface.

## 3. The three hardest attacks, pre-answered

**From extraction: "Your playback screen consumes a reliable systems-inference that doesn't exist. Users will anchor on wrong statements or rubber-stamp them."** Backwards: strikeable statements are how you make a *fallible* extractor shippable. A wrong statement shown pre-spend costs one tap to fix; a wrong inference discovered at play costs the whole world, because editing is forbidden (`prd_world_creation.md:38`) and genesis is one-shot (`worldgenesis.go:171-175`). Rubber-stamping is the *fast lane working as designed* — the recommended-default pattern is already the product's answer to users who don't want to decide (`world_interview.v1.schema.json:55`). The surface is tolerant of extraction error by design; that tolerance is what buys extraction the right to be imperfect at launch.

**From gamedesign: "Confirmation is a spoiler machine — you'll show the player the world's secrets and kill discovery."** No. The playback shows the **constitution, never the plot**. It runs *before* genesis, so no authored secret exists yet to leak; it renders only entailments of the user's own words. The AC-7 boundary — the genesis document appears in no response body, ever (`durable-worlds-design.md:111-112`) — stays intact. The user owns the world's law because they wrote it; they discover its people, secrets and history in play, exactly as now. What each caste knows/believes about the others stays inside `history.knowledge` (`world_genesis.v1.schema.json:240-267`), unserved.

**From simarch: "Statements the user confirms are still prose. You've built a consent screen for vapor — see your own `standing` finding."** Conceded, and weaponized: that is exactly why M2 is stated as a hard gate, not a hope. The `standing` drop proves the failure mode is *already shipped*, with zero UX on top of it. My claim is about ordering: the confirmation surface and the enforceable representation must land as one contract — shown ⇒ committed ⇒ consumed — or neither is worth building. I will not defend a playback screen over fields that evaporate; simarch should not defend tables no surface lets the user correct before they're immutable.

## 4. What I would cut

- **Any settings panel, slider, genre picker, or "world type" chooser.** A fixed control *is* an ontology of what worlds usually have — GA-2/GA-3 dead on arrival (`prd_world_creation.md:177-180`).
- **Multi-step confirmation wizards** (one screen per places/cast/history part). Wizard hell, and the cast/history parts are AC-7-secret anyway. One screen of law, build-now always live.
- **Taxonomy questions in the interview** ("what are the lower castes called?", "how many ranks?"). Naming silence belongs to the seat (`world_genesis.txt:2`); counts belong to nobody (`prd_world_creation.md:70`).
- **Post-arrival world editing** as a correction channel. Append-only canon already answered this (`prd_world_creation.md:38`); pretending otherwise re-litigates a non-goal.
- **Rendering the brief or the genesis doc back as world content.** The prompt is not the fiction (open question 1's recommendation, `prd_world_creation.md:81`); the playback renders *entailments in world language*, never the artifacts themselves.
