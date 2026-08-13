package main

import (
	"strings"
	"testing"
)

// Who the player is speaking TO must reach the cognition seat as a FACT, not as a uuid buried in the
// attempt json. Founder-reported: "Mara, I want to rest here" bound the listener to Mara correctly and
// Jonas answered "Room's taken." — the narrator even wrote "cutting the request short before Mara can
// answer". One batch call decides for every mind at once, so with nothing marking the addressee the
// loudest character wins a conversation he was not part of.
func TestAddressedLabel_ResolvesWhoWasSpokenTo(t *testing.T) {
	const mara, jonas, door = "m-1", "j-1", "d-1"
	labels := map[string]string{mara: "Mara", jonas: "the muscle by the bar"}
	present := []string{mara, jonas}

	// A Communicated attempt: the listener is the addressee, rendered in this seat's own vocabulary.
	got := addressedLabel(Attempt{Type: "Communicated", ListenerID: mara}, labels, present)
	if !strings.Contains(got, "Mara") || !strings.Contains(got, mara) {
		t.Fatalf("addressed = %q, want the listener's label AND id", got)
	}

	// A question put to a present person addresses her too — that is the founder's "Mara, can I rest
	// here?" case, which decompose now routes as speech but which must also work when a QUERY rides
	// alongside it.
	if got := addressedLabel(Attempt{Type: "QUERY", QueryTargetIDs: []string{mara}}, labels, present); !strings.Contains(got, "Mara") {
		t.Fatalf("a question asked of a present actor must address her, got %q", got)
	}

	// A question about the DOOR addresses nobody: the engine answers it, and no mind owes a reply.
	if got := addressedLabel(Attempt{Type: "QUERY", QueryTargetIDs: []string{door}}, labels, present); got != "" {
		t.Fatalf("a question about scenery must address no one, got %q", got)
	}

	// Someone who is not in the room cannot be spoken to this beat.
	if got := addressedLabel(Attempt{Type: "Communicated", ListenerID: "absent-1"}, labels, present); got != "" {
		t.Fatalf("an absent listener must not be addressed, got %q", got)
	}

	// An ordinary act addresses no one.
	if got := addressedLabel(Attempt{Type: "ActorMoved", ToTargetID: door}, labels, present); got != "" {
		t.Fatalf("a move addresses no one, got %q", got)
	}
}

// ...and it must actually appear in the prompt the seat reads. A helper that computes the right string
// into a variable nobody renders would pass the test above and change nothing in play.
func TestCognitionPrompt_CarriesTheAddressedLine(t *testing.T) {
	scene := sceneInfo{LocationName: "The Rusted Kettle", Tension: "low"}
	minds := []npcMind{{ID: "m-1", Name: "Ilva"}}
	att := Attempt{Type: "Communicated", Stated: "Ilva, I want to rest here", ListenerID: "m-1", Content: "I want to rest here"}

	with := buildBatchPrompt(scene, minds, nil, "a young stranger", att, "{}", "Ilva (m-1)")
	if !strings.Contains(with, "ADDRESSED: Ilva (m-1)") {
		t.Fatalf("the batch prompt must state who was spoken to:\n%s", with)
	}
	// Omitted entirely when nobody was addressed — no empty header for the model to interpret.
	// Matched on the rendered LINE ("\nADDRESSED: "), not the bare word: the rules header itself
	// explains what ADDRESSED means, so a substring check on the word passes vacuously.
	without := buildBatchPrompt(scene, minds, nil, "a young stranger", Attempt{Type: "ActorMoved"}, "{}", "")
	if strings.Contains(without, "\nADDRESSED: ") {
		t.Fatalf("a beat that addresses no one must carry no ADDRESSED line:\n%s", without)
	}
	if !strings.Contains(with, "\nADDRESSED: ") {
		t.Fatal("the ADDRESSED line must be its own line, not folded into the attempt json")
	}
}
