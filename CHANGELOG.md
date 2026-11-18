# Changelog

All notable changes to this project will be documented in this file.

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
