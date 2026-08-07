package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Journey is one row of the `journey` table (loop state, not canon — the held_outcome precedent,
// migration 20260807100003_journey.sql): a span split into legs, whatever kind sustains it. Read
// fresh from the table on every input (pendingHeldOutcomes's own discipline); never held in server
// memory across calls. FrameID/StageID/GoalTarget are "" when the row carries no value for them
// (wait/watch never populate the travel-only frame/coord/goal columns); OriginCoord/GoalCoord are
// nil for the same reason.
type Journey struct {
	ID, WorldID, ActorID, Kind string
	Threshold                  json.RawMessage
	SpanSeconds                int64
	LegsTotal, LegsDone        int
	StartedTick, CurrentTick   int64
	FrameID, StageID           string // "" when absent
	OriginCoord, GoalCoord     json.RawMessage
	GoalTarget                 string
	Status                     string
}

// journeyTickThreshold is a `wait` journey's threshold shape: the absolute world tick the wait
// clears. Distinct from Sustain's own "for" shape (which carries a relative Seconds, not an
// absolute tick) — startJourney converts one into the other exactly once, at the journey's start.
type journeyTickThreshold struct {
	Kind string `json:"kind"`
	At   int64  `json:"at"`
}

const journeySelectCols = `journey_id, world_id, actor_id, kind, threshold, span_seconds, legs_total, legs_done,
	       started_tick, current_tick, COALESCE(frame_id::text,''), origin_coord, goal_coord,
	       COALESCE(goal_target::text,''), COALESCE(stage_id::text,''), status`

// scanJourney reads one journey row from row into a *Journey — the shared Scan target list for
// activeJourney's lookup, so the column order lives in exactly one place.
func scanJourney(row pgx.Row) (*Journey, error) {
	var j Journey
	var threshold, origin, goal []byte
	if err := row.Scan(&j.ID, &j.WorldID, &j.ActorID, &j.Kind, &threshold, &j.SpanSeconds,
		&j.LegsTotal, &j.LegsDone, &j.StartedTick, &j.CurrentTick, &j.FrameID, &origin, &goal,
		&j.GoalTarget, &j.StageID, &j.Status); err != nil {
		return nil, err
	}
	j.Threshold = json.RawMessage(threshold)
	if len(origin) > 0 {
		j.OriginCoord = json.RawMessage(origin)
	}
	if len(goal) > 0 {
		j.GoalCoord = json.RawMessage(goal)
	}
	return &j, nil
}

// activeJourney reads worldID/actorID's active journey fresh from the table — no server memory, no
// session object (design §4.1). Returns (nil, nil) when there is none: absence is the ordinary case
// on every input that is not mid-journey, never an error.
func (o *Orchestrator) activeJourney(ctx context.Context, worldID, actorID string) (*Journey, error) {
	row := o.DB.QueryRow(ctx, `SELECT `+journeySelectCols+`
		FROM journey WHERE world_id=$1::uuid AND actor_id=$2::uuid AND status='active' LIMIT 1`,
		worldID, actorID)
	j, err := scanJourney(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}

// targetCoord returns target's own coordinate (fn_target_position's `coord` OUT param, in its
// containing scene's local frame) as raw jsonb — the shared read startJourney uses for both the
// travel origin (the actor itself) and the goal (the move's target).
func (o *Orchestrator) targetCoord(ctx context.Context, worldID, target string) (json.RawMessage, error) {
	var coord []byte
	if err := o.DB.QueryRow(ctx,
		`SELECT coord FROM fn_target_position($1::uuid, $2::uuid)`,
		worldID, target).Scan(&coord); err != nil {
		return nil, err
	}
	return json.RawMessage(coord), nil
}

// startJourney turns an over-budget attempt into a journey row: an ActorMoved becomes `travel`
// (span = the real physics duration, fn_move_duration_actor), a `sustain.for` becomes `wait` (span
// = the stated seconds, threshold = the absolute tick it clears — R13, the span is never
// reclassified), and a `sustain.until_*` becomes `watch` (span = this world's horizon default,
// threshold = the predicate carried through verbatim). legs_total comes from fn_journey_legs
// (Task 2), already clamped to 5..10.
//
// It writes the row and COMMITS NOTHING TO CANON: starting a journey is not an event, only arrival
// is (design §4.4/§4.7) — this function never calls applyEvent/adjudicate.
func (o *Orchestrator) startJourney(ctx context.Context, worldID, actorID string, a Attempt, now int64) (*Journey, error) {
	j := &Journey{
		WorldID:     worldID,
		ActorID:     actorID,
		StartedTick: now,
		CurrentTick: now,
		LegsDone:    0,
		Status:      "active",
	}

	var frameID, goalTarget string
	var originCoord, goalCoord json.RawMessage

	switch {
	case a.Type == "ActorMoved":
		dur, err := o.fnMoveDurationActor(ctx, worldID, actorID, a.ToTargetID)
		if err != nil {
			return nil, fmt.Errorf("startJourney: move duration: %w", err)
		}
		j.Kind = "travel"
		j.SpanSeconds = dur
		j.Threshold = json.RawMessage(`{"kind":"span"}`) // unused by thresholdMet (travel tests CurrentTick-StartedTick directly); NOT NULL column needs a shape
		goalTarget = a.ToTargetID

		here, err := o.actorLocation(ctx, worldID, actorID)
		if err != nil {
			return nil, fmt.Errorf("startJourney: actor location: %w", err)
		}
		if err := o.DB.QueryRow(ctx,
			`SELECT COALESCE(attrs->>'parent_location_id','') FROM location_state WHERE world_id=$1::uuid AND entity_id=$2::uuid`,
			worldID, here).Scan(&frameID); err != nil {
			return nil, fmt.Errorf("startJourney: frame: %w", err)
		}

		if originCoord, err = o.targetCoord(ctx, worldID, actorID); err != nil {
			return nil, fmt.Errorf("startJourney: origin coord: %w", err)
		}
		if goalCoord, err = o.targetCoord(ctx, worldID, a.ToTargetID); err != nil {
			return nil, fmt.Errorf("startJourney: goal coord: %w", err)
		}

	case a.Sustain != nil && a.Sustain.Kind == "for":
		j.Kind = "wait"
		j.SpanSeconds = a.Sustain.Seconds
		th, err := json.Marshal(journeyTickThreshold{Kind: "tick", At: now + a.Sustain.Seconds})
		if err != nil {
			return nil, fmt.Errorf("startJourney: wait threshold: %w", err)
		}
		j.Threshold = th

	case a.Sustain != nil && (a.Sustain.Kind == "until_at" || a.Sustain.Kind == "until_attr"):
		// The Sustain shape carries no span of its own for a watch — R13's "stated span" only ever
		// applies to `for`. The horizon is this world's per-world default: reusing
		// fn_duration_class_seconds's own "extremely_long" tier (retunable, built-in-fallback —
		// same shape as every other per-world default this program touches) rather than inventing a
		// second per-world config table for a single number Task 2 did not ask for.
		horizon, err := o.durationClassSeconds(ctx, worldID, "extremely_long")
		if err != nil {
			return nil, fmt.Errorf("startJourney: watch horizon: %w", err)
		}
		j.Kind = "watch"
		j.SpanSeconds = horizon
		th, err := json.Marshal(a.Sustain)
		if err != nil {
			return nil, fmt.Errorf("startJourney: watch threshold: %w", err)
		}
		j.Threshold = th

	default:
		return nil, fmt.Errorf("startJourney: attempt is neither an over-budget ActorMoved nor a sustain: %+v", a)
	}

	if err := o.DB.QueryRow(ctx,
		`SELECT fn_journey_legs($1::uuid, $2::bigint)`, worldID, j.SpanSeconds).Scan(&j.LegsTotal); err != nil {
		return nil, fmt.Errorf("startJourney: leg count: %w", err)
	}

	if err := o.DB.QueryRow(ctx, `
		INSERT INTO journey (world_id, actor_id, kind, threshold, span_seconds, legs_total, legs_done,
		                      started_tick, current_tick, frame_id, origin_coord, goal_coord, goal_target, status)
		VALUES ($1::uuid, $2::uuid, $3, $4::jsonb, $5, $6, 0, $7, $7,
		        NULLIF($8,'')::uuid, $9::jsonb, $10::jsonb, NULLIF($11,'')::uuid, 'active')
		RETURNING journey_id`,
		worldID, actorID, j.Kind, string(j.Threshold), j.SpanSeconds, j.LegsTotal, now,
		frameID, nullableJSONText(originCoord), nullableJSONText(goalCoord), goalTarget,
	).Scan(&j.ID); err != nil {
		return nil, err
	}

	j.FrameID = frameID
	j.OriginCoord = originCoord
	j.GoalCoord = goalCoord
	j.GoalTarget = goalTarget
	return j, nil
}

// nullableJSONText renders raw as a *string parameter for a nullable jsonb column: nil (absent, the
// wait/watch case) becomes a real NULL rather than the literal 4-byte JSON "null".
func nullableJSONText(raw json.RawMessage) *string {
	if raw == nil {
		return nil
	}
	s := string(raw)
	return &s
}

// legSliceSeconds is what one leg covers: the SPAN LEFT divided by the LEGS REMAINING, so rounding
// never strands progress. When one leg remains it returns the whole remainder rather than a
// ceil-divided fraction of it — the two agree when the division is exact, but only the "whole
// remainder" rule guarantees the LAST leg closes the span exactly regardless of how span_seconds
// and legs_total divide.
func legSliceSeconds(j *Journey) int64 {
	remaining := j.SpanSeconds - (j.CurrentTick - j.StartedTick)
	legsRemaining := int64(j.LegsTotal - j.LegsDone)
	if legsRemaining <= 1 {
		return remaining
	}
	return (remaining + legsRemaining - 1) / legsRemaining // ceil(remaining / legsRemaining)
}

// thresholdMet switches on the journey's kind — travel resolves on distance covered (measured in
// ticks: fn_move_duration_actor already converts distance to ticks at journey start, so "covered"
// is just elapsed ticks against the span), wait on the clock reaching an absolute tick, watch on a
// deterministic state predicate evaluated in SQL. No model ever runs here (design §4.4).
func (o *Orchestrator) thresholdMet(ctx context.Context, j *Journey) (bool, error) {
	switch j.Kind {
	case "travel":
		return j.CurrentTick-j.StartedTick >= j.SpanSeconds, nil

	case "wait":
		var th journeyTickThreshold
		if err := json.Unmarshal(j.Threshold, &th); err != nil {
			return false, fmt.Errorf("thresholdMet: wait threshold: %w", err)
		}
		return j.CurrentTick >= th.At, nil

	case "watch":
		var s Sustain
		if err := json.Unmarshal(j.Threshold, &s); err != nil {
			return false, fmt.Errorf("thresholdMet: watch threshold: %w", err)
		}
		switch s.Kind {
		case "until_at":
			// The entity's containing scene (fn_target_position, the same resolver every other
			// target-scene read in this program uses) equals the watched place — one query.
			var met bool
			if err := o.DB.QueryRow(ctx,
				`SELECT scene = $3::uuid FROM fn_target_position($1::uuid, $2::uuid)`,
				j.WorldID, s.EntityID, s.PlaceID).Scan(&met); err != nil {
				return false, fmt.Errorf("thresholdMet: until_at: %w", err)
			}
			return met, nil

		case "until_attr":
			// attrs->>attr = value, read off whichever projection table the entity's kind owns —
			// one query, the same per-kind CASE shape fn_target_position itself uses.
			var met bool
			if err := o.DB.QueryRow(ctx, `
				SELECT COALESCE(
				  (CASE er.entity_kind
				     WHEN 'actor'    THEN ast.attrs->>$3
				     WHEN 'artifact' THEN art.attrs->>$3
				     WHEN 'location' THEN ls.attrs->>$3
				   END) = $4,
				  false)
				FROM entity_registry er
				LEFT JOIN actor_state    ast ON ast.world_id = er.world_id AND ast.entity_id = er.entity_id
				LEFT JOIN artifact_state art ON art.world_id = er.world_id AND art.entity_id = er.entity_id
				LEFT JOIN location_state  ls ON  ls.world_id = er.world_id AND  ls.entity_id = er.entity_id
				WHERE er.world_id = $1::uuid AND er.entity_id = $2::uuid`,
				j.WorldID, s.EntityID, s.Attr, s.Value).Scan(&met); err != nil {
				return false, fmt.Errorf("thresholdMet: until_attr: %w", err)
			}
			return met, nil

		default:
			return false, fmt.Errorf("thresholdMet: unknown watch predicate kind %q", s.Kind)
		}

	default:
		return false, fmt.Errorf("thresholdMet: unknown journey kind %q", j.Kind)
	}
}

// endJourney sets the status and nothing else — the row STAYS (never deleted here), so
// fn_world_now keeps reading its current_tick even after the journey stops (Task 1 assertion (d):
// time must never rewind when a journey ends, B-5).
func (o *Orchestrator) endJourney(ctx context.Context, j *Journey, status string) error {
	if _, err := o.DB.Exec(ctx, `UPDATE journey SET status=$1 WHERE journey_id=$2::uuid`, status, j.ID); err != nil {
		return err
	}
	j.Status = status
	return nil
}
