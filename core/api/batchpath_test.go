package main

import (
	"context"
	"testing"
)

// A newcomer must not isolate the whole cast.
//
// fn_isolated_npcs and fn_public_moment both call a record "shared" only when EVERY holder in the
// roster they are given holds it. Passing the full PRESENT roster put the player in that denominator,
// so any record the cast shares but the newcomer does not — most of a seeded world's history — read as
// private to every one of them. Every NPC isolated, the batch call never fired, and cognition became
// one sequential model call per NPC: measured at four per beat on the play world, ~4-6s of round trips
// the founder feels as slowness.
//
// The fixture is the exact shape: three NPCs who all hold the same record about the action, and a
// player who does not.
func TestIsolation_ANewcomerDoesNotIsolateTheWholeCast(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	var world, player, a, b, c, ev string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text, gen_random_uuid()::text,
		gen_random_uuid()::text, gen_random_uuid()::text, gen_random_uuid()::text, gen_random_uuid()::text`).
		Scan(&world, &player, &a, &b, &c, &ev); err != nil {
		t.Fatalf("mint ids: %v", err)
	}
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v\n%s", err, sql)
		}
	}
	mustExec(`INSERT INTO world (world_id, display_name) VALUES ($1,'isolation fixture')`, world)
	for _, id := range []string{player, a, b, c} {
		mustExec(`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		          VALUES ($1,$2,'actor','cast')`, id, world)
	}
	mustExec(`INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status, origin)
	          VALUES ($1,$2,'observation','the cast all saw this',10,0,'accepted','freeform')`, ev, world)
	// The three NPCs hold the SAME record, with the same content, about the player. The player does not
	// hold it — he was not there. That is the whole point: it is common knowledge among the cast.
	for _, holder := range []string{a, b, c} {
		var pid string
		if err := pool.QueryRow(ctx,
			`INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
			 VALUES ($1,$2,$3,'the cast all saw this','direct',10,10) RETURNING perception_id::text`,
			world, holder, ev).Scan(&pid); err != nil {
			t.Fatalf("insert perception: %v", err)
		}
		mustExec(`INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES ($1,$2,$3)`,
			pid, player, world)
	}

	o := &Orchestrator{DB: pool}
	npcs := []string{a, b, c}
	present := []string{player, a, b, c}
	actionIDs := []string{player}

	// THE OLD BEHAVIOUR, kept as the contrast that makes the fix legible: with the player in the
	// denominator the record is "private" to all three and the batch path is dead.
	withPlayer, err := o.isolatedNPCs(ctx, world, actionIDs, present, npcs)
	if err != nil {
		t.Fatalf("isolatedNPCs(present): %v", err)
	}
	if len(withPlayer) != 3 {
		t.Fatalf("fixture is not reproducing the defect: %d isolated with the player in the roster, want 3", len(withPlayer))
	}

	// THE FIX: shared among the minds being batched. Nobody isolates, so one batch call serves all three.
	withoutPlayer, err := o.isolatedNPCs(ctx, world, actionIDs, npcs, npcs)
	if err != nil {
		t.Fatalf("isolatedNPCs(npcs): %v", err)
	}
	if len(withoutPlayer) != 0 {
		t.Fatalf("a record every NPC holds must not isolate any of them; %d still isolated", len(withoutPlayer))
	}
}

// The narrowing must not break the wall it exists inside: a record ONE mind holds privately still
// isolates her, and only her.
func TestIsolation_AGenuineSecretStillIsolatesItsHolder(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	var world, player, a, b, ev string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text, gen_random_uuid()::text,
		gen_random_uuid()::text, gen_random_uuid()::text, gen_random_uuid()::text`).
		Scan(&world, &player, &a, &b, &ev); err != nil {
		t.Fatalf("mint ids: %v", err)
	}
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	mustExec(`INSERT INTO world (world_id, display_name) VALUES ($1,'secret fixture')`, world)
	for _, id := range []string{player, a, b} {
		mustExec(`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name)
		          VALUES ($1,$2,'actor','cast')`, id, world)
	}
	mustExec(`INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status, origin)
	          VALUES ($1,$2,'private_disclosure','only A knows',10,0,'accepted','freeform')`, ev, world)
	var pid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
		 VALUES ($1,$2,$3,'only A knows','told',10,10) RETURNING perception_id::text`, world, a, ev).Scan(&pid); err != nil {
		t.Fatalf("insert perception: %v", err)
	}
	mustExec(`INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES ($1,$2,$3)`, pid, player, world)

	o := &Orchestrator{DB: pool}
	npcs := []string{a, b}
	isolated, err := o.isolatedNPCs(ctx, world, []string{player}, npcs, npcs)
	if err != nil {
		t.Fatalf("isolatedNPCs: %v", err)
	}
	if len(isolated) != 1 || isolated[0] != a {
		t.Fatalf("only the secret's holder may isolate; got %v", isolated)
	}
}
