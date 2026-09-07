package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestSpeechPerception_WhitespaceCannotTeachName(t *testing.T) {
	const listener = "00000000-0000-0000-0000-000000000001"
	const owner = "00000000-0000-0000-0000-000000000002"
	facts := speechPerceptionFactsDoc{Listeners: []speechPerceptionFactsListener{{
		ActorID: listener, References: []speechPerceptionRef{{ActorID: owner}},
	}}}
	for _, heard := range []string{"   ", "Jonas"} {
		raw := fmt.Sprintf(`{"schema_version":"speech_perception/1","listeners":[{"listener_id":%q,"attention":{"kind":"abstain"},"heard_words":%q,"name_associations":[{"name":"Jonas","owner":{"actor_id":%q,"description":null,"about_actor_ids":[]}}]}]}`, listener, heard, owner)
		_, err := DecodeAndValidateSpeechPerception(raw, facts)
		if heard == "   " && err == nil {
			t.Fatal("a judgment with no heard words was allowed to teach a name")
		}
		if heard == "Jonas" && err != nil {
			t.Fatalf("structurally consistent judgment rejected: %v", err)
		}
	}
}

// The flat owner contract: actor_id, description and about_actor_ids are independent and each
// required to be present — null is a value, omission is not (SpeechNameOwner.UnmarshalJSON) — and
// the cross-field rules (recognition membership, non-blank description, at least one of
// actor_id/description, related references requiring both membership and a description) are
// DecodeAndValidateSpeechPerception's. This proves the Go decoder applies that contract; it says
// nothing about which shape a model would actually choose to emit.
func TestSpeechPerception_OwnerContractStructuralChecks(t *testing.T) {
	const listenerID = "10000000-0000-0000-0000-000000000001"
	const recognized = "10000000-0000-0000-0000-000000000002"
	const related = "10000000-0000-0000-0000-000000000003"
	const stranger = "10000000-0000-0000-0000-000000000004"
	facts := speechPerceptionFactsDoc{Listeners: []speechPerceptionFactsListener{{
		ActorID:    listenerID,
		References: []speechPerceptionRef{{ActorID: recognized}, {ActorID: related}},
	}}}
	build := func(owner string) string {
		return fmt.Sprintf(`{"schema_version":"speech_perception/1","listeners":[{"listener_id":%q,"attention":{"kind":"abstain"},"heard_words":"Jonas is here.","name_associations":[{"name":"Jonas","owner":%s}]}]}`, listenerID, owner)
	}

	accepted := map[string]string{
		"recognizedAlone":           fmt.Sprintf(`{"actor_id":%q,"description":null,"about_actor_ids":[]}`, recognized),
		"recognizedPlusDescription": fmt.Sprintf(`{"actor_id":%q,"description":"Mara's brother","about_actor_ids":[%q]}`, recognized, related),
		"describedAlone":            fmt.Sprintf(`{"actor_id":null,"description":"the muscle by the bar","about_actor_ids":[%q]}`, related),
	}
	for name, owner := range accepted {
		t.Run("accepts_"+name, func(t *testing.T) {
			if _, err := DecodeAndValidateSpeechPerception(build(owner), facts); err != nil {
				t.Fatalf("structurally valid owner rejected: %v", err)
			}
		})
	}

	rejected := map[string]string{
		"actorIDKeyMissing":                `{"description":"a stranger passing through","about_actor_ids":[]}`,
		"descriptionKeyMissing":            fmt.Sprintf(`{"actor_id":%q,"about_actor_ids":[]}`, recognized),
		"aboutActorIDsKeyMissing":          fmt.Sprintf(`{"actor_id":%q,"description":null}`, recognized),
		"aboutActorIDsExplicitNull":        fmt.Sprintf(`{"actor_id":%q,"description":null,"about_actor_ids":null}`, recognized),
		"obsoleteKindDiscriminator":        fmt.Sprintf(`{"kind":"recognized_actor","actor_id":%q,"description":null,"about_actor_ids":[]}`, recognized),
		"neitherRecognitionNorDescription": `{"actor_id":null,"description":null,"about_actor_ids":[]}`,
		"blankDescription":                 `{"actor_id":null,"description":"   ","about_actor_ids":[]}`,
		"relatedActorWithNoDescription":    fmt.Sprintf(`{"actor_id":%q,"description":null,"about_actor_ids":[%q]}`, recognized, related),
		"unsupportedRelatedActor":          fmt.Sprintf(`{"actor_id":null,"description":"a face in the crowd","about_actor_ids":[%q]}`, stranger),
		"unrecognizedOwnerActor":           fmt.Sprintf(`{"actor_id":%q,"description":null,"about_actor_ids":[]}`, stranger),
	}
	for name, owner := range rejected {
		t.Run("rejects_"+name, func(t *testing.T) {
			if _, err := DecodeAndValidateSpeechPerception(build(owner), facts); err == nil {
				t.Fatalf("structurally invalid owner %s accepted", owner)
			}
		})
	}
}

