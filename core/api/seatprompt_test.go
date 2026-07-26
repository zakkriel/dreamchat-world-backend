package main

// Seat-boundary prompt assembly — the founder-gate fix. The decompose and narrate drivers drop
// req.Payload (they send only req.Prompt), so the perception lines and the candidate whitelist never
// reached the live model: decompose could not bind a real id, and the narrator, handed the bare
// instruction, invented a scene and broke frame. These tests pin the fix at the seat boundary:
//
//   (a) the DECOMPOSE seat's Prompt carries a candidate's uuid AND name AND the player's raw text,
//       with the raw text positioned AFTER the candidates (the mutable tail) — through the REAL HTTP
//       handler with a capturing driver (wall_test.go / station_e_exit_test.go pattern).
//   (b) the NARRATE seat's Prompt carries a real perception line's content AND the anti-invention
//       header marker — again through the real handler.
//   (c) a UNIT test: the narrate builder with ZERO lines still carries the sparse-is-correct rule and
//       fabricates no content section.
//
// (a) and (b) reuse setupWallWorld + runWallBeat: a fresh world with player K co-located with Mara
// (M) and Jonas (J) at L, so the decompose payload has a real candidate whitelist and the narrate
// payload has real perception lines.

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// (a) The decompose seat's Prompt carries a candidate uuid + name + the raw player text, text last.
func TestSeatPrompt_DecomposeCarriesCandidatesAndPlayerText(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true)

	// Benign beat: the raw text names Mara but binds her via the candidate whitelist, not the input.
	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`

	seats := runWallBeat(t, ctx, pool, id, beatText, chain)
	dec := seats[SeatDecompose.Name]
	if len(dec.reqs) == 0 {
		t.Fatalf("(a) decompose seat was never called")
	}
	p := dec.reqs[0].Prompt

	// Mara is a present candidate: BOTH her uuid and her name must reach the seat (so a real id binds).
	mIdx := strings.Index(p, id.M)
	if mIdx < 0 {
		t.Fatalf("(a) decompose Prompt missing candidate uuid %s — the whitelist did not reach the seat\n%s", id.M, p)
	}
	if !strings.Contains(p, "Mara") {
		t.Fatalf("(a) decompose Prompt missing candidate name Mara\n%s", p)
	}

	// The raw player text is present AND sits AFTER the candidate block (the mutable tail): the
	// cache-native layout the header promises, and what lets the model read the scene before the input.
	tIdx := strings.Index(p, beatText)
	if tIdx < 0 {
		t.Fatalf("(a) decompose Prompt missing the raw player text %q\n%s", beatText, p)
	}
	if tIdx < mIdx {
		t.Fatalf("(a) player text (idx %d) must sit AFTER the candidates (candidate uuid idx %d) — the tail\n%s", tIdx, mIdx, p)
	}

	// Keep SQL test 14's every-perception-has-a-subject invariant satisfied for this world's commits.
	perceptionSubjectBackfill(t, ctx, pool, 60000)
}

// (b) The narrate seat's Prompt carries a real perception line AND the anti-invention header marker.
func TestSeatPrompt_NarrateCarriesPerceptionsAndAntiInvention(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true)

	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`

	seats := runWallBeat(t, ctx, pool, id, beatText, chain)
	nar := seats[SeatNarrate.Name]
	if len(nar.reqs) == 0 {
		t.Fatalf("(b) narrate seat was never called")
	}
	req := nar.reqs[0]
	p := req.Prompt

	// The bounding header reached the model (it did NOT before the fix — the driver dropped everything
	// but the bare instruction, so the narrator invented a scene and offered out-of-world frames).
	if !strings.Contains(p, narrateAntiInventionMarker) {
		t.Fatalf("(b) narrate Prompt missing anti-invention marker %q — the narrator is not bounded\n%s", narrateAntiInventionMarker, p)
	}

	// The payload the seat actually received (its perception lines) must now appear IN the prompt — the
	// exact vector the driver used to drop. Assert on a line the seat truly held (known at runtime).
	if len(req.Payload.Lines) == 0 {
		t.Fatalf("(b) narrate payload had zero perception lines — the fixture is too sparse to prove the fix")
	}
	line := req.Payload.Lines[0]
	if !strings.Contains(p, line) {
		t.Fatalf("(b) narrate Prompt missing a real perception line %q — the payload never reached the model\n%s", line, p)
	}

	perceptionSubjectBackfill(t, ctx, pool, 60000)
}

