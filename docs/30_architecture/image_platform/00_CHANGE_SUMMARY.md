# DreamChat Image Platform V2 Addendum — Change Summary

This package improves the existing DreamChat Image Platform PRDs and tech docs instead of replacing them wholesale.

The existing docs already establish strong foundations:

- standalone image platform service
- API-first async generation jobs
- character/place visual identities
- asset packs and variants
- storage, retrieval, versioning, and cache strategy
- Go service, Postgres, Redis, S3, provider adapters
- OpenAPI-first implementation

This addendum adds the missing product/technical detail discovered during UX work:

1. NPC expression sprite sheets
2. deterministic slicing into expression assets
3. location state packs for scene background continuity
4. artifact/item visuals connected to the Aux Context Sidebar and inventory
5. strict distinction between UI inspection and world/canon actions
6. runtime visual selection policy
7. no real-time generation during normal play unless essential
8. asset metadata extensions for sprite sheets and scene-state matching

Recommended integration:

- Treat `prd/08_visual_asset_packs_and_sprite_sheets_prd.md` as a new PRD after existing PRD 04.
- Treat `tech/architecture/sprite_sheet_pipeline.md` as an architecture addendum after existing asset-versioning docs.
- Treat `tech/api/asset_pack_and_sprite_sheet_api_addendum.md` as an API addendum to the existing OpenAPI contract.
- Treat `tech/db/schema_extension_sprite_sheets.sql` as a migration proposal, not final production SQL.
