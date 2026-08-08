# Running the full integrated stack from zero

> Three repositories, four processes. Every command below was executed in order on
> 2026-08-08 and the outputs are copied from that run — where something is slow, or
> bites, it says so.

The stack:

| Piece | Repo | Port |
|---|---|---|
| Postgres (world) | `dreamchat-world-backend` | **5432** |
| World backend API | `dreamchat-world-backend` | **8080** |
| Image Platform (+ its own Postgres, Redis, MinIO) | `dreamchat-Image-Platform` | **8081** (its Postgres on **5433**) |
| Frontend | `dreamchat-frontend` | **5173** |

The two Postgres instances deliberately differ: the image platform remaps its own
to 5433 so both stacks run side by side. **The world backend owns 5432.**

Only the first two are required. The image platform is optional — with no image
config the world runs exactly as before and every `image` field is `null`. The
frontend is optional for backend work; `curl` reaches everything.

---

## 0. Prerequisites

Docker (the only sanctioned runtime) and Go. `make doctor` checks the first:

```bash
cd dreamchat-world-backend
make doctor          # → docker OK: Docker version 29.5.3, ...
```

---

## 1. Database from zero

```bash
make reset           # ~4s: down -v, up, migrate, seed
```

`reset` is destructive: it drops the volume and rebuilds. It seeds two worlds —
the **Mara 0A fixture** and **The Drowned Lantern**, the one anybody plays.

> **The database is shared and stateful, and other people may be driving it.**
> If a frontend or another agent is working against this instance, coordinate
> before running `reset` — it wipes their world mid-session. `make migrate`
> applies new migrations *without* destroying anything and is almost always the
> one you want during a session.

---

## 2. The battery, in the order that works

Order matters and is not cosmetic:

```bash
make reset && make test \
  && cd core/api && DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' go test -count=1 ./... \
  && go vet ./... && cd ../.. && make schema-check && make reset && make schema-contract
```

Verified output: **66 files, 410 tests, `Result: PASS`**, then `ok dreamchat/core/api`,
then `SPEC-011 schema contract … coverage 11/17 published schemas`.

Why the order:

- **pgTAP needs a fresh seed.** `make test` after a played-in session fails on
  state, not on code — several tests say so in their own failure messages.
- **Go tests write into the fixture world by design**, so they run *after* pgTAP.
- **`make schema-check` does not reseed.** It leaves the database migrated but
  empty, which breaks the next Go run — hence the second `make reset` before
  `schema-contract`.
- `DATABASE_URL` is **not** exported in the shell, and `-count=1` is required
  (a cached `ok` proves nothing).

---

## 3. World backend

```bash
cd core/api
DATABASE_URL='postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable' \
DREAMCHAT_MODE=debug \
DREAMCHAT_BRIDGE=fake \
DREAMCHAT_CORS_ORIGINS=http://localhost:5173 \
go run .
# → dreamchat world backend (read-only compendium API) on :8080 (debug=true)
```

| Variable | Meaning |
|---|---|
| `DATABASE_URL` | required; no default |
| `DREAMCHAT_MODE=debug` | honours `?viewer=<uuid>` to play as someone specific. Not needed for ordinary play — the world states its own player. |
| `DREAMCHAT_BRIDGE=fake` | deterministic seats, no API keys. Omit and every seat wants `ANTHROPIC_API_KEY`. |
| `DREAMCHAT_CORS_ORIGINS` | comma-separated exact origins. **Unset ⇒ CORS off** and a browser on another origin cannot call the API at all. `*` refuses to boot. |
| `DREAMCHAT_IMAGE_BASE_URL` / `DREAMCHAT_IMAGE_API_TOKEN` | optional; see §5 |

Smoke it:

```bash
curl -s localhost:8080/worlds | jq -c '[.worlds[].display_name]'
# ["Mara 0A Fixture","The Drowned Lantern"]

W=22222222-2222-2222-2222-222222222222
curl -s "localhost:8080/worlds/$W/scene/current" | jq -c '{schema_version, place:.place.label}'
# {"schema_version":"scene_current/2","place":"The Drowned Lantern"}
```

No `?viewer=` is needed: each world names its own player (Kade here).

---

## 4. Frontend

```bash
cd ../dreamchat-frontend
npm install
npm run dev          # → http://localhost:5173
```

It reads the API base from its own config (SPEC-020) and must point at
`http://localhost:8080`. Local dev is same-origin through the Vite proxy, but
direct calls must work too — which is what `DREAMCHAT_CORS_ORIGINS` above is for.

