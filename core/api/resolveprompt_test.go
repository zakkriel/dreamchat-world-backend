package main

import (
	"strings"
	"testing"
)

// Prompt-content pin: the founder-gate fix where a referee ruling embedded an NPC's spoken reply
// inside another event's truth text ("...her voice drops... 'Sit down. Now...'") instead of emitting
// it as its own Communicated event — the narration belt then correctly refused an unverifiable
// speech card, and her words rendered as unattributed world-prose. The fix is prompt-text only
// (prompts/resolve.txt); this test pins the rule marker reaches the referee's system header.
func TestResolveSystemHeader_DialogueIsItsOwnCommunicatedEvent(t *testing.T) {
	if !strings.Contains(resolveSystemHeader, resolveSpeechIsOwnEventMarker) {
		t.Fatalf("resolve.txt missing the dialogue-is-its-own-event rule marker %q — a referee ruling could still bury an NPC's spoken words inside another event's truth/appearance text:\n%s", resolveSpeechIsOwnEventMarker, resolveSystemHeader)
	}
}
