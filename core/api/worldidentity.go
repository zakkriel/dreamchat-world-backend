// Governed-by: ADR-P027 — relevance is how much of a thing exists; genesis assigns it and the fill
// authors to the level, rather than to one fullness for everyone.
package main

// worldidentity.go — the understanding pass and the fill it governs.
//
// Genesis is five steps (docs/design/2026-08-26-world-identity-and-the-understanding-pass.md):
// take an input · understand the intention · transcribe what was given · fill, always governed
// by that understanding · emit the completed document. This file is steps 2 and 4. The old
// single-shot world_genesis seat is not the live path.
//
// Code schedules; the model interprets (design §7). Founder 2026-08-28: fill is a few batches in
// product order — places, key history, lives, objects, then a second pass — not one call per rule.
// Identity rules stay in context as law. After sufficiency, one review seat (not the filler, no
// generation context) names breaches; tagged pieces drop; the belt then runs; repair once if needed.

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

//go:embed prompts/world_understanding.txt schema/world_identity.v1.schema.json
var worldUnderstandingFS embed.FS

//go:embed prompts/world_fill.txt schema/world_fill.v1.schema.json
var worldFillFS embed.FS

//go:embed prompts/world_fill_review.txt schema/world_fill_review.v1.schema.json
var worldFillReviewFS embed.FS

var (
	worldUnderstandingSystemHeader = mustReadUnderstandingFile("prompts/world_understanding.txt")
	worldIdentitySchemaJSON        = mustReadUnderstandingFile("schema/world_identity.v1.schema.json")
	worldFillSystemHeader          = mustReadFillFile("prompts/world_fill.txt")
	worldFillSchemaJSON            = mustReadFillFile("schema/world_fill.v1.schema.json")
	worldFillReviewSystemHeader    = mustReadFillReviewFile("prompts/world_fill_review.txt")
	worldFillReviewSchemaJSON      = mustReadFillReviewFile("schema/world_fill_review.v1.schema.json")
)

func mustReadUnderstandingFile(name string) string {
	b, err := worldUnderstandingFS.ReadFile(name)
	if err != nil {
		panic("worldidentity: embed " + name + ": " + err.Error())
	}
	return string(b)
}

func mustReadFillFile(name string) string {
	b, err := worldFillFS.ReadFile(name)
	if err != nil {
		panic("worldfill: embed " + name + ": " + err.Error())
	}
	return string(b)
}

func mustReadFillReviewFile(name string) string {
	b, err := worldFillReviewFS.ReadFile(name)
	if err != nil {
		panic("worldfillreview: embed " + name + ": " + err.Error())
	}
	return string(b)
}

const (
	worldIdentityBriefMarker        = "BRIEF (the user's own words — infer identity from this, invent no places or people):"
	worldIdentityAnswersMarker      = "ANSWERS (the user's replies — stated, outranking inference):"
	worldFillIdentityMarker         = "IDENTITY (immutable for this genesis — every invention answers to it):"
	worldFillWorkMarker             = "WORK ITEM (answer only this):"
	worldFillAlreadyMarker          = "ALREADY AUTHORED. Cross-reference these by the EXACT string inside the quotes and nothing else — never the descriptor, never the quotes, never the two joined. Do not re-emit these names; deepen only if the work item demands a new position:"
	worldFillRejectedMarker         = "YOUR LAST ANSWER TO THIS WORK ITEM WAS REJECTED:"
	worldFillOwedMarker             = "STILL OWED. Canon already references these by name and the belt REFUSES the world until each one exists. If this batch is the one that authors them, it MUST author every one:"
	worldFillReviewExclusionsMarker = "EXCLUSIONS AND DEPARTURE (what this world is not):"
	worldFillReviewNamesMarker      = "FINISHED NAMES (what fill authored):"
)

// worldIdentity is world_identity/1 decoded.
type worldIdentity struct {
	Condition struct {
		Text   string `json:"text"`
		Origin string `json:"origin"`
		Cause  string `json:"cause,omitempty"`
	} `json:"condition"`
	Bargain struct {
		Text      string `json:"text"`
		Therefore string `json:"therefore"`
	} `json:"bargain"`
	Faces []struct {
		Life string `json:"life"`
		Text string `json:"text"`
	} `json:"faces,omitempty"`
	Departure struct {
		Neighbour string `json:"neighbour"`
		HowNot    string `json:"how_not"`
	} `json:"departure"`
	Scarce          string `json:"scarce,omitempty"`
	WronglyAbundant string `json:"wrongly_abundant,omitempty"`
	Consequence     *struct {
		What string `json:"what"`
		Who  string `json:"who"`
	} `json:"consequence,omitempty"`
	Exclusions []struct {
		Never       string `json:"never"`
		Because     string `json:"because"`
		Kind        string `json:"kind"`
		Enforcement string `json:"enforcement,omitempty"`
	} `json:"exclusions"`
	Register      string `json:"register"`
	ContentDemand struct {
		Text      string `json:"text"`
		Therefore string `json:"therefore"`
	} `json:"content_demand"`
	Voice          []string `json:"voice"`
	WorkedExamples []struct {
		Tries   string `json:"tries"`
		Happens string `json:"happens"`
	} `json:"worked_examples,omitempty"`
	Functions []struct {
		Function string `json:"function"`
		Answer   string `json:"answer"`
	} `json:"functions"`
	Rules []identityRule `json:"rules"`
	Flat  bool           `json:"flat,omitempty"`
}

type identityRule struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Text      string `json:"text"`
	Therefore string `json:"therefore"`
}

type fillFragment struct {
	Empty             bool               `json:"empty"`
	WhyEmpty          string             `json:"why_empty,omitempty"`
	World             *genesisDoc        `json:"-"`
	WorldRaw          json.RawMessage    `json:"world,omitempty"`
	RegionRaw         json.RawMessage    `json:"region,omitempty"`
	Places            []genesisPlace     `json:"places,omitempty"`
	Ways              []genesisWay       `json:"ways,omitempty"`
	Factions          []genesisFaction   `json:"factions,omitempty"`
	Concepts          []genesisConcept   `json:"concepts,omitempty"`
	Cast              []genesisActor     `json:"cast,omitempty"`
	Objects           []genesisObject    `json:"objects,omitempty"`
	History           []genesisEvent     `json:"history,omitempty"`
	Arrival           *genesisArrival    `json:"arrival,omitempty"`
	ArrivalCandidates []genesisCandidate `json:"arrival_candidates,omitempty"`
}

type taggedName struct {
	Kind string
	Name string
	Rule string
}

type workItem struct {
	ID        string
	Kind      string
	Text      string
	Therefore string
	// Subject names the one thing this item is about: a top location, a faction, or the group a pack of
	// people shares. Per-item work carries it so a review can name exactly which call to blame.
	Subject string
	// Members are the entities a pack call authors, and nobody else.
	Members []string
	// Scope is the compiled mandate — the only already-authored names this call is shown.
	Scope fillScope
}

func (id *worldIdentity) validate() error {
	if strings.TrimSpace(id.Condition.Text) == "" {
		return refuse("the identity has no condition")
	}
	if strings.TrimSpace(id.Bargain.Text) == "" || strings.TrimSpace(id.Bargain.Therefore) == "" {
		return refuse("the identity has no bargain with a therefore")
	}
	if strings.TrimSpace(id.Departure.Neighbour) == "" || strings.TrimSpace(id.Departure.HowNot) == "" {
		return refuse("the identity has no departure")
	}
	if len(id.Voice) != 3 {
		return refuse("voice must be three sentences of narration")
	}
	if len(id.Functions) != 20 {
		return refuse("the twenty universal functions are required (%d answered)", len(id.Functions))
	}
	if len(id.Rules) == 0 {
		return refuse("the identity emitted no rules to fill from")
	}
	if len(id.Exclusions) == 0 {
		return refuse("exclusions must be present — an empty list is not neutral, but omitting the slot is")
	}
	// A rule's `kind` is DESCRIPTIVE and nothing reads it. Design §7 made rules the work plan — the
	// scheduler dispatched on this field — but the founder's 2026-08-28 ordering ruling replaced that
	// with six fixed batches, and scheduleWork now ignores the identity's rules entirely. The only code
	// that ever read this field was the check that refused the build over it.
	//
	// Measured live 2026-08-28: a build was refused 48 seconds in for `kind: "happen"` — the model had
	// reached for the EXCLUSION vocabulary (exist-kind / happen-kind), which is a reasonable slip and
	// changes nothing downstream. Refusing a world over a label that drives no behaviour is exactly the
	// constriction the founder called out. An unrecognised kind is logged so drift stays visible.
	known := map[string]bool{"constraining": true, "generative": true, "prohibiting": true, "voicing": true}
	seen := map[string]bool{}
	for _, r := range id.Rules {
		if strings.TrimSpace(r.ID) == "" || seen[r.ID] {
			return refuse("rule ids must be unique and non-empty")
		}
		seen[r.ID] = true
		if strings.TrimSpace(r.Kind) == "" {
			return refuse("rule %q has no kind — say what sort of rule it is", r.ID)
		}
		if !known[r.Kind] {
			log.Printf("identity: rule %q has kind %q, outside the described four — kept, nothing dispatches on it", r.ID, r.Kind)
		}
		if strings.TrimSpace(r.Therefore) == "" {
			return refuse("rule %q has no therefore", r.ID)
		}
	}
	return nil
}

// A build is a namespace, then content. The namespace is three cheap sequential calls that name
// everything; the content is parallel waves that author it. Nothing in a content wave can invent a
// dangling reference, because every name it could refer to already exists — which is the whole reason
// the waves are safe to run at once (design 2026-08-31, ADR-P027).
//
// This replaced a descent/ascent pair that authored every entity to the same fullness. Measured
// 2026-08-30: 33 entities cost 76,288 output tokens — 2,312 per entity — because a location-keeper was
// written as richly as a city's ruler. Relevance is the fix, and it is an instruction given BEFORE
// authoring, never a filter applied after.

// depthBudget turns the author-facing `depth` (1-5) into breadth. Depth buys MORE WORLD, never deeper
// entities: how rich a single thing gets is relevance's job, not depth's.
type depthBudget struct {
	Level        int
	TopLocations int
	SubPerTop    int
	PeoplePerTop int
	Label        string
}

func budgetForDepth(d int) depthBudget {
	switch {
	case d <= 1:
		return depthBudget{1, 2, 5, 7, "a locality: one or two places and what sits inside them"}
	case d == 2:
		return depthBudget{2, 3, 6, 12, "a town and its quarters"}
	case d == 3:
		return depthBudget{3, 6, 8, 15, "a city and its region"}
	case d == 4:
		return depthBudget{4, 12, 8, 15, "a province and the cities in it"}
	default:
		return depthBudget{5, 24, 10, 15, "continents and the cities on them"}
	}
}

// fillScope is the COMPILED MANDATE: exactly which already-authored names this one call may see.
// Assembled by code, so a call carries the identity, the concepts, its own slice of the namespace and
// nothing else — instead of the whole growing document restated at every step. That restatement is what
// made every call cost the world all over again.
//
// Whole is for the namespace calls, where there is barely anything to see yet.
type fillScope struct {
	Whole    bool
	Places   []string
	Factions []string
	Concepts []string
	People   []string
}

// conceptsWork runs first and alone. What a world argues over decides what its regions, factions and
// people even ARE, so it sits above geography the way the identity sits above everything.
func conceptsWork() workItem {
	return workItem{ID: "concepts", Kind: "namespace", Scope: fillScope{Whole: true},
		Text: "What bodies of knowledge does this world argue over — schools of thought, doctrines, a trade's craft, " +
			"a contested history? For each: its name, what it actually is in a line, and what is contested about it. " +
			"Nothing below this call may contradict what you write here. Set relevance 2 for the ones this world " +
			"turns on and 1 for the rest. Author no places, people, factions or events.",
		Therefore: "what a world argues over decides what everything in it is"}
}

// scaffoldOneWork names the top of the world: the largest places, the factions that span them, and any
// object the brief makes structural. Names and one-line briefs only — links live IN the description.
func scaffoldOneWork(b depthBudget) workItem {
	return workItem{ID: "scaffold-1", Kind: "namespace", Scope: fillScope{Whole: true},
		Text: fmt.Sprintf("Name the top of this world and nothing more. About %d location(s) at the LARGEST scale the "+
			"brief implies — read the scale off the brief: %s. Then the factions that span them, and any object the "+
			"brief makes structural. For each, emit only: canonical_name, descriptor, kind, extent_class, tension, a tag, and "+
			"relevance. Leave `within` empty on these — they are the top. NO sub-locations, NO people, NO canon "+
			"events: the next call does that. Set relevance 1 unless the brief makes something central.",
			b.TopLocations, b.Label),
		Therefore: "everything below has to hang off names that already exist"}
}

