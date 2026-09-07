# naming-wall · tech

**Repo:** `dreamchat-world-backend` · **Cluster:** WE-4 · The naming wall ·
**Parent bounded context:** World Engine

This file holds how the domain is built — storage, the definition and its two application points,
validation, traps. `naming-wall.product.md` holds what it means; `naming-wall.seams.md` holds what
crosses its boundary.

Line numbers into `core/db/schema.sql` are as of 2026-08-27; the file is regenerated, so re-locate by
grep before relying on one.

---

## Storage

- **`name_knowledge`** (`schema.sql:4020`) — names learned IN PLAY: `(world_id, holder_id, entity_id,
  name, learned_tick, source_event_id FK)`, `PRIMARY KEY (world_id, holder_id, entity_id)` — the PK
  preserves first recorded recognition. Genesis-seeded name knowledge stays in `perception_record`;
  `fn_perceived_name` (`:2645`) unions both and takes the earliest, so a seeded name stays
  authoritative for a viewer born knowing it.
- Everything else the wall reads it does not own: `entity_registry` (canonical names, descriptors),
  the `*_state` descriptor attrs, and visible `perception_record.spoken` (see `seams.md`).

## The definition — once, in SQL

**`fn_unearned_names(world, viewer)`** (`schema.sql:2982`; COMMENT at `:3055` names both consumers).
A name is unearned when all of: canonical name non-empty · the entity is not the viewer (*"rewriting
a man's own name to the descriptor strangers use for him is not perception, it is amnesia"*, `:2992`)
· `fn_display_name` differs from the canonical name · the label does not already contain the name
(the Ballast Crate clause, `:2999`). Token guarding adds each distinctive word of unearned **actor**
names, with six exclusions (`:3020-3037`) including the lowercase-corpus test — the corpus CTE
(`:3011-3018`) reads summaries, `payload->>'spoken'` (`:3013`), and every descriptor. `ORDER BY
length DESC` (`:3045-3047`) is **part of the shared definition** — longest first, so "Silas Holton"
is rewritten before "Silas" can bite into it.

`fn_unheard_names(world, viewer)` derives from that identity definition, excluding only terms
already present in the holder's visible stored speech. `NamingWall` uses this word guard;
`fn_viewer_text` retains the stricter identity guard. Neither predicate interprets name ownership.

**The label chain** it compares against: `fn_display_name` (`:1503`) is `COALESCE(fn_perceived_name,
actor_state descriptor, artifact_state descriptor, canonical_name)`. No registry-descriptor branch
and no group branch — see Traps. `fn_display_names_distinct` (`:1522`) adds a perceived anchor when
labels collide, applied once over the whole set (*"a collision is a property of the group"*);
callers rely on input order (`:1549`). `fn_batch_display_name` (`:1078`) shows a name only when
every batch mind resolves the same one — over-strictness is dull, never a leak.

## The two application points

1. **The seam — `fn_viewer_text`**: account prose is rewritten into holder-relative labels.
   The shared speech application calls it before appending that holder's accepted heard words.
   It never rewrites those words merely because their owner was not identified.
2. **The belt — `NamingWall`, `core/api/namingwall.go`**: loaded once per beat
   (`beatsstream.go:435`), it covers what a seat invents on its own. `Violations()` rejects a leaking
   narration segment inside `DecodeAndValidateNarration` (`narration.go:156`) — a model rewrites
   better than any substitution; `Scrub()` on the plain-prose fallback (`beatsstream.go:565`) and
   `scrubAll()` on NPC telegraph wind-ups (`:625`) — the two paths with no model to re-ask. A belt
   that cannot load **fails LOUD, never closed** (`beatsstream.go:437-440`): killing a beat over a
   projection read is not on the table, and fail-silent is what got us here.

## Teaching — recognized owners, not scanned words

Both commit doors call `fn_apply_speech_perception` for Communicated events. The accepted resolve
judgment, not a search through canonical names, decides each listener's associations (`ADR-038`).
A non-null owner `actor_id` writes `name_knowledge` and anchors a `perception_subject` to that
actor. The optional description is retained independently, including when recognition is present.
Related `about_actor_ids` add subject links, never names; they require a description. Without an
explicit owner `actor_id`, a description creates no actor and grants no label.

`perception_record.spoken` stores the holder's actual heard words. `fn_perceived_speech` supplies
the narration quote check; canonical `payload.spoken` is not a substitute for receiver-specific
hearing. Blocked listeners get no heard record; hidden ruled speech produces none, even for the
speaker. The historical backfill requires the complete source utterance in the existing record.

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `ADR-038` | Shared speech perception and the word/owner distinction. | A valid association is applied identically through ordinary and ruled speech. |
| `B-1`, `I-3` | Player surfaces consume the holder's perceptions. | Reading canonical speech instead exposes words a receiver may never have heard. |
| `D-6` | The word guard derives from the SQL identity definition, not a Go restatement. | Duplicated name predicates drift. |

### What you may not decide alone

1. **Teaching through string matching.** The old scanner is removed; recognition belongs to resolve.
2. **Granting recognition merely to retain a quote.** Heard words already have their own storage.
3. **Authoring a group wall or aliases.** Their re-entry conditions remain in `product.md`.
4. **Claiming lexical checks prove meaning.** The word guard cannot prove how prose identifies someone.

## Validation for this domain

pgTAP in `core/db/tests/`: `25_perception_naming_wall*`, `124_speech_perception*`,
`125_speech_perception_facts*`, `28_spoken_words*`, `29_article_aware*`, `43_perceived_name*`,
`44_display_name*`, `46_wall_clause*`, `121_name_token_wall*`, `27_distinguishing_labels*`,
`26_in_world_label*`. Go: `core/api/namingwall_test.go` (the founder's leak as a test),
`wall_test.go` `TestWall_NameStringConfinedToKnower`, `promptnames_test.go`. The `make reset` and
`-count=1` warnings in `perception-and-knowledge.tech.md` §Validation apply verbatim here.

**What counts as evidence:** a REFUSAL or a rewrite, reproduced. A nil belt is a legal state
(fail-LOUD) and `Violations` is nil-safe, so "clean text passed" proves nothing — the belt deleted
also passes clean text. Evidence is `TestNamingWall_RefusesTheFoundersLeak` shape: an unearned name
present, and refused.

**What counts as ceremony:** asserting a substitution when the viewer knows every name, or
treating a hand-authored association as evidence that the real model interpreted the scene.
Database application checks and production-binding interpretation probes answer different questions.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The wall was loaded, correct by its own rule, and useless.** Genesis stored slug join-keys (`silas_holton`) as canonical names; seats humanised them; whole-string matching found nothing. Token guarding exists because of this. | Migration `20260821120000:5` (the Ironmoor breach, live play 2026-08-20). |
| **A name lowercased in canon slips the token net.** Admitted hole; the whole-string row still stands behind it. | `digest/S13b` §16, quoting `20260821120000`'s own header. |
| **`speaker_label` has no belt of its own.** It reads `fn_display_name` straight; its protection is viewer-relativity at source. Bypassing `fn_display_name` to the registry is a `B-1` breach. | `core/api/namingwall_test.go:166,230`; `beathandler.go:103`. |
| **The ruled door historically never taught names.** | Superseded by the shared application in `20260906184826_shared_speech_perception.sql`; `124_speech_perception_test.sql` exercises both doors and actor-specific reads. |
| **A comment asserts a fallback that does not exist.** `worldgenesiscommit.go:285-288` says the *registry* descriptor is what `fn_display_name` falls back to; the function (`schema.sql:1503-1514`) reads only `*_state` attrs descriptors. Round-B review found the same (`digest/S07a` §8). Both sides recorded; not resolved here. | Compare the comment against the function. |
| **"The wall covers groups" is a refuted finding that keeps looking true.** The `unearned` CTE has no kind filter, but the predicate fails for groups anyway. | `digest/S07a` §8: claimed, refuted, conceded (*"my G5 was wrong"*). |

## Open questions

3. **The alias question** — a second name for a known person. Kept out deliberately
   (`product.md`); unanswered, not forgotten.
4. **The group-wall re-entry condition** — when a collective acts and earns a page, does it also earn
   a wall? Ruled territory.
