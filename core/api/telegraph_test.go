package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Task 5 — Telegraph: the wind-up commits, the outcome holds, the beat ends (RULINGS-2026-07-24
// §1, §3). Each test mints a FRESH, RANDOM world so the world-first lookups and the canon counts
// see ONLY these rows (the DB is not reset between `go test` runs).

// telegraphIDs holds the freshly-minted ids for one test invocation.
type telegraphIDs struct{ World, P, J, L, L2 string }

// setupTelegraphWorld mints a fresh world + player P + NPC Jonas + two locations, and co-locates
// P and J at L (moves projected into actor_state by the state_mutation trigger). A brand-new world
// every invocation → hermetic and re-runnable with no DB reset.
func setupTelegraphWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool) telegraphIDs {
	t.Helper()
	var id telegraphIDs
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.P, &id.J, &id.L, &id.L2); err != nil {
		t.Fatalf("mint ids: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($1,$5,'actor','Player'),
		 ($2,$5,'actor','Jonas'),
		 ($3,$5,'location','The Drowned Lantern'),
		 ($4,$5,'location','The Bar')`,
		id.P, id.J, id.L, id.L2, id.World); err != nil {
		t.Fatalf("seed telegraph entities: %v", err)
	}
	// Co-locate P and J at L (each move is the actor's latest state → fn_actors_at returns both).
	for i, actor := range []string{id.P, id.J} {
		if _, err := pool.Exec(ctx, `
			WITH ev AS (
			  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
			  VALUES (gen_random_uuid(),$1,'move','telegraph-colocate',$2,0,'accepted',now(),'public','fast_path')
			  RETURNING event_id
			),
			ep AS (
			  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
			  SELECT event_id,$3,'actor','instigator' FROM ev
			)
			INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
			SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($4::text),$2,0 FROM ev`,
			id.World, int64(100+i), actor, id.L); err != nil {
			t.Fatalf("colocate %s: %v", actor, err)
		}
	}
	return id
}

func telegraphBaseTick(t *testing.T, ctx context.Context, pool *pgxpool.Pool, world string) int64 {
	t.Helper()
	var bt int64
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+100`,
		world).Scan(&bt); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	return bt
}