// scaffoldTwoSchedule fixes the ENTIRE namespace before any content call runs: one item per top location
// and per world faction, each naming what sits inside it. After this wave every location, faction and
// person in the world has a name, so a later parallel call referencing one resolves by construction.
//
// This is the correction the founder made to my design: I had proposed one thread per region owning
// everything in it, which made factions second-class and grouped people by geography rather than by who
// they belong to. Naming everything first means a content call's unit can differ by type.
func scaffoldTwoSchedule(doc *genesisDoc, b depthBudget) []workItem {
	var items []workItem
	concepts := conceptNames(doc)
	for _, top := range topLocations(doc) {
		items = append(items, workItem{
			ID: "scaffold-2", Kind: "namespace", Subject: top,
			Scope: fillScope{Places: append(topLocations(doc), top), Factions: factionNames(doc), Concepts: concepts},
			Text: fmt.Sprintf("Name what sits inside %q, and nothing outside it. About %d location(s) at smaller scales, "+
				"each with `within` naming what contains it — nest as deep as the place deserves, down to the smallest "+
				"place a body can stand in. Then about %d people, each in one of these locations, and any faction local "+
				"to here. Names, descriptors, kinds, extents, tensions, tags and RELEVANCE only.\n\n"+
				"RELEVANCE IS THE POINT OF THIS CALL. Most of what you name is relevance 1: it exists, it has a look and "+
				"a tag, and that is complete. Give relevance 2 to what a visitor would plausibly deal with, and 3 only to "+
				"the few who matter here — a keeper, a rival, the one who decides. A world of all-3s costs fifty times a "+
				"world that is honest about who matters.\n\n"+
				"THERE IS A FLOOR, THOUGH: this place is somewhere a visitor may walk into, so give the location "+
				"itself at least relevance 2, and give AT LEAST ONE person here relevance 3 — the one who keeps it, "+
				"serves it, or decides here. A location where everything is relevance 1 is a world with no scene in "+
				"it: nothing is described and nobody can be dealt with.",
				top, b.SubPerTop, b.PeoplePerTop),
			Therefore: "a name that exists cannot be a dangling reference later",
		})
	}
	for _, f := range factionNames(doc) {
		items = append(items, workItem{
			ID: "scaffold-2", Kind: "namespace", Subject: f,
			Scope: fillScope{Places: topLocations(doc), Factions: factionNames(doc), Concepts: concepts},
			Text: "Name the people who make up " + f + ": who leads it, who serves it, who resents it, who left. Each " +
				"needs a name, a descriptor, a tag, `belongs_to` naming this faction, `starts_in` naming one of the " +
				"locations above, and RELEVANCE. Most are relevance 1. Name no new locations.",
			Therefore: "an institution is the people in it, and they have to exist before they can be authored",
		})
	}
	return items
}

// The belt refuses for six reasons, and only three of them mean the fill actually failed.
//
// Four live builds were thrown away for a comma in a name-shaped field, and a fifth for the same name
// arriving as both a person and a location. Each time I repaired the instance. So here is the whole
// refusal surface, sorted once, by whether a finished world can honestly be repaired into a valid one:
//
//	dangling reference    -> reconcileReferences   (the name before the prose is real)
//	level content missing -> settleUnauthored      (it reached the level it reached)
//	closed-set violation  -> normaliseClosedSets   (one word from a fixed list)
//	malformed row         -> dropMalformed         (drop the row, keep the world)
//	NAMESPACE COLLISION   -> resolveNameCollisions (one name belongs to one kind)
//	structurally empty    -> REFUSE. A world with no locations, nobody in it, no history or no way out
//	                         is not something bookkeeping can rescue, and the repair pass gets one try.
//	                         The player holding knowledge they did not earn stays a refusal too: that is
//	                         I-3, and quietly deleting the leak would hide a real defect.
//
// reconcileDocument runs them in dependency order. Nothing here authors anything.
func reconcileDocument(doc *genesisDoc) {
	normaliseClosedSets(doc)
	dropMalformed(doc)
	resolveNameCollisions(doc)
	dropUnstorable(doc)
	normalisePersonNames(doc)
	// References after the passes that DROP rows: a row they removed is a name canon may still point at.
	reconcileReferences(doc)
	resolveArrivalCollision(doc)
	reconcileArrival(doc)
	settleUnauthored(doc)
}

// normaliseClosedSets snaps a value that is one word from a fixed list. A model reaching for a synonym
// is not a broken world: `tension: "uneasy"` is a fine English answer to a question whose legal answers
// happen to be frantic | tense | normal | calm | none.
//
// Defaults ADD NOTHING. Neutral tension, an open way, a moderate strength, and knowledge held by having
// been told — never `direct`, because direct knowledge is a claim about presence, and inventing one would
// put a person at an event they were not at.
func normaliseClosedSets(doc *genesisDoc) {
	snapSet := func(field, name, value, fallback string, legal map[string]bool) string {
		v := strings.ToLower(strings.TrimSpace(value))
		if legal[v] {
			return v
		}
		if v != "" {
			log.Printf("closed set: %s of %q is %q, which is not in the list — using %q", field, name, value, fallback)
		}
		return fallback
	}
	if !genesisExtentClasses[doc.Region.ExtentClass] {
		doc.Region.ExtentClass = snapSet("extent_class", "the region", doc.Region.ExtentClass, "medium", genesisExtentClasses)
	}
	for i := range doc.Places {
		doc.Places[i].Tension = snapSet("tension", doc.Places[i].CanonicalName, doc.Places[i].Tension, "normal", genesisTensions)
		doc.Places[i].ExtentClass = snapSet("extent_class", doc.Places[i].CanonicalName, doc.Places[i].ExtentClass, "small", genesisExtentClasses)
	}
	for i := range doc.Ways {
		doc.Ways[i].State = snapSet("state", doc.Ways[i].Descriptor, doc.Ways[i].State, "open", genesisWayStates)
	}
	for i := range doc.Cast {
		a := &doc.Cast[i]
		if strings.TrimSpace(a.Malleability) != "" && !genesisStrengths[a.Malleability] {
			a.Malleability = snapSet("malleability", a.CanonicalName, a.Malleability, "moderate", genesisStrengths)
		}
		for j := range a.Traits {
			a.Traits[j].Strength = snapSet("trait strength", a.CanonicalName, a.Traits[j].Strength, "moderate", genesisStrengths)
		}
	}
	for i := range doc.History {
		for j := range doc.History[i].Knowledge {
			k := &doc.History[i].Knowledge[j]
			k.EpistemicType = snapSet("epistemic_type", k.Holder, k.EpistemicType, "told", genesisEpistemic)
		}
	}
}

// dropMalformed removes a row that cannot BE a row: no name, no descriptor where one is structural, a
// trait with no key. Dropping is honest and cheap; refusing a finished world over one nameless entry is
// neither. Whatever it removes, reconcileReferences cleans up after.
func dropMalformed(doc *genesisDoc) {
	places := doc.Places[:0]
	for i, p := range doc.Places {
		if strings.TrimSpace(p.CanonicalName) == "" || strings.TrimSpace(p.Descriptor) == "" || strings.TrimSpace(p.Kind) == "" {
			log.Printf("malformed: dropping location %d — no name, descriptor or kind", i+1)
			continue
		}
		places = append(places, p)
	}
	doc.Places = places

	cast := doc.Cast[:0]
	for i, a := range doc.Cast {
		if strings.TrimSpace(a.CanonicalName) == "" || strings.TrimSpace(a.Descriptor) == "" {
			log.Printf("malformed: dropping cast member %d — no name or descriptor", i+1)
			continue
		}
		traits := a.Traits[:0]
		for _, tr := range a.Traits {
			if strings.TrimSpace(tr.Key) == "" || strings.TrimSpace(tr.Manner) == "" {
				continue
			}
			traits = append(traits, tr)
		}
		a.Traits = traits
		cast = append(cast, a)
	}
	doc.Cast = cast

	factions := doc.Factions[:0]
	for i, f := range doc.Factions {
		if strings.TrimSpace(f.CanonicalName) == "" || strings.TrimSpace(f.Descriptor) == "" || strings.TrimSpace(f.Kind) == "" {
			log.Printf("malformed: dropping faction %d — no name, descriptor or kind", i+1)
			continue
		}
		factions = append(factions, f)
	}
	doc.Factions = factions

	concepts := doc.Concepts[:0]
	for i, c := range doc.Concepts {
		if strings.TrimSpace(c.CanonicalName) == "" || strings.TrimSpace(c.WhatItIs) == "" {
			log.Printf("malformed: dropping concept %d — no name, or it does not say what it is", i+1)
			continue
		}
		concepts = append(concepts, c)
	}
	doc.Concepts = concepts

	ways := doc.Ways[:0]
	for i, w := range doc.Ways {
		if strings.TrimSpace(w.Descriptor) == "" {
			log.Printf("malformed: dropping way %d — no descriptor", i+1)
			continue
		}
		ways = append(ways, w)
	}
	doc.Ways = ways

	for i := range doc.History {
		knowledge := doc.History[i].Knowledge[:0]
		for _, k := range doc.History[i].Knowledge {
			if strings.TrimSpace(k.Content) == "" || strings.TrimSpace(k.Holder) == "" {
				continue
			}
			knowledge = append(knowledge, k)
		}
		doc.History[i].Knowledge = knowledge
	}
}

// resolveNameCollisions enforces the one rule the engine cannot bend: ONE NAME BELONGS TO ONE THING.
// References here are names alone, so a name that is both a location and a person is unresolvable by
// construction — the engine cannot tell which one canon meant.
//
// Measured live 2026-08-31: refused at 1,460 s and $0.037 because "Colegio de Auscultadores de Ossa"
// arrived as both. That is a model filing an institution twice, not a world that cannot exist.
//
// PRECEDENCE IS BY HOW MUCH DEPENDS ON THE NAME, never by which is more interesting:
//
//	location > person > faction > concept > object
//
// A location is what people stand in, ways join, events happen in and objects sit in, so dropping one
// orphans everything above it. An object is referenced by nothing.
//
// Duplicates WITHIN a kind are MERGED with the same deepening the waves use, so two half-answers about
// one thing become one whole answer instead of one being thrown away.
func resolveNameCollisions(doc *genesisDoc) {
	taken := map[string]string{}

	places := doc.Places[:0]
	for _, p := range doc.Places {
		name := strings.TrimSpace(p.CanonicalName)
		if have := findPlace(&genesisDoc{Places: places}, name); have != nil {
			log.Printf("collision: two locations called %q — merged", name)
			deepenPlace(have, p)
			continue
		}
		taken[name] = "location"
		places = append(places, p)
	}
	doc.Places = places

	cast := doc.Cast[:0]
	for _, a := range doc.Cast {
		name := strings.TrimSpace(a.CanonicalName)
		if kind, clash := taken[name]; clash && kind != "person" {
			log.Printf("collision: %q is a %s and arrived again as a person — dropping the person, because a %s is what everything else references",
				name, kind, kind)
			continue
		}
		if have := findActor(&genesisDoc{Cast: cast}, name); have != nil {
			log.Printf("collision: two people called %q — merged", name)
			deepenActor(have, a)
			continue
		}
		taken[name] = "person"
		cast = append(cast, a)
	}
	doc.Cast = cast

	factions := doc.Factions[:0]
	for _, f := range doc.Factions {
		name := strings.TrimSpace(f.CanonicalName)
		if kind, clash := taken[name]; clash && kind != "faction" {
			log.Printf("collision: %q is a %s and arrived again as a faction — dropping the faction", name, kind)
			continue
		}
		if have := findFaction(&genesisDoc{Factions: factions}, name); have != nil {
			deepenFaction(have, f)
			continue
		}
		taken[name] = "faction"
		factions = append(factions, f)
	}
	doc.Factions = factions

	concepts := doc.Concepts[:0]
	for _, c := range doc.Concepts {
		name := strings.TrimSpace(c.CanonicalName)
		if kind, clash := taken[name]; clash && kind != "concept" {
			log.Printf("collision: %q is a %s and arrived again as a concept — dropping the concept", name, kind)
			continue
		}
		if have := findConcept(&genesisDoc{Concepts: concepts}, name); have != nil {
			deepenConcept(have, c)
			continue
		}
		taken[name] = "concept"
		concepts = append(concepts, c)
	}
	doc.Concepts = concepts

	objects := doc.Objects[:0]
	for _, o := range doc.Objects {
		name := strings.TrimSpace(o.CanonicalName)
		if kind, clash := taken[name]; clash && kind != "object" {
			log.Printf("collision: %q is a %s and arrived again as an object — dropping the object", name, kind)
			continue
		}
		if have := findObject(&genesisDoc{Objects: objects}, name); have != nil {
			deepenObject(have, o)
			continue
		}
		taken[name] = "object"
		objects = append(objects, o)
	}
	doc.Objects = objects
}

