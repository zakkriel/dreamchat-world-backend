package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// The style definer.
//
// One place decides what a world can look like, what each look is called, and what prose is sent to
// an image model to produce it. Everything else — genesis, the art reconciler, the picker the user
// sees, anything added later — reads this and nothing else.
//
// It is a module rather than a column of strings because the catalogue is going to be reused hard:
// the same five looks will front world creation, per-character overrides, regeneration with a
// different style, and whatever surface comes after. A second copy of "anime means cel shaded" in a
// frontend constant is how those surfaces start disagreeing about what a world already chose.
//
// WHY A STYLE IS A NAMED PROFILE AND NOT A PER-REQUEST PROMPT. The image platform keys its reuse
// cache on style_profile_id, so a style has to be a stable, addressable profile or nothing is ever
// reused. Presets get one profile each, tenant-wide, shared by every world that picks them. A custom
// description gets a profile named after the HASH of its prose, so two worlds described the same way
// share one profile and one cache, and a third described differently gets its own.

// artStyleLatitude is appended to every style's positive prompt.
//
// It is here, once, for the same reason the seat prompts carry one byte-identical block: a look that
// forgets it is allowed to be explicit is a look that quietly censors, and five copies of this
// sentence would drift into five different thresholds. See prompts/README.md.
const artStyleLatitude = "unflinching, explicit when the subject is explicit, full anatomy rendered"

// artStyleNegative is refused by every style.
//
// Censorship in a picture is a composition, not a refusal: a bar, a blur, a pixelated patch, a coy
// crop, a cutaway. Naming them is how an image model is told not to compose them.
const artStyleNegative = "text, watermark, censorship bar, black bar, pixelation, blurred subject, coy crop, tasteful cutaway"

// ArtStyle is one entry in the catalogue: what it is called on screen, and what it means to a model.
type ArtStyle struct {
	// Key is the stable identifier stored on the world and sent by a client. Never displayed.
	Key string `json:"key"`
	// Label is what a person picking a style reads.
	Label string `json:"label"`
	// Blurb is one line of plain description, so a picker does not have to show raw prompt prose.
	Blurb string `json:"blurb"`
	// look is the style's own prompt fragment. Unexported: a caller composes through Positive(),
	// which guarantees the latitude travels with it.
	look string
}

// Positive is the full positive prompt for this style — its look, then the latitude.
func (s ArtStyle) Positive() string {
	look := strings.TrimSpace(s.look)
	if look == "" {
		return artStyleLatitude
	}
	return look + ", " + artStyleLatitude
}

// Negative is the full negative prompt for this style.
func (s ArtStyle) Negative() string { return artStyleNegative }

// ProfileName is the style profile this style is stored under on the image platform.
//
// Stable and content-derived: the same choice always resolves to the same profile, so the platform's
// reuse cache works and a world does not mint a new style every time it draws something.
func (s ArtStyle) ProfileName() string {
	if s.Key == artStyleCustomKey {
		return "dreamchat-custom-" + customStyleDigest(s.look)
	}
	return "dreamchat-" + s.Key
}

// artStyleCustomKey marks a style the user wrote themselves rather than picked.
const artStyleCustomKey = "custom"

// artStyleCustomPrefix is how a client asks for one: "custom:oil painting, heavy impasto".
const artStyleCustomPrefix = artStyleCustomKey + ":"

// artStyleMaxCustom bounds a written style. Long enough for a real description, short enough that it
// cannot smuggle a novel into every image prompt this world ever renders.
const artStyleMaxCustom = 400

// artStylePresets is the catalogue, in the order a picker should show it.
//
// The look fragments are written in the vocabulary image models actually respond to — medium, line,
// shading, light — rather than the name of a genre, because "manhwa" alone means far less to a model
// than what manhwa looks like.
var artStylePresets = []ArtStyle{
	{
		Key:   "anime",
		Label: "Anime",
		Blurb: "Cel-shaded animation: clean lines, flat colour, expressive faces.",
		look:  "anime illustration, cel shaded, clean confident linework, flat colour blocking, expressive faces, vivid palette",
	},
	{
		Key:   "realistic",
		Label: "Realistic",
		Blurb: "Photographic: natural light, real skin and fabric, shallow focus.",
		look:  "photorealistic, natural light, fine skin texture and fabric detail, shallow depth of field, film grain",
	},
	{
		Key:   "manhwa",
		Label: "Manhwa",
		Blurb: "Korean webtoon: soft digital painting, crisp lineart, luminous light.",
		look:  "korean manhwa webtoon art, soft digital painting, crisp clean lineart, luminous rim light, graded pastel palette",
	},
	{
		Key:   "comic",
		Label: "Comic",
		Blurb: "Western comic book: bold ink, halftone shading, strong panels.",
		look:  "western comic book art, bold ink outlines, halftone shading, high contrast, dynamic panel composition",
	},
	{
		Key:   "3d",
		Label: "3D",
		Blurb: "Rendered CG: physical shading, soft skin, cinematic lighting.",
		look:  "3d rendered, physically based shading, subsurface scattering skin, cinematic key light, volumetric depth",
	},
}

// artStyleFallback is what a world drawn before styles existed keeps using.
//
// Its profile name is the one already live on the platform, so nothing already illustrated is
// re-keyed and no existing art is orphaned by adding this module.
var artStyleFallback = ArtStyle{
	Key:   "default",
	Label: "House",
	Blurb: "The house look.",
	look:  "painterly, soft rim light, cinematic",
}

// ArtStyleCatalogue returns the styles a person may choose from, in display order.
func ArtStyleCatalogue() []ArtStyle {
	out := make([]ArtStyle, len(artStylePresets))
	copy(out, artStylePresets)
	return out
}

// ResolveArtStyle turns a stored or submitted choice into the style to draw with.
//
// Accepts a preset key ("anime"), a written style ("custom:oil painting, heavy impasto"), or the
// empty string for a world that never chose — which is every world that predates this module, and is
// why the empty case is a value rather than an error.
func ResolveArtStyle(choice string) (ArtStyle, error) {
	choice = strings.TrimSpace(choice)
	if choice == "" {
		return artStyleFallback, nil
	}

	if rest, ok := strings.CutPrefix(choice, artStyleCustomPrefix); ok {
		prose := strings.Join(strings.Fields(rest), " ")
		if prose == "" {
			return ArtStyle{}, fmt.Errorf("describe the style you want, or pick one of the %d offered", len(artStylePresets))
		}
		if len(prose) > artStyleMaxCustom {
			return ArtStyle{}, fmt.Errorf("that style description is %d characters; keep it under %d", len(prose), artStyleMaxCustom)
		}
		return ArtStyle{
			Key:   artStyleCustomKey,
			Label: "Your own",
			Blurb: prose,
			look:  prose,
		}, nil
	}

	for _, s := range artStylePresets {
		if s.Key == choice {
			return s, nil
		}
	}
	if choice == artStyleFallback.Key {
		return artStyleFallback, nil
	}
	return ArtStyle{}, fmt.Errorf("%q is not a style; choose one of %s, or send %s followed by your own description",
		choice, strings.Join(artStyleKeys(), ", "), artStyleCustomPrefix)
}

func artStyleKeys() []string {
	keys := make([]string, 0, len(artStylePresets))
	for _, s := range artStylePresets {
		keys = append(keys, s.Key)
	}
	return keys
}

// customStyleDigest names a written style by its content, so identical prose is one profile and one
// reuse cache rather than a new profile per world.
func customStyleDigest(prose string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.Join(strings.Fields(prose), " "))))
	return hex.EncodeToString(sum[:])[:12]
}
