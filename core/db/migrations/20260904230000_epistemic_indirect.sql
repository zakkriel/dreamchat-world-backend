-- migrate:up

-- `indirect` — knowledge perceived THROUGH A MEDIUM: a recording, a spell, a dream, a scrying.
-- Governed-by: ADR-037 — an unwitnessed event is canon, and `indirect` is how it becomes knowable.
--
-- WHY IT IS ITS OWN WORD. The closed set had no way to say "I perceived the thing itself, but not
-- with my own eyes at the time". The nearest words are both wrong: `told` puts a person's MIND
-- between you and the event, and `inference` puts your own REASONING there. A garage camera's tape
-- is neither — it is the event, displaced in time. Founder, 2026-09-04: "like direct but perceived
-- through another medium (a recording, a spell, a dream, etc)".
--
-- WHY THE ENGINE NEEDED IT NOW. A canon event nobody witnessed is legitimate (ADR-005 always said
-- "zero-to-N perceptions"; the genesis belt was refusing it anyway, and that refusal is removed in
-- the same round). But an unwitnessed event with no way to ever learn of it is a dead row. The
-- reveal path is: the medium HOLDS ITS OWN PERCEPTION of the event -- `perception_record.holder_id`
-- carries no FK and no kind check, so an artifact may hold one, the same way the "Common Knowledge"
-- faction pseudo-entity already does -- and whoever reaches that medium acquires theirs `indirect`,
-- derived from the medium's. Nothing but a perception ever links to canon
-- (`perception_record_source_event_id_fkey` is the only door, and it is NOT NULL), so the tape does
-- not point at the event: it holds a perception that does.
--
-- RELIABILITY IS NOT THIS COLUMN'S JOB. `perception_record` already carries `confidence` and
-- `distortion_level`. A dream is `indirect` with low confidence; a tape is `indirect` with high. The
-- type says how it arrived, the numbers say how much it is trusted. Adding a second word per medium
-- would put reliability in the vocabulary, where it does not belong.
--
-- STRICTLY ADDITIVE: no existing value changes meaning, no row is rewritten, and every perception
-- written before this migration keeps the type it had. D-5 routes the addition through an ADR rather
-- than an ad-hoc edit because `perception_record` lives in the frozen Master DDL (doc 03 §1.3).

ALTER TABLE perception_record DROP CONSTRAINT perception_record_epistemic_type_check;

ALTER TABLE perception_record ADD CONSTRAINT perception_record_epistemic_type_check
  CHECK (epistemic_type = ANY (ARRAY[
    'direct',      -- you were there
    'indirect',    -- a medium was: a recording, a spell, a dream (ADR-037)
    'shared',
    'told',        -- a person's mind was
    'overheard',
    'public',
    'rumor',
    'inference',   -- your own reasoning was
    'mistaken',
    'confirmed',
    'disputed'
  ]));

-- migrate:down

-- Reversible only while no row uses the new value; a live `indirect` perception would fail the
-- narrowed CHECK, which is the correct refusal rather than a silent data loss.
ALTER TABLE perception_record DROP CONSTRAINT perception_record_epistemic_type_check;

ALTER TABLE perception_record ADD CONSTRAINT perception_record_epistemic_type_check
  CHECK (epistemic_type = ANY (ARRAY[
    'direct', 'shared', 'told', 'overheard', 'public',
    'rumor', 'inference', 'mistaken', 'confirmed', 'disputed'
  ]));
