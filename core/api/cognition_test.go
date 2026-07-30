package main

import "testing"

func TestNPCDecisionsValidateActorsAndShape(t *testing.T) {
	present := []string{"33333333-3333-3333-3333-333333333333"}
	ok := `[{"actor_id":"33333333-3333-3333-3333-333333333333","decision":"none"}]`
	ds, err := DecodeAndValidateNPCDecisions(ok, present)
	if err != nil || len(ds) != 1 || ds[0].Reaction != nil {
		t.Fatalf("none-decision: %v %+v", err, ds)
	}
	ghost := `[{"actor_id":"99999999-9999-9999-9999-999999999999","decision":"none"}]`
	if _, err := DecodeAndValidateNPCDecisions(ghost, present); err == nil {
		t.Fatal("decision for a non-present actor accepted")
	}
	tele := `[{"actor_id":"33333333-3333-3333-3333-333333333333","decision":{"commit_kind":"telegraph","attempt":{"type":"ActorMoved","stated":"Jonas pushes off the bar, moving to cut in","to_target_id":"11111111-1111-1111-1111-111111111111"}}}]`
	ds, err = DecodeAndValidateNPCDecisions(tele, present)
	if err != nil || ds[0].Reaction == nil || ds[0].Reaction.CommitKind != "telegraph" {
		t.Fatalf("telegraph decision: %v %+v", err, ds)
	}
}

// TestNPCDecisionsRejectQuery guards the Task 4 gap: NPCs act, they never ask. A QUERY-typed
// NPC decision must be REJECTED by the belt, not fall through to applyNPCDecisions' default
// case (which would otherwise route a question into o.adjudicate as if it were an action).
func TestNPCDecisionsRejectQuery(t *testing.T) {
	present := []string{"33333333-3333-3333-3333-333333333333"}
	query := `[{"actor_id":"33333333-3333-3333-3333-333333333333","decision":{"commit_kind":"commit","attempt":{"type":"QUERY","stated":"Who is that at the bar?","query_target_ids":["11111111-1111-1111-1111-111111111111"]}}}]`
	if _, err := DecodeAndValidateNPCDecisions(query, present); err == nil {
		t.Fatal("QUERY-typed NPC decision accepted — NPCs must never ask")
	}
}

// TestNPCDecisionsRejectUnresolved closes the pre-existing untested UNRESOLVED exclusion
// alongside the QUERY one above: UNRESOLVED is a player-decompose-only clarify sentinel, not
// a valid NPC reaction.
func TestNPCDecisionsRejectUnresolved(t *testing.T) {
	present := []string{"33333333-3333-3333-3333-333333333333"}
	unresolved := `[{"actor_id":"33333333-3333-3333-3333-333333333333","decision":{"commit_kind":"commit","attempt":{"type":"UNRESOLVED","stated":"the door","reference":"the door","candidate_ids":["11111111-1111-1111-1111-111111111111","22222222-2222-2222-2222-222222222222"]}}}]`
	if _, err := DecodeAndValidateNPCDecisions(unresolved, present); err == nil {
		t.Fatal("UNRESOLVED-typed NPC decision accepted — NPCs must never emit a clarify sentinel")
	}
}
