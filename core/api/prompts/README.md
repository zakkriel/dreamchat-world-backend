# Fixed prompt rulebooks

## The latitude block (in every file, verbatim)

Every prompt here carries the same five paragraphs — `UNCENSORED BY DESIGN`, `NO MORAL OR POLITICAL
AUTHORSHIP`, `EXPLICIT IS THE EXPECTATION, NOT THE PERMISSION`, `WHEN THE MOMENT CALLS FOR IT`, and
`THE ONE FLOOR` — immediately after its role line. This is adult interactive fiction: the seats are
told so explicitly, because a model that has not been told will fade a scene, soften a line, attach a
content note, or decline outright.

The middle paragraph is the load-bearing one, and it is stated AFFIRMATIVELY on purpose. The first
version of this block was entirely prohibitive — "never sanitise, never soften, never fade to black"
— which tells a seat what not to do while leaving it to choose how far to go, and that choice
resolves conservatively every time. A seat could satisfy every prohibition in the block and still
close the door and cut to morning: it did not refuse, warn or disclaim, it just flinched, and nothing
prohibitive forbids flinching. Permission has to read as a standard being held to, not a limit being
allowed near. `WHEN THE MOMENT CALLS FOR IT` is the other half of that: explicit is not gratuitous,
and the tie-break when a seat cannot tell is to write it, because the failure being corrected is
always the flinch.

It is in EVERY seat, not just the narrator, and byte-identical rather than paraphrased. One seat
censoring is enough to break the world: the referee can refuse an outcome the narrator would have
written, and cognition can give a brutal character second thoughts nobody authored. Two seats with
differently-worded thresholds contradict each other mid-scene.

`promptlatitude_test.go` enforces both properties — presence in every `prompts/*.txt`, and that the
block is identical everywhere. A prompt file added without it fails the suite.


Every fixed prompt rulebook the model seats read, in one place, config-style — mirroring the
`core/api/schema/*.json` + `//go:embed` pattern used for the JSON Schema leashes. Each file below is
embedded verbatim at build time into a Go var (see the `//go:embed prompts/*.txt` directive next to
each var's declaration) and is byte-identical to the const it replaced.

- `decompose.txt` — the decompose seat. Injected as the STABLE HEADER (prompt-prefix, not the system
  field) at the top of `buildDecomposePrompt` (`beathandler.go`), before the SCENE/CANDIDATES/PLAYER
  INPUT sections.
- `narrate.txt` — the narrate seat. Injected as the STABLE HEADER (prompt-prefix) at the top of
  `buildNarratePrompt` (`narrateprompt.go`), before the THE WORLD/PLACE/PRESENT/WHAT JUST
  HAPPENED/RECENT BACKGROUND sections (delta-first: new events vs already-known context). THE WORLD is
  the world's global statement and is rendered by `narrateSceneBody`, never appended to this file: the
  plain fallback slices the header at `OUTPUT — STRUCTURED NARRATION SEGMENTS`, so anything added after
  that line reaches only the structured path. Rules meant for all three builders go BEFORE it.
- `cognition.txt` — the cognition seat, shared by BOTH the batch and isolated calls. Injected as the
  STABLE HEADER (prompt-prefix) at the top of `buildCognitionPrompt` (`cognitionprompt.go`), before
  the SCENE/THE MINDS YOU SPEAK FOR/(private)/PUBLIC MOMENT/IMMINENT sections.
- `resolve.txt` — the resolve (referee) seat. Injected as the STABLE HEADER (prompt-prefix) at the
  top of `buildResolvePrompt` (`resolveprompt.go`), before the FACTS/ATTEMPT(S)/PLAYER ANSWER/repair
  sections.
- `world_actor.txt` — the world_actor seat (Living World / Task 8, the station's only LLM boundary).
  Injected as the STABLE HEADER (prompt-prefix) at the top of `buildWorldActorPrompt`
  (`worldactorprompt.go`), before the WORLD/CURRENT SCENE/DRAWN SIZE sections. World-omniscient,
  TRUTH-side (mirrors resolve.txt's licensing) — never perception-scoped.
- `world_understanding.txt` — the understanding pass. Identity only; no places or people.
- `world_fill.txt` — one scheduled work item of identity-governed fill.
- `system-anthropic.txt` — every call made through the anthropic driver. Injected as the `system`
  FIELD of the Messages API request body (`anthropic.go`), not a prompt prefix — it rides alongside
  whatever prompt the calling seat assembled. Still driver-owned (D-13 keeps provider shaping in the
  driver), just readable here alongside the rest.

Dynamic world data — perceptions, candidates, scene facts, present minds, the public moment, held
attempts, the player's raw text — is assembled in CODE, per beat, at each seat's call site. These
files are only the fixed rulebooks: the standing instructions that never change from one call to the
next. Nothing in this folder varies per world, per actor, or per beat.
