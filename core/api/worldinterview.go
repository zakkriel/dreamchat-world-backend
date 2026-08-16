package main

// worldinterview.go — the Custom Creation lane: asking about a brief, one question at a time.
//
// STATELESS BY DESIGN. There is no interview table, no session and nothing to resume: the client carries
// the brief and every answer so far on each request, and the seat is asked what is still worth asking.
// That keeps the whole lane out of the schema, and it means an abandoned interview leaves nothing behind —
// which is the correct amount of debris for a conversation the user walked away from.
//
// The seat may say "nothing worth asking" at any point, including on the very first call, and that is a
// good answer rather than a failure: a three-paragraph brief may genuinely need no questions. The surface
// also lets the user build at any time, so this is advisory in both directions.

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed prompts/world_interview.txt schema/world_interview.v1.schema.json
var worldInterviewFS embed.FS

var (
	worldInterviewSystemHeader = mustReadInterviewFile("prompts/world_interview.txt")
	worldInterviewSchemaJSON   = mustReadInterviewFile("schema/world_interview.v1.schema.json")
)

func mustReadInterviewFile(name string) string {
	b, err := worldInterviewFS.ReadFile(name)
	if err != nil {
		panic("worldinterview: embed " + name + ": " + err.Error())
	}
	return string(b)
}

// InterviewOption is one offered answer. `Recommended` marks the default the surface highlights; the free
// text field the user can type instead is a property of the surface and never appears here.
type InterviewOption struct {
	Label       string `json:"label"`
	Implication string `json:"implication,omitempty"`
	Recommended bool   `json:"recommended,omitempty"`
}

// InterviewTurn is world_interview/1 decoded: either the next question, or nothing left to ask.
type InterviewTurn struct {
	Done     bool              `json:"done"`
	Question string            `json:"question,omitempty"`
	Why      string            `json:"why,omitempty"`
	Options  []InterviewOption `json:"options,omitempty"`
}

// askNextQuestion runs the interview seat once. A seat error or a malformed answer is NOT fatal to the
// journey: the honest fallback is "nothing more to ask", which drops the user into a build they can still
// run with what they have already said. Failing the whole lane because one question could not be authored
// would punish the user for an infrastructure problem they cannot see or fix.
func askNextQuestion(ctx context.Context, seat Driver, brief string, answers []InterviewAnswer) (InterviewTurn, error) {
	if seat == nil {
		return InterviewTurn{}, fmt.Errorf("askNextQuestion: no world_interview seat bound")
	}
	brief = strings.TrimSpace(brief)
	if brief == "" {
		return InterviewTurn{}, refuse("the brief is empty — there is nothing to ask about")
	}

	raw, err := seat.Generate(ctx, GenRequest{
		Prompt: buildWorldInterviewPrompt(brief, answers),
		Schema: json.RawMessage(worldInterviewSchemaJSON),
	})
	if err != nil {
		return InterviewTurn{Done: true}, fmt.Errorf("askNextQuestion: Generate: %w", err)
	}

	var turn InterviewTurn
	if err := json.Unmarshal([]byte(raw), &turn); err != nil {
		return InterviewTurn{Done: true}, fmt.Errorf("askNextQuestion: decode: %w", err)
	}

	// The belt: a question with no options, or an empty question, is not askable. Treat it as done rather
	// than render an unanswerable prompt — the surface must never show a question the user cannot act on.
	if turn.Done || strings.TrimSpace(turn.Question) == "" || len(turn.Options) == 0 {
		return InterviewTurn{Done: true}, nil
	}
	kept := make([]InterviewOption, 0, len(turn.Options))
	recommended := false
	for _, o := range turn.Options {
		if strings.TrimSpace(o.Label) == "" {
			continue
		}
		// At most one recommendation: two defaults is no default, and the surface highlights exactly one.
		if o.Recommended && recommended {
			o.Recommended = false
		}
		if o.Recommended {
			recommended = true
		}
		kept = append(kept, o)
	}
	if len(kept) == 0 {
		return InterviewTurn{Done: true}, nil
	}
	turn.Options = kept
	turn.Done = false
	return turn, nil
}

// buildWorldInterviewPrompt renders the standing rulebook, then the brief, then what has been asked and
// answered already — so the seat can see for itself what it must not ask again.
func buildWorldInterviewPrompt(brief string, answers []InterviewAnswer) string {
	var sb strings.Builder
	sb.WriteString(worldInterviewSystemHeader)
	sb.WriteString("\n\n")
	sb.WriteString(worldGenesisBriefMarker)
	sb.WriteString("\n")
	sb.WriteString(strings.TrimSpace(brief))
	sb.WriteString("\n\n")
	if len(answers) == 0 {
		sb.WriteString("ALREADY ASKED: nothing yet. This is the first question.\n")
		return sb.String()
	}
	sb.WriteString("ALREADY ASKED AND ANSWERED (never ask any of this again, and let it narrow what is still open):\n")
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
	return sb.String()
}
