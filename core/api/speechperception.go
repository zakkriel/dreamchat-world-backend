package main

// Governed-by: ADR-038 — code supplies facts, resolve interprets speech, core records the result.
import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed schema/speech_perception.v1.schema.json
var speechPerceptionSchemaJSON string

//go:embed prompts/speech_perception.txt
var speechPerceptionSystemHeader string

// Only identity fields are decoded. Knowledge, activities and physical facts reach resolve unchanged.
type speechPerceptionRef struct {
	ActorID string `json:"actor_id"`
}
type speechPerceptionFactsListener struct {
	ActorID    string                `json:"actor_id"`
	IsPlayer   bool                  `json:"is_player"`
	References []speechPerceptionRef `json:"references"`
}
type speechPerceptionFactsDoc struct {
	SchemaVersion string                          `json:"schema_version"`
	Listeners     []speechPerceptionFactsListener `json:"listeners"`
}
type SpeechAttention struct {
	Kind   string  `json:"kind"`
	Reason *string `json:"reason,omitempty"`
}
type SpeechNameOwner struct {
	ActorID       *string  `json:"actor_id"`
	Description   *string  `json:"description"`
	AboutActorIDs []string `json:"about_actor_ids"`
}

// Raw nullable fields distinguish an omitted judgment from an explicit null.
// Listener-specific reference and cross-field checks stay in the outer validator.
func (o *SpeechNameOwner) UnmarshalJSON(data []byte) error {
	var fields struct {
		ActorID       json.RawMessage `json:"actor_id"`
		Description   json.RawMessage `json:"description"`
		AboutActorIDs []string        `json:"about_actor_ids"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("owner: %w", err)
	}
	if fields.ActorID == nil || fields.Description == nil || fields.AboutActorIDs == nil {
		return fmt.Errorf("owner requires actor_id, description and an about_actor_ids array")
	}
	var decoded SpeechNameOwner
	if err := json.Unmarshal(fields.ActorID, &decoded.ActorID); err != nil {
		return fmt.Errorf("owner.actor_id: %w", err)
	}
	if err := json.Unmarshal(fields.Description, &decoded.Description); err != nil {
		return fmt.Errorf("owner.description: %w", err)
	}
	decoded.AboutActorIDs = fields.AboutActorIDs
	*o = decoded
	return nil
}

type SpeechNameAssociation struct {
	Name  string          `json:"name"`
	Owner SpeechNameOwner `json:"owner"`
}
type SpeechListenerJudgment struct {
	ListenerID       string                  `json:"listener_id"`
	Attention        SpeechAttention         `json:"attention"`
	NameAssociations []SpeechNameAssociation `json:"name_associations"`
	HeardWords       *string                 `json:"heard_words"`
}
type SpeechPerceptionResult struct {
	SchemaVersion string                   `json:"schema_version"`
	Listeners     []SpeechListenerJudgment `json:"listeners"`
}

// This checks application structure, not the correctness of attention or identification.
func DecodeAndValidateSpeechPerception(raw string, facts speechPerceptionFactsDoc) (SpeechPerceptionResult, error) {
	var result SpeechPerceptionResult
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, fmt.Errorf("speech perception JSON: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return result, fmt.Errorf("speech perception contains trailing JSON")
	}
	if result.SchemaVersion != "speech_perception/1" || result.Listeners == nil {
		return result, fmt.Errorf("speech perception requires schema_version speech_perception/1 and a listeners array")
	}
	candidates := make(map[string]speechPerceptionFactsListener, len(facts.Listeners))
	for _, listener := range facts.Listeners {
		candidates[listener.ActorID] = listener
	}
	seen := make(map[string]bool, len(result.Listeners))
	for _, judgment := range result.Listeners {
		listener, ok := candidates[judgment.ListenerID]
		if !ok || seen[judgment.ListenerID] {
			return result, fmt.Errorf("speech perception has an unknown or duplicate listener %q", judgment.ListenerID)
		}
		seen[judgment.ListenerID] = true
		if judgment.HeardWords == nil || judgment.NameAssociations == nil {
			return result, fmt.Errorf("listener %s requires heard_words and name_associations", listener.ActorID)
		}
		switch judgment.Attention.Kind {
		case "abstain":
			if judgment.Attention.Reason != nil {
				return result, fmt.Errorf("abstaining listener %s carries a block reason", listener.ActorID)
			}
		case "blocked":
			if listener.IsPlayer || judgment.Attention.Reason == nil || strings.TrimSpace(*judgment.Attention.Reason) == "" || *judgment.HeardWords != "" || len(judgment.NameAssociations) != 0 {
				return result, fmt.Errorf("listener %s has an incompatible attentional block", listener.ActorID)
			}
		default:
			return result, fmt.Errorf("listener %s has an unknown attention kind", listener.ActorID)
		}
		if strings.TrimSpace(*judgment.HeardWords) == "" && len(judgment.NameAssociations) > 0 {
			return result, fmt.Errorf("listener %s cannot learn names from no heard words", listener.ActorID)
		}
		references := make(map[string]bool, len(listener.References))
		for _, reference := range listener.References {
			references[reference.ActorID] = true
		}
		for _, association := range judgment.NameAssociations {
			if strings.TrimSpace(association.Name) == "" {
				return result, fmt.Errorf("listener %s has an empty associated name", listener.ActorID)
			}
			owner := association.Owner
			if owner.ActorID != nil && !references[*owner.ActorID] {
				return result, fmt.Errorf("listener %s has an unrecognized owner actor", listener.ActorID)
			}
			if owner.Description != nil && strings.TrimSpace(*owner.Description) == "" {
				return result, fmt.Errorf("listener %s has a blank owner description", listener.ActorID)
			}
			if owner.ActorID == nil && owner.Description == nil {
				return result, fmt.Errorf("listener %s has an owner with neither recognition nor a description", listener.ActorID)
			}
			if len(owner.AboutActorIDs) > 0 && owner.Description == nil {
				return result, fmt.Errorf("listener %s has related actors with no description", listener.ActorID)
			}
			for _, anchor := range owner.AboutActorIDs {
				if !references[anchor] {
					return result, fmt.Errorf("listener %s has an unsupported related actor", listener.ActorID)
				}
			}
		}
	}
	if len(seen) != len(candidates) {
		return result, fmt.Errorf("speech perception covers %d listeners, expected %d", len(seen), len(candidates))
	}
	return result, nil
}

type speechPerceptionInput struct {
	SpeakerID        string
	ListenerID       string
	Content          string
	Account          string
	Appearance       string
	ReceiverVariants []ReceiverVariant
}

func buildSpeechPerceptionPrompt(factsRaw string, facts speechPerceptionFactsDoc, in speechPerceptionInput) string {
	var prompt strings.Builder
	prompt.WriteString(speechPerceptionSystemHeader)
	fmt.Fprintf(&prompt, "\n\nEVENT:\nspeaker=%s\nlistener=%s\n", in.SpeakerID, in.ListenerID)
	words, _ := json.Marshal(in.Content)
	fmt.Fprintf(&prompt, "SPOKEN: %s\nACCOUNT: %s\n", words, in.Account)
	if in.Appearance != "" {
		fmt.Fprintf(&prompt, "DEFAULT APPEARANCE: %s\n", in.Appearance)
	}
	if len(in.ReceiverVariants) > 0 {
		prompt.WriteString("\nRECEIVER PERSPECTIVE:\n")
		for _, variant := range in.ReceiverVariants {
			fmt.Fprintf(&prompt, "- %s: %s\n", variant.ReceiverID, variant.Text)
		}
	}
	prompt.WriteString("\nFACTS (speech_perception_facts/1):\n")
	prompt.WriteString(factsRaw)
	prompt.WriteString("\n\nCANDIDATE LISTENERS (answer exactly once for each):\n")
	for _, listener := range facts.Listeners {
		fmt.Fprintf(&prompt, "- %s\n", listener.ActorID)
	}
	return prompt.String()
}

func fetchSpeechPerceptionFacts(ctx context.Context, q dbQuerier, worldID, speakerID, listenerID string) (string, speechPerceptionFactsDoc, error) {
	var raw []byte
	if err := q.QueryRow(ctx, `SELECT fn_speech_perception_facts($1::uuid,$2::uuid,$3::uuid,$4,$5)`, worldID, speakerID, listenerID, recencyTickWindow, recencyMaxRows).Scan(&raw); err != nil {
		return "", speechPerceptionFactsDoc{}, err
	}
	var facts speechPerceptionFactsDoc
	if err := json.Unmarshal(raw, &facts); err != nil {
		return "", facts, err
	}
	if facts.SchemaVersion != "speech_perception_facts/1" || facts.Listeners == nil {
		return "", facts, fmt.Errorf("invalid speech perception facts")
	}
	return string(raw), facts, nil
}

func (o *Orchestrator) judgeSpeechPerception(ctx context.Context, q dbQuerier, worldID string, in speechPerceptionInput) (json.RawMessage, error) {
	factsRaw, facts, err := fetchSpeechPerceptionFacts(ctx, q, worldID, in.SpeakerID, in.ListenerID)
	if err != nil {
		return nil, err
	}
	if len(facts.Listeners) == 0 {
		return json.RawMessage(`{"schema_version":"speech_perception/1","listeners":[]}`), nil
	}
	if o.Resolve == nil {
		return nil, fmt.Errorf("speech perception requires a resolve driver")
	}
	raw, err := o.Resolve.Generate(ctx, GenRequest{Schema: json.RawMessage(speechPerceptionSchemaJSON), Prompt: buildSpeechPerceptionPrompt(factsRaw, facts, in)})
	if err != nil {
		return nil, fmt.Errorf("speech perception Generate: %w", err)
	}
	judgment, err := DecodeAndValidateSpeechPerception(raw, facts)
	if err != nil {
		return nil, err
	}
	// SQL must consume the same field names and values the Go decoder accepted.
	return json.Marshal(judgment)
}

func (o *Orchestrator) attachSpeechPerceptionRuled(ctx context.Context, tx pgx.Tx, worldID string, event RuledEventV2) (RuledEventV2, error) {
	if event.Type != "Communicated" {
		return event, nil
	}
	event.SpeechPerception = nil
	if event.Visible != nil && !*event.Visible {
		return event, nil
	}
	judgment, err := o.judgeSpeechPerception(ctx, tx, worldID, speechPerceptionInput{SpeakerID: event.ActorID, ListenerID: event.ListenerID, Content: event.Content, Account: event.Truth, Appearance: event.Appearance, ReceiverVariants: event.ReceiverVariants})
	if err != nil {
		return event, err
	}
	event.SpeechPerception = judgment
	return event, nil
}

type cognitionPerception struct {
	Content    string
	SubjectIDs []string
}

// Committed perceptions trigger cognition after a reaction, whether or not anyone spoke.
func (o *Orchestrator) postCommittedCognition(ctx context.Context, worldID, playerID string, eventIDs []string, tick int64, seq int, trace *BeatTrace) (worldFirstResult, error) {
	if len(eventIDs) == 0 {
		return worldFirstResult{}, nil
	}
	rows, err := o.DB.Query(ctx, `SELECT pr.holder_id::text, pr.content,
	    COALESCE(listener.entity_id::text, ''),
	    COALESCE(fn_display_name(pr.world_id, pr.holder_id, listener.entity_id), ''),
	    ARRAY(SELECT ps.entity_id::text FROM perception_subject ps WHERE ps.perception_id = pr.perception_id)
	  FROM unnest($2::uuid[]) WITH ORDINALITY source(event_id, position)
	  JOIN perception_record pr ON pr.source_event_id = source.event_id
	  JOIN canon_event ce ON ce.event_id = source.event_id
	  LEFT JOIN event_participant listener ON listener.event_id = source.event_id
	    AND listener.role_qualifier = 'listener'
	  WHERE pr.world_id = $1::uuid AND pr.holder_id <> $3::uuid
	    AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
	    AND NOT (ce.event_type IN ('Communicated', 'private_disclosure') AND pr.epistemic_type = 'shared')
	    AND ce.status = 'accepted'
	    AND ce.payload->>'visible' IS DISTINCT FROM 'false'
	  ORDER BY source.position, pr.perception_id`, worldID, eventIDs, playerID)
	if err != nil {
		return worldFirstResult{}, err
	}
	defer rows.Close()
	parts := make(map[string][]string)
	perceived := make(map[string]cognitionPerception)
	for rows.Next() {
		var holder, content, listener, label string
		var subjects []string
		if err := rows.Scan(&holder, &content, &listener, &label, &subjects); err != nil {
			return worldFirstResult{}, err
		}
		if listener != "" {
			content += "\nThis speech was addressed to " + label + " (" + listener + ")."
		}
		parts[holder] = append(parts[holder], content)
		context := perceived[holder]
		context.SubjectIDs = append(context.SubjectIDs, subjects...)
		perceived[holder] = context
	}
	if err := rows.Err(); err != nil {
		return worldFirstResult{}, err
	}
	rows.Close()
	if len(parts) == 0 {
		return worldFirstResult{}, nil
	}
	for holder, contents := range parts {
		context := perceived[holder]
		context.Content = strings.Join(contents, "\n")
		slices.Sort(context.SubjectIDs)
		context.SubjectIDs = slices.Compact(context.SubjectIDs)
		perceived[holder] = context
	}
	return o.runCognition(ctx, worldID, playerID, playerID, Attempt{}, perceived, tick, seq, trace)
}
