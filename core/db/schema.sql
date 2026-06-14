SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: state_mutation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.state_mutation (
    mutation_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    event_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    entity_kind text NOT NULL,
    attribute_path text NOT NULL,
    old_value jsonb,
    new_value jsonb NOT NULL,
    valid_from_tick bigint NOT NULL,
    valid_from_seq integer DEFAULT 0 NOT NULL,
    status text DEFAULT 'applied'::text NOT NULL,
    CONSTRAINT state_mutation_status_check CHECK ((status = ANY (ARRAY['applied'::text, 'reversed'::text, 'dirty'::text])))
);


--
-- Name: apply_mutation(public.state_mutation); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.apply_mutation(m public.state_mutation) RETURNS void
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE
  -- strip leading 'attrs.' (6 chars) -> single-key JSON path under attrs (0A convention, Rider B)
  jpath text[] := string_to_array(substring(m.attribute_path from 7), '.');
BEGIN
  IF m.entity_kind = 'actor' THEN
    INSERT INTO actor_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(actor_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'location' THEN
    INSERT INTO location_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(location_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'artifact' THEN
    INSERT INTO artifact_state (entity_id, world_id, attrs, last_event_id, updated_at)
    VALUES (m.entity_id, m.world_id, jsonb_set('{}'::jsonb, jpath, m.new_value, true), m.event_id, now())
    ON CONFLICT (entity_id) DO UPDATE
      SET attrs = jsonb_set(artifact_state.attrs, jpath, m.new_value, true),
          last_event_id = m.event_id, updated_at = now();
  ELSIF m.entity_kind = 'relationship' THEN
    -- SPEC-001: doc 03 does not define mutation->(a_id,b_id) addressing. NO-OP stub in 0A.
    NULL;
  END IF;
END $$;


--
-- Name: canon_event_append_only(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.canon_event_append_only() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  IF ROW(NEW.event_id, NEW.world_id, NEW.scene_id, NEW.beat_id, NEW.event_type, NEW.summary,
         NEW.payload, NEW.schema_version, NEW.in_world_tick, NEW.in_world_label, NEW.beat_seq,
         NEW.temporal_uncertainty, NEW.recorded_at, NEW.visibility_scope, NEW.confidence,
         NEW.origin, NEW.template_id, NEW.source_refs)
     IS DISTINCT FROM
     ROW(OLD.event_id, OLD.world_id, OLD.scene_id, OLD.beat_id, OLD.event_type, OLD.summary,
         OLD.payload, OLD.schema_version, OLD.in_world_tick, OLD.in_world_label, OLD.beat_seq,
         OLD.temporal_uncertainty, OLD.recorded_at, OLD.visibility_scope, OLD.confidence,
         OLD.origin, OLD.template_id, OLD.source_refs)
  THEN
    RAISE EXCEPTION 'canon_event is append-only: only {status, accepted_at, superseded_by} may change (event %)', OLD.event_id;
  END IF;

  IF OLD.status IS DISTINCT FROM NEW.status
     AND NOT ( (OLD.status='proposed' AND NEW.status IN ('accepted','rejected'))
            OR (OLD.status='accepted' AND NEW.status IN ('retconned','superseded')) ) THEN
    RAISE EXCEPTION 'illegal canon_event status transition % -> %', OLD.status, NEW.status;
  END IF;
  RETURN NEW;
END $$;


--
-- Name: causal_bundle_append_only(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.causal_bundle_append_only() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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


--
-- Name: causal_bundle_assert_acyclic(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.causal_bundle_assert_acyclic() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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


--
-- Name: causal_bundle_input_immutable(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.causal_bundle_input_immutable() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  RAISE EXCEPTION 'causal_bundle_input is immutable: UPDATE forbidden (bundle %, input %)',
    OLD.bundle_id, OLD.input_ref;
END $$;


--
-- Name: fn_actor_page(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_actor_page(p_world_id uuid, p_viewer_id uuid, p_actor_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT CASE WHEN NOT fn_entity_visible(p_world_id, p_viewer_id, p_actor_id) THEN NULL
  ELSE json_build_object(
    'schema_version', 'actor_page/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'actor', json_build_object(
      'id',                         p_actor_id,
      'perceived_name',             fn_perceived_name(p_world_id, p_viewer_id, p_actor_id),
      'perceived_role',             NULL,
      'current_synthesis',          NULL,
      'last_known_status',          NULL,
      'known_artifacts',            '[]'::json,
      'collected_knowledge_groups', fn_collected_knowledge(p_world_id, p_viewer_id, p_actor_id),
      'inline_links',               '[]'::json
    )
  ) END;
$$;


--
-- Name: fn_collected_knowledge(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_collected_knowledge(p_world_id uuid, p_viewer_id uuid, p_target_id uuid) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  WITH about AS (
    SELECT v.perception_id, v.content, v.epistemic_type, v.valid_tick, v.confidence, ce.in_world_label
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) v
    JOIN perception_subject ps ON ps.perception_id = v.perception_id AND ps.entity_id = p_target_id
    JOIN canon_event ce ON ce.event_id = v.source_event_id
    WHERE ce.event_type <> 'world_genesis'
  ),
  items AS (
    SELECT json_build_object(
             'perception_id',    a.perception_id,
             'content',          a.content,
             'epistemic_type',   a.epistemic_type,
             'occurred_at_tick', a.valid_tick,
             'display_label',    a.in_world_label,
             'confidence',       a.confidence,
             'decay',  json_build_object('stale', false, 'last_confirmed_label', a.in_world_label),
             'source', json_build_object('epistemic_type', a.epistemic_type,
                                         'source_event_label', a.in_world_label)
           ) AS item,
           a.valid_tick AS sort_tick
    FROM about a
  )
  SELECT CASE WHEN count(*) = 0 THEN '[]'::json
              ELSE json_build_array(json_build_object(
                     'group_key',   p_target_id::text,
                     'group_label', fn_perceived_name(p_world_id, p_viewer_id, p_target_id),
                     'items',       coalesce(json_agg(i.item ORDER BY i.sort_tick), '[]'::json)
                   ))
         END
  FROM items i;
$$;


--
-- Name: fn_compendium_index(uuid, uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_index(p_world_id uuid, p_viewer_id uuid, p_kind text) RETURNS TABLE(entity_id uuid, perceived_name text)
    LANGUAGE sql STABLE
    AS $$
  SELECT DISTINCT er.entity_id,
         fn_perceived_name(p_world_id, p_viewer_id, er.entity_id)
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp       -- FILTER 1, unchanged
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id
  JOIN entity_registry er ON er.entity_id = ps.entity_id AND er.world_id = p_world_id
  WHERE er.entity_kind = p_kind;
$$;


--
-- Name: fn_compendium_index_json(uuid, uuid, text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_compendium_index_json(p_world_id uuid, p_viewer_id uuid, p_kind text) RETURNS json
    LANGUAGE sql STABLE
    AS $$
  SELECT json_build_object(
    'schema_version', 'compendium_index/1',
    'world_id',  p_world_id,
    'viewer_id', p_viewer_id,
    'kind',      p_kind,
    'entries', coalesce(
      (SELECT json_agg(json_build_object('id', entity_id, 'perceived_name', perceived_name)
                       ORDER BY entity_id)
       FROM fn_compendium_index(p_world_id, p_viewer_id, p_kind)), '[]'::json)
  );
$$;


--
-- Name: fn_entity_visible(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_entity_visible(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS boolean
    LANGUAGE sql STABLE
    AS $$
  SELECT EXISTS (
    SELECT 1
    FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp     -- FILTER 1, unchanged
    JOIN perception_subject ps ON ps.perception_id = vp.perception_id
    WHERE ps.entity_id = p_entity_id);
$$;


--
-- Name: fn_perceived_name(uuid, uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_perceived_name(p_world_id uuid, p_viewer_id uuid, p_entity_id uuid) RETURNS text
    LANGUAGE sql STABLE
    AS $$
  SELECT vp.content
  FROM fn_visible_perceptions(p_world_id, p_viewer_id) vp
  JOIN perception_subject ps ON ps.perception_id = vp.perception_id AND ps.entity_id = p_entity_id
  JOIN canon_event ce ON ce.event_id = vp.source_event_id
  WHERE ce.event_type = 'world_genesis'
  ORDER BY vp.acquired_tick
  LIMIT 1;
$$;


--
-- Name: perception_record; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perception_record (
    perception_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    holder_id uuid NOT NULL,
    source_event_id uuid NOT NULL,
    content text NOT NULL,
    epistemic_type text NOT NULL,
    sensory_mode text,
    confidence real DEFAULT 1.0 NOT NULL,
    distortion_level real DEFAULT 0 NOT NULL,
    acquired_tick bigint NOT NULL,
    valid_tick bigint NOT NULL,
    invalid_tick bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expired_at timestamp with time zone,
    visibility_scope text DEFAULT 'private'::text NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    importance real DEFAULT 5.0 NOT NULL,
    CONSTRAINT perception_record_epistemic_type_check CHECK ((epistemic_type = ANY (ARRAY['direct'::text, 'shared'::text, 'told'::text, 'overheard'::text, 'public'::text, 'rumor'::text, 'inference'::text, 'mistaken'::text, 'confirmed'::text, 'disputed'::text])))
);


--
-- Name: fn_visible_perceptions(uuid, uuid); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.fn_visible_perceptions(p_world_id uuid, p_viewer_id uuid) RETURNS SETOF public.perception_record
    LANGUAGE sql STABLE
    AS $$
  SELECT pr.*
  FROM perception_record pr
  WHERE pr.world_id = p_world_id
    AND pr.invalid_tick IS NULL
    AND pr.expired_at  IS NULL
    AND ( pr.holder_id = p_viewer_id
          OR pr.holder_id IN (
            SELECT er.entity_id FROM entity_registry er
            WHERE er.world_id = p_world_id
              AND er.entity_kind IN ('faction','group')
          )
        );
$$;


--
-- Name: forbid_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.forbid_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  RAISE EXCEPTION 'DELETE forbidden on % (append-only canon, ADR-001/006)', TG_TABLE_NAME;
END $$;


--
-- Name: replay_0a(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.replay_0a() RETURNS boolean
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
DECLARE ev RECORD; m state_mutation; diff_count int;
BEGIN
  DROP TABLE IF EXISTS snap_actor, snap_location, snap_artifact, snap_rel;
  CREATE TEMP TABLE snap_actor    ON COMMIT DROP AS SELECT * FROM actor_state;
  CREATE TEMP TABLE snap_location ON COMMIT DROP AS SELECT * FROM location_state;
  CREATE TEMP TABLE snap_artifact ON COMMIT DROP AS SELECT * FROM artifact_state;
  CREATE TEMP TABLE snap_rel      ON COMMIT DROP AS SELECT * FROM relationship_state;

  TRUNCATE actor_state, location_state, artifact_state, relationship_state;

  -- Rider C: domain-only deterministic order. recorded_at (volatile) excluded.
  FOR ev IN SELECT event_id FROM canon_event WHERE status='accepted'
            ORDER BY world_id, in_world_tick, beat_seq LOOP
    FOR m IN SELECT * FROM state_mutation WHERE event_id = ev.event_id
             ORDER BY valid_from_tick, valid_from_seq LOOP
      PERFORM apply_mutation(m);
    END LOOP;
  END LOOP;

  -- §6.5.1 per-table domain diff (exclude volatile updated_at; identity = PK).
  SELECT
      (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_actor
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM actor_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_location
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM location_state)) d)
    + (SELECT count(*) FROM (
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact)
        UNION ALL
        (SELECT entity_id,world_id,attrs,last_event_id,dirty FROM snap_artifact
         EXCEPT SELECT entity_id,world_id,attrs,last_event_id,dirty FROM artifact_state)) d)
    + (SELECT count(*) FROM (
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel)
        UNION ALL
        (SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM snap_rel
         EXCEPT SELECT world_id,a_id,b_id,attrs,last_event_id,dirty FROM relationship_state)) d)
  INTO diff_count;
  RETURN diff_count = 0;
END $$;


--
-- Name: sm_project(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.sm_project() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    AS $$
BEGIN
  IF (SELECT status FROM canon_event WHERE event_id = NEW.event_id) = 'accepted' THEN
    PERFORM apply_mutation(NEW);
  END IF;
  RETURN NEW;
END $$;


--
-- Name: actor_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.actor_state (
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: artifact_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artifact_state (
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: canon_event; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canon_event (
    event_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    scene_id uuid,
    beat_id uuid,
    event_type text NOT NULL,
    summary text NOT NULL,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    schema_version integer DEFAULT 1 NOT NULL,
    in_world_tick bigint NOT NULL,
    in_world_label text,
    beat_seq integer DEFAULT 0 NOT NULL,
    temporal_uncertainty boolean DEFAULT false NOT NULL,
    recorded_at timestamp with time zone DEFAULT now() NOT NULL,
    accepted_at timestamp with time zone,
    status text DEFAULT 'proposed'::text NOT NULL,
    visibility_scope text DEFAULT 'private'::text NOT NULL,
    confidence real,
    origin text DEFAULT 'fast_path'::text NOT NULL,
    template_id text,
    source_refs jsonb,
    superseded_by uuid,
    CONSTRAINT canon_event_origin_check CHECK ((origin = ANY (ARRAY['fast_path'::text, 'template'::text, 'freeform'::text, 'threshold'::text, 'backstage'::text, 'compensation'::text]))),
    CONSTRAINT canon_event_status_check CHECK ((status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'retconned'::text, 'superseded'::text])))
);


--
-- Name: causal_bundle; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.causal_bundle (
    bundle_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    effect_ref uuid NOT NULL,
    effect_kind text NOT NULL,
    semantics text NOT NULL,
    template_id text,
    status text DEFAULT 'valid'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT causal_bundle_effect_kind_check CHECK ((effect_kind = ANY (ARRAY['event'::text, 'mutation'::text]))),
    CONSTRAINT causal_bundle_semantics_check CHECK ((semantics = ANY (ARRAY['conjunctive'::text, 'disjunctive_member'::text, 'probabilistic'::text]))),
    CONSTRAINT causal_bundle_status_check CHECK ((status = ANY (ARRAY['valid'::text, 'invalidated'::text, 'pending_review'::text])))
);


--
-- Name: causal_bundle_input; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.causal_bundle_input (
    bundle_id uuid NOT NULL,
    input_ref uuid NOT NULL,
    input_kind text NOT NULL,
    role text NOT NULL,
    polarity smallint DEFAULT 1 NOT NULL,
    weight real DEFAULT 1.0 NOT NULL,
    necessity boolean DEFAULT true NOT NULL,
    CONSTRAINT causal_bundle_input_input_kind_check CHECK ((input_kind = ANY (ARRAY['event'::text, 'mutation'::text, 'perception'::text]))),
    CONSTRAINT causal_bundle_input_polarity_check CHECK ((polarity = ANY (ARRAY[1, '-1'::integer]))),
    CONSTRAINT causal_bundle_input_role_check CHECK ((role = ANY (ARRAY['trigger'::text, 'enabler'::text, 'blocker'::text, 'influence'::text])))
);


--
-- Name: entity_registry; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_registry (
    entity_id uuid DEFAULT gen_random_uuid() NOT NULL,
    world_id uuid NOT NULL,
    entity_kind text NOT NULL,
    canonical_name text NOT NULL,
    aliases text[] DEFAULT '{}'::text[] NOT NULL,
    descriptor text,
    current_scene_id uuid,
    created_by_event uuid,
    status text DEFAULT 'active'::text NOT NULL,
    CONSTRAINT entity_registry_status_check CHECK ((status = ANY (ARRAY['active'::text, 'inactive'::text, 'merged'::text])))
);


--
-- Name: event_participant; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_participant (
    event_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    entity_kind text NOT NULL,
    role_qualifier text NOT NULL,
    CONSTRAINT event_participant_entity_kind_check CHECK ((entity_kind = ANY (ARRAY['actor'::text, 'location'::text, 'artifact'::text, 'faction'::text, 'group'::text])))
);


--
-- Name: location_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.location_state (
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: perception_subject; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.perception_subject (
    perception_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    world_id uuid NOT NULL
);


--
-- Name: provenance_edge; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.provenance_edge (
    derived_id uuid NOT NULL,
    derived_kind text NOT NULL,
    source_id uuid NOT NULL,
    source_kind text NOT NULL,
    how_type text NOT NULL,
    CONSTRAINT provenance_edge_derived_kind_check CHECK ((derived_kind = ANY (ARRAY['perception'::text, 'mutation'::text, 'event'::text, 'bundle'::text]))),
    CONSTRAINT provenance_edge_how_type_check CHECK ((how_type = ANY (ARRAY['derived_from'::text, 'inferred_from'::text, 'reported_by'::text, 'witnessed_by'::text, 'compensates'::text, 'supersedes'::text]))),
    CONSTRAINT provenance_edge_source_kind_check CHECK ((source_kind = ANY (ARRAY['perception'::text, 'mutation'::text, 'event'::text])))
);


--
-- Name: relationship_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.relationship_state (
    world_id uuid NOT NULL,
    a_id uuid NOT NULL,
    b_id uuid NOT NULL,
    attrs jsonb DEFAULT '{}'::jsonb NOT NULL,
    dirty boolean DEFAULT false NOT NULL,
    last_event_id uuid
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying(128) NOT NULL
);


--
-- Name: actor_state actor_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.actor_state
    ADD CONSTRAINT actor_state_pkey PRIMARY KEY (entity_id);


--
-- Name: artifact_state artifact_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artifact_state
    ADD CONSTRAINT artifact_state_pkey PRIMARY KEY (entity_id);


--
-- Name: canon_event canon_event_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canon_event
    ADD CONSTRAINT canon_event_pkey PRIMARY KEY (event_id);


--
-- Name: causal_bundle_input causal_bundle_input_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.causal_bundle_input
    ADD CONSTRAINT causal_bundle_input_pkey PRIMARY KEY (bundle_id, input_ref, role);


--
-- Name: causal_bundle causal_bundle_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.causal_bundle
    ADD CONSTRAINT causal_bundle_pkey PRIMARY KEY (bundle_id);


--
-- Name: entity_registry entity_registry_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_registry
    ADD CONSTRAINT entity_registry_pkey PRIMARY KEY (entity_id);


--
-- Name: event_participant event_participant_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_participant
    ADD CONSTRAINT event_participant_pkey PRIMARY KEY (event_id, entity_id, role_qualifier);


--
-- Name: location_state location_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.location_state
    ADD CONSTRAINT location_state_pkey PRIMARY KEY (entity_id);


--
-- Name: perception_record perception_record_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_record
    ADD CONSTRAINT perception_record_pkey PRIMARY KEY (perception_id);


--
-- Name: perception_subject perception_subject_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_subject
    ADD CONSTRAINT perception_subject_pkey PRIMARY KEY (perception_id, entity_id);


--
-- Name: provenance_edge provenance_edge_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.provenance_edge
    ADD CONSTRAINT provenance_edge_pkey PRIMARY KEY (derived_id, source_id, how_type);


--
-- Name: relationship_state relationship_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.relationship_state
    ADD CONSTRAINT relationship_state_pkey PRIMARY KEY (world_id, a_id, b_id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: state_mutation state_mutation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.state_mutation
    ADD CONSTRAINT state_mutation_pkey PRIMARY KEY (mutation_id);


--
-- Name: idx_cb_effect; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cb_effect ON public.causal_bundle USING btree (effect_ref);


--
-- Name: idx_ce_beat; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_beat ON public.canon_event USING btree (beat_id);


--
-- Name: idx_ce_payload_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_payload_gin ON public.canon_event USING gin (payload);


--
-- Name: idx_ce_scene; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_scene ON public.canon_event USING btree (scene_id);


--
-- Name: idx_ce_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_status ON public.canon_event USING btree (world_id, status) WHERE (status = 'accepted'::text);


--
-- Name: idx_ce_world_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ce_world_time ON public.canon_event USING btree (world_id, in_world_tick, beat_seq);


--
-- Name: idx_ep_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ep_entity ON public.event_participant USING btree (entity_id);


--
-- Name: idx_er_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_er_name ON public.entity_registry USING btree (world_id, canonical_name);


--
-- Name: idx_er_scene; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_er_scene ON public.entity_registry USING btree (world_id, current_scene_id);


--
-- Name: idx_pe_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pe_source ON public.provenance_edge USING btree (source_id);


--
-- Name: idx_pr_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_active ON public.perception_record USING btree (holder_id) WHERE ((invalid_tick IS NULL) AND (expired_at IS NULL));


--
-- Name: idx_pr_holder; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_holder ON public.perception_record USING btree (holder_id, acquired_tick);


--
-- Name: idx_pr_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pr_source ON public.perception_record USING btree (source_event_id);


--
-- Name: idx_ps_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ps_entity ON public.perception_subject USING btree (entity_id);


--
-- Name: idx_ps_world; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_ps_world ON public.perception_subject USING btree (world_id);


--
-- Name: idx_sm_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sm_entity ON public.state_mutation USING btree (entity_id, valid_from_tick, valid_from_seq);


--
-- Name: idx_sm_event; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_sm_event ON public.state_mutation USING btree (event_id);


--
-- Name: uq_ce_accepted_order; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uq_ce_accepted_order ON public.canon_event USING btree (world_id, in_world_tick, beat_seq) WHERE (status = 'accepted'::text);


--
-- Name: canon_event trg_canon_event_append_only; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_canon_event_append_only BEFORE UPDATE ON public.canon_event FOR EACH ROW EXECUTE FUNCTION public.canon_event_append_only();


--
-- Name: canon_event trg_canon_event_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_canon_event_no_delete BEFORE DELETE ON public.canon_event FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: causal_bundle trg_causal_bundle_append_only; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_append_only BEFORE UPDATE ON public.causal_bundle FOR EACH ROW EXECUTE FUNCTION public.causal_bundle_append_only();


--
-- Name: causal_bundle_input trg_causal_bundle_input_immutable; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_input_immutable BEFORE UPDATE ON public.causal_bundle_input FOR EACH ROW EXECUTE FUNCTION public.causal_bundle_input_immutable();


--
-- Name: causal_bundle_input trg_causal_bundle_input_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_input_no_delete BEFORE DELETE ON public.causal_bundle_input FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: causal_bundle trg_causal_bundle_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_causal_bundle_no_delete BEFORE DELETE ON public.causal_bundle FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: causal_bundle_input trg_cbi_assert_acyclic; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_cbi_assert_acyclic BEFORE INSERT ON public.causal_bundle_input FOR EACH ROW EXECUTE FUNCTION public.causal_bundle_assert_acyclic();


--
-- Name: event_participant trg_event_participant_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_event_participant_no_delete BEFORE DELETE ON public.event_participant FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: perception_record trg_perception_record_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_perception_record_no_delete BEFORE DELETE ON public.perception_record FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: perception_subject trg_perception_subject_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_perception_subject_no_delete BEFORE DELETE ON public.perception_subject FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: provenance_edge trg_provenance_edge_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_provenance_edge_no_delete BEFORE DELETE ON public.provenance_edge FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: state_mutation trg_sm_project; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_sm_project AFTER INSERT ON public.state_mutation FOR EACH ROW EXECUTE FUNCTION public.sm_project();


--
-- Name: state_mutation trg_state_mutation_no_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_state_mutation_no_delete BEFORE DELETE ON public.state_mutation FOR EACH ROW EXECUTE FUNCTION public.forbid_delete();


--
-- Name: canon_event canon_event_superseded_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canon_event
    ADD CONSTRAINT canon_event_superseded_by_fkey FOREIGN KEY (superseded_by) REFERENCES public.canon_event(event_id);


--
-- Name: causal_bundle_input causal_bundle_input_bundle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.causal_bundle_input
    ADD CONSTRAINT causal_bundle_input_bundle_id_fkey FOREIGN KEY (bundle_id) REFERENCES public.causal_bundle(bundle_id);


--
-- Name: entity_registry entity_registry_created_by_event_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_registry
    ADD CONSTRAINT entity_registry_created_by_event_fkey FOREIGN KEY (created_by_event) REFERENCES public.canon_event(event_id);


--
-- Name: event_participant event_participant_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_participant
    ADD CONSTRAINT event_participant_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.canon_event(event_id);


--
-- Name: perception_record perception_record_source_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_record
    ADD CONSTRAINT perception_record_source_event_id_fkey FOREIGN KEY (source_event_id) REFERENCES public.canon_event(event_id);


--
-- Name: perception_subject perception_subject_perception_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.perception_subject
    ADD CONSTRAINT perception_subject_perception_id_fkey FOREIGN KEY (perception_id) REFERENCES public.perception_record(perception_id);


--
-- Name: state_mutation state_mutation_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.state_mutation
    ADD CONSTRAINT state_mutation_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.canon_event(event_id);


--
-- PostgreSQL database dump complete
--


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20260610090001'),
    ('20260610090002'),
    ('20260610090003'),
    ('20260610090004'),
    ('20260610090005'),
    ('20260610090006'),
    ('20260610090007'),
    ('20260611090001'),
    ('20260614090001'),
    ('20260614090002'),
    ('20260615090001');
