package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// mintRaw is a tiny helper: a mint envelope literal → json.RawMessage.
func mintRaw(s string) json.RawMessage { return json.RawMessage(s) }

// seeded is the "walk exists" starting vocabulary every world has (FINAL §2).
func seeded() map[string]bool { return map[string]bool{"walk": true} }

// (a) baseSpeed:0 → violation. base_speed must be a positive number (§2: speed 0 is meaningless as a
// TYPE default — the -100% floor produces 0 dynamically; a 0-speed type is a shape failure).
func TestValidateMints_A_MovementBaseSpeedZero(t *testing.T) {
	v := validateMints([]json.RawMessage{mintRaw(`{"movementTypeId":"sprint","baseSpeed":0}`)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(a) baseSpeed:0 must violate; got pass")
	}
}

// (b) modifierPercent:-150 → violation. Floor is -100% (below that is negative speed, meaningless, §2).
func TestValidateMints_B_ModifierBelowFloor(t *testing.T) {
	m := `{"statusTypeId":"crushed","actionType":"move","movementModifiers":[{"movementTypeId":"walk","modifierPercent":-150}]}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(b) modifierPercent:-150 must violate; got pass")
	}
}

// (c) modifierPercent:900 → PASS. NO upper cap (founder ruling, §2): +900% haste is legal data; the
// three nets (§8), not a numeric bound, guard a garbage mint.
func TestValidateMints_C_ModifierNoUpperCap(t *testing.T) {
	m := `{"statusTypeId":"hasted","actionType":"move","movementModifiers":[{"movementTypeId":"walk","modifierPercent":900}]}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) != 0 {
		t.Fatalf("(c) modifierPercent:900 must PASS (no upper cap); got %v", v)
	}
}

// (d) modifier referencing an unknown movement type → violation (mint-ordering, §8): a modifier may only
// reference movement types that already exist (seeded or minted earlier in THIS ruling).
func TestValidateMints_D_ModifierUnknownMovementType(t *testing.T) {
	m := `{"statusTypeId":"limping","actionType":"move","movementModifiers":[{"movementTypeId":"fly","modifierPercent":-30}]}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(d) modifier referencing unknown 'fly' must violate (ordering); got pass")
	}
}

// (d2) mint-ordering POSITIVE: a movement type minted EARLIER in the same slice satisfies a later
// modifier's reference (seeded ∪ minted-earlier-in-this-ruling).
func TestValidateMints_D2_ModifierRefersEarlierMint(t *testing.T) {
	mts := `{"movementTypeId":"climb","baseSpeed":0.4}`
	mod := `{"statusTypeId":"sure_footed","actionType":"move","movementModifiers":[{"movementTypeId":"climb","modifierPercent":20}]}`
	v := validateMints([]json.RawMessage{mintRaw(mts), mintRaw(mod)}, seeded())
	if len(v) != 0 {
		t.Fatalf("(d2) modifier referencing a type minted earlier in the slice must PASS; got %v", v)
	}
}

// (e) container max_room:300, size:1 (> 4^0 = 1) → violation. Mundane container max_room ≤ 4^(size-1) (§4).
func TestValidateMints_E_ContainerMaxRoomExceeds(t *testing.T) {
	v := validateMints([]json.RawMessage{mintRaw(`{"size":1,"maxRoom":300}`)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(e) container max_room:300 with size:1 (>4^0=1) must violate; got pass")
	}
}

// (e2) a well-sized container PASSES: size:3 → 4^2 = 16; max_room 10 ≤ 16.
func TestValidateMints_E2_ContainerWithinBound(t *testing.T) {
	v := validateMints([]json.RawMessage{mintRaw(`{"size":3,"maxRoom":10}`)}, seeded())
	if len(v) != 0 {
		t.Fatalf("(e2) container max_room:10 with size:3 (≤4^2=16) must PASS; got %v", v)
	}
}

// (f) valid climb/0.4 movement-type mint → pass.
func TestValidateMints_F_ValidMovementMint(t *testing.T) {
	v := validateMints([]json.RawMessage{mintRaw(`{"movementTypeId":"climb","baseSpeed":0.4}`)}, seeded())
	if len(v) != 0 {
		t.Fatalf("(f) valid climb/0.4 mint must PASS; got %v", v)
	}
}

// (g) a coordinate outside the parent extent → violation (§3: a minted coordinate must lie within the
// parent's extent). Envelope carries the parent's extent inline so validation stays DB-free.
func TestValidateMints_G_CoordinateOutsideExtent(t *testing.T) {
	m := `{"locationId":"11111111-1111-1111-1111-111111111111","parentLocationId":"22222222-2222-2222-2222-222222222222","coordinate":{"x":5000,"y":0},"parentExtent":{"w":2000,"h":2000}}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(g) coordinate x:5000 outside extent w:2000 must violate; got pass")
	}
}

