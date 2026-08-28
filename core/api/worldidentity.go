package main

// worldidentity.go — the understanding pass and the fill it governs.
//
// Genesis is five steps (docs/design/2026-08-26-world-identity-and-the-understanding-pass.md):
// take an input · understand the intention · transcribe what was given · fill, always governed
// by that understanding · emit the completed document. This file is steps 2 and 4. The old
// single-shot world_genesis seat is not the live path.
//
// Code schedules; the model interprets (design §7). Constraining rules run before generative
// ones (design §4). Tagging survives for scoped retraction; it is a review aid, not a gate (§7.3).
// Identity is stored beside the genesis document (design Q5 — decided here for this slice: beside,
// on world.world_identity, so play-loop minting can load it without changing world_genesis/1).
// Filling stops after the identity rules plus one sufficiency pass (design Q4 — decided here:
// one pass per rule, depth bounded by world_fill/1 array ceilings, then sufficiency if validate fails).

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

var (
	worldUnderstandingSystemHeader = mustReadUnderstandingFile("prompts/world_understanding.txt")
	worldIdentitySchemaJSON        = mustReadUnderstandingFile("schema/world_identity.v1.schema.json")
	worldFillSystemHeader          = mustReadFillFile("prompts/world_fill.txt")
	worldFillSchemaJSON            = mustReadFillFile("schema/world_fill.v1.schema.json")
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

const (
	worldIdentityBriefMarker   = "BRIEF (the user's own words — infer identity from this, invent no places or people):"
	worldIdentityAnswersMarker = "ANSWERS (the user's replies — stated, outranking inference):"
	worldFillIdentityMarker    = "IDENTITY (immutable for this genesis — every invention answers to it):"
	worldFillWorkMarker        = "WORK ITEM (answer only this):"
	worldFillAlreadyMarker     = "ALREADY AUTHORED (do not re-emit these names; deepen only if the work item demands a new position):"
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
	Scarce           string `json:"scarce,omitempty"`
	WronglyAbundant  string `json:"wrongly_abundant,omitempty"`
	Consequence      *struct {
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

func scheduleWork(id *worldIdentity) []workItem {
	order := []string{"constraining", "prohibiting", "generative", "voicing"}
	byKind := map[string][]workItem{}
	for _, r := range id.Rules {
		byKind[r.Kind] = append(byKind[r.Kind], workItem{ID: r.ID, Kind: r.Kind, Text: r.Text, Therefore: r.Therefore})
	}
	var out []workItem
	for _, k := range order {
		out = append(out, byKind[k]...)
	}
	out = append(out, workItem{
		ID:        "sufficiency",
		Kind:      "generative",
		Text:      "The arrival neighbourhood must be inhabited whether or not the bargain cares.",
		Therefore: "A visitor walks in on people, never into an empty room they must then search.",
	})
	return out
}

// authorWorld infers identity, then fills under it, and returns a document that has passed every belt check.
func authorWorld(ctx context.Context, understanding, fill Driver, brief string, answers []InterviewAnswer) (*genesisDoc, *worldIdentity, error) {
	id, err := inferIdentity(ctx, understanding, brief, answers)
	if err != nil {
		return nil, nil, err
	}
	doc, err := fillFromIdentity(ctx, fill, id, brief, answers)
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

func fillFromIdentity(ctx context.Context, seat Driver, id *worldIdentity, brief string, answers []InterviewAnswer) (*genesisDoc, error) {
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
	if err := doc.validate(); err != nil {
		frag, rerr := fillOne(ctx, seat, id, workItem{
			ID:        "repair",
			Kind:      "generative",
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

func (f *fillFragment) validate() error {
	if f.Empty {
		if strings.TrimSpace(f.WhyEmpty) == "" {
			return refuse("an empty fill must say why")
		}
		if fillHasContent(f) {
			return refuse("empty fill also carried entities")
		}
		return nil
	}
	if !fillHasContent(f) {
		return refuse("a non-empty fill invented nothing")
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
	if frag == nil || frag.Empty {
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
	for _, p := range soFar.Places {
		sb.WriteString("- place ")
		sb.WriteString(p.CanonicalName)
		sb.WriteString(" — ")
		sb.WriteString(p.Descriptor)
		sb.WriteString("\n")
	}
	for _, a := range soFar.Cast {
		sb.WriteString("- person ")
		sb.WriteString(a.CanonicalName)
		sb.WriteString(" hiding: ")
		sb.WriteString(a.Hiding)
		sb.WriteString(" starts_in: ")
		sb.WriteString(a.StartsIn)
		sb.WriteString("\n")
	}
	if !docWorldEmpty(soFar) {
		sb.WriteString("- world named ")
		sb.WriteString(soFar.World.DisplayName)
		sb.WriteString("\n")
	}
	if strings.TrimSpace(soFar.Arrival.CanonicalName) != "" {
		sb.WriteString("- arrival ")
		sb.WriteString(soFar.Arrival.CanonicalName)
		sb.WriteString(" in ")
		sb.WriteString(soFar.Arrival.Place)
		sb.WriteString("\n")
	}
	return sb.String()
}
