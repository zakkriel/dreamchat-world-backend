-- migrate:up
-- Phase 0B (chunk-2): implements the insert-time half of invariant I-4 (doc 07) per the
-- contract in doc 03 §1.4 ("Acyclicity is checked at bundle insert; bounded ancestor walk;
-- reject on cycle"), plus bundle topology immutability (ADR-006). The deferred nightly full
-- check is tracked as SPEC-005. NOT part of frozen doc 03 §1 migrations (kept out of 0002-0006),
-- same precedent as SPEC-002's …0007. No automated runtime path writes bundles before Phase 4
-- (ADR-008/029) — this only constrains the manual/Phase-4 write path.

-- (a) Insert-time acyclicity. A new input edge asserts input -> effect (input causes effect).
-- It closes a cycle iff the effect is already a causal ancestor of the input. Walk ancestors of
-- the new input; reject if the effect is reachable. Walk ALL edges regardless of status: bundle
-- status transitions are unspecified in the frozen contract, so an invalidated edge must still
-- block a cycle that could resurrect on re-validation (spec Rider B). Depth-capped (64) fail-safe.
CREATE FUNCTION causal_bundle_assert_acyclic() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  v_effect_ref  uuid;
  v_effect_kind text;
  v_cycle       boolean;
  v_maxdepth    int;
BEGIN
  SELECT effect_ref, effect_kind INTO v_effect_ref, v_effect_kind
  FROM causal_bundle WHERE bundle_id = NEW.bundle_id;

  WITH RECURSIVE anc(ref, kind, depth) AS (
    SELECT NEW.input_ref, NEW.input_kind, 0
    UNION ALL
    SELECT cbi.input_ref, cbi.input_kind, anc.depth + 1
    FROM anc
    JOIN causal_bundle cb
      ON cb.effect_ref = anc.ref AND cb.effect_kind = anc.kind
    JOIN causal_bundle_input cbi
      ON cbi.bundle_id = cb.bundle_id
    WHERE anc.depth < 64
  )
  SELECT bool_or(ref = v_effect_ref AND kind = v_effect_kind), max(depth)
  INTO v_cycle, v_maxdepth
  FROM anc;

  IF v_cycle THEN
    RAISE EXCEPTION
      'causal cycle rejected (I-4): effect %/% is already a causal ancestor of input %/% (bundle %)',
      v_effect_kind, v_effect_ref, NEW.input_kind, NEW.input_ref, NEW.bundle_id;
  END IF;

  IF v_maxdepth >= 64 THEN
    RAISE EXCEPTION
      'causal acyclicity depth cap (64) exceeded walking ancestors of input %/% — investigate (I-4)',
      NEW.input_kind, NEW.input_ref;
  END IF;

  RETURN NEW;
END $$;
CREATE TRIGGER trg_cbi_assert_acyclic
  BEFORE INSERT ON causal_bundle_input FOR EACH ROW
  EXECUTE FUNCTION causal_bundle_assert_acyclic();

-- (b) Topology immutability (ADR-006: invalidation never deletion; the insert-time walk is only
-- sound if edges cannot mutate after insert).
-- causal_bundle: append-only, only {status} may change (effect_ref/effect_kind/etc. frozen).
CREATE FUNCTION causal_bundle_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF ROW(NEW.bundle_id, NEW.world_id, NEW.effect_ref, NEW.effect_kind,
         NEW.semantics, NEW.template_id, NEW.created_at)
     IS DISTINCT FROM
     ROW(OLD.bundle_id, OLD.world_id, OLD.effect_ref, OLD.effect_kind,
         OLD.semantics, OLD.template_id, OLD.created_at)
  THEN
    RAISE EXCEPTION 'causal_bundle is append-only: only {status} may change (bundle %)', OLD.bundle_id;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_causal_bundle_append_only
  BEFORE UPDATE ON causal_bundle FOR EACH ROW EXECUTE FUNCTION causal_bundle_append_only();

-- causal_bundle_input: fully immutable (no UPDATE).
CREATE FUNCTION causal_bundle_input_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'causal_bundle_input is immutable: UPDATE forbidden (bundle %, input %)',
    OLD.bundle_id, OLD.input_ref;
END $$;
CREATE TRIGGER trg_causal_bundle_input_immutable
  BEFORE UPDATE ON causal_bundle_input FOR EACH ROW EXECUTE FUNCTION causal_bundle_input_immutable();

-- DELETE guards on both bundle tables (reuse forbid_delete() from migration 0002; ADR-006).
CREATE TRIGGER trg_causal_bundle_no_delete
  BEFORE DELETE ON causal_bundle FOR EACH ROW EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER trg_causal_bundle_input_no_delete
  BEFORE DELETE ON causal_bundle_input FOR EACH ROW EXECUTE FUNCTION forbid_delete();

-- migrate:down
DROP TRIGGER IF EXISTS trg_causal_bundle_input_no_delete ON causal_bundle_input;
DROP TRIGGER IF EXISTS trg_causal_bundle_no_delete ON causal_bundle;
DROP TRIGGER IF EXISTS trg_causal_bundle_input_immutable ON causal_bundle_input;
DROP TRIGGER IF EXISTS trg_causal_bundle_append_only ON causal_bundle;
DROP TRIGGER IF EXISTS trg_cbi_assert_acyclic ON causal_bundle_input;
DROP FUNCTION IF EXISTS causal_bundle_input_immutable();
DROP FUNCTION IF EXISTS causal_bundle_append_only();
DROP FUNCTION IF EXISTS causal_bundle_assert_acyclic();
