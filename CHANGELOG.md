# Changelog

## [0.2.4](https://github.com/kilip/opus/compare/v0.2.3...v0.2.4) (2026-05-18)


### New Features

* **dash:** add branded UI kit, layout shell, and /demo showcase ([c2ee0ec](https://github.com/kilip/opus/commit/c2ee0ec3ed49d9a780bf76c623db9fa2219c5cb9))
* **dash:** implement authentication flow and login page ([424a674](https://github.com/kilip/opus/commit/424a6741ad2c660243210e53fb28b3a7b2ab8b82))
* **dash:** setup Playwright E2E testing framework ([00f3e2c](https://github.com/kilip/opus/commit/00f3e2ccc433ed0d6af3dbe04fc3f6054617e187))
* **dev:** setup development tasks and hot reload ([747deb4](https://github.com/kilip/opus/commit/747deb4da91c4a16cd07608c38c86bb8c7044f27))
* **server:** concrete slog-backed structured logger implementation ([6fe1170](https://github.com/kilip/opus/commit/6fe117032104e078257319e7c7d9fc1c469baf6d))
* **server:** implement asynchronous queue and eventbus messaging ([876c737](https://github.com/kilip/opus/commit/876c73764e7af5a7bf22bb7d1847259c9be2c44a))
* **server:** implement configurable CORS middleware ([2710dde](https://github.com/kilip/opus/commit/2710dde039c2a26153480a2e6eaa5dcd65eaf27c))
* **server:** implement core database strategy ([d4e071a](https://github.com/kilip/opus/commit/d4e071a614d9bab4d2a52d998e07a064233f9b66))
* **server:** implement dash static serving and embedding ([6197b10](https://github.com/kilip/opus/commit/6197b10783c984d6babcab0cfd2f69b8686dc510))
* **server:** implement gofiber http delivery layer ([f6e12b4](https://github.com/kilip/opus/commit/f6e12b407e73bafdf3c04fee398d65adfe463984))
* **server:** implement server oauth, casbin rbac, and uuid v7 refactor ([af9f79a](https://github.com/kilip/opus/commit/af9f79ae5ff11ce80aca27281589b7992ff24b4a))
* **server:** implement stateful Cobra CLI commands ([26e24a0](https://github.com/kilip/opus/commit/26e24a08da6854a6b8698636f94cef9c4286a44d))


### Bug Fixes

* **server:** fix sqlite transaction locks and concurrent contention ([a9213fe](https://github.com/kilip/opus/commit/a9213fe8000d68fb98f44945308567e8510930f9))


### Documentation

* **adr:** add ADR-014 for dash static serving and embedding ([71c8c88](https://github.com/kilip/opus/commit/71c8c88b34bae5eeabd57a0ac264c1bf49b994d1))
* **agent:** reorder consolidation instructions in AGENTS.md ([761bce2](https://github.com/kilip/opus/commit/761bce2935dba2aac12f470b2be6fdac38fda42c))
* **get-opus:** add ADR-013 for npx get-opus installer ([3c7f998](https://github.com/kilip/opus/commit/3c7f998b5e6905036ded622718d7481a16f7178e))
* **server-architecture:** move delivery layer to internal/delivery ([9f8c0fe](https://github.com/kilip/opus/commit/9f8c0fe6048889e7276cb00ff9e65d5db1e35ba6))
* **server:** add ADR-012 module system and dependency injection ([4a09695](https://github.com/kilip/opus/commit/4a09695872b680478a4c5b13dc04e4dace7ecf45))
* **server:** refactor delivery layer structure and naming conventions ([03c0a87](https://github.com/kilip/opus/commit/03c0a877a495056b753b7598fbfe54213c655cac))
* **server:** relocate adapter to internal and rename repository files ([04679d9](https://github.com/kilip/opus/commit/04679d976a966787f55778e8b7465e013dcc1571))
* **server:** remove api prefix mentions in adr-004 and adr-011 ([12cb790](https://github.com/kilip/opus/commit/12cb790454e86e36be9c788046d10405bd8f950a))
* **server:** update ADR-001 for modular architecture and DI bootstrap ([7c2c5ee](https://github.com/kilip/opus/commit/7c2c5eefd4e523e229717fd8881d794a40ef434c))
* **server:** update ADR-008 server queue design ([94dfbf6](https://github.com/kilip/opus/commit/94dfbf609f5a657b8121731fe1ca24c4e49e43fc))

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
