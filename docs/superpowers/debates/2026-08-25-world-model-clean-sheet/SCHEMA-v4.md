# world_model/4 — the generative contract

v3 treated the document as a transcription of a brief. **It is not.** Genesis *invents*: a four-hundred
word brief must become a world that plays. v4 states that as a contract obligation rather than leaving
it implicit.

Delta over v3. Everything in v3 holds unless changed here.

---

## 1. What genesis is

Four acts, in order. The contract governs all four, not just the last.

1. **Understand** the brief as a world, not as text.
2. **Realise** everything stated or referenced by the brief.
3. **Infer** what the stated content entails but does not say.
4. **Emit** one valid, sufficient document.

> Genesis does not limit itself to the description. It builds enough that whatever comes next is
> playable.

### 1.1 The inference discipline

**Every invention must be downstream of something stated.** A guild exists because a trade was stated;
it does not exist because worlds have guilds. An invention that traces to nothing stated is genre bleed,
and under §2 it is mechanically detectable rather than a matter of taste.

**Invent as much as sufficiency requires, and no more.** The author's description stays load-bearing;
it is not buried under generated filler.

---

## 2. Provenance — new, and required on every element

```jsonc
"source": "stated"                                  // the brief said it
"source": { "inferred_from": ["<name>", "<name>"] } // and what it follows from
```

Four things depend on it, which is why it is not bookkeeping:

- the pre-build confirmation surface can only show what was **inferred**, since that is the only part
  worth correcting;
- **stated outranks inferred** on every contradiction, always;
- amending the brief re-infers dependents and leaves stated content untouched;
- an inference chain that bottoms out in nothing stated is a refusal (**R13**).

### 2.1 Load-bearing inference — derived, not authored

The confirmation surface shows only load-bearing inferences. An inference is load-bearing iff **either**
something else in the document references it, **or** it appears in `law`, `norms`, `accumulators`,
`opposition`, `offices`, or `excluded`. Everything else — names, descriptors, individual rooms — is
texture and stays hidden. No new field: this is computed.

---

## 3. Sufficiency — a second bar, distinct from validity

Validity (O1–O11) is a floor: the thing runs. Sufficiency is the real target: **the world is open.**

> **The sufficiency test: every boundary a player reaches must be a fictional boundary, never an
> authoring boundary.**

A locked door, a wall, a taboo, a sea with no boat — good, and the world is closed *on purpose*. An edge
reached because nobody authored past it — failure. The player experiences both as "I can't go there";
only one of them is a world.

Sufficiency is **not a count**. Checkable, without knowing what kind of world it is:

| # | Sufficiency condition |
|---|---|
| S1 | every extent reachable from an arrival has content — something to see, take, or speak to |
| S2 | every entity with `agency` a player can reach wants something they could act on |
| S3 | every name any authored text mentions resolves to an entity, or is explicitly `excluded` |
| S4 | every `passage` leads to an authored extent, or is `obstructed` for a fictional reason |
| S5 | every `opposition` and `accumulator` has somewhere to go — an outcome the world can reach |
| S6 | no authored text implies a place, group, office or practice that does not exist |

S3 and S6 are the ones that catch a thin world: a brief that mentions a Council produces a Council, and
a Council implies people, a seat, and something it decides.

---

## 4. Fixes from the v3 test

**F1 · Class–exemplar pairs.** Flagged independently by three encoders across three worlds — the
sharpest finding of the round. Some quantities are *both* canon-fact and engine-computed; forcing a
choice produced documents that "say the number twice and mean it once."

```jsonc
"decay": { "class": "brief", "exemplar": "three nights" }
```

The class remains the interface and the builder still owns the ladder — but must calibrate it so the
exemplar holds. One fact, one place. `exemplar` is fiction and may contain a number.

**F2 · `demand` does not imply `holding`.** The residual Grelda divergence. An entity that *consumes* a
substance needs only `demand`. `holding` is declared only if it *stores* one. *Forbidden:* `holding`
with a permanently empty `holds[]`.

**F3 · `interest` satisfies O3 for `collective` entities.** The contract had two keys for one idea and
an obligation naming only one, so a document that said what an institution wanted was refused for
saying it in the other field.

**F4 · `bulk_class` is required on `matter` entities that are not also `agency`**, and defaulted
otherwise. Requiring it on every named human produced `"moderate"` nine times — noise that invites a
builder to treat a body as cargo.

**F5 · R5 restated.** Promotion out of a `magnitude` entity is legal and expected; what is refused is
*authoring* an individual reference to a magnitude that was never promoted. Magnitudes are no longer
authored by subtraction, which "silently rots as play promotes more."

**F6 · `admits` may name an act.** Falling asleep is a passage into the dream, and
`admits: [{movement: "walk"}]` was a lie. A predicate may name a movement, condition, standing, office,
**or act**.

**New refusal — R13:** an `inferred_from` chain that does not terminate in stated content.

---

## 5. What "agreement" now means

The old metric is void. Two genesis runs from one brief **will** invent different worlds, and that is
correct behaviour, not a defect. Agreement is now four separate claims:

| Claim | Test |
|---|---|
| **Stated fidelity** | the `stated` layer is identical across runs |
| **Validity** | both satisfy O1–O11, violate no R1–R13 |
| **Sufficiency** | both satisfy S1–S6 |
| **Divergence is confined** | all differences live in the `inferred` layer |

A run that invents differently is fine. A run that *states* differently, or that is thin, is not.

---

## 6. Consequence

Every encoding made so far is now invalid: none carries provenance, and none was built to a sufficiency
bar. That is expected — until v4 there was nothing to be insufficient against.

The next test is therefore not another encoding. It is: **from the 400-word tier alone, produce a
document that satisfies S1–S6** — and then check whether it is recognisably the author's world or a
generic one wearing their nouns. That is the first test of the thing this schema actually exists to do.
