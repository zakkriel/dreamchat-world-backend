package main

import "encoding/json"

// Grounded Reasoning / Unit 3 — THE REASONING TRACE (the behind-the-curtain log), built LAST over the
// new code (design §Unit 3). BeatTrace is a PURE CAPTURE of what the beat pipeline already computed —
// NO new LLM call, NO new decision, NO new physics. The orchestrator appends to it stage-by-stage as a
// beat runs, and beathandler serializes it into the response under `reasoning_log` ONLY when the handler
// is in debug mode.
//
// TRUTH-REVEALING → DEBUG-ONLY (the security invariant). The trace carries truth-side reasoning: the
// referee's actual read of a scene, the truth-side fact sheet handed to adjudication — it can contain a
// secret's truth. That is correct for a developer looking behind the curtain of his OWN world, and it is
// exactly why the `reasoning_log` key is ABSENT from a non-debug response. The perception wall is not
// violated (RULINGS-2026-07-23 §9): the trace is a developer affordance, never a character perception,
// and a real player never gets debug.
//
// NIL-SAFE (non-debug pays ~zero). A nil *BeatTrace is the non-debug path: every append method below is
// nil-receiver-safe (a no-op on a nil receiver), so the orchestrator threads a nil trace through the
// existing pipeline with ZERO behavior change — no appends, and the one debug-only extra read (a move's
// fact sheet) is guarded at its call site by an explicit nil check so a non-debug beat never issues it.
type BeatTrace struct {
	// Decompose is the decoded beat_chain (the raw-text → chain stage): one entry per element
	// (attempt | QUERY | UNRESOLVED) with its type, the player's stated words, and the ids it bound.
	Decompose []TraceElement `json:"decompose"`
	// WorldFirst records each present NPC's world-first cognition decision — none | commit | telegraph —
	// in processing order, before the player's attempt resolved.
	WorldFirst []TraceDecision `json:"world_first,omitempty"`
	// Attempts is the per-passthrough physics: the perceived fact sheet the engine computed for the move
	// target (distance/duration/reachability, raw jsonb) and the §6 tension-budget gate the move passed.
	Attempts []TraceAttempt `json:"attempts,omitempty"`
	// Rulings is one entry per adjudicated attempt set: the TRUTH-SIDE fact sheet handed to the referee
	// (never walled — the reason the whole trace is debug-only) and the referee's own
	// reasoning → therefore → outcome.
	Rulings []TraceRuling `json:"rulings,omitempty"`
	// Queries is one entry per read-only QUERY answered this beat: the question + the perceived fact
	// sheet it was answered from (copied from the outcome in Finish — asking is not acting).
	Queries []TraceQuery `json:"queries,omitempty"`
	// HaltReason and Committed are the beat's outcome, copied in by Finish once the run returns — the
	// "what committed and why it stopped" line of the developer view.
	HaltReason string   `json:"halt_reason"`
	Committed  []string `json:"committed"`
}

// TraceElement is one decoded chain element (the decompose stage): its type, the player's words, and
// every entity id it bound (typed slots + query targets + UNRESOLVED candidates), for the id-level view.
type TraceElement struct {
	Type   string   `json:"type"`
	Stated string   `json:"stated,omitempty"`
	IDs    []string `json:"ids,omitempty"`
}

// TraceDecision is one NPC's world-first decision: her id, what she chose ("none" — did nothing;
// "commit" — acted this moment; "telegraph" — wound up a disruptive act, held for the reaction), and the
// acted/telegraphed attempt's words (empty for "none").
type TraceDecision struct {
	ActorID  string `json:"actor_id"`
	Decision string `json:"decision"`
	Stated   string `json:"stated,omitempty"`
}

// TraceAttempt is one passthrough attempt's physics + gate. FactSheet is the raw fn_fact_sheet jsonb
// (perceived flavor) so the developer sees the engine's real numbers verbatim (distance_m,
// move_duration_s, reachable, …). The MoveDuration/Budget/Fit fields describe the §6 tension-budget gate:
// a move's computed duration, the budget before it, the budget after (if it fit), and whether it fit.
type TraceAttempt struct {
	Type          string          `json:"type"`
	Stated        string          `json:"stated,omitempty"`
	FactSheet     json.RawMessage `json:"fact_sheet,omitempty"`
	MoveDurationS int64           `json:"move_duration_s,omitempty"`
	BudgetBefore  int64           `json:"budget_remaining_before,omitempty"`
	BudgetAfter   int64           `json:"budget_remaining_after,omitempty"`
	Fit           *bool           `json:"fit,omitempty"`
}

// TraceRuling captures one adjudicate call (a single attempt or a multi-actor collision): the acting
// ids, the TRUTH-SIDE fact sheet the referee reasoned from, and the ruling's own
// reasoning → therefore → outcome.kind.
type TraceRuling struct {
	ActorIDs  []string        `json:"actor_ids,omitempty"`
	FactSheet json.RawMessage `json:"fact_sheet,omitempty"`
	Reasoning string          `json:"reasoning"`
	Therefore string          `json:"therefore"`
	Outcome   string          `json:"outcome,omitempty"`
}

