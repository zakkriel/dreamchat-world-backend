-- migrate:up

-- The world stopped telling the player what time it was.
--
-- Every committed event ships with in_world_label NULL: apply_event's INSERT (and every sibling
-- commit path — apply_ruled_event, the ObjectRelocated/EntityCreated/portal/move-target variants)
-- names event_type, summary, in_world_tick, beat_seq and origin, and simply never sets the label.
-- Only the SEED authors one ("Backstory", "Arrival", "Day 1"). So the first committed beat ends the
-- only in-world time the player ever had, and three surfaces go blank at once:
--
--   * scene/current's now.display_label — the player's only time anchor, null from beat one;
--   * the Compendium's collected_knowledge_groups group_label — groups render unlabelled;
--   * (already repaired at the read side) synthesis prose, which was rendering "[Tick 51]" —
--     manufacturing a display label out of the logical tick, exactly what B-5 forbids.
--
-- B-5 / ADR-030: in-world time is a logical tick PLUS a display label, append-only. The tick is
-- operational and always advances; the label is AUTHORED and changes only when something authors it.
-- So the honest rule is CONTINUITY, not invention: an event with no label of its own inherits the
-- last label the world actually authored. "Day 1" stands until something says "Day 2". Nothing here
-- invents a label, and nothing here can produce one the world never wrote.
--
-- WHY A TRIGGER, and not the commit functions. There are ~18 INSERT INTO canon_event sites spread
-- across a dozen migrations, each inside a large function that later migrations re-copy verbatim to
-- extend. Patching them means duplicating those bodies again — and the SPEC-031 tuning has just shown
-- what that costs: copying a stale body silently reverted three later extensions. One BEFORE INSERT
-- trigger covers every site that exists and every site added later, and cannot drift out of sync
-- with any of them.
--
-- It composes with the existing append-only guard, which is BEFORE UPDATE (canon_event_append_only):
-- this fills the value on the way in, and the row is immutable from then on. Canon is still
-- append-only; it is simply no longer written with a hole in it.
--
-- REPLAY-SAFE. The carried value is a pure function of already-committed state read in commit order,
-- so a replay in the same order reproduces byte-identical labels. Determinism (I-1) holds.
--
-- An explicitly supplied label always wins — the trigger only fills NULL, so the seed and any future
-- authoring path keep full control.

CREATE OR REPLACE FUNCTION canon_event_carry_in_world_label() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.in_world_label IS NULL THEN
    SELECT ce.in_world_label INTO NEW.in_world_label
      FROM canon_event ce
     WHERE ce.world_id = NEW.world_id
       AND ce.in_world_label IS NOT NULL
       AND (ce.in_world_tick, ce.beat_seq) <= (NEW.in_world_tick, NEW.beat_seq)
     ORDER BY ce.in_world_tick DESC, ce.beat_seq DESC
     LIMIT 1;
  END IF;
  RETURN NEW;
END;
$$;

-- BEFORE INSERT only. The append-only guard owns UPDATE and must keep owning it.
CREATE TRIGGER trg_canon_event_carry_in_world_label
  BEFORE INSERT ON canon_event
  FOR EACH ROW EXECUTE FUNCTION canon_event_carry_in_world_label();

-- migrate:down

DROP TRIGGER IF EXISTS trg_canon_event_carry_in_world_label ON canon_event;
DROP FUNCTION IF EXISTS canon_event_carry_in_world_label();
