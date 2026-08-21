package main

// worldgenesiscommit.go — turning an authored world_genesis/1 document into a world you can walk into.
//
// This is the deterministic half of world creation, and it is deliberately the LARGER half: everything
// here is structure the model was never allowed to touch. It mirrors fn_instantiate_drowned_lantern
// (migration 20260813142100) step for step, because that hand-authored function is the only worked
// proof of what a playable world needs, and a generated world that differs from it structurally is a
// generated world that plays differently. Same tables, same order, same origin='fast_path', same tick
// ladder, same triggers — trg_validate_tension, sm_project and the append-only guards all fire on these
// inserts exactly as they fire on the template's, which is what makes AC-12 true rather than claimed.
//
// TWO TRANSACTIONS since the durable-worlds spec (2026-08-21). commitWorldContent writes everything
// the world IS — entities, naming, history, minds, opening state — the moment authoring succeeds;
// player_entity_id stays NULL, so the directory lists a real world that is not yet enterable.
// commitArrival is the later, retryable transaction that makes it playable: the player, the arrival,
// any people the chosen identity referenced into existence, and the stamp. A failure in EITHER leaves
// no half-world: content commits whole or not at all, and `playable:true` still means every rung
// exists, because only the arrival transaction may set it.
//
// WHY THE WRITES ARE DIRECT rather than through apply_event: genesis is the one moment when the actors
// an event would reference do not exist yet. apply_event validates against a populated world and cannot
// bootstrap one — which is exactly why the template writes canon_event/state_mutation/perception_record
// directly with origin='fast_path', and why doing the same here is precedent rather than a bypass.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
)

// errWorldAlreadyPlayable answers a final kickstart turn that lost the race (or repeated itself):
// the guarded player stamp found player_entity_id already set. The handler renders it as 409.
var errWorldAlreadyPlayable = errors.New("this world already has its player — walk in")

// The tick ladder, in the shape the hand-authored world already proves plays correctly:
//
//	25  world_genesis — who knows whose name before anything happens
//	30+ backstory     — one accepted event per authored beat of history, ascending
//	40  scene genesis — the whole opening state, ordered by beat_seq under one event
//	50  arrival       — the player walks in; the world's highest tick, and the live
//	                    handler mints the next beat after it
//
// Absolute rather than relative because a world's clock starts at 0 (fn_world_now returns 0 with no
// events) and every authored perception sets acquired_tick = valid_tick = its source event's tick, which
// satisfies I-9 by construction. The gaps are deliberate: they leave room to insert without renumbering.
const (
	genesisNamingTick        int64 = 25
	genesisBackstoryBaseTick int64 = 30
	genesisSceneTick         int64 = 40
	genesisArrivalTick       int64 = 50
)

// What a person can carry, in the units seed_world_defaults speaks (SPEC-026). The template gives the
// player 80; a generated player is a person like any other, so it gets the same rather than a new number
// invented here.
const genesisPlayerMaxLoad = 80

// genesisIDs is the id map for one build: every authored canonical_name resolved to the uuid the ENGINE
// minted for it. The seat can only join its parts by name (it cannot emit a uuid), so this map is where
// the authored document stops being prose and becomes a graph.
type genesisIDs struct {
	region string
	player string
	places map[string]string
	cast   map[string]string
	things map[string]string // objects and ways alike: both are artifacts
}

