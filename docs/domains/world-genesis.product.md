# world-genesis · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-10 · World genesis and world creation ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `world-genesis.tech.md` holds how it is built; `world-genesis.seams.md`
holds what crosses its boundary.

---

## Stage 2 — filling — is designed in full elsewhere

**`docs/design/2026-08-28-the-filling-stage.md`** (`SPEC-044`) is the governing design for how a world
gets its skeleton and meat once stage 1 has found its soul. It settles the product shape: lore is
knowledge and has holders; each layer is authored from the layers above it; public lore is what everyone
here knows and private lore is what makes a scene possible; people carry personality, a goal and what
they would sacrifice for it, and example phrases in their own voice; and circumstance and disposition are
authored separately so a person can have had the worst life and still be optimistic.


## What this domain is for

**One job: turning a brief into a playable world.** Interview, genesis, kickstart, arrival, refresh,
and the world directory. **This is the only path in the system that authors entities.**

Genesis is five steps: take an input · understand the intention · transcribe what was given · fill,
always governed by that understanding · emit the completed document
(`docs/design/2026-08-26-world-identity-and-the-understanding-pass.md`, "What this closes"). The
genesis document behaves as **compiler IR** (`digest/01_TOPIC_MAP.md` §WE-10): everything downstream
— play, perception, art — reads what genesis emitted, never the brief.

The product reason it is its own domain: **a filler that does not know what makes this world
different from the nearest familiar one fills from the familiar one** (the design §1). Living houses
become a haunted-house story. Every failure of world creation is some form of that sentence, and the
understanding pass — the design this domain is built around — exists to prevent it.

## Ubiquitous language

Use these words with these meanings. The design defines most of them; deviations are bugs.

| Term | Means, precisely |
|---|---|
| **Brief** | The author's free-text input (`docs/design/prd_world_creation.md:32` — one free-text brief). Everything genesis knows starts here. |
| **Lane** | Fast or Custom, chosen *after* the brief. Custom *only adds answers*; Fast infers everything (design §8.1: no interview, no confirmation screen — founder ruling 2026-08-27). |
| **Understanding pass** | Step 2 of genesis: infers the world's identity from the brief. Emits inferred items only — slots are minted because a world needed them, never asked as a form (design §3). |
| **Identity** | The rules a world makes about itself, each carrying its consequence (`therefore …`). **Invention rules, not description** — mood/tone/setting boxes are rejected outright (design §2, GA-2/GA-3). Emitted versioned, immutable for the duration of genesis (design §5). |
| **Condition** | What is simply true here. Statable by the author. |
| **Bargain** | What living with the condition costs. **Exactly one**; five bargains of equal weight produce mush (design §3.1). Usually derived, not stated. |
| **Face** | The one bargain as it meets a different life. The test for one-world-or-two: a second pressure derivable from the bargain is a face; one that is not means wrong altitude or two worlds stapled (design §3.1). |
| **Departure** | The nearest familiar thing this world resembles, and the specific way it is not that. Blocks the §1 failure by naming the neighbour the filler would default to (design §3.2). |
| **Exclusion** | The *never* half of identity, each with its reason — reasons generalise, lists do not. Marked **exist-kind** or **happen-kind**; happen-kind carries an enforcement marker (design §3.5, §9; `SPEC-036`). |
| **Register** | The size of a normal problem here — missing a funeral, or losing a war (design §3.6). |
| **Content demand** | A requirement, not a rating. A world's central pressure demands its content; "NSFW: allowed" governs nothing (design §3.7). |
| **Voice** | Imitable prose, never adjectives. Three sentences of actual narration (design §3.8). |
| **The twenty universal functions** | Ordinary life the pressure does not demand, phrased as human functions, never professions. One–two sentences each. The pass criterion is applied to THESE — could this answer exist in any other world? — there is no separate test (design §3.10–3.11). |
| **Rule kind** | Constraining · generative · prohibiting · voicing. Constraining rules are established before generative rules run (design §4). |
| **Origin** | Contingent (names a cause; play can change it) or axiomatic (bare fact; nothing to undo). The contingent/axiomatic mix is itself an identity output — **the mix is the game** (design §5). |
| **`stated` / `inferred`** | Provenance of every item. `stated` outranks `inferred` on every contradiction (design §6). |
| **Kickstart** | The character turn and the scenario/arrival turn, resumable. The LAST answer is the arrival transaction. |
| **Arrival** | **Chosen, not assigned** — after the cast exists, because candidates grounded in authored fiction are fiction (`docs/design/2026-08-20-kickstart-arrival-choice-design.md`, amends PRD AC-6). |
| **Refresh** | Mints a **new** world from a template and archives the source. Append-only; the old id stays a valid reference. |
| **`playable:false`** | A world created by `POST /worlds`: real, listed, not yet enterable — the honest state, not half-built (SPEC-028). |

**`entity`** is legal only as the engine supertype in internals; user-facing and PRD language is
Actor / Location / Artifact / Carrying / Timeline (repo `AGENTS.md` §Vocabulary, GA-2).

## What this domain is not

- **Not whether what genesis authors becomes true.** That is canon-recording; genesis'
  `origin='fast_path'` is the one documented `D-1` exception (see `seams.md`).
- **Not what a viewer may know about the authored world.** Perception's. Genesis authors the truth;
  it never decides visibility (`B-1`).
