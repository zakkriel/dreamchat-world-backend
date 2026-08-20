package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every seat reads the same latitude, or the world is inconsistently censored.
//
// A model seat that has not been told this is adult fiction will do the thing models do: fade the
// scene, soften the line, attach a note, or quietly decline. One seat doing that is enough to break
// the world — the referee can refuse an outcome the narrator would have written, or cognition can
// give a brutal character second thoughts the author never wrote. So the block is byte-identical
// across every prompt rather than paraphrased per seat, and this test is what keeps it that way.
//
// It reads the FILES rather than the embedded vars on purpose: a prompt added tomorrow gets caught
// here even though no Go code references it yet, which is exactly when it is cheapest to fix.

const uncensoredHeading = "UNCENSORED BY DESIGN."
const neutralityHeading = "NO MORAL OR POLITICAL AUTHORSHIP."

// The AFFIRMATIVE grant, and the reason it is asserted separately from the prohibitions above.
//
// "Never soften" only tells a seat what not to do; it still leaves it choosing how far to go, and a
// model resolves that choice conservatively every time. A seat can satisfy every prohibition in this
// block completely and still close the door and cut to morning — it did not refuse, warn or
// disclaim, it just flinched, and nothing prohibitive forbids flinching. Permission has to be stated
// as a standard being held to, not a limit being allowed near.
const explicitHeading = "EXPLICIT IS THE EXPECTATION, NOT THE PERMISSION."
const calledForHeading = "WHEN THE MOMENT CALLS FOR IT."
const floorLine = "THE ONE FLOOR: everyone depicted in a sexual context is an adult."

func TestEverySeatPromptCarriesTheLatitude(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("prompts", "*.txt"))
	if err != nil {
		t.Fatalf("glob prompts: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no prompt files found — this test is asserting nothing")
	}

	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(body)

		for _, required := range []string{uncensoredHeading, neutralityHeading, explicitHeading, calledForHeading, floorLine} {
			if !strings.Contains(src, required) {
				t.Errorf("%s is missing %q — every seat gets the same latitude or the world is censored in patches",
					filepath.Base(f), required)
			}
		}
	}
}

// The block must be IDENTICAL everywhere. A paraphrase in one seat is how two seats end up with
// different thresholds and the prose starts contradicting itself mid-scene.
func TestTheLatitudeBlockIsByteIdenticalAcrossSeats(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("prompts", "*.txt"))
	if err != nil {
		t.Fatalf("glob prompts: %v", err)
	}

	var canonical string
	var canonicalFrom string
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		block := extractLatitude(string(body))
		if block == "" {
			continue // absence is the other test's job
		}
		if canonical == "" {
			canonical, canonicalFrom = block, filepath.Base(f)
			continue
		}
		if block != canonical {
			t.Errorf("%s states the latitude differently from %s — it must be verbatim identical",
				filepath.Base(f), canonicalFrom)
		}
	}
	if canonical == "" {
		t.Fatal("no prompt carried the latitude block")
	}
}

// extractLatitude returns the block from its first heading through the floor line, or "".
func extractLatitude(src string) string {
	start := strings.Index(src, uncensoredHeading)
	if start < 0 {
		return ""
	}
	end := strings.Index(src, floorLine)
	if end < 0 {
		return ""
	}
	return src[start : end+len(floorLine)]
}

// The seats that actually reach a player are the ones worth naming: a leak of authorial judgement
// into narration or an NPC's reasoning is what a reader would see first.
func TestPlayerFacingSeatsEmbedTheLatitude(t *testing.T) {
	for name, prompt := range map[string]string{
		"narrate":      narrateSystemHeader,
		"cognition":    cognitionSystemHeader,
		"resolve":      resolveSystemHeader,
		"world_actor":  worldActorSystemHeader,
		"decompose":    decomposeSystemHeader,
		"place_author": placeAuthorSystemHeader,
		"anthropic":    anthropicSystemHeader,
	} {
		if !strings.Contains(prompt, uncensoredHeading) {
			t.Errorf("the embedded %s prompt does not carry the latitude — the file may have it while the binary does not", name)
		}
	}
}

// The image side gets the same latitude, in the only prompt this app authors for it.
//
// A picture is not censored by a refusal, it is censored by a composition — a bar, a blur, a coy
// crop, a cutaway. Those are named in the NEGATIVE prompt because that is how an image model is told
// not to compose them; policy prose in the positive prompt would do nothing except give a provider's
// own classifier something to read.
func TestImageStyleCarriesTheSameLatitude(t *testing.T) {
	f := newFakePlatform()
	c := testImageClient(t, f)

	if _, err := c.ensureStyle(context.Background(), mustResolve(t, "")); err != nil {
		t.Fatalf("ensureStyle: %v", err)
	}

	f.mu.Lock()
	body := f.lastStyleCreateBody
	f.mu.Unlock()
	if body == "" {
		t.Fatal("no style was created; this test asserts what we ask for when we create one")
	}

	for _, forbidden := range []string{"censorship bar", "coy crop", "tasteful cutaway", "blurred subject"} {
		if !strings.Contains(body, forbidden) {
			t.Errorf("the style's negative prompt must refuse %q — that is what censorship looks like in a picture", forbidden)
		}
	}
	if !strings.Contains(body, "unflinching") {
		t.Error("the style's positive prompt must carry the latitude the seats were given")
	}
	// The affirmative half. Refusing the bar and the blur says what not to draw; this says to draw
	// the thing itself, which is the instruction a model actually acts on.
	if !strings.Contains(body, "explicit when the subject is explicit") {
		t.Error("the style must ASK for explicitness, not merely decline to forbid it")
	}
}
