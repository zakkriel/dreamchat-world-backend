# world-model-contract · product

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-11 · The world-model contract and depth ·
**Parent bounded context:** World Engine

This file holds what the domain *means* — its job, its language, its product rules, what is
deliberately not built. `world-model-contract.tech.md` holds how it is built and where its corpus
lives; `world-model-contract.seams.md` holds what crosses its boundary.

---

## What this domain is for

**One job: the contract by which any world is described to the system that builds and runs it.**

The contract has two halves. The **author's half** says what a valid document must contain
(obligations) and what makes one invalid, rejected whole with the reason named (refusals). The
**reader's half** — *"the half that is usually forgotten"* — says what a builder **must derive** from
each authored value. The design's own bar: *"Two builders honouring these produce the same world.
Without them, 'same document, different world' is as fatal as 'same brief, different document.'"*

The split is the point: *"The document is the interface. Genesis produces it; Creation consumes it;
neither knows the other."* This domain owns what the document IS. WE-10 owns producing and committing
one.

## Ubiquitous language

| Term | Means, precisely |
|---|---|
| **Contract version** | One generation of the schema, `world_model/1` → `v7`. Each version's death is a constraint; v5+ are deltas over v3∪v4, never rewrites (`tech.md` §The chain). Cite counts with a version attached — every bare count in the corpus has rotted at least once. |
| **Facet** | A composable capability of an entity (`extent`, `matter`, `agency`, …). Facets replace kinds: v1 encoded kind as *which array a thing lived in* — "a closed ontology in the clothes of an open one" — and died on a living house that is place *and* agent. Frozen at eleven; a twelfth only by deleting one. |
| **Class vs number** | The author picks a class (`slow`, `vast`); the engine owns the number. Rung membership, order, polarity and dimension are grammar (code); the magnitudes are per-world vocabulary. |
| **`excluded[]`** | Negative canon: what does not exist or cannot happen here, binding on every authoring seat for the life of the world. **An empty one is not neutral** — a tier-1 encoding that omits "no cure exists" *permits* a cure: "two documents encoding the same world differ on whether the central tragedy is a tragedy." Hence the obligation: present, possibly empty, and *explicitly* so. |
| **Provenance** | Every element carries `"stated"` or `{"inferred_from": […]}`. Stated outranks inferred; an inference chain bottoming out in nothing stated is a refusal — genre bleed made mechanically detectable. |
| **Sufficiency** | The second bar, distinct from validity: a document can pass every refusal and still leave the player unable to leave the first room. |
| **Trial** | One seat encoding one test world against one candidate version. **Trials are reasoning, never state** — process residue from a dead directory (`tech.md` §Where the corpus lives). Nothing in a trial or a testworld fixture is a product decision. |

## What this domain is not

- **Not the genesis pipeline.** Elicitation, interview, kickstart, commit, the two lanes — WE-10 ·
  world-genesis (`ADR-P026`; its package exists).
- **Not the engine capabilities the reader's half obligates.** Time, space, perception, seats own
  their mechanisms; this domain states what a builder owes and measures the gap (`tech.md` §The
  audit).
- **Not the narrator's presentation** of the world statement — only the statement's source and shape.

## Product rules — decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `GA-2` / `GA-3` | System terms genre-agnostic; no fixed genre taxonomies. The contract's own P5 — "the system must never learn what a world is 'usually' like" — is this rule made structural; every example in the design is illustrative ONLY and may never be hardcoded. | A template library, a genre default, or an identifier traceable to one example world. |
| `D-9` | Document changes need empirical evidence once code runs. This domain's whole method: every version died on a *named test with a named failure mode*, never on review taste. | A v8 written "for its own sake" — the 2026-08-25 handover forbade exactly that. |
| `D-6` | The copy is the one that goes stale. | Restating the contract's rules here instead of citing version + section. |
| `ADR-P026` | World genesis is its own domain. | This package growing pipeline sections that are WE-10's. |

## What is deliberately not built here

- **No twelfth facet, ever, by addition.** "If a world needs a new facet that deletes nothing, this
  approach has failed and we say so rather than widening" — the declared falsification condition of
  the whole design, not a backlog item.
- **No flattened single-file contract.** The effective statement is the union v3∪v4∪v5∪v6∪v7, on
  purpose: superseded text stays in place *"because the wrong text is the evidence"* (v6, on why
  editing v2's table in place would destroy the record of a broken assumption).
- **No class ladders written by averaging the trial builders' anchors.** Closed decision, founder-
  ruled: "three readers disagreeing 10× on a rung is a requirement to decide something, not a range
  to split" (eight-increments plan §Closed decisions, git-history — `tech.md`).
- **No growth-during-play specification.** What may be invented after genesis and what is frozen is
  out of scope for all nine increments, recorded so it is not silently absorbed. An agent specifying
  it is deciding something new.
- **No relationship surface, no NPC idle life** downstream of this contract — those absences are
  perception's and cognition's (`B-11` territory), pointed at here only so nobody "fixes" them from
  the contract side.
