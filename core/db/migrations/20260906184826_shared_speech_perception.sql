-- migrate:up
--
-- Shared speech perception — one application path for Communicated, through both commit doors.
--
-- Governed-by: ADR-038 (engine, docs/law/02_world_state_adrs.md) — this migration is its evidence,
-- the same relationship ADR-037 has to migration 20260904230000. `perception_record` lives in the
-- frozen Master DDL (D-5); an addition to it is ADR-gated, not an ad-hoc migration.
--
-- Founder-approved behavior and the shared implementation contracts this migration executes were
-- fixed during planning; ADR-P025 places attention judgment with resolve and physical mechanics
-- with code; B-1/B-2 (perception-bound, valid-path knowledge), B-7 (knowledge transfer never copies
-- memory — a listener's record is her own, sourced, not a copy of the speaker's), D-1 (nothing
-- mutates canon directly; this stays a projection of an already-accepted event).
--
-- ── THE DISCREPANCY THIS CLOSES ─────────────────────────────────────────────────────────────────
-- Reproduced before this migration: an ordinary Communicated event taught a spoken name via
-- `generate_perceptions`' regex scan of `payload.spoken` (SPEC-033, migrations 20260809090007 and
-- 20260814170000). A RULED Communicated event never went through `generate_perceptions` at all —
-- `apply_ruled_event` broadcast a flat 'direct' perception of the truth/appearance/receiver-variant
-- text to every co-present actor, identically to every other ruled event type, with zero
-- speaker/listener distinction and zero name-teaching. Same words, same event type, two different
-- outcomes depending on which door committed it. This migration gives both doors ONE application
-- function for Communicated: `fn_apply_speech_perception`.
--
-- ── WHAT DECIDES WHO LEARNS WHAT, NOW ───────────────────────────────────────────────────────────
-- The regex scanner (`fn_names_in_text`) is removed as a learning source entirely — it always risked
-- teaching a name from a coincidentally-matching common noun or an unaddressed account (both bugs
-- this same scanner had to be re-patched for twice already: 20260814170000, then the token-guard in
-- 20260821120000). In its place: the resolve seat judges, per candidate listener and per named
-- association, a flat owner — actor_id (a person she can point to, or null), description (this
-- listener's own understood characterization, or null) and about_actor_ids (already-grounded related
-- actors the description concerns). There is no kind discriminator and no mutually exclusive cases:
-- a non-null actor_id always writes `name_knowledge`; a non-null description is always folded into
-- the stored content, even when the SAME association also recognizes an actor_id; about_actor_ids
-- always anchors the listener's new sourced perception to the actors the description CONCERNS — e.g.
-- "my brother Jonas" anchors to the speaker, never to the undescribed brother — so that knowledge is
-- actually discoverable later through the ordinary subject-keyed read paths instead of being
-- readable only by re-parsing canon_event.payload (B-7: "forbids the copy-the-record shortcut").
-- Structural validity (at least one of actor_id/description present; about_actor_ids requires a
-- description) is Go's job, checked before the judgment ever reaches this migration's functions.
--
-- ── THE NEW COLUMN ──────────────────────────────────────────────────────────────────────────────
-- `perception_record.spoken` holds what THIS HOLDER actually perceived as spoken — which can differ
-- from `canon_event.payload.spoken` (the truth-side words) for a ruled receiver whose heard_words the
-- model judged as muffled, partial, or otherwise not the source's literal words. It is NOT proof the
-- model judged accurately — it is the durable record of what was applied. NULL for a perception with
-- no heard words (blocked listeners get no row at all; an attended listener with empty heard_words
-- gets NULL here, not an empty string).
--
-- ── THE SHARED FUNCTION, AND THE ACCOUNT/QUOTE SPLIT ────────────────────────────────────────────
-- `fn_apply_speech_perception(event_id, account, listener_accounts)` is the single writer for both
-- doors. `account` is the BARE account text ONLY — `generate_perceptions` passes `ev.summary`
-- unmodified; `apply_ruled_event` passes the bare truth/appearance/receiver-variant text it always
-- has. Neither caller pre-bakes anyone's words into it. The function walls that bare account through
-- fn_viewer_text FIRST (identity rewriting, unchanged mechanism), and only THEN appends, per holder,
-- THAT holder's own words as a quote: the speaker's `payload.spoken`, or a listener's own judged
-- `heard_words` — never the other's. This split matters for two reasons a pre-baked-quote design
-- gets wrong: (1) an ambiguous heard name living inside a quote must never be identity-scrubbed by
-- fn_viewer_text, which a name genuinely appearing before walling would be; (2) a ruled listener
-- whose heard_words differ from the canonical spoken text (the entire point of receiver variants)
-- must see HER OWN words, not the speaker's, in her own account.
--
-- Both doors reach the shared function the same way generate_perceptions already was apply_event's
-- single dispatch point: apply_event's unconditional `PERFORM generate_perceptions(ev_id)` now
-- delegates Communicated to the shared function from inside generate_perceptions; apply_ruled_event,
-- which never called generate_perceptions for anything, now calls the shared function directly for
-- Communicated, carving that one event type out of its generic ruled broadcast loop. Every other
-- ruled event type keeps that loop verbatim — this is a speech-only change (D-2/GA-4 scope: no
-- physics, no unrelated event type touched).
--
-- ── STRUCTURAL GATE AT COMMIT, HARD FAILURE INSIDE THE SHARED FUNCTION ──────────────────────────
-- A visible Communicated attempt/ruling with no `speech_perception` object (or one missing a
-- `listeners` array) is `gate_reject`ed at the commit doors, at the SAME point the existing
-- co-location check already sits — before any INSERT, so nothing half-commits
-- (FINAL-action-contracts.md: deterministic machinery blocks, it does not award). That gate is
-- necessarily duplicated, on purpose, one level down: `fn_apply_speech_perception` itself RAISEs on
-- a missing/malformed judgment for any non-hidden event, unconditionally — there is no
-- speaker-only partial success and no path where the shared function silently does less than it
-- was asked. A caller that reaches the shared function without going through either gate (a direct
-- fixture, a future caller) gets the identical refusal, not a quieter one. Deep validation — exact
-- candidate coverage, same-world ids, player/NPC constraints, conflicting fields — stays Go's job
-- before it ever calls apply_event/apply_ruled_event; neither gate duplicates it.
--
-- ── HIDDEN SPEECH IS EXPLICIT, NOT AN UPSTREAM ACCIDENT ─────────────────────────────────────────
-- Hidden ruled speech (`visible:false`) must produce zero listeners, zero heard words, and zero
-- learning, INCLUDING no speaker perception. Both doors persist `payload.visible` on the committed
-- event (ordinary speech never sets the key at all, since it has no hidden concept), and
-- `fn_apply_speech_perception` checks it directly and returns before requiring or reading any
-- judgment — an explicit, self-contained part of the shared function's own contract, true for any
-- caller, not merely true because apply_ruled_event's separate `IF NOT visible THEN RETURN` happens
-- to skip calling it on the normal path.
--
-- ── THE TWO NEW READ FUNCTIONS ──────────────────────────────────────────────────────────────────
-- `fn_perceived_speech` backs beatHandler.speechTexts: the heard words a viewer actually holds, read
-- from stored `perception_record.spoken` instead of assuming every perception attached to a speech
-- event carries verbatim words.
--
-- `fn_unheard_names` is a SEPARATE, more permissive guard from `fn_unearned_names`: the identity wall
-- (fn_viewer_text, rendering the account) is unchanged and still runs first; but a name the viewer
-- LITERALLY heard spoken — stored verbatim in her own `perception_record.spoken` — must not also be
-- censored out of the quoted heard-words belt merely because she never learned WHOSE name it was.
-- Hearing a word and knowing its owner are different facts; this function is the lexical exemption
-- for the former, and it writes nothing — a read-only relaxation of one belt, never a second door
-- into name_knowledge. The literal-heard check is case-SENSITIVE and word-bounded, the same
-- strictness `fn_names_in_text` used for teaching (20260814170000's rationale carries over
-- unchanged: a word is "heard" when the utterance carries it, not when it coincidentally shares
-- letters with something else already in the account).
--
-- ── HISTORICAL BACKFILL ─────────────────────────────────────────────────────────────────────────
-- `fn_backfill_perception_spoken` populates `spoken` for pre-existing perception rows ONLY where the
-- holder's own recorded `content` demonstrably contains the complete recorded utterance — either the
-- content equals it exactly (the historical `say`-step case where summary=content, so no quoting was
-- ever appended) or contains it as an exact `"..."`-quoted substring (the account+quote pattern
-- `generate_perceptions` has assembled since 20260809090009). Anything else — most ruled receiver
-- variants, which render flavor text instead of quoting the words at all — is left NULL: "do not
-- populate it merely because a perception is attached to a speech event." Run once, forward, over
-- whatever the database already holds.

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 1. perception_record.spoken
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

