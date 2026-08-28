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
  *is* the first-hearing-wins rule. Genesis-seeded name knowledge stays in `perception_record`;
  `fn_perceived_name` (`:2645`) unions both and takes the earliest, so a seeded name stays
  authoritative for a viewer born knowing it.
- Everything else the wall reads it does not own: `entity_registry` (canonical names, descriptors),
  the `*_state` descriptor attrs, `canon_event.payload->>'spoken'` (see `seams.md`).

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

**The label chain** it compares against: `fn_display_name` (`:1503`) is `COALESCE(fn_perceived_name,
actor_state descriptor, artifact_state descriptor, canonical_name)`. No registry-descriptor branch
and no group branch — see Traps. `fn_display_names_distinct` (`:1522`) adds a perceived anchor when
labels collide, applied once over the whole set (*"a collision is a property of the group"*);
callers rely on input order (`:1549`). `fn_batch_display_name` (`:1078`) shows a name only when
every batch mind resolves the same one — over-strictness is dull, never a leak.

## The two application points

1. **The seam — `fn_viewer_text` at every perception write** (`:3062`): article-aware two-pass
   rewrite of unearned names into the holder's labels. Applied by WE-3's writers at
   `schema.sql:635` (`apply_ruled_event`, per receiver) and inside `generate_perceptions` at
   `:3456`, `:3477` (Communicated speaker/listener), `:3501` (move), `:3614` (ObjectRelocated). The
   seam is the real guarantee: perception content feeds every player-facing projection, *"fixing the
   seam fixes all of them"* (migration `20260809090005`).
2. **The belt — `NamingWall`, `core/api/namingwall.go`**: loaded once per beat
   (`beatsstream.go:435`), it covers what a seat invents on its own. `Violations()` rejects a leaking
   narration segment inside `DecodeAndValidateNarration` (`narration.go:156`) — a model rewrites
   better than any substitution; `Scrub()` on the plain-prose fallback (`beatsstream.go:565`) and
   `scrubAll()` on NPC telegraph wind-ups (`:625`) — the two paths with no model to re-ask. A belt
   that cannot load **fails LOUD, never closed** (`beatsstream.go:437-440`): killing a beat over a
   projection read is not on the table, and fail-silent is what got us here.

## Teaching — hearing a name

Inside `generate_perceptions`' Communicated arm, and **ordered by design**: `name_knowledge` is
inserted (`:3448`, `:3469`) *before* the holder's content is rendered (`:3456`, `:3477`), so
`fn_viewer_text` sees a just-earned name and leaves it standing — the wall keeps its single rule and
needs no speech exemption. `fn_names_in_text` (`:2546`) is case-SENSITIVE, word-bounded, and
documents its single caller — which is what licenses the asymmetry (`product.md`). `spoken` is NULL
for a nod or a shove, and NULL teaches nothing (`:3426-3428`). A speaker never learns his own name
from his own words (`:3451`, `:3472`).

## Technical decisions already made

| Id | What it settles | What breaks if you ignore it |
|---|---|---|
| `SPEC-033` | Landed. `name_knowledge` + teaching bolted to the fan-out; "could hear" = "got a perception row", never re-decided. | A second could-hear predicate drifts from the fan-out's. |
| `B-1`, `I-3` | The belt guards the player boundary; hidden truth (a canonical name) never reaches a payload unearned. | `I-3` is CI-enforced; the build fails. |
| `D-6` | One definition: the belt reads `fn_unearned_names`, never a Go restatement. | *"they MUST share the definition or the check is theatre"* (migration `20260809090006`). |

### What you may not decide alone

1. **Exempting any channel from the seam or the belt** — speech was the tempting one and was refused
   (`product.md` §deliberately-not-built).
2. **Changing the teaching source** — `payload.spoken` only is a founder-shaped correction with
   measured breaches behind it (migration `20260814170000`).
3. **Authoring a group wall** — its re-entry condition is ruled territory (`product.md`).
4. **Relaxing either half of the strictness asymmetry** — each direction's failure mode is documented
   and different.

## Validation for this domain

