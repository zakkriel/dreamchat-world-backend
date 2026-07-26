package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Station E — Task 7: THE WALL TESTS. The perception wall must hold BY CONSTRUCTION
// (RULINGS-2026-07-23 §5: "The wall holds by construction, not by prompt discipline. A secret never
// enters a shared prompt, so it cannot bleed into another NPC's reasoning… You cannot validate a
// leak away — you can only not create it."). These tests prove the construction by asserting on the
// EXACT prompts every seat is handed.
//
// The invariant, per seat (RULINGS-2026-07-23 §5, §6, and §9's wall note):
//   - The perception walls protect the CHARACTER-MIND seats — cognition (batch + isolated), decompose,
//     and narrate. A secret M holds privately must never appear in a prompt those seats can see for
//     anyone but M herself.
//   - The REFEREE (the resolve seat) is truth-side, LICENSED by design to read every involved party's
//     full canon (§9 "Wall note": "the resolver is the referee, truth-side, licensed by design to read
//     the involved parties' full canon. The perception walls protect the character-mind seats
//     (cognition, decompose, narrate), never the referee."). Blinding the referee is a bug, so (f)
//     pins the OPPOSITE direction: the secret MUST be present once M is a ruling participant.
//
// Cast (fresh, random world per subtest — hermetic, never seed-dependent, high ticks so nothing
// collides with the SQL suite or the seed world): player K, NPC M (holds one PRIVATE record about K),
// NPC J (holds nothing), all co-located at L.
//
// The secret is grounded HONESTLY as a single canon fact: a private observation event whose truthful
// summary IS the secret, with M as its participant, and M's own private perception of it (subject-
// linked to K). That one grounding drives BOTH walls at once:
//   - the cognition/decompose/narrate seats reach it only through M's private PERCEPTION (fn_private_
//     records) — so it rides her isolated call alone and nothing else;
//   - the referee reaches it only through gather_slice's recent_events (event_participant → canon
//     summary) — the licensed truth-side path, exercised only when M is a ruling participant.

// wallSecret is M's private line. Its two load-bearing substrings are what every assertion greps —
// they appear nowhere else in the fixture (not in a name, a location, a beat text, or an attempt).
const wallSecret = "She owes K's family a life-debt — she hid them in the cellar"

// wallSecretMarkers are the substrings the wall must keep out of / let into the right prompts.
var wallSecretMarkers = []string{"life-debt", "cellar"}

// wallIDs holds the freshly-minted ids for one subtest invocation.
type wallIDs struct{ World, K, M, J, L, SecretEvent, SecretPerception string }

