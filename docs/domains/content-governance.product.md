# content-governance · product

**Repo:** `dreamchat-world-backend` · **Cluster:** CG-1 · Private, public, and the latitude ·
**Parent bounded context:** Content Governance

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `content-governance.tech.md` holds how it is built;
`content-governance.seams.md` holds what crosses its boundary.

---

## What this domain is for

**One job: decide which governance regime a piece of content sits in, and whether media may be
generated from it — before anything leaves the core.**

The product ships mature adult fiction on purpose. `ADR-P016`'s Context says the product *"should not
be sanitized into a generic fantasy toy"*, and its policy principle is the domain's charter, quoted
because the wording is load-bearing: **"The platform should not silently moralize or sanitize private
world content. It should classify content, apply explicit eligibility rules, and route or block only
according to defined legal, platform, commercial, visibility, and provider constraints."**
(`docs/law/adr/ADR-P016_private_vs_public_world_governance.md:72`). This domain is where "classify,
never moralize" is decided; every other component either receives its verdicts (the Image Platform)
or carries its standing instruction (the seats' latitude block).

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Private** | Accessible exclusively by its creator/user. Nothing else. `ADR-P016`: *"Private = only the creator/user can access it."* |
| **Public-governed** | Any content another user can access **in any way** — invite, hidden link, collaboration, direct share included. The subtypes modulate strictness, never the regime (`ADR-P016`). |
| **The discriminator** | Who can access the content — not what it is, not who paid. *"Charging for it does not move it… the money was never the test"* (`ADR-P022` §"Where the latitude stops"). |
| **Hard-prohibited class** | Content blocked regardless of visibility, even in private mode (`ADR-P016`, "Hard Prohibited" section). The final list *"must be validated with legal counsel for launch markets."* |
| **Classification** | The core's verdict on content, produced **before** any media generation request (`E-1`). Today a PoC constant — see `tech.md`. |
| **Governance envelope** | The seven-field block every media request carries to the Image Platform, proving classification happened (`tech.md` §The envelope). |
| **Latitude block** | Five byte-identical paragraphs in every seat prompt, telling the seat what standard it is held to (`ADR-P022`). It *"creates no latitude"* — it transmits what `ADR-P016` already granted. |

## What this domain is not

- **Not the Image Platform's policy.** `D-3`: the platform never owns world truth; it *"receives only
  classified/authorized generation requests."* Its gate verifies and stores, never interprets
  (`image:ADR-I002`, in `dreamchat-Image-Platform`). If you find yourself teaching the platform what
  content means, you are on the wrong side of the seam.
- **Not the seats themselves.** Seat prompts belong to their owning domains (play-loop, world-genesis).
  This domain owns the one block they all carry, byte-identical, and the gate that enforces it.
- **Not `governance_test.go`.** Despite the name, that file gates `Governed-by:` doc headers —
  documentation plumbing, not content governance.
- **Not provider moderation.** What bfl or fal refuse is *"outside our control and the only remaining
  limit"* (`ADR-P022`). See `seams.md` for what neither side may do about a provider rejection.

## Product rules — decisions already made

Ids only; the law lives where the id resolves.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `E-1` | Classification (private/shareable/public/monetizable/media-eligible) happens in the core **before** any media generation request. | A media call without a prior classification verdict is a different architecture, and it wastes provider spend on undistributable assets (`ADR-P016`). |
| `E-2` | Mature content can exist in private worlds within legal/safety bounds; public distribution applies stricter eligibility. | Sanitizing private play enforces rules that only bind public distribution. |
| `ADR-P016` | The two regimes, the access discriminator, the hard-prohibited floor, the nine acceptance criteria. | Equating "private" with "not listed" — an unlisted link shared with one friend is fully public-governed. |
| `ADR-P022` | The latitude block: five paragraphs, affirmative not prohibitive, byte-identical, in every seat. | A prohibitive-only version was tried and failed — the seat *"flinched, and nothing prohibitive forbids flinching."* Paraphrasing per seat makes two seats disagree mid-scene. |
| `D-3` | The Image Platform is a separate service and never owns world truth. | Policy logic drifting into the platform makes content decisions nobody in the core made. |

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No eligibility service, no classifier, no `world_content_profile`.** The data model and seven
  endpoints the governance PRD specified do not exist anywhere in the schema or `core/api`
  (verified — `tech.md` §What exists). The PoC is private-single-user, which is *"precisely the
  regime that needs no sanitization"* (`docs/design/prd_world_creation.md:37`); classification is a
  constant until sharing exists.
- **No sharing, publishing, or moderation UI.** Explicit PRD non-goal and world-creation non-scope:
  *"Everything else in that PRD fires when sharing does, and sharing is not in this feature"*
  (`prd_world_creation.md:37`).
- **No marketplace or payment strategy.** `ADR-P016` Non-goals — and by the access discriminator,
  payment could not move content between regimes anyway (`ADR-P022`).
- **No real envelope signature.** `StubSignatureVerifier` passes everything (`TODO(core-signing)`).
  Deliberate on both sides: the canonicalization is a cross-system contract the platform *"correctly
  refused to invent"* (`core/api/imageclient.go:71-75`). See `seams.md` §seams that do not exist.
- **No final hard-prohibited list.** `ADR-P016` defers it to legal counsel per launch market. The
  named examples stand; the list is not closable from inside this repo.
