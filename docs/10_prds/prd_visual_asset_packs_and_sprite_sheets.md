# PRD 08 — Visual Asset Packs and NPC Sprite Sheets

## 1. Purpose

This PRD extends the DreamChat Image Platform to support the visual-novel / JRPG-inspired UX direction.

The product should not generate only one image for a place or one portrait for an NPC. It should create reusable visual ranges that can be selected during play without waiting for runtime generation.

This PRD focuses on:

- location scene variant packs
- NPC portrait and expression packs
- NPC expression sprite sheets
- artifact/item visuals for the Aux Context Sidebar and inventory
- runtime visual selection
- the distinction between UI inspection and world actions

## 2. Product Principle

DreamChat is text-first, but not text-only.

Visuals should make the world easier to enter and understand, but they should not become the source of truth.

Product rule:

> Generate visual range early. Reuse it during play.

## 3. Why This Matters

The chosen UX direction uses:

- scene-specific background art
- circular NPC/entity avatars
- active speaker highlighting
- visual artifact cards
- an Aux Context Sidebar
- inventory/artifact inspection

This means the image platform must support fast, reusable, coherent visual assets.

If the system generates images only on demand during play, the experience becomes slow, expensive, and visually inconsistent.

## 4. Location Scene Variant Packs

### Requirement

When a meaningful location is created or promoted, the image platform should generate a reusable location scene pack.

A location should not be represented by one static image only.

### Minimum PoC Pack

For PoC, a meaningful location should support:

1. `base_day_or_default`
2. `alternate_time_or_mood`

Example:

- market square, day, busy
- market square, night, quiet

### V1 Pack

For V1, a meaningful location should support a small variant set such as:

- day
- night
- crowded / active
- empty / quiet
- tense / dangerous
- calm / normal
- weather variant, if relevant
- damaged / changed state, if canonically justified

### Scene-State Matching

The DreamChat core app should pass scene-state hints to the Image Platform, such as:

- time of day
- weather
- crowd level
- danger/tension level
- public/private mood
- damaged/rebuilt/occupied state
- active faction/structure visual pressure, if visible

The Image Platform should return the closest matching existing asset before generating a new one.

### Acceptance Criteria

- A recurring location can appear in at least two different visual states.
- Returning to a location preserves visual identity.
- The system prefers existing assets before generating new ones.
- Visual changes reflect world state instead of random style drift.

## 5. NPC Portrait and Expression Packs

### Requirement

When a meaningful NPC/entity is created or promoted, the image platform should generate a reusable NPC visual pack.

A meaningful NPC should not have only one static portrait.

### Minimum PoC Pack

For PoC, a meaningful NPC should support:

1. base neutral portrait
2. suspicious / tense expression
3. warm / positive expression
4. angry / defensive expression

### V1 Expression Set

For V1, use a broader default expression taxonomy:

1. neutral
2. warm / happy
3. amused
4. suspicious
5. angry
6. afraid
7. sad
8. surprised
9. focused
10. exhausted / distressed

The exact expression set can be adapted by genre, maturity settings, and character type.

### Runtime Use

During play, the UI should be able to swap the NPC avatar expression instantly based on:

- current emotional state
- dialogue intent
- relationship stance
- trust/fear/anger values
- stress level
- recent user action
- scene tension
- narrator metadata

### Acceptance Criteria

- A meaningful NPC can visually react during dialogue without runtime generation.
- The same NPC remains recognizable across expressions.
- The UI can display the active speaker with the appropriate expression.
- If a requested expression is unavailable, the system falls back cleanly.

## 6. NPC Expression Sprite Sheet Optimization

### Requirement

The Image Platform should support generating a single NPC expression sprite sheet and slicing it into individual expression assets.

Instead of generating ten separate images, the service can generate one larger image containing a grid of ten expressions.

### Purpose

This reduces:

- generation calls
- cost
- latency
- prompt/token overhead
- visual drift between expressions

It also improves character consistency because all expressions are generated as one coherent visual set.

### Sprite Sheet Contract

The sprite sheet must follow a fixed layout contract.

Recommended V1 layout:

- 2 rows
- 5 columns
- 10 cells total
- same character identity
- same outfit unless intentionally varied
- same camera angle and crop
- same style and lighting family
- no text labels inside cells
- no decorative frames inside cells
- safe margin inside each cell
- no overlapping characters

