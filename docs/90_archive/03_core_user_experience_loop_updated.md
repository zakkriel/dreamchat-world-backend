# 03 Core User Experience Loop
## 1. Core UX Principle
The product should feel like returning to an ongoing world, not opening a blank chatbot.
The default experience is play-first.
The user should be able to enter a scene, understand what is happening, act naturally, and continue the world without needing to manage systems manually.
The deeper world controls should exist, but they should not dominate the default experience.
The app should support two layers:
### Play-first layer
This is the default mode.
It focuses on:
the current scene
the current conversation
the entities or forces involved
what the user can naturally do next
recent context needed to continue
The user should feel:
I am back in the world.
Not:
I am managing a dashboard.
### World workspace layer
This is optional depth.
It gives access to:
timeline
known entities
relationships
locations
artifacts
known world context
correction tools
creator/debug tools where allowed
This layer is for users who want more control, more inspection, or more trust in the system.
The product should borrow from two UX directions:
JRPG / visual-novel readability: visual scene, present entities, speaker focus, atmosphere.
Cursor-like workspace control: inspectable context, state changes, correction, structured world tools.
The result should be:
A text-first world experience that is not text-only.
The interface should use visuals to reduce cognitive load, not to become a full game UI.
Visuals should help the user quickly understand:
where the scene is
who or what is present
who is speaking or acting
what artifact or context matters now
what has recently changed
what unresolved threads may still matter
The product should remain genre-agnostic.
The same UX should support fantasy, sci-fi, political fiction, horror, realistic drama, companion-like scenarios, or any other setting style.
The visual content may change by genre, but the interaction structure should remain stable.

