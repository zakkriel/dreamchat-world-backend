package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Station E — Task 4 flow tests. Each run builds a FRESH, RANDOM world (world + entity ids minted
// per invocation) so the §5 lookups see ONLY these rows: no seed perception, no other test's beat,
// and no PRIOR run of this test can flip the split (the DB is not reset between `go test` runs).
// Cast mirrors the Task 3 fixture:
//   P (player) + M (holds a PRIVATE record about P) + J (holds nothing), all co-located.
// One Communicated P→M attempt: the action's bound ids = {M, P}, so M's private-about-P record
// flags her → ISOLATED seat; J has nothing → shared BATCH. This proves the split end-to-end: one
// cognition call per NPC, each in exactly ONE seat, the wall by construction.

// flowIDs holds the freshly-minted ids for one test invocation.
type flowIDs struct{ World, P, M, J, L, Note string }

// scriptedCognitionDriver counts calls and captures prompts, returning a fixed JSON body. One
// instance per seat so batch and isolated call counts / prompts are inspected independently.
type scriptedCognitionDriver struct {
	name    string
	body    string
	calls   int
	prompts []string
}

func (d *scriptedCognitionDriver) Name() string { return d.name }
func (d *scriptedCognitionDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *scriptedCognitionDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	if req.Schema == nil {
		return "", fmt.Errorf("%s: cognition driver used without a schema", d.name)
	}
	d.calls++
	d.prompts = append(d.prompts, req.Prompt)
	return d.body, nil
}

// countingResolveDriver wraps the fake resolver and counts Generate calls — the no-bypass probe.
type countingResolveDriver struct {
	inner Driver
	calls int
}

func (d *countingResolveDriver) Name() string                { return d.inner.Name() }
func (d *countingResolveDriver) Capabilities() CapabilitySet { return d.inner.Capabilities() }
func (d *countingResolveDriver) Generate(ctx context.Context, req GenRequest) (string, error) {
	d.calls++
	return d.inner.Generate(ctx, req)
}

// setupFlowWorld mints a fresh world + cast, links M's private-about-P record, and co-locates
// P/M/J at L (moves projected into actor_state by the state_mutation trigger). A brand-new world
// every invocation → hermetic and re-runnable with no DB reset.
func setupFlowWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool) flowIDs {
	t.Helper()
	var id flowIDs
	var eSec string
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.P, &id.M, &id.J, &id.L, &id.Note, &eSec); err != nil {
		t.Fatalf("mint ids: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($1,$6,'actor','Player'),
		 ($2,$6,'actor','Mara'),
		 ($3,$6,'actor','Jonas'),
		 ($4,$6,'location','The Drowned Lantern'),
		 ($5,$6,'artifact','sealed note')`,
		id.P, id.M, id.J, id.L, id.Note, id.World); err != nil {
		t.Fatalf("seed flow entities: %v", err)
	}

	// M's private record about P: a private source event, one perception held ONLY by M, subject P.
	// Not shared by all present → private; subject P is in the action's bound ids → M is isolated.
	if _, err := pool.Exec(ctx, `
		INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq, status, accepted_at, visibility_scope, origin)
		VALUES ($1,$2,'observation','the secret M alone saw',90,0,'accepted',now(),'private','fast_path')`,
		eSec, id.World); err != nil {
		t.Fatalf("seed secret event: %v", err)
	}
	var mPid string
	if err := pool.QueryRow(ctx, `
		INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
		VALUES ($1,$2,$3,'the ledger names the smuggler','direct',90,90) RETURNING perception_id`,
		id.World, id.M, eSec).Scan(&mPid); err != nil {
		t.Fatalf("seed secret perception: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES ($1,$2,$3)`,
		mPid, id.P, id.World); err != nil {
		t.Fatalf("seed secret subject: %v", err)
	}

	// Co-locate P, M, J at L (each move is the actor's latest state → fn_actors_at returns all).
	for i, actor := range []string{id.P, id.M, id.J} {
		if _, err := pool.Exec(ctx, `
			WITH ev AS (
			  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
			  VALUES (gen_random_uuid(),$1,'move','flow-colocate',$2,0,'accepted',now(),'public','fast_path')
			  RETURNING event_id
			),
			ep AS (
			  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
			  SELECT event_id,$3,'actor','instigator' FROM ev
			)
			INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
			SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($4::text),$2,0 FROM ev`,
			id.World, int64(95+i), actor, id.L); err != nil {
			t.Fatalf("colocate %s: %v", actor, err)
		}
	}
	return id
}

