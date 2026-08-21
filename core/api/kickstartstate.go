package main

// kickstartstate.go — the between-turns home of an unfinished world creation.
//
// The world itself commits when authoring ends (durable-worlds spec, 2026-08-21); what still needs
// a home between the kickstart turns is small: the chosen identity, the authored scenario options,
// any referenced-but-new people the character turn authored, and the running spend tally. It lives
// on the world row (`kickstart_state` jsonb) rather than in memory, because production lost two
// paid builds to the in-memory posture in one night: a process restart or a 15-minute TTL must
// never cost more than one re-asked question. Same secrecy discipline as `genesis_doc`: no route
// serves it, no projection selects it (AC-7 holds with the state in the database).
//
// There is deliberately no claim/lock. Saves happen only after a turn succeeds, the final commit
// guards the player stamp with `player_entity_id IS NULL`, and a concurrent duplicate answer
// therefore loses honestly (409) instead of racing.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// errNotResumable answers a kickstart against a world that exists but was never authored from a
// document — hand-seeded, templated, or created empty through POST /worlds. There is nothing to
// resume because nothing was ever half-created.
var errNotResumable = errors.New("this world was not authored from a brief — there is nothing to resume")

type kickstartIdentity struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
}

type kickstartScenario struct {
	Label       string `json:"label"`
	Place       string `json:"place"`
	Why         string `json:"why"`
	Stated      string `json:"stated"`
	Recommended bool   `json:"recommended,omitempty"`
}

// kickstartTally is the whole build's spend — every seat call across authoring and every kickstart
// turn — accumulated so the commit request can log one honest aggregate line (spec AC-10 posture,
// unchanged). JSON-tagged because it persists inside kickstart_state.
type kickstartTally struct {
	USD    float64 `json:"usd"`
	TokIn  int64   `json:"tok_in"`
	TokOut int64   `json:"tok_out"`
	Cached int64   `json:"cached"`
	Calls  int     `json:"calls"`
}

func (t *kickstartTally) add(usd float64, in, out, cached int64, calls int) {
	t.USD += usd
	t.TokIn += in
	t.TokOut += out
	t.Cached += cached
	t.Calls += calls
}

// kickstartState is what one unfinished creation still remembers between turns. Identity nil means
// the character question is still open — the same one-field state machine the draft store had.
type kickstartState struct {
	Identity  *kickstartIdentity  `json:"identity,omitempty"`
	Scenarios []kickstartScenario `json:"scenarios,omitempty"`
	NewCast   []genesisActor      `json:"new_cast,omitempty"`
	Tally     kickstartTally      `json:"tally"`
}

// creation is one unfinished world creation, loaded whole from the world row.
type creation struct {
	worldID string
	brief   string
	doc     *genesisDoc
	state   kickstartState
}

// uuidShaped gates the world id before it reaches a ::uuid cast, so a garbage id reads as "no such
// world" instead of a database syntax error surfacing as a 500.
var uuidShaped = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// loadCreation reads the row and sorts it into exactly one of the stated states: resumable,
// finished (errWorldAlreadyPlayable), not-authored (errNotResumable), or absent (errNoSuchWorld).
func loadCreation(ctx context.Context, pool *pgxpool.Pool, worldID string) (*creation, error) {
	if !uuidShaped.MatchString(worldID) {
		return nil, errNoSuchWorld
	}
	var brief *string
	var docJSON, stateJSON []byte
	var playerID *string
	err := pool.QueryRow(ctx,
		`SELECT brief, genesis_doc, kickstart_state, player_entity_id::text
		 FROM world WHERE world_id = $1::uuid`,
		worldID).Scan(&brief, &docJSON, &stateJSON, &playerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNoSuchWorld
	}
	if err != nil {
		return nil, fmt.Errorf("loadCreation: %w", err)
	}
	if playerID != nil {
		return nil, errWorldAlreadyPlayable
	}
	if docJSON == nil {
		return nil, errNotResumable
	}
	c := &creation{worldID: worldID}
	if brief != nil {
		c.brief = *brief
	}
	c.doc = &genesisDoc{}
	if err := json.Unmarshal(docJSON, c.doc); err != nil {
		return nil, fmt.Errorf("loadCreation: doc: %w", err)
	}
	if stateJSON != nil {
		if err := json.Unmarshal(stateJSON, &c.state); err != nil {
			return nil, fmt.Errorf("loadCreation: state: %w", err)
		}
	}
	return c, nil
}

// saveKickstartState persists the between-turns state. Guarded by IS NULL so a save racing the
// final commit cannot resurrect state on a finished world.
func saveKickstartState(ctx context.Context, pool *pgxpool.Pool, worldID string, st *kickstartState) error {
	raw, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("saveKickstartState: marshal: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE world SET kickstart_state = $2::jsonb
		 WHERE world_id = $1::uuid AND player_entity_id IS NULL`,
		worldID, string(raw)); err != nil {
		return fmt.Errorf("saveKickstartState: %w", err)
	}
	return nil
}

// mergeNewCast appends authored people not already present (case-insensitive by canonical name).
// A later turn may reference more people — a custom opening naming an uncle after the character
// turn named the parents — and each survives exactly once.
func mergeNewCast(existing, more []genesisActor) []genesisActor {
	seen := make(map[string]bool, len(existing))
	for _, a := range existing {
		seen[normName(a.CanonicalName)] = true
	}
	for _, a := range more {
		if key := normName(a.CanonicalName); !seen[key] {
			seen[key] = true
			existing = append(existing, a)
		}
	}
	return existing
}

// normName is the case-insensitive join key for canonical names.
func normName(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
