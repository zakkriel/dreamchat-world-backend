# DreamChat Architecture Document Set

This folder contains the first architecture document set for DreamChat.

## Files and Confidence

| File | Purpose | Confidence |
|---|---|---:|
| 03_platform_architecture.md | Full platform shape and repo/system boundaries | 86% |
| 04_world_core_architecture.md | Authoritative world core and invariants | 88% |
| 05_plug_and_play_module_architecture.md | Module runtime, contracts, lifecycle, examples | 84% |
| 06_memory_canon_timeline_architecture.md | Canon, event log, memory layers, timeline, correction | 85% |
| 07_ai_orchestration_architecture.md | LLM roles, prompt packing, structured output, async jobs | 78% |
| 08_frontend_architecture.md | Play-first UI, workspace layer, module UI slots | 82% |
| 09_image_platform_integration.md | Integration with existing standalone image platform | 83% |
| 10_marketplace_module_publishing_architecture.md | Future module marketplace and publishing model | 72% |

## Confidence Notes

### 03 Platform Architecture — 86%
High confidence. It matches the product direction: world-state-first, separate frontend, backend, and image platform, with a module runtime inside the world backend. Remaining uncertainty: exact deployment topology, workflow engine, and DB operational model.

### 04 World Core Architecture — 88%
High confidence. Authoritative world core, event log, correction, knowledge boundaries, and bounded backstage review are strongly aligned with the product promise. Remaining uncertainty: how much of this can be implemented in the first PoC without slowing the team.

### 05 Plug-and-Play Module Architecture — 84%
Good confidence. Manifest + capability contract + proposal/validation/commit is the right direction. Main uncertainty: exact runtime/sandbox model for third-party modules.

Recommended follow-up research prompt:

```text
Research 2025-2026 production plugin runtime architectures for SaaS marketplaces, focusing on Wasm plugin hosts, signed plugins, dynamic capability contracts, and safe user-installed extensions. Compare Extism, WASI component model, Grafana plugins, JetBrains plugins, VS Code extensions, and MCP-style tools for a web backend platform.
```

### 06 Memory / Canon / Timeline Architecture — 85%
High confidence. Layered memory and event-sourced canon are the right direction. Remaining uncertainty: implementation sequencing and how aggressively to introduce semantic/reflection jobs.

### 07 AI Orchestration Architecture — 78%
Below 80%. The direction is solid: AI proposes, core validates, structured output matters. But model orchestration is changing fast, and exact agent layout, number of calls, prompt packing strategy, and model mix need testing.

Recommended follow-up research prompt:

```text
Research 2025-2026 architectures for production LLM orchestration in stateful applications. Focus on structured outputs, tool-calling validation, event-sourced AI workflows, prompt packing, multi-agent orchestration, latency/cost optimization, and evaluation harnesses for long-running interactive systems.
```

### 08 Frontend Architecture — 82%
Good confidence. UX structure has been discussed deeply and aligns with play-first + workspace-optional principles. Remaining uncertainty: exact third-party module UI extensibility.

### 09 Image Platform Integration — 83%
Good confidence. Image service is already separate and should remain external to world truth. Remaining uncertainty: how much automated image contradiction detection is needed early.

### 10 Marketplace / Module Publishing Architecture — 72%
Below 80%. This is future-facing and has many unresolved choices: sandboxing, signing, review flow, monetization, module execution model, support burden, compatibility testing, legal/safety review, and public UGC governance.

Recommended follow-up research prompt:

```text
Research 2025-2026 marketplace architectures for user-published extensions/modules. Focus on security review, plugin signing, sandboxed execution, dependency/version management, monetization, moderation, compatibility testing, and operational support models from modern plugin ecosystems.
```

## Recommended Review Order

Review 03, 04, 05, and 06 first. Those define the core architecture. Files 07-10 depend on those decisions.