## 2. Main Screen Zones
The main screen should support a play-first experience with optional workspace depth.
The default screen should feel like entering a readable scene, not managing a dashboard.
The proposed main zones are:
Main Scene Canvas
Scene Participants
Conversation / Narration Panel
Aux Context Sidebar
World Workspace Navigation
## 2.1 Main Scene Canvas
The Main Scene Canvas is the visual and emotional center of the experience.
It should represent the current diegetic scene: the place, moment, or viewpoint from which the user is experiencing the world.
The canvas should usually answer:
Where are we?
This is important for immersion.
The app should avoid turning the main canvas into an abstract system-state surface during normal play.
If the user is reading a document, the canvas should still show the place or situation where that reading happens.
If the user is looking at a map, the canvas may show the room, table, screen, device, vehicle, command center, or location where the map is being used.
If the story cuts away to an event where the user-controlled entity is not present, that cutaway should still be treated as a proper scene with its own place, viewpoint, and atmosphere.
The visual scene should communicate quickly:
where the moment is happening
what the tone is
whether the situation is calm, tense, intimate, public, dangerous, procedural, etc.
what kind of world the user has entered
what the current atmosphere feels like
The visual scene should reduce cognitive load, but it should not replace rich written description.
The user should still be able to read the full scene prose when desired.
The product may support:
compact visual scene by default
expandable scene description
narrator-written environmental detail
atmosphere, sound, body language, and sensory context
longer prose when the scene benefits from it
Product rule:
Visuals make the scene readable. Prose makes the scene deep.
## 2.2 Scene Participants
The Scene Participants area shows who is present, speaking, reacting, or directly involved in the scene.
This area should be limited to characters, NPCs, user-controlled entities, or other beings/entities capable of presence, dialogue, reaction, or agency.
It should not show generic objects, locations, factions, documents, maps, or abstract systems as participant avatars.
Allowed participant types include:
user-controlled entity, if present or directly involved
NPCs/entities physically present
NPCs/entities currently speaking
NPCs/entities silently present but relevant
off-screen speakers, if actively communicating through a valid in-world channel
narrator/facilitator, if represented as part of the experience
Examples:
A guard holding a warrant can appear as a participant.
The warrant itself should not.
A cartographer explaining a map can appear as a participant.
The map itself should not.
A faction leader speaking for an organization can appear as a participant.
The organization itself should not.
A person calling through a radio can appear as an off-screen participant.
The radio itself should not.
The participant area should help the user understand:
who is here
who is speaking now
who is listening
who is silent but present
who has entered or left
who the user may naturally address
The default behavior should be:
The world decides who responds.
The user sends a natural action or message, and the scene engine decides who speaks, who stays silent, who interrupts, who reacts, who leaves, or whether the narrator should answer.
The user may optionally target a participant.
Targeting can happen through natural language:
“I ask Mara what she knows.”
Or through UI:
selecting/clicking an avatar before sending.
Targeting should not guarantee obedience.
A participant may refuse, avoid the question, misunderstand, lack the knowledge, be interrupted, or be unable to respond.
This preserves world authority.
## 2.3 Conversation / Narration Panel
The Conversation / Narration Panel is the main interaction surface.
This is where the user acts through natural language and receives narration, dialogue, and world response.
It should support:
user actions
user dialogue
narrator responses
NPC/entity dialogue
environmental description
consequences
interruptions
multi-entity responses
scene transitions
cutaways
The product is text-first here.
The core interaction remains:
User acts → world responds → world state updates → user continues.
The UI should not force the user into fixed commands or game buttons.
Suggested actions may exist later, but they should help stuck users without replacing natural language.
## 2.4 Aux Context Sidebar
The Aux Context Sidebar provides supporting context for the current moment.
This is where non-participant context belongs.
It should not be a generic wall of text.
It should behave like a contextual surface that can show summaries, artifacts, or interactive objects.
Primary tabs may include:
Current
Previously
Open Threads
The content can be text, but it can also be richer objects.
Examples:
document card
map
item card
key
weapon
photo
letter
warrant
public notice
relationship card
location card
rumor card
timeline card
message thread
faction status card
The Aux Sidebar should answer:
What matters right now?
Not:
What is everything the system knows?
Examples:
If a guard presents a warrant, the guard appears in Scene Participants and the warrant appears in the Aux Sidebar.
If the user studies a map, the map appears in the Aux Sidebar while the Main Scene Canvas keeps the user grounded in the place where the map is being used.
If the user receives a key, the key appears as an artifact or item card in the Aux Sidebar.
If a faction is creating pressure in the scene, the faction may appear as a context card, but not as a participant unless represented by a specific agent.
## 2.5 World Workspace Navigation
The World Workspace Navigation is the optional deeper layer.
It gives access to structured world tools without forcing them into the default play flow.
Possible sections:
Timeline
Entities
Relationships
Locations
Artifacts
Known World
Corrections
Settings
Creator Tools, if allowed
This is the Cursor-like side of the product.
It allows users to inspect, understand, and correct the world when needed.
But it should remain secondary to the play experience.
The product should not make the user feel that they must manage the world in order to enjoy it.
## 2.6 Known World, Not Omniscient World State
The normal user-facing experience should not expose full omniscient world state.
In play mode, the app should show what is known, perceived, public, remembered, or available to the user-controlled perspective.
This can include:
directly observed facts
public facts
remembered events
received information
rumors clearly marked as rumors
uncertain information clearly marked as uncertain
artifacts the user has access to
The app should avoid showing hidden truth by default.
A separate creator/debug/admin layer may expose authoritative world state, hidden facts, backstage state, and system internals.
Product rule:
Play mode shows the known/perceived world. Creator or debug mode may show authoritative world state.

