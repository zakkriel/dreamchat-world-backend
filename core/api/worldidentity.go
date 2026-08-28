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
	"strings"
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
	kinds := map[string]bool{"constraining": true, "generative": true, "prohibiting": true, "voicing": true}
	seen := map[string]bool{}
	for _, r := range id.Rules {
		if !kinds[r.Kind] {
			return refuse("rule %q has unknown kind %q", r.ID, r.Kind)
		}
		if strings.TrimSpace(r.ID) == "" || seen[r.ID] {
			return refuse("rule ids must be unique and non-empty")
		}
		seen[r.ID] = true
		if strings.TrimSpace(r.Therefore) == "" {
			return refuse("rule %q has no therefore", r.ID)
		}
	}
	return nil
}

func scheduleWork(_ *worldIdentity) []workItem {
	return []workItem{
		{ID: "places", Kind: "batch", Text: "Author the places this identity demands, and the ways between them.", Therefore: "geography exists before canon, lives, or objects."},
		{ID: "history", Kind: "batch", Text: "Author the key history that already happened in those places.", Therefore: "canon exists before the lives who still carry it."},
		{ID: "lives", Kind: "batch", Text: "Author the lives at a specific angle on the pressure, including anyone history named.", Therefore: "depth is a private cost, not a profession."},
		{ID: "objects", Kind: "batch", Text: "Author the objects that belong in those rooms and hands.", Therefore: "a body has something to touch."},
		{ID: "revise", Kind: "batch", Text: "Second pass: leftover positions and anything the first pass left thin.", Therefore: "depth is a second look, not a bigger first dump."},
		{ID: "sufficiency", Kind: "batch", Text: "The arrival neighbourhood must be inhabited whether or not the bargain cares.", Therefore: "A visitor walks in on people, never into an empty room they must then search."},
	}
}

// authorWorld infers identity, then fills under it, and returns a document that has passed every belt check.
func authorWorld(ctx context.Context, understanding, fill, review Driver, brief string, answers []InterviewAnswer, confirmed json.RawMessage, voice []string) (*genesisDoc, *worldIdentity, error) {
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
	doc, err := fillFromIdentity(ctx, fill, review, id, brief, answers)
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

func fillFromIdentity(ctx context.Context, seat, review Driver, id *worldIdentity, brief string, answers []InterviewAnswer) (*genesisDoc, error) {
	if seat == nil {
		return nil, fmt.Errorf("fillFromIdentity: no world_fill seat bound")
	}
	doc := &genesisDoc{}
	var tags []taggedName
	for _, item := range scheduleWork(id) {
		frag, err := fillOne(ctx, seat, id, item, brief, answers, doc)
		if err != nil {
			return nil, err
		}
		mergeFill(doc, frag, item.ID, &tags)
	}
	if review != nil {
		breaches, err := reviewFill(ctx, review, id, doc)
		if err != nil {
			return nil, err
		}
		retractBreaches(doc, breaches)
	}
	if err := doc.validate(); err != nil {
		frag, rerr := fillOne(ctx, seat, id, workItem{
			ID:        "repair",
			Kind:      "batch",
			Text:      "The belt refused the merged document: " + err.Error(),
			Therefore: "emit only what the belt is missing; do not re-author names already listed",
		}, brief, answers, doc)
		if rerr != nil {
			return nil, err
		}
		mergeFill(doc, frag, "repair", &tags)
		if err2 := doc.validate(); err2 != nil {
			return nil, err2
		}
	}
	return doc, nil
}

func fillOne(ctx context.Context, seat Driver, id *worldIdentity, item workItem, brief string, answers []InterviewAnswer, soFar *genesisDoc) (*fillFragment, error) {
	raw, err := seat.Generate(ctx, GenRequest{
		Prompt: buildWorldFillPrompt(id, item, brief, answers, soFar),
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
func (f *fillFragment) validate() error {
	if !fillHasContent(f) {
		return nil
	}
	for _, a := range f.Cast {
		if strings.TrimSpace(a.Hiding) == "" {
			return refuse("%q has no hiding — depth is the private cost", a.CanonicalName)
		}
		if identifierShapedName(strings.TrimSpace(a.CanonicalName)) {
			return refuse("%q reads like a join key, not a person's name", a.CanonicalName)
		}
	}
	return nil
}

func fillHasContent(f *fillFragment) bool {
	return len(f.WorldRaw) > 0 || len(f.RegionRaw) > 0 || len(f.Places) > 0 || len(f.Ways) > 0 ||
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
	for _, p := range frag.Places {
		if !hasPlace(doc, p.CanonicalName) {
			doc.Places = append(doc.Places, p)
			*tags = append(*tags, taggedName{Kind: "place", Name: p.CanonicalName, Rule: ruleID})
		}
	}
	for _, w := range frag.Ways {
		if !hasWay(doc, w) {
			doc.Ways = append(doc.Ways, w)
			*tags = append(*tags, taggedName{Kind: "way", Name: w.Descriptor, Rule: ruleID})
		}
	}
	for _, a := range frag.Cast {
		if !hasActor(doc, a.CanonicalName) {
			doc.Cast = append(doc.Cast, a)
			*tags = append(*tags, taggedName{Kind: "cast", Name: a.CanonicalName, Rule: ruleID})
		}
	}
	for _, o := range frag.Objects {
		if !hasObject(doc, o.CanonicalName) {
			doc.Objects = append(doc.Objects, o)
			*tags = append(*tags, taggedName{Kind: "object", Name: o.CanonicalName, Rule: ruleID})
		}
	}
	doc.History = append(doc.History, frag.History...)
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
	seenP, seenL := map[string]bool{}, map[string]bool{}
	owePerson := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || cast[name] || seenP[name] {
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

func buildWorldFillPrompt(id *worldIdentity, item workItem, brief string, answers []InterviewAnswer, soFar *genesisDoc) string {
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
	sb.WriteString("\ntext: ")
	sb.WriteString(item.Text)
	sb.WriteString("\ntherefore: ")
	sb.WriteString(item.Therefore)
	sb.WriteString("\n\n")
	sb.WriteString(worldFillAlreadyMarker)
	sb.WriteString("\n")
	// Quote the canonical_name and put every other field on its own labelled line. The first version
	// rendered `- place <name> — <descriptor>`, and a live Andantes build (2026-08-28) came back with
	// starts_in set to the ENTIRE line: "Colegio de Auscultadores — Sede del Colegio de
	// Auscultadores, en la cima del distrito". The belt refused it 234 seconds in. The em-dash cannot
	// be a boundary when the names in a world legitimately contain dashes and commas — so the name
	// gets quotes, and nothing shares its line.
	for _, p := range soFar.Places {
		sb.WriteString("- place \"")
		sb.WriteString(p.CanonicalName)
		sb.WriteString("\"\n    looks like: ")
		sb.WriteString(p.Descriptor)
		sb.WriteString("\n")
	}
	for _, a := range soFar.Cast {
		sb.WriteString("- person \"")
		sb.WriteString(a.CanonicalName)
		sb.WriteString("\"\n    hiding: ")
		sb.WriteString(a.Hiding)
		sb.WriteString("\n    starts_in: \"")
		sb.WriteString(a.StartsIn)
		sb.WriteString("\"\n")
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