func TestSpeechPerception_AcceptedFieldsReachApplication(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	id := setupFlowWorld(t, ctx, pool)
	const spoken = "Words accepted by the decoder."
	resolve := &scriptedCognitionDriver{name: "speech", body: fmt.Sprintf(
		`{"schema_version":"speech_perception/1","listeners":[{"listener_id":%q,"Attention":{"kind":"abstain"},"Heard_Words":%q,"name_associations":[]},{"listener_id":%q,"attention":{"kind":"abstain"},"heard_words":%q,"name_associations":[]}]}`,
		id.M, spoken, id.J, spoken)}
	orc := &Orchestrator{DB: pool, Resolve: resolve}
	tick := flowBaseTick(t, ctx, pool, id.World)
	_, err := orc.applyEvent(ctx, id.World, id.P, []byte(fmt.Sprintf(
		`{"type":"Communicated","stated":"Player speaks.","listener_id":%q,"content":%q}`, id.M, spoken)), tick, 0)
	if err != nil {
		t.Fatal(err)
	}
	var heard string
	if err := pool.QueryRow(ctx, `SELECT spoken FROM fn_perceived_speech($1,$2,$3)`,
		id.World, id.M, tick).Scan(&heard); err != nil {
		t.Fatalf("accepted listener judgment produced no heard speech: %v", err)
	}
	if heard != spoken {
		t.Fatalf("heard %q, want accepted words %q", heard, spoken)
	}
}

// Prescribed judgments test application parity, not model interpretation.
func TestSpeechPerception_VisibleDoorsShareRecognition(t *testing.T) {
	for _, door := range []string{"ordinary", "ruled"} {
		t.Run(door, func(t *testing.T) {
			pool := testPool(t)
			defer pool.Close()
			ctx := context.Background()
			id := setupFlowWorld(t, ctx, pool)
			var hidden bool
			if err := pool.QueryRow(ctx, `SELECT fn_actor_page($1,$2,$3) IS NULL`, id.World, id.P, id.J).Scan(&hidden); err != nil || !hidden {
				t.Fatalf("fixture actor already visible, error=%v", err)
			}
			const spoken = "This is Jonas, my brother."
			const description = "Mara's brother"
			resolve := &scriptedCognitionDriver{name: "speech", body: fmt.Sprintf(
				`{"schema_version":"speech_perception/1","listeners":[{"listener_id":%q,"attention":{"kind":"abstain"},"heard_words":%q,"name_associations":[{"name":"Jonas","owner":{"actor_id":%q,"description":%q,"about_actor_ids":[%q]}}]},{"listener_id":%q,"attention":{"kind":"abstain"},"heard_words":%q,"name_associations":[]}]}`,
				id.P, spoken, id.J, description, id.M, id.J, spoken)}
			orc := &Orchestrator{DB: pool, Resolve: resolve}
			tick := flowBaseTick(t, ctx, pool, id.World)
			var err error
			if door == "ordinary" {
				_, err = orc.applyEvent(ctx, id.World, id.M, []byte(fmt.Sprintf(
					`{"type":"Communicated","stated":"Mara introduces Jonas.","listener_id":%q,"content":%q}`, id.P, spoken)), tick, 0)
			} else {
				_, err = orc.applyRuledEvent(ctx, id.World, RuledEventV2{
					Type: "Communicated", ActorID: id.M, ListenerID: id.P, Truth: "Mara introduces Jonas.", Content: spoken,
				}, tick, 0)
			}
			if err != nil {
				t.Fatal(err)
			}
			var name, heard string
			if err := pool.QueryRow(ctx, `SELECT fn_actor_page($1,$2,$3)->'actor'->>'perceived_name'`,
				id.World, id.P, id.J).Scan(&name); err != nil {
				t.Fatalf("recognized actor did not become publicly readable: %v", err)
			}
			if err := pool.QueryRow(ctx, `SELECT spoken FROM fn_perceived_speech($1,$2,$3)`,
				id.World, id.P, tick).Scan(&heard); err != nil {
				t.Fatal(err)
			}
			if name != "Jonas" || heard != spoken {
				t.Fatalf("name=%q, heard=%q; want recognized Jonas and %q", name, heard, spoken)
			}
			var knowledge string
			if err := pool.QueryRow(ctx, `SELECT (fn_actor_page($1,$2,$3)->'actor'->'collected_knowledge_groups')::text`,
				id.World, id.P, id.J).Scan(&knowledge); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(knowledge, description) {
				t.Fatalf("recognized actor's knowledge lost the understood description: %s", knowledge)
			}
		})
	}
}
