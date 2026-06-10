# DreamChat Private vs Public Governance Package

This package contains the core product governance ADR and PRD for separating private-world freedom from public-distribution governance.

This belongs in the DreamChat core product/platform layer, not inside the Image Platform.

The Image Platform should not decide whether content is legally/platform eligible. It should receive already-classified, already-authorized generation requests from the core governance layer.

## Files

- `adr/ADR-016-private-vs-public-world-governance.md`
- `prd/PRD-core-private-public-content-governance.md`
- `schemas/world_content_governance.schema.json`
- `schemas/media_generation_eligibility.schema.json`

## Core Principle

Private single-user worlds support broad raw fictional freedom, subject to hard illegal/abuse/security boundaries.

Any content accessible by another user is public-governed, even if access is limited by invite, hidden link, collaboration, or direct share.

The core platform must classify content and approve/reject/route it before downstream services such as image generation are called.
