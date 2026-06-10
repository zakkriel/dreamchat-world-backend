# PRD — Core Private/Public Content Governance

## 1. Purpose

This PRD defines the core product and platform requirements for private-world freedom and public-distribution governance in DreamChat.

This PRD belongs to the DreamChat core platform, not the Image Platform.

The Image Platform should not decide whether content is legally/platform eligible. The core platform must classify and authorize content before requesting image generation or public distribution.

## 2. Product Context

DreamChat is a persistent AI RPG world product. Users can create or enter fictional worlds, interact with NPCs/entities, build relationships, trigger consequences, and return to a world that remembers what happened.

DreamChat should support mature fictional worlds. Worlds may include death, violence, horror, gore, strong language, sex, morally dark stories, intense relationships, betrayal, trauma, political conflict, coercive societies, slavery, abuse of power, and adult themes where appropriate to user settings and legal/safety boundaries.

The product should not be sanitized into a generic fantasy toy.

However, private fictional play and public distribution are not the same product/commercial/legal surface.

The core governance model must let private worlds remain raw while ensuring public content is classifiable, reviewable, reportable, age-gated, and governed.

## 3. Product Principle

```txt
Private worlds are user-controlled fictional spaces.
Public content is platform-distributed content.
```

A world or asset is private only if exactly one creator/user can access it.

If another user can access it in any way, it is public-governed.

This includes invite-only, hidden links, collaboration, direct sharing, public discovery, and marketplace publishing.

## 4. Goals

### 4.1 Preserve Raw Private World Immersion

Private users should be able to create dark, brutal, adult, coercive, violent, sexual, political, historical, horror, or morally complex worlds without the product behaving like a sanitized chatbot.

The system should not apply public marketplace rules to private play.

### 4.2 Govern Public Distribution

When content becomes accessible to another user, DreamChat must classify and govern it.

Public distribution requires:

- visibility tracking
- content classification
- age rating
- review status
- report mechanism
- delisting capability
- asset visibility controls
- audit logs

### 4.3 Block Hard Illegal/Abuse Classes

Some content classes are blocked regardless of private/public state.

The product must not generate, store, or distribute hard-prohibited illegal/abuse content.

Final prohibited classes must be validated with legal counsel for launch jurisdictions.

### 4.4 Prevent Downstream Service Misuse

The Image Platform, voice generation, video generation, export, and public publishing systems should not receive requests that the core platform has not classified and authorized.

### 4.5 Make Governance Explicit

Users and creators should understand:

- private world content is broadly user-controlled
- public publishing has rules
- public content may require review
- some content cannot be shared
- some generated media may remain private-only

## 5. Non-Goals

This PRD does not define the final legal policy for every jurisdiction.

This PRD does not define the final public marketplace policy.

This PRD does not implement image provider routing.

This PRD does not require public publishing for the PoC.

This PRD does not require proactive manual review of all private worlds.

This PRD does not define payment strategy for adult/public content.

## 6. Visibility Model

### 6.1 Top-Level Visibility

```ts
world_visibility = "private" | "public"
```

### 6.2 Private

A world is private when:

- only the creator/user can access it
- no other user can view, join, inspect, remix, fork, or open it
- no public URL exists
- no public thumbnail exists
- it is not discoverable
- it is not indexed
- it is not monetized

Private worlds support broad mature fictional freedom, subject only to hard illegal/abuse/security boundaries.

### 6.3 Public

A world, asset, or object is public-governed when any other user can access it in any way.

Public subtypes:

```ts
public_access_type =
  | "unlisted_link"
  | "invite_only"
  | "collaborative"
  | "discoverable"
  | "marketplace_free"
  | "marketplace_paid"
```

All public subtypes require governance.

Strictness may differ by subtype.

For example:

- invite-only may require age gating and reporting but not full marketplace review
- discoverable may require automated classification and public thumbnail review
- marketplace-paid may require stronger review and payment compliance

## 7. Content Classification Model

The core platform should classify worlds, scenes, assets, and generated-media requests.

### 7.1 World Content Profile

Each world should have a content profile.