// setupWallWorld mints a fresh world + cast and seeds M's one private record. The SAME canon fact is
// both (a) M's private perception content (fn_private_records → her isolated call) and (b) a canon
// event summary M participated in (gather_slice → the referee). That dual grounding is deliberate:
// it lets one fixture prove the wall keeps the fact out of the character-mind seats AND that the
// licensed referee can still see it (f). High literal ticks keep every row above the seed world.
//
// withSubjectLink controls the §6 about-ness link. true = M's record is subject-linked to K (the
// normal case: she is flagged isolated). false = the link is OMITTED — the (e) "missed flag": an
// UNLINKED secret is invisible to every mechanical lookup (§6), so M falls dull into the batch.
// (perception_subject is append-only canon — a missed flag is modelled by never writing the link,
// not by deleting it.)
func setupWallWorld(t *testing.T, ctx context.Context, pool *pgxpool.Pool, withSubjectLink bool) wallIDs {
	t.Helper()
	var id wallIDs
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&id.World, &id.K, &id.M, &id.J, &id.L, &id.SecretEvent); err != nil {
		t.Fatalf("mint ids: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
		 ($1,$5,'actor','Player'),
		 ($2,$5,'actor','Mara'),
		 ($3,$5,'actor','Jonas'),
		 ($4,$5,'location','The Drowned Lantern')`,
		id.K, id.M, id.J, id.L, id.World); err != nil {
		t.Fatalf("seed wall entities: %v", err)
	}

	// The secret's grounding: a PRIVATE observation event whose truthful summary IS the secret, with
	// M as its participant. Private + held by M alone ⇒ never in fn_public_moment (the batch's only
	// event source). M-as-participant ⇒ visible to gather_slice's recent_events when M is in a ruling.
	if _, err := pool.Exec(ctx, `
		WITH ev AS (
		  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
		  VALUES ($1,$2,'observation',$4,60000,0,'accepted',now(),'private','fast_path')
		  RETURNING event_id
		)
		INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
		SELECT event_id,$3,'actor','instigator' FROM ev`,
		id.SecretEvent, id.World, id.M, wallSecret); err != nil {
		t.Fatalf("seed secret event: %v", err)
	}

	// M's PRIVATE perception of that event; content = the secret. Held by M alone ⇒ fn_isolated_npcs
	// flags her (never public), fn_private_records carries it on HER call only.
	if err := pool.QueryRow(ctx, `
		INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type, acquired_tick, valid_tick)
		VALUES ($1,$2,$3,$4,'direct',60000,60000) RETURNING perception_id`,
		id.World, id.M, id.SecretEvent, wallSecret).Scan(&id.SecretPerception); err != nil {
		t.Fatalf("seed secret perception: %v", err)
	}
	// About-ness (RULINGS-2026-07-23 §6): the subject link is K. The action's bound ids intersect it
	// (every beat here involves K), so the mechanical §5 lookup flags M isolated. Omitted for (e).
	if withSubjectLink {
		if _, err := pool.Exec(ctx,
			`INSERT INTO perception_subject (perception_id, entity_id, world_id) VALUES ($1,$2,$3)`,
			id.SecretPerception, id.K, id.World); err != nil {
			t.Fatalf("seed secret subject: %v", err)
		}
	}

	// Co-locate K, M, J at L (each move is the actor's latest state → fn_actors_at returns all three).
	for i, actor := range []string{id.K, id.M, id.J} {
		if _, err := pool.Exec(ctx, `
			WITH ev AS (
			  INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
			  VALUES (gen_random_uuid(),$1,'move','wall-colocate',$2,0,'accepted',now(),'public','fast_path')
			  RETURNING event_id
			),
			ep AS (
			  INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier)
			  SELECT event_id,$3,'actor','instigator' FROM ev
			)
			INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
			SELECT $1,event_id,$3,'actor','attrs.location_id',to_jsonb($4::text),$2,0 FROM ev`,
			id.World, int64(60001+i), actor, id.L); err != nil {
			t.Fatalf("colocate %s: %v", actor, err)
		}
	}
	return id
}

// capturingSeatDriver records the FULL GenRequest handed to a seat on EVERY call, and returns a fixed
// reply. It reports structured_output so it can bind to any seat (narrate requires nothing; the rest
// require the structured floor). One instance per seat so each seat's prompts are inspected alone.
type capturingSeatDriver struct {
	name  string
	reply string
	reqs  []GenRequest
}

func (d *capturingSeatDriver) Name() string { return d.name }
func (d *capturingSeatDriver) Capabilities() CapabilitySet {
	return CapabilitySet{CapStructuredOutput: true}
}
func (d *capturingSeatDriver) Generate(_ context.Context, req GenRequest) (string, error) {
	d.reqs = append(d.reqs, req)
	return d.reply, nil
}

// seatTexts returns, per call, the COMPLETE text the seat saw: its Prompt AND its perception-bound
// Payload lines. The wall must hold across BOTH vectors (a leak could ride either) and across ALL
// calls, not just the first (RULINGS-2026-07-23 §5 — a secret must be in NO shared prompt, ever).
func seatTexts(d *capturingSeatDriver) []string {
	out := make([]string, 0, len(d.reqs))
	for _, r := range d.reqs {
		out = append(out, r.Prompt+"\n"+strings.Join(r.Payload.Lines, "\n"))
	}
	return out
}

// assertNoLeak fails if ANY of a seat's captured prompts (Prompt+Payload) contains ANY secret marker.
// A seat that was never called cannot leak — that is a legitimate pass for the no-leak direction; a
// test that REQUIRES a call asserts the call count separately.
func assertNoLeak(t *testing.T, seat string, d *capturingSeatDriver) {
	t.Helper()
	for i, txt := range seatTexts(d) {
		for _, m := range wallSecretMarkers {
			if strings.Contains(txt, m) {
				t.Fatalf("WALL BREACH — %s prompt (call %d) leaked secret marker %q:\n%s", seat, i, m, txt)
			}
		}
	}
}

// assertCarriesSecret fails unless the seat was called AND every marker appears across its prompts.
// Used only for M's isolated seat (b): her secret MUST ride her own call.
func assertCarriesSecret(t *testing.T, seat string, d *capturingSeatDriver) {
	t.Helper()
	if len(d.reqs) == 0 {
		t.Fatalf("%s was never called — M's isolated call MUST fire (was she flagged isolated?)", seat)
	}
	joined := strings.Join(seatTexts(d), "\n")
	for _, m := range wallSecretMarkers {
		if !strings.Contains(joined, m) {
			t.Fatalf("%s carried NO secret marker %q — M's private line did not ride her own isolated call", seat, m)
		}
	}
}

// wallBridge binds a fresh capturing driver to ALL SIX seats (including cognition_isolated, which the
// beat handler consults — station_d's harness omits it) and hands back the six so the test can read
// their captured prompts after the beat. The decompose reply is the scripted chain for this beat.
func wallBridge(t *testing.T, id wallIDs, decomposeChain string) (*Bridge, map[string]*capturingSeatDriver) {
	t.Helper()
	seats := map[string]*capturingSeatDriver{
		SeatDecompose.Name:         {name: "capture-decompose", reply: decomposeChain},
		SeatNarrate.Name:           {name: "capture-narrate", reply: "The tavern holds its breath."},
		SeatResolve.Name:           {name: "capture-resolve", reply: validRulingJSON(id.K, id.M, "nothing lands", "nothing shows")},
		SeatCognitionBatch.Name:    {name: "capture-batch", reply: "[]"},
		SeatCognitionIsolated.Name: {name: "capture-isolated", reply: "[]"},
		SeatWorldActor.Name:        {name: "capture-worldactor", reply: "[]"},
	}
	bound := map[string]Driver{}
	for k, v := range seats {
		bound[k] = v
	}
	bridge, err := NewBridgeWithDrivers(bound,
		SeatDecompose, SeatNarrate, SeatResolve, SeatCognitionBatch, SeatCognitionIsolated, SeatWorldActor)
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	return bridge, seats
}

// runWallBeat drives the FULL HTTP path (decompose → RunBeat → narrate) as player K, so every
// character-mind seat's real prompt is captured. Returns the six capturing drivers.
func runWallBeat(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id wallIDs, beatText, chain string) map[string]*capturingSeatDriver {
	t.Helper()
	bridge, seats := wallBridge(t, id, chain)
	h := NewBeatHandler(pool, true /*debug → ?viewer honored*/, bridge)

	req := httptest.NewRequest(http.MethodPost,
		"/worlds/"+id.World+"/beat?viewer="+id.K,
		strings.NewReader(`{"text":"`+beatText+`"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("beat status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	return seats
}

// TestWall_SharedSeatsNeverLeak_IsolatedCarries is (a)–(d): one beat where K presses M. M holds a
// private record about K whose subject intersects the action → she is pulled ISOLATED; J holds
// nothing → he sits in the shared BATCH. The one beat's captured prompts must show the wall holding.
func TestWall_SharedSeatsNeverLeak_IsolatedCarries(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true)

	// Benign beat: a Communicated K→M whose text carries NO marker, so any marker that appears in a
	// prompt came from the MACHINERY (M's private record / the referee's slice), never from the input.
	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`

	seats := runWallBeat(t, ctx, pool, id, beatText, chain)
	batch := seats[SeatCognitionBatch.Name]
	isolated := seats[SeatCognitionIsolated.Name]

	// (a) the SHARED batch prompt (public roster + public moment + imminent attempt) carries neither
	//     marker, on every batch call. Breaks loudly if a private line ever reached the shared payload.
	assertNoLeak(t, "cognition_batch (a)", batch)

	// (b) M's ISOLATED prompt DOES carry both markers — her secret rides alone, on her own call.
	assertCarriesSecret(t, "cognition_isolated (b)", isolated)

	// (c) the DECOMPOSE and NARRATE prompts carry neither — the wall protects EVERY character-mind
	//     seat (RULINGS-2026-07-23 §9 wall note), not just cognition. Both are perception-bound to K,
	//     who never holds M's private perception, so neither the prompt nor the payload can carry it.
	assertNoLeak(t, "decompose (c)", seats[SeatDecompose.Name])
	assertNoLeak(t, "narrate (c)", seats[SeatNarrate.Name])
	// Belt-and-suspenders: no shared/side seat leaks either (resolve isn't called on a passthrough).
	assertNoLeak(t, "resolve", seats[SeatResolve.Name])
	assertNoLeak(t, "world_actor", seats[SeatWorldActor.Name])

	// (d) M NEVER appears in a batch DECIDE FOR list while flagged. Her id DOES appear elsewhere in
	//     the batch prompt (she is the imminent Communicated's listener), so this must check the
	//     DECIDE FOR tail specifically — the seat must be told to speak for J only, not for M.
	if len(batch.reqs) == 0 {
		t.Fatalf("(d) no batch call captured — J should sit in the shared batch")
	}
	for i, r := range batch.reqs {
		tail := decideForTail(t, r.Prompt)
		if strings.Contains(tail, id.M) {
			t.Fatalf("(d) batch DECIDE FOR (call %d) names flagged M %s — the isolated NPC must not be a batch decidee:\n%s", i, id.M, tail)
		}
		if !strings.Contains(tail, id.J) {
			t.Fatalf("(d) batch DECIDE FOR (call %d) missing J %s:\n%s", i, id.J, tail)
		}
	}

	perceptionSubjectBackfill(t, ctx, pool, 60000)
}

// TestWall_DullNeverLeak is (e): the failure-asymmetry proof (RULINGS-2026-07-23 §5 — "a missed flag
// → that NPC reacts flat this action (dull, never a leak)"). Delete the perception_subject link
// (simulating a missed about-ness write) and rerun the SAME beat: M is NO LONGER flagged, she sits in
// the shared BATCH, and the secret appears in NO prompt ANYWHERE. Dull, not leaked — the safe failure.
func TestWall_DullNeverLeak(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	// The missed flag: seed M's private record with its about-ness link OMITTED (perception_subject is
	// append-only canon, so a missed flag is an unwritten link, not a deleted one). fn_isolated_npcs'
	// EXISTS(perception_subject …) now fails for M → she is not isolated; fn_private_records returns
	// nothing for her either.
	id := setupWallWorld(t, ctx, pool, false)

	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`

	seats := runWallBeat(t, ctx, pool, id, beatText, chain)
	batch := seats[SeatCognitionBatch.Name]
	isolated := seats[SeatCognitionIsolated.Name]

	// M SITS IN THE BATCH now: the isolated seat never fires, and the batch is told to speak for M.
	if len(isolated.reqs) != 0 {
		t.Fatalf("(e) isolated seat fired %d time(s) — with the flag missed, M must fall into the batch, not get her own call", len(isolated.reqs))
	}
	if len(batch.reqs) == 0 {
		t.Fatalf("(e) no batch call captured — M (and J) must sit in the shared batch")
	}
	sawM := false
	for _, r := range batch.reqs {
		if strings.Contains(decideForTail(t, r.Prompt), id.M) {
			sawM = true
		}
	}
	if !sawM {
		t.Fatalf("(e) M %s never appeared in a batch DECIDE FOR — the scenario did not actually put her in the batch", id.M)
	}

	// …and yet the secret is in NO prompt anywhere (batch, isolated, decompose, narrate, resolve,
	// world_actor). A missed flag makes M dull — it can NEVER turn into a leak (the asymmetry).
	for name, d := range seats {
		assertNoLeak(t, "(e) "+name, d)
	}
	// No perception_subject backfill here: the whole point is the UNLINKED secret. Backfilling would
	// derive a subject from the event participant and re-link it — defeating the missed-flag fixture.
}

