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