// (c) Unit: the narrate builder with ZERO lines still carries the sparse-is-correct rule and
// fabricates no content — a sparse moment is a short narration, per the header.
func TestBuildNarratePrompt_ZeroLinesSparseNoFabrication(t *testing.T) {
	p := buildNarratePrompt(PerceptionPayload{}, "", nil) // no lines, no candidates, no baseline

	if !strings.HasPrefix(p, narrateSystemHeader) {
		t.Fatalf("(c) narrate prompt must open with the bounding header (the stable cache prefix)")
	}
	if !strings.Contains(p, narrateSparseRuleMarker) {
		t.Fatalf("(c) zero-line narrate prompt missing the sparse-is-correct rule %q", narrateSparseRuleMarker)
	}
	// No fabricated content section: with no candidates and no lines, no bullet/roster body is emitted.
	if strings.Contains(p, "\n- ") {
		t.Fatalf("(c) zero-line narrate prompt fabricated a content bullet — nothing must be invented to fill the space:\n%s", p)
	}
}

// rosterOf returns the PRESENT roster slice of the rendered body (between PRESENT and the first
// perception section), so a test can assert who is listed as present without matching the bodies.
func rosterOf(body string) string {
	presentIdx := strings.Index(body, "PRESENT:")
	if presentIdx < 0 {
		return ""
	}
	end := len(body)
	// YOU ARE (the viewer-identity block) is rendered BETWEEN PRESENT and the perception body — it bounds
	// the roster too, so an alias is never mistaken for a present actor.
	for _, marker := range []string{"YOU ARE:", "WHAT JUST HAPPENED", "RECENT BACKGROUND"} {
		if i := strings.Index(body, marker); i >= 0 && i < end {
			end = i
		}
	}
	return body[presentIdx:end]
}

// (d) Unit: the narrate builder renders a PLACE line from the location candidate (kind 'location'),
// BEFORE PRESENT, and the location does NOT leak into the PRESENT actor roster. This is the founder-gate
// orientation fix — without PLACE the narrator had no "where", and the location was listed as if present.
func TestBuildNarratePrompt_PlaceLineFromLocationCandidate(t *testing.T) {
	payload := PerceptionPayload{
		Candidates: []Candidate{
			{ID: "a1", Name: "Mara", Kind: "actor"},
			{ID: "l1", Name: "The Drowned Lantern", Kind: "location"},
		},
		Lines: []string{"Mara watches you from the bar"},
	}
	p := buildNarratePrompt(payload, "", nil)

	// Assert on the RENDERED BODY, not the fixed header (whose rulebook example itself contains the
	// strings "PLACE: The Drowned Lantern" and "PRESENT: Mara").
	body := strings.TrimPrefix(p, narrateSystemHeader)

	placeIdx := strings.Index(body, "PLACE: The Drowned Lantern")
	if placeIdx < 0 {
		t.Fatalf("(d) narrate prompt missing the rendered PLACE line with the location's name:\n%s", body)
	}
	presentIdx := strings.Index(body, "PRESENT:")
	if presentIdx < 0 || placeIdx > presentIdx {
		t.Fatalf("(d) PLACE (idx %d) must be rendered BEFORE PRESENT (idx %d):\n%s", placeIdx, presentIdx, body)
	}
	// The location must NOT appear inside the PRESENT roster (it is a place, not a present actor).
	roster := rosterOf(body)
	if strings.Contains(roster, "The Drowned Lantern") {
		t.Fatalf("(d) location leaked into the PRESENT roster — it belongs only under PLACE:\n%s", roster)
	}
	if !strings.Contains(roster, "Mara") {
		t.Fatalf("(d) actor Mara missing from the PRESENT roster:\n%s", roster)
	}
}

