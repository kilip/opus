# Changelog

All notable changes to this project will be documented in this file.

## [v0.2.0](https://github.com/kilip/opus/compare/v0.1.0...v0.2.0) (2026-11-15)

### ✨ Features

- **api:** add authenticated account link and unlink endpoints ([a011abd](https://github.com/kilip/opus/commit/a011abdd3ca24ec7d0aec8a0742891beb596fb91))
- **api:** add facebook oauth provider without email auto-link ([27a1010](https://github.com/kilip/opus/commit/27a10107c2c7c490ec005a45b7db710ce7a4a780))
- **api:** add Google OAuth provider ([aae6687](https://github.com/kilip/opus/commit/aae66872516b2930d50ad7f1ce9a1cbebf9bbf58))
- **ui:** render dynamic OAuth provider buttons on LoginPage ([5b6585c](https://github.com/kilip/opus/commit/5b6585c9d68579f3d10534b909a60a7d60765a52))
- **api:** implement github oauth provider ([0fcfcae](https://github.com/kilip/opus/commit/0fcfcae51e4a50b080311b9d8b91c7f84dc7b63d))
- **api:** generic oauth authorize, callback and linking policy ([1071f85](https://github.com/kilip/opus/commit/1071f85fd4b984ba302d5401ba66e9f23f5425b5))
- **api:** add oauth account schema, provider interface and registry ([1905ea3](https://github.com/kilip/opus/commit/1905ea31dce44bc2c413a2f14fcf6f6cd500daca))
- **ui:** add OTP resend cooldown and rate limit message ([e311d9d](https://github.com/kilip/opus/commit/e311d9de6d36eb5084b743e53c4b5f9ca49efe95))
- **api:** add OTP request rate limiting ([53954b2](https://github.com/kilip/opus/commit/53954b243d0c720127c78d60947c638babd5b451))
- **api:** implement SMTP mailer with TLS and injection guards ([f7ccb73](https://github.com/kilip/opus/commit/f7ccb738d6a931b14f86ecc6d4f2c10ed18a44a8))
- **ui:** implement passwordless login and onboarding ([b355fc1](https://github.com/kilip/opus/commit/b355fc1b2763aae2851ec67ce25ee466627c6880))
- **ui:** add session state, RequireAuth guard and 401 handling ([8f1c507](https://github.com/kilip/opus/commit/8f1c5077ae59936411c24c3d202f7335bfa7cc5d))
- **api:** register setup routes before auth middleware ([38aa617](https://github.com/kilip/opus/commit/38aa617f65098d93cdaf84f62c1c275216949b5f))
- **api:** add setup handlers and RegisterSetup ([1615120](https://github.com/kilip/opus/commit/1615120fb36946d9d3f2d79b497637f647c74153))
- **api:** extend Repo with CountUsers and CreateUserAndSession ([84f1a41](https://github.com/kilip/opus/commit/84f1a416feb3ff5610ee78bad368e45074e17136))
- **api:** add setup DTOs ([76043b9](https://github.com/kilip/opus/commit/76043b94e4c246a9470f09ea8da3767e5c3ee598))
- **api:** add setup-closed sentinel error ([d0e5040](https://github.com/kilip/opus/commit/d0e5040786371759cc6a99083d40bc36c5980fde))
- **api:** expose SetNow for testable clock injection in handler ([c3011d2](https://github.com/kilip/opus/commit/c3011d244dde088053b3686833004c63ffab9263))
- **api:** wire mailer and OTP routes into auth facade and app ([24391d9](https://github.com/kilip/opus/commit/24391d9987ae25f551d68df8d086fa6db0304652))
- **api:** add OTP request and verify handlers with DTOs ([22a5aae](https://github.com/kilip/opus/commit/22a5aaec3f0fa639a79821d7ff874c88cd02a81f))
- **api:** add OTP repo interface and ent implementation ([cbd92b3](https://github.com/kilip/opus/commit/cbd92b3ca0ec67a29d8d35a094374e9a016e9e4d))
- **api:** add OTP domain types and sentinel errors ([97efc10](https://github.com/kilip/opus/commit/97efc108100d707ebc074394cfc86aece9f809af))
- **api:** wire auth middleware and session config into app ([7764bde](https://github.com/kilip/opus/commit/7764bde3e90d61419db07a3b9928e98b666b9570))
- **api:** add auth session domain, repo, and me/logout handlers ([619082d](https://github.com/kilip/opus/commit/619082dc34ea264eaf6580766fe10b0173b8ba94))
- **api:** generate ent code for session and otp ([6467fdd](https://github.com/kilip/opus/commit/6467fdd2ea57cc4f256fcc0e91c9e72bcb8aaa51))
- **api:** define session and otp ent schemas ([28c7f60](https://github.com/kilip/opus/commit/28c7f60e5334335b48388dd654a027629bb1f4df))
- **api:** add mailer interface and console implementation ([64a2c1a](https://github.com/kilip/opus/commit/64a2c1a1d5002ed8664c223b7d2e84b320bfdc43))
- **api:** implement internal user feature phase 0 ([12c5808](https://github.com/kilip/opus/commit/12c58083148750179d2630f2b4b1996832dfe11d))
- **api:** bootstrap ent sqlite client and user schema ([be5c58d](https://github.com/kilip/opus/commit/be5c58dbb145a42aca89a92ad36d4235c8bfdecf))

### 📦 Dependencies

- **deps:** bump go to 1.27.4 and x/crypto to v0.59.0 ([269942e](https://github.com/kilip/opus/commit/269942e63bde31945213a0b77db63e3635c0b1a4))
- **deps:** add golang.org/x/oauth2 ([7be5988](https://github.com/kilip/opus/commit/7be59888274b4dc0676b85eba545408de582a4dc))
- **deps:** sync ui/bun.lock with package.json ([ab85ac5](https://github.com/kilip/opus/commit/ab85ac58ea640fa14247958344a53a4743b1fd01))
- **deps:** add @sentry/vite-plugin dev dependency ([63f2503](https://github.com/kilip/opus/commit/63f2503bb82ba753eed0a1b87cd5d7f947ecbe07))
- **deps:** add sentry sdks for go fiber and react router ([310c4ef](https://github.com/kilip/opus/commit/310c4ef219f3e804daf3b060d5b211a60896ac75))
- **deps:** add ent and modernc sqlite pure go driver ([b7d15b0](https://github.com/kilip/opus/commit/b7d15b0cdea7daaf4a28e77b3f096ce6b1b17830))

### 🚀 Deployment

- **deploy:** add persistent storage and DATABASE_URL for SQLite ([670cad1](https://github.com/kilip/opus/commit/670cad17f190a7982cc03ee129f9a60cf690c691))

### ♻️ Code Refactoring

- **ui:** point task schema aliases at domain.* ([c29cc1f](https://github.com/kilip/opus/commit/c29cc1f45f9f80813e5cb4fa6b4b3a1825f8e730))
- **api:** move auth dto to domain, add service layer ([1d9b557](https://github.com/kilip/opus/commit/1d9b557d45720f48a2b247b936346209aa78d905))
- **api:** move task dto to domain, add service layer ([9454444](https://github.com/kilip/opus/commit/9454444061f703c336c4bdb47baf34b392fffd68))

### 📝 Documentation

- **docs:** update biome check command for ui config ([7446967](https://github.com/kilip/opus/commit/7446967580ee49d2f74599ce083944a31b30cb62))
- **docs:** remove opus-specific references from agy plugin readme ([c5ae525](https://github.com/kilip/opus/commit/c5ae525ef71a7e267f8d8ea69eb58ce48007736e))
- **docs:** document scout subagent in agy plugin readme ([f2eee5e](https://github.com/kilip/opus/commit/f2eee5efd4a1b55db05ce9a08bf2b10e4a69ea69))
- **docs:** update readme skill name from builder to vibe ([bcd8536](https://github.com/kilip/opus/commit/bcd853677c2dbe7679c840ca97b1c20aa04d813c))
- **docs:** document orchestrator agent in agy readme ([09d5fc4](https://github.com/kilip/opus/commit/09d5fc4fe8c9d56f0f2cb63c58ab7ce7b854192e))
- **docs:** mandate service layer and domain DTOs ([2d7056d](https://github.com/kilip/opus/commit/2d7056d7542950e3b11c4c4126b12470db848172))
- **docs:** remove VITE_SENTRY_DSN from env template ([75aac4e](https://github.com/kilip/opus/commit/75aac4ed04fcdb1b29aadc8e9fd03c5ae3bdc829))
- **docs:** add sentry dsn examples to env template ([42a0865](https://github.com/kilip/opus/commit/42a08650118359416f79516a9349a543b71621c3))

### 🧪 Tests

- **api:** add httpx Unauthorized error test ([8a8fc55](https://github.com/kilip/opus/commit/8a8fc55af6940b809e36012372a6f3e6df246ae8))
- **api:** add setup handler tests ([adae345](https://github.com/kilip/opus/commit/adae345575b7928328cbd154565968871a2297e0))
- **api:** add OTP handler tests ([a95f5eb](https://github.com/kilip/opus/commit/a95f5ebec822c7f545e4601a5d2d0bfa3e9a2361))
- **api:** test session and otp schemas on in-memory sqlite ([2951c8d](https://github.com/kilip/opus/commit/2951c8dfea500d562ded145df3c40f8698361105))

### 👷 Continuous Integration

- **tools:** run biome lint from ui directory ([2f981c5](https://github.com/kilip/opus/commit/2f981c51c4e874d2bdb66d3067f00d6e1c1e1d4e))

### 🔧 Tooling & Chores

- **release:** v0.2.0 ([a071508](https://github.com/kilip/opus/commit/a071508a73fbc13fdfcf30460edf1d1e2bbdddae))
- **deps:** bump go to 1.27.4 and x/crypto to v0.59.0 ([269942e](https://github.com/kilip/opus/commit/269942e63bde31945213a0b77db63e3635c0b1a4))
- **tools:** move biome config to ui ([850f67b](https://github.com/kilip/opus/commit/850f67bd699b8a666cf476f8f5053ebeaefb8fed))
- **ui:** regenerate api schema for oauth endpoints ([a99a13f](https://github.com/kilip/opus/commit/a99a13f702ce050a4e2d7e83e08bca44ee95dbff))
- **deps:** add golang.org/x/oauth2 ([7be5988](https://github.com/kilip/opus/commit/7be59888274b4dc0676b85eba545408de582a4dc))
- **tools:** make agy plugin subagents project-agnostic ([3688f60](https://github.com/kilip/opus/commit/3688f60d08845869f94ae35a7e3c00ccb5085247))
- **tools:** add scout recon subagent and narrow planner to read_file ([f9f7acf](https://github.com/kilip/opus/commit/f9f7acfc4591beb06c1473331a468dacc7ee9e5a))
- **tools:** require explicit /vibe trigger and drop indonesian terms ([0363257](https://github.com/kilip/opus/commit/036325782c8ffd42573e35a351972f5bdc3580db))
- **tools:** update vibe skill metadata and pr-manager mode ([7803ebf](https://github.com/kilip/opus/commit/7803ebf8413aeb43c012859483d90b8e310c58a8))
- **tools:** save planner output as artifact in builder skill ([58cbab2](https://github.com/kilip/opus/commit/58cbab21a938988a93e5fc2541caa95ccbca2bb7))
- **tools:** rename agy plugin to opus in plugin.json ([e2638e1](https://github.com/kilip/opus/commit/e2638e18402d7b4768e857595eb7b4418808b6c2))
- **tools:** set all agy subagents to flash ([2c18401](https://github.com/kilip/opus/commit/2c18401b5ff4e4627b25ffe6fb3bfc55dd5b5b6e))
- **ui:** regenerate openapi schema types ([d0e1940](https://github.com/kilip/opus/commit/d0e1940ed20ef9098de98769a2e2cf97e1717531))
- **tools:** set coder to flash and support subagents to flash_lite ([a175b39](https://github.com/kilip/opus/commit/a175b3932d25b87ebebb197441453e492b7ad3f8))
- **tools:** add agy builder plugin and subagents ([b035965](https://github.com/kilip/opus/commit/b03596592fe17c427778fd0a120bbe8687dd5b00))
- **tools:** ignored opus.db ([db1ced3](https://github.com/kilip/opus/commit/db1ced302fc0b2823a0be149b7a8b18cc13dd5ea))
- **tools:** fix ui formattng ([8934cbd](https://github.com/kilip/opus/commit/8934cbd531e57e918bd8ac3a271a30963ca34c55))
- **tools:** parse auth domain dto in swag spec ([d318ec9](https://github.com/kilip/opus/commit/d318ec96424fb28311a92a9360a10c5021c2743c))
- **api:** promote sentry-go to direct module dependency ([c6128c9](https://github.com/kilip/opus/commit/c6128c9862002fc33b78565b1ffd580a5dade169))
- **api:** regenerate OpenAPI spec after setup endpoints ([b840676](https://github.com/kilip/opus/commit/b84067619f43e0aec59efa4e7e510c78b0266470))
- **api:** regenerate OpenAPI spec after OTP handlers ([a3b49d2](https://github.com/kilip/opus/commit/a3b49d2d1c2c03e3863aed21b4d3de8bf5e66489))
- **api:** regenerate OpenAPI spec and UI types with auth endpoints ([fd2d77e](https://github.com/kilip/opus/commit/fd2d77eab5cabff964f8c96788f860bbe3ac5f61))
- **deps:** sync ui/bun.lock with package.json ([ab85ac5](https://github.com/kilip/opus/commit/ab85ac58ea640fa14247958344a53a4743b1fd01))
- **ui:** remove hardcoded sentry dsn fallback ([136a2b6](https://github.com/kilip/opus/commit/136a2b696b3437085240f20e91f140136e3c14af))
- **ui:** rely only on runtime SENTRY_DSN ([07ef5fe](https://github.com/kilip/opus/commit/07ef5fec6e241ba1c58ca41590f15cd6385d2952))
- **ui:** add runtime config.js script and entry support ([9f441b4](https://github.com/kilip/opus/commit/9f441b491bdcda8533fa398a31a833f549091bd1))
- **api:** support APP_VERSION env and dynamic config.js ([80a1425](https://github.com/kilip/opus/commit/80a142561288c40cb2685cf0409714d1f8f1dbb4))
- **ui:** configure sentry vite plugin for sourcemap uploads ([a724f7c](https://github.com/kilip/opus/commit/a724f7c8ccdc625ac32d181a11f4b91a90f3efcc))
- **deps:** add @sentry/vite-plugin dev dependency ([63f2503](https://github.com/kilip/opus/commit/63f2503bb82ba753eed0a1b87cd5d7f947ecbe07))
- **ui:** initialize sentry in react router client entry ([65adc7d](https://github.com/kilip/opus/commit/65adc7dc4a15de0195a441dfa260c62705098bbf))
- **api:** integrate sentry sdk with fiber error handling ([a2b53cd](https://github.com/kilip/opus/commit/a2b53cd6d13d69fa9d88e805e33b6df8206b2f1f))
- **deps:** add sentry sdks for go fiber and react router ([310c4ef](https://github.com/kilip/opus/commit/310c4ef219f3e804daf3b060d5b211a60896ac75))
- **deploy:** add persistent storage and DATABASE_URL for SQLite ([670cad1](https://github.com/kilip/opus/commit/670cad17f190a7982cc03ee129f9a60cf690c691))
- **tools:** remove local skills directory ([1e92052](https://github.com/kilip/opus/commit/1e920521b08566e1dc1cb004ba10699905fdaec4))
- **deps:** add ent and modernc sqlite pure go driver ([b7d15b0](https://github.com/kilip/opus/commit/b7d15b0cdea7daaf4a28e77b3f096ce6b1b17830))

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