ALTER TABLE public.perception_record ADD COLUMN spoken text;

COMMENT ON COLUMN public.perception_record.spoken IS
  'The words THIS HOLDER perceived as spoken, when the source was Communicated speech she attended '
  '(fn_apply_speech_perception). NULL for a non-speech perception, for an attended listener whose '
  'judged heard_words were empty, and for every perception predating this column that could not be '
  'conservatively backfilled (fn_backfill_perception_spoken). Never populated for a blocked listener '
  '— she holds no perception row from that event at all. Read by fn_perceived_speech and guarded, '
  'more permissively than canonical-name identity, by fn_unheard_names.';

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 2. Historical backfill — conservative, exact-content-or-exact-quote only
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE FUNCTION public.fn_backfill_perception_spoken() RETURNS integer
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  n integer;
BEGIN
  WITH candidates AS (
    SELECT pr.perception_id, TRIM(ce.payload->>'spoken') AS spoken_text
    FROM perception_record pr
    JOIN canon_event ce ON ce.event_id = pr.source_event_id
    WHERE pr.spoken IS NULL
      AND ce.event_type IN ('Communicated', 'private_disclosure')
      AND NULLIF(TRIM(COALESCE(ce.payload->>'spoken', '')), '') IS NOT NULL
      -- Demonstrably present as the COMPLETE utterance: exact content (the legacy say-step case,
      -- summary=content, no quote ever appended) or an exact "..."-quoted substring (the
      -- account+quote pattern generate_perceptions assembles). Nothing weaker — a paraphrase or a
      -- ruled flavor-text rendering that merely mentions the topic is not a recoverable quote.
      AND (
        pr.content = TRIM(ce.payload->>'spoken')
        OR position(('"' || TRIM(ce.payload->>'spoken') || '"') IN pr.content) > 0
      )
  )
  UPDATE perception_record pr
     SET spoken = c.spoken_text
    FROM candidates c
   WHERE pr.perception_id = c.perception_id;
  GET DIAGNOSTICS n = ROW_COUNT;
  RETURN n;
END $$;

COMMENT ON FUNCTION public.fn_backfill_perception_spoken() IS
  'One-time (but idempotent — only ever touches spoken IS NULL rows) historical backfill for '
  'perception_record.spoken. Conservative by design: exact content or exact quoted-utterance match '
  'only. Never run against payload-mention alone. See migration header for the receiver-variant case '
  'this deliberately leaves NULL.';