```ts
world_content_profile {
  world_id: string
  world_visibility: "private" | "public"
  public_access_type?: string

  mature_content_enabled: boolean
  age_rating: "general" | "teen" | "mature" | "adult" | "restricted"

  violence_level: "none" | "moderate" | "graphic" | "extreme"
  sexual_content_level: "none" | "romance" | "suggestive" | "explicit"
  horror_level: "none" | "dark" | "graphic" | "extreme"

  coercive_societies: boolean
  slavery_or_ownership_themes: boolean
  abuse_of_power_themes: boolean
  sexual_violence_as_world_event: boolean
  gore_enabled: boolean

  hard_prohibited_detected: boolean
  public_review_status: "not_required" | "pending" | "approved" | "rejected" | "restricted"
}
```

### 7.2 Content Classes

Recommended initial content classes:

```txt
general
mature_themes
adult_sexual_content
graphic_violence
gore
horror
coercive_society
slavery_or_ownership
abuse_of_power
sexual_violence_as_lore_or_event
explicit_sexual_violence_visual
real_person_sensitive
hard_prohibited
```

### 7.3 Hard-Prohibited Classes

Hard-prohibited classes are blocked regardless of visibility.

Examples:

- child sexual exploitation or sexualized minors
- real-person non-consensual intimate imagery
- illegal content in launch jurisdictions
- real-world extortion, doxxing, stalking, exploitation, or threats
- attempts to generate content for real-world abuse

The exact final list must be legally reviewed.

## 8. Public Governance Requirements

Public content must support:

- classification
- age gate if required
- review state
- asset visibility state
- reporting
- delisting
- appeal/review process later
- audit logs

### 8.1 Review States

```ts
review_status =
  | "not_required"
  | "pending"
  | "approved"
  | "rejected"
  | "restricted"
```

### 8.2 Asset Visibility States

```ts
asset_visibility =
  | "private_only"
  | "public_allowed"
  | "public_blocked"
  | "review_required"
```

A public world may contain private-only assets.

### 8.3 Public Publishing Flow

When user attempts to publish/share content:

```txt
User selects share/publish
  -> core platform classifies world/object/assets
  -> checks visibility transition private -> public
  -> assigns content classes and age rating
  -> checks hard-prohibited classes
  -> determines review requirement
  -> blocks, approves, or queues review
  -> writes audit event
```

## 9. Media Generation Eligibility

Image generation, video generation, voice generation, exports, and public thumbnails must pass core eligibility before downstream generation.

### 9.1 Eligibility States

```ts
media_generation_eligibility =
  | "eligible"
  | "blocked"
  | "review_required"
  | "provider_route_required"
```

### 9.2 Flow

```txt
World requests visual/media asset
  -> classify expected content
  -> check world visibility
  -> check user age/settings
  -> check hard-prohibited classes
  -> check jurisdiction/payment/platform rules if relevant
  -> determine provider capability class
  -> if eligible, call Image Platform
  -> else block before generation
```

### 9.3 Image Platform Contract

The Image Platform receives only approved generation jobs.

Required fields:

```ts
approved_generation_request {
  world_id: string
  visibility_mode: "private" | "public"
  asset_visibility: "private_only" | "public_allowed" | "public_blocked" | "review_required"
  content_classes: string[]
  age_rating: string
  provider_capability_class: string
  audit_required: boolean
  generation_intent: object
}
```

The Image Platform may still fail due to provider refusal, but it should not be responsible for legal/platform eligibility decisions.

## 10. UX Requirements

### 10.1 Private World UX

Private world UX should communicate:

- this is your private fictional world
- mature themes can be enabled
- public sharing is off
- generated assets are private-only by default
- public publishing will require classification/review

It should not interrupt private play with public-content review flows unless a hard-prohibited class is detected.

### 10.2 Public Publishing UX

When publishing, the UX should communicate:

- public content has rules
- some assets may remain private-only
- content may require age rating/review
- public thumbnails/descriptions are governed
- the world may be delisted or restricted if reported/rejected

### 10.3 Share Button UX

If user tries to share a private world:

```txt
This will make the world public-governed.
Before sharing, DreamChat needs to classify the world and decide which assets can be shown publicly.
```

