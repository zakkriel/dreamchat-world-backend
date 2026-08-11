package main

import (
	"strings"
	"testing"
)

const npcID = "2ac70000-0000-0000-0000-0000000000a4"

func attributed(id, kind, text string) string {
	return `[{"speaker_id":"` + id + `","kind":"` + kind + `","text":"` + text + `"}]`
}

// The live defect, as a test. The founder typed his own act and quote; the beat came back with an
// ACTION segment attributed to a hooded figure performing it. `action` carried no verbatim
// requirement, so nothing checked it.
func TestPlayerVoice_RefusesAnNPCPerformingThePlayersAct(t *testing.T) {
	belts := NarrationBelts{
		PresentIDs: []string{npcID},
		Player:     newPlayerVoice("I raise both hands, empty. 'Easy. I mean no trouble.'"),
	}

	bad := attributed(npcID, "action", "he raises both hands, empty: 'Easy. I mean no trouble.'")
	_, err := DecodeAndValidateNarration(bad, belts)
	if err == nil {
		t.Fatal("an NPC performed the player's act and quoted his words — accepted")
	}
	if !strings.Contains(err.Error(), "PLAYER's own") {
		t.Fatalf("rejection must say what is wrong so the repair prompt can act on it, got: %v", err)
	}

	// The CORRECT rendering of the same moment — second person, no speaker — must pass. A belt that
	// also refuses this would leave the narrator no way to narrate the beat at all.
	good := `[{"speaker_id":null,"kind":"narration","text":"You raise both hands, empty: 'Easy. I mean no trouble.'"}]`
	if _, err := DecodeAndValidateNarration(good, belts); err != nil {
		t.Fatalf("the player's own moment as narration must pass, got: %v", err)
	}
}

// The belt must not muzzle NPCs. An NPC acting or speaking on their own account shares vocabulary
// with the player's input all the time ("the bar", "hands") — only a real echo may trip it.
func TestPlayerVoice_LeavesGenuineNPCActionAlone(t *testing.T) {
	belts := NarrationBelts{
		PresentIDs: []string{npcID},
		Player:     newPlayerVoice("I raise both hands, empty. 'Easy. I mean no trouble.'"),
	}
	ok := attributed(npcID, "action", "he sets both hands flat on the bar and says nothing")
	if _, err := DecodeAndValidateNarration(ok, belts); err != nil {
		t.Fatalf("an NPC's own act sharing incidental words with the input must pass, got: %v", err)
	}
}

// Normalisation: a narrator that re-punctuates or re-cases the player's quote is doing the same thing
// and must not slip through on a comma.
func TestPlayerVoice_SurvivesRepunctuation(t *testing.T) {
	v := newPlayerVoice("I say 'Easy. I mean no trouble.'")
	if echo := v.Echoes("he shrugs: 'easy, i mean no trouble,'"); echo == "" {
		t.Fatal("a re-punctuated, re-cased echo of the player's quote slipped past")
	}
}

// Thresholds and nil-safety: a bare or tiny input yields an inert belt rather than one that rejects
// every segment containing a common word.
func TestPlayerVoice_InertOnShortOrEmptyInput(t *testing.T) {
	if v := newPlayerVoice(""); v != nil {
		t.Fatal("empty input must produce no belt")
	}
	if v := newPlayerVoice("I wait"); v != nil {
		t.Fatal("an input too short to be evidence must produce no belt")
	}
	var none *PlayerVoice
	if echo := none.Echoes("anything at all"); echo != "" {
		t.Fatalf("a nil belt must be inert, got %q", echo)
	}
	// A short quote is not evidence: "no" appearing in an NPC line proves nothing.
	if v := newPlayerVoice("I nod and say 'no'"); v != nil && v.Echoes("he says no") != "" {
		t.Fatal("a two-letter quote must not be treated as the player's voice")
	}
}
