# content-governance · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** CG-1 · Private, public, and the latitude ·
**Parent bounded context:** Content Governance

This file holds how the domain is built — what exists, the envelope path, validation, traps.
`content-governance.product.md` holds what it means; `content-governance.seams.md` holds what crosses
its boundary.

Line numbers are as of 2026-08-27; re-locate by grep before relying on one.

---

## What exists — and it is less than the law describes

The domain's paper surface is large (`ADR-P016`'s data model, two JSON schemas, seven planned
endpoints). Its built surface is exactly three things:

1. **The governance envelope** — `govEnvelope` + `newGovEnvelope`, `core/api/imageclient.go:185-215`
   (grep `newGovEnvelope`). Hosted inside the art-and-image-seam's files; this domain owns the
   fields' meaning, not the transport (`seams.md`).
2. **The latitude block** — five paragraphs in all ten `core/api/prompts/*.txt`, plus the image-medium
   analogue `artStyleLatitude` / `artStyleNegative` (`core/api/artstyle.go:35,41`), gated by
   `core/api/promptlatitude_test.go`.
3. **The law and its contract shapes** — `docs/law/adr/ADR-P016_*`, `ADR-P022_*`,
   `README_governance_pack.md`, and the two schema JSONs beside them
   (`world_content_governance.schema.json`, `media_generation_eligibility.schema.json`) — the shapes
   of the **unbuilt** eligibility service, not of anything running.

**There is no classifier, no eligibility service, no `world_content_profile`.** Verified 2026-08-27:
`grep -rn "eligibility\|content_profile\|publish-check" core/api --include='*.go'` finds only the
envelope comment in `imageclient.go`; `grep "world_visibility\|content_profile\|review_status"
core/db/schema.sql` finds nothing.

## The classification path, as built

`E-1` says classification happens before any media request. What implements it today:
`newGovEnvelope(issuedAt, contentClass)` stamps all seven fields —
`schema_version: "1.0"`, `classification_id: "cls_poc_default"`, `visibility: "private"`, the
per-call `content_class`, `authorized_by: "svc_world_backend"`, the pinned `issued_at`, and
`signature: "stub-unsigned-v1"` (`imageclient.go:205-215`).

Three call sites, each stamping the envelope **before** the platform call (grep `newGovEnvelope(`
in `core/api/imagehandler.go`): portrait anchor bootstrap (`:404`, `character_portrait`), sprite
pack (`:444`, `expression`), scene/cover (`:756`, `place_scene`).

So **E-1's ordering holds structurally — no media request leaves without an envelope — but the
classification content is a PoC constant.** Every request is classified private with one fixed
`classification_id`. Honest today (the PoC is private-single-user, `product.md` §not built), and the
first thing that becomes a lie the day sharing exists.

**Envelope discipline** (owned by the art seam, stated here because governance fields force it):
`issued_at` is passed in, never taken from the clock inside `newGovEnvelope` — the platform's
idempotency key hashes the whole body, so a rebuilt envelope under the same key is a 409, not a retry
(`imageclient.go:197-201`; proven by
`TestImageClient_PinnedEnvelopeSurvivesRetryButAFreshOneDoesNot`).

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `E-1` | Envelope before every platform call; no unclassified media request exists. | A new platform call site without `newGovEnvelope` ships ungoverned media generation. |
| `ADR-P022` | The block is enforced by test, byte-identical, files-not-vars. | *"A prompt file added tomorrow cannot ship without it"* — adding a seat means carrying the block. |
| `image:ADR-I002` (platform repo, `dreamchat-Image-Platform/docs/adr/ADR-I002-governance-and-cost-routing.md`) | The receiving gate verifies presence/freshness/issuer/signature and stores `content_class` opaque — never parses it. | Expecting the platform to catch a wrong classification: it cannot, by design. |
| `D-3` | Policy stays core-side. | See `product.md` table. |

### What you may not decide alone

1. **Setting `visibility` to anything but `"private"`.** That claims a classification the core cannot
   yet produce — it needs the eligibility service, which needs a ruling to start (`ADR-P016` phases).
2. **Changing the `content_class` vocabulary.** The strings land opaque in the platform's stored
   jobs and audit events; a rename orphans the audit history on a service we do not operate.
