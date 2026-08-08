# HANDOVER — the three-repo integration (2026-08-08)

**Supersedes `2026-08-07-after-the-connection-handover.md`.** That document ends with
"the founder has never played it" and "two things block a real playthrough". Both are
now false, and one of its claims was wrong when written — see §6.

**For the agent picking this up.** The backend, the frontend and the image platform are
integrated and driven end to end. Read §1 for what is true now, §5 for what is
deliberately absent, and §6 before trusting anything you remember.

---

## 1. State in one line

`main` = **`73ad799`**, CI green, zero open PRs, one branch. Fifteen PRs landed this
session (#37–#51). The founder's worked example — walk out, get interrupted, restate,
arrive — is **drivable end to end**, the frontend renders it, and actors have real
generated portraits.

---

## 2. What shipped, one line each

| PR | |
|---|---|
| **#37** | **CORS** (SPEC-021). Exact-match allowlist from `DREAMCHAT_CORS_ORIGINS`; `*` refuses to boot. Before this a browser could not call the API at all — the preflight 404'd because the router only matched GET/POST. |
| **#38** | **The fake bridge committed nothing, ever.** The factory built the decompose seat with a nil scripted table, so every input decomposed to `[]`. New `fake-intent` driver binds real ids from the candidate whitelist; `fake-resolve`/`fake-cognition` registered. |
| **#39** | **SPEC-030 — a way out of the room.** `ActorMoved` could only ever target the room you already stood in. Portals of the current room and the rooms they connect to are now candidates. |
| **#40** | **SPEC-029 — Compendium lenses computed from perception.** `current_synthesis`, `last_known_status`, `known_artifacts`, `key_actors`, `inline_links`, real `decay.stale`, knowledge grouped by source event. |
| **#41** | **The Harbormaster's Office.** A destination far enough that walking there starts a Journey; Dock Street gained a tension so its budget stopped being infinite. Second hooded figure added, making UNRESOLVED reachable in play. |
| **#42** | **The journey wedge.** A refused world-actor intrusion failed the whole beat, and because the pressure roll is pure, the retry rolled identically — a deterministic livelock. Refusal is now separable from failure (`errIntrusionRejected`). |
| **#43** | **SPEC-031 tuning** (founder-ruled): `medium` `climb_chunk_ticks` 3600 → 300. |
| **#44** | **`beat_frame/2`** — `unresolved_candidates` as `{id,label}`, and in-world label continuity via a `BEFORE INSERT` trigger (every commit path was writing `in_world_label` NULL). |
| **#45** | **Distinguishing detail in colliding labels** (founder-ruled). Two things wearing one name gain perceived detail — "by the bar" vs "by the ballast crate" — on both the display list and the candidate whitelist. |
| **#46** | **`UNRESOLVED.reference` quotes the player**, not a candidate label. |
| **#47** | **SPEC-032 — the way behind you.** A minted waystation was wired only to the goal, stranding the traveller: hard error, dead beat, and permanently null `where_label`. |
| **#48** | **SPEC-028 world registry + SPEC-019 theme + the viewer seam.** `GET`/`POST /worlds`; `ResolveViewer` reads `world.player_entity_id`. |
| **#49** | **Pin**: an interrupted journey still names its destination. |
| **#50** | **Image Platform client**, `image_slot` storage, the fetch redirect and the explicit generation trigger. |
| **#51** | **`scene_current/2` + `actor_page/2`** carry the `image` field. |

---

## 3. The truth now

**Worlds are real objects.** `GET /worlds` lists them with display name, theme tokens
and `playable`; `POST /worlds` creates one. A world row is a *directory entry*, not
canon — no state rides that surface.

**Identity is a fact the world records.** `world.player_entity_id` replaced the 0A stub
that looked for an actor literally named `Player` — a convention that had already
broken in the only world anyone plays, where the player is Kade. The seeded world now
answers `200` with no `?viewer=`. **This is still not auth**: it answers "who do I play
as here", never "who is calling".

**The Journey works and the world interrupts it.** Movement binds, portals gate
passage honestly (a shut door refuses with `premise_broken`), over-budget moves open a
journey, `continue` walks legs, and a beat-cutting eruption ends it with `goal_label`
intact so the player can restate.

**The contract is `beat_frame/2`**, `scene_current/2`, `actor_page/2`,
`compendium_index/1`, `timeline/1`, `location_page/1`, `artifact_page/1`,
`world_directory/1`, `world_created/1`. `make schema-contract` validates real payloads
against all of them in both directions: **32 payloads, 11/17 covered**.

**Images are integrated, pull-based.** Portraits generate through an explicit trigger,
`identity_id → asset_id` is persisted here (the platform's asset row cannot answer it),
and `GET /worlds/{w}/images/{asset_id}` redirects to a freshly minted presigned URL.

**The dev loop is honest.** `DREAMCHAT_BRIDGE=fake` now commits, moves, refuses, erupts
and journeys with no API keys. Narration is still a deterministic stub — it is a
mechanism testbed, not a playtest.

---

## 4. Standing cross-repo contracts

Break one of these and another repo breaks with you.

- **Payload versions are pinned exactly and fail the load on mismatch.** Adding a field
  is a breaking change: every payload is `additionalProperties: false`. The version
  moving IS the notification — clean cutover, no alias. The frontend has done three
  re-pins on this protocol; it works, but it costs them a PR each time, so batch.
- **`image_ref/1`**: `{"schema_version","asset_id","path"} | null`. `null` is the
  ordinary state and means "no picture yet". **Never a presigned URL** — those expire
  in ~15 minutes. Fetch `{apiBase}{path}`, optional `?tier=thumbnail|preview|final`.
- **Images are PULL.** `POST` → `202` + `job_id`, then poll to a terminal status. The
  platform's webhooks are a latency hint only and are ignored. Two traps, both real:
  the idempotency key hashes the whole body so `issued_at` must be pinned (or carried
  in the key), and polling must be bounded and jittered because denied requests count
  against your own window.
- **CORS is env-only.** `DREAMCHAT_CORS_ORIGINS`; no default, no hardcoded hostname.
- **The database is shared and stateful.** Coordinate before `make reset` when anyone
  else is driving — it wipes their world mid-session. `make migrate` is non-destructive
  and is usually what you want. Two mid-run reseeds during this session produced
  transient 500s in the frontend's logs.
- **The beat body field is `text`.** Not `input`.

---

## 5. Deliberately unbuilt, and why

- **The async channel** (`image.ready`, `projection.updated`, `backstage.applied`).
  Still absent, now with a worked example of the alternative: an image reference rides
  an existing payload as plain data, null until ready, and the frontend swaps it in on
  a later read. A channel whose only message is one a read already carries would be a
  subsystem justifying itself. When one exists for reasons that need it, images can
  ride it as a hint — the status the image platform gives its own webhooks.
- **The session model (B1).** Nothing knows who is calling. **`POST /worlds` is
  therefore unauthenticated: anyone who can reach the service can create a world.**
  Safe behind a private deployment, *not* safe on a public origin. It should be the
  first endpoint auth goes in front of, and `fn_world_directory()` is the single place
  the "worlds the caller may see" filter attaches — one `WHERE` clause and every caller
  inherits it. This is the largest known gap in the system.
- **Corrections, the correction-window frame, World Workspace, off-scene eruptions,
  multiplayer.** Untouched.
- **World templates.** `POST /worlds` authors no entities. A starter scene is authored
  fiction, and this service must not learn what a world is "usually" like (GA-2/GA-3).
- **Real faces.** The platform has `FAL_KEY` but mock wins route priority, so current
  art is placeholder. Real portraits need synthetic off plus the BFL-artifact → anchor
  → fal bootstrap. Founder's call.
- **`location_page` / `artifact_page` image fields.** They stay at `/1`. Bumping
  payloads with nothing to put in the field would cost two re-pins for nothing;
  `image_slot` already keys on `owner_kind`, so they are a field and a bump away.

---

## 6. Things the last handover got wrong, and lessons that cost real time

**The previous handover claimed the fake path made "scene state, journeys, continue,
frames and the trace all real".** The frames were. The world was not: nothing committed
(#38). Do not trust a claim about a dev path that no test exercises.

**Three defects this session were the same defect.** A fake hard-coding seeded ids,
"safe" because nothing reached it — until a later feature did. The world-actor fake
(#42) was the third instance. If a stand-in hardcodes an id, assume a future feature
will reach it.

**Inference cost more than measurement, twice.** I reported that interruption at
waystations needed a bounded exception to D-1's accessibility floor; measuring showed
an NPC *could* lawfully arrive and the real fault was a portal wired to the wrong end
(#47). I also reported the frontend's medium-tier report as a possible tuning bug;
replaying the real chance function showed medium fires 20/144 and wins the scan every
time — their extraction was mislabelling tiers. **Measure the thing before designing
around it.**

**Determinism plus a fatal path equals a permanent trap.** The pressure roll is a pure
function of committed state, so a failed beat rolled identically forever (#42). Any
"retry" of a deterministic decision that failed will fail the same way.

**The shared database is the most common cause of a mysterious red suite.** Several
tests only pass on a fresh seed and say so in their failure text. Before debugging,
re-read §2 of `docs/runbooks/full-stack-from-zero.md`.

**Hand-drive everything.** Every defect worth finding this session was found by curling
a running server or reading a real trace, not by a passing test suite. The live image
handshake found two contract facts in ten minutes that a faithful fake had missed for
an afternoon: `canonical_visual_traits` is required, and a deterministic idempotency
key 409s on the second attempt.

---

## 7. Where to look

- **Running everything**: `docs/runbooks/full-stack-from-zero.md` (verified 2026-08-08).
- **Open items and their rulings**: `docs/open-spec-items.md` — SPEC-019/021/028/029/
  030/031/032 all carry a Status line recording what landed and what was ruled.
- **The image contract**: `dreamchat-Image-Platform/docs/api/integration-quickstart.md`
  is the contract of record. Write against it, not against guesses.
- **The law**: `docs/00_strategy/06_rules_register.md`. Cite IDs in plans and PRs.