- **Not the art a built world gets.** Art-and-assets'. The *kick* is ours (`ADR-P021`); the
  reconciler, the styles, and the platform call are not.
- **Not where anything is or how far.** Space-and-journey derives geometry and duration; genesis
  authors places and coordinates.
- **Not the mechanics of publishing a schema.** Contracts-and-platform (SPEC-011, the pin rule).
  Genesis owns what the shapes *say*.
- **Not the creation UI.** `dream-weaver-visuals` renders our frames; it never decides a question or
  invents a field (`D-7`).
- **Not the world-model document schema itself.** That is WE-11 · The world-model contract — an
  unwritten neighbour package. Genesis fills the document; WE-11 owns what the document *is*.

## Product rules — decisions already made

Ids only; the law lives where the id resolves. Cite it, never restate it.

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `GA-2`, `GA-3` | Genre-agnostic system terms; no genre taxonomy anywhere in code, prompt or schema (`prd_world_creation.md:177` bans it by name). | Archetype poison: the world becomes a template of the nearest genre. |
| `B-4` | Player interiority: the system never authors the player character's inner state. | The arrival turn authoring what you feel overwrites the one reading that is yours. |
| `B-5`, `D-11` | In-world time is a logical tick + display label; **no numeric field anywhere in `world_genesis/1`** — colours, extents, distances are derived, because a colour taxonomy is a genre taxonomy wearing a hat (§WE-10). | A seat that emits a number smuggles genre back in. |
| `E-1` | Every seat call is driven, budgeted and configured per seat. | A hardcoded model name or SDK bypasses the cost and routing machinery. |
| `docs/design/2026-08-20-kickstart-arrival-choice-design.md` | Arrival is chosen, not assigned (amends PRD AC-6). | A teleport ending: the seat decides who you are. |
| `docs/design/2026-08-21-durable-worlds-design.md` | A built world is never lost (amends PRD AC-2). Written from two production deaths. | The cheap kickstart destroys the expensive world again. |
| `ADR-P021` | Art is automatic — genesis kicks a reconciler; never a manual trigger. | A world ships with no images, or a user is told to press a button that should not exist. |
| Design §8.1 (founder ruling 2026-08-27) | The Fast lane gets **no** pre-build step: no interview, no identity confirmation. The finished world is the confirmation; the mitigation is cheap regeneration. | *"A step that exists so the builder feels thorough is not a safeguard."* Re-adding one reopens a founder ruling. |
| Design §6.1 | A flat world is a legitimate outcome and is recorded as one. | A narrator puts a serial killer in the cosy village by scene three. |
| `ADR-P026` | This domain exists as a package; the understanding-pass design is live law; the testworld briefs stay in history. | Genesis decisions become uncitable again. |

## Relevance — how much of a thing exists

**Ruled 2026-08-31, `ADR-P027`.** The single most important thing to know before changing anything in
this domain.

A world of interest is far larger than a world anyone will ever look at, so entities are not all authored
to the same fullness. Every place, person, faction, concept and object carries a level, 1 to 4:

|level|earned by|what exists|
|---|---|---|
|1|being named|a name, a one-line look, and **one tag** — the characteristic thing a narrator can play with. This is COMPLETE, not a draft.|
|2|being referenced|enough to hold a scene|
|3|being interacted with|the full interior, and an image|
|4|being bound to the player|as 3, plus an asset nobody else may share|

**Two mechanics, and confusing them is the expensive mistake.** *Genesis assigns* — the scaffold tags
every name with the level it deserves, and the fill writes to that level. *Play promotes* — reaching
something raises it, and the promotion mints what the new level owes. Relevance **never falls**.

**New entities always enter at 1.** This is the law that makes it terminate; without it, promoting A
authors B in full, which authors C, forever.

**A relevance-1 person is a real person**, not a defective one: they exist, they are somewhere, and they
answer from their tag. Thin is a legal state and needs no placeholder.

**Why it is here and not in the tech doc:** it is the difference between a world with hundreds of named
people in it and a world with twelve. A relevance-1 entity costs about twenty output tokens; the same
entity written at 3 costs about 2,300.

## What is deliberately not built here

Each absence has a stated reason. Building one is reopening a decision, not filling a gap.

- **No world templates.** A starter scene is authored fiction, and shipping one as a template is
  exactly the archetype poison `GA-2`/`GA-3` forbid (area dossier trap, carried here). Do not add
  one because it would be convenient.
- **No Fast-lane interview or confirmation screen.** Design §8.1, founder-ruled, with the cost
  stated so it is chosen rather than discovered: a bargain inferred at the wrong altitude ships a
  coherently, confidently wrong world, and the author's first sight of it is the finished world.
- **No per-world generated code, ever.** Checks are instances of shapes already coded, with the
  world's values filled in — the discipline `core/api/tier1.go` states (design §9).
- **No identity-description boxes** (mood/tone/setting/magic-level). Rejected with three reasons at
  design §2; re-adding them is re-deciding it.
- **No enforcement path for a world's own rules — yet.** `SPEC-036`, deferred deliberately so
  genesis can proceed. Exist-kind rules are model-enforced at creation moments; happen-kind rules
  map to acts only when the engine recognises the act; a rule naming an unknown act is **narrator
  guidance, not enforcement**, and genesis must mark it as such (design §9).