// (f) Defect A: the VIEWER is excluded from his own PRESENT roster. Given a candidate whose id IS the
// viewer, that name must not appear under PRESENT (the "…and Kade" to Kade bug); the others stay.
func TestBuildNarratePrompt_ViewerExcludedFromPresent(t *testing.T) {
	const viewer = "kade-id"
	payload := PerceptionPayload{
		Candidates: []Candidate{
			{ID: viewer, Name: "Kade", Kind: "actor"},
			{ID: "m1", Name: "Mara", Kind: "actor"},
			{ID: "l1", Name: "The Drowned Lantern", Kind: "location"},
		},
		Lines: []string{"Mara watches you from the bar"},
	}
	roster := rosterOf(strings.TrimPrefix(buildNarratePrompt(payload, viewer, nil), narrateSystemHeader))
	if strings.Contains(roster, "Kade") {
		t.Fatalf("(f) the viewer Kade leaked into his own PRESENT roster:\n%s", roster)
	}
	if !strings.Contains(roster, "Mara") {
		t.Fatalf("(f) the other present actor Mara was dropped from PRESENT:\n%s", roster)
	}
}

// (g) Defect B: delta-first. A line whose perception_id is in the pre-beat baseline is RECENT
// BACKGROUND (never re-narrated); a line whose id is new is WHAT JUST HAPPENED. The window is NOT
// re-rendered wholesale every beat.
func TestBuildNarratePrompt_DeltaFirstSplitsNewFromBackground(t *testing.T) {
	payload := PerceptionPayload{
		Lines:   []string{"You stepped into the Drowned Lantern.", "Mara sets a tankard on the bar."},
		LineIDs: []string{"p-old", "p-new"},
	}
	preIDs := map[string]bool{"p-old": true} // the "stepped in" line was already held before this beat
	body := strings.TrimPrefix(buildNarratePrompt(payload, "", preIDs), narrateSystemHeader)

	jh := strings.Index(body, "WHAT JUST HAPPENED")
	bg := strings.Index(body, "RECENT BACKGROUND")
	if jh < 0 || bg < 0 {
		t.Fatalf("(g) expected both WHAT JUST HAPPENED and RECENT BACKGROUND sections:\n%s", body)
	}
	if jh > bg {
		t.Fatalf("(g) WHAT JUST HAPPENED must precede RECENT BACKGROUND:\n%s", body)
	}
	justHappened := body[jh:bg]
	background := body[bg:]
	if !strings.Contains(justHappened, "Mara sets a tankard on the bar.") {
		t.Fatalf("(g) the NEW perception is missing from WHAT JUST HAPPENED:\n%s", justHappened)
	}
	if strings.Contains(justHappened, "You stepped into the Drowned Lantern.") {
		t.Fatalf("(g) the already-known line was re-narrated as new (the ×3 bug):\n%s", justHappened)
	}
	if !strings.Contains(background, "You stepped into the Drowned Lantern.") {
		t.Fatalf("(g) the already-known line must appear under RECENT BACKGROUND:\n%s", background)
	}
}

// (h) Defect B part 2: the location candidate's scene description renders on the PLACE line.
func TestBuildNarratePrompt_PlaceDescriptionRendered(t *testing.T) {
	const desc = "Low beams, salt-rot, one hearth, a bar with a hatch, a back door to the alley."
	payload := PerceptionPayload{
		Candidates: []Candidate{
			{ID: "l1", Name: "The Drowned Lantern", Kind: "location", Description: desc},
		},
	}
	body := strings.TrimPrefix(buildNarratePrompt(payload, "", nil), narrateSystemHeader)
	if !strings.Contains(body, "PLACE: The Drowned Lantern — "+desc) {
		t.Fatalf("(h) PLACE line missing the rendered scene description:\n%s", body)
	}
}

