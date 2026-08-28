package main

// worldgenesis.go — authoring one world's FICTION from a user's brief (PRD: World Creation).
//
// This is place_author's discipline at world scale, and the split is the same one: the seat authors
// what a world IS, the engine authors everything about how it is BUILT. world_genesis/1 has no numeric
// field anywhere, so an id, a coordinate, a tick, a radius or a count cannot leave the model even if
// it tries — ids come from gen_random_uuid(), footprints from fn_extent_class_metres/fn_area_around,
// time from the ladder in worldgenesiscommit.go, acceptance from the gate. Qualities the engine needs
// as numbers (a trait's strength, how malleable a mind is) arrive as CLASSES and are mapped here.
//
// The seat is the leash; this file is the belt. Everything after the JSON decodes is a Go check, and
// the checks that matter are the CROSS-REFERENCES: the seat can only join its parts by canonical_name,
// so a way pointing at a place that does not exist, a secret held by nobody, or an arrival into an
// unauthored room are all expressible and all refused. A refusal is an ordinary answer here — the same
// posture the World Actor takes with errIntrusionRejected — and it never leaves a half-built world,
// because nothing has been written yet when validation runs.

import (
	"embed"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

//go:embed prompts/world_genesis.txt schema/world_genesis.v1.schema.json
var worldGenesisFS embed.FS

var (
	worldGenesisSystemHeader = mustReadGenesisFile("prompts/world_genesis.txt")
	worldGenesisSchemaJSON   = mustReadGenesisFile("schema/world_genesis.v1.schema.json")
)

func mustReadGenesisFile(name string) string {
	b, err := worldGenesisFS.ReadFile(name)
	if err != nil {
		panic("worldgenesis: embed " + name + ": " + err.Error())
	}
	return string(b)
}

// The markers the prompt uses to separate the standing rulebook from this build's inputs. Load-bearing
// in two directions: content tests assert the rulebook still carries its rules, and the deterministic
// fake driver (bridge_fakes.go) parses the brief back out of the prompt so DREAMCHAT_BRIDGE=fake can
// build a real world with no key. Change the strings and both follow.
const (
	worldGenesisBriefMarker   = "BRIEF (the user's own words — this is the specification, not a theme):"
	worldGenesisAnswersMarker = "ANSWERS (the user's replies to what was asked — these outrank your judgement):"
)

// errGenesisRefused marks a brief this seat could not turn into a coherent world. Like
// errIntrusionRejected it means "the honest answer is no", not "something broke": the caller reports
// the reason to the user and no world exists. Anything NOT wrapped in it is an infrastructure fault.
type genesisRefusal struct{ why string }

func (e *genesisRefusal) Error() string { return "world genesis refused: " + e.why }

func refuse(format string, args ...any) error {
	return &genesisRefusal{why: fmt.Sprintf(format, args...)}
}

// IsGenesisRefusal reports whether err is a refusal (a brief that cannot become a world) rather than a
// fault (a database or provider failure). The two get different answers on the wire.
func IsGenesisRefusal(err error) bool {
	var r *genesisRefusal
	return errors.As(err, &r)
}

// genesisDoc is world_genesis/1 decoded. Field-for-field with the schema; no field carries a number,
// which is the whole point.
type genesisDoc struct {
	World struct {
		DisplayName string `json:"display_name"`
		Tagline     string `json:"tagline"`
		Mood        string `json:"mood"`
		Ornament    string `json:"ornament"`
	} `json:"world"`
	Region struct {
		Descriptor  string `json:"descriptor"`
		ExtentClass string `json:"extent_class"`
	} `json:"region"`
	Places            []genesisPlace     `json:"places"`
	Ways              []genesisWay       `json:"ways"`
	Cast              []genesisActor     `json:"cast"`
	Objects           []genesisObject    `json:"objects"`
	History           []genesisEvent     `json:"history"`
	Arrival           genesisArrival     `json:"arrival"`
	ArrivalCandidates []genesisCandidate `json:"arrival_candidates,omitempty"`
}

type genesisPlace struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
	Kind          string `json:"kind"`
	Description   string `json:"description"`
	Tension       string `json:"tension"`
	ExtentClass   string `json:"extent_class"`
}