func findActor(d *genesisDoc, name string) *genesisActor {
	for i := range d.Cast {
		if d.Cast[i].CanonicalName == name {
			return &d.Cast[i]
		}
	}
	return nil
}

// snapName resolves a reference that names a real thing and then keeps talking.
//
// Exact match first; then the longest authored name the value BEGINS with, where the name ends at a
// boundary. That is how "Alto Omóplato, en el edificio de contraventanas de hueso." resolves to
// "Alto Omóplato" while "Altozano" does NOT resolve to "Alto".
//
// The boundary check is the guard here, not the longest-match: verified by mutation on 2026-08-31 —
// deleting the longest-match left the test green, deleting the boundary turned it red.
func snapName(value string, known []string) (string, bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", true
	}
	for _, n := range known {
		if v == n {
			return n, true
		}
	}
	best := ""
	for _, n := range known {
		if n == "" || !strings.HasPrefix(v, n) {
			continue
		}
		rest := strings.TrimSpace(v[len(n):])
		if rest != "" && !strings.HasPrefix(rest, ",") && !strings.HasPrefix(rest, "—") &&
			!strings.HasPrefix(rest, "-") && !strings.HasPrefix(rest, "(") && !strings.HasPrefix(rest, ";") &&
			!strings.HasPrefix(rest, ":") {
			continue
		}
		if len(n) > len(best) {
			best = n
		}
	}
	return best, best != ""
}

// reconcileReferences is one pass over EVERY cross-reference in the document, and it exists because the
// one-field-at-a-time version of it cost four live builds in a row.
//
// Every reference here is a name-shaped field, and a model writing prose fills name-shaped fields with
// prose. Measured live, in this order: `starts_in` (234 s), `faction.seat` (750 s), `way.to_place`
// (1,201 s). Each time I repaired the field that had just failed, and the next run failed on the next
// field. The disease is the CLASS, so this treats the class.
//
// What cannot resolve degrades in the cheapest honest way, and the choice differs per kind because what
// each costs to lose differs:
//
//   - a way            -> DROP THE EDGE. One connection; the belt needs one way and an exit.
//   - an object        -> drop it. Nothing references an object by name.
//   - a history event  -> drop the participant or the knowledge entry; drop the event only when that
//     leaves nobody holding it, because an event nobody knows cannot be perceived.
//   - a person's place -> KEEP THE PERSON, clear the placement. Canon references people by name, so
//     dropping one turns a repaired reference into a dangling one — strictly worse.
//
// Refusing was never the honest option here: the model authored a real world and named real things. The
// prose after the name is the mistake, and a mistake in one field is not worth twenty minutes of work.
func reconcileReferences(doc *genesisDoc) {
	places := placeNames(doc)
	factions := factionNames(doc)
	people := castNames(doc)

	ways := make([]genesisWay, 0, len(doc.Ways))
	for _, w := range doc.Ways {
		from, fok := snapName(w.FromPlace, places)
		to, tok := snapName(w.ToPlace, places)
		if !fok || !tok || from == "" || to == "" || from == to {
			log.Printf("reconcile: dropping the way %q — it leads from %q to %q and one of those is not a location here",
				w.Descriptor, w.FromPlace, w.ToPlace)
			continue
		}
		w.FromPlace, w.ToPlace = from, to
		ways = append(ways, w)
	}
	doc.Ways = ways

	// PEOPLE FIRST, because everything below references them. Placement is structural — the engine cannot
	// store a person who is nowhere — so an unplaceable person is dropped and canon is then reconciled
	// against who actually survived. That ordering is the point: clearing the placement instead would
	// just move the refusal to the belt, and reconciling canon first would leave it pointing at the
	// dropped.
	//
	// This should be rare by construction: canon naming a person who has no location creates a debt, and
	// the closing pass authors the location. What reaches here is what two closing rounds could not pay.
	cast := make([]genesisActor, 0, len(doc.Cast))
	for _, a := range doc.Cast {
		got, ok := snapName(a.StartsIn, places)
		if !ok || got == "" {
			log.Printf("reconcile: dropping %q — they start in %q, which no pass ever authored, and a person who is nowhere cannot be stored",
				a.CanonicalName, a.StartsIn)
			continue
		}
		a.StartsIn = got
		for j := range a.BelongsTo {
			snapped, _ := snapName(a.BelongsTo[j], factions)
			a.BelongsTo[j] = snapped
		}
		a.BelongsTo = nonEmpty(a.BelongsTo)
		cast = append(cast, a)
	}
	doc.Cast = cast
	people = castNames(doc)

	objects := make([]genesisObject, 0, len(doc.Objects))
	for _, o := range doc.Objects {
		inPlace, pok := snapName(o.Where.InPlace, places)
		held, hok := snapName(o.Where.CarriedBy, people)
		if !pok || !hok || (inPlace == "" && held == "") {
			log.Printf("reconcile: dropping the object %q — it sits in %q and is carried by %q, and neither resolves",
				o.CanonicalName, o.Where.InPlace, o.Where.CarriedBy)
			continue
		}
		o.Where.InPlace, o.Where.CarriedBy = inPlace, held
		if inPlace != "" {
			o.Where.CarriedBy = "" // exactly one somewhere, and a location is the storable one
		}
		objects = append(objects, o)
	}
	doc.Objects = objects

	history := make([]genesisEvent, 0, len(doc.History))
	for _, h := range doc.History {
		where, wok := snapName(h.Where, places)
		if !wok || where == "" {
			log.Printf("reconcile: dropping history %q — it happened in %q, which is not a location here",
				truncate(h.WhatHappened, 60), h.Where)
			continue
		}
		h.Where = where
		who := make([]string, 0, len(h.Who))
		for _, name := range h.Who {
			if got, ok := snapName(name, people); ok && got != "" {
				who = append(who, got)
			}
		}
		h.Who = who
		knowledge := make([]genesisKnowledge, 0, len(h.Knowledge))
		for _, k := range h.Knowledge {
			got, ok := snapName(k.Holder, people)
			if !ok || got == "" {
				continue
			}
			k.Holder = got
			knowledge = append(knowledge, k)
		}
		h.Knowledge = knowledge
		if len(h.Knowledge) == 0 {
			log.Printf("reconcile: dropping history %q — nobody who resolves knows it, and an event nobody holds cannot be perceived",
				truncate(h.WhatHappened, 60))
			continue
		}
		history = append(history, h)
	}
	doc.History = history

	for i := range doc.Factions {
		got, ok := snapName(doc.Factions[i].Seat, places)
		if !ok {
			log.Printf("reconcile: the faction %q is seated in %q, which resolves to no location — clearing the seat",
				doc.Factions[i].CanonicalName, doc.Factions[i].Seat)
		}
		doc.Factions[i].Seat = got
	}
	// `taught_by` accepts a PERSON or a faction. The belt was too narrow and the model was right: a craft
	// is taught by a person, and one live build filled this with "Auscultadora Mayor Del Vas" six times.
	// Clearing those threw away true content to satisfy a rule nobody had thought about.
	teachers := append(append([]string{}, factions...), people...)
	for i := range doc.Concepts {
		got, ok := snapName(doc.Concepts[i].TaughtBy, teachers)
		if !ok {
			log.Printf("reconcile: the concept %q is taught by %q, who is neither a faction nor a person here — clearing it",
				doc.Concepts[i].CanonicalName, doc.Concepts[i].TaughtBy)
		}
		doc.Concepts[i].TaughtBy = got
	}

	if got, ok := snapName(doc.Arrival.Place, places); ok && got != "" {
		doc.Arrival.Place = got
	}
}

