# 00 Parked Product Concepts

This file collects strong product concepts that should not be forced into the current PRD too early, but should remain available for later UX and product design.

These are not technical requirements. They are product-level concepts to revisit when the relevant PRD appears.

---

## 1. Social Memory Propagation

### Status

Parked for later Backstaging / Memory / World Continuity PRDs.

### Concept

Social Memory Propagation is the product mechanic where entities gain knowledge indirectly through other entities or information sources.

An entity should not only know what it directly experienced. It may also learn about events through the world around it.

### Example

- Action 1: Entity 1 and Entity 2 are both with the user.

- Action 50: Entity 1 leaves.

- Action 100: Entity 2 leaves and joins Entity 1.

- Action 300: The user meets Entity 1 and Entity 2 again.

In this case, Entity 1 may know about events from Action 50 to Action 100 if Entity 2 shared them.

But Entity 1 should not treat that knowledge as direct memory.

It is secondhand knowledge.

### Possible Information Sources

Social Memory Propagation can happen through:

- another entity

- gossip

- rumors

- public records

- warrants

- newspapers

- bard tales

- social media

- letters

- broadcasts

- faction reports

- institutional records

- surveillance

- local reputation

- cultural memory

- propaganda

- private testimony

### Product Rule

Indirect knowledge is not direct memory.

When an entity learns something indirectly, the product should preserve:

- source

- reliability

- trust level

- bias

- distortion

- omissions

- uncertainty

- whether the entity believes it

- whether the entity acts on it

### Why It Matters

This prevents the world from feeling frozen or isolated.

It also prevents entities from becoming magically omniscient.

The world can spread information, but information should travel with perspective, limits, and possible distortion.

### Later PRDs Where This Belongs

- Backstage Updates UX

- Memory and Continuity UX

- Entity/NPC Knowledge Boundaries

- Relationships and Social Context

- World State and Public Information

- Timeline and History UX

---

## 2. World Relevance Score

### Status

Parked for later Backstaging / Narrator / World Simulation PRDs.

### Concept

`world_relevance_score` expresses how much the world currently considers, reacts to, or orbits the user-controlled character/entity.

A low-relevance world evolves more independently.

A high-relevance world pulls more events, entities, consequences, and social attention toward the user-controlled character/entity.

### Product Use

This could influence:

- backstage updates

- narrator framing

- entity reactions

- social propagation

- public reputation

- unresolved threads

- how much world movement references the user-controlled character/entity

### Product Rule

The user is not automatically the center of the world.

They may become central through actions, relationships, power, reputation, selected experience style, or accumulated consequences.

### Later PRDs Where This Belongs

- Backstage Updates UX

- Narrator Authority and Difficulty UX

- Player/User-Controlled Entity Positioning

- World State and Continuity

- Relationship and Reputation Systems

---

## 3. Structural Depth Model

### Status

Parked for later World State / Backstaging / Power Structures PRDs.

### Concept

The product should distinguish between different depths of world structures.

A structure can be any group, institution, network, collective force, or organized system that affects the world.

Examples include governments, local offices, companies, guilds, families, criminal groups, schools, religious orders, laboratories, rebel cells, social movements, media networks, armies, or cultural institutions.

### Three Levels

#### Active Structures

Active Structures are directly simulated in the current world scope.

They have named entities, relationships, goals, conflicts, memory, and backstage updates.

#### Background Pressure

Background Pressure represents larger forces that influence the current world without being fully simulated yet.

They can create pressure through laws, rumors, prices, fear, opportunities, propaganda, social expectations, institutional consequences, or public mood.

#### Lore-Only Structures

Lore-Only Structures exist as context, history, or world flavor.

They do not actively affect the current world unless promoted later.

### Product Rule

Not every structure should be simulated at the same depth all the time.

The product should decide what is active now, what creates pressure, what remains lore, and what should be promoted because the user or world touched it.

### Promotion Rule

A structure can move from lore-only to background pressure, or from background pressure to active structure, when it becomes relevant.

This is not only a PoC shortcut. It is a full-product scaling principle.

### Why It Matters

This lets the product support large worlds without pretending every country, institution, family, market, faction, or social system is equally active at all times.

It also keeps the PoC contained while preserving a path toward larger world simulation later.

### Later PRDs Where This Belongs

- World State UX

- Backstage Updates UX

- Factions / Groups / Institutions UX

- Timeline and History UX

- World Creation UX

- Narrator Authority UX

- Relationship and Reputation Systems

---

## 4. Correction Propagation and Propagation Density

### Status

Parked for later Memory / Canon / Timeline Rewrite PRDs.

### Concept

Correction Propagation is the advanced feature where a correction to a past event can affect downstream consequences.

For the PoC, corrections should be limited to the current correction window or present-forward canon updates.

For the full product, deeper correction propagation may become useful, but it must be controlled carefully.

### Problem

A past event may have already created consequences.

For example:

- an entity was believed to have died

- another entity grieved

- another entity looked for revenge

- another entity took over a role

- a group changed strategy

- a location changed because the entity was gone

If the user later corrects that the entity did not die, the product should not blindly rewrite everything.

That can create reverse butterfly-effect collapse.

### Propagation Density

The product could estimate how far a correction would spread before allowing it.

#### Density 0

The correction changes no downstream consequences.

Safe to apply.

#### Density 1

The correction affects one direct consequence.

May be allowed with confirmation.

#### Density 2+

The correction affects multiple dependent consequences or a chain of consequences.

Should usually be blocked, converted into a present-forward correction, or offered as an advanced timeline fork/rewrite.

### Product Rule

The product should protect the world from uncontrolled reverse causality.

The user can correct the current moment easily.

Deep historical correction should require explicit handling, not automatic rewriting.

### Later PRDs Where This Belongs

- Memory and Canon Correction UX

- Timeline and History UX

- World State UX

- Advanced Creator Controls

- Timeline Forking / Alternate Continuity UX

---

## 5. Entity Competence and Action Boundaries

### Status

Parked for later Entity/NPC Design / Narrator / Safety-Aware Roleplay PRDs.

### Concept

Entities should be limited and enabled by what they plausibly know, believe, value, and are capable of doing.

An entity should not automatically be able to discuss, explain, perform, or support any topic just because the user brings it up.

Knowledge, skill, personality, social role, emotional stance, morality, risk tolerance, and setting context should shape what an entity can say or do.

### Product Rule

An entity's response should be constrained by its competence and character.

If a topic is outside the entity's knowledge or comfort zone, the entity should respond from that limitation.

It may:

- admit it does not know

- misunderstand the topic

- become worried, angry, disgusted, curious, or suspicious

- refuse to help

- redirect the conversation

- ask why the user is discussing it

- escalate to another entity or authority if appropriate

### Why It Matters

This makes entities feel real and prevents every entity from sounding like a universal expert.

It also creates better roleplay because limitations become part of characterization.

A gentle domestic character, a trained chemist, a soldier, a journalist, a doctor, a politician, and a child should not all respond to dangerous or specialized topics in the same way.

### Later PRDs Where This Belongs

- Entity/NPC Generation UX

- Entity/NPC Knowledge Boundaries

- Entity/NPC Skills and Capabilities

- Narrator Authority UX

- Safety-Aware Mature Content UX

- Relationship and Trust Systems