// commitWorldContent writes everything the world IS inside the caller's transaction and returns the
// new world id. Order is load-bearing and follows the template: the directory row and operating
// defaults first (every later call fails without them), then identity, then the events that justify
// knowledge, then knowledge, then state. The player and the arrival are deliberately absent —
// player_entity_id stays NULL until commitArrival — and the authored document itself lands on the
// row, because the kickstart turns author against it and the arrival commit recomputes deterministic
// geometry from it. It is server-side truth: no projection selects genesis_doc and no route serves
// it, so the AC-7 secrecy boundary the in-memory draft store used to hold now holds in the database.
func commitWorldContent(ctx context.Context, tx pgx.Tx, doc *genesisDoc, brief, artStyleChoice string) (string, error) {
	theme, err := json.Marshal(map[string]string{
		"schema_version": "world_theme/1",
		"accent":         genesisAccent(doc.World.DisplayName),
		"mood":           strings.TrimSpace(doc.World.Mood),
		"ornament":       strings.TrimSpace(doc.World.Ornament),
	})
	if err != nil {
		return "", fmt.Errorf("commitWorldContent: theme: %w", err)
	}

	worldID, err := createWorldTx(ctx, tx, strings.TrimSpace(doc.World.DisplayName), theme)
	if err != nil {
		return "", fmt.Errorf("commitWorldContent: create world: %w", err)
	}

	ids, err := registerEntities(ctx, tx, worldID, doc)
	if err != nil {
		return "", err
	}
	// The naming event is written for its side effect only: the per-viewer name perceptions that hang off
	// it. Nothing else may cite it — see the warning in writeMinds about what fn_perceived_name does with
	// anything sourced from a world_genesis event.
	if _, err := writeNamingEvent(ctx, tx, worldID, doc, ids); err != nil {
		return "", err
	}
	historyEventIDs, err := writeHistory(ctx, tx, worldID, doc, ids)
	if err != nil {
		return "", err
	}
	if err := writeMinds(ctx, tx, worldID, doc, ids, historyEventIDs); err != nil {
		return "", err
	}
	if err := writeOpeningState(ctx, tx, worldID, doc, ids); err != nil {
		return "", err
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("commitWorldContent: marshal doc: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE world SET tagline = $2, brief = $3, art_style = $4, genesis_doc = $5::jsonb
		 WHERE world_id = $1::uuid`,
		worldID, strings.TrimSpace(doc.World.Tagline), strings.TrimSpace(brief),
		nullableArtStyleChoice(artStyleChoice), string(docJSON)); err != nil {
		return "", fmt.Errorf("commitWorldContent: store doc: %w", err)
	}
	return worldID, nil
}

// commitArrival is the transaction that makes a committed world playable: the player entity, any
// people the chosen identity referenced into existence (newCast), the arrival event with the
// player's state and their one direct perception, the world_character row, and — last, guarded —
// the player_entity_id stamp that IS the mechanical definition of playable.
//
// doc is the world's stored genesis document with Arrival already set to the chosen identity and
// scenario, and it must have passed validate() over the MERGED cast (existing plus newCast) before
// this is called. newCast entries are registered and given minds here because they did not exist at
// content-commit time; their traits and secrets ground in the arrival event — the moment that made
// them real — never in the world_genesis event (see writeMinds on what fn_perceived_name does with
// anything sourced there). Their names and the player's are premise knowledge, mutual, acquired at
// the arrival tick: you know your own kin, and they know you (I-9 holds — nothing is learned before
// it happened, only exactly when).
func commitArrival(ctx context.Context, tx pgx.Tx, worldID string, doc *genesisDoc, newCast []genesisActor) error {
	ids, err := loadGenesisIDs(ctx, tx, worldID)
	if err != nil {
		return err
	}
	for _, a := range newCast {
		name := strings.TrimSpace(a.CanonicalName)
		id, err := registerActor(ctx, tx, worldID, name, strings.TrimSpace(a.Descriptor))
		if err != nil {
			return err
		}
		ids.cast[name] = id
	}
	if ids.player, err = registerActor(ctx, tx, worldID,
		strings.TrimSpace(doc.Arrival.CanonicalName), strings.TrimSpace(doc.Arrival.Descriptor)); err != nil {
		return err
	}

	arrivalEventID, err := writeArrival(ctx, tx, worldID, doc, ids, newCast)
	if err != nil {
		return err
	}
	for _, a := range newCast {
		if err := insertMind(ctx, tx, worldID, ids.cast[strings.TrimSpace(a.CanonicalName)], a,
			arrivalEventID, genesisArrivalTick); err != nil {
			return err
		}
	}
	if len(newCast) > 0 {
		// Mutual name knowledge between the player and each referenced person. Sourced from the one
		// world_genesis event because fn_perceived_name reads ONLY perceptions sourced there as names;
		// acquired at the arrival tick because that is when the premise made it true here.
		var namingEventID string
		if err := tx.QueryRow(ctx,
			`SELECT event_id::text FROM canon_event WHERE world_id = $1::uuid AND event_type = 'world_genesis'`,
			worldID).Scan(&namingEventID); err != nil {
			return fmt.Errorf("commitArrival: naming event: %w", err)
		}
		playerName := strings.TrimSpace(doc.Arrival.CanonicalName)
		for _, a := range newCast {
			name := strings.TrimSpace(a.CanonicalName)
			kinID := ids.cast[name]
			if err := writePerception(ctx, tx, worldID, ids.player, namingEventID, name, "told",
				genesisArrivalTick, []string{kinID}); err != nil {
				return fmt.Errorf("commitArrival: player knows %q: %w", name, err)
			}
			if err := writePerception(ctx, tx, worldID, kinID, namingEventID, playerName, "told",
				genesisArrivalTick, []string{ids.player}); err != nil {
				return fmt.Errorf("commitArrival: %q knows the player: %w", name, err)
			}
		}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO world_character (world_id, entity_id, descriptor, canonical_name)
		 VALUES ($1::uuid, $2::uuid, $3, $4)`,
		worldID, ids.player, strings.TrimSpace(doc.Arrival.Descriptor),
		strings.TrimSpace(doc.Arrival.CanonicalName)); err != nil {
		return fmt.Errorf("commitArrival: character row: %w", err)
	}

	// The stamp, last and guarded: until every entity and event is in place, a non-null
	// player_entity_id would advertise a world that cannot be entered — and IS NULL means two
	// concurrent final answers cannot both land.
	res, err := tx.Exec(ctx,
		`UPDATE world SET player_entity_id = $2::uuid, kickstart_state = NULL
		 WHERE world_id = $1::uuid AND player_entity_id IS NULL`,
		worldID, ids.player)
	if err != nil {
		return fmt.Errorf("commitArrival: stamp player: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errWorldAlreadyPlayable
	}
	return nil
}

// loadGenesisIDs rebuilds the canonical-name → entity-id map from entity_registry, for the arrival
// transaction that runs against a world committed earlier (possibly by another process entirely).
// The region lands in the places map under its descriptor name; nothing at arrival time needs it
// split out, and a place lookup by canonical name cannot collide with it in a validated document.
func loadGenesisIDs(ctx context.Context, tx pgx.Tx, worldID string) (*genesisIDs, error) {
	ids := &genesisIDs{
		places: map[string]string{},
		cast:   map[string]string{},
		things: map[string]string{},
	}
	rows, err := tx.Query(ctx,
		`SELECT entity_kind, canonical_name, entity_id::text FROM entity_registry
		 WHERE world_id = $1::uuid AND status = 'active'`, worldID)
	if err != nil {
		return nil, fmt.Errorf("loadGenesisIDs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind, name, id string
		if err := rows.Scan(&kind, &name, &id); err != nil {
			return nil, fmt.Errorf("loadGenesisIDs: scan: %w", err)
		}
		switch kind {
		case "location":
			ids.places[name] = id
		case "actor":
			ids.cast[name] = id
		default:
			ids.things[name] = id
		}
	}
	return ids, rows.Err()
}

// registerActor mints one actor id at arrival time — the player, or a person the identity
// referenced into existence. Same insert registerEntities uses, shared shape by construction.
func registerActor(ctx context.Context, tx pgx.Tx, worldID, canonicalName, descriptor string) (string, error) {
	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO entity_registry (world_id, entity_kind, canonical_name, descriptor, status)
		 VALUES ($1::uuid, 'actor', $2, $3, 'active') RETURNING entity_id::text`,
		worldID, canonicalName, descriptor).Scan(&id); err != nil {
		return "", fmt.Errorf("registerActor: %q: %w", canonicalName, err)
	}
	return id, nil
}

// nullableArtStyleChoice keeps "no choice" as SQL NULL rather than an empty string. The column's
// CHECK refuses blank text on purpose: "set, but to nothing" is a third state every reader would
// have to special-case beside NULL, and it means nothing a person chose.
func nullableArtStyleChoice(choice string) any {
	if strings.TrimSpace(choice) == "" {
		return nil
	}
	return choice
}

// registerEntities mints an id for every authored thing and writes entity_registry. The descriptor goes
// on the row as well as into attrs later: the registry descriptor is what fn_display_name falls back to
// before the canonical name, and the canonical name is the LAST resort — reaching it means a viewer read
// a name nobody gave them.
func registerEntities(ctx context.Context, tx pgx.Tx, worldID string, doc *genesisDoc) (*genesisIDs, error) {
	ids := &genesisIDs{
		places: make(map[string]string, len(doc.Places)),
		cast:   make(map[string]string, len(doc.Cast)),
		things: make(map[string]string, len(doc.Objects)+len(doc.Ways)),
	}

	insert := func(kind, canonicalName, descriptor string) (string, error) {
		var id string
		err := tx.QueryRow(ctx,
			`INSERT INTO entity_registry (world_id, entity_kind, canonical_name, descriptor, status)
			 VALUES ($1::uuid, $2, $3, $4, 'active') RETURNING entity_id::text`,
			worldID, kind, canonicalName, descriptor).Scan(&id)
		if err != nil {
			return "", fmt.Errorf("registerEntities: %s %q: %w", kind, canonicalName, err)
		}
		return id, nil
	}

	var err error
	if ids.region, err = insert("location", strings.TrimSpace(doc.Region.Descriptor), strings.TrimSpace(doc.Region.Descriptor)); err != nil {
		return nil, err
	}
	for _, p := range doc.Places {
		id, err := insert("location", strings.TrimSpace(p.CanonicalName), strings.TrimSpace(p.Descriptor))
		if err != nil {
			return nil, err
		}
		ids.places[strings.TrimSpace(p.CanonicalName)] = id
	}
	for _, a := range doc.Cast {
		id, err := insert("actor", strings.TrimSpace(a.CanonicalName), strings.TrimSpace(a.Descriptor))
		if err != nil {
			return nil, err
		}
		ids.cast[strings.TrimSpace(a.CanonicalName)] = id
	}
	for _, o := range doc.Objects {
		id, err := insert("artifact", strings.TrimSpace(o.CanonicalName), strings.TrimSpace(o.Descriptor))
		if err != nil {
			return nil, err
		}
		ids.things[strings.TrimSpace(o.CanonicalName)] = id
	}
	// A way is an artifact whose attrs.connects joins two rooms — portals are not a table. Keyed by a
	// composite name so two doors with the same descriptor cannot collide.
	for _, w := range doc.Ways {
		key := wayKey(w)
		id, err := insert("artifact", key, strings.TrimSpace(w.Descriptor))
		if err != nil {
			return nil, err
		}
		ids.things[key] = id
	}
	return ids, nil
}

// wayKey names a portal uniquely without inventing prose. The descriptor alone is not unique — a world
// may honestly hold two "a low door" — and entity_registry.canonical_name is a join key here.
func wayKey(w genesisWay) string {
	return fmt.Sprintf("%s (%s → %s)", strings.TrimSpace(w.Descriptor), strings.TrimSpace(w.FromPlace), strings.TrimSpace(w.ToPlace))
}

// writeNamingEvent lays down the one world_genesis event and the name knowledge that hangs off it.
//
// WHO KNOWS WHOSE NAME is a knowledge claim like any other and needs a valid path (B-2), so it is not
// blanket: two people know each other's names when the authored history put them in the same event. A
// pair who never shared a moment do not know each other, which is honest and emergent rather than a
// convenience. THE PLAYER APPEARS NOWHERE IN THIS: they know no one's name, and no one knows theirs.
func writeNamingEvent(ctx context.Context, tx pgx.Tx, worldID string, doc *genesisDoc, ids *genesisIDs) (string, error) {
	var eventID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, event_type, summary, in_world_tick, in_world_label,
		                          status, accepted_at, origin, visibility_scope)
		 VALUES ($1::uuid, 'world_genesis', $2, $3, 'Genesis', 'accepted', now(), 'fast_path', 'public')
		 RETURNING event_id::text`,
		worldID, "the world as it stood before anyone arrived", genesisNamingTick).Scan(&eventID); err != nil {
		return "", fmt.Errorf("writeNamingEvent: event: %w", err)
	}

	// Who shared a moment with whom, from the authored history.
	together := map[string]map[string]bool{}
	for _, h := range doc.History {
		for _, a := range h.Who {
			for _, b := range h.Who {
				a, b = strings.TrimSpace(a), strings.TrimSpace(b)
				if a == b {
					continue
				}
				if together[a] == nil {
					together[a] = map[string]bool{}
				}
				together[a][b] = true
			}
		}
	}
	for holder, known := range together {
		holderID, ok := ids.cast[holder]
		if !ok {
			continue
		}
		for name := range known {
			subjectID, ok := ids.cast[name]
			if !ok {
				continue
			}
			if err := writePerception(ctx, tx, worldID, holderID, eventID, name, "told",
				genesisNamingTick, []string{subjectID}); err != nil {
				return "", fmt.Errorf("writeNamingEvent: %q knows %q: %w", holder, name, err)
			}
		}
	}
	return eventID, nil
}

// writeHistory writes one accepted event per authored beat of history, oldest first, and the perceptions
// it left behind. Ascending ticks from genesisBackstoryBaseTick: position in the authored array is the
// ONLY ordering signal the seat may give (it cannot emit a tick), and this is where that becomes time.
//
// The events are AttributeChanged with no mutations, matching the template's backstory exactly. That is
// not a shrug: generate_perceptions has no AttributeChanged branch, so nothing fans out automatically and
// every belief is written explicitly against the event that justifies it — which is the only way to
// author knowledge that satisfies I-2 without faking a fan-out nobody received.
func writeHistory(ctx context.Context, tx pgx.Tx, worldID string, doc *genesisDoc, ids *genesisIDs) ([]string, error) {
	eventIDs := make([]string, 0, len(doc.History))
	for i, h := range doc.History {
		tick := genesisBackstoryBaseTick + int64(i)
		var eventID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO canon_event (world_id, scene_id, event_type, summary, in_world_tick, in_world_label,
			                          status, accepted_at, origin)
			 VALUES ($1::uuid, $2::uuid, 'AttributeChanged', $3, $4, 'Backstory', 'accepted', now(), 'fast_path')
			 RETURNING event_id::text`,
			worldID, ids.places[strings.TrimSpace(h.Where)], strings.TrimSpace(h.WhatHappened), tick).Scan(&eventID); err != nil {
			return nil, fmt.Errorf("writeHistory: event %d: %w", i+1, err)
		}
		eventIDs = append(eventIDs, eventID)

		if err := addParticipant(ctx, tx, eventID, ids.places[strings.TrimSpace(h.Where)], "location", "setting"); err != nil {
			return nil, fmt.Errorf("writeHistory: event %d setting: %w", i+1, err)
		}
		for _, who := range h.Who {
			actorID, ok := ids.cast[strings.TrimSpace(who)]
			if !ok {
				continue
			}
			if err := addParticipant(ctx, tx, eventID, actorID, "actor", "subject"); err != nil {
				return nil, fmt.Errorf("writeHistory: event %d participant %q: %w", i+1, who, err)
			}
		}

		// The knowledge this event left. Subject-linked to the people it was about and the place it
		// happened, which is what makes those things EXIST for the holder (fn_entity_visible).
		for _, k := range h.Knowledge {
			holderID, ok := ids.cast[strings.TrimSpace(k.Holder)]
			if !ok {
				continue
			}
			subjects := []string{ids.places[strings.TrimSpace(h.Where)]}
			for _, who := range h.Who {
				if id, ok := ids.cast[strings.TrimSpace(who)]; ok {
					subjects = append(subjects, id)
				}
			}
			if err := writePerception(ctx, tx, worldID, holderID, eventID,
				strings.TrimSpace(k.Content), k.EpistemicType, tick, subjects); err != nil {
				return nil, fmt.Errorf("writeHistory: event %d knowledge for %q: %w", i+1, k.Holder, err)
			}
		}
	}
	return eventIDs, nil
}

// writeMinds writes personality_core and trait_provenance for the cast — and for nobody else.
//
// THE PLAYER GETS NO CORE. Not an empty one, not a default one: no row. B-4 is absolute, the template
// says it in as many words ("Kade gets NO core (premise, not a mind)"), and a generated world does not get
// to be the exception. Every trait is grounded in an event (D-11): the first authored moment the person
// took part in, falling back to the world's opening moment when they took part in none.
func writeMinds(ctx context.Context, tx pgx.Tx, worldID string, doc *genesisDoc, ids *genesisIDs,
	historyEventIDs []string) error {

	if len(historyEventIDs) == 0 {
		return fmt.Errorf("writeMinds: no history events to ground minds in")
	}

	// First moment each person appears in, so a trait or a secret traces to something that happened to
	// them. The TICK travels with the id: a perception's acquired_tick may never precede its source
	// event's own tick (I-9), so grounding in event i means acquiring at event i's tick, not at the
	// ladder's base.
	type moment struct {
		eventID string
		tick    int64
	}
	groundedIn := make(map[string]moment, len(doc.Cast))
	for i, h := range doc.History {
		if i >= len(historyEventIDs) {
			break
		}
		for _, who := range h.Who {
			who = strings.TrimSpace(who)
			if _, seen := groundedIn[who]; !seen {
				groundedIn[who] = moment{eventID: historyEventIDs[i], tick: genesisBackstoryBaseTick + int64(i)}
			}
		}
	}
	opening := moment{eventID: historyEventIDs[0], tick: genesisBackstoryBaseTick}

	for _, a := range doc.Cast {
		name := strings.TrimSpace(a.CanonicalName)
		// The moment a trait or a secret traces back to (D-11).
		ground, ok := groundedIn[name]
		if !ok {
			ground = opening
		}
		if err := insertMind(ctx, tx, worldID, ids.cast[name], a, ground.eventID, ground.tick); err != nil {
			return err
		}
	}
	return nil
}

// insertMind writes one actor's personality_core, trait provenance and secret. Shared between the
// content commit (cast, grounded in their first authored moment) and the arrival commit (people the
// identity referenced into existence, grounded in the arrival event — the moment that made them real).
func insertMind(ctx context.Context, tx pgx.Tx, worldID, actorID string, a genesisActor,
	groundEventID string, groundTick int64) error {

	name := strings.TrimSpace(a.CanonicalName)
	// traits/1: real traits are objects {value, manner}; schema_version and speech_manner are
	// strings at the top level. Shape copied from the template because the cognition seats read it.
	traits := map[string]any{
		"schema_version": "traits/1",
		"speech_manner":  strings.TrimSpace(a.SpeechManner),
	}
	for _, t := range a.Traits {
		traits[traitKey(t.Key)] = map[string]any{
			"value":  strengthValue(t.Strength),
			"manner": strings.TrimSpace(t.Manner),
		}
	}
	traitsJSON, err := json.Marshal(traits)
	if err != nil {
		return fmt.Errorf("insertMind: %q traits: %w", name, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO personality_core (world_id, actor_id, traits, malleability)
		 VALUES ($1::uuid, $2::uuid, $3::jsonb, $4)`,
		worldID, actorID, string(traitsJSON), malleabilityValue(a.Malleability)); err != nil {
		return fmt.Errorf("insertMind: %q core: %w", name, err)
	}

	for _, t := range a.Traits {
		if _, err := tx.Exec(ctx,
			`INSERT INTO trait_provenance (world_id, actor_id, trait_key, event_id)
			 VALUES ($1::uuid, $2::uuid, $3, $4::uuid) ON CONFLICT DO NOTHING`,
			worldID, actorID, traitKey(t.Key), groundEventID); err != nil {
			return fmt.Errorf("insertMind: %q trait %q provenance: %w", name, t.Key, err)
		}
	}

	// The secret. One private perception, held by one person, about themselves — which is what makes
	// it invisible to everyone else through fn_visible_perceptions, and what the engine's own
	// planted-secret test (I-3) goes looking for leaks of.
	//
	// IT MUST NOT HANG OFF THE world_genesis EVENT, and this cost a live build to learn. In this
	// engine `world_genesis` is not a general-purpose "before play" event: `fn_perceived_name` treats
	// EVERY perception sourced from one and subject-linked to an entity as that entity's NAME. A
	// secret parked there is read straight back as the holder's name — the archivist's compendium
	// entry rendered her forgery scheme where her name belonged. Grounding it in the moment that
	// caused it is both the fix and the more honest provenance (B-2: knowledge arrives by a path).
	return writePerception(ctx, tx, worldID, actorID, groundEventID,
		strings.TrimSpace(a.Hiding), "direct", groundTick, []string{actorID})
}

// traitKey normalises an authored disposition word into the key shape the template uses: lowercase,
// underscores, no spaces. personality_core keys are read by prompt assembly, and a key with a space in it
// reads as two words to whatever consumes it.
func traitKey(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	k = strings.ReplaceAll(k, " ", "_")
	k = strings.ReplaceAll(k, "-", "_")
	return k
}

// writeOpeningState writes the whole opening world as state_mutation rows under ONE scene-genesis event,
// ordered by beat_seq — the template's shape, and the reason it works: every (entity, path) pair is
// written exactly once, so replaying the events reproduces the same state (I-1).
//
// This is where the engine draws what the model was forbidden to: the region's footprint from its extent
// class via fn_extent_class_metres + fn_area_around, and every place's coordinates from a deterministic
// ring inside that footprint. The model said "small"; the engine says how many metres that is.
func writeOpeningState(ctx context.Context, tx pgx.Tx, worldID string, doc *genesisDoc, ids *genesisIDs) error {
	var eventID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, scene_id, event_type, summary, in_world_tick, in_world_label,
		                          status, accepted_at, origin)
		 VALUES ($1::uuid, $2::uuid, 'AttributeChanged', $3, $4, 'Scene', 'accepted', now(), 'fast_path')
		 RETURNING event_id::text`,
		worldID, ids.places[strings.TrimSpace(doc.Arrival.Place)],
		"the world as the visitor found it", genesisSceneTick).Scan(&eventID); err != nil {
		return fmt.Errorf("writeOpeningState: event: %w", err)
	}

	seq := 0
	set := func(entityID, kind, path string, value any) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("writeOpeningState: marshal %s: %w", path, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
			                             new_value, valid_from_tick, valid_from_seq)
			 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::jsonb, $7, $8)`,
			worldID, eventID, entityID, kind, path, string(raw), genesisSceneTick, seq); err != nil {
			return fmt.Errorf("writeOpeningState: %s on %s: %w", path, entityID, err)
		}
		seq++
		return nil
	}

	// The region: centre of the world's coordinate space, with a footprint the engine sizes.
	radius, err := genesisExtentRadius(ctx, tx, worldID, doc.Region.ExtentClass)
	if err != nil {
		return err
	}
	centre := map[string]float64{"x": 0, "y": 0}
	area, err := genesisAreaAround(ctx, tx, centre, radius)
	if err != nil {
		return err
	}
	if err := set(ids.region, "location", "attrs.coordinates", centre); err != nil {
		return err
	}
	if err := set(ids.region, "location", "attrs.area", area); err != nil {
		return err
	}
	if err := set(ids.region, "location", "attrs.description", strings.TrimSpace(doc.Region.Descriptor)); err != nil {
		return err
	}
	// A region with no tension reads as 'none' ⇒ an infinite beat budget, which is the exact condition
	// that made the Journey unreachable before SPEC-030. Every location gets stamped, parents included.
	if err := set(ids.region, "location", "attrs.tension", "calm"); err != nil {
		return err
	}

	// The rooms, on a ring inside the region. Deterministic from position so the same document always
	// produces the same map — a build is reproducible even though the fiction that fed it was not.
	placeCoords := genesisPlaceCoords(doc.Places, radius)
	for _, p := range doc.Places {
		name := strings.TrimSpace(p.CanonicalName)
		id := ids.places[name]
		coord := placeCoords[name]
		if err := set(id, "location", "attrs.parent_location_id", ids.region); err != nil {
			return err
		}
		if err := set(id, "location", "attrs.coordinates", coord); err != nil {
			return err
		}
		if err := set(id, "location", "attrs.description", strings.TrimSpace(p.Description)); err != nil {
			return err
		}
		if err := set(id, "location", "attrs.tension", p.Tension); err != nil {
			return err
		}
	}

	// The cast: where they stand, and the descriptor that is all a stranger sees. Without the descriptor
	// fn_display_name falls through to the canonical name, which is a naming-wall breach by default.
	for _, a := range doc.Cast {
		id := ids.cast[strings.TrimSpace(a.CanonicalName)]
		where := strings.TrimSpace(a.StartsIn)
		if err := set(id, "actor", "attrs.location_id", ids.places[where]); err != nil {
			return err
		}
		if err := set(id, "actor", "attrs.coordinates", placeCoords[where]); err != nil {
			return err
		}
		if err := set(id, "actor", "attrs.descriptor", strings.TrimSpace(a.Descriptor)); err != nil {
			return err
		}
		if err := set(id, "actor", "attrs.max_load", genesisPlayerMaxLoad); err != nil {
			return err
		}
	}

	// The objects.
	for _, o := range doc.Objects {
		id := ids.things[strings.TrimSpace(o.CanonicalName)]
		if err := set(id, "artifact", "attrs.descriptor", strings.TrimSpace(o.Descriptor)); err != nil {
			return err
		}
		if err := set(id, "artifact", "attrs.kind", strings.TrimSpace(o.Kind)); err != nil {
			return err
		}
		switch {
		case strings.TrimSpace(o.Where.InPlace) != "":
			where := strings.TrimSpace(o.Where.InPlace)
			if err := set(id, "artifact", "attrs.location_id", ids.places[where]); err != nil {
				return err
			}
			if err := set(id, "artifact", "attrs.coordinates", placeCoords[where]); err != nil {
				return err
			}
		default:
			holder := strings.TrimSpace(o.Where.CarriedBy)
			if err := set(id, "artifact", "attrs.contained_by", ids.cast[holder]); err != nil {
				return err
			}
		}
	}

	// The ways. A portal is an artifact whose attrs.connects holds both room ids; fn_portal_permits reads
	// open and locked off it, and fn_actor_move_permitted refuses a move that no portal allows.
	for _, w := range doc.Ways {
		id := ids.things[wayKey(w)]
		from, to := strings.TrimSpace(w.FromPlace), strings.TrimSpace(w.ToPlace)
		open, locked := wayFlags(w.State)
		if err := set(id, "artifact", "attrs.descriptor", strings.TrimSpace(w.Descriptor)); err != nil {
			return err
		}
		if err := set(id, "artifact", "attrs.connects", []string{ids.places[from], ids.places[to]}); err != nil {
			return err
		}
		if err := set(id, "artifact", "attrs.open", open); err != nil {
			return err
		}
		if err := set(id, "artifact", "attrs.locked", locked); err != nil {
			return err
		}
		if err := set(id, "artifact", "attrs.location_id", ids.places[from]); err != nil {
			return err
		}
		if err := set(id, "artifact", "attrs.coordinates", placeCoords[from]); err != nil {
			return err
		}
	}
	return nil
}