pgTAP in `core/db/tests/`: `25_perception_naming_wall*`, `26_hearing_teaches*`,
`27_hearing_teaches_only_spoken*`, `28_spoken_words*`, `29_article_aware*`, `43_perceived_name*`,
`44_display_name*`, `46_wall_clause*`, `121_name_token_wall*`, `27_distinguishing_labels*`,
`26_in_world_label*`. Go: `core/api/namingwall_test.go` (the founder's leak as a test),
`wall_test.go` `TestWall_NameStringConfinedToKnower`, `promptnames_test.go`. The `make reset` and
`-count=1` warnings in `perception-and-knowledge.tech.md` §Validation apply verbatim here.

**What counts as evidence:** a REFUSAL or a rewrite, reproduced. A nil belt is a legal state
(fail-LOUD) and `Violations` is nil-safe, so "clean text passed" proves nothing — the belt deleted
also passes clean text. Evidence is `TestNamingWall_RefusesTheFoundersLeak` shape: an unearned name
present, and refused.

**What counts as ceremony:** asserting a substitution on a viewer who has earned every name
(`Scrub` is identity there, `namingwall.go:69`), and any teaching test that feeds the name through
`summary` rather than `payload.spoken` — it asserts exactly the behaviour `20260814170000` deleted.

## Traps, with receipts

| The trap | The receipt |
|---|---|
| **The wall was loaded, correct by its own rule, and useless.** Genesis stored slug join-keys (`silas_holton`) as canonical names; seats humanised them; whole-string matching found nothing. Token guarding exists because of this. | Migration `20260821120000:5` (the Ironmoor breach, live play 2026-08-20). |
| **A name lowercased in canon slips the token net.** Admitted hole; the whole-string row still stands behind it. | `digest/S13b` §16, quoting `20260821120000`'s own header. |
| **`speaker_label` has no belt of its own.** It reads `fn_display_name` straight; its protection is viewer-relativity at source. Bypassing `fn_display_name` to the registry is a `B-1` breach. | `core/api/namingwall_test.go:166,230`; `beathandler.go:103`. |
| **The ruled door never teaches.** `apply_ruled_event` writes `payload.spoken` (`schema.sql:569-572`) and applies `fn_viewer_text` (`:635`) but inserts no `name_knowledge` — the only writers are `:3448`/`:3469` inside `generate_perceptions`. A name spoken through the ruled door is rewritten out of the listener's perception without ever being learnable — the pre-`SPEC-033` breach, alive on one path. `[INFER]` on impact: the ruled door has no traffic yet (`perception-and-knowledge.tech.md` §The second door). | Grep `name_knowledge` in `schema.sql`; the ruled-door fix is SPEC-038-shaped, see Open questions. |
| **A comment asserts a fallback that does not exist.** `worldgenesiscommit.go:285-288` says the *registry* descriptor is what `fn_display_name` falls back to; the function (`schema.sql:1503-1514`) reads only `*_state` attrs descriptors. Round-B review found the same (`digest/S07a` §8). Both sides recorded; not resolved here. | Compare the comment against the function. |
| **"The wall covers groups" is a refuted finding that keeps looking true.** The `unearned` CTE has no kind filter, but the predicate fails for groups anyway. | `digest/S07a` §8: claimed, refuted, conceded (*"my G5 was wrong"*). |

## Open questions

1. **Does the ruled-door fix include hearing-teaches?** The fix's shape depends on SPEC-038
   (`perception-and-knowledge.tech.md` §The second door is the one home); this domain's stake is the
   missing `name_knowledge` write, recorded in Traps.
2. **`SPEC-033`'s entry lags the code.** `docs/open-spec-items.md:983-990` still describes
   `payload.spoken` as an unbuilt "tightening option… explicitly NOT to be built until it earns its
   place"; migrations `20260809090009` and `20260814170000` built exactly that. Both sides recorded;
   amending a spec entry is a ruling, not a package edit.
3. **The alias question** — a second name for a known person. Kept out deliberately
   (`product.md`); unanswered, not forgotten.
4. **The group-wall re-entry condition** — when a collective acts and earns a page, does it also earn
   a wall? Ruled territory.
