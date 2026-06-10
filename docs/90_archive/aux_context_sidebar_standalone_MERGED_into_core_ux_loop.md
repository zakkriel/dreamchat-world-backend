## Aux Context Sidebar — Design and Feature Definition

### Purpose

The Aux Context Sidebar is a contextual support surface for the current experience.

It is not a memory log. It is not a quest tracker. It is not an omniscient world-state panel. It is not a dashboard the user must manage to enjoy the world.

Its job is to support the current moment by showing useful context, inspectable details, system interpretation, and known information without overloading narration.

The Aux Sidebar should help the user understand:

- what matters now

- what they are looking at

- how the system understood their intent

- what the user-controlled perspective currently knows or can access

The Aux Sidebar should remain story/theme agnostic. The same structure should work for fantasy, modern drama, sci-fi, horror, political fiction, companion-like experiences, detective stories, workplace drama, or any other world style.

### Product Rule

Aux follows user attention, not database structure.

The sidebar should adapt based on what the user is doing, selecting, asking about, or trying to understand.

The MVP lenses are:

- Current

- Inspect

- Intent

- Known

These are lenses, not rigid content buckets.

## Design Principles

### 1. Fewer boxes, more flow

The Aux Sidebar should avoid over-clustering information into many hard cards.

Use visual separation sparingly:

- soft dividers

- spacing

- subtle section headings

- one primary content flow

- minimal icons

- restrained emphasis

The UI should feel like an elegant contextual note surface, not an inventory spreadsheet.

### 2. Story-driven, not system-driven

Content should be written in the language of the current world and current situation.

Avoid fixed taxonomies unless necessary.

For example, do not force generic fixed sections like:

- Rumors

- Combat State

- Quest Status

- Faction Pressure

- Relationship Status

Instead, the sidebar should adapt its language to the selected context.

A political thriller may show “What your source implied.” A romance scene may show “What still hangs between you.” A sci-fi investigation may show “What the scan reveals.” A fantasy market may show “What the stall hides in plain sight.”

### 3. Theme-agnostic interaction

The system can show objects, people, places, relationships, documents, messages, memories, or inferred context, but the UX should not assume a specific genre.

Do not design around fixed fantasy concepts like quests, potions, wounds, mana, relics, guilds, or combat unless the current world actually contains them.

### 4. Known/perceived boundary

The Aux Sidebar must respect what the user-controlled entity could plausibly know, perceive, remember, infer, or access.

It must not reveal hidden world truth in normal play.

Unknown world facts can only appear if they enter the user’s knowledge field through a valid path, such as observation, conversation, records, public information, message, investigation, memory, or social propagation.

### 5. Decay is not visibility

Decay should not decide whether something is hidden or shown.

Decay means the system has lower confidence that a previously known state is still current.

If relevant, the UI may express uncertainty using language such as:

- “Last known…”

- “You have not confirmed this recently.”

- “This may no longer be accurate.”

- “This is remembered, not verified.”

Visibility is driven by relevance, current context, user attention, and knowledge boundary.

# Lens 1: Current

## Purpose

Current answers:

What matters right now?

It is the default Aux lens.

It should summarize the present situation without duplicating the narration. It gives the user enough context to orient, decide, and continue.

Current may include:

- current location or situation description

- present atmosphere or tone

- visible or immediately relevant details

- nearby interactable elements

- current participants’ relevant posture or behavior

- immediate context not fully described in narration

- active reminders tied to the current situation

- what the user may naturally notice now

Current should not show every open thread or every remembered fact.

## Design

Current should feel like a living situational note.

Recommended structure:

- short title

- compact contextual description

- optional environmental/situational detail

- “what matters now” flow

- relevant interactive details, if present

Avoid making it a generic list of tasks.

## Examples

### Example A — Fantasy market

**Dawnfall Market**

A crowded trade hub where information moves almost as fast as coin. The morning crowd is loud, bright, and difficult to read.

What matters now:

- Seren is watching you more closely than the other merchants are.

- Kael has stayed within reach, but he has not interrupted.

- Liora’s stall is busy enough that a quiet question might go unnoticed.

### Example B — Modern workplace drama

**Conference Room 4B**

The quarterly planning meeting has stalled. Everyone is pretending to review the numbers, but the disagreement is no longer about the spreadsheet.

What matters now:

- Marta has stopped taking notes.

- Jonas keeps returning to the hiring plan.

- The CFO has not spoken since you challenged the forecast.

### Example C — Sci-fi station

**Docking Ring C**

