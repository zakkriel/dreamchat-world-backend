# Collected Knowledge is grouped by subject (2026-08-09)

Amends SPEC-029 (#40). Frontend report: Mara's dossier renders **25 identical "Arrival" headings**
(`../ROUND-CLOSING-REPORT-2026-08-09.md`, FE-A).

## The measurement, first

Reproduced before touching anything, because the report named a symptom and not a cause: 25 accepted
events inside one authored moment (`in_world_label = 'Arrival'`, carried forward to every later
event in that moment by `trg_canon_event_carry_in_world_label`), each yielding one perception the
Player holds about Mara.

```
BEFORE                                    AFTER
event:…900025  label=Arrival  items=1     subject:bbbb…  label=<null, unheaded>  items=19
event:…900024  label=Arrival  items=1     subject:a400…  label=Sealed Note       items=5
… 23 more, every one "Arrival" …          subject:dddd…  label=Tavern            items=4
event:…000001  label=Day 1    items=2
28 groups                                 3 groups
```

## Two defects, not one

**The visible half — the label is a clock.** `in_world_label` is B-5's authored *display companion
to the logical tick*: the name of a moment, not of a subject. It is deliberately non-unique — the
trigger propagates it forward precisely so every event in a moment shares it — so using it as a
group heading guarantees collisions the moment a scene contains more than one event.

**The half nobody reported — one group per event is one group per log line.** Both Compendium PRDs
rule this out in the same words: *"Collected Knowledge should be grouped by topic, not by raw
timeline/log order… The user should not read an Actor page as a log… The raw event order belongs
more naturally in Timeline"* (Actors PRD §10; Artifacts PRD, Collected Knowledge). Fixing only the
heading — merging groups that share a moment label — would have produced one group called "Arrival"
holding 25 unrelated facts. That is still time, still not a topic, and still a log.

## The decision: group by about-ness

A topic has to come from something the world records. Exactly four signals exist per perception:

| signal | verdict |
|---|---|
| `epistemic_type` | HOW you learned it. The PRD lists source as a *within-group* attribute, and every item already ships it — grouping by it duplicates a field the frontend renders per line. |
| `source_event_id` | WHICH event. The log order the PRDs are ruling out. |
| `in_world_label` | WHEN. The defect above. |
| `perception_subject` | **WHAT IT IS ABOUT.** Written at write time by whatever creates the perception (SPEC-008 / ADR-035), for exactly this purpose. |

About-ness is the only axis that is genuinely a topic, and it is already the shape the mockups use —
"The informant" and "Dark Foxes connection" are entities. Nothing is invented and no model is
consulted: SQL cannot call one, and a synthesised topic would assert more than the viewer holds
(B-1). The alternatives the ruling asked about were both considered and both rejected above:
*suppress redundant labels* fixes the symptom and keeps the log; *a single group* is the pre-#40
behaviour SPEC-029 was raised to fix.

## Four sub-rulings

**A topic is a thing that recurs.** A record about several entities is filed under exactly **one** —
the co-subject it shares the most of this page's records with, for this viewer; ties break by label,
then id. Not repeated under each: the same paragraph printed twice under two headings is a new
redundancy defect on a surface whose whole complaint was redundancy. "What topics matter" (PRD §10)
is answered by recurrence, which is a fact about the viewer's own knowledge, fully deterministic.

**Headings use `fn_display_name`.** `fn_perceived_name` was tried first and **measured**: it reads
only the `world_genesis` naming substrate, so it returns NULL for the Sealed Note — a thing the
Player has observed and can plainly see — and the best topic on the page fell into the remainder.
`fn_display_name` is this repo's single answer to "what does *this viewer* call that thing", and is
already what the beat candidate whitelist and the scene surface put in front of the player every
beat. Inventing a fifth naming rule for one field would be the drift; if that function's
canonical-name tail ever leaks, the fix belongs in `fn_display_name` and all its callers.

**The reader is never a topic.** The viewer co-subjects nearly every record they hold — they were
there. An unfiltered pass filed two of Mara's records under a group headed **"Player"**. The viewer
is excluded, so a reader never appears as a subject of the dossier they are reading.

**The remainder is unheaded and first.** Records about the page's own subject and nothing else
nameable get `group_key = 'subject:' || <the page's own entity id>` and `group_label = null`. It is
emitted first because a heading-less block placed *between* two headed groups reads as belonging to
the heading above it.

## The shape does not move — `actor_page/2` stays

`group_key` / `group_label` / `items` are unchanged, and so is every item field. What changes is the
**value** of `group_key` (`event:<uuid>` → `subject:<uuid>`) and what `group_label` means. No field
added, removed or retyped, so `actor_page/2`, `location_page/1` and `artifact_page/1` all stay put.
Adding a field would have been breaking under `additionalProperties: false`; this is not.

Frontend contract, now written into the published schemas' `description` fields on all three pages:

- `group_key` is `subject:<entity uuid>` and is stable across reads.
- The group keyed `subject:<the page's own entity id>` is the remainder: **first, and `group_label`
  is null** — render its items with no heading.
- Other groups follow by recurrence, then recency, then id. Items inside a group stay in in-world
  chronological order, so a topic reads as it evolved.

Not changed, and available if the frontend ever wants a bump: each item's `source` object still
duplicates `epistemic_type`, which the FE-needs doc §2.5 flagged as ignored. Removing it is a shape
change and would cost a re-pin for a field nobody reads — not worth a version on its own.

## Test coverage

`core/db/tests/26_knowledge_grouping_test.sql` (10), against the reproduction fixture above rather
than an approximation of it:

- 25 records in one moment no longer render as 25 groups; no group is headed `Arrival`; no group key
  starts with `event:`.
- A recurring co-subject becomes a topic labelled with the viewer's own name for it (twice over).
- The viewer is never a topic.
- The remainder is keyed by the page's own subject, comes first, and its label is JSON null.
- **Every record appears exactly once** across all groups, and **regrouping loses nothing** — the
  item count equals the number of held non-genesis records about the target.
- The wall: a topic built from the Player's records never appears on Jonas's copy of the same page.

`core/db/tests/24_compendium_lenses_test.sql`'s event-keying assertion is retargeted to subject
keying — the multi-subject fixture record now files under `subject:<Sealed Note>`.
