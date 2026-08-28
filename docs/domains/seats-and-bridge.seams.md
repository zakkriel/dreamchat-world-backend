# seats-and-bridge · seams

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-13 · Seats, the LLM bridge, and cost ·
**Parent bounded context:** World Engine

A seam belongs to two domains: one side owns a fact, the other consumes it and must not re-derive
or re-decide it. `seats-and-bridge.product.md` holds what the domain means;
`seats-and-bridge.tech.md` holds how it is built.

---

## What this domain consumes

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| consumes | **Operations / the release** | the environment (`DREAMCHAT_SEAT_*`, `DREAMCHAT_PROVIDER_*`) | Seat behaviour is environment-resolved by design; `ADR-P024` makes setting it part of the release. This domain fails closed at boot when it is absent — it never invents a default. |
| consumes | **Each seat's consumer domain** | the assembled prompt and the schema | The consumer owns prompt content and the published schema; the bridge transports both verbatim. Provider shaping (tool wrapping, root naming, fencing) is confined to the driver (`D-13`) and must never leak upward. |

## What this domain provides

| Direction | Domain | What crosses | The expectation |
|---|---|---|---|
| provides | **Play loop** (WE-7) | the `decompose`, `narrate`, `resolve` drivers (`beatsstream.go:321-363,484`) | A bound driver meets its seat's floor — the consumer never re-checks capability, and never constructs a driver itself. Acceptance of *content* stays with the consumer's belts; the bridge never validates output. Play-loop must not re-derive routing, temperature, or budget: those are seat properties resolved here. |
| provides | **NPC cognition** (WE-8) | the `cognition_batch` / `cognition_isolated` drivers, via the Orchestrator built in `beatsstream.go:322-323` | Same contract as play-loop. One call per action is cognition's law, not the bridge's — the bridge counts and prices calls, never rations them. |
| provides | **Living world** (WE-12) | the `world_actor` and `place_author` drivers | Same contract. The place_author schema-leash keeping geometry out of the model's hands is the seat definition's job (`bridge.go:114-118`), not the consumer's. |
| provides | **World genesis** (WE-10) | the three creation seats (`world_genesis`, `world_interview`, `world_kickstart`), consumed by `worldgenesishandler.go` | Same contract, plus cost: the genesis spend ceiling env var lives in the consumer (`worldgenesishandler.go:67`) but mirrors this domain's beat variable exactly, so an operator learns one convention. WE-10's package (written) owns the seats' prompts and schemas. |
| provides | **Content governance** (CG-1) | the latitude block, byte-identical in every seat prompt | The block *tells* seats the standard `E-2`/`ADR-P016` already granted; it grants nothing (`ADR-P022`). Governance owns the private/public discriminator; this domain owns delivery and byte-identity. |
| provides | **The naming wall** (WE-4) | raw seat output, unwalled | The wall is applied downstream (the `NamingWall` belt at beat emit, per perception-and-knowledge.seams.md). The bridge must never substitute a name; the wall must never assume the bridge did. |
| provides | **Art & image seam** | the same latitude, in the image medium's vocabulary | `promptlatitude_test.go` asserts `artstyle.go`'s `artStyleLatitude`/`artStyleNegative` carry the analogue — censorship in a picture is a *composition*, so the negative prompt names compositions. The style catalogue itself is the art domain's; only the latitude assertion is this domain's. |

**A lesson, not a runtime seam:** capability-as-reported-fact is credited to the Image Platform's
route-label defect (`bridge.go:21-22`; `image:ADR-016` is the platform's own record). Nothing
crosses at runtime — the two services never share a driver.

## The seams that do not exist

- **No per-seat spend budget.** The cost sink measures and warns (`DREAMCHAT_BEAT_COST_WARN_USD`);
  nothing refuses a call for cost. An agent adding a spend refusal is deciding something new —
  budgets that block are a product decision nobody has made.
- **No runtime config-fitness check.** The `ADR-P024` gate (boot check that a seat's resolved
  parameters can serve its contract) is deliberately unbuilt — see product.md §deliberately not
  built. Until it exists, the seam between this domain and operations is a written sequence, not a
  gate.
- **No streaming in the anthropic driver.** Only `openaicompat.go` implements `StreamingDriver`;
  the narrate path type-asserts and falls back identically (`bridge.go:81-93`). No decision forbids
  an anthropic implementation; none exists, and adding one is additive, not a redesign.