// TestTelegraphEndsBeat: the world-first seat telegraphs Jonas's cut-in on the player's FIRST
// attempt of a two-attempt chain. The wind-up commits as canon, a held_outcome row holds Jonas's
// full intended act, the beat ends, and neither of the player's attempts runs (the chain is
// discarded — the world seized the moment first).
func TestTelegraphEndsBeat(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupTelegraphWorld(t, ctx, pool)
	baseTick := telegraphBaseTick(t, ctx, pool, id.World)

	const windUp = "Jonas pushes off the bar, moving to cut in"

	// Batch cognition: Jonas TELEGRAPHS a full ActorMoved (his real intended act) — disruptive.
	batch := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.J +
		`","decision":{"commit_kind":"telegraph","attempt":{"type":"ActorMoved","stated":"` + windUp +
		`","to_location_id":"` + id.L2 + `"}}}]`}
	isolated := &scriptedCognitionDriver{name: "scripted-isolated", body: `[]`}
	resolve := &countingResolveDriver{inner: NewFakeResolveDriver()}
	orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated, WorldActor: NewFakeWorldActorDriver()}

	// The player's two-attempt chain: cross to the bar, then slip Jonas the note. World-first fires
	// on the FIRST attempt and ends the beat before either step resolves.
	chain := []Attempt{
		{Type: "ActorMoved", Stated: "I cross to the bar", ToLocationID: id.L2},
		{Type: "Communicated", Stated: "I slip him the note", ListenerID: id.J, Content: "here"},
	}
	outcome, err := orc.RunBeat(ctx, id.World, id.P, chain, baseTick)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}

	// (a) the beat ended on the telegraph, and Telegraphs surfaced the stated wind-up.
	if outcome.HaltReason != "telegraph" {
		t.Fatalf("HaltReason = %q, want telegraph", outcome.HaltReason)
	}
	if len(outcome.Telegraphs) != 1 || outcome.Telegraphs[0] != windUp {
		t.Fatalf("Telegraphs = %v, want [%q]", outcome.Telegraphs, windUp)
	}

	// (b) a canon AttributeChanged with origin='telegraph' exists; summary = the stated wind-up.
	var evID, summary, etype string
	if err := pool.QueryRow(ctx,
		`SELECT event_id::text, summary, event_type FROM canon_event WHERE world_id=$1 AND origin='telegraph'`,
		id.World).Scan(&evID, &summary, &etype); err != nil {
		t.Fatalf("read telegraph canon event: %v", err)
	}
	if etype != "AttributeChanged" {
		t.Fatalf("wind-up event_type = %q, want AttributeChanged", etype)
	}
	if summary != windUp {
		t.Fatalf("wind-up summary = %q, want %q (canon summary = truth)", summary, windUp)
	}

	// (c) co-located actors (the player, and Jonas himself) hold perceptions of the wind-up, each
	//     carrying perception_subject rows (about-ness written by the engine, not backfilled).
	var playerPerceptions int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM perception_record WHERE source_event_id=$1::uuid AND holder_id=$2::uuid`,
		evID, id.P).Scan(&playerPerceptions); err != nil {
		t.Fatalf("read player perception: %v", err)
	}
	if playerPerceptions < 1 {
		t.Fatalf("player perceptions of wind-up = %d, want >=1 (co-located witness)", playerPerceptions)
	}
	var subjectRows int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM perception_subject ps
		 JOIN perception_record pr ON pr.perception_id = ps.perception_id
		 WHERE pr.source_event_id=$1::uuid`,
		evID).Scan(&subjectRows); err != nil {
		t.Fatalf("read perception_subject rows: %v", err)
	}
	if subjectRows < 1 {
		t.Fatalf("perception_subject rows for wind-up = %d, want >=1 (about-ness)", subjectRows)
	}

	// (d) exactly one held_outcome row, status pending, whose attempt round-trips to Jonas's
	//     telegraphed act (his full ActorMoved is preserved for the reaction beat to resolve).
	var heldCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM held_outcome WHERE world_id=$1`, id.World).Scan(&heldCount); err != nil {
		t.Fatalf("count held_outcome: %v", err)
	}
	if heldCount != 1 {
		t.Fatalf("held_outcome rows = %d, want exactly 1", heldCount)
	}
	var status, heldActor, attemptJSON string
	if err := pool.QueryRow(ctx,
		`SELECT status, actor_id::text, attempt::text FROM held_outcome WHERE world_id=$1`,
		id.World).Scan(&status, &heldActor, &attemptJSON); err != nil {
		t.Fatalf("read held_outcome: %v", err)
	}
	if status != "pending" {
		t.Fatalf("held_outcome.status = %q, want pending", status)
	}
	if heldActor != id.J {
		t.Fatalf("held_outcome.actor_id = %q, want Jonas %q", heldActor, id.J)
	}
	var gotAttempt Attempt
	if err := json.Unmarshal([]byte(attemptJSON), &gotAttempt); err != nil {
		t.Fatalf("held attempt not valid Attempt JSON: %v", err)
	}
	wantAttempt := Attempt{Type: "ActorMoved", Stated: windUp, ToLocationID: id.L2}
	gotJSON, _ := json.Marshal(gotAttempt)
	wantJSON, _ := json.Marshal(wantAttempt)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("held attempt = %s, want %s (must round-trip to the telegraphed act)", gotJSON, wantJSON)
	}

	// (e) the chain was DISCARDED: neither of the player's attempts committed. Player attempts
	//     commit via apply_event with origin='freeform', so zero 'freeform' canon events proves it.
	var freeform int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin='freeform'`,
		id.World).Scan(&freeform); err != nil {
		t.Fatalf("count freeform canon: %v", err)
	}
	if freeform != 0 {
		t.Fatalf("freeform canon events = %d, want 0 (both player attempts discarded)", freeform)
	}

	// The wind-up went straight through apply_ruled_event (p_origin='telegraph') — it never touches
	// the resolve seat.
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (telegraph commits via apply_ruled_event, not resolve)", resolve.calls)
	}
}

// TestTelegraphPremiseGeneralized_EntityDestroyed: the generalized premise re-check now covers
// EntityDestroyed. When the target the player named is no longer in entity_registry — a prior
// world commit destroyed it — the chain halts premise_broken (the old two-case switch never
// checked EntityDestroyed targets at all, and would have carried on to adjudication).
func TestTelegraphPremiseGeneralized_EntityDestroyed(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupTelegraphWorld(t, ctx, pool)
	baseTick := telegraphBaseTick(t, ctx, pool, id.World)

	// The world-first seat does nothing this moment; the world simply changed earlier (the target
	// was destroyed), so the premise no longer holds when the player's step is re-checked.
	batch := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.J + `","decision":"none"}]`}
	isolated := &scriptedCognitionDriver{name: "scripted-isolated", body: `[]`}
	resolve := &countingResolveDriver{inner: NewFakeResolveDriver()}
	orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated, WorldActor: NewFakeWorldActorDriver()}

	// A target uuid that is NOT registered in this world — "destroyed", no registry row remains.
	const goneTarget = "dead0000-0000-0000-0000-000000000000"
	chain := []Attempt{{Type: "EntityDestroyed", Stated: "I smash the lantern", TargetID: goneTarget}}

	outcome, err := orc.RunBeat(ctx, id.World, id.P, chain, baseTick)
	if err != nil {
		t.Fatalf("RunBeat: %v", err)
	}
	if outcome.HaltReason != "premise_broken" {
		t.Fatalf("HaltReason = %q, want premise_broken (EntityDestroyed target gone)", outcome.HaltReason)
	}
	// Nothing of the player's ran: no freeform canon events, and the resolver was never consulted.
	var freeform int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM canon_event WHERE world_id=$1 AND origin='freeform'`,
		id.World).Scan(&freeform); err != nil {
		t.Fatalf("count freeform canon: %v", err)
	}
	if freeform != 0 {
		t.Fatalf("freeform canon events = %d, want 0 (premise broke before any commit)", freeform)
	}
	if resolve.calls != 0 {
		t.Fatalf("resolve calls = %d, want 0 (premise_broken halts before routing)", resolve.calls)
	}
}
