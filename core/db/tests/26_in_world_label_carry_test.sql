BEGIN;
SELECT plan(4);

-- The in-world label carry-forward trigger (B-5 / ADR-030). Every commit path inserts canon_event
-- without an in_world_label, so before this the first committed beat ended the only in-world time the
-- player ever had: scene/current's now.display_label and the Compendium's group_label both went null
-- and stayed null. The rule is CONTINUITY, never invention — an unlabelled event inherits the last
-- label the world actually authored, and nothing can conjure a label the world never wrote.

-- A private world so the assertions cannot be perturbed by seed ordering.
INSERT INTO entity_registry (entity_id, world_id, entity_kind, canonical_name) VALUES
  ('7a5e0000-0000-0000-0000-00000000000a','7a5e0000-0000-0000-0000-0000000000ff','actor','Carrier');

-- (1) No labelled event anywhere in this world yet → nothing to carry, and the trigger must NOT
--     invent one. Silence is the honest answer.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('7a5e0000-0000-0000-0000-0000000000e1','7a5e0000-0000-0000-0000-0000000000ff',
        'AttributeChanged','before any label',10,0,'accepted',now(),'public','fast_path');
SELECT is(
  (SELECT in_world_label FROM canon_event WHERE event_id='7a5e0000-0000-0000-0000-0000000000e1'),
  NULL,
  'no authored label yet: the trigger carries nothing and invents nothing');

-- (2) An EXPLICIT label always wins — the seed and any future authoring path keep full control.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('7a5e0000-0000-0000-0000-0000000000e2','7a5e0000-0000-0000-0000-0000000000ff',
        'AttributeChanged','the label is authored here',20,0,
        'Day 1','accepted',now(),'public','fast_path');
SELECT is(
  (SELECT in_world_label FROM canon_event WHERE event_id='7a5e0000-0000-0000-0000-0000000000e2'),
  'Day 1',
  'an explicitly authored label is never overwritten');

-- (3) THE FIX: a later unlabelled commit inherits it. "Day 1" stands until something authors "Day 2".
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('7a5e0000-0000-0000-0000-0000000000e3','7a5e0000-0000-0000-0000-0000000000ff',
        'Communicated','a committed beat with no label of its own',30,0,
        'accepted',now(),'public','fast_path');
SELECT is(
  (SELECT in_world_label FROM canon_event WHERE event_id='7a5e0000-0000-0000-0000-0000000000e3'),
  'Day 1',
  'an unlabelled commit inherits the last authored label — the player keeps a time anchor');

-- (4) The carry follows the NEWEST authored label, not the first one ever written.
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         in_world_label, status, accepted_at, visibility_scope, origin)
VALUES ('7a5e0000-0000-0000-0000-0000000000e4','7a5e0000-0000-0000-0000-0000000000ff',
        'AttributeChanged','time moves on',40,0,
        'Day 2','accepted',now(),'public','fast_path');
INSERT INTO canon_event (event_id, world_id, event_type, summary, in_world_tick, beat_seq,
                         status, accepted_at, visibility_scope, origin)
VALUES ('7a5e0000-0000-0000-0000-0000000000e5','7a5e0000-0000-0000-0000-0000000000ff',
        'Communicated','after the world moved on',50,0,
        'accepted',now(),'public','fast_path');
SELECT is(
  (SELECT in_world_label FROM canon_event WHERE event_id='7a5e0000-0000-0000-0000-0000000000e5'),
  'Day 2',
  'the carry tracks the most recent authored label, so a new day actually starts');

SELECT * FROM finish();
ROLLBACK;