// TestWall_NameStringConfinedToKnower is the §3 naming-reach wall (Defect C): a canonical name must
// never reach a character-mind seat past a knowledge path. Kade's name is known ONLY to Mara (privately);
// no batch mind (Jonas) knows it. So the literal string "Kade" must appear in NO seat's prompt EXCEPT
// Mara's isolated call — the batch labels him by his descriptor, and decompose/narrate (perception-bound
// to Kade, who does not know his own registered name) never surface it either. Same self-contained
// fixture family as the other wall tests; high ticks keep it clear of the seed world.
func TestWall_NameStringConfinedToKnower(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true) // K/M/J co-located; M holds a private secret about K → isolated

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v\n%s", err, sql)
		}
	}
	// Give the player a distinctive canonical name and a stranger's descriptor; Jonas will see only the
	// descriptor (he has no name-knowledge of K).
	mustExec(`UPDATE entity_registry SET canonical_name='Kade' WHERE entity_id=$1 AND world_id=$2`, id.K, id.World)
	mustExec(`UPDATE actor_state SET attrs = jsonb_set(attrs,'{descriptor}','"a young stranger"'::jsonb)
	          WHERE entity_id=$1 AND world_id=$2`, id.K, id.World)

	// Mara PRIVATELY knows K's name — a world_genesis-sourced name perception she holds ALONE, subject K.
	// Held-by-one ⇒ only her calls resolve "Kade" (fn_display_name); the wall holds by construction.
	var genEvt, namePid string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&genEvt); err != nil {
		t.Fatalf("mint genesis id: %v", err)
	}
	mustExec(`INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
	          VALUES ($1,$2,'world_genesis','names',59000,0,'accepted',now(),'public','fast_path')`, genEvt, id.World)
	if err := pool.QueryRow(ctx,
		`INSERT INTO perception_record (world_id,holder_id,source_event_id,content,epistemic_type,acquired_tick,valid_tick)
		 VALUES ($1,$2,$3,'Kade','told',59000,59000) RETURNING perception_id`,
		id.World, id.M, genEvt).Scan(&namePid); err != nil {
		t.Fatalf("seed name perception: %v", err)
	}
	mustExec(`INSERT INTO perception_subject (perception_id,entity_id,world_id) VALUES ($1,$2,$3)`, namePid, id.K, id.World)

	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`
	seats := runWallBeat(t, ctx, pool, id, beatText, chain)

	// The batch labels Kade by his DESCRIPTOR (Jonas does not know his name) and NEVER by "Kade".
	batch := seats[SeatCognitionBatch.Name]
	if len(batch.reqs) == 0 {
		t.Fatalf("no batch call captured — Jonas should sit in the shared batch")
	}
	sawDescriptor := false
	for i, txt := range seatTexts(batch) {
		if strings.Contains(txt, "Kade") {
			t.Fatalf("WALL BREACH — batch prompt (call %d) named Kade, but no batch mind knows his name:\n%s", i, txt)
		}
		if strings.Contains(txt, "a young stranger") {
			sawDescriptor = true
		}
	}
	if !sawDescriptor {
		t.Fatalf("batch prompt never labelled Kade by his descriptor — the relabel did not run")
	}

	// "Kade" appears ONLY in Mara's isolated prompt (her private name-knowledge), and NOWHERE else.
	isolated := seats[SeatCognitionIsolated.Name]
	if len(isolated.reqs) == 0 {
		t.Fatalf("Mara's isolated call never fired — she must be pulled isolated by her secret")
	}
	if !strings.Contains(strings.Join(seatTexts(isolated), "\n"), "Kade") {
		t.Fatalf("Mara's isolated prompt did NOT carry her known name for Kade — fn_display_name did not resolve it")
	}
	for name, d := range seats {
		if name == SeatCognitionIsolated.Name {
			continue
		}
		for i, txt := range seatTexts(d) {
			if strings.Contains(txt, "Kade") {
				t.Fatalf("WALL BREACH — %s prompt (call %d) leaked the name \"Kade\" past a knowledge path:\n%s", name, i, txt)
			}
		}
	}

	perceptionSubjectBackfill(t, ctx, pool, 59000)
}

// TestWall_RefereeStaysSighted is (f): the OPPOSITE direction. The referee (resolve seat) is truth-
// side, LICENSED to read every involved party's full canon (RULINGS-2026-07-23 §9 wall note). When M
// is a ruling participant, her secret — grounded as a canon event she participated in — MUST reach
// the referee via gather_slice's recent_events. This test fails if someone ever "fixes" the referee
// into the same blindness the character-mind seats are given. It is the combined-ruling case: M's
// held act collides with K's reaction in ONE ruling, so several actors' truth-side state is licensed
// into a single judgment (§9).
func TestWall_RefereeStaysSighted(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true)

	// Beat 1 (seeded directly): M telegraphs a disruptive act → a pending held_outcome. This makes M
	// a genuine ruling PARTICIPANT (an actor in the combined set), the §9 trigger.
	seedHeld(t, ctx, pool, id.World, id.M,
		Attempt{Type: "AttributeChanged", Stated: "Mara steels herself", TargetID: id.M}, 65000)
	held, err := pendingHeldOutcomes(ctx, pool, id.World)
	if err != nil {
		t.Fatalf("pendingHeldOutcomes: %v", err)
	}
	if len(held) != 1 {
		t.Fatalf("pending held = %d, want 1 (M's telegraphed act)", len(held))
	}

	// Beat 2: K reacts. M's held act + K's first action → ONE combined ruling (§9). The reaction text
	// carries NO marker, so any marker in the referee's prompt came from gather_slice (M's canon),
	// not from the input — the whole point of (f).
	resolve := &capturingResolveDriver{name: "capture-resolve", ruling: validRulingJSON(id.K, id.M, "K's demand lands hard", "K leans in")}
	orc := &Orchestrator{
		DB:                pool,
		Resolve:           resolve,
		CognitionBatch:    NewFakeCognitionDriver(),
		CognitionIsolated: NewFakeCognitionDriver(),
		WorldActor:        NewFakeWorldActorDriver(),
	}
	reactionChain := []Attempt{{Type: "AttributeChanged", Stated: "I confront Mara over what she's hiding", TargetID: id.M}}

	out, err := orc.RunReactionBeat(ctx, id.World, id.K, reactionChain, held, 70000, "")
	if err != nil {
		t.Fatalf("RunReactionBeat: %v", err)
	}
	if out.HaltReason != "completed" {
		t.Fatalf("(f) reaction HaltReason = %q, want completed", out.HaltReason)
	}
	if resolve.calls < 1 {
		t.Fatalf("(f) referee was never consulted (resolve.calls=%d) — the combined ruling did not run", resolve.calls)
	}

	// The referee IS sighted: every combined-ruling prompt carries the secret (via recent_events'
	// canon summary for the participant M). If a marker is missing, the licensed referee was blinded.
	for i, p := range resolve.prompts {
		for _, m := range wallSecretMarkers {
			if !strings.Contains(p, m) {
				t.Fatalf("(f) referee prompt (call %d) MISSING marker %q — the truth-side referee was blinded to an involved party's canon:\n%s", i, m, p)
			}
		}
	}

	perceptionSubjectBackfill(t, ctx, pool, 60000)
}
