# art-and-image-seam · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** IP-2 · The seam, from the engine's side ·
**Parent bounded context:** World Engine

A seam belongs to two domains; each row declares an expectation — one side owns a fact, the other
consumes it and must not re-derive or re-decide it. The mirror rows live in IP-1's package
(`dreamchat-Image-Platform`), CG-1's, and the surface packages; where mirrors differ, the moderator
reconciles.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **image-service** (IP-1) | `job_id`, then `GET /v1/jobs/{id}` + `/assets` to a terminal status | Pull is the contract of record (`image:ADR-006`); ignoring webhooks entirely is still correct. Ids are durable, download URLs expire — never persisted. Reuse is the default and the reuse key folds `style_profile_id` and `anchor_asset_id` — never re-request to "refresh", always send the anchor. The platform never learns what a world IS (`D-3`); this side never re-derives provider routing, cost, or asset storage. Transport discipline (429 split, pinned `issued_at`, backoff): `tech.md` §The client and §Storage. |
| consumes | **world-genesis** (WE-10) | one detached `kickArt(pool, client, worldID)` after the world frame (`worldgenesishandler.go:578`) | Genesis kicks and walks away; it never commissions art itself, never waits on it, and never wires image calls into the genesis transaction (`ADR-P021`, `D-8`). Any other creation path adds **nothing** — the ticker inherits it. |
| consumes | **the world model** (`*_state`) | authored fiction: `location_state.attrs.description`, the world tagline, `attrs->>'descriptor'` for artifacts, `attrs ? 'connects'` for portal exclusion | No description, no picture — this side never invents fiction at the boundary. The asymmetry (read `*_state`, never perception) is documented at `imagehandler.go` ~:669-672; one home in `tech.md` §The write path. |
| consumes | **content-governance** (CG-1) | what the seven envelope fields MEAN: classification stance, `content_class` vocabulary, the `E-1` before-any-media ordering, the stub-signature posture | This side owns the transport (build, pin, replay verbatim) and never invents classification values — today's constants are recorded as an open question in `tech.md`, not a decision. CG-1 never re-decides transport discipline. `artstyle.go`'s latitude/negative blocks: the file is this domain's, the standard they implement is CG-1's. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **perception-and-knowledge** (WE-3) | nothing at generation time — that is the point | Mirror of its Art & Assets row: generation reads authoritative `*_state`, never `perception_record`; "fixing" generation to read perception breaks a documented decision. In return this domain never re-derives visibility: the existence gate stays perception's (`fn_actor_page` NULLs via `fn_entity_visible`, `schema.sql:825`, before any image field is built). |
| provides | **compendium-surfaces** (UX-1) and **play-surface** | `image_ref/1` fields on pages and `scene_current/4` (portrait, backdrop, `sprites`) | Asset id + path, never a presigned URL; `NULL` is ordinary and the placeholder is the contract (`D-8`). Consumers never persist or re-request URLs, never re-derive the entity→asset mapping, never perception-scope the two fields that are deliberately not (`tech.md` §The read path). |
| provides | **world creation UX** | `GET /worlds/art-styles` — key, label, blurb | Deliberately NOT the prompt prose: `look` is unexported, so what a key means to a model stays tunable server-side (`ADR-P023`). A client picks by key; a frontend constant restating a look is the drift `ADR-P023` exists to prevent. |
| provides | **the sprite vocabulary** | four emotion variants (`neutral·happy·angry·sad`), bust framing, 3:4 ratio | The backend owns what a "happy" bust means; the platform renders caller-defined cells verbatim (`imagehandler.go`, grep `pack cells verbatim`). The platform's PRD 08 ten-expression taxonomy is its own proposal, not this vocabulary. |

## The seams that do not exist

- **Signing.** `stub-unsigned-v1` is an honest stub; the canonicalization is an undesigned
  cross-repo contract (`tech.md` §Open questions 3). Do not invent crypto on either side.
- **Real classification.** The envelope carries PoC constants; the day shareable worlds arrive,
  the classification seam to CG-1 must be built, not improvised (`tech.md` §Open questions 1).
- **Sprite sheets.** The engine consumes single-variant generations; sheet-and-slice lives only as
  the platform's PRD 08. An engine agent asked for "more expressions" is opening this seam, and
  should say so.
- **Relationship-driven expression swapping.** The PRD lists relationship stance and trust/fear
  values among expression drivers, while the relationship *surface* is deliberately unbuilt
  (WE-3's product file, "no relationship surface"). Both statements stand; whether an expression
  swap counts as surfacing relationship state is unruled. Recorded, not resolved.
