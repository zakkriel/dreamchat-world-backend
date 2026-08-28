# naming-wall · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-4 · The naming wall ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `naming-wall.tech.md` holds how it is built; `naming-wall.seams.md` holds
what crosses its boundary.

---

## What this domain is for

**One job: what a viewer's surfaces may call each thing, given what that viewer has earned.**

A name is knowledge, not a label. Canon knows everything as its canonical name; a viewer knows a
thing by the name someone gave them, or by a descriptor ("the muscle by the bar") until then. This
domain decides, per viewer, which canonical names are **unearned**, substitutes the viewer's own
label everywhere prose could carry one, and records the moment a name becomes earned — hearing it
spoken teaches it (`SPEC-033`).

The product reason it is its own domain: perception (WE-3) decides *whether you noticed*; the wall
decides *what you would call it*. The founder caught the difference failing in play — narration
reading "Jonas planted between her and the room" to a player who had only ever perceived "the muscle
by the bar" (`core/api/namingwall.go:19-23`). A mechanism, not a discipline: *"no prompt wording
would have saved it"* (migration `20260809090005`).

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Canonical name** | The registry's viewer-agnostic name (`entity_registry.canonical_name`). Reaching it in a player-facing string means a viewer read a name nobody gave them. |
| **Label / display name** | What ONE viewer calls a thing: `fn_display_name` — perceived name, else descriptor, else canonical as last resort. |
| **Earned / unearned** | A name is earned when a knowledge path delivered it; unearned otherwise. Defined once, in SQL (`fn_unearned_names`) — see `tech.md` §The definition. |
| **The seam** | `fn_viewer_text` applied at every perception write: content is rendered per holder before it is stored. The source fix. |
| **The belt** | `NamingWall` (`core/api/namingwall.go`): the API-boundary check for what a seat invents on its own. Checks the seam's own SQL definition, never a second predicate. |
| **Teaching** | `payload.spoken` → `fn_names_in_text` → `name_knowledge`. Only words actually said teach; *"A nod is not an introduction"* (migration `20260814170000:42`). |
| **Token guarding** | For unearned **actor** names only, each distinctive word is also guarded (the Ironmoor fix, migration `20260821120000`). |

## What this domain is not

- **Not who perceives.** WE-3 decides who gets a perception row; the wall renders its text. *"The
  wall never decides who perceives"* (`perception-and-knowledge.seams.md`, provides→WE-4).
- **Not what an actor may believe or say.** Bluffing is legal: the candidate whitelist governs what
  an actor may *bind* (name as a target), never what they may *speak of*. An unknown name as a topic
  of speech is never blocked — it resolves truth-side through the listener's knowledge
  (`docs/law/rulings/RULINGS-2026-07-23-grilling-session.md` §3). The whitelist itself is decompose's
  (WE-7); the wall only supplies its labels.
- **Not presentation.** The frontend renders labels it is handed; it never resolves names.
- **Not the referee's blindfold.** Resolve is truth-side and licensed to read canonical everything;
  the wall protects the character-mind seats and player surfaces (`core/api/wall_test.go:22-27`).

## Product rules — decisions already made

Ids only; the law lives where the id resolves.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `B-1` | Surfaces render from the holder's perception; the canonical name is hidden truth like any other. | A name in a payload the viewer never earned is a leak, however the UI masks it. |
| `B-2` | Names enter a viewer's knowledge only through valid in-world paths. | Passing a raw name through "just for speech" grants knowledge without recording it — the same breach with a nicer story (migration `20260809090005:44`). |
| `SPEC-033` | Hearing teaches, if present; teaching reads `payload.spoken` only; first hearing wins. | Teaching from the referee's account taught four names from one ordinary sentence — a nod taught "Kade". |
| `GA-2` | Wall vocabulary is genre-agnostic: name, label, descriptor — no genre terms in the mechanism. | A genre-flavored term in core vocabulary fails the three-genre test. |

**The strictness asymmetry is law**, stated in migration `20260814170000`: rewriting
(`fn_viewer_text`) is case-INSENSITIVE — over-catching only hides a name already unearned; teaching
(`fn_names_in_text`) is case-SENSITIVE — over-teaching is a wall breach, under-teaching costs one
beat of "the muscle by the bar".

## What is deliberately not built here

- **No wall for groups.** `fn_display_name` has no group branch, so a collective falls through to
  its canonical name — *"Its name is speakable at tick 0"* — and inherits **no** wall; *"if you ever
  want one you must author it"* (`digest/S07a` §8, verified against `core/db/schema.sql:1503-1514`).
  Group `legibility` was ruled illegal until a group-side gate exists. Round-B papers repeatedly
  assumed groups were covered; the claim was refuted and conceded — do not re-assume it.
- **No aliases.** `name_knowledge`'s PK is one name per (holder, subject); a later different name
  *"is the alias question, which nothing in the thin slice can answer honestly — it stays out rather
  than being guessed at"* (migration `20260809090007:43-45`).
- **No speech exemption.** Deliberately refused; see `B-2` row above — one home, migration
  `20260809090005:44`.
- **No case preservation in Scrub.** The label is world data, written as stored; *"the wall is worth
  a lowercase article"* (`core/api/namingwall.go:105-107`).
- **No token guarding for places and objects.** Actors only: their names are proper nouns; place and
  object names are English, and guarding their words *"would eat the narrator's vocabulary one common
  noun at a time"* (migration `20260821120000:35`).