func flowBaseTick(t *testing.T, ctx context.Context, pool *pgxpool.Pool, world string) int64 {
	t.Helper()
	var baseTick int64
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT max(in_world_tick) FROM canon_event WHERE world_id=$1),0)+100`,
		world).Scan(&baseTick); err != nil {
		t.Fatalf("base tick: %v", err)
	}
	return baseTick
}

// decideForTail returns the DECIDE FOR line (the prompt's closing mutable tail) so a test can
// assert exactly which ids a seat was told to speak for — without matching ids that appear in the
// roster or the imminent attempt JSON earlier in the prompt.
func decideForTail(t *testing.T, prompt string) string {
	t.Helper()
	i := strings.Index(prompt, "DECIDE FOR:")
	if i < 0 {
		t.Fatalf("prompt missing DECIDE FOR line:\n%s", prompt)
	}
	return prompt[i:]
}

func TestCognitionFlow(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	// The player's own attempt for every subtest: Communicated P→M (bound ids {M} + player = {M,P}).
	playerGreetsMara := func() []Attempt {
		return []Attempt{{Type: "Communicated", Stated: "I greet Mara", ListenerID: id.M, Content: "hello Mara"}}
	}

	// (a)+(b): the seat split, and a batch decision for the isolated NPC is rejected by the validator.
	t.Run("seat split; misbehaving batch decision rejected", func(t *testing.T) {
		baseTick := flowBaseTick(t, ctx, pool, id.World)

		// Batch MISBEHAVES: it returns a decision for Mara, who is NOT in the batch allowlist [Jonas].
		batch := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.M +
			`","decision":{"commit_kind":"commit","attempt":{"type":"Communicated","stated":"Mara blurts the secret","listener_id":"` + id.P +
			`","content":"the ledger names the smuggler"}}}]`}
		isolated := &scriptedCognitionDriver{name: "scripted-isolated", body: `[{"actor_id":"` + id.M + `","decision":"none"}]`}
		resolve := &countingResolveDriver{inner: NewFakeResolveDriver()}

		orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated, WorldActor: NewFakeWorldActorDriver()}

		outcome, err := orc.RunBeat(ctx, id.World, id.P, playerGreetsMara(), baseTick)
		if err != nil {
			t.Fatalf("RunBeat: %v", err)
		}

		// (a) one call per NPC per action, each in exactly one seat.
		if batch.calls != 1 {
			t.Fatalf("batch Generate calls = %d, want 1", batch.calls)
		}
		if isolated.calls != 1 {
			t.Fatalf("isolated Generate calls = %d, want 1", isolated.calls)
		}
		// (a) DECIDE FOR = [Jonas] for the batch, [Mara] for the isolated seat.
		bTail := decideForTail(t, batch.prompts[0])
		if !strings.Contains(bTail, id.J) || strings.Contains(bTail, id.M) {
			t.Fatalf("batch DECIDE FOR = %q, want just Jonas (%s), not Mara (%s)", bTail, id.J, id.M)
		}
		iTail := decideForTail(t, isolated.prompts[0])
		if !strings.Contains(iTail, id.M) || strings.Contains(iTail, id.J) {
			t.Fatalf("isolated DECIDE FOR = %q, want just Mara (%s), not Jonas (%s)", iTail, id.M, id.J)
		}

		// (b) the batch's decision FOR Mara is rejected (non-present-for-this-call): nothing
		// Mara-authored commits. Only the player's own Communicated lands → exactly one committed.
		if len(outcome.Committed) != 1 {
			t.Fatalf("committed = %d %v, want exactly 1 (player only; batch M-decision rejected)", len(outcome.Committed), outcome.Committed)
		}
		if resolve.calls != 0 {
			t.Fatalf("resolve calls = %d, want 0 (no adjudicated commit in this subtest)", resolve.calls)
		}
	})

	// (c) An NPC commit of an ADJUDICATED type routes through the resolve seat — no bypass.
	t.Run("no bypass: adjudicated NPC commit hits resolve", func(t *testing.T) {
		baseTick := flowBaseTick(t, ctx, pool, id.World)

		// Jonas (batch) commits an AttributeChanged — an adjudicated type. It MUST hit resolve.
		batch := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.J +
			`","decision":{"commit_kind":"commit","attempt":{"type":"AttributeChanged","stated":"Jonas grabs the note","target_id":"` + id.Note + `"}}}]`}
		isolated := &scriptedCognitionDriver{name: "scripted-isolated", body: `[{"actor_id":"` + id.M + `","decision":"none"}]`}
		resolve := &countingResolveDriver{inner: NewFakeResolveDriver()}

		orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated, WorldActor: NewFakeWorldActorDriver()}

		if _, err := orc.RunBeat(ctx, id.World, id.P, playerGreetsMara(), baseTick); err != nil {
			t.Fatalf("RunBeat: %v", err)
		}
		if resolve.calls < 1 {
			t.Fatalf("resolve calls = %d, want >=1 (adjudicated NPC commit must route through resolve — no bypass)", resolve.calls)
		}
	})

	// (d) An NPC commit of a PASSTHROUGH type commits via apply_event, never touching resolve.
	t.Run("passthrough NPC commit never touches resolve", func(t *testing.T) {
		baseTick := flowBaseTick(t, ctx, pool, id.World)

		// Jonas (batch) commits a Communicated — a passthrough type. Resolve must stay untouched.
		batch := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.J +
			`","decision":{"commit_kind":"commit","attempt":{"type":"Communicated","stated":"Jonas greets","listener_id":"` + id.P +
			`","content":"well met"}}}]`}
		isolated := &scriptedCognitionDriver{name: "scripted-isolated", body: `[{"actor_id":"` + id.M + `","decision":"none"}]`}
		resolve := &countingResolveDriver{inner: NewFakeResolveDriver()}

		orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated, WorldActor: NewFakeWorldActorDriver()}

		outcome, err := orc.RunBeat(ctx, id.World, id.P, playerGreetsMara(), baseTick)
		if err != nil {
			t.Fatalf("RunBeat: %v", err)
		}
		if resolve.calls != 0 {
			t.Fatalf("resolve calls = %d, want 0 (passthrough Communicated must NOT touch the resolve seat)", resolve.calls)
		}
		// Player's Communicated + Jonas's Communicated both commit via apply_event.
		if len(outcome.Committed) != 2 {
			t.Fatalf("committed = %d %v, want 2 (player + Jonas, both passthrough)", len(outcome.Committed), outcome.Committed)
		}
	})
}
