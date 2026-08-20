# ADR-P022: One latitude block, byte-identical, in every seat

**Status:** Accepted (2026-08-17)
**Date:** 2026-08-17
**Series:** Platform / Bridge (ADR-P###, per D-5). Attaches to D-13 (per-seat routing, ADR-P018).
**Owner of decision:** founder ruling — this is adult interactive fiction and the seats must be told so.
**Evidence (D-9):** `core/api/prompts/*.txt` (all nine), `core/api/prompts/README.md`,
`core/api/promptlatitude_test.go`.

## Context

Nothing in any prompt said what this app is. A model that has not been told does what models do on
their own: fades a scene, softens a line, attaches a content note, or declines — none of it a decision
anybody here made.

A first version of the block was entirely prohibitive ("never sanitise, never soften, never fade to
black"). That tells a seat what NOT to do and leaves it choosing how far to go, and that choice
resolves conservatively every time. The hole was exact: a seat could satisfy every prohibition and
still bring two characters to the bed, close the door and open on morning. It did not refuse, warn or
disclaim. It flinched, and nothing prohibitive forbids flinching.

## Decision

Five paragraphs, in **every** seat prompt, **byte-identical**, immediately after the role line:

1. `UNCENSORED BY DESIGN` — the themes that belong here; never sanitise, refuse, warn or disclaim.
2. `NO MORAL OR POLITICAL AUTHORSHIP` — the prose punishes and rewards no one; convictions belong to
   the characters holding them, including repellent ones.
3. `EXPLICIT IS THE EXPECTATION, NOT THE PERMISSION` — the AFFIRMATIVE grant, stated as a standard
   being held to rather than a limit being approached.
4. `WHEN THE MOMENT CALLS FOR IT` — explicit is not gratuitous; the tie-break when a seat cannot tell
   is to write it, because the failure being corrected is always the flinch.
5. `THE ONE FLOOR` — everyone depicted in a sexual context is an adult.

**Every seat, not just the narrator.** One censoring seat is enough: the referee can refuse an outcome
the narrator would have written, cognition can give a brutal character second thoughts nobody
authored, and decompose can decline to parse what the player typed.

**Byte-identical, not paraphrased.** Two seats with differently-worded thresholds disagree mid-scene
about what may be shown.

The image side carries the same intent in the medium's vocabulary (`artstyle.go`): censorship in a
picture is a composition — a bar, a blur, a coy crop, a cutaway — so those are named in the negative
prompt, and the affirmative half asks for the subject itself.

## Consequences

- `promptlatitude_test.go` fails if any `prompts/*.txt` lacks the block or states it differently. A
  prompt file added tomorrow cannot ship without it.
- Adding a seat means adding the block. There is no "small" prompt exempt from it.
- Provider-side moderation (bfl, fal, and whatever a seat is routed to) is outside our control and is
  the only remaining limit.