// writeArrival makes the world's opening moment real: one ActorMoved event, the player's state, and
// the player's whole epistemic state — one perception of their own arrival. No roster of who is in
// the room, because that would fake a fan-out they never received. Everyone present is visible to
// them situationally, through co-location; nobody is KNOWN to them (kin excepted, written by the
// caller), so every compendium page 404s and every name renders as a descriptor.
//
// newCast are placed here too, under the same event and tick: they enter the world's record at the
// moment the premise that references them does. Returns the arrival event id so the caller can
// ground their minds in it.
func writeArrival(ctx context.Context, tx pgx.Tx, worldID string, doc *genesisDoc, ids *genesisIDs,
	newCast []genesisActor) (string, error) {
	placeID := ids.places[strings.TrimSpace(doc.Arrival.Place)]

	var eventID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO canon_event (world_id, scene_id, event_type, summary, in_world_tick, in_world_label,
		                          status, accepted_at, origin)
		 VALUES ($1::uuid, $2::uuid, 'ActorMoved', $3, $4, 'Arrival', 'accepted', now(), 'fast_path')
		 RETURNING event_id::text`,
		worldID, placeID, strings.TrimSpace(doc.Arrival.Stated), genesisArrivalTick).Scan(&eventID); err != nil {
		return "", fmt.Errorf("writeArrival: event: %w", err)
	}
	if err := addParticipant(ctx, tx, eventID, ids.player, "actor", "instigator"); err != nil {
		return "", fmt.Errorf("writeArrival: instigator: %w", err)
	}
	if err := addParticipant(ctx, tx, eventID, placeID, "location", "setting"); err != nil {
		return "", fmt.Errorf("writeArrival: setting: %w", err)
	}

	radius, err := genesisExtentRadius(ctx, tx, worldID, doc.Region.ExtentClass)
	if err != nil {
		return "", err
	}
	placeCoords := genesisPlaceCoords(doc.Places, radius)
	coord := placeCoords[strings.TrimSpace(doc.Arrival.Place)]

	seq := 0
	set := func(entityID, path string, value any) error {
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("writeArrival: marshal %s: %w", path, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
			                             new_value, valid_from_tick, valid_from_seq)
			 VALUES ($1::uuid, $2::uuid, $3::uuid, 'actor', $4, $5::jsonb, $6, $7)`,
			worldID, eventID, entityID, path, string(raw), genesisArrivalTick, seq); err != nil {
			return fmt.Errorf("writeArrival: %s: %w", path, err)
		}
		seq++
		return nil
	}
	if err := set(ids.player, "attrs.location_id", placeID); err != nil {
		return "", err
	}
	if err := set(ids.player, "attrs.descriptor", strings.TrimSpace(doc.Arrival.Descriptor)); err != nil {
		return "", err
	}
	if err := set(ids.player, "attrs.coordinates", coord); err != nil {
		return "", err
	}
	if err := set(ids.player, "attrs.max_load", genesisPlayerMaxLoad); err != nil {
		return "", err
	}

	// The referenced people, placed exactly as the opening state placed the cast — same paths, same
	// shape — just at the arrival tick, under the arrival event.
	for _, a := range newCast {
		id := ids.cast[strings.TrimSpace(a.CanonicalName)]
		where := strings.TrimSpace(a.StartsIn)
		if err := set(id, "attrs.location_id", ids.places[where]); err != nil {
			return "", err
		}
		if err := set(id, "attrs.coordinates", placeCoords[where]); err != nil {
			return "", err
		}
		if err := set(id, "attrs.descriptor", strings.TrimSpace(a.Descriptor)); err != nil {
			return "", err
		}
		if err := set(id, "attrs.max_load", genesisPlayerMaxLoad); err != nil {
			return "", err
		}
	}

	if err := writePerception(ctx, tx, worldID, ids.player, eventID,
		strings.TrimSpace(doc.Arrival.Stated), "direct", genesisArrivalTick,
		[]string{ids.player, placeID}); err != nil {
		return "", err
	}
	return eventID, nil
}