// (i) Unit: viewer-relative rendering inputs. The builder renders a YOU ARE block from the payload's
// ViewerAliases (the viewer's descriptor + any self-known name), positioned BETWEEN the PRESENT roster
// and the perception body, and the fixed header carries the viewer-relative RULE marker — so the model
// gets both the inputs (YOU ARE / PRESENT) and the rule to bind a third-person reference to the viewer.
func TestBuildNarratePrompt_ViewerIdentityBlockAndRule(t *testing.T) {
	const desc = "a young stranger, dark-haired"
	payload := PerceptionPayload{
		Candidates: []Candidate{
			{ID: "j1", Name: "the muscle", Kind: "actor"},
			{ID: "l1", Name: "The Drowned Lantern", Kind: "location"},
		},
		Lines:         []string{"Jonas steps between Mara and the stranger"},
		ViewerAliases: []string{desc},
	}
	p := buildNarratePrompt(payload, "kade-id", nil)

	// The viewer-relative rule reached the model (from the fixed header).
	if !strings.Contains(p, narrateViewerRelativeMarker) {
		t.Fatalf("(i) narrate prompt missing the viewer-relative rule marker %q\n%s", narrateViewerRelativeMarker, p)
	}
	// Assert on the RENDERED BODY, not the fixed header (whose example text itself mentions YOU ARE).
	body := strings.TrimPrefix(p, narrateSystemHeader)
	youIdx := strings.Index(body, "YOU ARE: "+desc)
	if youIdx < 0 {
		t.Fatalf("(i) narrate prompt missing the YOU ARE block with the seeded descriptor:\n%s", body)
	}
	presentIdx := strings.Index(body, "PRESENT:")
	whatIdx := strings.Index(body, "WHAT JUST HAPPENED")
	if presentIdx < 0 || presentIdx > youIdx {
		t.Fatalf("(i) YOU ARE (idx %d) must be rendered AFTER PRESENT (idx %d):\n%s", youIdx, presentIdx, body)
	}
	if whatIdx < 0 || youIdx > whatIdx {
		t.Fatalf("(i) YOU ARE (idx %d) must be rendered BEFORE the perception body (idx %d):\n%s", youIdx, whatIdx, body)
	}
	// The descriptor is a viewer ALIAS, not a present actor — it must not sit in the PRESENT roster.
	if roster := rosterOf(body); strings.Contains(roster, desc) {
		t.Fatalf("(i) the viewer's descriptor leaked into the PRESENT roster — it belongs under YOU ARE:\n%s", roster)
	}
}

// (j) Unit: a viewer with no descriptor and no self-known name has no aliases ⇒ NO YOU ARE block (no
// empty header line to confuse the model).
func TestBuildNarratePrompt_NoViewerAliasesOmitsYouAre(t *testing.T) {
	payload := PerceptionPayload{
		Candidates: []Candidate{{ID: "l1", Name: "The Drowned Lantern", Kind: "location"}},
		Lines:      []string{"Mara watches you from the bar"},
	}
	body := strings.TrimPrefix(buildNarratePrompt(payload, "", nil), narrateSystemHeader)
	if strings.Contains(body, "YOU ARE:") {
		t.Fatalf("(j) YOU ARE rendered though the viewer has no aliases:\n%s", body)
	}
}