---

## 5. Image Platform (optional)

Its own compose, its own Postgres on 5433, API on 8081.

```bash
cd ../dreamchat-Image-Platform
# see that repo's docs/api/integration-quickstart.md — it is the contract of record
docker compose up -d
```

Two settings decide whether anything works:

- **`ALLOW_SYNTHETIC_PROVIDERS=true`** (or a real `FAL_KEY`). It defaults to
  **false in every environment**, and without it every generation returns
  `422 route_capability_mismatch` — the first thing to check when nothing renders.
- **`GOVERNANCE_AUTHORIZED_ISSUERS` must contain `svc_world_backend`.** Under the
  default `log_only` an unknown issuer still returns `202` while recording
  `eligibility_blocked`, so it works right up until enforcement is switched on and
  then fails everywhere at once.

Point the world backend at it and restart:

```bash
DREAMCHAT_IMAGE_BASE_URL=http://localhost:8081 \
DREAMCHAT_IMAGE_API_TOKEN=dci_dev_<prefix>_<secret> \
# ...plus the §3 variables
go run .
```

The token needs `styles:write`, `images:write`, `jobs:read`, `images:read`. It is
read once at startup, never written to the database and never logged.

Generate and fetch:

```bash
curl -s -X POST "localhost:8080/worlds/$W/images/portraits" | jq -c
# {"schema_version":"image_portraits/1","requested":5,"completed":5,"failed":0,"skipped":0}

P=$(curl -s "localhost:8080/worlds/$W/scene/current" | jq -r '.participants[0].image.path')
curl -sL -o /tmp/p.png "localhost:8080$P?tier=final" && file -b /tmp/p.png
# PNG image data, 1024 x 1024, 8-bit/color RGB, non-interlaced
```

Generation is **explicit** — that endpoint, never a read. Portraits survive
`make migrate` but not `make reset`, which drops `image_slot` with everything
else; re-run the trigger after a reseed.

---

## 6. End-to-end smoke (all four up)

```bash
W=22222222-2222-2222-2222-222222222222
B=http://localhost:8080

curl -s $B/worlds | jq -c '[.worlds[]|{display_name,playable}]'
curl -s -o /dev/null -w '%{http_code}\n' -X OPTIONS \
  -H 'Origin: http://localhost:5173' -H 'Access-Control-Request-Method: POST' \
  $B/worlds/$W/beats                                    # 204
curl -s "$B/worlds/$W/scene/current" | jq -c '.schema_version'   # "scene_current/2"
curl -s -N -X POST "$B/worlds/$W/beats" \
  -H 'Content-Type: application/json' \
  -d '{"text":"tell Mara about the sealed note"}' | grep -c '^data:'   # 6+ frames
```

The beat body field is **`text`**, not `input`. A beat returns a stream of
`beat_frame/2` frames: `interpretation → narration* → scene → journey → result → trace`
(trace only in debug).

Walk out and travel:

```bash
curl -s -N -X POST "$B/worlds/$W/beats" -H 'Content-Type: application/json' \
  -d '{"text":"go to Dock Street"}' >/dev/null
curl -s -N -X POST "$B/worlds/$W/beats" -H 'Content-Type: application/json' \
  -d "{\"text\":\"go to the Harbormaster's Office\"}" | \
  sed 's/^data: //' | jq -c 'select(.kind=="journey")|.journey'
# {"active":true,"kind":"travel","goal_label":"Harbormaster's Office","legs_total":5,...}

curl -s -N -X POST "$B/worlds/$W/beats/continue" >/dev/null   # ×4 → status "arrived"
```

---

## 7. When it does not work

| Symptom | Cause |
|---|---|
| `make test` fails on a played-in database | pgTAP needs a fresh seed. `make reset && make test`. |
| Go tests fail after a session of play | Same. The battery order in §2 exists for this. |
| Browser cannot reach the API at all | `DREAMCHAT_CORS_ORIGINS` unset — the boot log says so. |
| Every beat returns "the world could not resolve that beat" | Read the server log; the player-facing message is deliberately generic. |
| `422 route_capability_mismatch` | Image platform has no identity-capable provider — §5. |
| `409 idempotency_conflict` | A generation body was rebuilt under an existing key. The envelope's `issued_at` must be pinned, not regenerated. |
| Portraits vanished | `make reset` dropped `image_slot`. Re-run the trigger. |
| A beat 500s right after a reseed | The server caches nothing, but an open session's world ids are gone. Reload the frontend. |