// writePerception writes one perception_record and its subject links. acquired_tick = valid_tick = the
// source event's tick, which is how I-9 ("you cannot learn it before it happened") holds by construction
// rather than by checking.
func writePerception(ctx context.Context, tx pgx.Tx, worldID, holderID, sourceEventID, content,
	epistemicType string, tick int64, subjects []string) error {

	var perceptionID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
		                                acquired_tick, valid_tick)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $6) RETURNING perception_id::text`,
		worldID, holderID, sourceEventID, content, epistemicType, tick).Scan(&perceptionID); err != nil {
		return fmt.Errorf("writePerception: record: %w", err)
	}
	seen := make(map[string]bool, len(subjects))
	for _, subjectID := range subjects {
		if subjectID == "" || seen[subjectID] {
			continue
		}
		seen[subjectID] = true
		if _, err := tx.Exec(ctx,
			`INSERT INTO perception_subject (perception_id, entity_id, world_id)
			 VALUES ($1::uuid, $2::uuid, $3::uuid) ON CONFLICT DO NOTHING`,
			perceptionID, subjectID, worldID); err != nil {
			return fmt.Errorf("writePerception: subject %s: %w", subjectID, err)
		}
	}
	return nil
}

func addParticipant(ctx context.Context, tx pgx.Tx, eventID, entityID, kind, role string) error {
	if entityID == "" {
		return nil
	}
	_, err := tx.Exec(ctx,
		`INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
		 VALUES ($1::uuid, $2::uuid, $3, $4) ON CONFLICT DO NOTHING`,
		eventID, entityID, kind, role)
	return err
}

// genesisExtentRadius asks the world's own physics what a size class means in metres. Per-world by
// design (extent_class_metres is keyed by world_id), so a world may later scale differently without any
// of this code knowing a number.
func genesisExtentRadius(ctx context.Context, tx pgx.Tx, worldID, class string) (float64, error) {
	var radius float64
	if err := tx.QueryRow(ctx, `SELECT fn_extent_class_metres($1::uuid, $2)`, worldID, class).Scan(&radius); err != nil {
		return 0, fmt.Errorf("genesisExtentRadius: %q: %w", class, err)
	}
	if radius <= 0 {
		return 0, fmt.Errorf("genesisExtentRadius: %q resolved to %v metres", class, radius)
	}
	return radius, nil
}

func genesisAreaAround(ctx context.Context, tx pgx.Tx, centre map[string]float64, radius float64) (json.RawMessage, error) {
	raw, err := json.Marshal(centre)
	if err != nil {
		return nil, fmt.Errorf("genesisAreaAround: centre: %w", err)
	}
	var area []byte
	if err := tx.QueryRow(ctx, `SELECT fn_area_around($1::jsonb, $2)`, string(raw), radius).Scan(&area); err != nil {
		return nil, fmt.Errorf("genesisAreaAround: %w", err)
	}
	return json.RawMessage(area), nil
}

// genesisPlaceCoords lays the rooms out on a ring inside the region, evenly spaced, in authored order.
//
// A ring rather than a grid because what matters mechanically is DISTANCE: travel time is distance/speed,
// and a beat's budget comes from the origin's tension, so rooms need to be far enough apart that leaving
// one can exceed a beat and become a journey (the SPEC-030 lesson — before it, every destination fitted
// inside one beat and the journey machinery was unreachable). Evenly spaced on a ring at 0.6 of the
// region's radius gives neighbours that are close and opposite rooms that are genuinely far, from nothing
// but the size class the seat chose.
func genesisPlaceCoords(places []genesisPlace, regionRadius float64) map[string]map[string]float64 {
	coords := make(map[string]map[string]float64, len(places))
	n := len(places)
	if n == 0 {
		return coords
	}
	ring := regionRadius * 0.6
	for i, p := range places {
		angle := 2 * math.Pi * float64(i) / float64(n)
		coords[strings.TrimSpace(p.CanonicalName)] = map[string]float64{
			"x": math.Round(ring * math.Cos(angle)),
			"y": math.Round(ring * math.Sin(angle)),
		}
	}
	return coords
}

// genesisAccent derives the theme's accent colour from the world's name.
//
// The seat is not asked for it: a hex triplet is a number, and the schema forbids numbers. Deriving it
// from the name keeps the same world looking the same every time it is built, and keeps this from being a
// palette the service has opinions about (GA-3 — a colour taxonomy is a genre taxonomy wearing a hat).
// The hues are spread around the wheel; saturation and lightness are fixed so every accent is legible
// against the dark surfaces the frontend actually uses.
func genesisAccent(displayName string) string {
	var sum uint32
	for _, r := range displayName {
		sum = sum*31 + uint32(r)
	}
	palette := []string{
		"#c9a227", "#7a8b99", "#a2543f", "#5f7a6b", "#8a6fa0",
		"#b5763c", "#4f6d8c", "#9c5a6b", "#6b8f5a", "#a08a5f",
	}
	return palette[sum%uint32(len(palette))]
}