// (g2) a coordinate INSIDE the parent extent → pass.
func TestValidateMints_G2_CoordinateInsideExtent(t *testing.T) {
	m := `{"locationId":"11111111-1111-1111-1111-111111111111","parentLocationId":"22222222-2222-2222-2222-222222222222","coordinate":{"x":100,"y":50},"parentExtent":{"w":2000,"h":2000}}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) != 0 {
		t.Fatalf("(g2) coordinate {100,50} inside extent {2000,2000} must PASS; got %v", v)
	}
}

// (h) a parent_location_id forming a cycle → violation. Simplest self-contained case: a self-parent.
func TestValidateMints_H_SelfParentCycle(t *testing.T) {
	m := `{"locationId":"11111111-1111-1111-1111-111111111111","parentLocationId":"11111111-1111-1111-1111-111111111111"}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(h) self-parent must violate (cycle); got pass")
	}
}

// (h2) a two-node cycle among co-minted locations → violation (A→B→A).
func TestValidateMints_H2_TwoNodeCycle(t *testing.T) {
	a := `{"locationId":"aaaaaaaa-0000-0000-0000-000000000001","parentLocationId":"aaaaaaaa-0000-0000-0000-000000000002"}`
	b := `{"locationId":"aaaaaaaa-0000-0000-0000-000000000002","parentLocationId":"aaaaaaaa-0000-0000-0000-000000000001"}`
	v := validateMints([]json.RawMessage{mintRaw(a), mintRaw(b)}, seeded())
	if len(v) == 0 {
		t.Fatalf("(h2) A→B→A cycle among co-minted locations must violate; got pass")
	}
}

// (h3) a parent pointing OUT of the slice (an existing committed location) is NOT a cycle → pass. The
// committed hierarchy is kept acyclic by the SQL guard; the slice walk terminates safely at a foreign id.
func TestValidateMints_H3_ParentOutsideSlice(t *testing.T) {
	m := `{"locationId":"11111111-1111-1111-1111-111111111111","parentLocationId":"99999999-9999-9999-9999-999999999999","coordinate":{"x":1,"y":1},"parentExtent":{"w":10,"h":10}}`
	v := validateMints([]json.RawMessage{mintRaw(m)}, seeded())
	if len(v) != 0 {
		t.Fatalf("(h3) parent outside the slice must PASS (not a cycle); got %v", v)
	}
}

// Empty mints → no violations (the common ruling: nothing minted).
func TestValidateMints_EmptyIsPass(t *testing.T) {
	if v := validateMints(nil, seeded()); len(v) != 0 {
		t.Fatalf("empty mints must PASS; got %v", v)
	}
}

// An unrecognizable mint shape → violation (defensive: the discriminator matched nothing).
func TestValidateMints_UnknownShape(t *testing.T) {
	v := validateMints([]json.RawMessage{mintRaw(`{"foo":"bar"}`)}, seeded())
	if len(v) == 0 {
		t.Fatalf("unknown mint shape must violate; got pass")
	}
	if !strings.Contains(strings.Join(v, " "), "mint 0") {
		t.Fatalf("violation should be indexed (mint 0 ...); got %v", v)
	}
}
