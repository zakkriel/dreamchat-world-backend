package main

import "testing"

func TestRulingTherefore(t *testing.T) {
	// The recorded failure class: reasoning concludes one way, outcome flips.
	flip := `{"reasoning":"Jonas her protector is gone; she is more vulnerable","therefore":"succeeds","outcome":{"kind":"bounce","reason":"cannot"}}`
	if _, err := DecodeAndValidateRuling(flip); err == nil {
		t.Fatal("therefore=succeeds with outcome=bounce accepted — the flip bug survives")
	}
	ok := `{"reasoning":"she is guarded; the press does not land","therefore":"fails","outcome":{"kind":"resolved","events":[{"type":"AttributeChanged","summary":"Mara hardens and deflects","participant_ids":["33333333-3333-3333-3333-333333333333"],"visible":true}]}}`
	r, err := DecodeAndValidateRuling(ok)
	if err != nil {
		t.Fatalf("valid failure ruling rejected: %v", err)
	}
	if r.Outcome.Kind != "resolved" || len(r.Outcome.Events) != 1 {
		t.Fatalf("bad decode: %+v", r)
	}
	// A failure IS an outcome and writes canon; impossibility bounces with none.
	imp := `{"reasoning":"he has no wings; flight is not possible for this actor","therefore":"impossible","outcome":{"kind":"bounce","reason":"cannot fly"}}`
	if _, err := DecodeAndValidateRuling(imp); err != nil {
		t.Fatalf("bounce ruling rejected: %v", err)
	}
}