SELECT public.fn_backfill_perception_spoken();

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 3. Candidate set — the small, cheap function both the facts sheet and any coverage check share
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE FUNCTION public.fn_speech_perception_candidates(
  p_world_id uuid,
  p_speaker  uuid,
  p_intended_listener uuid
) RETURNS TABLE(actor_id uuid)
    LANGUAGE sql STABLE
    AS $$
  -- Co-present actors except the speaker...
  SELECT a.entity_id
  FROM actor_state s
  JOIN fn_actors_at(p_world_id, (s.attrs->>'location_id')::uuid) a ON true
  WHERE s.world_id = p_world_id AND s.entity_id = p_speaker
    AND a.entity_id <> p_speaker
  UNION
  -- ...including the addressed recipient when valid — a safety net for the addressee, independent
  -- of whether fn_actors_at's read of the speaker's OWN location happens to already cover her (it
  -- normally does; "when valid" means "is actually a registered actor in this world", not "is
  -- co-located", which the commit-time gate enforces separately and unconditionally).
  SELECT p_intended_listener
  WHERE p_intended_listener IS NOT NULL
    AND p_intended_listener <> p_speaker
    AND EXISTS (
      SELECT 1 FROM entity_registry er
      WHERE er.world_id = p_world_id AND er.entity_id = p_intended_listener
        AND er.entity_kind = 'actor'
    )
$$;

COMMENT ON FUNCTION public.fn_speech_perception_candidates(uuid, uuid, uuid) IS
  'The listener candidate set for one speech event: co-present actors except the speaker, plus the '
  'addressed recipient when she is a real actor in this world. Deliberately cheap (no fact assembly) '
  'so a commit-side coverage check never has to rebuild fn_speech_perception_facts to verify the '
  'model judged exactly this set.';

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 4. Facts for one speech event — measurements for the model, never a verdict (fn_fact_sheet's
--    own division, reused, not its exact field list). Resolve is the referee (truth-side, unwalled,
--    matching fn_fact_sheet's own p_truth_side=true convention for this seat) — activities and
--    physics expose the recorded facts as they stand, not a cognition-seat-scoped reduction.
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE FUNCTION public.fn_speech_perception_facts(
  p_world_id       uuid,
  p_speaker        uuid,
  p_intended_listener uuid,
  p_recency_window bigint,
  p_recency_limit  integer
) RETURNS jsonb
    LANGUAGE sql STABLE
    AS $$
  WITH speaker_loc AS (
    SELECT (attrs->>'location_id')::uuid AS loc
    FROM actor_state WHERE world_id = p_world_id AND entity_id = p_speaker
  ),
  cand AS (
    SELECT actor_id FROM fn_speech_perception_candidates(p_world_id, p_speaker, p_intended_listener)
  ),
  sheet AS (
    -- Computed ONCE for every candidate together (fn_fact_sheet's own targets array), then matched
    -- per listener below by the target's own `id` key — never a positional targets[0] assumption,
    -- and never one redundant fn_fact_sheet call per candidate re-deriving the same scene position.
    SELECT fn_fact_sheet(p_world_id, p_speaker, ARRAY(SELECT actor_id FROM cand), true) AS fs
  )
  SELECT jsonb_build_object(
    'schema_version', 'speech_perception_facts/1',
    'listeners', COALESCE((
      SELECT jsonb_agg(
        jsonb_build_object(
          'actor_id',  c.actor_id,
          'is_player', c.actor_id = (SELECT player_entity_id FROM world WHERE world_id = p_world_id),
          -- Identity metadata belongs to this listener's view, not the speaker's private knowledge.
          'label', fn_display_name(p_world_id, c.actor_id, c.actor_id),
          -- References are relative to THIS LISTENER, not the speaker: name understanding is
          -- grounded in what each listener knows, not the referee's or speaker's truth. Present
          -- actors (so a clear present gesture can ground a first-meeting recognition) UNION every
          -- ACTOR this listener already has sourced perception knowledge of (so an ABSENT
          -- previously-introduced person can still be recognized by description — the walkthrough's
          -- "already knows Mara's brother personally" case). Filtered to entity_kind='actor': a
          -- perception's subject can be a location or an artifact, and neither is a name reference.
          'references', COALESCE((
            SELECT jsonb_agg(jsonb_build_object(
                     'actor_id', r.actor_id,
                     'label', fn_display_name(p_world_id, c.actor_id, r.actor_id)))
            FROM (
              SELECT entity_id AS actor_id
              FROM fn_actors_at(p_world_id, (SELECT loc FROM speaker_loc))
              UNION
              SELECT ps.entity_id
              FROM fn_visible_perceptions(p_world_id, c.actor_id) vp
              JOIN perception_subject ps ON ps.perception_id = vp.perception_id
              JOIN entity_registry er
                ON er.world_id = p_world_id AND er.entity_id = ps.entity_id AND er.entity_kind = 'actor'
            ) r
          ), '[]'::jsonb),
          -- Sourced perception contents this listener already holds — the SAME bounded recent
          -- window beatHandler.payload() already uses (recencyTickWindow/recencyMaxRows), passed in
          -- rather than re-invented (contract point 1): newest p_recency_limit rows within
          -- p_recency_window ticks of her newest, presented oldest-first.
          'knowledge', COALESCE((
            SELECT jsonb_agg(k.content ORDER BY k.acquired_tick)
            FROM (
              SELECT vp.content, vp.acquired_tick
              FROM fn_visible_perceptions(p_world_id, c.actor_id) vp
              WHERE vp.acquired_tick >= (
                      SELECT max(acquired_tick) FROM fn_visible_perceptions(p_world_id, c.actor_id)
                    ) - p_recency_window
              ORDER BY vp.acquired_tick DESC
              LIMIT p_recency_limit
            ) k
          ), '[]'::jsonb),
          -- Recorded activity, truth-side and unreduced: resolve is the referee reasoning over real
          -- facts (fn_fact_sheet's own truth_side=true), not an NPC-cognition seat the wall must
          -- protect a secret from. `held` carries the actual pending attempt (her full recorded
          -- intention), `journey` carries the full recorded row, `statuses` carries whatever
          -- actor_state.attrs.statuses already records (the same array fn_effective_speed reads) —
          -- no invented mechanic, no curated subset.
          'activities', jsonb_build_object(
            'held', (
              SELECT jsonb_build_object('attempt', ho.attempt, 'status', ho.status, 'created_tick', ho.created_tick)
              FROM held_outcome ho
              WHERE ho.world_id = p_world_id AND ho.actor_id = c.actor_id AND ho.status = 'pending'
              LIMIT 1
            ),
            'journey', (
              SELECT jsonb_build_object(
                       'kind', j.kind, 'status', j.status, 'threshold', j.threshold,
                       'span_seconds', j.span_seconds, 'legs_total', j.legs_total,
                       'legs_done', j.legs_done, 'started_tick', j.started_tick,
                       'current_tick', j.current_tick, 'goal_coord', j.goal_coord,
                       'goal_target', j.goal_target
                     )
              FROM journey j
              WHERE j.world_id = p_world_id AND j.actor_id = c.actor_id AND j.status = 'active'
              LIMIT 1
            ),
            'statuses', COALESCE((
              SELECT a.attrs->'statuses' FROM actor_state a
              WHERE a.world_id = p_world_id AND a.entity_id = c.actor_id
            ), '[]'::jsonb)
          ),
          -- Existing computed physics only (fn_fact_sheet's per-target object, looked up by its own
          -- `id` field from the ONE shared call above) — no invented hearing range, no acoustic
          -- attenuation, no facing rule. Registry names are not physical measurements.
          'physics', (
            SELECT t - 'name' FROM sheet, jsonb_array_elements(sheet.fs->'targets') t
            WHERE t->>'id' = c.actor_id::text
          )
        )
        ORDER BY c.actor_id
      )
      FROM cand c
    ), '[]'::jsonb)
  );
$$;

COMMENT ON FUNCTION public.fn_speech_perception_facts(uuid, uuid, uuid, bigint, integer) IS
  'Measurements for the speech_perception seat, one object per listener candidate '
  '(fn_speech_perception_candidates): label and actor references relative to the listener, '
  'her own recent sourced knowledge, truth-side recorded activity '
  '(full held attempt, full journey row, actor_state.attrs.statuses), and existing physics matched '
  'by target id from one shared fn_fact_sheet call. Supplies no attention judgment and no name '
  'association — that is the model''s job, applied by fn_apply_speech_perception.';

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 5. The shared perception application function — the one thing both commit doors call
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE FUNCTION public.fn_apply_speech_perception(
  p_event_id uuid,
  p_account  text,
  p_listener_accounts jsonb
) RETURNS integer
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev     canon_event;
  sp     jsonb;
  spk    uuid;
  n      integer := 0;
  pid    uuid;
  lst_el jsonb;
  lst_id uuid;
  heard  text;
  acct   text;
  assoc  jsonb;
  spoken_by_speaker text;
  walled text;
BEGIN
  SELECT * INTO ev FROM canon_event WHERE event_id = p_event_id AND status = 'accepted';
  IF NOT FOUND THEN RETURN 0; END IF;
  IF ev.event_type NOT IN ('Communicated', 'private_disclosure') THEN RETURN 0; END IF;

  -- Hidden ruled speech: an explicit, self-contained decision of THIS function, not merely
  -- something a caller happens never to trigger. Both commit doors persist payload.visible on the
  -- committed event (ordinary speech never sets the key at all — it has no hidden concept, and a
  -- missing key here is correctly "not hidden"). Zero listeners, zero heard words, zero learning,
  -- INCLUDING no speaker perception.
  IF (ev.payload->>'visible') = 'false' THEN
    RETURN 0;
  END IF;

  -- No silent fallback, for ANY caller: a visible Communicated event with no accepted judgment (or
  -- a malformed one) is refused outright, never applied as a quieter "speaker learns nothing,
  -- listener gets nothing" partial success. The commit doors already gate_reject this before
  -- committing the event at all; this is the same requirement enforced one level down, so a direct
  -- caller (a fixture, a future code path) that reaches this function without going through either
  -- gate gets the identical refusal.
  sp := ev.payload->'speech_perception';
  IF jsonb_typeof(sp) IS DISTINCT FROM 'object' OR jsonb_typeof(sp->'listeners') IS DISTINCT FROM 'array' THEN
    RAISE EXCEPTION
      'fn_apply_speech_perception: event % is visible Communicated speech with no accepted '
      'speech_perception judgment (or it is malformed) — refusing to half-apply. Every caller, '
      'including direct fixtures, must supply an accepted judgment; there is no scanner fallback '
      'and no speaker-only partial success.', p_event_id;
  END IF;

  -- The speaker's own record is deterministic and unconditional: she authored the utterance, so she
  -- always perceives it. No model judgment is consulted for her own words, and no name-learning
  -- scan runs on them either. The account is walled for identity FIRST, and her own spoken words
  -- (payload.spoken) are appended as a quote AFTER walling — never the other way around, so a name
  -- inside the quote is never touched by the identity wall.
  SELECT entity_id INTO spk FROM event_participant
    WHERE event_id = p_event_id AND role_qualifier = 'speaker' LIMIT 1;

  IF spk IS NOT NULL THEN
    spoken_by_speaker := NULLIF(TRIM(COALESCE(ev.payload->>'spoken', '')), '');
    walled := fn_viewer_text(ev.world_id, spk, p_account);
    IF spoken_by_speaker IS NOT NULL AND position(spoken_by_speaker IN walled) = 0 THEN
      walled := walled || ' — "' || spoken_by_speaker || '"';
    END IF;
    INSERT INTO perception_record (world_id, holder_id, source_event_id, content, spoken,
                                   epistemic_type, acquired_tick, valid_tick)
    VALUES (ev.world_id, spk, p_event_id, walled, spoken_by_speaker,
            'shared', ev.in_world_tick, ev.in_world_tick)
    RETURNING perception_id INTO pid;
    INSERT INTO perception_subject (perception_id, entity_id, world_id)
    SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
    WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
    n := n + 1;
  END IF;

  FOR lst_el IN SELECT value FROM jsonb_array_elements(sp->'listeners') LOOP
    -- Blocked (or an absent/malformed attention field): no perception, no learning. The block
    -- reason already lives in canon_event.payload.speech_perception verbatim (preserved by the
    -- commit doors); nothing further is recorded here.
    IF COALESCE(lst_el->'attention'->>'kind', '') <> 'abstain' THEN
      CONTINUE;
    END IF;

    lst_id := NULLIF(lst_el->>'listener_id', '')::uuid;
    IF lst_id IS NULL THEN CONTINUE; END IF;

    -- Same account/quote split as the speaker, but with THIS listener's own account override
    -- (ruled receiver variants) and THIS listener's own heard_words — never the speaker's canonical
    -- words. A receiver-variant listener who heard something different from what was truly said
    -- must see HER OWN words here, which a pre-baked shared quote could never express.
    heard := NULLIF(TRIM(COALESCE(lst_el->>'heard_words', '')), '');
    acct  := COALESCE(p_listener_accounts ->> (lst_id::text), p_account);
    walled := fn_viewer_text(ev.world_id, lst_id, acct);
    IF heard IS NOT NULL AND position(heard IN walled) = 0 THEN
      walled := walled || ' — "' || heard || '"';
    END IF;

    -- Retain the holder's description even when the same association recognizes an actor,
    -- so future facts reads do not have to re-derive it from the utterance.
    FOR assoc IN SELECT value FROM jsonb_array_elements(COALESCE(lst_el->'name_associations', '[]'::jsonb)) LOOP
      IF NULLIF(TRIM(COALESCE(assoc->'owner'->>'description', '')), '') IS NOT NULL
         AND NULLIF(TRIM(COALESCE(assoc->>'name', '')), '') IS NOT NULL THEN
        walled := walled || ' (' || (assoc->>'name') || ': '
                  || TRIM(assoc->'owner'->>'description') || ')';
      END IF;
    END LOOP;

    INSERT INTO perception_record (world_id, holder_id, source_event_id, content, spoken,
                                   epistemic_type, acquired_tick, valid_tick)
    VALUES (ev.world_id, lst_id, p_event_id, walled, heard, 'told', ev.in_world_tick, ev.in_world_tick)
    RETURNING perception_id INTO pid;
    INSERT INTO perception_subject (perception_id, entity_id, world_id)
    SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
    WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
    n := n + 1;

    -- Apply recognition and related-subject links independently. Go has already checked
    -- reference membership and required a description for related actors.
    FOR assoc IN SELECT value FROM jsonb_array_elements(COALESCE(lst_el->'name_associations', '[]'::jsonb)) LOOP
      IF NULLIF(assoc->'owner'->>'actor_id', '') IS NOT NULL THEN
        INSERT INTO name_knowledge (world_id, holder_id, entity_id, name, learned_tick, source_event_id)
        VALUES (ev.world_id, lst_id, (assoc->'owner'->>'actor_id')::uuid,
                assoc->>'name', ev.in_world_tick, p_event_id)
        ON CONFLICT DO NOTHING;
        INSERT INTO perception_subject (perception_id, entity_id, world_id)
        VALUES (pid, (assoc->'owner'->>'actor_id')::uuid, ev.world_id)
        ON CONFLICT DO NOTHING;
      END IF;

      -- Related actors make this knowledge discoverable through subject-keyed reads.
      -- They are not the name's owner and gain no name_knowledge from this link.
      INSERT INTO perception_subject (perception_id, entity_id, world_id)
      SELECT pid, NULLIF(x.value #>> '{}', '')::uuid, ev.world_id
      FROM jsonb_array_elements(COALESCE(assoc->'owner'->'about_actor_ids', '[]'::jsonb)) x
      WHERE NULLIF(x.value #>> '{}', '') IS NOT NULL
      ON CONFLICT DO NOTHING;
    END LOOP;
  END LOOP;

  RETURN n;
END $$;

COMMENT ON FUNCTION public.fn_apply_speech_perception(uuid, text, jsonb) IS
  'The one Communicated perception writer for both commit doors. Hidden (payload.visible=false):
  zero rows for anyone, no exception. Otherwise requires an accepted payload.speech_perception
  (well-formed object with a listeners array) or RAISEs — no speaker-only fallback for a missing or
  malformed judgment. Speaker: unconditional shared perception, account walled then her own spoken
  words appended as a quote. Each attended (attention.kind=abstain) listener: a told perception,
  account walled then HER OWN heard_words appended, then three independent per-association
  consequences with no kind discriminator: a non-null owner.actor_id always teaches name_knowledge
  and an extra perception_subject link (recognition); a non-null owner.description is always folded
  into the stored content, even when the same association also recognizes an actor_id; a non-empty
  owner.about_actor_ids always adds perception_subject links for the actors the description
  concerns. Related links require a description, checked upstream in Go. A blocked listener gets no
  row at all.';

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 6. generate_perceptions — Communicated now delegates to the shared function, passing the bare
--    account only; the regex scanner is gone as a learning source (move/ObjectRelocated arms
--    carried forward verbatim)
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION public.generate_perceptions(p_event_id uuid) RETURNS integer
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev canon_event;
  n  integer := 0;
BEGIN
  SELECT * INTO ev FROM canon_event WHERE event_id = p_event_id AND status = 'accepted';
  IF NOT FOUND THEN RETURN 0; END IF;

  IF ev.event_type IN ('private_disclosure', 'Communicated') THEN
    -- The bare account only (ev.summary, no quote baked in) — fn_apply_speech_perception walls it
    -- and appends each holder's own words (the speaker's payload.spoken; each listener's own
    -- judged heard_words) AFTER walling, exactly once, so an ambiguous name inside a quote is never
    -- scrubbed by the identity wall and a receiver's own differing heard words are never silently
    -- replaced by the canonical ones. Ordinary speech has one account for everyone (no
    -- receiver-variant concept on this door), so the listener-account map is empty and every
    -- listener falls back to the same bare account.
    n := n + fn_apply_speech_perception(p_event_id, ev.summary, '{}'::jsonb);
  END IF;

  IF ev.event_type IN ('move', 'ActorMoved') THEN
    DECLARE
      mover uuid;
      dest  uuid;
      other uuid;
      pid   uuid;
    BEGIN
      SELECT entity_id INTO mover FROM event_participant
        WHERE event_id = p_event_id AND role_qualifier = 'instigator' LIMIT 1;
      SELECT (new_value #>> '{}')::uuid INTO dest FROM state_mutation
        WHERE event_id = p_event_id AND attribute_path = 'attrs.location_id' LIMIT 1;
      IF mover IS NOT NULL THEN
        INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                       acquired_tick, valid_tick)
        VALUES (ev.world_id, mover, p_event_id, fn_viewer_text(ev.world_id, mover, ev.summary),
                'direct', ev.in_world_tick, ev.in_world_tick)
        RETURNING perception_id INTO pid;
        INSERT INTO perception_subject (perception_id, entity_id, world_id)
        SELECT pid, ep.entity_id, ev.world_id FROM event_participant ep
        WHERE ep.event_id = p_event_id ON CONFLICT DO NOTHING;
        n := n + 1;
        IF dest IS NOT NULL THEN
          FOR other IN SELECT entity_id FROM fn_actors_at(ev.world_id, dest)
                        WHERE entity_id <> mover LOOP
            INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                           acquired_tick, valid_tick)
            VALUES (ev.world_id, mover, p_event_id,
                    'On arriving, I noticed someone already here.', 'direct',
                    ev.in_world_tick, ev.in_world_tick)
            RETURNING perception_id INTO pid;
            INSERT INTO perception_subject (perception_id, entity_id, world_id)
            VALUES (pid, other, ev.world_id);
            n := n + 1;
          END LOOP;
        END IF;
      END IF;
    END;
  END IF;

  -- ── SPEC-034/035: ObjectRelocated (unchanged; carried forward verbatim) ─────────────────────────
  IF ev.event_type = 'ObjectRelocated' THEN
    DECLARE
      or_obj  uuid;
      or_dest uuid;
      or_kind text;
      or_who  uuid;
      or_pid  uuid;
    BEGIN
      SELECT sm.entity_id, (sm.new_value #>> '{}')::uuid
        INTO or_obj, or_dest
        FROM state_mutation sm
       WHERE sm.event_id = p_event_id
         AND sm.attribute_path = 'attrs.contained_by'
       LIMIT 1;

      IF or_obj IS NOT NULL THEN
        SELECT entity_kind INTO or_kind FROM entity_registry
          WHERE entity_id = or_dest AND world_id = ev.world_id;

        FOR or_who IN
          SELECT DISTINCT h FROM (
            SELECT entity_id AS h FROM event_participant
              WHERE event_id = p_event_id AND role_qualifier IN ('instigator', 'witness')
            UNION
            SELECT or_dest WHERE or_dest IS NOT NULL AND or_kind = 'actor'
          ) s WHERE h IS NOT NULL
        LOOP
          INSERT INTO perception_record (world_id, holder_id, source_event_id, content,
                                         epistemic_type, acquired_tick, valid_tick)
          VALUES (ev.world_id, or_who, p_event_id,
                  fn_viewer_text(ev.world_id, or_who, ev.summary), 'direct',
                  ev.in_world_tick, ev.in_world_tick)
          RETURNING perception_id INTO or_pid;

          INSERT INTO perception_subject (perception_id, entity_id, world_id)
          SELECT or_pid, e, ev.world_id FROM (
            SELECT or_obj AS e
            UNION SELECT or_dest WHERE or_dest IS NOT NULL
            UNION SELECT entity_id FROM event_participant WHERE event_id = p_event_id
          ) s WHERE e IS NOT NULL
          ON CONFLICT DO NOTHING;

          n := n + 1;
        END LOOP;
      END IF;
    END;
  END IF;

  RETURN n;
END $$;

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 7. apply_event — require an already-judged speech_perception for visible Communicated speech;
--    preserve it on the committed event alongside payload.spoken
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION public.apply_event(p_world_id uuid, p_actor_id uuid, p_attempt jsonb, p_tick bigint, p_seq integer, p_origin text, p_legacy_types boolean DEFAULT false) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type      text;
  ev_id        uuid;
  listener     uuid;
  to_target    uuid;
  v_scene      uuid;
  v_coord      jsonb;
  object_eid   uuid;
  dest_eid     uuid;
  target_eid   uuid;
  here         uuid;
  vis_scope    text;
  final_type   text;
  v_old_holder uuid;
  v_witness    uuid;
BEGIN
  ev_type := p_attempt->>'type';

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = p_actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_attempt->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, p_actor_id, to_target) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_attempt->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
    IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

    -- Ordinary speech has no hidden/visible distinction (that is a ruled-only concept) — every
    -- ordinary Communicated attempt requires an already-judged speech_perception, structurally: a
    -- jsonb object with a listeners array. Deep validation (candidate coverage, same-world ids,
    -- player/NPC constraints) is Go's job before this call; this is the same class of presence
    -- check as the witnesses gate below/20260825140000 — block malformed or absent input loudly,
    -- never guess or fall back to the old scanner.
    IF jsonb_typeof(p_attempt->'speech_perception') IS DISTINCT FROM 'object'
       OR jsonb_typeof(p_attempt->'speech_perception'->'listeners') IS DISTINCT FROM 'array' THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'ObjectRelocated' THEN
    object_eid := (p_attempt->>'object_id')::uuid;
    dest_eid   := (p_attempt->>'dest_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = object_eid AND world_id = p_world_id)
    OR NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = dest_eid  AND world_id = p_world_id)
    THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

    IF p_attempt ? 'witnesses'
       AND jsonb_typeof(p_attempt->'witnesses') NOT IN ('array', 'null') THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

    IF jsonb_typeof(p_attempt->'witnesses') = 'array' THEN
      SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      FOR v_witness IN SELECT (value #>> '{}')::uuid FROM jsonb_array_elements(p_attempt->'witnesses') LOOP
        IF NOT EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = v_witness) THEN
          RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
        END IF;
      END LOOP;
    END IF;
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_attempt->>'target_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    IF NULLIF(btrim(COALESCE(p_attempt->>'descriptor','')),'') IS NULL THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  IF p_legacy_types THEN
    final_type := CASE ev_type
      WHEN 'Communicated' THEN 'private_disclosure'
      WHEN 'ActorMoved'   THEN 'move'
      ELSE ev_type
    END;
  ELSE
    final_type := ev_type;
  END IF;

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin, payload)
  VALUES (ev_id, p_world_id, final_type, p_attempt->>'stated',
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin,
          -- payload.spoken: unchanged. payload.speech_perception: preserved verbatim, alongside it,
          -- whenever the caller attached one — same durability, same "stored only when non-empty,
          -- and only for speech" rule. Ordinary speech never carries payload.visible: it has no
          -- hidden concept, and a missing key is correctly read downstream as "not hidden".
          (CASE WHEN ev_type = 'Communicated'
                 AND NULLIF(TRIM(COALESCE(p_attempt->>'content','')),'') IS NOT NULL
                THEN jsonb_build_object('spoken', TRIM(p_attempt->>'content'))
                ELSE '{}'::jsonb END)
          || (CASE WHEN ev_type = 'Communicated' AND p_attempt ? 'speech_perception'
                    THEN jsonb_build_object('speech_perception', p_attempt->'speech_perception')
                    ELSE '{}'::jsonb END));

  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, p_actor_id, 'actor', 'speaker'),
      (ev_id, listener,   'actor', 'listener');
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, p_actor_id, 'actor', 'instigator');
  END IF;

  IF ev_type = 'ObjectRelocated' AND jsonb_typeof(p_attempt->'witnesses') = 'array' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
    SELECT ev_id, w.eid, 'actor', 'witness'
      FROM (SELECT DISTINCT (value #>> '{}')::uuid AS eid
              FROM jsonb_array_elements(p_attempt->'witnesses')) w
     WHERE w.eid IS NOT NULL
       AND w.eid <> p_actor_id
       AND w.eid <> dest_eid
    ON CONFLICT DO NOTHING;
  END IF;

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, p_actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  IF ev_type = 'EntityCreated' THEN
    PERFORM fn_apply_entity_created(ev_id, p_world_id,
      NULLIF(p_attempt->>'target_id','')::uuid,
      p_attempt->>'new_entity_kind',
      p_attempt->>'canonical_name',
      p_attempt->>'descriptor',
      COALESCE(p_attempt->'new_attrs', '{}'::jsonb));
  END IF;

  PERFORM generate_perceptions(ev_id);

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 8. apply_ruled_event — same structural gate (only when visible), persists payload.visible always
--    for Communicated; Communicated carved out of the generic broadcast loop into the shared
--    application function, passing the bare per-receiver account only
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION public.apply_ruled_event(p_world_id uuid, p_ruled jsonb, p_tick bigint, p_seq integer, p_origin text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  ev_type    text;
  actor_id   uuid;
  ev_id      uuid;
  listener   uuid;
  to_target  uuid;
  v_scene    uuid;
  v_coord    jsonb;
  object_eid uuid;
  dest_eid   uuid;
  target_eid uuid;
  here       uuid;
  vis_scope  text;
  truth_text text;
  appear_txt text;
  visible    boolean;
  receiver   uuid;
  recv_text  text;
  var_text   text;
  pid        uuid;
  participant_ids uuid[];
  v_old_holder uuid;
BEGIN
  ev_type  := p_ruled->>'type';
  actor_id := (p_ruled->>'actor_id')::uuid;
  truth_text := p_ruled->>'truth';
  appear_txt := NULLIF(TRIM(COALESCE(p_ruled->>'appearance', '')), '');
  visible    := CASE
    WHEN p_ruled ? 'visible' AND (p_ruled->>'visible') = 'false' THEN false
    ELSE true
  END;

  IF NOT EXISTS (
    SELECT 1 FROM entity_registry
    WHERE entity_id = actor_id AND world_id = p_world_id AND entity_kind = 'actor'
  ) THEN
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  IF ev_type = 'ActorMoved' THEN
    to_target := (p_ruled->>'to_target_id')::uuid;
    IF NOT fn_actor_move_permitted(p_world_id, actor_id, to_target) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'Communicated' THEN
    listener := (p_ruled->>'listener_id')::uuid;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = actor_id;
    IF NOT EXISTS (
      SELECT 1 FROM fn_actors_at(p_world_id, here) WHERE entity_id = listener
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

    -- Visible ruled speech requires an already-judged speech_perception, same structural check as
    -- the ordinary door. Hidden speech (visible:false) is explicitly exempt: Go skips the model
    -- entirely and sends no speech_perception field, and payload.visible=false (persisted below)
    -- is what makes fn_apply_speech_perception itself refuse to require one.
    IF visible AND (
         jsonb_typeof(p_ruled->'speech_perception') IS DISTINCT FROM 'object'
         OR jsonb_typeof(p_ruled->'speech_perception'->'listeners') IS DISTINCT FROM 'array'
       ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'ObjectRelocated' THEN
    object_eid := (p_ruled->>'object_id')::uuid;
    dest_eid   := (p_ruled->>'dest_id')::uuid;
    IF NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = object_eid AND world_id = p_world_id)
    OR NOT EXISTS (SELECT 1 FROM entity_registry WHERE entity_id = dest_eid  AND world_id = p_world_id)
    THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;
    IF EXISTS (SELECT 1 FROM artifact_state
               WHERE world_id = p_world_id AND entity_id = dest_eid AND attrs ? 'max_room') THEN
      IF fn_occupied_room(p_world_id, dest_eid)
         + fn_volume(COALESCE((SELECT (attrs->>'size')::int FROM artifact_state
                               WHERE world_id = p_world_id AND entity_id = object_eid), 1))
         > (SELECT (attrs->>'max_room')::numeric FROM artifact_state
            WHERE world_id = p_world_id AND entity_id = dest_eid)
      THEN
        RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
      END IF;
    END IF;

  ELSIF ev_type IN ('OwnershipAccessChanged', 'EntityDestroyed', 'AttributeChanged') THEN
    target_eid := (p_ruled->>'target_id')::uuid;
    IF NOT EXISTS (
      SELECT 1 FROM entity_registry WHERE entity_id = target_eid AND world_id = p_world_id
    ) THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSIF ev_type = 'EntityCreated' THEN
    IF NULLIF(btrim(COALESCE(p_ruled->>'descriptor','')),'') IS NULL THEN
      RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
    END IF;

  ELSE
    RETURN jsonb_build_object('event_id', NULL, 'halt_reason', 'gate_reject');
  END IF;

  ev_id := gen_random_uuid();

  vis_scope := CASE ev_type WHEN 'Communicated' THEN 'private' ELSE 'public' END;

  INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                           status, accepted_at, visibility_scope, origin, payload)
  VALUES (ev_id, p_world_id, ev_type, truth_text,
          p_tick, p_seq, 'accepted', now(), vis_scope, p_origin,
          (CASE WHEN ev_type = 'Communicated'
                 AND NULLIF(TRIM(COALESCE(p_ruled->>'content','')),'') IS NOT NULL
                THEN jsonb_build_object('spoken', TRIM(p_ruled->>'content'))
                ELSE '{}'::jsonb END)
          || (CASE WHEN ev_type = 'Communicated' AND p_ruled ? 'speech_perception'
                    THEN jsonb_build_object('speech_perception', p_ruled->'speech_perception')
                    ELSE '{}'::jsonb END)
          -- Persisted unconditionally for Communicated (true or false) so fn_apply_speech_perception
          -- can decide hidden-vs-visible from the event itself, explicitly, rather than trusting
          -- that it is only ever invoked when visible.
          || (CASE WHEN ev_type = 'Communicated'
                    THEN jsonb_build_object('visible', visible)
                    ELSE '{}'::jsonb END));

  IF ev_type = 'Communicated' THEN
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier) VALUES
      (ev_id, actor_id, 'actor', 'speaker'),
      (ev_id, listener, 'actor', 'listener');
    participant_ids := ARRAY[actor_id, listener];
  ELSE
    INSERT INTO event_participant (event_id, entity_id, entity_kind, role_qualifier)
      VALUES (ev_id, actor_id, 'actor', 'instigator');
    participant_ids := ARRAY[actor_id];
  END IF;

  IF ev_type = 'ActorMoved' THEN
    SELECT scene, coord INTO v_scene, v_coord FROM fn_target_position(p_world_id, to_target);
    INSERT INTO state_mutation (world_id, event_id, entity_id, entity_kind, attribute_path,
                                new_value, valid_from_tick, valid_from_seq)
    VALUES
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.location_id', to_jsonb(v_scene::text), p_tick, p_seq),
      (p_world_id, ev_id, actor_id, 'actor', 'attrs.coordinates', v_coord,                 p_tick, p_seq);
  END IF;

  IF ev_type = 'ObjectRelocated' THEN
    SELECT (attrs->>'contained_by')::uuid INTO v_old_holder
      FROM artifact_state WHERE world_id = p_world_id AND entity_id = object_eid;
    PERFORM fn_apply_carry_change(ev_id, p_world_id, object_eid, v_old_holder, dest_eid);
  END IF;

  IF ev_type = 'EntityCreated' THEN
    PERFORM fn_apply_entity_created(ev_id, p_world_id,
      NULLIF(p_ruled->>'target_id','')::uuid,
      p_ruled->>'new_entity_kind',
      p_ruled->>'canonical_name',
      p_ruled->>'descriptor',
      COALESCE(p_ruled->'new_attrs', '{}'::jsonb));
  END IF;

  IF NOT visible THEN
    RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
  END IF;

  SELECT (a.attrs->>'location_id')::uuid INTO here
    FROM actor_state a
    WHERE a.world_id = p_world_id AND a.entity_id = actor_id;

  IF ev_type = 'Communicated' THEN
    -- SHARED PERCEPTION APPLICATION (migration header): the ruled door no longer broadcasts flat
    -- 'direct' text to every co-present actor for speech. Receiver_variants still differentiate
    -- WHAT each listener's bare account looks like (resolved into a listener_id->text map here,
    -- unchanged mechanism, never a pre-baked quote); WHO perceives at all, what she learns, and the
    -- quote appended after walling are now the shared function's job — the same one the ordinary
    -- door uses.
    DECLARE
      v_listener_accounts jsonb := '{}'::jsonb;
      v_speaker_txt        text;
      v_recv_txt           text;
    BEGIN
      FOR receiver IN SELECT entity_id FROM fn_actors_at(p_world_id, here) WHERE entity_id <> actor_id LOOP
        v_recv_txt := NULL;
        IF p_ruled ? 'receiver_variants' THEN
          SELECT rv->>'text' INTO v_recv_txt FROM jsonb_array_elements(p_ruled->'receiver_variants') AS rv
            WHERE (rv->>'receiver_id')::uuid = receiver LIMIT 1;
        END IF;
        v_listener_accounts := v_listener_accounts
          || jsonb_build_object(receiver::text, COALESCE(v_recv_txt, appear_txt, truth_text));
      END LOOP;

      v_speaker_txt := NULL;
      IF p_ruled ? 'receiver_variants' THEN
        SELECT rv->>'text' INTO v_speaker_txt FROM jsonb_array_elements(p_ruled->'receiver_variants') AS rv
          WHERE (rv->>'receiver_id')::uuid = actor_id LIMIT 1;
      END IF;

      PERFORM fn_apply_speech_perception(
        ev_id, COALESCE(v_speaker_txt, appear_txt, truth_text), v_listener_accounts);
    END;
  ELSE
    FOR receiver IN
      SELECT entity_id FROM fn_actors_at(p_world_id, here)
      UNION
      SELECT actor_id
    LOOP
      var_text := NULL;
      IF p_ruled ? 'receiver_variants' THEN
        SELECT rv->>'text' INTO var_text
          FROM jsonb_array_elements(p_ruled->'receiver_variants') AS rv
          WHERE (rv->>'receiver_id')::uuid = receiver
          LIMIT 1;
      END IF;

      recv_text := fn_viewer_text(p_world_id, receiver, COALESCE(var_text, appear_txt, truth_text));

      INSERT INTO perception_record (world_id, holder_id, source_event_id, content, epistemic_type,
                                     acquired_tick, valid_tick)
      VALUES (p_world_id, receiver, ev_id, recv_text, 'direct', p_tick, p_tick)
      RETURNING perception_id INTO pid;

      DECLARE
        part_id uuid;
      BEGIN
        FOREACH part_id IN ARRAY participant_ids LOOP
          INSERT INTO perception_subject (perception_id, entity_id, world_id)
            VALUES (pid, part_id, p_world_id);
        END LOOP;
      END;
    END LOOP;
  END IF;

  RETURN jsonb_build_object('event_id', ev_id, 'halt_reason', 'committed');
END $$;

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 9. apply_beat — forward an already-accepted speech_perception on say steps (never a scanner
--    fallback: a say step with none now gate_rejects downstream in apply_event, exactly as any
--    other caller-supplied Communicated attempt would)
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION public.apply_beat(p_world_id uuid, p_actor_id uuid, p_chain jsonb, p_start_tick bigint, p_tick_cap bigint, p_origin text DEFAULT 'fast_path'::text) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  step       jsonb;
  idx        int := 0;
  cur_tick   bigint := p_start_tick;
  cur_seq    int := 0;
  start_tick bigint := p_start_tick;
  committed  jsonb := '[]'::jsonb;
  halt       text := 'completed';
  dur        bigint;
  here       uuid;
  listener   uuid;
  next_step  jsonb;
  next_ok    boolean;
  attempt    jsonb;
  result     jsonb;
BEGIN
  FOR step IN SELECT * FROM jsonb_array_elements(p_chain) LOOP
    idx := idx + 1;
    SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
      WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;

    IF step->>'type' = 'say' THEN
      listener := (step->>'listener')::uuid;
      attempt := jsonb_build_object(
        'type',        'Communicated',
        'stated',      COALESCE(step->>'content', 'say'),
        'listener_id', listener,
        'content',     step->>'content',
        -- Forwarded as-is: an accepted speech_perception the caller already attached to this step.
        -- Absent (jsonb NULL when the key is missing) forwards as JSON null, which apply_event's
        -- structural gate correctly refuses for speech — no scanner fallback resurrected here.
        'speech_perception', step->'speech_perception'
      );
      dur := 0;
    ELSIF step->>'type' = 'move' THEN
      attempt := jsonb_build_object(
        'type',         'ActorMoved',
        'stated',       'move',
        'to_target_id', step->>'to'
      );
      dur := fn_move_duration(p_world_id, here, (step->>'to')::uuid);
    ELSE
      halt := 'gate_reject'; EXIT;
    END IF;

    IF (cur_tick + dur) - start_tick > p_tick_cap THEN
      halt := 'turn_budget'; EXIT;
    END IF;

    result := apply_event(p_world_id, p_actor_id, attempt, cur_tick, cur_seq, p_origin, true);

    IF result->>'halt_reason' = 'gate_reject' THEN
      halt := 'gate_reject'; EXIT;
    END IF;

    committed := committed || to_jsonb(result->>'event_id');

    IF dur > 0 THEN cur_tick := cur_tick + dur; cur_seq := 0; ELSE cur_seq := cur_seq + 1; END IF;

    next_step := p_chain -> idx;
    IF step->>'type' = 'move' AND next_step IS NOT NULL AND next_step->>'type' = 'say' THEN
      SELECT (a.attrs->>'location_id')::uuid INTO here FROM actor_state a
        WHERE a.world_id = p_world_id AND a.entity_id = p_actor_id;
      next_ok := EXISTS (SELECT 1 FROM fn_actors_at(p_world_id, here)
                         WHERE entity_id = (next_step->>'listener')::uuid);
      IF NOT next_ok THEN halt := 'stop_check'; EXIT; END IF;
    END IF;
  END LOOP;

  RETURN jsonb_build_object('committed', committed, 'halt_reason', halt,
                            'ticks_advanced', cur_tick - start_tick);
END $$;

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 10. fn_unheard_names / fn_perceived_speech — the two new read functions Presentation consumes
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

CREATE FUNCTION public.fn_unheard_names(p_world_id uuid, p_viewer uuid)
  RETURNS TABLE(canonical_name text, label text)
  LANGUAGE sql STABLE
  AS $$
  SELECT u.canonical_name, u.label
  FROM fn_unearned_names(p_world_id, p_viewer) u
  WHERE NOT EXISTS (
    SELECT 1
    FROM fn_visible_perceptions(p_world_id, p_viewer) vp
    WHERE vp.spoken IS NOT NULL
      AND vp.spoken ~ ('\m' || fn_regexp_quote(u.canonical_name) || '\M')
  )
$$;

COMMENT ON FUNCTION public.fn_unheard_names(uuid, uuid) IS
  'fn_unearned_names, minus any name/token the viewer LITERALLY heard spoken (verbatim in her own '
  'visible perception_record.spoken) — case-SENSITIVE, word-bounded, same strictness as the '
  'hearing-teaches path this migration retires. A lexical fact only: never writes name_knowledge, '
  'never claims to prove the prose interpreted the word correctly. fn_viewer_text (the account wall) '
  'is unaffected and keeps using fn_unearned_names unchanged — this is a separate, more permissive '
  'guard for the Go-side heard-words belt only.';

CREATE FUNCTION public.fn_perceived_speech(p_world_id uuid, p_viewer uuid, p_since_tick bigint)
  RETURNS TABLE(speaker_id uuid, spoken text)
  LANGUAGE sql STABLE
  AS $$
  SELECT ep.entity_id AS speaker_id, vp.spoken
  FROM fn_visible_perceptions(p_world_id, p_viewer) vp
  JOIN event_participant ep ON ep.event_id = vp.source_event_id AND ep.role_qualifier = 'speaker'
  WHERE vp.spoken IS NOT NULL
    AND vp.acquired_tick >= p_since_tick
  ORDER BY vp.acquired_tick, vp.perception_id;
$$;

COMMENT ON FUNCTION public.fn_perceived_speech(uuid, uuid, bigint) IS
  'The heard words a viewer currently, validly holds (fn_visible_perceptions filtered to spoken IS '
  'NOT NULL), paired with the speaker of that perception''s source event, since a given tick. Backs '
  'beatHandler.speechTexts in place of treating any perception attached to a speech event as heard '
  'words — a blocked listener holds no row here at all, and an attended one with no recoverable '
  'words is correctly absent too.';

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 11. Remove the regex scanner as a learning source (dead code: its one caller, the old
--     generate_perceptions Communicated arm, is gone; fn_unearned_names never used it)
-- ═══════════════════════════════════════════════════════════════════════════════════════════════

DROP FUNCTION IF EXISTS public.fn_names_in_text(uuid, text);

-- ═══════════════════════════════════════════════════════════════════════════════════════════════
-- 12. Cognition shared-moment definition: a SPEECH source is shared only on unanimous content
-- ═══════════════════════════════════════════════════════════════════════════════════════════════
--
-- fn_public_moment / fn_isolated_npcs / fn_private_records (20260724110003) all define "shared" as
-- "every present holder holds a perception of this source event", then pick the MODAL (most common)
-- content as its one public face. That rule is correct for non-speech perception, where everyone's
-- account of an observed event is expected to agree and a stray divergent read is the interesting
-- case to isolate. It is WRONG for speech once receiver variants and per-listener heard_words exist
-- (this migration): several present holders can each hold a GENUINELY DIFFERENT, equally legitimate
-- account of the SAME Communicated event, and the old rule would let whichever content happened to
-- be more common stand in as "what was publicly said" — silently handing a majority's (or even a
-- two-out-of-three plurality's) words to a listener who does not share them, exactly the leak the
-- receiver-variant/heard_words machinery exists to prevent.
--
-- Fix, applied identically to all three functions' shared CTE: a SPEECH source (event_type
-- Communicated or private_disclosure) is shared only when every present holder's content is
-- IDENTICAL (count(DISTINCT content) = 1), not merely modal-majority. Anything less unanimous stays
-- private to each holder individually — which is the correct outcome: two divergent listeners are
-- each isolated for that record, not folded into a batch face neither of them actually perceived.
-- Non-speech sources are completely unaffected; the modal rule is unchanged for them (regression:
-- 107_cognition_lookups_test.sql case (b), an 'observation' event, still asserts the modal-majority
-- behavior and still passes). New regression for the speech case: 107_cognition_lookups_test.sql
-- cases (j)-(l).