3. **Adding, removing, or rewording a latitude paragraph.** Founder ruling (`ADR-P022` "Owner of
   decision"); the prohibitive-vs-affirmative shape was tried both ways.
4. **Designing the envelope signature.** Cross-system contract; the platform explicitly waits for
   core to design it (`TODO(core-signing)`). Inventing it here unilaterally breaks their startup
   refusal logic (enforce + stub is refused in live).

## Validation for this domain

- `go test -run 'Latitude' -count=1 ./core/api/` — the four latitude gates
  (`TestEverySeatPromptCarriesTheLatitude`, `…ByteIdenticalAcrossSeats`, `…EmbeddedPrompt…`,
  `TestImageStyleCarriesTheSameLatitude`).
- `go test -run 'GovEnvelope|PinnedEnvelope' -count=1 ./core/api/` — seven fields present, pinned
  envelope survives retry.
- `-count=1` always — the `core/api` suite is seed-dependent (`perception-and-knowledge.tech.md`).

**What counts as evidence here: this domain fails silently twice over.** A seat that flinches
produces valid prose; a blocked envelope under the platform's `log_only` default still returns 202
and the job runs (`imageclient.go:66-68`). Neither errors. Proof of governance acceptance is the
platform's `audit_events` showing `media.eligibility_verified` — never the HTTP status.

**What counts as ceremony here:** asserting a 202/`Requested++` and calling the envelope validated.
That test passes with `authorized_by` misspelled, `issued_at` dropped, or the whole envelope zeroed —
`log_only` proceeds regardless. The latitude gates are the opposite: they read the files, so they
fail with the behaviour deleted.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **202 does not mean the envelope was accepted.** Under `log_only` an unknown issuer or missing field audits `eligibility_blocked` and proceeds; under `enforce` the same request is a 403. Misconfiguration is invisible until enforcement flips. | `imageclient.go:66-68`; the platform quickstart's own warning (`digest/S12b…md` Topic 10). |
| **`ADR-P016`'s H1 reads `# ADR-016`.** Copy the heading and the citation gate resolves it to *engine* ADR-016 (corrections), not this file — a namespace collision, not a reject. Cite `ADR-P016` (the filename). | `README_governance_pack.md` "Known, not fixed here"; `ci/check_citations.sh` on bare `ADR-016`. |
| **The governance PRD is deleted but still cited.** Commit `88486c1` (2026-08-27, the docs consolidation) deleted `docs/10_prds/prd_private_public_content_governance.md` with no rename target, while `README_governance_pack.md` (its Files table), `docs/design/prd_world_creation.md:37,73` (section-level cites) and `E-1`'s source column ("ADR-P016 + governance PRD") still point at it. Recoverable: `git show 88486c1^:docs/10_prds/prd_private_public_content_governance.md`. Recorded, not resolved — whether it was consolidation debris or an intentional cut is a ruling. | `git show 88486c1 --name-status \| grep private_public` → `D`. |
| **`governance_test.go` is not this domain.** It gates `Governed-by:` doc headers. | `core/api/governance_test.go` (read its header). |
| **Two comments claim the `content_class` vocabulary differently.** `imageclient.go:203`: it carries *"our own vocabulary rather than one negotiated with them"*. `imagehandler.go:753-755`: *"Their ContentClass enum is character_portrait\|place_scene\|artifact\|expression\|angle_variant"* — which are the platform's **AssetType** values (`apigen.gen.go:47`), while the platform's governance `content_class` is a plain opaque string (`apigen.gen.go:953-954`). Both sides recorded; which comment is right is not decided here. | The three cited lines; friction log 2026-08-27. |
| **`ADR-P022`'s evidence line says "all nine" prompt files; there are ten.** All ten carry the block (the gate proves it); the count in the ADR is stale. Do not "fix" a prompt count from the ADR. | `git ls-files 'core/api/prompts/*.txt' \| wc -l` → 10; `grep -l 'UNCENSORED BY DESIGN' core/api/prompts/*.txt \| wc -l` → 10. |

## Open questions

1. **Where does the governance PRD live now?** Deleted by `88486c1`, cited in three live places
   (trap table). Restore, re-home, or re-point the citations — founder's call.
2. **Who designs `core-signing`, and what flips `enforce`?** The platform refuses enforce+stub in
   live; the go/no-go it prescribes is zero `eligibility_blocked` audit rows on the traffic about to
   be enforced. Nothing in this repo owns that milestone.
3. **What triggers building the eligibility service?** The PRD staged it at phase 2 of 5; phase 1's
   data model is also unbuilt. Is it gated on sharing, on marketplace, or parked indefinitely?
4. **Is the `content_class` vocabulary ours or theirs?** (Trap table, both sides.) Needs one owner
   before any new call site picks a value like the scene path's "honest nearest" `place_scene`
   compromise (`imagehandler.go:753`).