### 10.4 Public Asset UX

If a specific asset is blocked from public view:

```txt
This asset can remain in your private world, but it cannot be used as a public thumbnail or public gallery image.
```

Avoid moralizing language. Use distribution/governance language.

## 11. API Requirements

### 11.1 Core Governance Endpoints

```txt
GET  /v1/worlds/{world_id}/content-profile
PUT  /v1/worlds/{world_id}/content-profile
POST /v1/worlds/{world_id}/classify
POST /v1/worlds/{world_id}/publish-check
POST /v1/assets/{asset_id}/visibility-check
POST /v1/media/eligibility-check
GET  /v1/review-cases/{review_case_id}
```

### 11.2 Publish Check Request

```json
{
  "world_id": "world_123",
  "target_visibility": "public",
  "public_access_type": "invite_only",
  "include_assets": true
}
```

### 11.3 Publish Check Response

```json
{
  "eligible": true,
  "review_required": true,
  "age_rating": "adult",
  "content_classes": ["mature_themes", "graphic_violence", "coercive_society"],
  "blocked_asset_ids": [],
  "private_only_asset_ids": ["asset_123"],
  "review_case_id": "review_456"
}
```

### 11.4 Media Eligibility Request

```json
{
  "world_id": "world_123",
  "visibility_mode": "private",
  "media_type": "image",
  "expected_content_description": "graphic battlefield aftermath with visible gore",
  "requested_asset_type": "scene_image"
}
```

### 11.5 Media Eligibility Response

```json
{
  "eligibility": "provider_route_required",
  "content_classes": ["graphic_violence", "gore"],
  "asset_visibility": "private_only",
  "age_rating": "adult",
  "provider_capability_class": "raw_mature_visuals",
  "audit_required": true,
  "reason_code": "private_adult_mature_content_allowed_with_raw_provider"
}
```

## 12. Audit Requirements

Audit events should be written for:

- visibility changes
- publishing attempts
- publish approvals
- publish rejections
- public asset blocking
- media eligibility blocks
- hard-prohibited detections
- review case creation
- delisting
- user reports

Example:

```ts
audit_event {
  id: string
  event_type: "world.publish_check" | "media.eligibility_blocked" | "asset.public_blocked"
  actor_user_id: string
  world_id: string
  target_resource_type: string
  target_resource_id: string
  decision: string
  reason_code: string
  created_at: datetime
}
```

## 13. Implementation Phases

### Phase 1 — Core Data Model

- world visibility field
- public access subtype field
- world content profile table
- asset visibility field
- review status fields
- audit event table

### Phase 2 — Eligibility Service

- classify request shape
- publish-check endpoint
- media eligibility endpoint
- hard-prohibited class gate
- private/public decision logic

### Phase 3 — Image Platform Integration

- Image Platform receives only approved generation jobs
- generation jobs include content classes, visibility, provider capability class
- blocked requests never call provider

### Phase 4 — Public Sharing / Publishing

- share/publish transition UX
- review queue placeholder
- public asset visibility
- age rating labels
- reporting endpoint

### Phase 5 — Marketplace / Monetization Later

- creator trust score
- payment compliance review
- stricter public asset review
- marketplace-specific restrictions

## 14. Acceptance Criteria

This PRD is implemented when:

- a world can be marked private or public
- private means only one user can access it
- invite-only and hidden-link access are treated as public-governed
- every world has a content profile
- public publishing runs through a publish check
- assets can be private-only even if a world is public
- media generation checks eligibility before calling downstream services
- hard-prohibited content is blocked before generation/storage/distribution
- audit events exist for visibility and eligibility decisions
- user-facing language separates private freedom from public governance
- the Image Platform does not own content-policy decisions

## 15. Open Questions

- Which launch jurisdictions define the first hard legal boundary set?
- What adult age-verification mechanism is needed before public adult content?
- Which payment providers support the desired public/marketplace content profile?
- How much public review can be automated vs manual at launch?
- Should private worlds be end-to-end encrypted or simply access-controlled?
- How long should blocked media-generation requests be retained in logs?
- What is the appeal process for rejected public content?
