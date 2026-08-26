package main

import (
	"context"
	"strings"
)

// WorldStatement is the world's GLOBAL statement — what this world IS, said once, as opposed to the
// per-beat world content the narrator already receives.
//
// The distinction is the whole point and it is easy to get wrong. The narrate prompt has never been
// starved of this world's own prose: PLACE carries the room's Tier-2 description, PRESENT carries the
// viewer's labels for the cast, and the perception lines are authored text (narrateprompt.go:157-268).
// All of that is per-beat and local. What no prompt has ever carried is the one sentence that says
// what world this is at all — so the narrator renders every room correctly and has no idea which world
// the rooms are in.
//
// WHERE IT COMES FROM, AND WHAT IT MUST NEVER BE. Every field below is the COMMITTED DOCUMENT's own
// content, denormalised onto the world row by commitWorldContent (worldgenesiscommit.go:84-94 for the
// theme, :124-129 for tagline and genesis_doc). It is deliberately NOT world.brief. The brief is the
// prose a user typed; the column comment states the rule outright — "Operational provenance, never
// rendered: no projection selects it" (core/db/schema.sql:4234) — and world_genesis.v1's tagline is
// specified as "Authored, never composed from the brief verbatim". Handing the narrator the brief
// while the document is short of it hands it material the state cannot back, which is exactly the
// founder-gate bug NEVER CONTRADICT OR EXTEND THE STATE exists to stop (narrateprompt.go:14-17,28):
// the driver dropped the payload, the narrator was left holding an instruction with nothing behind it
// and invented an entire scene. There is therefore no field here that can carry the brief, and
// worldStatementQuery does not select it. Both facts are tested (worldstatement_test.go).
//
// WHAT THIS BECOMES WHEN THE NEW CONTRACT LANDS. These five are what the committed world row actually
// has today, under world_genesis/1. world_model/6 replaces them with a richer global statement, and
// this struct is the seam that absorbs it: `world.premise` supersedes Premise (today's authored
// tagline), the `vocabulary` section supersedes Mood/Ornament with the world's minted terms, and
// `law[]` plus the `excluded[]` list become statable here for the first time. Until then Mood and
// Ornament ARE the minted vocabulary this world has — world_genesis.v1 specifies both as free
// vocabulary the author coins, not an enum. Nothing here waits on that landing.
type WorldStatement struct {
	Name     string // world.display_name      ← genesis_doc.world.display_name
	Premise  string // world.tagline           ← genesis_doc.world.tagline — one line of fiction in the world's own voice
	Mood     string // world.theme->>'mood'    ← genesis_doc.world.mood — one coined word for the atmosphere
	Ornament string // world.theme->>'ornament'← genesis_doc.world.ornament — one coined word for the visual register
	Region   string // genesis_doc.region.descriptor — the parent place every other place sits inside
}

// Empty reports whether there is nothing to say. A world authored before this path existed, or a
// hand-authored template row with no genesis document, yields an empty statement and the narrate
// prompt simply omits the block — the same discipline YOU ARE follows for a viewer with no aliases.
// An absent statement must never render an empty header line for the model to reason about.
func (s WorldStatement) Empty() bool {
	return strings.TrimSpace(s.Name) == "" &&
		strings.TrimSpace(s.Premise) == "" &&
		strings.TrimSpace(s.Mood) == "" &&
		strings.TrimSpace(s.Ornament) == "" &&
		strings.TrimSpace(s.Region) == ""
}

// worldStatementQuery is a named constant so the "never the brief" guarantee is checkable rather than
// a matter of trust: a test asserts this string does not mention the brief column. A query built
// inline could grow one in a later edit with nothing to catch it.
const worldStatementQuery = `SELECT display_name,
	        COALESCE(tagline, ''),
	        COALESCE(theme->>'mood', ''),
	        COALESCE(theme->>'ornament', ''),
	        COALESCE(genesis_doc->'region'->>'descriptor', '')
	 FROM world WHERE world_id = $1::uuid`

// loadWorldStatement reads one world's global statement. Called once per beat, like the naming wall
// (beatsstream.go:431): the statement changes only at genesis, so this is a single indexed row read on
// the primary key and never a per-candidate cost.
//
// It takes dbQuerier rather than *pgxpool.Pool (the same subset applyRuledEventOnQuerier uses,
// orchestrator.go:1691-1694) so the DB-backed test can run it against a world committed inside a
// rolled-back transaction — canon tables carry forbid_delete triggers, so a test world can never be
// cleaned up any other way.
func loadWorldStatement(ctx context.Context, q dbQuerier, worldID string) (WorldStatement, error) {
	var s WorldStatement
	if err := q.QueryRow(ctx, worldStatementQuery, worldID).
		Scan(&s.Name, &s.Premise, &s.Mood, &s.Ornament, &s.Region); err != nil {
		return WorldStatement{}, err
	}
	return s, nil
}