CREATE OR REPLACE FUNCTION public.fn_public_moment(
    p_world_id uuid,
    p_present  uuid[],
    p_k        int
)
RETURNS TABLE(source_event_id uuid, acquired_tick bigint, content text)
LANGUAGE sql
STABLE
AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick, ce.event_type
    FROM perception_record pr
    JOIN canon_event ce ON ce.event_id = pr.source_event_id
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3, 5
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content,
           min(h.tick) AS tick
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
       AND (
         NOT bool_or(h.event_type IN ('Communicated', 'private_disclosure'))
         OR count(DISTINCT h.content) = 1
       )
  ), recent AS (
    -- the LAST p_k shared source events by tick (most recent), deterministic tie-break by id
    SELECT s.source_event_id, s.tick, s.modal_content
    FROM shared s
    ORDER BY s.tick DESC, s.source_event_id DESC
    LIMIT p_k
  )
  SELECT r.source_event_id, r.tick AS acquired_tick, r.modal_content AS content
  FROM recent r
  ORDER BY r.tick ASC, r.source_event_id ASC   -- append-only, cache-native
$$;

CREATE OR REPLACE FUNCTION public.fn_isolated_npcs(
    p_world_id   uuid,
    p_action_ids uuid[],
    p_present    uuid[],
    p_npcs       uuid[]
)
RETURNS TABLE(actor_id uuid)
LANGUAGE sql
STABLE
AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick, ce.event_type
    FROM perception_record pr
    JOIN canon_event ce ON ce.event_id = pr.source_event_id
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3, 5
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
       AND (
         NOT bool_or(h.event_type IN ('Communicated', 'private_disclosure'))
         OR count(DISTINCT h.content) = 1
       )
  )
  -- an NPC is isolated iff she holds >=1 PRIVATE record whose about-ness intersects the action ids
  SELECT DISTINCT pr.holder_id AS actor_id
  FROM perception_record pr
  LEFT JOIN shared s
    ON s.source_event_id = pr.source_event_id
   AND s.modal_content   = pr.content        -- matches only the public (modal) face
  WHERE pr.world_id = p_world_id
    AND pr.holder_id = ANY(p_npcs)
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    AND s.source_event_id IS NULL             -- private: no matching public face
    -- name-knowledge is an identity substrate, not a secret: a world_genesis-sourced record must
    -- never pull its holder isolated (§3; mirrors the fn_actor_page tripwire).
    AND NOT EXISTS (
      SELECT 1 FROM canon_event ce
      WHERE ce.event_id = pr.source_event_id AND ce.event_type = 'world_genesis'
    )
    AND EXISTS (
      SELECT 1 FROM perception_subject ps
      WHERE ps.perception_id = pr.perception_id
        AND ps.entity_id = ANY(p_action_ids)  -- one-hop id intersection (ADR-035)
    )