func castNames(d *genesisDoc) []string {
	out := make([]string, 0, len(d.Cast))
	for _, a := range d.Cast {
		if n := strings.TrimSpace(a.CanonicalName); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func nonEmpty(in []string) []string {
	out := in[:0]
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// settleUnauthored is the last honest act before the belt: anything still owing content at this point is
// recorded at the level it actually reached, which is 1.
//
// Every wave has run. Whatever is still thin is thin because a late pass NAMED it — the closing pass
// authors the people canon references, and it runs after the wave that would have given them a standing.
// The alternatives are both worse: refuse a thirteen-minute build over a person who could simply be
// thin, or leave a relevance-2 person with no standing in the document and let the engine meet them.
//
// This is not the merge ratchet. The ratchet stops one ANSWER from lowering another answer's level;
// this reconciles the finished document with what was actually written in it, and it loses nothing —
// a description already authored stays, it is just no longer owed.
//
// Measured live 2026-08-31: a build was refused at 806 seconds and $0.09 for exactly this.
func settleUnauthored(doc *genesisDoc) {
	for i := range doc.Cast {
		if personOwing(doc.Cast[i]) {
			log.Printf("settle: %q reached relevance 1, not %d — no pass authored what the higher level owes",
				doc.Cast[i].CanonicalName, doc.Cast[i].Relevance)
			doc.Cast[i].Relevance = 1
		}
	}
	for i := range doc.Places {
		if placeOwing(doc.Places[i]) {
			log.Printf("settle: the location %q reached relevance 1, not %d",
				doc.Places[i].CanonicalName, doc.Places[i].Relevance)
			doc.Places[i].Relevance = 1
		}
	}
	owing := map[string]bool{}
	for _, n := range factionsOwing(doc) {
		owing[n] = true
	}
	for i := range doc.Factions {
		if owing[strings.TrimSpace(doc.Factions[i].CanonicalName)] {
			log.Printf("settle: the faction %q reached relevance 1, not %d",
				doc.Factions[i].CanonicalName, doc.Factions[i].Relevance)
			doc.Factions[i].Relevance = 1
		}
	}
	for i := range doc.Concepts {
		if doc.Concepts[i].Relevance >= 2 && strings.TrimSpace(doc.Concepts[i].Contested) == "" {
			doc.Concepts[i].Relevance = 1
		}
	}
}

// ensurePlayableFloor guarantees the world has at least one scene in it.
//
// Measured live 2026-08-31 on the Andantes brief: the scaffold returned FIVE locations and TEN people,
// every single one at relevance 1. That is a coherent answer to "most of what you name is relevance 1"
// and a world nobody can play: nowhere has a description, and nobody has a standing, a manner or
// anything they will not say. The prompt asks for a floor too, but a prompt is a request and this is a
// guarantee — and assigning relevance is exactly what genesis is for (ADR-P027 §2).
//
// It runs BETWEEN the namespace and the content waves, so anything it promotes is then authored by the
// waves rather than left owing. Promotion only ever raises (the ratchet), and it promotes the FEWEST
// things that make a scene: one location, and one person standing in it.
func ensurePlayableFloor(doc *genesisDoc) {
	// The location with the most people in it is where a scene is most likely to happen.
	best, bestCount := -1, -1
	for i, p := range doc.Places {
		n := 0
		for _, a := range doc.Cast {
			if strings.TrimSpace(a.StartsIn) == strings.TrimSpace(p.CanonicalName) {
				n++
			}
		}
		if n > bestCount {
			best, bestCount = i, n
		}
	}
	if best < 0 {
		return // no locations at all; the belt refuses that on its own terms
	}
	described := false
	for _, p := range doc.Places {
		if p.Relevance >= 2 {
			described = true
			break
		}
	}
	if !described {
		log.Printf("fill: no location rose above relevance 1 — promoting %q to 2 so somewhere is described",
			doc.Places[best].CanonicalName)
		doc.Places[best].Relevance = 2
	}
	// Someone a scene can turn to, standing in a location that will be described.
	speakable := false
	for _, a := range doc.Cast {
		if a.Relevance >= 3 {
			speakable = true
			break
		}
	}
	if speakable {
		return
	}
	where := strings.TrimSpace(doc.Places[best].CanonicalName)
	for i, a := range doc.Cast {
		if strings.TrimSpace(a.StartsIn) == where {
			log.Printf("fill: nobody rose above relevance 1 — promoting %q in %q to 3 so the world has one person to deal with",
				a.CanonicalName, where)
			doc.Cast[i].Relevance = 3
			return
		}
	}
	if len(doc.Cast) > 0 {
		log.Printf("fill: nobody rose above relevance 1 and nobody stands in %q — promoting %q to 3",
			where, doc.Cast[0].CanonicalName)
		doc.Cast[0].Relevance = 3
	}
}

// contentSchedule is the parallel half: every wave authors to the relevance the scaffold assigned, and
// entities left at relevance 1 are ALREADY COMPLETE and cost nothing here. That is the saving.
//
// Waves run in order; items inside a wave run at once. Geography precedes people only so that a person's
// location has its description by the time anyone stands in it.
func contentSchedule(doc *genesisDoc, b depthBudget) [][]workItem {
	concepts := conceptNames(doc)
	var geography, factions []workItem
	for _, top := range topLocations(doc) {
		tree := locationTree(doc, top)
		// NOT gated on whether a description is owed. Measured live 2026-08-31: the scaffold returned
		// every location at relevance 1, the wave was skipped, and the build was refused 13 minutes in
		// for "nothing joins the places" — because connectivity and canon have no other owner, and
		// neither of them scales with relevance.
		geography = append(geography, workItem{
			ID: "geography", Kind: "content", Subject: top,
			Scope: fillScope{Places: tree, Factions: factionNames(doc), Concepts: concepts},
			Text: "Two jobs, and the first one is not optional.\n\n" +
				"ONE — JOIN THEM UP. Author the `ways` that connect the locations listed below to each other and to " +
				"their container, so that from any one of them a body can reach the others. A world where nothing " +
				"joins the places cannot be walked into and will be thrown away. This has nothing to do with " +
				"relevance: a location at relevance 1 still has doors.\n\n" +
				"TWO — AUTHOR WHAT HAPPENED HERE. At least one canon event, and every event needs at least one " +
				"holder who is one of the people named below — a knower need never have been present.\n\n" +
				"AND, where the level asks for it: each location listed at relevance 2 or more gets its description " +
				"— what it is, what it was, what it is for, what a stranger sees first. A location listed at " +
				"relevance 1 is FINISHED; do not describe it.",
			Therefore: "a place is what happened in it, and a place nothing joins is not a place",
		})
	}
	for _, f := range factionsOwing(doc) {
		factions = append(factions, workItem{
			ID: "faction", Kind: "content", Subject: f,
			Scope: fillScope{Places: placeNames(doc), Factions: factionNames(doc), Concepts: concepts, People: factionMembers(doc, f)},
			Text: "Author " + f + " and nothing else. What it controls, what it publishes, what it buries. If it is " +
				"relevance 3, also what it wants, what it would sacrifice for that, and where it sits. Its position on " +
				"the concepts above, and what it did that its own members disagree about.",
			Therefore: "an institution is its people's arguments about it",
		})
	}
	waves := [][]workItem{}
	if len(geography) > 0 {
		waves = append(waves, geography)
	}
	if len(factions) > 0 {
		waves = append(waves, factions)
	}
	if people := peoplePacks(doc, concepts); len(people) > 0 {
		waves = append(waves, people)
	}
	return waves
}

// peoplePacks groups the people who are OWED content — relevance 2 and up — into packs of about ten,
// grouped by faction where they have one and by location otherwise, so a pack shares a context and the
// unique part of its prompt is small.
//
// Ten is the founder's number and it is a cost decision, not a quality one: a person authored alone got
// the best answers and cost ~2,300 output tokens; a pack shares one prompt head. People at relevance 1
// appear in no pack at all, because they are already complete.
func peoplePacks(doc *genesisDoc, concepts []string) []workItem {
	const packSize = 10
	groups := map[string][]string{}
	var order []string
	for _, a := range doc.Cast {
		name := strings.TrimSpace(a.CanonicalName)
		if name == "" || !personOwing(a) {
			continue
		}
		key := strings.TrimSpace(a.StartsIn)
		if len(a.BelongsTo) > 0 && strings.TrimSpace(a.BelongsTo[0]) != "" {
			key = strings.TrimSpace(a.BelongsTo[0])
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], name)
	}
	var items []workItem
	for _, key := range order {
		names := groups[key]
		for len(names) > 0 {
			n := packSize
			if len(names) < n {
				n = len(names)
			}
			pack := names[:n]
			names = names[n:]
			items = append(items, workItem{
				ID: "people", Kind: "content", Subject: key, Members: pack,
				Scope: fillScope{Places: placeNames(doc), Factions: factionNames(doc), Concepts: concepts, People: pack},
				Text: "Author exactly the people named in THIS ITEM and nobody else, each to the relevance they were " +
					"given.\n\nRelevance 2: what they are to the place they are in, how they speak, one thing they " +
					"will not say, and at least one trait with the manner it shows in.\n\nRelevance 3: all of that, " +
					"plus the interior — what happened to them growing up, what they believe including what they " +
					"believe that is false, what they say to themselves, what still shows in how they behave, what " +
					"they want and what they would give up for it, and three or four lines in their own voice. Their " +
					"upbringing and their temperament are allowed to disagree: the worst life and an optimistic " +
					"disposition is a person, not a mistake.\n\nAND WHAT THEY KNOW ABOUT THE OTHERS in this pack — " +
					"what they have right, what they have wrong, what they suspect and cannot prove. A perception " +
					"belongs to ONE holder, so two of them may contradict each other; that disagreement is the world " +
					"working. DO NOT INVENT NEW CANON and do not author anyone not named here.",
				Therefore: "uniqueness comes from circumstance, and character comes from what they did with it",
			})
		}
	}
	return items
}

// --- what is still owed, per level (ADR-P027 §5) --------------------------------------------------

func personOwing(a genesisActor) bool {
	if a.Relevance >= 2 && (strings.TrimSpace(a.Standing) == "" || strings.TrimSpace(a.SpeechManner) == "" ||
		strings.TrimSpace(a.Hiding) == "" || len(a.Traits) == 0) {
		return true
	}
	if a.Relevance >= 3 && (strings.TrimSpace(a.Goal) == "" ||
		len(a.Beliefs)+len(a.Mantras)+len(a.Traumas)+len(a.ExamplePhrases) == 0 && strings.TrimSpace(a.Upbringing) == "") {
		return true
	}
	return false
}

func placeOwing(p genesisPlace) bool {
	return p.Relevance >= 2 && strings.TrimSpace(p.Description) == ""
}

func factionsOwing(doc *genesisDoc) []string {
	var out []string
	for _, f := range doc.Factions {
		owed := f.Relevance >= 2 && (strings.TrimSpace(f.Controls) == "" ||
			strings.TrimSpace(f.Publishes) == "" || strings.TrimSpace(f.Buries) == "")
		if f.Relevance >= 3 && strings.TrimSpace(f.Goal) == "" {
			owed = true
		}
		if owed {
			out = append(out, strings.TrimSpace(f.CanonicalName))
		}
	}
	return out
}

// --- namespace readers ----------------------------------------------------------------------------

// topLocations are the places nothing contains: the roots of the tree, and the unit the scaffold and the
// geography wave are sliced by.
func topLocations(d *genesisDoc) []string {
	var out []string
	for _, p := range d.Places {
		if strings.TrimSpace(p.Within) == "" && strings.TrimSpace(p.CanonicalName) != "" {
			out = append(out, strings.TrimSpace(p.CanonicalName))
		}
	}
	return out
}

// locationTree is a top location and everything nested under it, to any depth.
func locationTree(d *genesisDoc, top string) []string {
	inside := map[string][]string{}
	for _, p := range d.Places {
		w := strings.TrimSpace(p.Within)
		if w != "" {
			inside[w] = append(inside[w], strings.TrimSpace(p.CanonicalName))
		}
	}
	out := []string{top}
	for i := 0; i < len(out); i++ {
		for _, child := range inside[out[i]] {
			if child != "" {
				out = append(out, child)
			}
		}
	}
	return out
}

func placeNames(d *genesisDoc) []string {
	var out []string
	for _, p := range d.Places {
		if n := strings.TrimSpace(p.CanonicalName); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func factionNames(d *genesisDoc) []string {
	var out []string
	for _, f := range d.Factions {
		if n := strings.TrimSpace(f.CanonicalName); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func conceptNames(d *genesisDoc) []string {
	var out []string
	for _, c := range d.Concepts {
		if n := strings.TrimSpace(c.CanonicalName); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func factionMembers(d *genesisDoc, faction string) []string {
	var out []string
	for _, a := range d.Cast {
		for _, b := range a.BelongsTo {
			if strings.TrimSpace(b) == faction {
				out = append(out, strings.TrimSpace(a.CanonicalName))
				break
			}
		}
	}
	return out
}

// arrivalWork is the last layer: the world header, the region, and the way in. It runs after the ascent so
// the visitor arrives somewhere already inhabited that already has a history.
func arrivalWork() workItem {
	return workItem{ID: "arrival", Kind: "arrival",
		Text:      "Name the world and the region it sits in, and bring a stranger into it. Which of these places do they arrive in, what is the one sentence they know, and who is already there? THE ARRIVAL IS A STRANGER: a new name, appearing nowhere above, and never one of the world's own people.",
		Therefore: "a visitor arrives among people, never into an empty place they must search"}
}

// authorWorld infers identity, then fills under it, and returns a document that has passed every belt check.
func authorWorld(ctx context.Context, understanding, fill, review Driver, brief string, answers []InterviewAnswer, confirmed json.RawMessage, voice []string, depth int) (*genesisDoc, *worldIdentity, error) {
	var id *worldIdentity
	var err error
	if len(confirmed) > 0 && string(confirmed) != "null" {
		id = &worldIdentity{}
		if err = json.Unmarshal(confirmed, id); err != nil {
			return nil, nil, refuse("the confirmed identity came back malformed (%v)", err)
		}
		if err = id.validate(); err != nil {
			return nil, nil, err
		}
	} else {
		id, err = inferIdentity(ctx, understanding, brief, answers)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(voice) == 3 {
		ok := true
		for i := range voice {
			voice[i] = strings.TrimSpace(voice[i])
			if voice[i] == "" {
				ok = false
			}
		}
		if ok {
			id.Voice = voice
		}
	}
	doc, err := fillFromIdentity(ctx, fill, review, id, brief, answers, depth)
	if err != nil {
		return nil, id, err
	}
	return doc, id, nil
}

func inferIdentity(ctx context.Context, seat Driver, brief string, answers []InterviewAnswer) (*worldIdentity, error) {
	if seat == nil {
		return nil, fmt.Errorf("inferIdentity: no world_understanding seat bound")
	}
	brief = strings.TrimSpace(brief)
	if brief == "" {
		return nil, refuse("the brief is empty — there is nothing to build from")
	}
	raw, err := seat.Generate(ctx, GenRequest{
		Prompt: buildWorldUnderstandingPrompt(brief, answers),
		Schema: json.RawMessage(worldIdentitySchemaJSON),
	})
	if err != nil {
		return nil, fmt.Errorf("inferIdentity: Generate: %w", err)
	}
	var id worldIdentity
	if err := json.Unmarshal([]byte(raw), &id); err != nil {
		return nil, refuse("the identity came back malformed (%v)", err)
	}
	if err := id.validate(); err != nil {
		return nil, err
	}
	return &id, nil
}

func fillFromIdentity(ctx context.Context, seat, review Driver, id *worldIdentity, brief string, answers []InterviewAnswer, depth int) (*genesisDoc, error) {
	if seat == nil {
		return nil, fmt.Errorf("fillFromIdentity: no world_fill seat bound")
	}
	doc := &genesisDoc{}
	var tags []taggedName
	b := budgetForDepth(depth)

	// THE NAMESPACE. Three sequential calls, and they must be sequential: each one decides what the next
	// is allowed to name. Concepts first, because what a world argues over decides what its places,
	// factions and people even are. These are the only calls that see the whole document, and there is
	// barely anything in it to see.
	for _, item := range []workItem{conceptsWork(), scaffoldOneWork(b)} {
		frag, err := fillOnce(ctx, seat, id, item, brief, answers, doc)
		if err != nil {
			return nil, err
		}
		mergeFill(doc, frag, mergeTag(item), &tags)
	}
	// Scaffold 2 fixes the ENTIRE namespace, one call per top location and per faction, in parallel.
	// They cannot invent each other's names because scaffold 1 already fixed the names above them, and
	// they name nothing outside their own subject. After this wave every entity in the world has a name
	// and a relevance — so every later call resolves by construction rather than by luck.
	if items := scaffoldTwoSchedule(doc, b); len(items) > 0 {
		answered := runWave(ctx, seat, id, items, brief, answers, doc, &tags)
		log.Printf("fill: scaffold 2 ran %d call(s) together, %d answered", len(items), answered)
	}
	log.Printf("fill: namespace is %d place(s) (%d top), %d faction(s), %d concept(s), %d person(s); depth %d (%s)",
		len(doc.Places), len(topLocations(doc)), len(doc.Factions), len(doc.Concepts), len(doc.Cast), b.Level, b.Label)
	ensurePlayableFloor(doc)
	log.Printf("fill: owed at relevance 2+ — %d place(s), %d faction(s), %d person(s); everything else is complete at 1",
		countPlacesOwing(doc), len(factionsOwing(doc)), countPeopleOwing(doc))

	// THE CONTENT. Waves in order, items inside a wave at once. Every call authors to the relevance the
	// scaffold assigned, and an entity left at relevance 1 appears in no call at all — that absence is
	// the saving, and it is why depth is affordable at all.
	for wi, wave := range contentSchedule(doc, b) {
		answered := runWave(ctx, seat, id, wave, brief, answers, doc, &tags)
		log.Printf("fill: content wave %d (%s) ran %d call(s) together, %d answered", wi+1, wave[0].ID, len(wave), answered)
	}

	// The arrival is last and alone: the world header, the region, and the way in, into somewhere already
	// inhabited that already has a history.
	if frag, err := fillOnce(ctx, seat, id, arrivalWork(), brief, answers, doc); err != nil {
		log.Printf("fill arrival could not answer: %v", err)
	} else {
		mergeFill(doc, frag, "arrival", &tags)
	}
	// Closing passes: pay what canon owes instead of refusing it. Two rounds, because authoring the
	// owed places can itself name a person, and that person is then owed. Founder 2026-08-28: a gap is
	// worse than an invention that clicks, so the pipeline's answer to an unpaid name is to write it.
	for round := 0; round < 2; round++ {
		people, places := fillDebts(doc)
		if len(people) == 0 && len(places) == 0 {
			break
		}
		log.Printf("closing pass %d: %d person and %d place reference(s) still unpaid", round+1, len(people), len(places))
		item := workItem{
			ID:   "closing",
			Kind: "batch",
			Text: fmt.Sprintf("Author exactly these and nothing else: %d owed person(s) and %d owed place(s), listed under STILL OWED.\n\n"+
				"ALL OF THEM AT RELEVANCE 1. Canon named them; nobody has met them. A name, a one-line descriptor, a "+
				"kind or a place to stand, and a tag — and nothing more. Do not give them a standing, a manner, a "+
				"secret or an inner life: this pass runs after the one that authors those, so anything deeper you "+
				"write here is content nobody will ever finish.",
				len(people), len(places)),
			Therefore: "a world whose canon points at nothing cannot be stored or walked into",
		}
		frag, err := fillOne(ctx, seat, id, item, brief, answers, doc, "")
		// Truncation is the failure mode here — measured live 2026-08-28, an unfocused closing pass ran
		// to the 16384-token ceiling and came back as unexpected EOF. One retry, told what went wrong,
		// costs far less than the build.
		if err != nil && IsGenesisRefusal(err) {
			log.Printf("closing pass %d rejected, asking once more: %v", round+1, err)
			frag, err = fillOne(ctx, seat, id, item, brief, answers, doc,
				err.Error()+" — emit ONLY the owed names, nothing else; a long answer is what broke the last one")
		}
		if err != nil {
			log.Printf("closing pass %d could not answer: %v", round+1, err)
			break
		}
		if !fillHasContent(frag) {
			break
		}
		mergeFill(doc, frag, "closing", &tags)
	}
	if review != nil {
		breaches, err := reviewFill(ctx, review, id, doc)
		if err != nil {
			return nil, err
		}
		retractBreaches(doc, breaches)
	}
	// Bookkeeping before the belt sees it, never authorship: drop what cannot be stored, and make the
	// arrival offer coherent. Both cost a leaf at worst; neither costs the world.
	reconcileDocument(doc)
	if err := doc.validate(); err != nil {
		frag, rerr := fillOne(ctx, seat, id, workItem{
			ID:   "repair",
			Kind: "batch",
			Text: "The belt refused the merged document: " + err.Error() +
				"\n\nAnything you author here that the belt did not explicitly ask for goes in at relevance 1.",
			Therefore: "emit only what the belt is missing; do not re-author names already listed",
		}, brief, answers, doc, "")
		if rerr != nil {
			return nil, err
		}
		mergeFill(doc, frag, "repair", &tags)
		reconcileDocument(doc)
		if err2 := doc.validate(); err2 != nil {
			return nil, err2
		}
	}
	return doc, nil
}

// mergeTag names the work item a piece of content came from, for scoped retraction. Per-item work carries
// its subject, so "person" becomes "person:Adaeze" and a review can name exactly which call to blame.
// fillOnce is one work item with the pipeline's two mercies. A malformed answer gets ONE retry, told
// exactly what was wrong: measured live 2026-08-28, a batch spelled `places` as `place` and
// DisallowUnknownFields ended a 147-second build. An unpaid reference gets ONE nudge at the call that
// made it, which is the cheapest place to fix it. Neither loosens the belt — the fragment still has to
// parse and validate.
func fillOnce(ctx context.Context, seat Driver, id *worldIdentity, item workItem, brief string, answers []InterviewAnswer, doc *genesisDoc) (*fillFragment, error) {
	frag, err := fillOne(ctx, seat, id, item, brief, answers, doc, "")
	if err != nil && IsGenesisRefusal(err) {
		log.Printf("fill %s rejected, retrying once: %v", mergeTag(item), err)
		frag, err = fillOne(ctx, seat, id, item, brief, answers, doc, err.Error())
	}
	if err != nil {
		return nil, err
	}
	if bad := frag.danglingRefs(doc); len(bad) > 0 {
		log.Printf("fill %s left %d reference(s) unpaid, asking once: %s", mergeTag(item), len(bad), strings.Join(bad, "; "))
		again, aerr := fillOne(ctx, seat, id, item, brief, answers, doc,
			"you referenced "+strings.Join(bad, "; ")+" — author those, in this same answer, alongside what you already wrote. Do not drop the reference.")
		if aerr == nil && fillHasContent(again) {
			frag = again
		}
	}
	return frag, nil
}

// runWave runs every item in a wave at once and merges them serially afterwards.
//
// The concurrency is safe by construction, not by hope: each call READS the document to build its prompt
// and check its references, and nothing writes until the wave has finished. Merging cannot be concurrent
// — mergeFill mutates the document and dedupes by name, and two writers would produce the duplicated
// canon that a blind append already cost us once.
//
// One item failing costs that item, never the build. The belt is what refuses a world, and the arrival
// is the one thing it cannot do without.
func runWave(ctx context.Context, seat Driver, id *worldIdentity, items []workItem, brief string, answers []InterviewAnswer, doc *genesisDoc, tags *[]taggedName) int {
	frags := make([]*fillFragment, len(items))
	var wg sync.WaitGroup
	for i, item := range items {
		wg.Add(1)
		go func(i int, item workItem) {
			defer wg.Done()
			frag, err := fillOnce(ctx, seat, id, item, brief, answers, doc)
			if err != nil {
				log.Printf("fill %s could not answer, continuing without it: %v", mergeTag(item), err)
				return
			}
			frags[i] = frag
		}(i, item)
	}
	wg.Wait()
	merged := 0
	for i, frag := range frags {
		if frag == nil {
			continue
		}
		mergeFill(doc, frag, mergeTag(items[i]), tags)
		merged++
	}
	return merged
}

func countPlacesOwing(d *genesisDoc) int {
	n := 0
	for _, p := range d.Places {
		if placeOwing(p) {
			n++
		}
	}
	return n
}

func countPeopleOwing(d *genesisDoc) int {
	n := 0
	for _, a := range d.Cast {
		if personOwing(a) {
			n++
		}
	}
	return n
}

// splitPersonWork separates the per-item person work from the layer work. Only the person items are
// safe to run together — the layers above and below them are ordered by construction.
func splitPersonWork(items []workItem) (people, rest []workItem) {
	for _, it := range items {
		if it.ID == "person" && strings.TrimSpace(it.Subject) != "" {
			people = append(people, it)
			continue
		}
		rest = append(rest, it)
	}
	return people, rest
}

func mergeTag(item workItem) string {
	if s := strings.TrimSpace(item.Subject); s != "" {
		return item.ID + ":" + s
	}
	return item.ID
}

func fillOne(ctx context.Context, seat Driver, id *worldIdentity, item workItem, brief string, answers []InterviewAnswer, soFar *genesisDoc, priorError string) (*fillFragment, error) {
	raw, err := seat.Generate(ctx, GenRequest{
		Prompt: buildWorldFillPrompt(id, item, brief, answers, soFar, priorError),
		Schema: json.RawMessage(worldFillSchemaJSON),
	})
	if err != nil {
		return nil, fmt.Errorf("fillOne %s: Generate: %w", item.ID, err)
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.DisallowUnknownFields()
	var frag fillFragment
	if err := dec.Decode(&frag); err != nil {
		return nil, refuse("fill for %s came back malformed (%v)", item.ID, err)
	}
	if err := frag.validate(); err != nil {
		return nil, err
	}
	// Dangling references are NOT an error here. The invention is usually right — a life that needs a
	// low-tail district on a walking creature has understood the brief better than a schedule that
	// only authored places in the first layer. Unpaid references are work, so they are reported to the caller
	// and paid by a retry or by the closing pass. Nothing is thrown away for making one.
	return &frag, nil
}

// validate checks the content a fragment actually carries. It deliberately does NOT police the
// model's `empty` flag against that content.
//
// `empty` is a SELF-REPORT, and self-reports contradict themselves. Measured live 2026-08-28: the
// revise batch answered `{"empty":false}` with 34 tokens and no entities, and the old code refused
// the whole build — 168 seconds and $0.0066 thrown away over a boolean, when the honest reading of
// that answer is "I had nothing to add", which is exactly what a second pass over an already
// sufficient world should say. The CONTENT is the fact; the flag is commentary.
//
// Nothing is weakened by this. The real guard is genesisDoc.validate() on the merged document, which
// is what refuses a world that is actually missing something, and it still runs.
// validate is the FRAGMENT belt, and it may only check what is true of any answer at any stage. Depth
// is NOT checked here: the scaffold names a relevance-3 person before anybody authors them, and the
// people wave fills them in — that two-stage split is the whole design (ADR-P027), so a fragment-level
// depth check would refuse the scaffold for doing its job. It cost a run to find out.
//
// The join-key check is not repeated here either. It lives at the belt, and normalisePersonNames runs
// before the belt to title-case a name that is merely uncapitalised — rewriting every reference to it as
// it goes. Checking here fired FIRST and harder: measured live 2026-08-28, an 803-second build was
// refused for the cast name "un aprendiz de 27 años" ("a 27-year-old apprentice"), which the
// document-level normaliser would have turned into a speakable name a moment later. A fragment-level
// copy of a document-level rule just means the repair never gets to run.
//
// What IS local: the level itself, and the one line a person at any level answers from.
func (f *fillFragment) validate() error {
	if !fillHasContent(f) {
		return nil
	}
	for _, a := range f.Cast {
		if !validRelevance(a.Relevance) {
			return refuse("%q arrived with relevance %d — say how much of this person exists, 1 to 4", a.CanonicalName, a.Relevance)
		}
		if strings.TrimSpace(a.Tag) == "" {
			return refuse("%q has no tag — one line of temperament is the whole personality of someone nobody has met yet", a.CanonicalName)
		}
	}
	for _, p := range f.Places {
		if !validRelevance(p.Relevance) {
			return refuse("the location %q arrived with relevance %d — say how much of it exists, 1 to 4", p.CanonicalName, p.Relevance)
		}
	}
	for _, fa := range f.Factions {
		if !validRelevance(fa.Relevance) {
			return refuse("the faction %q arrived with relevance %d — say how much of it exists, 1 to 4", fa.CanonicalName, fa.Relevance)
		}
	}
	return nil
}

func (f *fillFragment) danglingRefs(doc *genesisDoc) []string {
	places := map[string]bool{}
	for _, p := range doc.Places {
		places[strings.TrimSpace(p.CanonicalName)] = true
	}
	for _, p := range f.Places {
		places[strings.TrimSpace(p.CanonicalName)] = true
	}
	people := map[string]bool{}
	for _, a := range doc.Cast {
		people[strings.TrimSpace(a.CanonicalName)] = true
	}
	for _, a := range f.Cast {
		people[strings.TrimSpace(a.CanonicalName)] = true
	}

	var bad []string
	place := func(name, what string) {
		name = strings.TrimSpace(name)
		if name == "" || places[name] {
			return
		}
		bad = append(bad, fmt.Sprintf("%s names the place %q, which no batch has authored", what, name))
	}
	person := func(name, what string) {
		name = strings.TrimSpace(name)
		if name == "" || people[name] {
			return
		}
		bad = append(bad, fmt.Sprintf("%s names the person %q, who no batch has authored", what, name))
	}

	for _, pl := range f.Places {
		place(pl.Within, fmt.Sprintf("the place %q", pl.CanonicalName))
	}
	for _, a := range f.Cast {
		place(a.StartsIn, fmt.Sprintf("the person %q", a.CanonicalName))
	}
	for _, o := range f.Objects {
		place(o.Where.InPlace, fmt.Sprintf("the object %q", o.CanonicalName))
		person(o.Where.CarriedBy, fmt.Sprintf("the object %q", o.CanonicalName))
	}
	for _, w := range f.Ways {
		place(w.FromPlace, fmt.Sprintf("the way %q", w.Descriptor))
		place(w.ToPlace, fmt.Sprintf("the way %q", w.Descriptor))
	}
	for i, h := range f.History {
		place(h.Where, fmt.Sprintf("history event %d", i+1))
	}
	if f.Arrival != nil {
		place(f.Arrival.Place, "the arrival")
	}
	return bad
}

func fillHasContent(f *fillFragment) bool {
	return len(f.WorldRaw) > 0 || len(f.RegionRaw) > 0 || len(f.Places) > 0 || len(f.Ways) > 0 ||
		len(f.Factions) > 0 || len(f.Concepts) > 0 ||
		len(f.Cast) > 0 || len(f.Objects) > 0 || len(f.History) > 0 || f.Arrival != nil || len(f.ArrivalCandidates) > 0
}

func docWorldEmpty(d *genesisDoc) bool {
	return strings.TrimSpace(d.World.DisplayName) == ""
}

func mergeFill(doc *genesisDoc, frag *fillFragment, ruleID string, tags *[]taggedName) {
	// Gated on CONTENT, not on the model's `empty` flag — same reason validate() ignores it. A
	// fragment that says empty:true and then hands over three places has authored three places, and
	// dropping them because of its own mislabelling would lose real work silently.
	if frag == nil || !fillHasContent(frag) {
		return
	}
	if len(frag.WorldRaw) > 0 && docWorldEmpty(doc) {
		var w struct {
			DisplayName string `json:"display_name"`
			Tagline     string `json:"tagline"`
			Mood        string `json:"mood"`
			Ornament    string `json:"ornament"`
		}
		if json.Unmarshal(frag.WorldRaw, &w) == nil {
			doc.World.DisplayName = w.DisplayName
			doc.World.Tagline = w.Tagline
			doc.World.Mood = w.Mood
			doc.World.Ornament = w.Ornament
		}
	}
	if len(frag.RegionRaw) > 0 && strings.TrimSpace(doc.Region.Descriptor) == "" {
		_ = json.Unmarshal(frag.RegionRaw, &doc.Region)
	}
	// Deepening, not just adding. The scaffold names a thing thin and a later wave fills it in, so a
	// second mention of an existing name is the NORMAL case and dropping it would silently discard the
	// content the belt is about to demand. Before relevance existed every layer authored everything at
	// once, so skipping a duplicate was right; under the layered fill it would refuse a world for a
	// description that was in fact written.
	for _, p := range frag.Places {
		if have := findPlace(doc, p.CanonicalName); have != nil {
			deepenPlace(have, p)
			continue
		}
		doc.Places = append(doc.Places, p)
		*tags = append(*tags, taggedName{Kind: "place", Name: p.CanonicalName, Rule: ruleID})
	}
	for _, w := range frag.Ways {
		if !hasWay(doc, w) {
			doc.Ways = append(doc.Ways, w)
			*tags = append(*tags, taggedName{Kind: "way", Name: w.Descriptor, Rule: ruleID})
		}
	}
	for _, f := range frag.Factions {
		if have := findFaction(doc, f.CanonicalName); have != nil {
			deepenFaction(have, f)
		} else {
			doc.Factions = append(doc.Factions, f)
			*tags = append(*tags, taggedName{Kind: "faction", Name: f.CanonicalName, Rule: ruleID})
		}
	}
	for _, c := range frag.Concepts {
		if have := findConcept(doc, c.CanonicalName); have != nil {
			deepenConcept(have, c)
			continue
		}
		doc.Concepts = append(doc.Concepts, c)
		*tags = append(*tags, taggedName{Kind: "concept", Name: c.CanonicalName, Rule: ruleID})
	}
	for _, a := range frag.Cast {
		if !hasActor(doc, a.CanonicalName) {
			doc.Cast = append(doc.Cast, a)
			*tags = append(*tags, taggedName{Kind: "cast", Name: a.CanonicalName, Rule: ruleID})
			continue
		}
		// The ascent revisits people who already exist. Deepening them is the point of that pass, so an
		// existing person is UPDATED with whatever the later answer added rather than being discarded.
		for i := range doc.Cast {
			if doc.Cast[i].CanonicalName == a.CanonicalName {
				deepenActor(&doc.Cast[i], a)
				break
			}
		}
	}
	for _, o := range frag.Objects {
		if have := findObject(doc, o.CanonicalName); have != nil {
			deepenObject(have, o)
			continue
		}
		doc.Objects = append(doc.Objects, o)
		*tags = append(*tags, taggedName{Kind: "object", Name: o.CanonicalName, Rule: ruleID})
	}
	// Canon dedupes on (what happened, where). This was a blind append, which was already wrong — the
	// ascent revisits layers and a repair pass re-answers — and became a hazard the moment the
	// per-person calls ran together: seven independent writers describing the same night produced seven
	// events. Canon is the shared record; a perception is per-holder and may disagree freely, but the
	// event underneath it exists once.
	for _, h := range frag.History {
		if !hasEvent(doc, h) {
			doc.History = append(doc.History, h)
			continue
		}
		// Same event, arriving again with witnesses or knowledge the first telling did not have.
		for i := range doc.History {
			if sameEvent(doc.History[i], h) {
				deepenEvent(&doc.History[i], h)
				break
			}
		}
	}
	if frag.Arrival != nil && strings.TrimSpace(doc.Arrival.CanonicalName) == "" {
		doc.Arrival = *frag.Arrival
	}
	if len(frag.ArrivalCandidates) == 3 && len(doc.ArrivalCandidates) == 0 {
		doc.ArrivalCandidates = frag.ArrivalCandidates
	}
}

func hasPlace(d *genesisDoc, name string) bool {
	for _, p := range d.Places {
		if p.CanonicalName == name {
			return true
		}
	}
	return false
}
func sameEvent(a, b genesisEvent) bool {
	return strings.EqualFold(strings.TrimSpace(a.WhatHappened), strings.TrimSpace(b.WhatHappened)) &&
		strings.TrimSpace(a.Where) == strings.TrimSpace(b.Where)
}

func hasEvent(d *genesisDoc, h genesisEvent) bool {
	for _, existing := range d.History {
		if sameEvent(existing, h) {
			return true
		}
	}
	return false
}

// deepenEvent folds a second telling of the same event into the first: more witnesses, more holders.
// Nothing is overwritten — canon is append-only in spirit here too, and two holders of the same event
// are expected to believe different things about it.
func deepenEvent(have *genesisEvent, add genesisEvent) {
	have.Who = appendNew(have.Who, add.Who)
	for _, k := range add.Knowledge {
		dup := false
		for _, existing := range have.Knowledge {
			if strings.TrimSpace(existing.Holder) == strings.TrimSpace(k.Holder) &&
				strings.EqualFold(strings.TrimSpace(existing.Content), strings.TrimSpace(k.Content)) {
				dup = true
				break
			}
		}
		if !dup {
			have.Knowledge = append(have.Knowledge, k)
		}
	}
}

func hasFaction(d *genesisDoc, name string) bool {
	for _, f := range d.Factions {
		if f.CanonicalName == name {
			return true
		}
	}
	return false
}

func hasConcept(d *genesisDoc, name string) bool {
	for _, c := range d.Concepts {
		if c.CanonicalName == name {
			return true
		}
	}
	return false
}

// deepenActor fills in what a later pass learned about someone who already exists, and never overwrites
// what an earlier pass already said. The ascent exists to connect and complete, not to rewrite: a person
// authored on the way down keeps their hiding and their goal, and gains the beliefs, mantras, traumas and
// phrases the way back up had space for.
func deepenActor(have *genesisActor, add genesisActor) {
	ratchetRelevance(&have.Relevance, add.Relevance)
	fillIfEmpty(&have.Tag, add.Tag)
	str := func(dst *string, src string) {
		if strings.TrimSpace(*dst) == "" && strings.TrimSpace(src) != "" {
			*dst = src
		}
	}
	str(&have.Descriptor, add.Descriptor)
	str(&have.Standing, add.Standing)
	str(&have.SpeechManner, add.SpeechManner)
	str(&have.Hiding, add.Hiding)
	str(&have.Malleability, add.Malleability)
	str(&have.StartsIn, add.StartsIn)
	str(&have.Upbringing, add.Upbringing)
	str(&have.Goal, add.Goal)
	str(&have.Sacrifice, add.Sacrifice)
	if len(have.Traits) == 0 {
		have.Traits = add.Traits
	}
	have.Beliefs = appendNew(have.Beliefs, add.Beliefs)
	have.Mantras = appendNew(have.Mantras, add.Mantras)
	have.ExamplePhrases = appendNew(have.ExamplePhrases, add.ExamplePhrases)
	have.BelongsTo = appendNew(have.BelongsTo, add.BelongsTo)
	for _, tr := range add.Traumas {
		seen := false
		for _, existing := range have.Traumas {
			if existing.WhatHappened == tr.WhatHappened {
				seen = true
				break
			}
		}
		if !seen {
			have.Traumas = append(have.Traumas, tr)
		}
	}
}

func appendNew(have, add []string) []string {
	seen := make(map[string]bool, len(have))
	for _, s := range have {
		seen[strings.TrimSpace(s)] = true
	}
	for _, s := range add {
		if k := strings.TrimSpace(s); k != "" && !seen[k] {
			seen[k] = true
			have = append(have, s)
		}
	}
	return have
}

func hasActor(d *genesisDoc, name string) bool {
	for _, a := range d.Cast {
		if a.CanonicalName == name {
			return true
		}
	}
	return false
}
func hasObject(d *genesisDoc, name string) bool {
	for _, o := range d.Objects {
		if o.CanonicalName == name {
			return true
		}
	}
	return false
}
func hasWay(d *genesisDoc, w genesisWay) bool {
	for _, x := range d.Ways {
		if x.FromPlace == w.FromPlace && x.ToPlace == w.ToPlace {
			return true
		}
	}
	return false
}

type fillBreach struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Why  string `json:"why"`
}

// resolveArrivalCollision handles the arrival sharing a name with one of the world's own people.
//
// The belt refuses that, correctly: the visitor is a premise, nobody knows them and they know nothing,
// so they cannot also be a person with a life and a place. But the resolution is not to throw the world
// away. Canon is the record of what happened — append-only and immutable (D-1, I-1, I-2) — and the cast
// are embedded in it, in hands and in places. The ARRIVAL is authored last and is attached to nothing,
// so the arrival is the side that yields. The record is never edited to suit a later choice.
//
// So an alternative the seat already authored takes over: a candidate whose name is not in the cast,
// keeping the place and the opening line the arrival was authored with. Nothing is invented.
//
// Measured live 2026-08-28, twice. First the closing pass authored the visitor into the cast (fixed by
// never owing them); then the repair pass did the same thing 789 seconds in. Plugging one caller at a
// time was the wrong shape — the collision resolves itself here, whoever caused it.
func resolveArrivalCollision(doc *genesisDoc) {
	arrival := strings.TrimSpace(doc.Arrival.CanonicalName)
	if arrival == "" {
		return
	}
	// A name is TAKEN if the world's own people hold it, or if CANON uses it. Canon is the record of
	// what happened: append-only, immutable (D-1, I-1, I-2), authored before the arrival, and never
	// edited to make a later choice fit. If the newcomer's name collides with the record, the NEWCOMER
	// yields — nothing that happened becomes unhappened.
	taken := make(map[string]bool, len(doc.Cast))
	for _, a := range doc.Cast {
		taken[strings.TrimSpace(a.CanonicalName)] = true
	}
	for _, h := range doc.History {
		for _, w := range h.Who {
			taken[strings.TrimSpace(w)] = true
		}
		for _, k := range h.Knowledge {
			taken[strings.TrimSpace(k.Holder)] = true
		}
	}
	if !taken[arrival] {
		return
	}
	for _, c := range doc.ArrivalCandidates {
		name := strings.TrimSpace(c.CanonicalName)
		if name == "" || taken[name] || strings.TrimSpace(c.Descriptor) == "" {
			continue
		}
		log.Printf("resolveArrivalCollision: the visitor %q is also one of the world's people; the arrival becomes %q, an alternative the seat authored", arrival, name)
		doc.Arrival.CanonicalName = c.CanonicalName
		doc.Arrival.Descriptor = c.Descriptor
		if strings.TrimSpace(c.Why) != "" {
			doc.Arrival.Why = c.Why
		}
		return
	}
	log.Printf("resolveArrivalCollision: the visitor %q is also one of the world's people and no authored alternative is free — the belt will refuse", arrival)
}

// reconcileArrival makes the offered three coherent with the arrival instead of refusing the world for
// their disagreement.
//
// The belt's rule (worldgenesis.go) is that when arrival_candidates are present there must be exactly
// three, distinct, and exactly ONE of them must BE the arrival — the recommended default. Measured
// live 2026-08-28: a 550-second build authored a good arrival and three good candidates, none of which
// was the arrival, and the whole world was thrown away on the last check before commit.
//
// Nothing about that needed a refusal. The arrival IS the recommendation; the candidates are the
// alternatives beside it. So the arrival takes a seat among them, replacing the weakest claim to one
// (the first), and the rest stand. That is reconciliation, not invention — every string here was
// authored by the seat. Founder 2026-08-28: a gap is worse than an invention that clicks, and throwing
// away nine minutes of world over a bookkeeping mismatch is the worst gap of all.
//
// If the offer cannot be made coherent at exactly three distinct names, the LIST is dropped rather than
// the world: an incomplete offer is no offer, and the arrival alone is a legitimate, playable world.
// dropUnstorable removes entities the engine cannot store, instead of refusing the world that contains
// them.
//
// Measured live 2026-08-28: a 472-second build with 27,383 tokens of authored world was refused for
// "object 7 has no canonical_name" — one nameless thing in a long list. The schema already asks for a
// name (minLength 1); in json_object mode that ask is advisory and the Go belt is the enforcement, so a
// seat can and does slip one through.
//
// Scoped deliberately to LEAVES — objects and history events. Nothing in the document points at an
// object, and nothing points at an event, so removing one cannot orphan a reference. Places and people
// are NOT dropped here: things stand in them and canon names them, so an unnamed one is a structurally
// broken fragment and the belt should keep saying so plainly.
//
// Founder 2026-08-28: a gap is worse than an invention that clicks — and losing an entire authored
// world over one nameless prop is the worst gap available.
// normalisePersonNames title-cases a person's name that is merely uncapitalised, instead of refusing
// the world for it.
//
// identifierShapedName (worldgenesis.go) refuses a person whose name is cased script with no capital
// anywhere. It exists for a real reason with receipts — the Ironmoor breach of 2026-08-20, where
// genesis emitted slug join-keys as people's canonical names and players read them. But its heuristic
// assumes English capitalisation, and measured live 2026-08-28 it refused a 407-second Andantes build
// over "once familias" — Spanish for "eleven families", a perfectly speakable collective in a
// Spanish-language world.
//
// Capitalisation is typography, not content. So the name is normalised and every reference to it is
// rewritten with it, which is why the rename map exists: renaming a person without rewriting
// history.who and objects.where.carried_by would orphan them and refuse the world for a different
// reason one check later.
//
// UNDERSCORES ARE STILL REFUSED, deliberately. "silas_holton" is not a capitalisation slip, it is a
// machine identifier, and the naming wall treats it as a breach. Normalising it would hide the signal
// the Ironmoor incident taught us to watch for. Places and objects are untouched — the existing guard
// already exempts them.
func normalisePersonNames(doc *genesisDoc) {
	rename := map[string]string{}
	fix := func(name string) string {
		n := strings.TrimSpace(name)
		if n == "" || strings.Contains(n, "_") {
			return name
		}
		if !identifierShapedName(n) {
			return name
		}
		titled := titleCasePersonName(n)
		if titled == n {
			return name
		}
		log.Printf("normalisePersonNames: %q -> %q (uncapitalised, not a join key)", n, titled)
		rename[n] = titled
		return titled
	}

	for i := range doc.Cast {
		doc.Cast[i].CanonicalName = fix(doc.Cast[i].CanonicalName)
	}
	doc.Arrival.CanonicalName = fix(doc.Arrival.CanonicalName)
	for i := range doc.ArrivalCandidates {
		doc.ArrivalCandidates[i].CanonicalName = fix(doc.ArrivalCandidates[i].CanonicalName)
	}
	if len(rename) == 0 {
		return
	}

	apply := func(s string) string {
		if v, ok := rename[strings.TrimSpace(s)]; ok {
			return v
		}
		return s
	}
	for i := range doc.History {
		for j := range doc.History[i].Who {
			doc.History[i].Who[j] = apply(doc.History[i].Who[j])
		}
		for j := range doc.History[i].Knowledge {
			doc.History[i].Knowledge[j].Holder = apply(doc.History[i].Knowledge[j].Holder)
		}
	}
	for i := range doc.Objects {
		doc.Objects[i].Where.CarriedBy = apply(doc.Objects[i].Where.CarriedBy)
	}
}

// titleCasePersonName capitalises the first letter of each word and leaves the rest alone, so
// "once familias" becomes "Once Familias" without touching interior letters a language may care about.
func titleCasePersonName(name string) string {
	out := []rune(name)
	atWordStart := true
	for i, r := range out {
		switch {
		case unicode.IsSpace(r) || r == '-' || r == '\'':
			atWordStart = true
		case atWordStart:
			out[i] = unicode.ToUpper(r)
			atWordStart = false
		}
	}
	return string(out)
}

func dropUnstorable(doc *genesisDoc) {
	objects := make([]genesisObject, 0, len(doc.Objects))
	for i, o := range doc.Objects {
		if strings.TrimSpace(o.CanonicalName) == "" || strings.TrimSpace(o.Descriptor) == "" || strings.TrimSpace(o.Kind) == "" {
			log.Printf("dropUnstorable: object %d is missing a name, descriptor or kind — dropping it and keeping the world", i+1)
			continue
		}
		objects = append(objects, o)
	}
	doc.Objects = objects

	history := make([]genesisEvent, 0, len(doc.History))
	for i, h := range doc.History {
		if strings.TrimSpace(h.WhatHappened) == "" {
			log.Printf("dropUnstorable: history event %d has no account of what happened — dropping it", i+1)
			continue
		}
		history = append(history, h)
	}
	doc.History = history
}

func reconcileArrival(doc *genesisDoc) {
	resolveArrivalCollision(doc)
	if len(doc.ArrivalCandidates) == 0 {
		return
	}
	arrival := strings.TrimSpace(doc.Arrival.CanonicalName)
	if arrival == "" {
		doc.ArrivalCandidates = nil
		return
	}

	matches := 0
	for _, c := range doc.ArrivalCandidates {
		if strings.TrimSpace(c.CanonicalName) == arrival {
			matches++
		}
	}
	if matches == 0 {
		log.Printf("reconcileArrival: seating the arrival %q among its own candidates", arrival)
		doc.ArrivalCandidates[0] = genesisCandidate{
			Descriptor:    doc.Arrival.Descriptor,
			CanonicalName: doc.Arrival.CanonicalName,
			Why:           doc.Arrival.Why,
		}
	}

	// Distinct, arrival first, then the alternatives in the order they were authored.
	seen := map[string]bool{}
	out := make([]genesisCandidate, 0, len(doc.ArrivalCandidates))
	for _, c := range doc.ArrivalCandidates {
		name := strings.TrimSpace(c.CanonicalName)
		if name == "" || seen[name] || strings.TrimSpace(c.Descriptor) == "" || strings.TrimSpace(c.Why) == "" {
			continue
		}
		seen[name] = true
		if name == arrival {
			out = append([]genesisCandidate{c}, out...)
			continue
		}
		out = append(out, c)
	}
	if len(out) > 3 {
		// Keep the arrival and the first two alternatives.
		out = out[:3]
	}
	if len(out) != 3 || strings.TrimSpace(out[0].CanonicalName) != arrival {
		log.Printf("reconcileArrival: offer cannot be made coherent (%d distinct); dropping the list and keeping the arrival", len(out))
		doc.ArrivalCandidates = nil
		return
	}
	doc.ArrivalCandidates = out
}

func reviewFill(ctx context.Context, seat Driver, id *worldIdentity, doc *genesisDoc) ([]fillBreach, error) {
	raw, err := seat.Generate(ctx, GenRequest{
		Prompt: buildWorldFillReviewPrompt(id, doc),
		Schema: json.RawMessage(worldFillReviewSchemaJSON),
	})
	if err != nil {
		return nil, fmt.Errorf("reviewFill: Generate: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.DisallowUnknownFields()
	var out struct {
		Breaches []fillBreach `json:"breaches"`
	}
	if err := dec.Decode(&out); err != nil {
		return nil, refuse("fill review came back malformed (%v)", err)
	}
	return out.Breaches, nil
}

func buildWorldFillReviewPrompt(id *worldIdentity, doc *genesisDoc) string {
	var sb strings.Builder
	sb.WriteString(worldFillReviewSystemHeader)
	sb.WriteString("\n\n")
	sb.WriteString(worldFillReviewExclusionsMarker)
	sb.WriteString("\n")
	sb.WriteString("neighbour: ")
	sb.WriteString(id.Departure.Neighbour)
	sb.WriteString("\nhow_not: ")
	sb.WriteString(id.Departure.HowNot)
	sb.WriteString("\n")
	for _, e := range id.Exclusions {
		sb.WriteString("- never ")
		sb.WriteString(e.Never)
		sb.WriteString(" because ")
		sb.WriteString(e.Because)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(worldFillReviewNamesMarker)
	sb.WriteString("\n")
	for _, p := range doc.Places {
		sb.WriteString("- place ")
		sb.WriteString(p.CanonicalName)
		sb.WriteString("\n")
	}
	for _, a := range doc.Cast {
		sb.WriteString("- person ")
		sb.WriteString(a.CanonicalName)
		sb.WriteString("\n")
	}
	for _, o := range doc.Objects {
		sb.WriteString("- object ")
		sb.WriteString(o.CanonicalName)
		sb.WriteString("\n")
	}
	for _, w := range doc.Ways {
		sb.WriteString("- way ")
		sb.WriteString(w.Descriptor)
		sb.WriteString("\n")
	}
	for _, h := range doc.History {
		sb.WriteString("- history ")
		sb.WriteString(h.WhatHappened)
		sb.WriteString("\n")
	}
	return sb.String()
}

func retractBreaches(doc *genesisDoc, breaches []fillBreach) {
	for _, b := range breaches {
		name := strings.TrimSpace(b.Name)
		if name == "" {
			continue
		}
		switch b.Kind {
		case "place":
			var keep []genesisPlace
			for _, p := range doc.Places {
				if p.CanonicalName != name {
					keep = append(keep, p)
				}
			}
			doc.Places = keep
			var ways []genesisWay
			for _, w := range doc.Ways {
				if w.FromPlace != name && w.ToPlace != name {
					ways = append(ways, w)
				}
			}
			doc.Ways = ways
		case "cast":
			var keep []genesisActor
			for _, a := range doc.Cast {
				if a.CanonicalName != name {
					keep = append(keep, a)
				}
			}
			doc.Cast = keep
		case "object":
			var keep []genesisObject
			for _, o := range doc.Objects {
				if o.CanonicalName != name {
					keep = append(keep, o)
				}
			}
			doc.Objects = keep
		case "way":
			var keep []genesisWay
			for _, w := range doc.Ways {
				if w.Descriptor != name {
					keep = append(keep, w)
				}
			}
			doc.Ways = keep
		case "history":
			var keep []genesisEvent
			for _, h := range doc.History {
				if h.WhatHappened != name {
					keep = append(keep, h)
				}
			}
			doc.History = keep
		}
	}
}

func buildWorldUnderstandingPrompt(brief string, answers []InterviewAnswer) string {
	var sb strings.Builder
	sb.WriteString(worldUnderstandingSystemHeader)
	sb.WriteString("\n\n")
	sb.WriteString(worldIdentityBriefMarker)
	sb.WriteString("\n")
	sb.WriteString(strings.TrimSpace(brief))
	if len(answers) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(worldIdentityAnswersMarker)
		sb.WriteString("\n")
		for _, a := range answers {
			q, v := strings.TrimSpace(a.Question), strings.TrimSpace(a.Answer)
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

// fillDebts reports what canon has already referenced but nothing has authored yet: people named by
// history, and places named by history, cast or arrival.
//
// The founder's batch order is places -> key history -> lives -> objects, which means history NAMES
// people the lives batch has not written yet. That forward reference is deliberate and correct — canon
// comes before the lives who carry it — but the first version only ASKED the prompt to honour it, and
// a prompt sentence is not a guarantee. Measured live 2026-08-28: history named "Auscultadora Mayor
// Del Vas", lives never authored her, and genesisDoc.validate() refused the world 227 seconds later
// with the one permitted repair unable to fix it (it answered in 59 tokens).
//
// So the debt is carried forward in code and restated on every subsequent call until it is paid. This
// changes no ordering and removes no authority from the belt; it just stops the pipeline discovering
// four minutes late what it already knew after batch two.
func fillDebts(d *genesisDoc) (people []string, places []string) {
	cast := make(map[string]bool, len(d.Cast))
	for _, a := range d.Cast {
		cast[strings.TrimSpace(a.CanonicalName)] = true
	}
	known := make(map[string]bool, len(d.Places))
	for _, p := range d.Places {
		known[strings.TrimSpace(p.CanonicalName)] = true
	}
	// The VISITOR is never owed. They are a premise, not one of the world's people: nobody knows them
	// and they know nothing but their own arrival. Measured live 2026-08-28: history named the visitor
	// as a knowledge holder, fillDebts reported them as an unauthored person, the closing pass
	// dutifully authored them INTO the cast, and the belt then refused the world for "the player is
	// also in the cast" — a collision this pipeline manufactured for itself, 614 seconds in.
	visitor := strings.TrimSpace(d.Arrival.CanonicalName)

	seenP, seenL := map[string]bool{}, map[string]bool{}
	owePerson := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || cast[name] || seenP[name] || (visitor != "" && name == visitor) {
			return
		}
		seenP[name] = true
		people = append(people, name)
	}
	owePlace := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || known[name] || seenL[name] {
			return
		}
		seenL[name] = true
		places = append(places, name)
	}
	for _, h := range d.History {
		owePlace(h.Where)
		for _, w := range h.Who {
			owePerson(w)
		}
		for _, k := range h.Knowledge {
			owePerson(k.Holder)
		}
	}
	for _, a := range d.Cast {
		owePlace(a.StartsIn)
	}
	for _, o := range d.Objects {
		owePlace(o.Where.InPlace)
	}
	owePlace(d.Arrival.Place)
	return people, places
}

func buildWorldFillPrompt(id *worldIdentity, item workItem, brief string, answers []InterviewAnswer, soFar *genesisDoc, priorError string) string {
	var sb strings.Builder
	sb.WriteString(worldFillSystemHeader)
	sb.WriteString("\n\n")
	sb.WriteString(worldGenesisBriefMarker)
	sb.WriteString("\n")
	sb.WriteString(strings.TrimSpace(brief))
	if len(answers) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(worldGenesisAnswersMarker)
		sb.WriteString("\n")
		for _, a := range answers {
			q, v := strings.TrimSpace(a.Question), strings.TrimSpace(a.Answer)
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
	body, _ := json.Marshal(id)
	sb.WriteString("\n\n")
	sb.WriteString(worldFillIdentityMarker)
	sb.WriteString("\n")
	sb.Write(body)
	sb.WriteString("\n\n")
	sb.WriteString(worldFillWorkMarker)
	sb.WriteString("\n")
	sb.WriteString("id: ")
	sb.WriteString(item.ID)
	sb.WriteString("\nkind: ")
	sb.WriteString(item.Kind)
	if s := strings.TrimSpace(item.Subject); s != "" {
		sb.WriteString("\nthis item is about exactly one thing: \"")
		sb.WriteString(s)
		sb.WriteString("\" — author within it and nothing outside it")
	}
	if len(item.Members) > 0 {
		sb.WriteString("\nauthor exactly these and nobody else:")
		for _, m := range item.Members {
			sb.WriteString("\n  - \"")
			sb.WriteString(m)
			sb.WriteString("\" at relevance ")
			sb.WriteString(strconv.Itoa(relevanceOfPerson(soFar, m)))
		}
	}
	sb.WriteString("\ntext: ")
	sb.WriteString(item.Text)
	sb.WriteString("\ntherefore: ")
	sb.WriteString(item.Therefore)
	sb.WriteString("\n\n")
	sb.WriteString(worldFillAlreadyMarker)
	sb.WriteString("\n")
	// THE COMPILED MANDATE. A call is shown its own slice of the namespace, never the whole growing
	// document. Measured 2026-08-30: every call received the entire ALREADY AUTHORED block, so a build
	// cost ~13,300 input tokens per call and re-sent the world nineteen times — and the output was worse
	// for it, because restating what exists crowds out authoring what does not.
	//
	// The namespace calls see everything, because at that point everything is a handful of names.
	scope := item.Scope
	whole := scope.Whole
	want := func(names []string, n string) bool {
		if whole {
			return true
		}
		for _, x := range names {
			if x == n {
				return true
			}
		}
		return false
	}
	// Quote the canonical_name and put every other field on its own labelled line. The first version
	// rendered `- place <name> — <descriptor>`, and a live Andantes build (2026-08-28) came back with
	// starts_in set to the ENTIRE line: "Colegio de Auscultadores — Sede del Colegio de
	// Auscultadores, en la cima del distrito". The belt refused it 234 seconds in. The em-dash cannot
	// be a boundary when the names in a world legitimately contain dashes and commas — so the name
	// gets quotes, and nothing shares its line.
	for _, p := range soFar.Places {
		if !want(scope.Places, strings.TrimSpace(p.CanonicalName)) {
			continue
		}
		sb.WriteString("- location \"")
		sb.WriteString(p.CanonicalName)
		sb.WriteString("\"\n    relevance: ")
		sb.WriteString(strconv.Itoa(p.Relevance))
		sb.WriteString("\n    looks like: ")
		sb.WriteString(p.Descriptor)
		if w := strings.TrimSpace(p.Within); w != "" {
			sb.WriteString("\n    within: \"")
			sb.WriteString(w)
			sb.WriteString("\"")
		}
		if placeOwing(p) {
			sb.WriteString("\n    OWED: a description")
		}
		sb.WriteString("\n")
	}
	for _, f := range soFar.Factions {
		if !want(scope.Factions, strings.TrimSpace(f.CanonicalName)) {
			continue
		}
		sb.WriteString("- ")
		sb.WriteString(f.Kind)
		sb.WriteString(" \"")
		sb.WriteString(f.CanonicalName)
		sb.WriteString("\"\n    relevance: ")
		sb.WriteString(strconv.Itoa(f.Relevance))
		sb.WriteString("\n    tag: ")
		sb.WriteString(f.Tag)
		if strings.TrimSpace(f.Controls) != "" {
			sb.WriteString("\n    controls: ")
			sb.WriteString(f.Controls)
		}
		sb.WriteString("\n")
	}
	// Concepts travel with every scoped call: they are what the world argues over, they are few, and a
	// person authored without them is a person with no position on anything.
	for _, c := range soFar.Concepts {
		if !want(scope.Concepts, strings.TrimSpace(c.CanonicalName)) {
			continue
		}
		sb.WriteString("- concept \"")
		sb.WriteString(c.CanonicalName)
		sb.WriteString("\"\n    is: ")
		sb.WriteString(c.WhatItIs)
		if strings.TrimSpace(c.Contested) != "" {
			sb.WriteString("\n    contested: ")
			sb.WriteString(c.Contested)
		}
		sb.WriteString("\n")
	}
	for _, a := range soFar.Cast {
		if !want(scope.People, strings.TrimSpace(a.CanonicalName)) {
			continue
		}
		sb.WriteString("- person \"")
		sb.WriteString(a.CanonicalName)
		sb.WriteString("\"\n    relevance: ")
		sb.WriteString(strconv.Itoa(a.Relevance))
		sb.WriteString("\n    tag: ")
		sb.WriteString(a.Tag)
		sb.WriteString("\n    starts_in: \"")
		sb.WriteString(a.StartsIn)
		sb.WriteString("\"")
		if len(a.BelongsTo) > 0 {
			sb.WriteString("\n    belongs to: ")
			sb.WriteString(strings.Join(a.BelongsTo, ", "))
		}
		if strings.TrimSpace(a.Hiding) != "" {
			sb.WriteString("\n    hiding: ")
			sb.WriteString(a.Hiding)
		}
		if personOwing(a) {
			sb.WriteString("\n    OWED: the content relevance ")
			sb.WriteString(strconv.Itoa(a.Relevance))
			sb.WriteString(" demands")
		}
		sb.WriteString("\n")
	}
	if !docWorldEmpty(soFar) {
		sb.WriteString("- world named ")
		sb.WriteString(soFar.World.DisplayName)
		sb.WriteString("\n")
	}
	if strings.TrimSpace(soFar.Arrival.CanonicalName) != "" {
		sb.WriteString("- arrival \"")
		sb.WriteString(soFar.Arrival.CanonicalName)
		sb.WriteString("\"\n    in place: \"")
		sb.WriteString(soFar.Arrival.Place)
		sb.WriteString("\"\n")
	}
	if strings.TrimSpace(priorError) != "" {
		sb.WriteString("\n")
		sb.WriteString(worldFillRejectedMarker)
		sb.WriteString("\n")
		sb.WriteString(strings.TrimSpace(priorError))
		sb.WriteString("\nEmit ONLY the fields this schema names, spelled exactly as the schema spells them. Answer the same work item again.\n")
	}
	if people, places := fillDebts(soFar); len(people) > 0 || len(places) > 0 {
		sb.WriteString("\n")
		sb.WriteString(worldFillOwedMarker)
		sb.WriteString("\n")
		for _, n := range people {
			sb.WriteString("  person \"")
			sb.WriteString(n)
			sb.WriteString("\"\n")
		}
		for _, n := range places {
			sb.WriteString("  place \"")
			sb.WriteString(n)
			sb.WriteString("\"\n")
		}
	}
	return sb.String()
}

// relevanceOfPerson reads the level the scaffold assigned, so a pack prompt can name it per person
// rather than telling the model to work it out.
func relevanceOfPerson(d *genesisDoc, name string) int {
	for _, a := range d.Cast {
		if strings.TrimSpace(a.CanonicalName) == name {
			return a.Relevance
		}
	}
	return 1
}

// --- deepening ------------------------------------------------------------------------------------
//
// A later wave adds to a thing the scaffold named thin. Every one of these takes what is missing and
// NEVER overwrites what is there: the scaffold's name, kind and placement are the namespace everything
// else resolved against, so a content call that renames a location would break every reference to it.
//
// Relevance RATCHETS. A later answer may raise a level (play promotes, and a wave may find something
// matters more than the scaffold thought) but never lower one, because content already authored at the
// higher level would become unvalidatable — and ADR-P027 makes the ratchet law for play as well.

func ratchetRelevance(have *int, add int) {
	if validRelevance(add) && add > *have {
		*have = add
	}
}

func fillIfEmpty(have *string, add string) {
	if strings.TrimSpace(*have) == "" && strings.TrimSpace(add) != "" {
		*have = add
	}
}

func findPlace(d *genesisDoc, name string) *genesisPlace {
	for i := range d.Places {
		if d.Places[i].CanonicalName == name {
			return &d.Places[i]
		}
	}
	return nil
}

func findFaction(d *genesisDoc, name string) *genesisFaction {
	for i := range d.Factions {
		if d.Factions[i].CanonicalName == name {
			return &d.Factions[i]
		}
	}
	return nil
}

func findConcept(d *genesisDoc, name string) *genesisConcept {
	for i := range d.Concepts {
		if d.Concepts[i].CanonicalName == name {
			return &d.Concepts[i]
		}
	}
	return nil
}

func findObject(d *genesisDoc, name string) *genesisObject {
	for i := range d.Objects {
		if d.Objects[i].CanonicalName == name {
			return &d.Objects[i]
		}
	}
	return nil
}

func deepenPlace(have *genesisPlace, add genesisPlace) {
	ratchetRelevance(&have.Relevance, add.Relevance)
	fillIfEmpty(&have.Description, add.Description)
	fillIfEmpty(&have.Tension, add.Tension)
	fillIfEmpty(&have.Tag, add.Tag)
	// `within` is the namespace's tree. A content call may only ATTACH a place that was floating, never
	// re-parent one, or the geography wave's own scope stops meaning anything.
	fillIfEmpty(&have.Within, add.Within)
}

func deepenFaction(have *genesisFaction, add genesisFaction) {
	ratchetRelevance(&have.Relevance, add.Relevance)
	fillIfEmpty(&have.Tag, add.Tag)
	fillIfEmpty(&have.Controls, add.Controls)
	fillIfEmpty(&have.Publishes, add.Publishes)
	fillIfEmpty(&have.Buries, add.Buries)
	fillIfEmpty(&have.Goal, add.Goal)
	fillIfEmpty(&have.Sacrifice, add.Sacrifice)
	fillIfEmpty(&have.Seat, add.Seat)
}

func deepenConcept(have *genesisConcept, add genesisConcept) {
	ratchetRelevance(&have.Relevance, add.Relevance)
	fillIfEmpty(&have.Tag, add.Tag)
	fillIfEmpty(&have.Contested, add.Contested)
	fillIfEmpty(&have.TaughtBy, add.TaughtBy)
}

func deepenObject(have *genesisObject, add genesisObject) {
	ratchetRelevance(&have.Relevance, add.Relevance)
	fillIfEmpty(&have.Tag, add.Tag)
	// Placement is not deepened: an object is in exactly one somewhere, and two waves disagreeing about
	// where is a conflict the belt should see rather than a gap to fill.
}
