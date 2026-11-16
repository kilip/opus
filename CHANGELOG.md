# Changelog

All notable changes to this project will be documented in this file.

## [v0.3.0-rc1](https://github.com/kilip/opus/compare/v0.2.0...v0.3.0-rc1) (2026-11-16)

### ✨ Features

- **ui:** add empty state to tasks page ([1bf39ba](https://github.com/kilip/opus/commit/1bf39ba1fca76c35b7e9a688358c3dbd5cf85d97))
- **ui:** make activity author optional in Chatter and task actions ([cad893a](https://github.com/kilip/opus/commit/cad893a3b071dd128399ae57d3438763e23d2260))
- **api:** switch task feature to Ent repo and remove seed data ([fc83fe0](https://github.com/kilip/opus/commit/fc83fe0cdaf0cc794b0c3020841d3dfe158f8cd5))

### 🚀 Deployment

- **deploy:** prefix platform and global env vars with APP_ in Helm chart ([3de2054](https://github.com/kilip/opus/commit/3de2054526128edbf8884613c5c2507bd82bd6c6))
- **deploy:** update auth env vars and secret keys in helm chart ([7401652](https://github.com/kilip/opus/commit/74016521e291b8f47e5f66d356d5c29ede7c0d54))

### ♻️ Code Refactoring

- **api:** move shared task repo types to types.go ([a9c247a](https://github.com/kilip/opus/commit/a9c247ac86870068c59c77492a867e7a92861121))
- **api:** prefix platform and global env vars with APP_ ([6cd4d58](https://github.com/kilip/opus/commit/6cd4d58fd6645957d7a9fc99dfbd29a86c36c2f7))
- **api:** use UUID v7 via shared newID helper for Ent entity IDs ([21fb922](https://github.com/kilip/opus/commit/21fb922712fd81a07ddf795d0faf79522c4baf03))
- **api:** rename auth and oauth env vars to AUTH_* ([1a75e4d](https://github.com/kilip/opus/commit/1a75e4d904d2a9bba8b6d2af82bfdb025446b2a8))

### 📝 Documentation

- **docs:** document APP_ env var prefix convention and reference table ([8bfc536](https://github.com/kilip/opus/commit/8bfc536c4774d4af340b334bd6b46dd8df1ee4ef))
- **docs:** document UUID v7 newID convention in AGENTS.md ([a34cbe1](https://github.com/kilip/opus/commit/a34cbe159ff8c73a9a9a1f600935829b3c91fc4e))
- **docs:** update auth env vars table in agents guide ([f275979](https://github.com/kilip/opus/commit/f2759798fc55c74d1386c62cc44a7cf31bb4080d))

### 🔧 Tooling & Chores

- **release:** v0.3.0-rc1 ([907db3e](https://github.com/kilip/opus/commit/907db3eed3ae0c3e2d815e7753a8c897045ab2a4))
- **release:** prepare 0.3.0-rc1 ([0ded97f](https://github.com/kilip/opus/commit/0ded97fd7b83c36aff56ce7d7ab35344b31cee72))
- **tools:** allow test files in service package to import test deps ([e00501c](https://github.com/kilip/opus/commit/e00501c764590dcfbed0fbb0fe18157ead471277))
- **api:** add ent task repo with key counter and tests ([4b2a8a6](https://github.com/kilip/opus/commit/4b2a8a641b66db85f0e39f40dc07df2df0c3f13c))
- **api:** add task, activity, and counter ent schemas ([716e300](https://github.com/kilip/opus/commit/716e300bc4965f342bf379e28c869a1d8a07061b))
- **tools:** rename SENTRY_DSN to APP_SENTRY_DSN in .env.example ([12172c7](https://github.com/kilip/opus/commit/12172c79c7db5e5350da76bd3e18a42a42e4276c))
- **deploy:** prefix platform and global env vars with APP_ in Helm chart ([3de2054](https://github.com/kilip/opus/commit/3de2054526128edbf8884613c5c2507bd82bd6c6))
- **deploy:** update auth env vars and secret keys in helm chart ([7401652](https://github.com/kilip/opus/commit/74016521e291b8f47e5f66d356d5c29ede7c0d54))

## [v0.2.0](https://github.com/kilip/opus/releases/tag/v0.2.0) (2026-11-16)

### ✨ Features

- **api:** wire event bus into app lifecycle and publish task events ([522d7f1](https://github.com/kilip/opus/commit/522d7f1f90f4dc4a054f4b49e44844ca46dadd3b))
- **api:** add core in-process eventbus package ([1592803](https://github.com/kilip/opus/commit/1592803a35b584aa60ca937bca8e405a3d8c9990))
- **api:** add auth audit log ([de84de0](https://github.com/kilip/opus/commit/de84de0bd8d532b707d0eccd27ef58b63f37d80f))
- **api:** add csrf origin check middleware ([5f862b5](https://github.com/kilip/opus/commit/5f862b5eca883977a0fb7b36da14aca2eac3f3ea))
- **ui:** add connected accounts settings page ([c5e56a4](https://github.com/kilip/opus/commit/c5e56a4b617c883c54577685844e6f0fd0c5d310))
- **api:** add authenticated account link and unlink endpoints ([a6a2c33](https://github.com/kilip/opus/commit/a6a2c33fd6dcc8306c8bd2b0d611fc5efa66d1dd))
- **api:** add facebook oauth provider without email auto-link ([63858a1](https://github.com/kilip/opus/commit/63858a1fae10e8062b1559c0d40296c7082271d8))
- **api:** add Google OAuth provider ([037ea66](https://github.com/kilip/opus/commit/037ea669e6ff3ba30b4f7383ea6e85c976700d45))
- **ui:** render dynamic OAuth provider buttons on LoginPage ([8371609](https://github.com/kilip/opus/commit/83716095b6eec19d40128bd83750b202b62b6391))
- **api:** implement github oauth provider ([183aa8f](https://github.com/kilip/opus/commit/183aa8f966889662e625410ffde6da6d6727f789))
- **api:** generic oauth authorize, callback and linking policy ([e9f1696](https://github.com/kilip/opus/commit/e9f16961b84de3b7f250c26fa215d9766a5cece0))
- **api:** add oauth account schema, provider interface and registry ([c7517ba](https://github.com/kilip/opus/commit/c7517ba480757bafeb5fa1afc545790d621fd1f0))
- **ui:** add OTP resend cooldown and rate limit message ([91946ba](https://github.com/kilip/opus/commit/91946ba6cef56314c678adda5bbc43fb270fd7af))
- **api:** add OTP request rate limiting ([3fd32a0](https://github.com/kilip/opus/commit/3fd32a08fd262e357c15c2d6333d256025fe6458))
- **api:** implement SMTP mailer with TLS and injection guards ([af1b7a5](https://github.com/kilip/opus/commit/af1b7a59f327b8f52e6aa19131cbb422e2434a89))
- **ui:** implement passwordless login and onboarding ([4508d30](https://github.com/kilip/opus/commit/4508d306d0f34a2b8d332b676e517773e22c1b9d))
- **ui:** add session state, RequireAuth guard and 401 handling ([1612a72](https://github.com/kilip/opus/commit/1612a72b4771ed30d277c17e84185811e766c7e4))
- **api:** register setup routes before auth middleware ([a4a6067](https://github.com/kilip/opus/commit/a4a6067a8566ac72c4c068c57dda5f2db27f1575))
- **api:** add setup handlers and RegisterSetup ([3a08018](https://github.com/kilip/opus/commit/3a080180098a2906da83dcc367f4e2b1d8ec3287))
- **api:** extend Repo with CountUsers and CreateUserAndSession ([02462e1](https://github.com/kilip/opus/commit/02462e196b7f3b56b24936d5bd57e367c771324c))
- **api:** add setup DTOs ([717f912](https://github.com/kilip/opus/commit/717f912dad52ed88bac521ef8b1134a96eacf050))
- **api:** add setup-closed sentinel error ([ce3be40](https://github.com/kilip/opus/commit/ce3be40237004284830f4e826c558cced574f2d9))
- **api:** expose SetNow for testable clock injection in handler ([6d9b269](https://github.com/kilip/opus/commit/6d9b2693e22a84de83d8e4dff2dd954f09b1fb08))
- **api:** wire mailer and OTP routes into auth facade and app ([f1b5ce2](https://github.com/kilip/opus/commit/f1b5ce26db483e5f2466d74211e8de26a74dee42))
- **api:** add OTP request and verify handlers with DTOs ([0bf6dfd](https://github.com/kilip/opus/commit/0bf6dfd72ebde9ae55f2fc6c1792f4e620512063))
- **api:** add OTP repo interface and ent implementation ([f3d917a](https://github.com/kilip/opus/commit/f3d917ad5254fbd951aa8efc02c5394a4e668e9f))
- **api:** add OTP domain types and sentinel errors ([68c5b3e](https://github.com/kilip/opus/commit/68c5b3ea71d54d2b1b30661670ec4f4f5e7efe48))
- **api:** wire auth middleware and session config into app ([a956468](https://github.com/kilip/opus/commit/a956468ff92c8b516960ecc55e11db8fe301c590))
- **api:** add auth session domain, repo, and me/logout handlers ([6c82474](https://github.com/kilip/opus/commit/6c82474bcab45d55c71d2766d47b38fd40ce131c))
- **api:** generate ent code for session and otp ([236793d](https://github.com/kilip/opus/commit/236793deabde798c32fb3d5d169df78d7d6fb503))
- **api:** define session and otp ent schemas ([5185f25](https://github.com/kilip/opus/commit/5185f259617be81f382af1c73cdcc673693d4be0))
- **api:** add mailer interface and console implementation ([cc4b36c](https://github.com/kilip/opus/commit/cc4b36c91ff0c09a750240bbc04982ae1503cd8b))
- **api:** implement internal user feature phase 0 ([381dc33](https://github.com/kilip/opus/commit/381dc33b1cfe1bb2f6b455d63987f2e2c889e4dd))
- **api:** bootstrap ent sqlite client and user schema ([01d8307](https://github.com/kilip/opus/commit/01d83077d3f201187fba8384e2850f54441ef7f0))
- **ui:** add api envelope client, task mutations, and update shared utils ([5b7a550](https://github.com/kilip/opus/commit/5b7a5502ee73c2e1a626230e6e118b3a1dd94ca7))
- **api:** establish modular monolith core, httpx envelope, and task module ([6d905cd](https://github.com/kilip/opus/commit/6d905cde70a3bdad4dd782b0035d546ac64e53de))

### 📦 Dependencies

- **deps:** bump go to 1.27.4 and x/crypto to v0.59.0 ([baefa82](https://github.com/kilip/opus/commit/baefa820445db6707560a8443e14586b661c9fb3))
- **deps:** add golang.org/x/oauth2 ([d5df987](https://github.com/kilip/opus/commit/d5df987de31d21c7a6387fb098e647b3bd9229f8))
- **deps:** sync ui/bun.lock with package.json ([6a326c4](https://github.com/kilip/opus/commit/6a326c4d9d601f67782357cb6c5dc27863d7a03d))
- **deps:** add @sentry/vite-plugin dev dependency ([9033573](https://github.com/kilip/opus/commit/903357313f7e0444be7324eb687d18a5f06b4c45))
- **deps:** add sentry sdks for go fiber and react router ([85bfe2f](https://github.com/kilip/opus/commit/85bfe2f8c289205eafd2bea89c76b96cd1d7ff64))
- **deps:** add ent and modernc sqlite pure go driver ([2d4aa7a](https://github.com/kilip/opus/commit/2d4aa7a38c99d59752475e888b84799474db688e))
- **deps:** track go.sum and remove from .gitignore ([29819dd](https://github.com/kilip/opus/commit/29819dd967d999fb6596fc90a62ff1f024d5b98e))
- **deps:** add github.com/sethvargo/go-envconfig dependency ([131d719](https://github.com/kilip/opus/commit/131d71997bdb3014bba194ba1b35a6ec5e3a54c3))
- **deps:** add renovate configuration ([f1c6444](https://github.com/kilip/opus/commit/f1c6444d540f25f7d24cea60ce8b2c33381b10ca))

### 🚀 Deployment

- **deploy:** support auth existingSecret and plain env vars ([cf35b9f](https://github.com/kilip/opus/commit/cf35b9f803afae0c1a718276449bc34fe99ae9d7))
- **deploy:** add persistent storage and DATABASE_URL for SQLite ([4904c7d](https://github.com/kilip/opus/commit/4904c7d50a1643eafc2683f3c72c44e2df5ef716))
- **deploy:** add dockerfile and helm chart ([5e34996](https://github.com/kilip/opus/commit/5e34996243eec4cf43fbbb1361abb287e461f04e))

### ♻️ Code Refactoring

- **tools:** replace telegram send.sh with python-telegram-bot script ([aa73288](https://github.com/kilip/opus/commit/aa732886a80993cdca67bdeec449ea71530d24a4))
- **ui:** point task schema aliases at domain.* ([f894ad1](https://github.com/kilip/opus/commit/f894ad1e7fa721e6c43ee524f2f4a7ac753bfc3c))
- **api:** move auth dto to domain, add service layer ([3abe80d](https://github.com/kilip/opus/commit/3abe80d50495cf41ff4f8d1b0032d9ab0033ef96))
- **api:** move task dto to domain, add service layer ([75ab321](https://github.com/kilip/opus/commit/75ab321e05a0c19f6fd372f6647a8106babf5101))

### 📝 Documentation

- **docs:** document auth feature in AGENTS.md ([e9552fa](https://github.com/kilip/opus/commit/e9552fa67a6ad6703bfb10a6ed18964cea531307))
- **docs:** add mise qc, format, and test to verification commands ([2f909f6](https://github.com/kilip/opus/commit/2f909f6356c951ccc564b32cb9184b7321d05936))
- **docs:** add feature env var prefix rule ([b01d4ec](https://github.com/kilip/opus/commit/b01d4eca345db4490299f7129982897b1e77fef5))
- **docs:** document telegram skill in agy plugin readme ([c93635d](https://github.com/kilip/opus/commit/c93635dce7e9389939cd98a0972a31dbc964b470))
- **docs:** update biome check command for ui config ([2ed83e4](https://github.com/kilip/opus/commit/2ed83e4816b3463e0999e6eca964561d833c7fd0))
- **docs:** remove opus-specific references from agy plugin readme ([10670e0](https://github.com/kilip/opus/commit/10670e06c2cf3cc148387a591c254542ea5b2c8f))
- **docs:** document scout subagent in agy plugin readme ([20b307d](https://github.com/kilip/opus/commit/20b307df1202295dd3f3275e24ba510b242f9233))
- **docs:** update readme skill name from builder to vibe ([c6b23d2](https://github.com/kilip/opus/commit/c6b23d2c713d5a8348f04561f58f95eda667a942))
- **docs:** document orchestrator agent in agy readme ([79a30ce](https://github.com/kilip/opus/commit/79a30ce30aae841e1f4e05318148ebfa72506889))
- **docs:** mandate service layer and domain DTOs ([d145bb5](https://github.com/kilip/opus/commit/d145bb59597ddad174bcdcc6d293e1ef62882b0d))
- **docs:** remove VITE_SENTRY_DSN from env template ([b314611](https://github.com/kilip/opus/commit/b3146117c80bc26c91cd4ca81d6a1b78a6f3bfb2))
- **docs:** add sentry dsn examples to env template ([ec05024](https://github.com/kilip/opus/commit/ec0502422bb4bf6016efe203f0436156475ba880))
- **docs:** align verification commands with ci workflow ([5877c0a](https://github.com/kilip/opus/commit/5877c0aac38794596b5ffa0297d030d0a5eb472b))
- **docs:** update AGENTS.md for module topology and httpx envelope ([bbffc1e](https://github.com/kilip/opus/commit/bbffc1ee551d1091c9bf87e56427c5f12090c6b8))
- **docs:** add release type and atomic staging commit rules ([90baf9d](https://github.com/kilip/opus/commit/90baf9dbd95135cd620f863319dcd6e5474c05be))

### 🧪 Tests

- **api:** add httpx Unauthorized error test ([d4522ab](https://github.com/kilip/opus/commit/d4522ab9ef38a22c53894d6c6469174acb650b13))
- **api:** add setup handler tests ([6f48a3b](https://github.com/kilip/opus/commit/6f48a3bcac7eac168db009d022769acab02c1033))
- **api:** add OTP handler tests ([6638d92](https://github.com/kilip/opus/commit/6638d92c693a82d4c5ba9c4ced9365b0825a27dc))
- **api:** test session and otp schemas on in-memory sqlite ([8b70fb9](https://github.com/kilip/opus/commit/8b70fb969769704f9123bc9e336acc7445c2c43f))

### 👷 Continuous Integration

- **tools:** verify generated Ent code in CI ([cfbd658](https://github.com/kilip/opus/commit/cfbd658abee2089b221b830a2e7971a66e16e75d))
- **tools:** run biome lint from ui directory ([c8c45f8](https://github.com/kilip/opus/commit/c8c45f8afc9066ceff5add8c7e2edf3512c4ab0f))
- **release:** reset manifest version to 0.0.0 ([d7c6669](https://github.com/kilip/opus/commit/d7c6669a83f37ce7048600f2f9185256870b93cb))
- **release:** add checkout step before changelog generation ([3631936](https://github.com/kilip/opus/commit/3631936b86702f56a6f7e0aae320afc0c2bb218d))
- **tools:** upgrade golangci-lint-action to v9 ([1ccb99f](https://github.com/kilip/opus/commit/1ccb99f510f3ecf862cc3f381387c2111c5dea39))
- **tools:** add builder skill for model-pinned subagent execution ([9ea6574](https://github.com/kilip/opus/commit/9ea6574854a65f9a649748e8e8348d3488585afe))
- **tools:** add ci lint and container build workflows ([14f4965](https://github.com/kilip/opus/commit/14f49650a7375cfed93b3d36d3e424fac3f76bed))
- **release:** add release-please config and changelog workflow ([bf8b731](https://github.com/kilip/opus/commit/bf8b731ed68bf370ebd71fe7aac6a92fb04294f3))

### 🔧 Tooling & Chores

- **release:** v0.2.0 ([0dd9c7e](https://github.com/kilip/opus/commit/0dd9c7ecfd51625793a680ddc2a1d59c85cf7dae))
- **tools:** show curl errors in telegram send.sh ([4d70bbc](https://github.com/kilip/opus/commit/4d70bbc10e8ba339771e747d471daafcad59b809))
- **tools:** update depguard rules for feature architecture ([29116e9](https://github.com/kilip/opus/commit/29116e9216e4cd9fbb70fbeb194076fc9a0263c4))
- **deploy:** support auth existingSecret and plain env vars ([cf35b9f](https://github.com/kilip/opus/commit/cf35b9f803afae0c1a718276449bc34fe99ae9d7))
- **tools:** fix YAML frontmatter in telegram skill and agy agents ([47271a4](https://github.com/kilip/opus/commit/47271a4d5aa10e8cc4c58312b079e59dc183a026))
- **tools:** run all checks before exiting in mise qc ([3893be8](https://github.com/kilip/opus/commit/3893be83014a3c9599669769c675f8839138b4a0))
- **ui:** format opus.css with biome ([63180f2](https://github.com/kilip/opus/commit/63180f2ee146e3ed581da0b5071ddd8c9aa8ce5a))
- **tools:** add format, test, and qc tasks to mise.toml ([5cf9a70](https://github.com/kilip/opus/commit/5cf9a70907bbf4c30c56ed3dfe8cc165129bfdca))
- **tools:** remove invalid grep_search tool from coder ([a8e867f](https://github.com/kilip/opus/commit/a8e867fd619a5f960b3395c4d1d6dfd42180b9a2))
- **tools:** use run_command for scout discovery ([09ed181](https://github.com/kilip/opus/commit/09ed1819cf47eb56bb27bdbeaa1826da0887d561))
- **tools:** add agent task issue template and PR template ([f380f09](https://github.com/kilip/opus/commit/f380f09ecd0a2b9139bc014252bc4484ea9af69f))
- **tools:** format agy extension configs with biome ([b201b5b](https://github.com/kilip/opus/commit/b201b5bc03c2991c99fe41760b7bc4bfd3793029))
- **tools:** add retry with backoff to telegram send.sh ([3666fcf](https://github.com/kilip/opus/commit/3666fcfdcbf8036c4cdc5cb4cd44192af10c601a))
- **tools:** compact agy skills and agents context and use flash_lite ([7dea773](https://github.com/kilip/opus/commit/7dea7732e3f549fa42a9c32de23ba3d074d9ac61))
- **tools:** send optional telegram summary on vibe completion ([7ccda23](https://github.com/kilip/opus/commit/7ccda23c688239be91249227fc71d0c2e7dc9efc))
- **tools:** harden telegram send.sh and update plugin metadata ([bc44f09](https://github.com/kilip/opus/commit/bc44f09292a62ceac023bab6f25fea4f987e1876))
- **tools:** support gitignored settings.json in telegram skill ([fce6107](https://github.com/kilip/opus/commit/fce61078d00cb7211ef42bd3eda21a14507f87d4))
- **tools:** add telegram skill to agy plugin ([0e31a73](https://github.com/kilip/opus/commit/0e31a735dc647af1bb0c3606ebffbc05e2344f2e))
- **tools:** ignore skill settings.json ([972401b](https://github.com/kilip/opus/commit/972401b84757316683e60eea7e4d1340b220b174))
- **deps:** bump go to 1.27.4 and x/crypto to v0.59.0 ([baefa82](https://github.com/kilip/opus/commit/baefa820445db6707560a8443e14586b661c9fb3))
- **tools:** move biome config to ui ([83ac658](https://github.com/kilip/opus/commit/83ac65844c80b5d029e69e702a9f7b7c1e01e530))
- **ui:** regenerate api schema for oauth endpoints ([87721c5](https://github.com/kilip/opus/commit/87721c579068d532e0a26fc5b25f83f0f9de0c0f))
- **deps:** add golang.org/x/oauth2 ([d5df987](https://github.com/kilip/opus/commit/d5df987de31d21c7a6387fb098e647b3bd9229f8))
- **tools:** make agy plugin subagents project-agnostic ([f8dc10e](https://github.com/kilip/opus/commit/f8dc10eb17f155ade211e07945a63c65d6ff91d6))
- **tools:** add scout recon subagent and narrow planner to read_file ([f91275c](https://github.com/kilip/opus/commit/f91275cdade9ad5bbaed31fa7800b0237b328118))
- **tools:** require explicit /vibe trigger and drop indonesian terms ([64cd65c](https://github.com/kilip/opus/commit/64cd65ce5c0ab91d4355a3cf6c6ddae5a7b1de64))
- **tools:** update vibe skill metadata and pr-manager mode ([ce4144e](https://github.com/kilip/opus/commit/ce4144eae82a4ceb54748b67f060425b414c6457))
- **tools:** save planner output as artifact in builder skill ([8ff5eb6](https://github.com/kilip/opus/commit/8ff5eb6a662faf7017ea48573b5cf67b705605e0))
- **tools:** rename agy plugin to opus in plugin.json ([583ac41](https://github.com/kilip/opus/commit/583ac410d08383037adf807fdf02998a9b5126a2))
- **tools:** set all agy subagents to flash ([a3f1476](https://github.com/kilip/opus/commit/a3f147622589afd68affda95467ba86dcffd236d))
- **ui:** regenerate openapi schema types ([395b80d](https://github.com/kilip/opus/commit/395b80d547760a0484486878dabed8b6d6512ca5))
- **tools:** set coder to flash and support subagents to flash_lite ([4557f87](https://github.com/kilip/opus/commit/4557f87245a52e0353407d86b77b41ca140ac084))
- **tools:** add agy builder plugin and subagents ([f832eae](https://github.com/kilip/opus/commit/f832eaeddfa6a195ad9758da05484b51167046de))
- **tools:** ignored opus.db ([b8ab1cc](https://github.com/kilip/opus/commit/b8ab1ccd42ed376f00b2a934cd919d0ac2d8b9f0))
- **tools:** fix ui formattng ([3ae4f02](https://github.com/kilip/opus/commit/3ae4f023dd3c2001403240f1ecdfdd9390b7e1b5))
- **tools:** parse auth domain dto in swag spec ([b0f6907](https://github.com/kilip/opus/commit/b0f6907a9d1016451da507c25e8e2832b19f192d))
- **api:** promote sentry-go to direct module dependency ([1d36e6e](https://github.com/kilip/opus/commit/1d36e6eb7cc2d8df2ac17bda46237fa9a8c5a0d9))
- **api:** regenerate OpenAPI spec after setup endpoints ([6fd37c0](https://github.com/kilip/opus/commit/6fd37c0613032208191782e2dbddb26ef8ec7be1))
- **api:** regenerate OpenAPI spec after OTP handlers ([fb4904f](https://github.com/kilip/opus/commit/fb4904ff0ed78a95c0a8c02e0b1d0ca21f574e21))
- **api:** regenerate OpenAPI spec and UI types with auth endpoints ([df9d1a7](https://github.com/kilip/opus/commit/df9d1a7c6afe044a81f7080b3351c0f8a9f61fda))
- **deps:** sync ui/bun.lock with package.json ([6a326c4](https://github.com/kilip/opus/commit/6a326c4d9d601f67782357cb6c5dc27863d7a03d))
- **ui:** remove hardcoded sentry dsn fallback ([bcc50f5](https://github.com/kilip/opus/commit/bcc50f5c17669e618a7353d83b1396186e185424))
- **ui:** rely only on runtime SENTRY_DSN ([f0011de](https://github.com/kilip/opus/commit/f0011de4b046092de6210a98141fed70c73a9010))
- **ui:** add runtime config.js script and entry support ([d11dcf6](https://github.com/kilip/opus/commit/d11dcf6ba2cbe2a7e57b2d2a8c7ac10f9d7298dc))
- **api:** support APP_VERSION env and dynamic config.js ([c2ff4ad](https://github.com/kilip/opus/commit/c2ff4ad441a656bcd9a6dd733ccaf75830fc94c8))
- **ui:** configure sentry vite plugin for sourcemap uploads ([c2bc0bd](https://github.com/kilip/opus/commit/c2bc0bdfaf7436685fd93dc3a9de024e9b068342))
- **deps:** add @sentry/vite-plugin dev dependency ([9033573](https://github.com/kilip/opus/commit/903357313f7e0444be7324eb687d18a5f06b4c45))
- **ui:** initialize sentry in react router client entry ([54b03a6](https://github.com/kilip/opus/commit/54b03a62197f249e6548d37e64b5833b79a67bcf))
- **api:** integrate sentry sdk with fiber error handling ([f7846f8](https://github.com/kilip/opus/commit/f7846f8a7cef1f7e713f28434a0c3482ec2d4ce7))
- **deps:** add sentry sdks for go fiber and react router ([85bfe2f](https://github.com/kilip/opus/commit/85bfe2f8c289205eafd2bea89c76b96cd1d7ff64))
- **deploy:** add persistent storage and DATABASE_URL for SQLite ([4904c7d](https://github.com/kilip/opus/commit/4904c7d50a1643eafc2683f3c72c44e2df5ef716))
- **tools:** remove local skills directory ([dcd0d30](https://github.com/kilip/opus/commit/dcd0d30396c8f631d57449224efe12579a00d75b))
- **deps:** add ent and modernc sqlite pure go driver ([2d4aa7a](https://github.com/kilip/opus/commit/2d4aa7a38c99d59752475e888b84799474db688e))
- **release:** v0.1.0 (#2) ([f175116](https://github.com/kilip/opus/commit/f175116ac9aeaacec5a360a0126aa8684d5edbc1))
- **ui:** track build/client/.keep for go embed ([f4e3d68](https://github.com/kilip/opus/commit/f4e3d68d219e379cf794c221d3e373d816c325db))
- **tools:** fix slash delimiter breakage in fix-emoji ([db55984](https://github.com/kilip/opus/commit/db5598415e1728481d1cc10e5384b7dab34c95a3))
- **deps:** track go.sum and remove from .gitignore ([29819dd](https://github.com/kilip/opus/commit/29819dd967d999fb6596fc90a62ff1f024d5b98e))
- **deps:** add github.com/sethvargo/go-envconfig dependency ([131d719](https://github.com/kilip/opus/commit/131d71997bdb3014bba194ba1b35a6ec5e3a54c3))
- **tools:** update swag search paths for internal packages ([60ee9fc](https://github.com/kilip/opus/commit/60ee9fc5ee8dd960edf59e82950ca0d5561dbcbc))
- **deps:** add renovate configuration ([f1c6444](https://github.com/kilip/opus/commit/f1c6444d540f25f7d24cea60ce8b2c33381b10ca))
- **deploy:** add dockerfile and helm chart ([5e34996](https://github.com/kilip/opus/commit/5e34996243eec4cf43fbbb1361abb287e461f04e))
- **tools:** initial commit ([37105d0](https://github.com/kilip/opus/commit/37105d08a3bbab85096d729d54043324414d6529))

## [v0.1.0](https://github.com/kilip/opus/releases/tag/v0.1.0) (2026-11-14)

### ✨ Features

- **ui:** add api envelope client, task mutations, and update shared utils ([8a828c5](https://github.com/kilip/opus/commit/8a828c53a59429a45e41bcc1c246500f49a975f9))
- **api:** establish modular monolith core, httpx envelope, and task module ([5e492ec](https://github.com/kilip/opus/commit/5e492ec21cb06596ff598ef9dfdec32d9da4d8ec))

### 📦 Dependencies

- **deps:** track go.sum and remove from .gitignore ([00ab60e](https://github.com/kilip/opus/commit/00ab60e7941320ec06ac812a865a9071b7319711))
- **deps:** add github.com/sethvargo/go-envconfig dependency ([948dd58](https://github.com/kilip/opus/commit/948dd5868b40907bab8538eee3e923c0419d8e59))
- **deps:** add renovate configuration ([ebdc1f6](https://github.com/kilip/opus/commit/ebdc1f6ff764b15401f3e5cb7da5b66e9de7abc8))

### 🚀 Deployment

- **deploy:** add dockerfile and helm chart ([cb66cb5](https://github.com/kilip/opus/commit/cb66cb59760f203095c27402c6220dd9220d08fa))

### 📝 Documentation

- **docs:** align verification commands with ci workflow ([97cc964](https://github.com/kilip/opus/commit/97cc9643ef9c57ce95f27b023d68b4da165a57c7))
- **docs:** update AGENTS.md for module topology and httpx envelope ([c2ca708](https://github.com/kilip/opus/commit/c2ca708f67122c1db428cb8dd7f5339157fc3ff6))
- **docs:** add release type and atomic staging commit rules ([5e6dbcf](https://github.com/kilip/opus/commit/5e6dbcfe92946faefdc21dfd3c8fd6faf8ae4764))

### 👷 Continuous Integration

- **release:** reset manifest version to 0.0.0 ([ec170e2](https://github.com/kilip/opus/commit/ec170e2e32487f09f5ed0ddfeb7af251e16d4551))
- **release:** add checkout step before changelog generation ([6f49368](https://github.com/kilip/opus/commit/6f493682851ac6f2ef24c233316ec04583cd899f))
- **tools:** upgrade golangci-lint-action to v9 ([5259a07](https://github.com/kilip/opus/commit/5259a07aa51102ab7cdfb766dec94dbd880a2afc))
- **tools:** add builder skill for model-pinned subagent execution ([f75e978](https://github.com/kilip/opus/commit/f75e9782cb4df1078f027e65bbba62bb3d4b092c))
- **tools:** add ci lint and container build workflows ([9008d9e](https://github.com/kilip/opus/commit/9008d9ece78544c2e548bcdc422ad9ea099afc44))
- **release:** add release-please config and changelog workflow ([27d29c7](https://github.com/kilip/opus/commit/27d29c73a882d2e15cca6dce1a634c8f9334770b))

### 🔧 Tooling & Chores

- **release:** v0.1.0 ([6fa702a](https://github.com/kilip/opus/commit/6fa702a3691883677c9d0282b16095232bf0ca83))
- **ui:** track build/client/.keep for go embed ([14302cd](https://github.com/kilip/opus/commit/14302cd4249b87f72660f398d519ec25dd71e804))
- **tools:** fix slash delimiter breakage in fix-emoji ([56c2510](https://github.com/kilip/opus/commit/56c2510cec123868a8b33389a4db6ece4391fa98))
- **deps:** track go.sum and remove from .gitignore ([00ab60e](https://github.com/kilip/opus/commit/00ab60e7941320ec06ac812a865a9071b7319711))
- **deps:** add github.com/sethvargo/go-envconfig dependency ([948dd58](https://github.com/kilip/opus/commit/948dd5868b40907bab8538eee3e923c0419d8e59))
- **tools:** update swag search paths for internal packages ([4758faa](https://github.com/kilip/opus/commit/4758faa7a0125e0c2c9ac57587bb5be7d5506257))
- **deps:** add renovate configuration ([ebdc1f6](https://github.com/kilip/opus/commit/ebdc1f6ff764b15401f3e5cb7da5b66e9de7abc8))
- **deploy:** add dockerfile and helm chart ([cb66cb5](https://github.com/kilip/opus/commit/cb66cb59760f203095c27402c6220dd9220d08fa))
- **tools:** initial commit ([e68f6da](https://github.com/kilip/opus/commit/e68f6da8cff86e62ac1fbb0768f83999e1f2c127))