// (k) FLOW — viewer-relative rendering end to end, through the REAL HTTP path (setupWallWorld +
// runWallBeat, the wall-test family). A perception the VIEWER holds names the NPC by his CANONICAL name
// ("Jonas") AND refers to the viewer in the THIRD PERSON ("the stranger") — the exact live defect. The
// captured narrate prompt must carry everything the model needs to relabel BOTH references: (a) YOU ARE
// with the viewer's descriptor (so "the stranger" → "you"), (b) the PRESENT roster with the VIEWER's
// label for the NPC ("the muscle", never "Jonas"), and the viewer-relative RULE. We assert the prompt's
// INPUTS + RULE, not the model's output — the relabel itself is the model's job.
func TestSeatPrompt_NarrateCarriesViewerIdentityAndPresentLabel(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupWallWorld(t, ctx, pool, true)

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v\n%s", err, sql)
		}
	}

	// The viewer wears a stranger's descriptor (his YOU ARE alias). The NPC wears one too, and the viewer
	// knows no name for him — so his PRESENT label is the descriptor "the muscle", never "Jonas".
	const viewerDesc = "a young stranger, dark-haired"
	mustExec(`UPDATE actor_state SET attrs = jsonb_set(attrs,'{descriptor}', to_jsonb($3::text))
	          WHERE entity_id=$1 AND world_id=$2`, id.K, id.World, viewerDesc)
	mustExec(`UPDATE actor_state SET attrs = jsonb_set(attrs,'{descriptor}','"the muscle"'::jsonb)
	          WHERE entity_id=$1 AND world_id=$2`, id.J, id.World)

	// A perception the VIEWER holds that names the NPC canonically AND refers to the viewer in the third
	// person — the line the relabel must fix. Accepted source event + subject (invariant-clean). Tick
	// 60003 sits inside the beat's recency window (max_tick+1) and clear of the accepted-order unique
	// index (uq_ce_accepted_order) via a distinct beat_seq.
	const perceptionLine = "Jonas moves between Mara and the stranger"
	var srcEvt string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&srcEvt); err != nil {
		t.Fatalf("mint src evt: %v", err)
	}
	mustExec(`INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
	          VALUES ($1,$2,'observation','viewer wind-up',60003,7,'accepted',now(),'public','fast_path')`, srcEvt, id.World)
	mustExec(`INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier) VALUES ($1,$2,'actor','instigator')`, srcEvt, id.K)
	var pid string
	if err := pool.QueryRow(ctx,
		`INSERT INTO perception_record (world_id,holder_id,source_event_id,content,epistemic_type,acquired_tick,valid_tick)
		 VALUES ($1,$2,$3,$4,'direct',60003,60003) RETURNING perception_id`,
		id.World, id.K, srcEvt, perceptionLine).Scan(&pid); err != nil {
		t.Fatalf("seed viewer perception: %v", err)
	}
	mustExec(`INSERT INTO perception_subject (perception_id,entity_id,world_id) VALUES ($1,$2,$3)`, pid, id.K, id.World)

	const beatText = "I press Mara for the truth"
	chain := `[{"type":"Communicated","stated":"I press Mara for the truth","listener_id":"` + id.M + `","content":"tell me what happened"}]`
	seats := runWallBeat(t, ctx, pool, id, beatText, chain)

	nar := seats[SeatNarrate.Name]
	if len(nar.reqs) == 0 {
		t.Fatalf("(k) narrate seat was never called")
	}
	p := nar.reqs[0].Prompt

	// The viewer-relative RULE reached the model.
	if !strings.Contains(p, narrateViewerRelativeMarker) {
		t.Fatalf("(k) narrate prompt missing the viewer-relative rule marker %q\n%s", narrateViewerRelativeMarker, p)
	}
	body := strings.TrimPrefix(p, narrateSystemHeader)

	// (a) YOU ARE carries the viewer's descriptor — the alias that lets "the stranger" resolve to "you".
	if !strings.Contains(body, "YOU ARE: "+viewerDesc) {
		t.Fatalf("(k) narrate prompt missing YOU ARE with the viewer's descriptor %q\n%s", viewerDesc, body)
	}
	// (b) PRESENT carries the VIEWER's label for the NPC ("the muscle"), never his canonical name.
	roster := rosterOf(body)
	if !strings.Contains(roster, "the muscle") {
		t.Fatalf("(k) PRESENT roster missing the viewer's label \"the muscle\" for the NPC:\n%s", roster)
	}
	if strings.Contains(roster, "Jonas") {
		t.Fatalf("(k) the NPC's canonical name \"Jonas\" leaked into the PRESENT roster — the viewer's label must stand:\n%s", roster)
	}
	// The line needing the relabel is actually in the prompt (its canonical name AND third-person viewer
	// reference), so the inputs above are exactly what lets the model fix it.
	if !strings.Contains(body, perceptionLine) {
		t.Fatalf("(k) the perception line needing relabel is missing from the narrate prompt:\n%s", body)
	}

	perceptionSubjectBackfill(t, ctx, pool, 60000)
}