type genesisWay struct {
	Descriptor string `json:"descriptor"`
	FromPlace  string `json:"from_place"`
	ToPlace    string `json:"to_place"`
	State      string `json:"state"`
}

type genesisTrait struct {
	Key      string `json:"key"`
	Strength string `json:"strength"`
	Manner   string `json:"manner"`
}

type genesisActor struct {
	Descriptor    string         `json:"descriptor"`
	CanonicalName string         `json:"canonical_name"`
	Standing      string         `json:"standing"`
	SpeechManner  string         `json:"speech_manner"`
	Traits        []genesisTrait `json:"traits"`
	Hiding        string         `json:"hiding"`
	Malleability  string         `json:"malleability,omitempty"`
	StartsIn      string         `json:"starts_in"`
}

type genesisObject struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
	Kind          string `json:"kind"`
	Where         struct {
		InPlace   string `json:"in_place,omitempty"`
		CarriedBy string `json:"carried_by,omitempty"`
	} `json:"where"`
}

type genesisKnowledge struct {
	Holder        string `json:"holder"`
	Content       string `json:"content"`
	EpistemicType string `json:"epistemic_type"`
}

type genesisEvent struct {
	WhatHappened string             `json:"what_happened"`
	Where        string             `json:"where"`
	Who          []string           `json:"who,omitempty"`
	Knowledge    []genesisKnowledge `json:"knowledge"`
}

type genesisArrival struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
	Place         string `json:"place"`
	Stated        string `json:"stated"`
	Why           string `json:"why,omitempty"`
}

type genesisCandidate struct {
	Descriptor    string `json:"descriptor"`
	CanonicalName string `json:"canonical_name"`
	Why           string `json:"why"`
}

// InterviewAnswer is one question the user was asked and what they said back. Carried by the client on
// every request rather than stored: the interview has no session, no table and nothing to resume.
type InterviewAnswer struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// authorWorld lives in worldidentity.go — identity, then fill.

