# ADR-016 — Private vs Public World Governance

## Status

Accepted for core product/platform architecture.

## Context

DreamChat is a persistent AI RPG world product. Its core promise is that users can create and return to fictional worlds with memory, consequences, dynamic NPCs, relationships, in-world time, and continuity.

The product direction includes mature fictional worlds. Worlds may contain death, violence, horror, gore, strong language, sex, morally dark stories, intense relationships, betrayal, trauma, political conflict, coercive societies, slavery, abuse of power, and adult themes where appropriate to user settings and legal/safety boundaries.

The product should not be sanitized into a generic fantasy toy. At the same time, public distribution, public discovery, sharing, creator publishing, marketplace access, monetization, payment processors, hosting providers, and legal obligations create a different risk surface from private single-user play.

A product distinction is needed between:

- what can exist in a private fictional world
- what can be shared with another user
- what can be publicly discovered
- what can be monetized
- what can be rendered as generated media
- what is legally or commercially prohibited

This decision belongs in the DreamChat core platform layer, not the Image Platform. The Image Platform should not be responsible for deciding whether content is legally/platform eligible. It should receive generation requests only after the core platform has classified and authorized the request.

## Decision

DreamChat separates **private single-user worlds** from **public-governed content**.

A world, asset, character, scene, scenario, template, image, text excerpt, or generated media object is private only when it is accessible exclusively by its creator/user.

Any content accessible by another user is considered public-governed, even when access is limited through:

- invite
- hidden link
- direct share
- collaboration
- group world
- friend access
- published character
- published scenario
- public asset pack
- marketplace listing
- remix/fork availability

The platform uses two top-level visibility modes:

```ts
visibility_mode = "private" | "public"
```

Public content may have access subtypes:

```ts
public_access_type =
  | "unlisted_link"
  | "invite_only"
  | "collaborative"
  | "discoverable"
  | "marketplace_free"
  | "marketplace_paid"
```

But all public subtypes are governed by distribution rules.

## Policy Principle

Private worlds are treated as user-controlled fictional spaces with broad mature-content freedom, subject only to hard illegal/abuse/security boundaries.

Public content is treated as platform-distributed content and must be classified, age-gated, reviewed where needed, reportable, delistable, and auditable.

The platform should not silently moralize or sanitize private world content. It should classify content, apply explicit eligibility rules, and route or block only according to defined legal, platform, commercial, visibility, and provider constraints.

## Core Rule

```txt
Private = only the creator/user can access it.
Public = any other user can access it in any way.
```

## Scope

This ADR governs:

- world visibility
- character visibility
- scene visibility
- asset visibility
- media visibility
- scenario/template visibility
- sharing
- publishing
- marketplace distribution
- image-generation eligibility
- public review requirements
- user reporting
- moderation/audit requirements

## Non-Goals

This ADR does not define the full mature-content policy text.

This ADR does not define exact country-by-country legal determinations.

This ADR does not define final payment provider strategy.

This ADR does not implement the Image Platform provider routing itself.

This ADR defines the core architectural separation and governance responsibility.

## Architecture Consequences

### 1. Core Platform Owns Content Eligibility

The core platform must classify content before downstream services are called.

For example, image generation flow becomes:

```txt
User/world requests visual asset
  -> Core content classifier / eligibility service
  -> visibility + content class + age + jurisdiction + provider capability decision
  -> if eligible, call Image Platform
  -> Image Platform generates using approved provider route
```

The Image Platform receives:

- approved generation intent
- content capability class
- visibility mode
- allowed provider class
- required logging/audit flags
- asset visibility restrictions

The Image Platform does not independently decide if the underlying fictional content is permitted.

### 2. Private Play Should Not Be Public-Policy Governed

Private world play should not be interrupted by public publishing rules.

Private mode may still enforce hard legal/abuse/security boundaries, but it should not behave like a public marketplace review process.

### 3. Public Distribution Requires Governance

When content becomes accessible to another user, public rules apply.

Public content requires some combination of:

- content classification
- age gate
- content tags
- review status
- report mechanism
- delisting mechanism
- audit trail
- creator/account trust scoring
- asset visibility control

### 4. Public Subtypes May Have Different Strictness

Public-governed content has access subtypes.

An invite-only world may require less review than a marketplace-paid world, but it is still public-governed because another user can access it.

### 5. Generated Media Requires Pre-Eligibility

If the expected content is outside legal/platform eligibility, the system should not call the Image Platform.

This prevents wasting provider calls and avoids creating illegal, non-distributable, or unsupported media assets.

### 6. Public Asset Visibility Can Differ from World Visibility

A public world may contain private-only assets.

For example:

- the world is public
- the public thumbnail is approved
- some generated images remain private-only
- some scenes are not shareable
- some character variants are not allowed as public gallery assets

## Data Model Requirements

Every world should have:

```ts
world_visibility: "private" | "public"
public_access_type?: "unlisted_link" | "invite_only" | "collaborative" | "discoverable" | "marketplace_free" | "marketplace_paid"
content_profile_id: string
review_status: "not_required" | "pending" | "approved" | "rejected" | "restricted"
```

Every shareable object should have:

```ts
visibility_mode: "private" | "public"
public_access_type?: string
content_classification: string[]
age_rating: "general" | "teen" | "mature" | "adult" | "restricted"
review_status: "not_required" | "pending" | "approved" | "rejected" | "restricted"
asset_visibility: "private_only" | "public_allowed" | "public_blocked" | "review_required"
```

Every media-generation request should include or derive:

```ts
content_eligibility: "eligible" | "blocked" | "review_required" | "provider_route_required"
legal_risk_class: "low" | "medium" | "high" | "blocked"
provider_capability_class: string
visibility_mode: "private" | "public"
asset_visibility: "private_only" | "public_allowed" | "public_blocked" | "review_required"
```

## Hard Boundary Examples

The platform should maintain a hard-prohibited class independent of private/public visibility.

Examples include:

- child sexual exploitation or sexualized minors
- real-person non-consensual intimate imagery
- content that is illegal in target operating jurisdictions
- abuse, exploitation, or doxxing of real people
- attempts to use the platform for real-world harm, threats, extortion, or exploitation

The final list must be validated with legal counsel for launch markets.

## Product Consequences

Positive:

- preserves raw private world immersion
- avoids hidden moral sanitization of private play
- gives clear governance point before sharing/publishing
- protects the image service from policy responsibility
- supports future public publishing and marketplace safely
- supports provider routing and self-hosted media paths
- creates a clean legal/commercial review surface

Tradeoffs:

- requires content classification infrastructure
- requires review states before public publishing
- adds product complexity around visibility and shareability
- requires clear user-facing language
- requires policy/legal review before launch

## Acceptance Criteria

This ADR is implemented when:

- worlds have private/public visibility
- any access by another user is treated as public-governed
- public content has review/classification status
- private play is not governed by public marketplace rules
- hard-prohibited classes are blocked even in private mode
- image generation is called only after core eligibility approval
- Image Platform does not own content-policy decisions
- public assets can be blocked/reviewed separately from private assets
- audit logs exist for publishing, review, blocking, and takedown decisions

## Related Documents

- Product Vision and Promise
- Core User Experience Loop
- Image Platform PRDs
- Image Platform Technical ADRs
- Future Mature Content Policy PRD
- Future Public Publishing / Marketplace PRD
