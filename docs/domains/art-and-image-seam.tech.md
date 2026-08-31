# art-and-image-seam · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** IP-2 · The seam, from the engine's side ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, write and read paths, validation, traps.
`art-and-image-seam.product.md` holds what it means; `art-and-image-seam.seams.md` holds what
crosses its boundary.

Line numbers are as of 2026-08-27; re-locate by grep before relying on one.

---

## Transformation is anchor conditioning, and `transform_only` is a 501

Measured 2026-08-31 against production. Do not reach for `render.transform_only: true` — it returns
**501, not implemented**. Nothing is coming; it is not a gap to work around.

**The transformation workflow is `subject.anchor_asset_id`.** Reference-conditioning is the platform's
actual design (`image:ADR-017`), so pointing a new generation at an existing asset IS how you produce a
related image rather than a fresh one. It also skips the anchor bootstrap:

| | images | time | cost |
|---|---|---|---|
| a subject nobody has drawn | anchor **then** portrait | ~12 s | $0.08 |
| a subject conditioned on an existing asset | one generation | **~6 s** | **$0.04** |

Also measured, and worth knowing before quoting any latency here:

- **generation itself is ~6 s** — `0.7 s` to accept, terminal at `6.04 s`
- **the delays are ours**: `artCommissionInterval = 2 * time.Minute` for the sweep, plus a poll backoff
  of 1 s ×1.8 capped at 15 s with full jitter. **A 6-second image can take over two minutes to surface.**
- generation is **reference-conditioned**: without an anchor it fails in 1.5 s with
  `missing_reference_assets`. Anchor → portrait is sequential per subject; across subjects it is parallel.
- **$0.04 per image**, not the $0.01 in the quickstart example.

Anything a player is waiting for needs a foreground path — commission on demand, poll tightly for the
first ~10 s — and the 2-minute sweep is correct only for work nobody is watching (`SPEC-045`).


## Storage

**`image_slot`** — PK `(world_id, owner_kind, owner_id, variant)`. `owner_kind` CHECK
`('actor','location','artifact','world')` — a world cover is its own owner (`20260809090003`).
`variant` CHECK `('default','neutral','happy','angry','sad')` (`20260820200000`). Beside the ids:
`asset_id` (last known), `job_id` (in flight), `last_error`, and **`idempotency_key` stored WITH
`issued_at`** — the one home for the pinning fact: the platform's idempotency key hashes the whole
request body, so an envelope rebuilt with a fresh timestamp is a different body under the same key
and returns `409 idempotency_conflict`; a retry replays the identical envelope verbatim
(`imageclient.go`, `newGovEnvelope` comment; `20260808100005`). Two partial indexes drive the
drain: in flight (`job_id IS NOT NULL`) and unfilled (`asset_id IS NULL AND job_id IS NULL`).
Wall-clock timestamps are telemetry, never in-world time (`B-5`). Full DDL:
`grep -n 'CREATE TABLE public.image_slot' core/db/schema.sql`.

## The write path — a reconciler, two triggers

`core/api/artcommission.go`. Genesis kicks `kickArt` after the world frame, detached
(`worldgenesishandler.go:578`); a ticker sweeps every non-archived world every 2 minutes. `kickArt`
is a `var` for a doctrine reason: a direct call is unobservable, and a silent missing call is
exactly the failure that started this (failure-log row 40) — so genesis-commissions-art **is a
test**, `genesisart_test.go`. Order inside a sweep: `pendingArtCount` (pure SQL, BEFORE any HTTP —
the fills open with `ensureStyle`, a round trip) → fill portraits/scenes, paged, with a pass
ceiling → 20-minute per-world timeout, `inFlightWorlds` keeps one sweep per world. All outside the
genesis transaction (`D-8`): a provider outage delays pictures, never destroys an authored world.

**What gets a picture: authored fiction, or nothing** (`imagehandler.go`, grep `ONE RULE DECIDES`,
~:655-672). A place renders from `location_state.attrs.description`, a world from its tagline;
waystations and container areas carry no description and are structurally not targets. Portals are
excluded structurally (`attrs ? 'connects'`), not textually — the descriptor test alone let three
doors through and billed for them.

**Generation reads `*_state`, never `perception_record`, on purpose** (`imagehandler.go` ~:669-672):
*"a picture is of the THING, not of anyone's opinion of it, and the prompt goes to a private
service, never to a player. B-1 governs what reaches the FRONTEND."* This is the one home for the
asymmetry; `seams.md` and the perception package's Art & Assets row point here.

## The read path

- **`fn_image_ref(world, owner_kind, owner)`** (`core/db/schema.sql`, grep the name) — asset id +
  path back to this service, never a URL. `NULL` is the ordinary state; the frontend renders a
  placeholder and swaps on a later read (`D-8`), no polling.
- **`fn_sprite_set`** — all four emotion variants or `NULL` (`CASE WHEN count(*) = 4`): three right
  faces and one wrong one is worse than waiting.
- **Two fields are deliberately not perception-scoped, and they are the only two:** the portrait
  (`fn_actor_page`, comment at `schema.sql:844-849`) and the scene backdrop
  (`core/api/scenehandler.go:35-38`). Same reason both times: the room you are standing in and the
  face in front of you are not secrets; the wall governs what a viewer *knows*. The existence gate
  is untouched — `fn_actor_page` returns `NULL` via `fn_entity_visible` before the image field is
  ever built (`schema.sql:825`).

## The client

