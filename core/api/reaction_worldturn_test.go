package main

import (
	"context"
	"strings"
	"testing"
)

// A reaction beat must still give the world its turn.
//
// Founder-reported, reproduced on live Railway AFTER #83 landed: "Mara, I want to rest here" bound
// Mara as the listener and Mara never answered. The trace showed why — a held telegraph was pending
// from the previous beat, so the input took RunReactionBeat, which went ruling → return. No cognition
// seat ran for ANY mind: no batch call, no ADDRESSED line, no decision. The only NPC motion the player
// saw was the held act resolving, which is indistinguishable from the world ignoring him.
//
// The scripted batch driver here answers for the addressed NPC. If the world turn is skipped, its
// decision never runs and the assertion fails — which is exactly what happened before the fix.
func TestReactionBeat_RunsTheWorldTurnSoTheAddressedNPCCanAnswer(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupReactionWorld(t, ctx, pool)

	// ── Beat 1: an NPC telegraphs, so the next input arrives as a REACTION. ──
	baseTick := reactionBaseTick(t, ctx, pool, id.World)
	tele := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.J +
		`","decision":{"commit_kind":"telegraph","attempt":{"type":"ActorMoved","stated":"Jonas pushes off the bar, moving to cut in","to_target_id":"` + id.L2 + `"}}}]`}
	orc1 := &Orchestrator{DB: pool, Resolve: NewFakeResolveDriver(), CognitionBatch: tele,
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`}, WorldActor: NewFakeWorldActorDriver()}
	out1, err := orc1.RunBeat(ctx, id.World, id.P, []Attempt{
		{Type: "ActorMoved", Stated: "I cross to the bar", ToTargetID: id.L2},
	}, baseTick, nil)
	if err != nil {
		t.Fatalf("beat 1: %v", err)
	}
	if out1.HaltReason != "telegraph" {
		t.Fatalf("fixture needs a pending telegraph; halt = %q", out1.HaltReason)
	}
	held, err := pendingHeldOutcomes(ctx, pool, id.World)
	if err != nil || len(held) != 1 {
		t.Fatalf("pending held = %d (err=%v), want 1", len(held), err)
	}

	// ── Beat 2: the player SPEAKS TO an NPC while the held act is pending. ──
	reactTick := reactionBaseTick(t, ctx, pool, id.World)
	const reply = "the room's yours if the coin is"
	answering := &scriptedCognitionDriver{name: "answering-batch", body: `[{"actor_id":"` + id.W +
		`","decision":{"commit_kind":"commit","attempt":{"type":"Communicated","stated":"she answers the stranger","listener_id":"` +
		id.P + `","content":"` + reply + `"}}}]`}
	orc2 := &Orchestrator{DB: pool, Resolve: &capturingResolveDriver{name: "capture-resolve",
		ruling: validRulingJSON(id.P, id.J, "The cut-in is checked.", "A scuffle at the bar.")},
		CognitionBatch:    answering,
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver()}

	out2, err := orc2.RunReactionBeat(ctx, id.World, id.P,
		[]Attempt{{Type: "Communicated", Stated: "I ask her for a room", ListenerID: id.W, Content: "can I rest here?"}},
		held, reactTick, "", nil)
	if err != nil {
		t.Fatalf("beat 2 RunReactionBeat: %v", err)
	}

	// The addressed NPC's answer must be canon. Before the fix, RunReactionBeat returned straight after
	// the ruling and this event never existed.
	var spoken string
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(payload->>'spoken','') FROM canon_event
		  WHERE world_id=$1 AND event_type='Communicated' AND in_world_tick=$2
		    AND payload->>'spoken' = $3`, id.World, reactTick, reply).Scan(&spoken); err != nil {
		t.Fatalf("the addressed NPC never got a turn in the reaction beat (no answer in canon): %v", err)
	}
	if spoken != reply {
		t.Fatalf("spoken = %q, want %q", spoken, reply)
	}
	if out2.HaltReason != "completed" {
		t.Fatalf("halt = %q, want completed", out2.HaltReason)
	}
	if len(out2.Committed) < 2 {
		t.Fatalf("committed %d events, want the ruling AND the NPC's answer", len(out2.Committed))
	}
}

// A fresh telegraph inside a reaction beat must still end the beat, exactly as it does on the ordinary
// path — the world turn was added, not given different rules.
func TestReactionBeat_FreshTelegraphStillEndsTheBeat(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupReactionWorld(t, ctx, pool)

	baseTick := reactionBaseTick(t, ctx, pool, id.World)
	tele := &scriptedCognitionDriver{name: "scripted-batch", body: `[{"actor_id":"` + id.J +
		`","decision":{"commit_kind":"telegraph","attempt":{"type":"ActorMoved","stated":"Jonas pushes off the bar, moving to cut in","to_target_id":"` + id.L2 + `"}}}]`}
	orc1 := &Orchestrator{DB: pool, Resolve: NewFakeResolveDriver(), CognitionBatch: tele,
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`}, WorldActor: NewFakeWorldActorDriver()}
	if _, err := orc1.RunBeat(ctx, id.World, id.P, []Attempt{
		{Type: "ActorMoved", Stated: "I cross to the bar", ToTargetID: id.L2}}, baseTick, nil); err != nil {
		t.Fatalf("beat 1: %v", err)
	}
	held, _ := pendingHeldOutcomes(ctx, pool, id.World)

	reactTick := reactionBaseTick(t, ctx, pool, id.World)
	orc2 := &Orchestrator{DB: pool, Resolve: &capturingResolveDriver{name: "capture-resolve",
		ruling: validRulingJSON(id.P, id.J, "The cut-in is checked.", "A scuffle at the bar.")},
		CognitionBatch: &scriptedCognitionDriver{name: "telegraph-again", body: `[{"actor_id":"` + id.J +
			`","decision":{"commit_kind":"telegraph","attempt":{"type":"ActorMoved","stated":"Jonas winds up again","to_target_id":"` + id.L2 + `"}}}]`},
		CognitionIsolated: &scriptedCognitionDriver{name: "quiet-iso", body: `[]`},
		WorldActor:        NewFakeWorldActorDriver()}

	out, err := orc2.RunReactionBeat(ctx, id.World, id.P,
		[]Attempt{{Type: "Communicated", Stated: "I ask her for a room", ListenerID: id.W, Content: "can I rest here?"}},
		held, reactTick, "", nil)
	if err != nil {
		t.Fatalf("RunReactionBeat: %v", err)
	}
	if out.HaltReason != "telegraph" {
		t.Fatalf("halt = %q, want telegraph — a fresh wind-up ends the reaction beat too", out.HaltReason)
	}
	if len(out.Telegraphs) == 0 || !strings.Contains(out.Telegraphs[0], "winds up again") {
		t.Fatalf("telegraphs = %v, want the fresh wind-up surfaced", out.Telegraphs)
	}
}
