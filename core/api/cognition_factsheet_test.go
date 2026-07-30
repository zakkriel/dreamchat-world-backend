package main

import (
	"context"
	"strings"
	"testing"
)

// TestWorldFirst_FeedsPerceivedFactSheetToCognition is the Grounded Reasoning Unit 1 → cognition proof:
// the NPC minds must reason FROM the engine's computed physics facts, PERCEPTION-SCOPED (truth_side=FALSE
// — the minds are walled, RULINGS-2026-07-23 §5/§9). It seeds a fresh world where the player is about to
// act on a CLOSED crate co-located with two NPCs, drives worldFirst through the real orchestrator path,
// and asserts:
//
//	(1) the captured cognition prompt carries a COMPUTED FACTS block with the engine's real distance;
//	(2) the closed crate's contents (the stone id) are ABSENT from BOTH perceived prompts (the wall
//	    withholds them), though the truth-side sheet WOULD show them (so the withholding is meaningful);
//	(3) the cognition.txt rulebook ships the reason-from-computed-facts rule (marker reaches the header);
//	(4) VIEWER CHOICE: the batch sheet is computed for the acting PLAYER (public spatial view), the
//	    isolated sheet for the NPC herself — the two sheets differ and each rides its own prompt.
//
// Hermetic (fresh random world, no seed dependency), following the wall_test / resolve_factsheet capturing
// pattern. The scripted cognition drivers return "[]" (every mind decides "none") so nothing commits.
func TestWorldFirst_FeedsPerceivedFactSheetToCognition(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Fresh world + cast: location L; player K (acting viewer); NPC J (holds nothing → shared BATCH);
	// NPC M (holds a PRIVATE record about K → pulled ISOLATED); a CLOSED crate holding a stone.
	var world, loc, k, j, m, crate, stone string
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&world, &loc, &k, &j, &m, &crate, &stone); err != nil {
		t.Fatalf("mint ids: %v", err)
	}

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v\n%s", err, sql)
		}
	}
	mustExec(`INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		($1,$7,'location','The Common Room'),($2,$7,'actor','Kade'),($3,$7,'actor','Jonas'),
		($4,$7,'actor','Mara'),($5,$7,'artifact','crate'),($6,$7,'artifact','stone')`,
		loc, k, j, m, crate, stone, world)
	mustExec(`INSERT INTO location_state (entity_id, world_id, attrs)
		VALUES ($1,$2,'{"coordinates":{"x":0,"y":0},"extent":{"w":2000,"h":2000}}'::jsonb)`, loc, world)
	mustExec(`INSERT INTO movement_type (world_id, movement_type_id, base_speed_mps) VALUES ($1,'walk',1.4)`, world)
	// K at {0,0}, M at {2,0}, J at {5,0} — all co-located at L (fn_actors_at reads attrs.location_id).
	mustExec(`INSERT INTO actor_state (entity_id, world_id, attrs) VALUES
		($1,$2, jsonb_build_object('location_id',$3::text,'coordinates',jsonb_build_object('x',0,'y',0),'max_load',40)),
		($4,$2, jsonb_build_object('location_id',$3::text,'coordinates',jsonb_build_object('x',2,'y',0),'max_load',40)),
		($5,$2, jsonb_build_object('location_id',$3::text,'coordinates',jsonb_build_object('x',5,'y',0),'max_load',40))`,
		k, world, loc, m, j)
	// The CLOSED crate at {1,0} (max_room ⇒ a container; open=false ⇒ perceived-side contents withheld),
	// holding the stone (contained_by = crate).
	mustExec(`INSERT INTO artifact_state (entity_id, world_id, attrs) VALUES
		($1,$3, jsonb_build_object('location_id',$4::text,'coordinates',jsonb_build_object('x',1,'y',0),'max_room',4,'open',false)),
		($2,$3, jsonb_build_object('contained_by',$1::text))`,
		crate, stone, world, loc)

	// M's private record about K (subject K) → fn_isolated_npcs flags her ISOLATED; J holds nothing → BATCH.
	var eSec, mPid string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&eSec); err != nil {
		t.Fatalf("mint secret event: %v", err)
	}
	mustExec(`INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		VALUES ($1,$2,'observation','the secret M alone saw',90,0,'accepted',now(),'private','fast_path')`, eSec, world)
	if err := pool.QueryRow(ctx,
		`INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
		 VALUES ($1,$2,$3,'the ledger names the smuggler','direct',90,90) RETURNING perception_id`,
		world, m, eSec).Scan(&mPid); err != nil {
		t.Fatalf("seed secret perception: %v", err)
	}
	mustExec(`INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES ($1,$2,$3)`, mPid, k, world)

	// The imminent attempt: K acts on the CLOSED crate. Bound ids = {crate}; +player = actionIDs [crate,K]
	// (worldFirst appends the player after collectParticipantIDs — this array + order mirrors it exactly).
	attempt := Attempt{Type: "AttributeChanged", Stated: "I pry at the crate", TargetID: crate}

	// Ground truth: the exact sheets worldFirst must embed. Batch viewer = the acting PLAYER K (public
	// spatial view); isolated viewer = the NPC M herself (her own read). truthSheet proves the crate's
	// contents exist truth-side, so their perceived-side absence below is a real withholding, not vacuous.
	var wantBatchSheet, wantIsoSheet, truthSheet, batchDist string
	if err := pool.QueryRow(ctx, `
		SELECT fn_fact_sheet($1::uuid,$2::uuid,ARRAY[$4,$2]::uuid[],false)::text,
		       fn_fact_sheet($1::uuid,$3::uuid,ARRAY[$4,$2]::uuid[],false)::text,
		       fn_fact_sheet($1::uuid,$2::uuid,ARRAY[$4,$2]::uuid[],true)::text,
		       (fn_fact_sheet($1::uuid,$2::uuid,ARRAY[$4,$2]::uuid[],false)->'targets'->0->>'distance_m')`,
		world, k, m, crate).Scan(&wantBatchSheet, &wantIsoSheet, &truthSheet, &batchDist); err != nil {
		t.Fatalf("compute expected sheets: %v", err)
	}
	// Fixture sanity: real geometry (K→crate ~1 m), the two perceived sheets DIFFER (different viewers),
	// and the truth-side sheet DOES carry the stone id.
	if !strings.HasPrefix(batchDist, "1") {
		t.Fatalf("fixture sanity: expected K→crate ~1 m, got distance=%q", batchDist)
	}
	if wantBatchSheet == wantIsoSheet {
		t.Fatalf("fixture sanity: batch (viewer K) and isolated (viewer M) sheets must differ")
	}
	if !strings.Contains(truthSheet, stone) {
		t.Fatalf("fixture sanity: truth-side sheet must carry the stone id (else the withholding assertion is vacuous)")
	}

	// Drive worldFirst with capturing cognition drivers (every mind decides "none" → nothing commits).
	batch := &scriptedCognitionDriver{name: "scripted-batch", body: "[]"}
	isolated := &scriptedCognitionDriver{name: "scripted-isolated", body: "[]"}
	orc := &Orchestrator{DB: pool, CognitionBatch: batch, CognitionIsolated: isolated, WorldActor: NewFakeWorldActorDriver()}

	if _, err := orc.worldFirst(ctx, world, k, attempt, 80000, 0); err != nil {
		t.Fatalf("worldFirst: %v", err)
	}
	if len(batch.prompts) == 0 {
		t.Fatalf("batch cognition never fired — Jonas should sit in the shared batch")
	}
	if len(isolated.prompts) == 0 {
		t.Fatalf("isolated cognition never fired — Mara should be pulled isolated by her secret")
	}
	batchPrompt := batch.prompts[0]
	isoPrompt := isolated.prompts[0]

	// (1) the COMPUTED FACTS block reaches the batch mind, carrying the engine's real distance.
	if !strings.Contains(batchPrompt, "COMPUTED FACTS (engine-computed truth about this moment") {
		t.Fatalf("batch cognition prompt missing the COMPUTED FACTS block — the mind still guesses:\n%s", batchPrompt)
	}
	if !strings.Contains(batchPrompt, batchDist) {
		t.Fatalf("batch prompt missing the computed distance %q — the mind is not reasoning from the engine's math:\n%s", batchDist, batchPrompt)
	}

	// (2) the CLOSED crate's contents (the stone id) are WITHHELD from BOTH perceived prompts (the wall),
	//     even though the truth-side sheet would show them (asserted above).
	if strings.Contains(batchPrompt, stone) {
		t.Fatalf("PERCEIVED GATE BREACH — batch prompt leaked the closed crate's contents (stone %s):\n%s", stone, batchPrompt)
	}
	if strings.Contains(isoPrompt, stone) {
		t.Fatalf("PERCEIVED GATE BREACH — isolated prompt leaked the closed crate's contents (stone %s):\n%s", stone, isoPrompt)
	}

	// (3) the cognition.txt rulebook ships the reason-from-computed-facts rule (the marker reaches header).
	if !strings.Contains(cognitionSystemHeader, cognitionFactsRuleMarker) {
		t.Fatalf("cognition.txt missing the reason-from-computed-facts rule marker %q — the rule never reaches the minds", cognitionFactsRuleMarker)
	}

	// (4) VIEWER CHOICE: the batch prompt carries the PLAYER-viewer sheet; the isolated prompt carries the
	//     NPC-viewer sheet. Each rides its own prompt and they are not swapped.
	if !strings.Contains(batchPrompt, wantBatchSheet) {
		t.Fatalf("batch prompt missing the PLAYER-viewer fact sheet (batch = player-viewer per spec).\nwant:\n%s\n\ngot:\n%s", wantBatchSheet, batchPrompt)
	}
	if !strings.Contains(isoPrompt, wantIsoSheet) {
		t.Fatalf("isolated prompt missing the NPC-viewer fact sheet (isolated = her own perception).\nwant:\n%s\n\ngot:\n%s", wantIsoSheet, isoPrompt)
	}
	if strings.Contains(batchPrompt, wantIsoSheet) {
		t.Fatalf("batch prompt carries the NPC-viewer sheet — the viewer choice is swapped (batch must be player-viewer)")
	}
}