// (e) The payload's Lines are a BOUNDED RECENT WINDOW, not the holder's whole remembered life. A viewer
// with >20 old perceptions plus a run of recent ones gets back only the newest recencyMaxRows within
// recencyTickWindow ticks of the newest row, oldest-first. Live bug: the narrator recited a full travel
// log every beat. Fixture uses controlled acquired_ticks; the viewer is placed via a state_mutation
// (trigger-projected → replay_0A stays diff-free) and every perception gets a subject (SQL test 14).
func TestPayload_RecencyWindow_KeepsRecentDropsOldBeyondWindow(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	var world, viewer, loc, moveEvt, srcEvt string
	if err := pool.QueryRow(ctx,
		`SELECT gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid()`,
	).Scan(&world, &viewer, &loc, &moveEvt, &srcEvt); err != nil {
		t.Fatalf("mint ids: %v", err)
	}

	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("exec: %v\n%s", err, sql)
		}
	}

	// Registry: the viewer (actor) + the location entity.
	mustExec(`INSERT INTO entity_registry (entity_id,world_id,entity_kind,canonical_name) VALUES
	          ($1,$3,'actor','Recency Viewer'),($2,$3,'location','Recency Room')`, viewer, loc, world)

	// Place the viewer at the location via a move event + state_mutation (trigger projects actor_state).
	// A DIRECT actor_state insert would have no backing mutation and replay_0A() would fail to rebuild it.
	mustExec(`INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
	          VALUES ($1,$2,'move','place recency viewer',1,0,'accepted',now(),'public','fast_path')`, moveEvt, world)
	mustExec(`INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier) VALUES ($1,$2,'actor','instigator')`, moveEvt, viewer)
	mustExec(`INSERT INTO state_mutation (world_id,event_id,entity_id,entity_kind,attribute_path,new_value,valid_from_tick,valid_from_seq)
	          VALUES ($1,$2,$3,'actor','attrs.location_id',to_jsonb($4::text),1,0)`, world, moveEvt, viewer, loc)

	// One ACCEPTED source event for the perceptions (so none is an orphan under SQL test 50).
	mustExec(`INSERT INTO canon_event (event_id,world_id,event_type,summary,in_world_tick,beat_seq,status,accepted_at,visibility_scope,origin)
	          VALUES ($1,$2,'observation','recency source',1,1,'accepted',now(),'public','fast_path')`, srcEvt, world)
	mustExec(`INSERT INTO event_participant (event_id,entity_id,entity_kind,role_qualifier) VALUES ($1,$2,'actor','instigator')`, srcEvt, viewer)

	// Each perception is held by the viewer (→ visible), sourced to the accepted event, with a subject.
	insertPerc := func(tick int, content string) {
		var pid string
		if err := pool.QueryRow(ctx,
			`INSERT INTO perception_record (world_id,holder_id,source_event_id,content,epistemic_type,acquired_tick,valid_tick)
			 VALUES ($1,$2,$3,$4,'direct',$5,$5) RETURNING perception_id`,
			world, viewer, srcEvt, content, tick).Scan(&pid); err != nil {
			t.Fatalf("insert perception %s: %v", content, err)
		}
		mustExec(`INSERT INTO perception_subject (perception_id,entity_id,world_id) VALUES ($1,$2,$3)`, pid, viewer, world)
	}
	// OLD block: ticks 100..124 (25 rows) — far below the newest, must be dropped by the tick window.
	for tick := 100; tick <= 124; tick++ {
		insertPerc(tick, fmt.Sprintf("OLD-%d", tick))
	}
	// RECENT block: ticks 971..1000 (30 rows, all within 50 of max=1000) — the cap keeps the newest 20.
	for tick := 971; tick <= 1000; tick++ {
		insertPerc(tick, fmt.Sprintf("REC-%d", tick))
	}

	h := &beatHandler{pool: pool, dbg: true}
	p, err := h.payload(ctx, world, viewer)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}

	// Cap: at most recencyMaxRows lines.
	if len(p.Lines) != recencyMaxRows {
		t.Fatalf("want exactly recencyMaxRows=%d lines (cap), got %d: %v", recencyMaxRows, len(p.Lines), p.Lines)
	}
	// Oldest-first + cap boundary: the kept 20 are ticks 981..1000.
	if p.Lines[0] != "REC-981" {
		t.Fatalf("oldest kept line = %q, want REC-981 (newest 20, oldest-first)\n%v", p.Lines[0], p.Lines)
	}
	if p.Lines[len(p.Lines)-1] != "REC-1000" {
		t.Fatalf("newest line = %q, want REC-1000\n%v", p.Lines[len(p.Lines)-1], p.Lines)
	}
	for _, l := range p.Lines {
		if strings.HasPrefix(l, "OLD-") {
			t.Fatalf("recency window leaked an OLD line %q — rows older than the tick window must be dropped\n%v", l, p.Lines)
		}
		if l == "REC-971" || l == "REC-980" {
			t.Fatalf("cap failed: kept %q, older than the newest %d rows\n%v", l, recencyMaxRows, p.Lines)
		}
	}
	if !containsStr(p.Lines, "REC-1000") {
		t.Fatalf("the recent perception REC-1000 is missing from the payload\n%v", p.Lines)
	}
}

func containsStr(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