## 3. Interaction Granularity and Canon Clustering
The product should not copy the long-form response pattern of many current companion or roleplay products.
Those products often generate large response blocks of 10,000 to 15,000 characters, mixing dialogue, description, multiple speakers, scene movement, and consequences into one long answer.
This product should use a more interactive rhythm.
NPCs/entities and the user should have more opportunities to interact.
Narrator descriptions should appear in smaller, playable pieces that the user can respond to, interrupt, ignore, or continue through.
The product should therefore separate visible interaction from canonical world processing.
## 3.1 Interaction Unit Nomenclature
The agreed hierarchy is:
Message → Beat → Scene Segment → Scene
### Message
A Message is the smallest visible UI unit.
Examples:
one NPC line
one narrator sentence or paragraph
one small reaction
one user input
one short environmental description
Messages are presentation-level units.
A Message should not automatically become a full canonical world action.
### Beat
A Beat is a short exchange cluster.
Examples:
the user says something
one NPC responds
the narrator describes the reaction
another NPC reacts
the user can reply or press Continue
A Beat is the main interactive rhythm of the experience.
A Beat should usually contain a small number of messages, enough to feel alive without becoming a long wall of text.
A default Beat may contain around 1-6 visible messages, but the exact number should depend on the scene.
### Scene Segment
A Scene Segment is a larger chunk within a scene.
Examples:
the opening exchange
a negotiation at the table
walking through the city
inspecting a document
an argument escalating
a quiet aftermath
the group deciding what to do next
A Scene Segment can contain multiple Beats.
This is a useful unit for clustering memory and canon processing more meaningfully than individual messages.
### Scene
A Scene is the full diegetic situation.
Examples:
meeting at the harbor office
dinner with Mara
escape from the checkpoint
reading the warrant in the apartment
receiving a call in a parked car
watching a broadcast from a safehouse
A Scene can contain multiple Scene Segments.
## 3.2 Canon Clustering Rule
Visible interaction can be granular.
Canon commits should be clustered.
Product rule:
Do not make every visible message equal one canonical world action.
A Beat may produce zero, one, or several canonical changes.
A Scene Segment may produce a more meaningful grouped memory/canon update.
A Scene may produce a larger summary, timeline entry, and state update.
This prevents the world model from becoming noisy, expensive, and fragmented.
## 3.3 Canon Event / World Update
A Canon Event or World Update is a committed state change.
Examples:
an entity learned something
a relationship changed
the user-controlled entity received an item
a rumor started spreading
a participant left the scene
a promise was made
a location state changed
a conflict escalated
a belief was updated
an object changed ownership
These are world-state-level changes.
They should be derived from Messages, Beats, and Scene Segments, but they should not be one-to-one with them.
## 3.4 Continue Button
The Continue button should advance the current scene by one small Beat.
It should not trigger uncontrolled autoplay or simulate a large number of events.
When the user presses Continue, the world may produce:
one narrator description
one NPC/entity line
one reaction
one small environmental change
one small state change
no state change, if nothing meaningful happened
Then the scene should pause again for user input or another Continue.
Product rule:
Continue advances the current moment. It does not fast-forward the world.
## 3.5 Preventing Runaway Interaction
The product should avoid generating too many actions too quickly.
It should not create 500 meaningful actions in 40 seconds just because the user keeps pressing Continue.
Possible guardrails:
Continue advances by Beat, not by Scene or time jump.
Meaningful canon writes are clustered.
Long progression requires explicit user intent.
Time jumps require explicit or strongly implied in-world action.
The user should be able to interrupt the flow.
The system should pause before major consequences, transitions, or irreversible changes.
## 3.6 Text Area and Layout Implications
The central text panel should be larger than a normal chat bubble area, but smaller and more controlled than long-form RP walls.
Target feel:
Shorter than long-form RP, richer than messenger chat.
To reclaim vertical space:
place avatars beside text, not in a full separate row
avoid wasting a full line on speaker metadata when possible
slightly reduce text size only if readability remains strong
keep comfortable line height
place helper text inside the input field
use placeholder/helper text such as: “Write an action, speak, or type / for options...”
avoid a persistent helper row under the input field unless necessary
allow the conversation/narration area to show a full Beat without immediate scrolling
The design should support short messages, but it should not look like WhatsApp.
It needs enough space for prose, dialogue, and scene flow.
Product rule:
The interface should make interaction frequent without making the world feel shallow.