`core/api/imageclient.go`, written against the platform's `integration-quickstart.md` — "copied
from a sequence they executed against a real stack, not inferred from an OpenAPI file" (header,
~:22-24). Nil client = platform not configured = every reference stays null; disabled is normal.
`do` never retries — retry policy belongs to the caller that knows whether a request is safe to
repeat. Polling backoff is required client behaviour: the limiter counts denied requests, so naive
fixed-interval polling pins your own window. Shape: start ~1s · ×1.8 · cap ~15s · FULL jitter
(`pollBackoff`, ~:626-655), 40 attempts. The two 429s differ: `Retry-After` is authoritative on
`rate_limit_exceeded`, deliberately absent on `concurrent_jobs_exceeded`, which clears on a job
reaching terminal state (~:169-171). `errAssetGone` has two shapes: `404 not_found`, and **200
with a retired status** (see Traps). `anchor_asset_id` is always sent: the reuse key folds it, and
omitting it serves portraits drawn from a replaced anchor.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-P021` | Reconciler; cheap-check-first; outside the genesis transaction. | See `product.md`'s table — one home. |
| `ADR-P023` | `artstyle.go` is the one place a look lives; catalogue served (`GET /worlds/art-styles`), `look` unexported; custom styles named by hash of normalised prose. | Prompt prose ships to clients and can never be tuned server-side; unhashed customs never share a cache. |
| `image:ADR-006` | Async jobs, pulled to terminal status. | Polling a terminal job, or trusting a push, wastes the rate window or waits forever. |
| `D-3` | Ids over the wire; the mapping lives in `image_slot`. | A persisted presigned URL rots in ~15 minutes. |
| `E-1` | The envelope is built engine-side before any generation call (`newGovEnvelope`, all four call sites). | The platform is handed policy it must never decide. |
| `B-5` | Slot timestamps are telemetry. | Wall-clock leaks into in-world surfaces. |

### What you may not decide alone

1. **Adding a variant** to the closed CHECK set, or **a style preset** — and never one named for a
   genre (`GA-2`; `ADR-P023` names `cyberpunk` as the violation).
2. **Making generation read perception** — reopens the documented asymmetry (§The write path).
3. **Adding a commissioning call to a creation path** (`ADR-P021`'s own header consequence).
4. **Building the async channel or a webhook consumer** — both deliberately absent (`product.md`).
5. **Inventing classification values** — CG-1 owns the envelope's meaning (`seams.md`).

## Validation for this domain

`cd core/api && go test -run 'Art|Image|Genesis.*Art|Sprite' -count=1 .` — always `-count=1` (the
suite is seed-dependent). The live handshake `imagelive_test.go` is gated on `DREAMCHAT_IMAGE_LIVE`
and needs a running platform; it never runs in CI.

- **What counts as evidence:** this domain fails silently — `null` is the ordinary state, so a
  dead reconciler is indistinguishable from "no art yet". That is why `genesisart_test.go`
  observes the `kickArt` seam itself, and why `reclaimed` exists: without it the mosaics survived
  a live art flip invisibly.
- **What counts as ceremony:** asserting `image: null` — it passes with the reconciler deleted.
  Same for a fill count that never asserts which owners were excluded: the count looked right
  while three doors were billed.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **A swallowed platform error costs real money.** `ensureStyle` once ignored a 403 (token missing `styles:read`) and minted a fresh profile per call — 25 identical rows, and the reuse key folds `style_profile_id`, so the cache never hit: every regeneration billed full price. | `imageclient.go` (grep `styles:read`). |
| **Retired-in-place returns 200 with working URLs.** Nothing 404'd after the fal flip, so the fill query never saw an empty slot. | `imageclient.go` (grep `retired IN PLACE`); `reapRetiredAssets`. |
| **A fresh timestamp under an old key is a 409.** | §Storage — the one home for the pinning fact. |
| **The descriptor test lets doors through.** | §The write path; `TestPendingArtCount_IgnoresAPortal` (`artcommission_test.go:78`). |
| **`/images/regenerate` must refuse BEFORE deleting** — clearing slots with no platform configured is a destructive no-op wearing a 200. | `imagehandler.go:145-146`. |
| **Route order:** `/worlds/art-styles` is matched FIRST or an id matcher reads `art-styles` as a world id. | `main.go:41-43`. |
| **A sixth art style once broke a test `ADR-P023` promised could not exist** — a test hardcoded the list. Tests derive from `ArtStyleCatalogue()`. | failure-log row 4. |

## Open questions

1. **Classification is a constant.** `newGovEnvelope` hardcodes `cls_poc_default` + `private`
   (`imageclient.go:208-209`) while `E-1` requires real classification before any media request.
   Who mints classification ids when shareable/public worlds arrive — and is that CG-1's build or
   this domain's wiring? (CG-1's package carries the meaning side of this question.)
2. **The scene provider pin is a disclosed deviation.** `DREAMCHAT_IMAGE_SCENE_PROVIDER` exists
   because an unpinned scene request resolves MOCK even with real keys present; "the real fix is
   theirs" (`imageclient.go:60-63`). When the platform fixes its default, does the knob retire?
3. **Signing is a stub** — `stub-unsigned-v1` passes any non-empty string pending the platform's
   `TODO(core-signing)` cross-repo contract (`imageclient.go:71-76`). Who designs it, and where
   does the canonicalization spec live?
4. **World covers ride `place_scene`** because the platform's `content_class` enum has no
   world-cover member — "switch if they ever add one" (`imagehandler.go:753-756`).
