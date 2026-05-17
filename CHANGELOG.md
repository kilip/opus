# Changelog

## [0.2.3](https://github.com/kilip/opus/compare/v0.2.2...v0.2.3) (2026-05-17)


### New Features

* add base config model and reloadable interface ([c02dd8b](https://github.com/kilip/opus/commit/c02dd8b6881d9a14be8f4e83be556dfed813a22f))
* **ci:** add github action for continuous integration ([934f997](https://github.com/kilip/opus/commit/934f9979f441868d5f9698a5e2d9ada26fb5e541))
* **ci:** configure automated releases with release-please and goreleaser ([43868f3](https://github.com/kilip/opus/commit/43868f37ae6a24d34ad0bcbf2f4e042d8ffdcbc0))
* **ci:** configure global automated releases ([b287fa3](https://github.com/kilip/opus/commit/b287fa3f76cbc024867caefbe8e2002c43973347))
* **ci:** customize release PR title pattern ([eb25945](https://github.com/kilip/opus/commit/eb25945b9f792a5ead7d942447299a622eee2549))
* **config:** implement hybrid configuration resolution and watch reloading ([6a58110](https://github.com/kilip/opus/commit/6a58110e59ea8c1d850903c1d479a32edc034295))
* **config:** implement server configuration loader and schema generation per ADR-002 ([14ac418](https://github.com/kilip/opus/commit/14ac418a5bd6956bff195ea020d24d50edfb9235))
* **dash:** add offline banner and network hook ([c61c913](https://github.com/kilip/opus/commit/c61c913233543b0f36ec2a6323c11e33e5dc67df))
* **dash:** add shared api client ([42464d4](https://github.com/kilip/opus/commit/42464d4eddda72cb683ce50e7b731a177b7e8104))
* **dash:** integrate biome and replace eslint ([ff8234b](https://github.com/kilip/opus/commit/ff8234b559a28729450197e9ddce67470f89e214))
* **dash:** scaffold basic routes and mount app ([958e983](https://github.com/kilip/opus/commit/958e983b4d4041da7da1e238be89d7cf629bded8))
* implement config directory auto-creation ([1576aee](https://github.com/kilip/opus/commit/1576aeed0992bd34acbf4e68b4b65ca9f9de31c2))
* implement viper loading with path resolution and fallbacks ([0d93ecd](https://github.com/kilip/opus/commit/0d93ecdb517ee795a2ae0365621eb90f64a459dd))


### Documentation

* add ADR-002 for server configuration management ([f724aa4](https://github.com/kilip/opus/commit/f724aa4dd905c1b5d54d8f3c229eee7c80917384))
* add ADR-003 for frontend architecture ([b348534](https://github.com/kilip/opus/commit/b3485346297f9d59dfd2dd97474a35472ba96a96))
* add ADR-004 api response contract ([0b19c79](https://github.com/kilip/opus/commit/0b19c793c4d3cf754475fdeb43b6c957871e5a40))
* add ADR-005 delivery layer using gofiber v3 ([fdb0f44](https://github.com/kilip/opus/commit/fdb0f44886bfde80273170b4ca2cd05126476e64))
* add ADR-006 server logger architecture ([b7f774b](https://github.com/kilip/opus/commit/b7f774b89b3e988605d057e3378d6c1a1f98d409))
* add ADR-007 orm and database strategy ([2ff373b](https://github.com/kilip/opus/commit/2ff373bfe28e22e07b2294a7c59b422329459f81))
* add ADR-008 server queue architecture ([96c047d](https://github.com/kilip/opus/commit/96c047db9fb9649ab7c9a34789a08404ad802c18))
* add ADR-009 server testing strategy ([b5b73a7](https://github.com/kilip/opus/commit/b5b73a7926150129238b70bd5682e21e6ae6bd06))
* add ADR-010 server coding and linting standards ([b69aa29](https://github.com/kilip/opus/commit/b69aa29ffe7eb063372a3560a98883691493b710))
* add ADR-011 authentication and authorization ([f1541c1](https://github.com/kilip/opus/commit/f1541c1dc553ffcea95cdf0863651e1816d8e7c7))
* add initial ADR and project documentation ([2615ed1](https://github.com/kilip/opus/commit/2615ed19fda2bbda3ac08da787f1139cc8a60126))
* **ADR-004:** remove /api prefix from URL path examples ([f92b8be](https://github.com/kilip/opus/commit/f92b8beacdc6149618788689b63e5b653482bfc3))
* clarify that directory structure in ADR-001 is illustrative ([f3ded85](https://github.com/kilip/opus/commit/f3ded854cc099732580f6307c59404885a8d3939))
* **config:** add project specs, agent workflows, and guidelines documentation ([7b33b12](https://github.com/kilip/opus/commit/7b33b12aae77c569a72179bb6ef4c2d855c267dc))
* **dash:** add Charcoal and Rust branding guidelines ([a9bdd26](https://github.com/kilip/opus/commit/a9bdd26f0e8c20d633d09130809716bb207051b5))
* **dash:** update implementation plan for routing and offline setup ([36d3d70](https://github.com/kilip/opus/commit/36d3d70652a86ee4fc1a0a1bd9b3016851d27a53))
* expand and refine ADR-006 server logger architecture ([09b7d6f](https://github.com/kilip/opus/commit/09b7d6f4d9d469a4336e4ff9a4ba5b28110c3c09))
* refactor AGENTS.md with comprehensive project architecture ([a4b9ea1](https://github.com/kilip/opus/commit/a4b9ea1c6f41dcab64c8bbcbbc794842f30c4bf1))
* refine conventional commits guidelines in AGENTS.md ([7cb7591](https://github.com/kilip/opus/commit/7cb7591ffda43b7be9a53166a6f91e2dfb94f525))
* remove /api prefix from all ADRs ([f5b93e4](https://github.com/kilip/opus/commit/f5b93e457c62ed282b2165fbbf506afc00c5fd97))
* remove /v1/ prefix references from ADR-004 ([3a2e57b](https://github.com/kilip/opus/commit/3a2e57b46cf30d442400a1a5bfb6d831265cdeea))
* rename ADR-005 to server-delivery-layer-with-gofiber-v3 ([0b33ed7](https://github.com/kilip/opus/commit/0b33ed75402610c8e0ff5eba4beebe1f36b71bb1))
