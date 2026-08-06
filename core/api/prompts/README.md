# Fixed prompt rulebooks

Every fixed prompt rulebook the model seats read, in one place, config-style — mirroring the
`core/api/schema/*.json` + `//go:embed` pattern used for the JSON Schema leashes. Each file below is
embedded verbatim at build time into a Go var (see the `//go:embed prompts/*.txt` directive next to
each var's declaration) and is byte-identical to the const it replaced.

- `decompose.txt` — the decompose seat. Injected as the STABLE HEADER (prompt-prefix, not the system
  field) at the top of `buildDecomposePrompt` (`beathandler.go`), before the SCENE/CANDIDATES/PLAYER
  INPUT sections.
- `narrate.txt` — the narrate seat. Injected as the STABLE HEADER (prompt-prefix) at the top of
  `buildNarratePrompt` (`narrateprompt.go`), before the PLACE/PRESENT/WHAT JUST HAPPENED/RECENT
  BACKGROUND sections (delta-first: new events vs already-known context).
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
- `system-anthropic.txt` — every call made through the anthropic driver. Injected as the `system`
  FIELD of the Messages API request body (`anthropic.go`), not a prompt prefix — it rides alongside
  whatever prompt the calling seat assembled. Still driver-owned (D-13 keeps provider shaping in the
  driver), just readable here alongside the rest.

Dynamic world data — perceptions, candidates, scene facts, present minds, the public moment, held
attempts, the player's raw text — is assembled in CODE, per beat, at each seat's call site. These
files are only the fixed rulebooks: the standing instructions that never change from one call to the
next. Nothing in this folder varies per world, per actor, or per beat.