# 4. Core Play Loop and Response Rules
This section captures the product decisions from the spoken working session. It should be folded into the main Core User Experience Loop PRD as implementation-facing product requirements, while keeping the default experience play-first and genre-agnostic.
## 4.1 Core Play Loop
The user-facing loop should stay simple and playable:
1. Enter or resume the current situation.
2. Orient: understand where the user is, who or what is relevant, and what matters now.
3. Act, speak, inspect, or press Continue.
4. The system interprets the user intent and applies world pushback.
5. The world responds and returns a new playable situation.
6. The user may correct the current moment before continuing.
The user should not feel that they are managing a state machine. The system can use internal scene, beat, canon, memory, and backstage concepts, but the visible experience should feel like naturally continuing inside the world.
## 4.2 User Input as Ordered Intent Units
The system should not treat a complex user message as one flat command. It should decompose complex input into a small sequence of ordered intent units.
The intent unit model should stay abstract. The PRD should not define a long taxonomy of action types such as combat, romance, investigation, movement, item use, or dialogue. The product benefits from LLM abstraction instead of trying to code every world mechanic from scratch.
A minimal intent unit may contain:
• the relevant text or implied user intent
• an optional target
• an optional condition
• an optional expected effect
• an optional risk or uncertainty signal
Product rule: The system should understand the full user intent, but it should not guarantee that the full chain executes.
## 4.3 World Pushback Resolution
World Pushback Resolution prevents the user from chaining unlimited actions and forcing outcomes before the world can react.
The system should estimate how much of the ordered intent chain can plausibly happen before the world pushes back. Pushback can come from time, distance, resistance, social pressure, physical constraint, urgency, risk, competing agency, or any other contextually relevant force.
This should remain abstract. The product should not require a hardcoded list of possible pushback types. The LLM should reason from the current world context and the validator layer can later check plausibility, canon, tone, and genre consistency.
Product rule: The longer and more consequential the action chain, the more chances the world has to react, interrupt, resist, or create consequences.
Example: If the user tries to perform a long chain such as attacking someone, crossing a dangerous space, leaving the area, recovering resources, resting, and then returning, the system should not blindly execute the whole chain. It may resolve the early intent units, then let the world interrupt or push back once the chain exceeds what the current situation plausibly allows.
## 4.4 World Response
A world response should not be a long automatic continuation that consumes the whole situation. It should resolve the current beat and return the user to agency.
A world response should do three abstract things:
7. Resolve what just happened: how far the user intent got and what actually became true.
8. Reflect world continuation or pushback: how the world reacts according to current context, constraints, pressure, entities, timing, and situation.
9. Return a new playable situation: the user should understand the current moment and have a meaningful next point of agency.
The PRD should not define world response through genre-specific status lists. The same abstraction should work for adventure, politics, horror, romance, realistic drama, companion-like situations, workplace scenarios, and other modes.
Product rule: LLMs provide flexible scene reasoning. Validators protect continuity, genre, canon, plausibility, tone, and boundaries.
## 4.5 Narration and Aux Context Sidebar
Not everything needs to be pushed into narration. The narrator should highlight the most important immediate context, especially the information the user needs to understand the current moment.
The Aux Context Sidebar should support the user without turning the response into a memory dump or dashboard. It should answer:
What matters now?
It should not answer:
Everything the system knows.
Narration and the sidebar should work together: narration carries the playable moment; the sidebar provides relevant supporting context, artifacts, reminders, and known threads when they matter now.
## 4.6 Correction Window and Interpretation Confidence
The correction window exists because the system may misunderstand user intent or resolve the world in a way the user needs to repair.
The correction window should be local and limited. The user can correct the current or most recent interpretation/outcome before it is accepted into ongoing history. The product should not automatically rewind the entire world or trigger uncontrolled reverse butterfly effects.
Correction rules:
• Corrections should mostly affect the current moment.
• Corrections may affect one direct linked consequence if that consequence has not propagated too far.
• After the correction window, corrections should usually become present-forward updates.
• Deep historical rewrites should require explicit advanced handling, not automatic recalculation.
The system should also estimate confidence in its interpretation of user input. If confidence is low, it may highlight the interpretation before continuing.
This threshold should be configurable by the user. A low threshold asks only when the system is very uncertain. A high threshold allows users to validate most ambiguous interpretations.
Product rule: Interpretation confidence is a user-control setting, not a hardcoded product behavior.
## 4.7 Scene Transitions as Internal Mechanics
A scene is primarily an internal product, memory, and processing unit. The user should not need to know that the product has changed scenes. The experience should feel seamless, like continuing through the world, not managing chapters or scene cards.
User-driven transitions are just user intent. If the user tries to leave, escape, hang up, travel, sleep, walk away, or otherwise change the situation, the world resolves that intent like any other. It may succeed, fail, partially succeed, or be interrupted.
World-driven transitions are also not automatic scene endings. A world event can push the situation toward change, but the user can often still react if reaction is plausible.
Agency is meaningful but not infinite. The world should not keep asking forever just to preserve agency. If the user refuses to act, runs out of time, or chooses inaction under pressure, consequences can happen.
Hard world consequences are allowed when they are contextually earned. For example, if the user has repeatedly created pressure with a dangerous group, the world may eventually hit back hard, including a sudden forced situation change. This should be rare, justified by prior context, and not the default interaction style.
## 4.8 Contextual Surfacing of Known Information
The product should not show the user a perfect memory log, an omniscient world feed, or every open thread that exists in the world.
The UI should surface information only when it is relevant to the current context and when the user-controlled entity could plausibly know, remember, perceive, infer, or access it.
Contextual surfacing can be triggered by:
• current location
• current situation
• current participants
• current artifact or object being inspected
• current relationship context
• active conversation topic
• recent user intent
• in-world time pressure
• previous unresolved commitments connected to the current context
Visibility is driven by relevance, current context, and the user knowledge boundary.
The system must not reveal hidden world events just because they are important. If the user has no valid knowledge path to an event, the UI should not expose it. The event may later become knowable through valid channels such as rumor, witness, message, news, visible consequences, investigation, public announcement, or social propagation.
## 4.9 Decay Is Review Pressure, Not Visibility Logic
Decay should not mean “hide old information.”
Decay means the system has lower confidence that the last known state of something is still current. It creates pressure for the world to review or refresh a known state before using it.
Example: The system may know that John owned the gym last month. After enough in-world time passes, that fact may still be remembered, but its current validity may be uncertain. The product should treat it as: Last known: John owned the gym. Current status may need review.
Decay does not decide by itself whether something is shown to the user. Relevance, current context, and knowledge boundaries decide what can be surfaced. Decay informs whether the known state may need review before being used.
Product rule: The past should reappear when the present context makes it relevant and knowable. Decay tells the system whether the known state may need review.
## 4.10 Product Rules Summary
• World response resolves the current beat and returns playable agency.
• Do not hardcode endless genre, action, or status mechanics.
• Use LLM abstraction first; add validators later to protect boundaries.
• Narration highlights the critical now-context; the sidebar supports “what matters now.”
• The correction window is local and should not become an automatic world rollback system.
• Interpretation confidence threshold should be configurable by the user.
• Scene is an internal mechanic; user experience should stay seamless.
• User-driven transitions are intents, not automatic exits.
• World-driven transitions create pressure, not automatic cutscenes.
• Agency is meaningful but not infinite.
• Hard world consequences are allowed when contextually earned.
• Contextual surfacing depends on relevance, current context, and knowledge boundary.
• Decay means review pressure/current-state confidence, not visibility.
• Unknown world events must not be surfaced unless the user has a valid knowledge path.