// buildWorldGenesisPrompt renders the standing rulebook, then this build's inputs. Assembled in Go from
// plain values — the prompt file is a fixed rulebook and never carries world data (prompts/README.md).
func buildWorldGenesisPrompt(brief string, answers []InterviewAnswer) string {
	var sb strings.Builder
	sb.WriteString(worldGenesisSystemHeader)
	sb.WriteString("\n\n")
	sb.WriteString(worldGenesisBriefMarker)
	sb.WriteString("\n")
	sb.WriteString(strings.TrimSpace(brief))
	if len(answers) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(worldGenesisAnswersMarker)
		sb.WriteString("\n")
		for _, a := range answers {
			q := strings.TrimSpace(a.Question)
			v := strings.TrimSpace(a.Answer)
			if q == "" || v == "" {
				continue
			}
			sb.WriteString("- ")
			sb.WriteString(q)
			sb.WriteString("\n  → ")
			sb.WriteString(v)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// The closed sets. Duplicated from the schema on purpose: the schema constrains a cooperative provider,
// this constrains reality. A driver that ignores response_format (or a fake that drifts) reaches here.
var (
	genesisExtentClasses = map[string]bool{"intimate": true, "small": true, "medium": true, "large": true, "vast": true}
	genesisTensions      = map[string]bool{"frantic": true, "tense": true, "normal": true, "calm": true, "none": true}
	genesisWayStates     = map[string]bool{"open": true, "shut": true, "locked": true}
	genesisStrengths     = map[string]bool{"faint": true, "moderate": true, "strong": true, "defining": true}
	genesisEpistemic     = map[string]bool{
		"direct": true, "told": true, "overheard": true, "public": true, "rumor": true, "inference": true,
	}
)

// validate is the belt. Every check here is something the schema permits and the world cannot survive:
// a dangling reference, a name used twice, a room nobody can leave, a secret held by nobody, or the
// player being handed knowledge they did not earn. All of them are refusals — the document is wrong,
// not the machine.
func (d *genesisDoc) validate() error {
	if strings.TrimSpace(d.World.DisplayName) == "" {
		return refuse("the world has no name")
	}
	if strings.TrimSpace(d.World.Tagline) == "" {
		return refuse("the world has no tagline")
	}
	if strings.TrimSpace(d.World.Mood) == "" || strings.TrimSpace(d.World.Ornament) == "" {
		return refuse("the world has no mood or ornament")
	}
	if strings.TrimSpace(d.Region.Descriptor) == "" {
		return refuse("the region has no descriptor")
	}
	if !genesisExtentClasses[d.Region.ExtentClass] {
		return refuse("the region's extent_class %q is outside the closed set", d.Region.ExtentClass)
	}

	// Places first: everything else references them.
	if len(d.Places) < 2 {
		return refuse("a world needs at least two places, so there is somewhere to go")
	}
	places := make(map[string]bool, len(d.Places))
	for i, p := range d.Places {
		name := strings.TrimSpace(p.CanonicalName)
		switch {
		case name == "":
			return refuse("place %d has no canonical_name", i+1)
		case places[name]:
			return refuse("two places are both called %q — a name is a join key here and must be unique", name)
		case strings.TrimSpace(p.Descriptor) == "":
			return refuse("the place %q has no descriptor, so a stranger has no way to see it", name)
		case strings.TrimSpace(p.Description) == "":
			return refuse("the place %q has no description, so the narrator has nothing to work from", name)
		case strings.TrimSpace(p.Kind) == "":
			return refuse("the place %q has no kind", name)
		case !genesisTensions[p.Tension]:
			return refuse("the place %q has tension %q, outside the closed set", name, p.Tension)
		case !genesisExtentClasses[p.ExtentClass]:
			return refuse("the place %q has extent_class %q, outside the closed set", name, p.ExtentClass)
		}
		places[name] = true
	}

	// The cast. Names must not collide with places: one namespace, because references are by name alone.
	if len(d.Cast) == 0 {
		return refuse("a world needs at least one person in it")
	}
	cast := make(map[string]bool, len(d.Cast))
	for i, a := range d.Cast {
		name := strings.TrimSpace(a.CanonicalName)
		switch {
		case name == "":
			return refuse("cast member %d has no canonical_name", i+1)
		case cast[name]:
			return refuse("two people are both called %q — names are join keys and must be unique", name)
		case places[name]:
			return refuse("%q is both a person and a place", name)
		case identifierShapedName(name):
			return refuse("%q reads like a join key, not a person's name — name people as a voice would say them, never snake_case or all-lowercase", name)
		case strings.TrimSpace(a.Descriptor) == "":
			return refuse("%q has no descriptor, and a descriptor is all a stranger sees", name)
		case strings.TrimSpace(a.Standing) == "":
			return refuse("%q has no standing", name)
		case strings.TrimSpace(a.SpeechManner) == "":
			return refuse("%q has no speech_manner", name)
		case strings.TrimSpace(a.Hiding) == "":
			return refuse("%q is hiding nothing — every person here holds one thing they will not say", name)
		case len(a.Traits) == 0:
			return refuse("%q has no traits", name)
		case !places[strings.TrimSpace(a.StartsIn)]:
			return refuse("%q starts in %q, which is not a place in this world", name, a.StartsIn)
		case a.Malleability != "" && !genesisStrengths[a.Malleability]:
			return refuse("%q has malleability %q, outside the closed set", name, a.Malleability)
		}
		for _, t := range a.Traits {
			if strings.TrimSpace(t.Key) == "" {
				return refuse("%q has a trait with no key", name)
			}
			if !genesisStrengths[t.Strength] {
				return refuse("%q's trait %q has strength %q, outside the closed set", name, t.Key, t.Strength)
			}
			if strings.TrimSpace(t.Manner) == "" {
				return refuse("%q's trait %q has no manner", name, t.Key)
			}
		}
		cast[name] = true
	}

	// The player. A premise, not a mind — and a stranger to everyone, which starts with not being
	// someone the cast already contains.
	player := strings.TrimSpace(d.Arrival.CanonicalName)
	switch {
	case player == "":
		return refuse("the arriving player has no name")
	case cast[player]:
		return refuse("the player %q is also in the cast — the player is not one of the world's own people", player)
	case places[player]:
		return refuse("the player %q is also a place", player)
	case identifierShapedName(player):
		return refuse("the player's name %q reads like a join key, not a person's name — name people as a voice would say them, never snake_case or all-lowercase", player)
	case strings.TrimSpace(d.Arrival.Descriptor) == "":
		return refuse("the player has no descriptor, so the room cannot see them walk in")
	case strings.TrimSpace(d.Arrival.Stated) == "":
		return refuse("the arrival is unstated, and that sentence is the only thing the player knows")
	case !places[strings.TrimSpace(d.Arrival.Place)]:
		return refuse("the player arrives in %q, which is not a place in this world", d.Arrival.Place)
	}

	// Ways. Distinct real endpoints, and — the SPEC-030 floor — a way out of the room you start in.
	if len(d.Ways) == 0 {
		return refuse("nothing joins the places, so no one can leave the room they start in")
	}
	arrivalHasExit := false
	for i, w := range d.Ways {
		from, to := strings.TrimSpace(w.FromPlace), strings.TrimSpace(w.ToPlace)
		switch {
		case strings.TrimSpace(w.Descriptor) == "":
			return refuse("way %d has no descriptor", i+1)
		case !places[from]:
			return refuse("a way leads from %q, which is not a place in this world", from)
		case !places[to]:
			return refuse("a way leads to %q, which is not a place in this world", to)
		case from == to:
			return refuse("the way %q joins %q to itself", w.Descriptor, from)
		case !genesisWayStates[w.State]:
			return refuse("the way %q is %q, outside the closed set", w.Descriptor, w.State)
		}
		if from == strings.TrimSpace(d.Arrival.Place) || to == strings.TrimSpace(d.Arrival.Place) {
			arrivalHasExit = true
		}
	}
	if !arrivalHasExit {
		return refuse("nothing leads out of %q, where the player arrives", d.Arrival.Place)
	}

	// Someone to walk in on. Not a mechanical requirement — an empty opening is legal and inert, and
	// the whole promise of the surface is a world that was busy before you got there.
	someoneThere := false
	for _, a := range d.Cast {
		if strings.TrimSpace(a.StartsIn) == strings.TrimSpace(d.Arrival.Place) {
			someoneThere = true
			break
		}
	}
	if !someoneThere {
		return refuse("nobody is in %q when the player walks in", d.Arrival.Place)
	}

	// Arrival candidates: optional, but when present the offer must be coherent — exactly three,
	// every field filled, names distinct, and exactly one of them IS the arrival (the recommendation).
	if len(d.ArrivalCandidates) > 0 {
		if len(d.ArrivalCandidates) != 3 {
			return refuse("arrival candidates must number exactly three, got %d", len(d.ArrivalCandidates))
		}
		seen := map[string]bool{}
		matches := 0
		for _, c := range d.ArrivalCandidates {
			name := strings.TrimSpace(c.CanonicalName)
			if name == "" || strings.TrimSpace(c.Descriptor) == "" || strings.TrimSpace(c.Why) == "" {
				return refuse("an arrival candidate is missing its name, descriptor or why")
			}
			if identifierShapedName(name) {
				return refuse("the arrival candidate %q reads like a join key, not a person's name — name people as a voice would say them, never snake_case or all-lowercase", name)
			}
			if seen[name] {
				return refuse("two arrival candidates share the name %q", name)
			}
			seen[name] = true
			if name == strings.TrimSpace(d.Arrival.CanonicalName) {
				matches++
			}
		}
		if matches != 1 {
			return refuse("exactly one arrival candidate must be the arrival itself (the recommended default); %d match", matches)
		}
	}

	// Objects: somewhere findable, exactly one somewhere.
	if len(d.Objects) == 0 {
		return refuse("the world has no objects that matter")
	}
	seenObject := make(map[string]bool, len(d.Objects))
	for i, o := range d.Objects {
		name := strings.TrimSpace(o.CanonicalName)
		in, held := strings.TrimSpace(o.Where.InPlace), strings.TrimSpace(o.Where.CarriedBy)
		switch {
		case name == "":
			return refuse("object %d has no canonical_name", i+1)
		case seenObject[name]:
			return refuse("two objects are both called %q", name)
		case strings.TrimSpace(o.Descriptor) == "":
			return refuse("the object %q has no descriptor", name)
		case strings.TrimSpace(o.Kind) == "":
			return refuse("the object %q has no kind", name)
		case in == "" && held == "":
			return refuse("the object %q is nowhere — it must sit in a place or be carried by someone", name)
		case in != "" && held != "":
			return refuse("the object %q is both in %q and carried by %q", name, in, held)
		case in != "" && !places[in]:
			return refuse("the object %q sits in %q, which is not a place in this world", name, in)
		case held != "" && !cast[held]:
			return refuse("the object %q is carried by %q, who is not in this world", name, held)
		}
		seenObject[name] = true
	}

	// History, and the rule that matters most: none of it belongs to the player.
	if len(d.History) == 0 {
		return refuse("nothing happened before the player arrived")
	}
	for i, h := range d.History {
		if strings.TrimSpace(h.WhatHappened) == "" {
			return refuse("event %d has no account of what happened", i+1)
		}
		if !places[strings.TrimSpace(h.Where)] {
			return refuse("event %d happened in %q, which is not a place in this world", i+1, h.Where)
		}
		for _, who := range h.Who {
			if !cast[strings.TrimSpace(who)] {
				return refuse("event %d involves %q, who is not in this world", i+1, who)
			}
		}
		if len(h.Knowledge) == 0 {
			return refuse("event %d left nobody knowing anything", i+1)
		}
		for _, k := range h.Knowledge {
			holder := strings.TrimSpace(k.Holder)
			switch {
			case holder == player:
				return refuse("the player is given knowledge of event %d, but they were not there and know nothing yet", i+1)
			case !cast[holder]:
				return refuse("event %d is known by %q, who is not in this world", i+1, holder)
			case strings.TrimSpace(k.Content) == "":
				return refuse("event %d gives %q an empty belief", i+1, holder)
			case !genesisEpistemic[k.EpistemicType]:
				return refuse("event %d has %q knowing by %q, outside the closed set", i+1, holder, k.EpistemicType)
			}
		}
	}

	// The tick ladder has to fit under the arrival with room to spare; the schema's ceiling guarantees
	// it, and this is the assertion that keeps the two in step if the ceiling ever moves.
	if genesisBackstoryBaseTick+int64(len(d.History)) > genesisSceneTick {
		return refuse("too much history to place before the world opens")
	}
	return nil
}

// strengthValue maps the qualitative class the seat may emit onto the number the engine stores. The
// values are spread rather than evenly spaced: "defining" has to read as a trait that overrides the
// others, and "faint" has to be weak enough to lose an argument with one.
func strengthValue(class string) float64 {
	switch class {
	case "faint":
		return 0.2
	case "strong":
		return 0.75
	case "defining":
		return 0.95
	default: // "moderate", and anything the belt already vetted as present-but-unset
		return 0.5
	}
}

// malleabilityValue maps the same class onto personality_core.malleability, which the schema CHECKs as
// >0 and <=1. Unset means moderate: a mind that can be changed by what happens, but not easily.
func malleabilityValue(class string) float64 {
	switch class {
	case "faint":
		return 0.15
	case "strong":
		return 0.6
	case "defining":
		return 0.85
	default:
		return 0.35
	}
}

// wayFlags turns the authored state into the two booleans a portal artifact actually carries. "shut"
// and "locked" differ in exactly one way that matters: a shut door can be opened in play.
func wayFlags(state string) (open bool, locked bool) {
	switch state {
	case "open":
		return true, false
	case "locked":
		return false, true
	default: // "shut"
		return false, false
	}
}

// identifierShapedName reports a person's "name" that is machine-shaped rather than speakable:
// snake_case ("silas_holton") or cased script with no capital anywhere ("silas holton").
//
// Why it matters (the Ironmoor breach, live play 2026-08-20): genesis emitted slug join-keys as
// people's canonical names, the registry stored them, and the naming wall guarded strings no model
// ever writes — the seats humanised the slugs to "Silas" and "Emmett" and the player read both.
// The wall now also guards name tokens (migration 20260821120000), but a person's registry name is
// what lenses, labels and naming events will one day SAY, so an unspeakable one is refused at the
// source. Scripts with no case of their own (CJK and the like) carry no capitals and pass untouched;
// places and objects are exempt — their names are made of ordinary English and were never the leak.
func identifierShapedName(name string) bool {
	if strings.Contains(name, "_") {
		return true
	}
	cased, upper := false, false
	for _, r := range name {
		if unicode.IsUpper(r) || unicode.IsTitle(r) {
			upper = true
		}
		if unicode.IsLower(r) || unicode.IsUpper(r) || unicode.IsTitle(r) {
			cased = true
		}
	}
	return cased && !upper
}