### Default Cell Order

1. neutral
2. warm
3. amused
4. suspicious
5. angry
6. afraid
7. sad
8. surprised
9. focused
10. exhausted

### Output

A sprite sheet job should store:

- original sprite sheet asset
- sliced expression assets
- expression metadata for each slice
- crop coordinates
- visual identity version
- generation job reference
- quality status
- fallback expression mapping

### Acceptance Criteria

- The system can generate one 2x5 sprite sheet.
- The system can slice it into 10 expression assets deterministically.
- Each sliced asset is addressable and reusable like a normal visual asset.
- The sliced assets preserve the NPC identity closely enough for UI use.

## 7. Artifact and Item Visuals

### Requirement

Important artifacts/items should be generated as visual assets when they matter to the current scene, inventory, or Aux Context Sidebar.

Examples:

- red gem necklace
- invitation letter
- noble family sigil
- warrant
- map
- key
- potion
- weapon
- relic
- public notice

### UX Rule

Artifacts are context objects, not scene participants.

A merchant can appear as an avatar.

A necklace, warrant, or letter should appear in the Aux Context Sidebar, inventory, or artifact inspector.

### Acceptance Criteria

- Important artifacts can be visually inspected.
- Artifact visuals do not appear as participant avatars unless the artifact is sentient or agentic.
- The right sidebar can focus on a selected artifact without forcing a world action.

## 8. UI Inspection vs World Action

### Requirement

The platform must support the distinction between UI interaction and world/canon action.

Examples:

- opening inventory is a UI action
- clicking a trinket in the sidebar is usually a UI inspection
- casting Detect Magic on the trinket is a world action
- buying the trinket is a world action
- pinning the trinket to Current context is a UI action unless the world state changes

### Purpose

This prevents visual browsing from polluting canon and memory.

The app should not treat every click as a meaningful world event.

### Product Rule

> UI inspection helps the user understand the world. World actions change the world.

### Acceptance Criteria

- The app can display item details without writing canon events.
- The app can distinguish inspection from action.
- The app records canon only when the user or world actually changes something.

## 9. Runtime Generation Policy

### Requirement

Normal play should not wait on image generation unless the visual is essential to the next moment.

The image platform should prefer:

1. exact existing asset match
2. closest existing variant
3. fallback base asset
4. background generation request
5. blocking generation only when necessary

### Blocking Generation Allowed When

- a new meaningful NPC is introduced and must be visible now
- a new meaningful place is entered and no visual exists
- the user explicitly requests the image before continuing
- the scene depends on an artifact visual to understand the moment

### Acceptance Criteria

- Common interactions do not block on image generation.
- Runtime generation is visible as a controlled job, not random latency.
- The product can continue with a fallback image when appropriate.

## 10. Scope Priority

### PoC

PoC should validate:

- base location image + one variant for key locations
- base NPC portrait + three expressions for key NPCs
- one manual or semi-automated sprite sheet slicing test
- artifact image display in the Aux Context Sidebar
- UI inspection vs world action distinction

### V1

V1 should include:

- automated sprite sheet generation and slicing
- expression metadata and fallback mapping
- location scene-state matching
- asset retrieval-before-generation
- artifact visual cards
- runtime expression selection

### Later

Later versions may include:

- user-controlled regeneration
- creator asset editing
- LoRA/fine-tuned character consistency
- animation
- voice/image sync
- video
- marketplace asset packs

## 11. Alternatives Considered

### Static single image per NPC/place

Good for early PoC, but weak for expressive visual-novel UX.

### Separate image per expression

Works but costs more and may create visual drift.

### On-demand runtime image generation

Flexible but slow and expensive. Not recommended for normal play.

### Sprite sheet generation and slicing

Preferred for meaningful NPC expression packs.

### Text-only descriptions

Still necessary, but not enough for the chosen visual UX direction.

## 12. Metrics and Validation

Track:

- generation cost per visual identity
- number of assets generated per NPC/place
- sprite sheet slice success rate
- expression match fallback rate
- visual drift rejection rate
- cache/reuse hit rate
- blocking generation rate during play
- time-to-first-visual for new NPC/place
- user correction/regeneration rate

Success means:

- meaningful NPCs are visually expressive without runtime generation
- recurring locations remain recognizable across variants
- artifact visuals improve interaction without turning every click into canon
- runtime play remains fast
- image costs remain predictable
