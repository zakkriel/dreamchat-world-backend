# DreamChat Private vs Public Governance Package

This package contains the core product governance ADR and PRD for separating private-world freedom from public-distribution governance.

This belongs in the DreamChat core product/platform layer, not inside the Image Platform.

The Image Platform should not decide whether content is legally/platform eligible. It should receive already-classified, already-authorized generation requests from the core governance layer.

## Files

All four exist. The paths below were wrong in every case until 2026-08-27: this file was written when
the package was a portable folder lifted out of the Image Platform, and it kept that folder's layout
(`adr/`, `prd/`, `schemas/`) after the contents were filed into this repo's real directories.

| File | Where it actually is |
|---|---|
| the ADR | `ADR-P016_private_vs_public_world_governance.md` — **this directory** |
| the PRD | `../../10_prds/prd_private_public_content_governance.md` |
| world-content classification schema | `world_content_governance.schema.json` — **this directory** |
| media-eligibility schema | `media_generation_eligibility.schema.json` — **this directory** |

**The ADR is `ADR-P016`, not `ADR-016`.** The `P` is not decoration. `dreamchat-Image-Platform` has its
own `016-provider-capability-reconciliation.md`, so a bare `ADR-016` names two different decisions in
two repos — the collision that `D-5` and `workspace:ADR-W002` exist to prevent, and that
`../../../../harness/check.sh adr-namespace` enforces. Cite `ADR-P016`. A bare `ADR-016` will not
resolve against this repo and `ci/check_citations.sh` will fail the build for it.

> **Known, not fixed here:** `ADR-P016_private_vs_public_world_governance.md`'s own H1 still reads
> `# ADR-016`, from before the rename. An agent that copies the heading cites an id that does not
> resolve. Left alone deliberately — it is the founder's decision record, and a parallel digest job has
> published a guarantee that ADR decision text is byte-identical. Reported, not edited.

## Core Principle

**Stated in `ADR-P016` §Decision, and not restated here.** `D-6`: the law lives in one place and a copy
is always the one that goes stale. This file's own job is the seam below, which the ADR does not state.

## Why this is core, not Image Platform

Classification happens in the core **before** any generation request is made. The Image Platform
receives already-classified, already-authorised requests and never decides whether content is legally
or platform eligible (`E-1`, `E-2`, and `D-3` — the image platform is handed a subject and a style and
never learns what a world IS).

This is the sentence `docs/areas/art-and-assets.md:25` and `docs/areas/contracts-and-platform.md:28`
cite this file for.
