package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// These unit tests pin the cache-native prompt layout (RULINGS-2026-07-23 §5): a stable
// prefix (header → scene → minds) then, for the isolated seat only, the secret block, then
// the append-only public moment, then the mutable tail (imminent attempt). Order is the
// contract: providers cache the shared prefix, so anything that changes per call must sit at
// the TAIL. The five section markers must appear in exactly this order.

// A small helper: assert a appears strictly before b in s (both must be present).
func assertOrder(t *testing.T, s, a, b string) {
	t.Helper()
	ia, ib := strings.Index(s, a), strings.Index(s, b)
	if ia < 0 {
		t.Fatalf("marker %q missing from prompt", a)
	}
	if ib < 0 {
		t.Fatalf("marker %q missing from prompt", b)
	}
	if ia >= ib {
		t.Fatalf("marker %q (idx %d) must appear before %q (idx %d)", a, ia, b, ib)
	}
}

func sampleScene() sceneInfo {
	return sceneInfo{
		LocationName: "the Drowned Lantern",
		Tension:      "tense",
		Present: []rosterEntry{
			{ID: playerID, Name: "Player"},
			{ID: maraID, Name: "Mara"},
			{ID: jonasID, Name: "Jonas"},
		},
	}
}

func sampleImminent() Attempt {
	return Attempt{Type: "Communicated", Stated: "I slip her the note", ListenerID: maraID, Content: "meet me at the dock"}
}

// Prompt-content pin: the founder-gate fix's minds-side half (companion to the referee's
// dialogue-is-its-own-event rule in resolve.txt). A mind whose decision includes speaking words
// must commit a Communicated attempt, never a quote folded into an ActorMoved's or
// AttributeChanged's stated text. This test pins the rule marker reaches the cognition header.
func TestCognitionSystemHeader_SpeakingIsItsOwnCommunicatedAttempt(t *testing.T) {
	if !strings.Contains(cognitionSystemHeader, cognitionSpeechIsOwnAttemptMarker) {
		t.Fatalf("cognition.txt missing the speaking-is-its-own-attempt rule marker %q — a mind's decision could still bury spoken words inside an ActorMoved/AttributeChanged stated text:\n%s", cognitionSpeechIsOwnAttemptMarker, cognitionSystemHeader)
	}
}

// (a) Batch prompt: the five section markers in layout order.
func TestBuildBatchPrompt_SectionOrder(t *testing.T) {
	minds := []npcMind{{ID: jonasID, Name: "Jonas", Traits: json.RawMessage(`{"wary":0.7}`), Malleability: 0.4}}
	moment := []momentLine{{Content: "a torch flares in the doorway", Tick: 700}}
	p := buildBatchPrompt(sampleScene(), minds, moment, "Player", sampleImminent(), "", "")

	if !strings.HasPrefix(p, cognitionSystemHeader) {
		t.Fatalf("batch prompt must open with cognitionSystemHeader (the stable cache prefix)")
	}
	assertOrder(t, p, "SCENE", "THE MINDS YOU SPEAK FOR")
	assertOrder(t, p, "THE MINDS YOU SPEAK FOR", "PUBLIC MOMENT")
	assertOrder(t, p, "PUBLIC MOMENT", "IMMINENT:")

	// The batch carries NO private block — that marker belongs to the isolated seat only.
	if strings.Contains(p, "WHAT ONLY YOU KNOW") {
		t.Fatalf("batch prompt must NOT contain the private block (wall invariant)")
	}
}

// (b) Isolated prompt: the private block sits AFTER the minds and BEFORE the public moment,
// and carries the private lines.
func TestBuildIsolatedPrompt_PrivateLinesBetweenMindsAndMoment(t *testing.T) {
	mind := npcMind{ID: maraID, Name: "Mara", Traits: json.RawMessage(`{"guarded":0.9}`), Malleability: 0.3}
	private := []privateLine{{Content: "the ledger names the smuggler", Tick: 701}}
	moment := []momentLine{{Content: "a torch flares in the doorway", Tick: 700}}
	p := buildIsolatedPrompt(sampleScene(), mind, private, moment, "Player", sampleImminent(), "", "")

	assertOrder(t, p, "THE MINDS YOU SPEAK FOR", "WHAT ONLY YOU KNOW (private, yours alone):")
	assertOrder(t, p, "WHAT ONLY YOU KNOW (private, yours alone):", "PUBLIC MOMENT")
	if !strings.Contains(p, "the ledger names the smuggler") {
		t.Fatalf("isolated prompt must carry the NPC's private line")
	}
	// The private line must appear inside the private block, before the public moment.
	assertOrder(t, p, "the ledger names the smuggler", "PUBLIC MOMENT")
}

// (c) A mind with no personality core renders name-only (the seed may lag; never fail the beat).
func TestBuildBatchPrompt_NameOnlyMind(t *testing.T) {
	minds := []npcMind{{ID: jonasID, Name: "Jonas"}} // no Traits, no Malleability
	p := buildBatchPrompt(sampleScene(), minds, nil, "Player", sampleImminent(), "", "")

	if !strings.Contains(p, "Jonas") {
		t.Fatalf("name-only mind must still render its name")
	}
	if !strings.Contains(p, "(no personality core yet)") {
		t.Fatalf("a mind with no core must render the name-only marker, not a fabricated core")
	}
	if strings.Contains(p, "malleability") {
		t.Fatalf("name-only mind must NOT render a malleability line")
	}
}

// (d) The imminent attempt is the LAST section (the mutable tail): every earlier marker
// precedes it, and the stated wind-up + the attempt JSON + DECIDE FOR ride at the very end.
func TestBuildBatchPrompt_ImminentIsLast(t *testing.T) {
	minds := []npcMind{{ID: jonasID, Name: "Jonas", Traits: json.RawMessage(`{"wary":0.7}`), Malleability: 0.4}}
	moment := []momentLine{{Content: "a torch flares in the doorway", Tick: 700}}
	imm := sampleImminent()
	p := buildBatchPrompt(sampleScene(), minds, moment, "Player", imm, "", "")

	for _, earlier := range []string{"SCENE", "THE MINDS YOU SPEAK FOR", "PUBLIC MOMENT"} {
		assertOrder(t, p, earlier, "IMMINENT:")
	}
	// The wind-up stated text and the attempt JSON both live in the tail (after IMMINENT).
	assertOrder(t, p, "IMMINENT:", imm.Stated)
	assertOrder(t, p, "IMMINENT:", "DECIDE FOR:")
	// The DECIDE FOR line closes the prompt (nothing structural follows the mutable tail).
	tail := p[strings.Index(p, "DECIDE FOR:"):]
	if strings.Contains(tail, "SCENE") || strings.Contains(tail, "PUBLIC MOMENT") {
		t.Fatalf("no section may follow the mutable tail; DECIDE FOR closes the prompt")
	}
	if !strings.Contains(tail, jonasID) {
		t.Fatalf("DECIDE FOR must list the decided-for id(s)")
	}
}