$$;

CREATE OR REPLACE FUNCTION public.fn_private_records(
    p_world_id   uuid,
    p_npc        uuid,
    p_action_ids uuid[],
    p_present    uuid[]
)
RETURNS TABLE(content text, acquired_tick bigint)
LANGUAGE sql
STABLE
AS $$
  WITH held AS (
    -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
    SELECT pr.source_event_id, pr.holder_id, pr.content, min(pr.acquired_tick) AS tick, ce.event_type
    FROM perception_record pr
    JOIN canon_event ce ON ce.event_id = pr.source_event_id
    WHERE pr.world_id = p_world_id AND pr.holder_id = ANY(p_present)
      AND pr.source_event_id IS NOT NULL
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
    GROUP BY 1, 2, 3, 5
  ), shared AS (
    SELECT h.source_event_id,
           mode() WITHIN GROUP (ORDER BY h.content) AS modal_content
    FROM held h
    GROUP BY h.source_event_id
    HAVING count(DISTINCT h.holder_id) = cardinality(p_present)
       AND (
         NOT bool_or(h.event_type IN ('Communicated', 'private_disclosure'))
         OR count(DISTINCT h.content) = 1
       )
  ), freshest AS (
    -- cap is a v1 dial; §10's retrieval assembly refines it in Station I. Keep the FRESHEST 20, not
    -- the oldest: a private cap must drop old records first, never the ones most likely to matter
    -- now. Full tie-break (content, perception_id) keeps the 20-row cut deterministic.
    SELECT pr.content, pr.acquired_tick, pr.perception_id
    FROM perception_record pr
    LEFT JOIN shared s
      ON s.source_event_id = pr.source_event_id
     AND s.modal_content   = pr.content
    WHERE pr.world_id = p_world_id
      AND pr.holder_id = p_npc
      -- cognition reads CURRENT knowledge only; a stale copy must never flip the modal face.
      AND pr.invalid_tick IS NULL AND pr.expired_at IS NULL
      AND s.source_event_id IS NULL             -- private records only
      -- name-knowledge is an identity substrate, not a secret: a world_genesis-sourced record must
      -- never ride the private block (§3; mirrors the fn_actor_page tripwire).
      AND NOT EXISTS (
        SELECT 1 FROM canon_event ce
        WHERE ce.event_id = pr.source_event_id AND ce.event_type = 'world_genesis'
      )
      AND EXISTS (
        SELECT 1 FROM perception_subject ps
        WHERE ps.perception_id = pr.perception_id
          AND ps.entity_id = ANY(p_action_ids)  -- subjects intersect the action's bound ids
      )
    ORDER BY pr.acquired_tick DESC, pr.content DESC, pr.perception_id DESC
    LIMIT 20
  )
  SELECT f.content, f.acquired_tick
  FROM freshest f
  ORDER BY f.acquired_tick ASC, f.content ASC, f.perception_id ASC   -- present oldest-of-the-freshest first
