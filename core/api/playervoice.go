package main

import (
	"regexp"
	"strings"
)

// PlayerVoice is the belt against the narrator putting the player's own words and acts into a present
// NPC's mouth.
//
// Caught in live play: the founder typed `I raise both hands, empty. 'Easy. I mean no trouble.'` and
// the beat came back with an ACTION segment attributed to "a hooded figure by the ballast crate"
// reading `he raises both hands, empty: 'Easy. I mean no trouble.'` — a stranger performing the
// player's act, quoting the player's words.
//
// The prompt already forbade it (ORIENTATION FOR IDS: "never attribute a segment to the person you
// narrate TO"), and the prompt is where it is fixed. This is the belt, because the failure is
// mechanically checkable: the player's own input is the one string in the beat we know for certain
// nobody else said.
//
// Only NPC-ATTRIBUTED segments are checked. The narrator re-rendering the player's act in second
// person is correct and required — that is what a narration segment is for.
type PlayerVoice struct {
	spans []string // lowercased fragments that came from the player this beat
}

// newPlayerVoice extracts what the player said and did from his raw beat input.
//
// Two kinds of span, because they fail differently:
//   - QUOTED fragments ('Easy. I mean no trouble.') — the player's literal words. An NPC segment
//     containing these is putting his speech in someone else's mouth. Short quotes ("no") are skipped:
//     a common word inside quotes is not evidence.
//   - The WHOLE input, for the unquoted case ("I raise both hands and step back"). Long enough that
//     an incidental overlap ("the bar") cannot trip it.
func newPlayerVoice(input string) *PlayerVoice {
	v := &PlayerVoice{}
	norm := normaliseVoice(input)
	if len(norm) >= 16 {
		v.spans = append(v.spans, norm)
	}
	for _, m := range voiceQuoted.FindAllStringSubmatch(input, -1) {
		for _, g := range m[1:] {
			if q := normaliseVoice(g); len(q) >= 8 {
				v.spans = append(v.spans, q)
			}
		}
	}
	if len(v.spans) == 0 {
		return nil // nothing long enough to test against: an inert belt, not a broken one
	}
	return v
}

// Straight and curly quotes both, because a player types whatever his keyboard gives him.
var voiceQuoted = regexp.MustCompile(`'([^']{2,})'|"([^"]{2,})"|“([^”]{2,})”|‘([^’]{2,})’`)

// Everything that is not a letter or a digit becomes a space. A narrator repeating the player's line
// re-punctuates it freely — "Easy. I mean no trouble." comes back as "easy, i mean no trouble," — and
// matching on the words alone is what makes the belt survive that. Unicode-aware so an apostrophe or
// an em dash inside a quote does not split a span into unmatchable pieces.
var voicePunct = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// normaliseVoice reduces text to lowercase words separated by single spaces.
func normaliseVoice(s string) string {
	return strings.TrimSpace(voicePunct.ReplaceAllString(strings.ToLower(s), " "))
}

// Echoes returns the player fragment an NPC-attributed segment is repeating, or "" when the segment
// is clean. nil-safe.
func (v *PlayerVoice) Echoes(text string) string {
	if v == nil || text == "" {
		return ""
	}
	hay := normaliseVoice(text)
	for _, span := range v.spans {
		if strings.Contains(hay, span) {
			return span
		}
	}
	return ""
}
