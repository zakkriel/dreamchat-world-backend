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
	"strings"
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
		// One retry per batch, with the rejection fed back. A single malformed answer must not throw
		// away every batch before it: measured live 2026-08-28, the revise batch spelled `places` as
		// `place` and DisallowUnknownFields ended a 147-second build. The belt is NOT loosened — the
		// fragment still has to parse and validate; it just gets told what was wrong and asked once
		// more. Costs one call in the failure case and nothing in the happy path.
		frag, err := fillOne(ctx, seat, id, item, brief, answers, doc, "")
		if err != nil && IsGenesisRefusal(err) {
			log.Printf("fill %s rejected, retrying once: %v", item.ID, err)
			frag, err = fillOne(ctx, seat, id, item, brief, answers, doc, err.Error())
		}
		if err != nil {
			return nil, err
		}
		// An unpaid reference gets ONE nudge, in context, at the batch that made it — cheapest place to
		// fix it. If the second answer still dangles we keep the better of the two anyway: the debt is
		// carried forward and the closing pass authors it. A good invention is never discarded for
		// arriving before its room.
		if bad := frag.danglingRefs(doc); len(bad) > 0 {
			log.Printf("fill %s left %d reference(s) unpaid, asking once: %s", item.ID, len(bad), strings.Join(bad, "; "))
			again, aerr := fillOne(ctx, seat, id, item, brief, answers, doc,
				"you referenced "+strings.Join(bad, "; ")+" — author those, in this same answer, alongside what you already wrote. Do not drop the reference.")
			if aerr == nil && fillHasContent(again) {
				frag = again
			}
		}
		mergeFill(doc, frag, item.ID, &tags)
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
			Text: fmt.Sprintf("Author exactly these and nothing else: %d owed person(s) and %d owed place(s), listed under STILL OWED.",
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
	detachVisitorFromCanon(doc)
	dropUnstorable(doc)
	normalisePersonNames(doc)
	reconcileArrival(doc)
	if err := doc.validate(); err != nil {
		frag, rerr := fillOne(ctx, seat, id, workItem{
			ID:        "repair",
			Kind:      "batch",
			Text:      "The belt refused the merged document: " + err.Error(),
			Therefore: "emit only what the belt is missing; do not re-author names already listed",
		}, brief, answers, doc, "")
		if rerr != nil {
			return nil, err
		}
		mergeFill(doc, frag, "repair", &tags)
		detachVisitorFromCanon(doc)
		dropUnstorable(doc)
		normalisePersonNames(doc)
		reconcileArrival(doc)
		if err2 := doc.validate(); err2 != nil {
			return nil, err2
		}
	}
	return doc, nil
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
	// only authored rooms in batch one. Unpaid references are work, so they are reported to the caller
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
func (f *fillFragment) validate() error {
	if !fillHasContent(f) {
		return nil
	}
	for _, a := range f.Cast {
		if strings.TrimSpace(a.Hiding) == "" {
			return refuse("%q has no hiding — depth is the private cost", a.CanonicalName)
		}
		// The join-key check is NOT repeated here. It lives at the belt, and normalisePersonNames runs
		// before the belt to title-case a name that is merely uncapitalised — rewriting every reference
		// to it as it goes. Checking here fired FIRST and harder: measured live 2026-08-28, an 803-second
		// build was refused for the cast name "un aprendiz de 27 años" ("a 27-year-old apprentice"),
		// which the document-level normaliser would have turned into a speakable name a moment later.
		// A fragment-level copy of a document-level rule just means the repair never gets to run.
	}
	return nil
}

// danglingRefs reports references this fragment makes that resolve NOWHERE — neither in the document
// so far nor inside the fragment itself.
//
// Why this exists at the batch instead of only at the belt. Seven of the eight live refusals on
// 2026-08-28 were one class of fault: a cross-batch reference that did not resolve. Every one was
// discovered by genesisDoc.validate() two to four minutes in, naming one bad reference at a time,
// with the batch that made it long finished. Checking here means the batch that wrote the reference is
// the batch asked to fix it, twenty seconds later, with the offending name quoted — and the existing
// one-shot retry does the asking.
//
// This is NOT a check on whether an invention is welcome. A batch that decides this world needs a
// low-tail district is doing the job fill exists to do, and the prompt now tells lives and objects to
// author the rooms they need. The only fault this reports is a reference left UNPAID — named by nobody,
// authored by nobody — which the engine cannot store and the retry is asked to fix by authoring it.
//
// PEOPLE named by history are exempt: history deliberately runs before lives and names the people lives
// will author. That forward debt is fillDebts()' job, not an error.
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

// resolveArrivalCollision handles the arrival sharing a name with one of the world's own people.
//
// The belt refuses that, correctly: the visitor is a premise, nobody knows them and they know nothing,
// so they cannot also be a person with a life and a room. But the resolution is not to throw the world
// away. detachVisitorFromCanon has already guaranteed no canon points at the visitor, which makes the
// ARRIVAL the safe side to change — the cast member is embedded in history and hands, the visitor is
// not attached to anything.
//
// So an alternative the seat already authored takes over: a candidate whose name is not in the cast,
// keeping the room and the opening line the arrival was authored with. Nothing is invented.
//
// Measured live 2026-08-28, twice. First the closing pass authored the visitor into the cast (fixed by
// never owing them); then the repair pass did the same thing 789 seconds in. Plugging one caller at a
// time was the wrong shape — the collision resolves itself here, whoever caused it.
func resolveArrivalCollision(doc *genesisDoc) {
	arrival := strings.TrimSpace(doc.Arrival.CanonicalName)
	if arrival == "" {
		return
	}
	inCast := false
	for _, a := range doc.Cast {
		if strings.TrimSpace(a.CanonicalName) == arrival {
			inCast = true
			break
		}
	}
	if !inCast {
		return
	}
	cast := make(map[string]bool, len(doc.Cast))
	for _, a := range doc.Cast {
		cast[strings.TrimSpace(a.CanonicalName)] = true
	}
	for _, c := range doc.ArrivalCandidates {
		name := strings.TrimSpace(c.CanonicalName)
		if name == "" || cast[name] || strings.TrimSpace(c.Descriptor) == "" {
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

// detachVisitorFromCanon removes the visitor from history that should never have named them.
//
// Product law, not preference: the visitor knows nothing except their own arrival and nobody in the
// world knows them (design §"THE VISITOR KNOWS NOTHING"; the belt already refuses an event that hands
// the player knowledge). An event listing them is mis-attributed rather than interesting, so the
// attribution goes and the event stays — unless the visitor was its ONLY witness, in which case the
// event cannot be true and it goes too.
//
// This runs before the belt so a premise violation costs an attribution, never the world.
func detachVisitorFromCanon(doc *genesisDoc) {
	visitor := strings.TrimSpace(doc.Arrival.CanonicalName)
	if visitor == "" {
		return
	}
	kept := make([]genesisEvent, 0, len(doc.History))
	for i, h := range doc.History {
		who := make([]string, 0, len(h.Who))
		for _, w := range h.Who {
			if strings.TrimSpace(w) == visitor {
				log.Printf("detachVisitorFromCanon: history event %d listed the visitor %q as present — removing the attribution", i+1, visitor)
				continue
			}
			who = append(who, w)
		}
		knowledge := make([]genesisKnowledge, 0, len(h.Knowledge))
		for _, k := range h.Knowledge {
			if strings.TrimSpace(k.Holder) == visitor {
				log.Printf("detachVisitorFromCanon: history event %d gave the visitor %q knowledge — removing it", i+1, visitor)
				continue
			}
			knowledge = append(knowledge, k)
		}
		if len(knowledge) == 0 {
			// The belt requires at least one event, so emptying canon here trades one refusal for
			// another. Only keep it when dropping would leave nothing at all.
			if len(kept) == 0 && i == len(doc.History)-1 {
				log.Printf("detachVisitorFromCanon: history event %d was known only to the visitor, but it is the only event this world has — keeping it for the belt to judge", i+1)
				kept = append(kept, h)
				continue
			}
			log.Printf("detachVisitorFromCanon: history event %d was known only to the visitor, which cannot be true — dropping the event", i+1)
			continue
		}
		h.Who = who
		h.Knowledge = knowledge
		kept = append(kept, h)
	}
	doc.History = kept
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