$$;

-- migrate:down
--
-- REFUSES OUTRIGHT. apply_event, apply_ruled_event, generate_perceptions, and apply_beat are
-- REWRITTEN (not reverted) above to require an accepted speech_perception judgment and to route
-- every Communicated commit through fn_apply_speech_perception. fn_public_moment, fn_isolated_npcs,
-- and fn_private_records are also rewritten (not reverted) to require unanimous content for a
-- speech source to count as shared. Dropping fn_apply_speech_perception (or the
-- perception_record.spoken column those bodies read and write) while leaving those seven callers in
-- their post-migration form would not restore prior behavior — it would break every future
-- Communicated commit outright, and it would destroy already-recorded heard words for perceptions
-- written after this migration applied. A partial reverse that deletes some dependencies while
-- leaving their callers unchanged is worse than no reverse at all, so this migration does not
-- attempt one: rolling back the schema changes here requires first rolling back (or hand-reverting)
-- all seven function bodies above, which this migration deliberately does not do, for the same
-- reason 20260814170000 does not restore the breach it fixed.
--
-- This mirrors the release gate in the migration header: do not apply this migration to production
-- while the old binary still serves, and do not treat a binary-only rollback as safe once it has
-- been applied. If a genuine rollback is ever required, it needs a hand-authored migration that
-- restores all seven function bodies to their pre-20260906184826 form AND repopulates
-- perception_record.spoken's dependents, not this file's `down`.
DO $$ BEGIN
  RAISE EXCEPTION
    'migration 20260906184826_shared_speech_perception is forward-only: apply_event, '
    'apply_ruled_event, generate_perceptions, and apply_beat are rewritten (not reverted) to '
    'require fn_apply_speech_perception, so dropping it or perception_record.spoken here would '
    'break every future Communicated commit rather than restore prior behavior. There is no safe '
    'automatic reverse for this migration; see its own migrate:down comment for what a genuine '
    'rollback would require.';
END $$;