// TraceQuery is one read-only QUERY answered this beat: the question + the perceived fact sheet.
type TraceQuery struct {
	Stated    string          `json:"stated"`
	FactSheet json.RawMessage `json:"fact_sheet,omitempty"`
}

// NewBeatTrace opens a trace for a beat, capturing the decompose stage (the decoded chain) up front.
// The handler builds one ONLY in debug (else it threads nil); the orchestrator fills the rest.
func NewBeatTrace(chain []Attempt) *BeatTrace {
	t := &BeatTrace{Decompose: make([]TraceElement, 0, len(chain)), Committed: []string{}}
	for _, a := range chain {
		t.Decompose = append(t.Decompose, TraceElement{Type: a.Type, Stated: a.Stated, IDs: attemptBoundIDs(a)})
	}
	return t
}

// attemptBoundIDs gathers EVERY entity id a decoded attempt bound — the typed action slots plus a
// QUERY's targets and an UNRESOLVED's candidates — de-duplicated in a stable order, for the id-level
// decompose view. (Wider than collectParticipantIDs, which is the adjudication slice-gather set.)
func attemptBoundIDs(a Attempt) []string {
	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	add(a.ToTargetID)
	add(a.ListenerID)
	add(a.ObjectID)
	add(a.DestID)
	add(a.TargetID)
	add(a.GranteeID)
	for _, id := range a.ComponentIDs {
		add(id)
	}
	for _, id := range a.QueryTargetIDs {
		add(id)
	}
	for _, id := range a.CandidateIDs {
		add(id)
	}
	return ids
}

// appendDecisions records each present NPC's world-first decision (none | commit | telegraph). No-op on
// a nil receiver (the non-debug path).
func (t *BeatTrace) appendDecisions(decisions []NPCDecision) {
	if t == nil {
		return
	}
	for _, d := range decisions {
		e := TraceDecision{ActorID: d.ActorID, Decision: "none"}
		if d.Reaction != nil {
			e.Decision = d.Reaction.CommitKind // "commit" | "telegraph"
			e.Stated = d.Reaction.Attempt.Stated
		}
		t.WorldFirst = append(t.WorldFirst, e)
	}
}

// appendMove records one passthrough MOVE's physics + §6 budget gate: the perceived fact sheet (raw
// jsonb; "" ⇒ omitted), the computed duration, the budget before it, and whether it fit (which yields
// the budget after). No-op on a nil receiver. Called ONLY when the trace is non-nil (the fact sheet is a
// debug-only read guarded at the call site), so a non-debug beat issues no extra work.
func (t *BeatTrace) appendMove(a Attempt, factSheet string, durationS, budgetBefore int64, fit bool) {
	if t == nil {
		return
	}
	after := budgetBefore
	if fit {
		after = budgetBefore - durationS
	}
	fitCopy := fit
	t.Attempts = append(t.Attempts, TraceAttempt{
		Type:          a.Type,
		Stated:        a.Stated,
		FactSheet:     rawOrNil(factSheet),
		MoveDurationS: durationS,
		BudgetBefore:  budgetBefore,
		BudgetAfter:   after,
		Fit:           &fitCopy,
	})
}

// appendRuling records one adjudicated ruling (the referee's reasoning → therefore → outcome) plus the
// TRUTH-SIDE fact sheet it reasoned from. No-op on a nil receiver. actorIDs/factSheet are the exact
// values adjudicate already assembled — pure capture.
func (t *BeatTrace) appendRuling(actorIDs []string, factSheet string, r RulingV2) {
	if t == nil {
		return
	}
	t.Rulings = append(t.Rulings, TraceRuling{
		ActorIDs:  append([]string(nil), actorIDs...),
		FactSheet: rawOrNil(factSheet),
		Reasoning: r.Reasoning,
		Therefore: r.Therefore,
		Outcome:   r.Outcome.Kind,
	})
}

// Finish copies the beat's terminal state off the outcome once the run returns: the halt reason, the
// committed ids, and the read-only query answers (asking is not acting — the QUERY fact sheets are
// accumulated on the outcome, so the trace mirrors them here rather than at a second append site). No-op
// on a nil receiver.
func (t *BeatTrace) Finish(out BeatOutcome) {
	if t == nil {
		return
	}
	t.HaltReason = out.HaltReason
	if len(out.Committed) > 0 {
		t.Committed = append([]string(nil), out.Committed...)
	}
	for _, qa := range out.QueryAnswers {
		t.Queries = append(t.Queries, TraceQuery{Stated: qa.Stated, FactSheet: qa.FactSheet})
	}
}

// rawOrNil turns a raw jsonb string into a json.RawMessage, or nil when empty (so the field is omitted
// rather than serialized as invalid empty JSON).
func rawOrNil(s string) json.RawMessage {
	if s == "" {
		return nil
	}
	return json.RawMessage(s)
}
