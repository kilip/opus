# Changelog

All notable changes to this project will be documented in this file.

## [v0.3.1-rc3](https://github.com/kilip/opus/compare/v0.3.0-rc3...v0.3.1-rc3) (2026-11-20)

### 🐛 Bug Fixes

- **api:** allow inline bootstrap scripts in CSP ([877e85d](https://github.com/kilip/opus/commit/877e85d6a4611d23cdb83a04763fab39b87f2379))
- **api:** log 4xx request errors at warn level ([09672e1](https://github.com/kilip/opus/commit/09672e13d9957c1a5b82f556d11f151435f21053))

## [v0.3.0-rc3](https://github.com/kilip/opus/compare/v0.3.0-rc2...v0.3.0-rc3) (2026-11-19)

### ✨ Features

- **api:** sync mounted agy credentials into HOME on startup ([ed5aee4](https://github.com/kilip/opus/commit/ed5aee486bdfa2c20cf09416120f7db14a35c894))
- **api:** add opus doctor diagnostics command ([6824d15](https://github.com/kilip/opus/commit/6824d15471b0cdb80512de6aa1e97716b625df71))
- **api:** add opus service lifecycle command ([b19e305](https://github.com/kilip/opus/commit/b19e3057b6903acef0466e9cf78ed1569de40758))
- **ui:** add Settings page for GitHub token management ([98268d7](https://github.com/kilip/opus/commit/98268d71ed0c248a9d6a41f89f93c67e9db108a5))
- **api:** add encrypted GitHub token settings ([70dd80a](https://github.com/kilip/opus/commit/70dd80a886ecdf76e97646388d0975a087007389))
- **api:** add opus self-update command ([9041e9b](https://github.com/kilip/opus/commit/9041e9b0bdd2bcb68ff3f806bd5720c8e1cde659))
- **api:** add opus uninstall command ([bb224fc](https://github.com/kilip/opus/commit/bb224fccceb13e310c39a8b3891e03cac750e093))
- **ui:** add node edit dialog, network info, and Wake-on-LAN action ([b6715b9](https://github.com/kilip/opus/commit/b6715b9252bbbef30af589afea8bd0464de07d61))
- **api:** add Wake-on-LAN endpoint POST /api/nodes/{id}/wake ([6da2ce5](https://github.com/kilip/opus/commit/6da2ce50ce6780adbdbcd413b2ce1cc2968da24d))
- **api:** report active hostname, IP, and MAC on worker join and hello ([504f07b](https://github.com/kilip/opus/commit/504f07baf7de5a253e21a48ddd09b7e1393231a2))
- **api:** add hostname, IP, MAC fields and PATCH /api/nodes/{id} ([84c1966](https://github.com/kilip/opus/commit/84c196677f400a3331b38018ab8624328ffee2d0))
- **api:** forward slog error logs to sentry ([ca39f30](https://github.com/kilip/opus/commit/ca39f3038bd15c4de8233e23bf99e4a85669a60d))

### 🐛 Bug Fixes

- **api:** propagate context, WorkingDirectory, and launchd start lifecycle ([0512c5f](https://github.com/kilip/opus/commit/0512c5fe10e334fb905001cf185e82e0c4e06d6d))
- **api:** harden git clone URLs, join-token, and project validation ([a44035e](https://github.com/kilip/opus/commit/a44035e7b85b4d9e1ec4afe16e2a37be65ca5787))
- **api:** add SSE keepalive, session Close, and shutdown order ([95738b4](https://github.com/kilip/opus/commit/95738b44705aaf7fa8212122c5c67d9ce8abd331))
- **api:** add /api/readyz, helmet security headers, and SPA 404 ([0728def](https://github.com/kilip/opus/commit/0728def631517c5a405eb93392aa73e00818b346))
- **api:** enable SQLite WAL, pooling, and schema version guard ([43532c3](https://github.com/kilip/opus/commit/43532c365437da1dffd270fd42b031c5255e9c00))
- **api:** fail runs on dispatch error, session failure, and node grace ([e083690](https://github.com/kilip/opus/commit/e083690531b09711e11cead307a9697ad7491086))
- **api:** expose session LatestEventTime for activity staleness ([8b36972](https://github.com/kilip/opus/commit/8b36972ed29a06b70ae7e816042e998acd5e0863))
- **api:** harden MCP rate limiter and worker policy ([687a678](https://github.com/kilip/opus/commit/687a678413c6ec4cf4645383db8cbfc6811b2f32))
- **api:** enforce WebSocket limits, keepalive, and origins ([95b8b53](https://github.com/kilip/opus/commit/95b8b53244cffcc1fafe63f7af5a3ab0bc836a13))
- **api:** verify node ownership in session worker events ([98c1a73](https://github.com/kilip/opus/commit/98c1a73ce7fd9f01a54f0d484e03002a5a1768b5))
- **api:** add run-scoped MCP token and handler scoping ([6e555bc](https://github.com/kilip/opus/commit/6e555bc0b2d31fa98167cf4d374e190fb9db25cf))
- **ui:** cap SessionLogStream entries and reconnect SSE with backoff ([7dfc7eb](https://github.com/kilip/opus/commit/7dfc7eb771f298c9d2f098d39758d2d43f76c194))
- **ui:** add shell ErrorBoundary, public 404, and dashboard error state ([5fe618f](https://github.com/kilip/opus/commit/5fe618f46d77690865c70533cce0a9ee6c4fd730))
- **ui:** add accessible Dialog, ErrorState, and modal migrations ([f55b079](https://github.com/kilip/opus/commit/f55b0797dc751bb759c48ce4f551424ce3d73289))
- **ui:** harden auth redirect, 401 navigation, and safe next URLs ([116f8f2](https://github.com/kilip/opus/commit/116f8f2cdbf4a56f9e899b390c9322762564027a))
- **ui:** disable unrequested sourcemaps and strip .map files ([30976b0](https://github.com/kilip/opus/commit/30976b00b75ed72f119c04245cb5a7bc239e1197))
- **ui:** add setup token field and X-Setup-Token header to onboarding ([2c3535f](https://github.com/kilip/opus/commit/2c3535f8bbe5e604d7b60fc1d6f39df3b5bdba1f))
- **api:** harden setup token, atomic OTP, proxy trust, secret, and mailer ([23d9979](https://github.com/kilip/opus/commit/23d9979562dfd302033317106fcadcb312e26461))
- **api:** implement worker start_run runner, session events, and cleanup ([c6e59db](https://github.com/kilip/opus/commit/c6e59dbe27af96400532c54e96bcd028f29e79b9))
- **api:** wire run pipeline, HubTransport, tracker bus, and stage timeout ([ee297ac](https://github.com/kilip/opus/commit/ee297ac4e268d02a7d12102df6272a2f05327ee1))
- **api:** skip github auth check when no github projects exist ([9c8e1be](https://github.com/kilip/opus/commit/9c8e1beaff300f2ffcdf7ed40902acfd1d7b466a))

### 📦 Dependencies

- **deps:** keep ent codegen sums in go.sum ([a4db42b](https://github.com/kilip/opus/commit/a4db42b5869645a57bf84d1569e27f3cf58f239e))

## [v0.3.0-rc2](https://github.com/kilip/opus/compare/v0.3.0-rc1...v0.3.0-rc2) (2026-11-18)

### ✨ Features

- **api:** add /vibe skill and Opus MCP config to agy plugin ([ea7a2bd](https://github.com/kilip/opus/commit/ea7a2bd228cd7a6862bff7acb0cc35555f0e0d10))
- **api:** inject Opus MCP config into agy sessions ([bfba0c1](https://github.com/kilip/opus/commit/bfba0c16905db60cf73665d2a6f62ada9e764ac2))
- **api:** add Streamable HTTP POST /mcp endpoint ([749be79](https://github.com/kilip/opus/commit/749be792618872ccfa1183297f1204a5b58601d6))
- **ui:** show agent tool calls and report in runs ([6c0de90](https://github.com/kilip/opus/commit/6c0de909b7af4dbd06fa1747e17c4939bf50d906))
- **api:** add agent feature and MCP tool service ([7c34dbd](https://github.com/kilip/opus/commit/7c34dbd8e9e760a7ad816816768444b48c619b88))
- **api:** add end-to-end vibe run smoke test ([227b5aa](https://github.com/kilip/opus/commit/227b5aa7eafd824b97cc004086ddf55c58b3753e))
- **ui:** add dashboard overview page ([35c20cb](https://github.com/kilip/opus/commit/35c20cb939974cfee3150ad10950ce2b96b04362))
- **ui:** add runs list and detail page ([9ec106f](https://github.com/kilip/opus/commit/9ec106f4630045c1be5e757ce03b57e54ed00512))
- **api:** add run failure, pause, cancel, recovery, and API ([8de5141](https://github.com/kilip/opus/commit/8de51415c331a5f36b623482bc3574bc87e3da11))
- **api:** add claim, prepare, run session, and cleanup stages ([8bc746d](https://github.com/kilip/opus/commit/8bc746df47d8ef1771c3cf9d99981272177ffda4))
- **api:** add policy spec and enforcer for ACP ([1702a4c](https://github.com/kilip/opus/commit/1702a4ce3f8f7570f5845e749d398360e6ab6c50))
- **api:** add run prompt builder ([2bff2ba](https://github.com/kilip/opus/commit/2bff2ba6c30b52ab32d3ebef8d0db51c1f51d293))
- **api:** add run eligibility checks and reconciler ([81bd11b](https://github.com/kilip/opus/commit/81bd11bc458e2374a432670d4f56f901bed365c1))
- **api:** add Telegram notifier for run and node events ([7c301ee](https://github.com/kilip/opus/commit/7c301ee3df7752e4745e0145be82e47cafefa922))
- **api:** add Run entity, state machine, and events ([ee8157b](https://github.com/kilip/opus/commit/ee8157b394a0a22f1b61717de6330323282917dd))
- **api:** add session records, SSE stream, and FIFO queue ([b1cfb42](https://github.com/kilip/opus/commit/b1cfb42ec5e443967612c74e014963cac09d1fca))
- **api:** add agy ACP stdio client ([62fca50](https://github.com/kilip/opus/commit/62fca50c2737907cb42d97dfbfdc87132fb312ca))
- **api:** add worker git clone and worktree operations ([c0ba757](https://github.com/kilip/opus/commit/c0ba75772475f30474a07c040029b83e6fdc94b0))
- **api:** add worker runtime skeleton and capability reporting ([c3bfe81](https://github.com/kilip/opus/commit/c3bfe81519639ea5f7d15a4770d506c86ad713ba))
- **ui:** add GitHub context to tasks and projects ([c82930a](https://github.com/kilip/opus/commit/c82930a4f26bc5444ccbe28c1fd8a669d72e4272))
- **api:** add GitHub outbound tracker sync ([3fc9650](https://github.com/kilip/opus/commit/3fc9650e83240684d518f4996bc2d72b1505a8a3))
- **api:** add GitHub pull request tracking ([3b4ae7a](https://github.com/kilip/opus/commit/3b4ae7acb21aeaddd7f12eda1c8cc369de75c279))
- **api:** add cursor-based inbound GitHub issue sync ([8b089f3](https://github.com/kilip/opus/commit/8b089f3c9b3ab261cc666f1ccaef8bf246935ddd))
- **api:** add synced_at field and import tracking to task ([503dbb5](https://github.com/kilip/opus/commit/503dbb5e771727b28b01f8cbb1f6c811c9f22240))
- **api:** add tracker feature skeleton, sync state, and gh runner ([67a8048](https://github.com/kilip/opus/commit/67a8048632db9565ed9864e49ff9eee3af50ff6c))
- **ui:** add blocked stage and filter to tasks feature ([6c81360](https://github.com/kilip/opus/commit/6c81360da3235537fc3aae88e5e688fda2540a94))
- **api:** expose task service methods and add blocked status ([56f476b](https://github.com/kilip/opus/commit/56f476b3e6aa7988603fac8f28e78fdb27875c2b))
- **ui:** add project picker and filter to tasks ([6f97e58](https://github.com/kilip/opus/commit/6f97e58836ce668a90368f15e5e05ff379aa5f11))
- **api:** require projectId on tasks and enforce project delete guard ([85b05ec](https://github.com/kilip/opus/commit/85b05ec1a5da0f479ea286a169e719525c3bc638))
- **ui:** add projects page ([50680cb](https://github.com/kilip/opus/commit/50680cb1c13d1adf474dc1217600fe0e01b00627))
- **api:** add per-node path mapping to project feature ([734eea1](https://github.com/kilip/opus/commit/734eea1ce356e8b06eb1035057c18f07a7a7d9ac))
- **api:** add project Ent schema, feature CRUD, and tests ([690a851](https://github.com/kilip/opus/commit/690a851a7ea421f20d4c9215023e72baa16234aa))
- **ui:** add nodes page ([be9adb3](https://github.com/kilip/opus/commit/be9adb313af0b2400429b38c9d8c3f3b77b3b15f))
- **api:** add delete node endpoint ([eacec04](https://github.com/kilip/opus/commit/eacec04eddc8609b9adb959d6135f2a035fd335b))
- **api:** add opus join and opus worker commands ([ee6e4a3](https://github.com/kilip/opus/commit/ee6e4a3e1d440b516d5bbffca3f30f6dc455d500))
- **api:** add worker control channel client and in-process master path ([80a9503](https://github.com/kilip/opus/commit/80a95035c79b2a388cf801a71e5d858aaf06370e))
- **api:** add node heartbeat and offline detection ([34ccf79](https://github.com/kilip/opus/commit/34ccf798935ca53ba2b2bdcfa8942cce04f8a6f8))
- **api:** add node control channel server and hub ([8c9f915](https://github.com/kilip/opus/commit/8c9f915a860cde1236f27c6521864d3b9517dd48))
- **api:** add join exchange endpoint for worker nodes ([2ae62ad](https://github.com/kilip/opus/commit/2ae62ad862d918cd915f4460c8ed459eb532fa9c))
- **api:** add node join token generation and consumption ([a7a8a53](https://github.com/kilip/opus/commit/a7a8a537676670484f1ff8a1364b92d3202da022))
- **api:** add node registry and master self-registration ([9e03ee5](https://github.com/kilip/opus/commit/9e03ee5274464bbd8a498c3a84478840fe6aa7a7))

### 📦 Dependencies

- **deps:** add modelcontextprotocol go-sdk ([27ce89f](https://github.com/kilip/opus/commit/27ce89ff7fd33282ca5d12816decc06fb894421e))
- **deps:** add gofiber contrib websocket v3 dependency ([3cd8431](https://github.com/kilip/opus/commit/3cd843145fa2a1353d8b74326734b8edaae5de89))

### ♻️ Code Refactoring

- **ui:** update task status in-progress in UI model and schema ([32145ee](https://github.com/kilip/opus/commit/32145ee5529f4b72b079c8e8ad631c3699a8a5c1))
- **api:** rename task status in_progress to in-progress ([7a2aef3](https://github.com/kilip/opus/commit/7a2aef30ea338650783ef50c0e5263eaa75a777b))

### 📝 Documentation

- **docs:** add project docs, env example, and update AGENTS.md ([afe7bc0](https://github.com/kilip/opus/commit/afe7bc06236d7d236fae5e127c3fec28701aff9f))
- **docs:** document internal/agent and extensions/agy boundaries ([b6a7915](https://github.com/kilip/opus/commit/b6a7915d2e1e1cd8e705f7b3d56be6938c8fbffd))
- **docs:** update AGENTS.md and operations notes ([1be3361](https://github.com/kilip/opus/commit/1be3361e0af2839e956324e1f68c922d3c0ea88b))
- **docs:** spike and document agy acp behavior ([b2d12a8](https://github.com/kilip/opus/commit/b2d12a8fceaa479842323714b10d9a3dd825d402))
- **docs:** document TRACKER_POLL_INTERVAL in AGENTS.md ([f68db36](https://github.com/kilip/opus/commit/f68db36630c9929d1deb8a35f0f526ff124a36cf))

### 🧪 Tests

- **api:** standardize test helper env vars ([ee0cf84](https://github.com/kilip/opus/commit/ee0cf845d07ce3730cd3e427f12173eb48ceba45))
- **api:** add end-to-end smoke test for agent MCP ([84ca256](https://github.com/kilip/opus/commit/84ca2567edd5371707ee27b71bcdbc42f07fa06c))

### 👷 Continuous Integration

- **tools:** sync issue status labels to github project ([0e39115](https://github.com/kilip/opus/commit/0e3911500177179894b09d0b78c9de695d1d2969))

### 🔧 Tooling & Chores

- **release:** v0.3.0-rc2 ([cc15ce8](https://github.com/kilip/opus/commit/cc15ce8fa863d8c66f18e4bb18db955e764177bb))
- **release:** prepare 0.3.0-rc2 ([26b30e9](https://github.com/kilip/opus/commit/26b30e958b09cfa4e27a8e81642f447c323e0a86))
- **ui:** format ui files with editorconfig 2-space indent ([8a68d23](https://github.com/kilip/opus/commit/8a68d23d3fe63fff891dfcec1b1c2f6549ecfe32))
- **tools:** move biome config to repo root for editorconfig ([b8df225](https://github.com/kilip/opus/commit/b8df2259fe2fb2c85cd8f3ee35d66fb51defa037))
- **ui:** standardize vite env vars to APP_* prefix ([5726df5](https://github.com/kilip/opus/commit/5726df590463419ce752143b58a6367a1243e7ac))
- **tools:** enforce feature boundaries in golangci-lint ([e865559](https://github.com/kilip/opus/commit/e86555972affa9adee6971a4fd3ee5237a5b1faa))
- **deps:** add modelcontextprotocol go-sdk ([27ce89f](https://github.com/kilip/opus/commit/27ce89ff7fd33282ca5d12816decc06fb894421e))
- **tools:** include run handler and domain in swag init ([4bd30df](https://github.com/kilip/opus/commit/4bd30df226c4b90b55363fdd3f61784f5280526b))
- **tools:** add session packages to swag init ([9aeb756](https://github.com/kilip/opus/commit/9aeb7561840a3b4a01b9c411640973a7198f8b2e))
- **tools:** remove agy agents from extension ([ec1ffa4](https://github.com/kilip/opus/commit/ec1ffa489d2e9172176fc15780647f28070ba3a7))
- **tools:** remove agy skills from extension ([dbb40e5](https://github.com/kilip/opus/commit/dbb40e528f3fbaf3d05486a390b2c4e8b86c594b))
- **ui:** regenerate API schema types for task syncedAt ([85fd44a](https://github.com/kilip/opus/commit/85fd44a637f7abff9b2715604d237d0d10da1064))
- **tools:** add tracker packages to api-spec ([6ae0c06](https://github.com/kilip/opus/commit/6ae0c06eca6104041de9a1a172c5cad183e40a72))
- **tools:** add project packages to api-spec ([fb00164](https://github.com/kilip/opus/commit/fb001645a6d9b67bb2fca0de07b00e15e667965a))
- **api:** regenerate openapi spec and ui schema ([dbb4026](https://github.com/kilip/opus/commit/dbb40261a9b18acd899f89715ae4234528f7b18b))
- **api:** regenerate openapi spec and ui schema ([3b1d9dd](https://github.com/kilip/opus/commit/3b1d9dd2f7da6fd5ca2652aa6aaa23aecf838cb7))
- **deps:** add gofiber contrib websocket v3 dependency ([3cd8431](https://github.com/kilip/opus/commit/3cd843145fa2a1353d8b74326734b8edaae5de89))
- **api:** regenerate openapi spec and ui schema ([46116c2](https://github.com/kilip/opus/commit/46116c2e274da6a8c8a5f43f8218783fa95d8155))
- **tools:** rename vibe skill to express ([3cf192d](https://github.com/kilip/opus/commit/3cf192d42a8d08eadb853134a64d9713296beaeb))
- **ui:** regenerate openapi schema types for join tokens ([71e4770](https://github.com/kilip/opus/commit/71e47702d025061365b4dcb994eda1f5d21a461f))
- **tools:** add node feature to swag init dirs ([7aa60f9](https://github.com/kilip/opus/commit/7aa60f96ab1a6bc8650523be51c24c1f98b948b4))
- **tools:** use project status script in vibe and pr-manager ([3725e89](https://github.com/kilip/opus/commit/3725e89f95f6babef55e01ab264a400dc993439e))
- **tools:** add 1-line phase progress status to vibe skill ([e285319](https://github.com/kilip/opus/commit/e2853192b8a289f66377a377ae383c7637e138de))
- **tools:** enforce in-place checkbox sync via sed in verifier ([e229791](https://github.com/kilip/opus/commit/e2297919054063704b9a3979fd48a271b5420d36))
- **tools:** delegate vibe notification format to telegram skill ([1d3d0a4](https://github.com/kilip/opus/commit/1d3d0a480b8ec6f20845ac2b8e96ec933f11d6dc))
- **tools:** force IPv4 and retry Bot.initialize in send.py ([92ac947](https://github.com/kilip/opus/commit/92ac947eac04328404eec65ec774127a2677343f))

## [v0.3.0-rc1](https://github.com/kilip/opus/compare/v0.2.0...v0.3.0-rc1) (2026-11-16)

### ✨ Features

- **ui:** add empty state to tasks page ([1bf39ba](https://github.com/kilip/opus/commit/1bf39ba1fca76c35b7e9a688358c3dbd5cf85d97))
- **ui:** make activity author optional in Chatter and task actions ([cad893a](https://github.com/kilip/opus/commit/cad893a3b071dd128399ae57d3438763e23d2260))
- **api:** switch task feature to Ent repo and remove seed data ([fc83fe0](https://github.com/kilip/opus/commit/fc83fe0cdaf0cc794b0c3020841d3dfe158f8cd5))

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

## [v0.1.0](https://github.com/kilip/opus/releases/tag/v0.1.0) (2026-11-14)

### ✨ Features

- **ui:** add api envelope client, task mutations, and update shared utils ([8a828c5](https://github.com/kilip/opus/commit/8a828c53a59429a45e41bcc1c246500f49a975f9))
- **api:** establish modular monolith core, httpx envelope, and task module ([5e492ec](https://github.com/kilip/opus/commit/5e492ec21cb06596ff598ef9dfdec32d9da4d8ec))

### 📦 Dependencies

- **deps:** track go.sum and remove from .gitignore ([00ab60e](https://github.com/kilip/opus/commit/00ab60e7941320ec06ac812a865a9071b7319711))
- **deps:** add github.com/sethvargo/go-envconfig dependency ([948dd58](https://github.com/kilip/opus/commit/948dd5868b40907bab8538eee3e923c0419d8e59))
- **deps:** add renovate configuration ([ebdc1f6](https://github.com/kilip/opus/commit/ebdc1f6ff764b15401f3e5cb7da5b66e9de7abc8))