The station is running on emergency lighting. The public announcement system keeps repeating the same evacuation instruction, but nobody nearby seems to be moving.

What matters now:

- The sealed hatch to Bay C-12 is still warm.

- Your access card worked once, then failed.

- The maintenance drone is waiting for a command authorization.

# Lens 2: Inspect

## Purpose

Inspect answers:

What am I looking at?

Inspect activates when the user selects, clicks, mentions, opens, studies, or focuses on something.

It is used for deeper detail that should not overload narration.

Inspect may show:

- object detail

- environmental detail

- item/artifact detail

- document/message/map detail

- person/entity detail

- relationship detail

- location detail

- visible clues

- available interaction suggestions

- sensory or contextual observations

Inspect should not become a fixed inventory/object schema. It should describe what matters about the thing in the current world.

## Design

Inspect should feel like zooming attention.

The layout should be more prose-led than box-led.

Recommended structure:

- selected subject

- contextual description

- what the user notices

- what may be interacted with

- optional “you could…” suggestions

The “you could…” section should not replace free input. It is only a lightweight affordance.

## Examples

### Example A — Fantasy object

**Red Gem Necklace**

A deep red gem, cut with care and set in warm gold. It feels pleasantly heavy in your palm, warmed slightly as if it has been close to skin.

What you notice:

- The gem catches light from within, not only on its surface.

- The chain has fine hairline wear near the clasp.

- A maker’s mark is hidden beneath the setting.

You could:

- Look closer at the mark.

- Ask whether anyone recognizes the craftsmanship.

- Keep it concealed and continue the conversation.

### Example B — Modern text message

**Unread Message from Elena**

The message is short, but unusually formal for her.

What you notice:

- She avoids your name.

- She asks to meet “somewhere neutral.”

- She sends the address of a café she normally dislikes.

You could:

- Reply directly.

- Check when she last contacted you.

- Ask someone nearby if they know the café.

### Example C — Sci-fi terminal

**Maintenance Terminal**

The screen is cracked, but still responsive. The login prompt has been bypassed, leaving an unfinished diagnostics panel open.

What you notice:

- The last command was interrupted mid-run.

- The system flags a pressure anomaly near Docking Ring C.

- Someone manually disabled automatic alerts.

You could:

- Resume the diagnostic.

- Check the access history.

- Disconnect before the system notices the session.

# Lens 3: Intent

## Purpose

Intent answers:

How did the system understand what I am trying to do?

Intent is used when the user writes a complex input, when interpretation confidence is low, or when the user chooses to inspect/correct the system’s reading.

Intent should help the user trust the interaction without turning play into debugging.

It may show:

- interpreted goal

- ordered intent flow

- conditions detected

- targets or referenced entities

- dependencies

- uncertain interpretations

- confidence level

- editable clarifications

Intent should not use a large predefined node taxonomy.

It should show the minimum useful interpretation of the user’s message.

## Design

Intent should be calm and readable.

Avoid excessive icons, colored status labels, combat-specific terms, or rigid flowchart visuals.

Use simple ordered prose.

Each interpreted unit should be editable or correctable.

The system should say, effectively:

Here is what I think you mean. Adjust anything I got wrong.

## Examples

### Example A — Fantasy action

User input:

“If Ravene is badly hurt, I use the last potion on her. Otherwise I throw the spear at the guard and tell Gori to flank.”

Intent interpretation:

- First, check Ravene’s condition.

- If she appears in immediate danger, use the last potion on her.

- If she is not in immediate danger, attack the guard from range.

- Also call to Gori and try to coordinate his movement.

Confidence: Medium.

Possible ambiguity:

- “Flank” is interpreted as a tactical instruction to Gori, not as your own movement.

### Example B — Companion-like emotional scene

User input:

“I don’t answer right away. I look at him and try to understand if he’s actually sorry or just saying what I want to hear.”

Intent interpretation:

- Stay silent for a moment.

- Study his emotional response.

- Try to judge whether the apology feels sincere.

- Do not accept or reject the apology yet.

Confidence: High.

Possible ambiguity:

- Silence may be perceived by him as hesitation or emotional distance.

### Example C — Modern investigation

User input:

“I ask the receptionist about the missing visitor log, but if she gets nervous I back off and check the cameras instead.”

Intent interpretation:

- Ask the receptionist about the missing visitor log.

- Watch her reaction.

- If she seems uncomfortable or defensive, stop pressing.

- Look for access to camera records instead.

Confidence: Medium.

Possible ambiguity:

- “Back off” is interpreted as reducing pressure, not leaving the building.

# Lens 4: Known

## Purpose

Known answers:

What do I currently know, remember, infer, or have access to about this?

Known is not omniscient truth.

It shows knowledge from the user-controlled perspective.

Known may include:

- directly observed facts

- remembered interactions

- public information

- received information

- accessible records

- indirect information

- uncertain information

- last-known state

- reasonable inferences

- relationship or history context

Known must preserve uncertainty.

If something is indirect, partial, old, biased, or unverified, the UI should say so.

Known should not have fixed sections like “Rumors” unless the current world/context makes that the natural form of knowledge.

## Design

Known should feel like a memory and knowledge lens, not a database inspector.

It should be written as contextual knowledge paragraphs or short entries, with light labels only when useful:

- Known

- Remembered

- Last known

- Unconfirmed

- Inferred

- Publicly known

- Directly observed

- Told by someone

- Accessible record

These labels are optional language tools, not fixed UI requirements.

## Examples

### Example A — Fantasy NPC

**Seren**

Known:

Seren works around Dawnfall Market and seems to trade in information more than goods. She noticed you quickly, but did not approach until she had watched you for a while.

Remembered:

Kael warned you that people who “know too much too cheaply” are dangerous in this district.

Unconfirmed:

Someone near the eastern stalls mentioned that Seren has contacts outside the market. You do not know whether that is true.

Inferred:

She may already be testing whether you are worth helping, selling out, or ignoring.

### Example B — Modern relationship

**Adrian**

Known:

Adrian has avoided direct conflict before, but he usually answers messages quickly. Today he has been slower and more careful with his wording.

Remembered:

The last serious argument ended when he changed the topic rather than answering the question.

Last known:

He said he was going to stay at his sister’s apartment this week. You have not confirmed whether he actually did.

Inferred:

He may be trying to keep the conversation controlled rather than open.

### Example C — Sci-fi organization

**Helix Transit Authority**

Publicly known:

Helix controls most docking permissions across the inner ring and publishes strict movement schedules.

Known from records:

Your temporary clearance was approved yesterday, but only for civilian maintenance corridors.

Unconfirmed:

The station workers seem to believe Helix is hiding the real reason for the lockdown.

Last known:

The authority’s local director was still on-station two days ago. That may no longer be current.

## AUX Interaction Rules

### 1. Auto-switching

The sidebar may automatically switch lenses based on user attention.

Examples:

- User enters or resumes a situation → Current.

- User selects an object or document → Inspect.

- User writes a complex action → Intent.

- User asks “what do I know about this?” → Known.

Auto-switching should be helpful, not aggressive.

The user should be able to manually switch lenses.

### 2. No forced fixed sections

The system should avoid forcing the same blocks into every lens.

For example, Known does not always need:

- Facts

- Rumors

- Last-known

- Inferred

Those are possible expressions, not mandatory categories.

The structure should emerge from the story, theme, and available knowledge.

### 3. Minimal icon use

Icons should support scanning, not define the system.

Avoid one icon per status or mechanic. Avoid making the sidebar look like a stat dashboard. Use icons only for major section identity or important affordances.

### 4. Narration remains primary for urgency

The AUX should not include a dedicated pressure meter or urgency score in MVP.

Urgency, tension, and world pushback should mostly be carried by narration.

The sidebar may reflect current context, but it should not turn world pressure into a visible numeric system.

### 5. Corrections and confidence

If the Intent lens shows low or medium confidence, the user should be able to clarify the interpretation before the world continues.

The confidence threshold should be configurable in user settings.

### 6. Knowledge boundary enforcement

The AUX must never reveal hidden truth by default.

If information is not knowable from the user-controlled perspective, it should not appear in Current, Inspect, Intent, or Known.

Creator/debug modes may expose more, but normal play should preserve the known/perceived world boundary.

## MVP Feature Set

The MVP Aux Sidebar should include:

- Four lenses: Current, Inspect, Intent, Known.

- Manual lens switching.

- Auto-switching based on user attention.

- Current lens for immediate situation support.

- Inspect lens for selected/focused details.

- Intent lens for interpreted user input and confidence.

- Known lens for user-perspective knowledge.

- Knowledge-boundary filtering across all lenses.

- Story/theme-agnostic content generation.

- Minimal icons and low box density.

## MVP Product Rule

The Aux Sidebar should make the world easier to understand without making the user feel like they are operating the world.

It supports immersion. It does not replace narration. It does not expose omniscient truth. It follows attention. It stays genre-agnostic.