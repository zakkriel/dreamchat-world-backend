package main

import (
	"context"
	"strings"
	"testing"
)

// TestAdjudicate_FeedsTruthSideFactSheet is the Grounded Reasoning Unit 1 → adjudication proof: the
// referee must reason FROM the engine's computed physics facts, not guess them ("the reasoning was
// detached from the math"). It seeds two actors co-located in ONE scene a KNOWN 2 m apart, adjudicates
// an AttributeChanged Kade→Mara through the real orchestrator path, and asserts the captured resolve
// prompt carries a FACTS-computed section with the REAL distance/duration/reachability fn_fact_sheet
// computes — and that the resolve.txt rulebook ships the reason-from-facts rule.
//
// Truth-side (RULINGS-2026-07-23 §9: the referee is truth-side, never walled). Hermetic (a fresh world
// minted here — no seed dependency), following the wall_test capturing-driver pattern. The canned
// ruling ignores the new FACTS section, so the existing ruled tests stay green (the fake driver never
// reads it).
func TestAdjudicate_FeedsTruthSideFactSheet(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Fresh world + cast: location L (one scene), actor Kade (the acting viewer), actor Mara (target).
	var world, loc, kade, mara string
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&world, &loc, &kade, &mara); err != nil {
		t.Fatalf("mint ids: %v", err)
	}

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v\n%s", err, sql)
		}
	}
	mustExec(`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		($1,$4,'location','The Common Room'),($2,$4,'actor','Kade'),($3,$4,'actor','Mara')`,
		loc, kade, mara, world)
	mustExec(`INSERT INTO location_state (entity_id, world_id, attrs)
		VALUES ($1,$2,'{"coordinates":{"x":0,"y":0},"extent":{"w":2000,"h":2000}}'::jsonb)`, loc, world)
	// Kade at {0,0}, Mara at {2,0} in the SAME scene → fn_distance = 2 m, reachable trivially (same room,
	// no portal). CEIL(2 / 1.4) = 2 s move duration. This is the design's "lean on Mara: same room,
	// reachable, ~2 m" fact sheet, computed by the F contract functions.
	mustExec(`INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		($1,$3, jsonb_build_object('location_id',$2::text,'coordinates',jsonb_build_object('x',0,'y',0),'max_load',40)),
		($4,$3, jsonb_build_object('location_id',$2::text,'coordinates',jsonb_build_object('x',2,'y',0),'max_load',40))`,
		kade, loc, world, mara)
	mustExec(`INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps) VALUES ($1,'walk',1.4)`, world)

	// Ground truth: the SAME fact sheet the referee must be handed. adjudicate builds involved ids =
	// [actor, targets] = [Kade, Mara] for a Kade→Mara AttributeChanged, so this array + order matches
	// exactly what the orchestrator embeds (the whole sheet must ride verbatim).
	var wantSheet, distStr, durStr string
	if err := pool.QueryRow(ctx, `
		SELECT fs::text, (fs->'targets'->0->>'distance_m'), (fs->'targets'->0->>'move_duration_s')
		FROM (SELECT fn_fact_sheet($1::uuid,$2::uuid,ARRAY[$2,$3]::uuid[],true) AS fs) q`,
		world, kade, mara).Scan(&wantSheet, &distStr, &durStr); err != nil {
		t.Fatalf("compute expected fact sheet: %v", err)
	}
	// Fixture sanity: the geometry must have produced a REAL non-zero distance (~2 m), or the assertions
	// below would pass vacuously against a degenerate all-zeros sheet.
	if !strings.HasPrefix(distStr, "2") || durStr != "2" {
		t.Fatalf("fixture sanity: expected distance ~2 m / duration 2 s, got distance=%q duration=%q", distStr, durStr)
	}

	// Capture the referee's prompt. The canned ruling succeeds an AttributeChanged Kade→Mara (both ids
	// in the slice), ignoring the new FACTS section entirely.
	resolve := &capturingResolveDriver{name: "capture-resolve", ruling: validRulingJSON(kade, mara, "Kade leans in close to Mara.", "Kade leans in.")}
	orc := &Orchestrator{
		DB:             pool,
		Resolve:        resolve,
		CognitionBatch: NewFakeCognitionDriver(),
		WorldActor:     NewFakeWorldActorDriver(),
	}

	if _, err := orc.adjudicate(ctx, world, []ActorAttempt{
		{ActorID: kade, Attempt: Attempt{Type: "AttributeChanged", Stated: "I lean in close to Mara", TargetID: mara}},
	}, nil, 80000, 0, "", nil); err != nil {
		t.Fatalf("adjudicate: %v", err)
	}
	if len(resolve.prompts) == 0 {
		t.Fatalf("referee was never consulted — no resolve prompt captured")
	}
	prompt := resolve.prompts[0]

	// (1) The COMPUTED FACTS section reaches the referee (renamed from "FACTS (computed by the engine …)"
	// to "COMPUTED FACTS (…)" so it is distinct from the gathered-slice "FACTS (the gathered slice)" section).
	if !strings.Contains(prompt, "COMPUTED FACTS (engine-computed") {
		t.Fatalf("resolve prompt missing the engine-computed COMPUTED FACTS section — the referee still guesses:\n%s", prompt)
	}
	// (2) It carries the REAL computed distance (2 m), not a guess — the whole point of grounded reasoning.
	if !strings.Contains(prompt, distStr) {
		t.Fatalf("resolve prompt missing the computed distance %q — the referee is not reasoning from the engine's math:\n%s", distStr, prompt)
	}
	// (3) The full truth-side fact sheet is embedded verbatim (distance + duration + reachability together).
	if !strings.Contains(prompt, wantSheet) {
		t.Fatalf("resolve prompt missing the computed fact sheet.\nwant substring:\n%s\n\ngot prompt:\n%s", wantSheet, prompt)
	}
	// (4) The resolve.txt rulebook ships the reason-from-computed-facts rule (the marker reaches the header).
	if !strings.Contains(resolveSystemHeader, resolveFactsRuleMarker) {
		t.Fatalf("resolve.txt missing the reason-from-computed-facts rule marker %q — the rule never reaches the referee", resolveFactsRuleMarker)
	}
}
