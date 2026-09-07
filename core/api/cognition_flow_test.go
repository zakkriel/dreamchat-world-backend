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
	reply   func(GenRequest) string
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
	if d.reply != nil {
		return d.reply(req), nil
	}
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

		outcome, err := orc.RunBeat(ctx, id.World, id.P, playerGreetsMara(), baseTick, nil)
		if err != nil {
			t.Fatalf("RunBeat: %v", err)
		}

		if len(batch.prompts) == 0 || len(isolated.prompts) == 0 {
			t.Fatal("both cognition paths must run for this isolation scenario")
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
	})

	// A spoken NPC decision must actually become sourced speech, not just reach a driver.
	t.Run("NPC speech is committed through the shared path", func(t *testing.T) {
		baseTick := flowBaseTick(t, ctx, pool, id.World)
		batch := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.J +
			`","decision":{"commit_kind":"commit","attempt":{"type":"Communicated","stated":"Jonas greets","listener_id":"` + id.P +
			`","content":"well met"}}}]`}
		isolated := &scriptedCognitionDriver{name: "scripted-isolated", body: `[{"actor_id":"` + id.M + `","decision":"none"}]`}
		resolve := &countingResolveDriver{inner: NewFakeResolveDriver()}

		orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated, WorldActor: NewFakeWorldActorDriver()}

		outcome, err := orc.RunBeat(ctx, id.World, id.P, playerGreetsMara(), baseTick, nil)
		if err != nil {
			t.Fatalf("RunBeat: %v", err)
		}
		var spokenCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM canon_event ce
		  JOIN event_participant ep USING(event_id)
		  WHERE ce.event_id = ANY($1::uuid[]) AND ep.entity_id=$2::uuid
		    AND ep.role_qualifier='speaker' AND ce.payload->>'spoken'='well met'`,
			outcome.Committed, id.J).Scan(&spokenCount); err != nil {
			t.Fatal(err)
		}
		if spokenCount == 0 {
			t.Fatal("the NPC's spoken decision was not committed")
		}
	})
}

// These fixtures prescribe application results; they do not evaluate model interpretation.
func TestSpeechCognition_UnsaidAndMissedWordsStayOut(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	const words = "speech-only-after-commit"
	const intention = "unspoken-intention-marker"
	batch := &scriptedCognitionDriver{name: "batch", body: "[]"}
	isolated := &scriptedCognitionDriver{name: "isolated", body: "[]"}
	resolve := &scriptedCognitionDriver{name: "accepted-perception", body: fmt.Sprintf(
		`{"schema_version":"speech_perception/1","listeners":[
		  {"listener_id":%q,"attention":{"kind":"blocked","reason":"recorded distraction"},"name_associations":[],"heard_words":""},
		  {"listener_id":%q,"attention":{"kind":"abstain"},"name_associations":[],"heard_words":%q}]}`,
		id.M, id.J, words)}
	orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated}
	_, err := orc.RunBeat(ctx, id.World, id.P, []Attempt{{
		Type: "Communicated", Stated: intention, ListenerID: id.M, Content: words,
	}}, flowBaseTick(t, ctx, pool, id.World), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.prompts) == 0 || len(isolated.prompts) == 0 {
		t.Fatal("the pre-speech interruption scenario did not reach both minds")
	}
	for _, prompt := range []string{batch.prompts[0], isolated.prompts[0]} {
		if strings.Contains(prompt, words) || strings.Contains(prompt, intention) {
			t.Fatal("pre-speech cognition received unsaid information")
		}
	}
	heardByJonas := false
	for _, prompt := range append(batch.prompts, isolated.prompts...) {
		tail := decideForTail(t, prompt)
		if strings.Contains(tail, id.M) && strings.Contains(prompt, words) {
			t.Fatal("the listener who missed the speech received its words")
		}
		if strings.Contains(tail, id.J) && strings.Contains(prompt, words) {
			heardByJonas = true
		}
	}
	if !heardByJonas {
		t.Fatal("the actual listener never received the committed speech")
	}
}

func TestSpeechCognition_ReceiverAccountsRemainSeparate(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	const source = "source-only-code"
	const maraWords = "mara-only-code"
	const jonasWords = "jonas-only-code"
	resolve := &scriptedCognitionDriver{name: "accepted-perception", body: fmt.Sprintf(
		`{"schema_version":"speech_perception/1","listeners":[
		  {"listener_id":%q,"attention":{"kind":"abstain"},"name_associations":[],"heard_words":%q},
		  {"listener_id":%q,"attention":{"kind":"abstain"},"name_associations":[],"heard_words":%q}]}`,
		id.M, maraWords, id.J, jonasWords)}
	batch := &scriptedCognitionDriver{name: "batch", body: "[]"}
	isolated := &scriptedCognitionDriver{name: "isolated", body: "[]"}
	orc := &Orchestrator{DB: pool, Resolve: resolve, CognitionBatch: batch, CognitionIsolated: isolated}
	tick := flowBaseTick(t, ctx, pool, id.World)
	result, err := orc.applyRuledEvent(ctx, id.World, RuledEventV2{
		Type: "Communicated", ActorID: id.P, ListenerID: id.M, Content: source,
		Truth: "The player gives a code.",
		ReceiverVariants: []ReceiverVariant{
			{ReceiverID: id.M, Text: "The player gives Mara a code."},
			{ReceiverID: id.J, Text: "The player gives Jonas a code."},
		},
	}, tick, 0)
	if err != nil {
		t.Fatal(err)
	}
	eventID, ok := result["event_id"].(string)
	if !ok || eventID == "" {
		t.Fatalf("speech did not commit: %v", result)
	}
	if _, err := orc.postCommittedCognition(ctx, id.World, id.P, []string{eventID}, tick, 1, nil); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, prompt := range append(batch.prompts, isolated.prompts...) {
		tail := decideForTail(t, prompt)
		for actor, ownWords := range map[string]string{id.M: maraWords, id.J: jonasWords} {
			if !strings.Contains(tail, actor) {
				continue
			}
			otherWords := maraWords
			if actor == id.M {
				otherWords = jonasWords
			}
			if !strings.Contains(prompt, ownWords) || strings.Contains(prompt, otherWords) || strings.Contains(prompt, source) {
				t.Fatalf("receiver %s did not get only its own accepted speech", actor)
			}
			seen[actor] = true
		}
	}
	if !seen[id.M] || !seen[id.J] {
		t.Fatal("both actual receivers must get their own cognition context")
	}
}

func TestSpeechCognition_CueDoesNotDisclosePrivateRecipient(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	var recipient string
	if err := pool.QueryRow(ctx, `INSERT INTO entity_registry(entity_id,world_id,entity_kind,canonical_name)
		VALUES(gen_random_uuid(),$1,'actor','private contact') RETURNING entity_id::text`, id.World).Scan(&recipient); err != nil {
		t.Fatal(err)
	}
	batch := &scriptedCognitionDriver{name: "batch", body: "[]"}
	isolated := &scriptedCognitionDriver{name: "isolated", body: "[]"}
	orc := &Orchestrator{DB: pool, CognitionBatch: batch, CognitionIsolated: isolated}
	_, err := orc.worldFirst(ctx, id.World, id.P, Attempt{
		Type: "Communicated", ListenerID: recipient, Content: "not-yet-said", Stated: "private-intention",
	}, flowBaseTick(t, ctx, pool, id.World), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	prompts := append(batch.prompts, isolated.prompts...)
	if len(prompts) == 0 {
		t.Fatal("pre-speech cognition did not run")
	}
	for _, prompt := range prompts {
		if strings.Contains(prompt, recipient) || strings.Contains(prompt, "not-yet-said") || strings.Contains(prompt, "private-intention") {
			t.Fatal("a speaking cue disclosed the intended recipient or unsaid content")
		}
	}
}

func TestSpeechCognition_TelegraphDoesNotSkipDueEvents(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	const spoken = "speech-before-pending-event"
	cognition := &scriptedCognitionDriver{name: "cognition", reply: func(req GenRequest) string {
		if strings.Contains(req.Prompt, spoken) && strings.Contains(decideForTail(t, req.Prompt), id.M) {
			return fmt.Sprintf(`[{"actor_id":%q,"decision":{"commit_kind":"telegraph","attempt":{"type":"Communicated","stated":"Mara begins to answer.","listener_id":%q,"content":"One moment."}}}]`, id.M, id.P)
		}
		return "[]"
	}}
	orc := &Orchestrator{DB: pool, Resolve: &fakeResolveDriver{}, CognitionBatch: cognition, CognitionIsolated: cognition}
	baseTick := flowBaseTick(t, ctx, pool, id.World)
	duration, err := orc.nonMoveDurationSeconds(ctx, id.World, "instant")
	if err != nil || duration <= 0 {
		t.Fatalf("clock-advancing speech duration=%d, error=%v", duration, err)
	}
	pending := lgInsertPending(t, ctx, pool, id.World, baseTick+duration, "small", id.J,
		fmt.Sprintf(`{"type":"Communicated","stated":"Jonas speaks at the appointed time.","listener_id":%q,"content":"The appointed hour has come."}`, id.P))
	out, err := orc.RunBeat(ctx, id.World, id.P, []Attempt{
		{Type: "Communicated", Stated: "Player speaks.", ListenerID: id.M, Content: spoken},
	}, baseTick, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.HaltReason != "telegraph" {
		t.Fatalf("halt=%q, want the post-speech telegraph", out.HaltReason)
	}
	if status := lgPendingStatus(t, ctx, pool, pending); status != "fired" {
		t.Fatalf("due event left %q after speech advanced the clock", status)
	}
}

func TestSpeechCognition_UsesPrivateKnowledgeOfPerceivedSubjects(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	const secret = "Mara alone knows Jonas hid the silver."
	baseTick := flowBaseTick(t, ctx, pool, id.World)
	_, err := pool.Exec(ctx, `WITH event AS (
		INSERT INTO canon_event(world_id,event_type,summary,in_world_tick,beat_seq,status,origin)
		VALUES($1,'AttributeChanged',$5,$4,0,'accepted','fast_path') RETURNING event_id
	), perception AS (
		INSERT INTO perception_record(world_id,holder_id,source_event_id,content,epistemic_type,acquired_tick,valid_tick)
		SELECT $1,$2,event_id,$5,'direct',$4,$4 FROM event RETURNING perception_id
	)
	INSERT INTO perception_subject(perception_id,entity_id,world_id)
	SELECT perception_id,$3,$1 FROM perception`, id.World, id.M, id.J, baseTick, secret)
	if err != nil {
		t.Fatal(err)
	}
	batch := &scriptedCognitionDriver{name: "batch", body: "[]"}
	isolated := &scriptedCognitionDriver{name: "isolated", body: "[]"}
	orc := &Orchestrator{DB: pool, Resolve: &fakeResolveDriver{}, CognitionBatch: batch, CognitionIsolated: isolated}
	event, err := orc.applyEvent(ctx, id.World, id.P, []byte(fmt.Sprintf(
		`{"type":"Communicated","listener_id":%q,"stated":"Player asks Jonas.","content":"Where is the silver?"}`, id.J)), baseTick+1, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = orc.postCommittedCognition(ctx, id.World, id.P, []string{event["event_id"].(string)}, baseTick+2, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, prompt := range append(batch.prompts, isolated.prompts...) {
		forMara := strings.Contains(decideForTail(t, prompt), id.M)
		hasSecret := strings.Contains(prompt, secret)
		if forMara && hasSecret {
			found = true
		}
		if !forMara && hasSecret {
			t.Fatal("Mara's private knowledge reached another listener's response")
		}
	}
	if !found {
		t.Fatal("Mara's response omitted her private knowledge about the perceived addressee")
	}
}

func TestReactionBeat_NonSpeechOutcomeStillAllowsResponse(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	tick := flowBaseTick(t, ctx, pool, id.World)
	seedHeld(t, ctx, pool, id.World, id.J,
		Attempt{Type: "AttributeChanged", TargetID: id.P, Stated: "He reaches for the stranger."}, tick)
	held, err := pendingHeldOutcomes(ctx, pool, id.World)
	if err != nil || len(held) != 1 {
		t.Fatalf("pending reaction: held=%v, err=%v", held, err)
	}
	const unsaid = "The vault key is under the stair."
	const perceived = "A hand brushes the sleeve."
	const response = "She raises a hand."
	cognition := &scriptedCognitionDriver{name: "cognition", reply: func(req GenRequest) string {
		if strings.Contains(req.Prompt, unsaid) {
			t.Fatal("post-reaction cognition received the interrupted words")
		}
		if strings.Contains(decideForTail(t, req.Prompt), id.M) && strings.Contains(req.Prompt, perceived) {
			return fmt.Sprintf(`[{"actor_id":%q,"decision":{"commit_kind":"telegraph","attempt":{"type":"Communicated","stated":%q,"listener_id":%q,"content":"One moment."}}}]`, id.M, response, id.P)
		}
		return "[]"
	}}
	orc := &Orchestrator{
		DB: pool, CognitionBatch: cognition, CognitionIsolated: cognition,
		Resolve: &capturingResolveDriver{name: "resolve",
			ruling: validRulingJSON(id.P, id.M, "Player touches the sleeve instead of speaking.", perceived)},
	}
	out, err := orc.RunReactionBeat(ctx, id.World, id.P,
		[]Attempt{{Type: "Communicated", Stated: "I reveal the hiding place.", ListenerID: id.M, Content: unsaid}},
		held, tick+1, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.HaltReason != "telegraph" || len(out.Telegraphs) != 1 || out.Telegraphs[0] != response {
		t.Fatalf("perceived non-speech outcome did not produce the response: halt=%q, telegraphs=%v", out.HaltReason, out.Telegraphs)
	}
}
