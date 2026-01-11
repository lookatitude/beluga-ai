# Changelog

## [1.6.1](https://github.com/lookatitude/beluga-ai/compare/v1.6.0...v1.6.1) (2026-01-11)


### Bug Fixes

* Rename shadowed 'errors' variable to 'errs' in monitoring tests ([eb7bc1b](https://github.com/lookatitude/beluga-ai/commit/eb7bc1bab26ef9a0b1a1a5c163c583d2c2392817))

## [1.6.0](https://github.com/lookatitude/beluga-ai/compare/v1.5.0...v1.6.0) (2026-01-11)


### Features

* Implement comprehensive multimodal framework ([015f0d6](https://github.com/lookatitude/beluga-ai/commit/015f0d600ee9421eec5367fa023a3160afa58f76))
* **multimodal:** Complete OpenAI and Gemini provider implementations ([4d20dc8](https://github.com/lookatitude/beluga-ai/commit/4d20dc824b1051ec1b93d0852cf3358037bbf9ac))


### Bug Fixes

* resolve integration test failures in multimodal package ([c561579](https://github.com/lookatitude/beluga-ai/commit/c56157986c0ba51910f2866c39b3e33689a1118e))
* resolve linter issues in core package ([1d570ca](https://github.com/lookatitude/beluga-ai/commit/1d570ca484da0aca0f03ad8c75f6010abd32e84a))
* **security:** address gosec security warnings ([bed4a17](https://github.com/lookatitude/beluga-ai/commit/bed4a17528951aa7eeaf35c59b1111b5b7bb4440))

## [1.5.0](https://github.com/lookatitude/beluga-ai/compare/v1.4.3...v1.5.0) (2026-01-11)


### Features

* **s2s:** add mock infrastructure for S2S provider tests ([f5481c7](https://github.com/lookatitude/beluga-ai/commit/f5481c73fe6180f1ef15eeb45fd2f0e82201e4ea))
* **s2s:** complete mock infrastructure and dependency injection refactoring ([efcfea0](https://github.com/lookatitude/beluga-ai/commit/efcfea01f604c0b0c8153e59464336d9646f335d))
* **s2s:** complete SendAudio() implementation for all providers ([d1f818c](https://github.com/lookatitude/beluga-ai/commit/d1f818cefacae0cede883532d838f0e0ea348411))
* **s2s:** complete SendAudio() implementation with optimizations and tests ([bf61b13](https://github.com/lookatitude/beluga-ai/commit/bf61b1372786bca43dca01c5ea8bb23a4b00bcbc))
* **voice/s2s:** Add Speech-to-Speech (S2S) package with multi-provider support ([9416f38](https://github.com/lookatitude/beluga-ai/commit/9416f383a7e45a9450cdf1b1f3b334fbf6d23520))
* **voice/s2s:** Add Speech-to-Speech package with multi-provider support ([dbab024](https://github.com/lookatitude/beluga-ai/commit/dbab024f3269e2d98f8c6d7514a33042453e7b15))


### Bug Fixes

* resolve import cycles in chatmodels and embeddings packages ([b3b1531](https://github.com/lookatitude/beluga-ai/commit/b3b1531829b6ab7ed3cb048bc21fcbe48a98a461))
* **security:** handle error in test helper JSON encoding ([a8888a3](https://github.com/lookatitude/beluga-ai/commit/a8888a32cf0102b01f86fd2bdbade13921b52df7))
* update integration helper to use concrete registry type ([c06b271](https://github.com/lookatitude/beluga-ai/commit/c06b2715f0adc74b84b53983bc7f477ed3a6f76a))
* **voice/s2s:** Fix go vet errors and standardize package patterns ([c7f374c](https://github.com/lookatitude/beluga-ai/commit/c7f374c429cab79d6157f54d828cfbd118b47610))

## [1.5.0](https://github.com/lookatitude/beluga-ai/compare/v1.4.0...v1.5.0) (2026-01-04)


### Features

* implement real-time voice agent support with streaming capabilities ([8544223](https://github.com/lookatitude/beluga-ai/commit/8544223d0025fc8edbb08f12e7f9e166b42bb8a8))


### Bug Fixes

* **ci:** correct boolean input defaults and add PIPESTATUS comment ([a5a661e](https://github.com/lookatitude/beluga-ai/commit/a5a661eb7e80e1b85e32668340a3e9f258f89531))
* correct golangci-lint exit status check in CI workflow ([fb00382](https://github.com/lookatitude/beluga-ai/commit/fb00382a59de0e762cb4b51c3a1fff291a1c8ce8))
* correct syntax errors in test analyzer fixture files ([64167b1](https://github.com/lookatitude/beluga-ai/commit/64167b1ddfc312eb9d4504866923cacb730d6337))
* exclude react package from all linters to prevent panic ([eb8bb05](https://github.com/lookatitude/beluga-ai/commit/eb8bb05b4edb93851cb9f80675d5774a742286f6))
* improve workflow change detection logging ([a7ab186](https://github.com/lookatitude/beluga-ai/commit/a7ab186a897450c21cb48178ad5b68d3f74124d1))
* increase package declaration check to 100 lines ([0f144e5](https://github.com/lookatitude/beluga-ai/commit/0f144e553b7100ba2aeb9ec91045c634a9f1498b))
* increase package declaration check to 50 lines ([95be08a](https://github.com/lookatitude/beluga-ai/commit/95be08a3ea4d6b697cbc0e7ed0e3fc3ce17c5b28))
* make commit step more resilient in lint workflow ([c6f7766](https://github.com/lookatitude/beluga-ai/commit/c6f7766d2825915f1e2400649b66ac21d24accac))
* make doc check a warning for initial PR ([aa104b8](https://github.com/lookatitude/beluga-ai/commit/aa104b867e36e1d9fda915729a0e564478dbb709))
* make workflow more lenient for formatting differences ([b2138ad](https://github.com/lookatitude/beluga-ai/commit/b2138adf6375c9d73c3fabaa3cb0264f619ea7ea))
* resolve all CI/CD security and linting errors ([e317fdd](https://github.com/lookatitude/beluga-ai/commit/e317fddceaba535f79debc31e54dddb326412c9f))
* resolve CI/CD linting and security issues ([ea0f3cf](https://github.com/lookatitude/beluga-ai/commit/ea0f3cf2c6d50edc166b040bb7722e4836e913a3))
* resolve remaining CI/CD errors ([6ffb2a5](https://github.com/lookatitude/beluga-ai/commit/6ffb2a50b71b893c6edf2182236551b4169745f0))
* update lint workflow to properly handle package filtering ([5869756](https://github.com/lookatitude/beluga-ai/commit/5869756b4ffc817273d92357e5da678246766de2))
* update package declaration validation to handle files with comments ([b17092f](https://github.com/lookatitude/beluga-ai/commit/b17092fd8cd8a9f0ddd55c2421b2729f6b417074))
* update workflow to properly detect doc changes in PRs ([9fdbcc5](https://github.com/lookatitude/beluga-ai/commit/9fdbcc5300535cfd6aa471a3f32ba9b08f2f8ede))
* use loop-based approach for linting to handle package paths correctly ([1d6c39d](https://github.com/lookatitude/beluga-ai/commit/1d6c39d606b131b9f539ed4e01d8eb91ebc67b01))
* use loop-based linting to properly exclude react package ([86e6038](https://github.com/lookatitude/beluga-ai/commit/86e60386e55495bd8deac9734fc9ba6a37337e8c))

## [2.0.0](https://github.com/lookatitude/beluga-ai/compare/v1.3.1...v2.0.0) (2025-11-26)


### ⚠ BREAKING CHANGES

* **voice:** This release introduces the Voice Agents feature, a major new capability for the Beluga AI framework.

### Features

* adjust GitHub workflows with manual triggers and enhanced CI/CD ([1c84903](https://github.com/lookatitude/beluga-ai/commit/1c84903c87a385ee173400e4160243e66ecd9f8d))
* **voice:** introduce Voice Agents feature with comprehensive test coverage ([fdeb7d1](https://github.com/lookatitude/beluga-ai/commit/fdeb7d13ba29ff4b7bb647464eafa86d8ac87c2e))


### Bug Fixes

* apply lint auto-fixes and format code ([6fe68f3](https://github.com/lookatitude/beluga-ai/commit/6fe68f356f8d4765ad26686f0754cff7800c06e4))
* change version field format for golangci-lint v2.0.1 ([fe01327](https://github.com/lookatitude/beluga-ai/commit/fe0132770177308c9d0cfe533dbbd27a2402e520))
* Fix GitHub workflows, coverage calculation, PR checks, and documentation generation ([cce44aa](https://github.com/lookatitude/beluga-ai/commit/cce44aae97074a9d928d789140e3bc0b3cc78710))
* migrate golangci-lint config to v2 format ([c6a9f15](https://github.com/lookatitude/beluga-ai/commit/c6a9f15cbcb3a04b2ce2fd39a47e933a2f106d90))
* remove deprecated --out-format flag from golangci-lint v2 ([e87b355](https://github.com/lookatitude/beluga-ai/commit/e87b355d72f991301f6abcd58c1da264c2c8ffef))
* remove invalid typecheck disable from golangci config ([eee9f16](https://github.com/lookatitude/beluga-ai/commit/eee9f16326495ab8c30d89ac485eb08c070218a9))
* resolve all CI pipeline issues and ensure all checks pass ([078ddf3](https://github.com/lookatitude/beluga-ai/commit/078ddf32406bfe3b7914eddf124bf30b620376fc))
* resolve all test issues and linting problems ([c3c1da0](https://github.com/lookatitude/beluga-ai/commit/c3c1da07309315e540a32ba7ea4f75dee6e9b0f4))
* resolve CI golangci-lint panic and release workflow issues ([51c504c](https://github.com/lookatitude/beluga-ai/commit/51c504c22b06fff5502a2affac0229fc98259ef6))
* set version to string format for golangci-lint v2.0.1 ([aa340ed](https://github.com/lookatitude/beluga-ai/commit/aa340ed5a93c109a3dc467218d6c64cf4d26a5f5))
* simplify golangci-lint config to v2 compatible format ([06cedfb](https://github.com/lookatitude/beluga-ai/commit/06cedfb2235b42d62ec95211b658b75c82accfe0))
* update build job to properly filter test-only packages ([c05891f](https://github.com/lookatitude/beluga-ai/commit/c05891f5e36bedb8cf8291a4180a63a9f2a9c39d))
* update golangci config for v2.0.1 compatibility ([98222e3](https://github.com/lookatitude/beluga-ai/commit/98222e32d54a598333c23db220748256659868e4))
* update golangci-lint to v2 and use v6 action ([8007264](https://github.com/lookatitude/beluga-ai/commit/8007264dd2287d40efb2d8f76b06b11fcffca8e8))
* update golangci-lint-action to v7 for golangci-lint v2 support ([a42da78](https://github.com/lookatitude/beluga-ai/commit/a42da783a9aa99499b100f7e5b659e06f7b3a08e))

## [Unreleased]

### Features

* **voice/s2s:** Add Speech-to-Speech (S2S) package with multi-provider support
  * Support for Amazon Nova 2 Sonic, Grok Voice Agent, Gemini 2.5 Flash Native Audio, and OpenAI Realtime
  * Built-in and external reasoning modes
  * Automatic provider fallback with circuit breaker pattern
  * Bidirectional streaming support
  * Full observability with OTEL metrics, tracing, and structured logging
  * Health checks for provider availability
  * Integration with voice session package as alternative to STT+TTS pipeline
  * Memory and orchestration integration for external reasoning mode

### Features

* **voice:** add comprehensive Voice Agents feature with STT, TTS, VAD, Turn Detection, Transport, Noise Cancellation, and Session Management packages
  * Speech-to-Text (STT) package with providers: Deepgram, Google Cloud, Azure Speech, OpenAI Whisper
  * Text-to-Speech (TTS) package with providers: OpenAI, Google Cloud, Azure Speech, ElevenLabs
  * Voice Activity Detection (VAD) package with providers: Silero, Energy-based, WebRTC, RNNoise
  * Turn Detection package with providers: Heuristic, ONNX
  * Transport package with providers: WebRTC, WebSocket
  * Noise Cancellation package with providers: Spectral Subtraction, RNNoise, WebRTC
  * Session Management package with complete voice interaction lifecycle management
  * Comprehensive integration tests, contract tests, and end-to-end tests
  * Performance benchmarks for all packages
  * Complete documentation and usage examples

### Testing

* **voice:** increase test coverage to 80%+ for all voice packages
  * Add WebSocket mock server utility for testing streaming sessions
  * Improve Deepgram STT coverage from 54.2% to 90.3%
  * Improve Azure STT coverage from 61.0% to 92.1%
  * Improve RNNoise noise provider coverage from 74.6% to 90.1%
  * Improve Spectral noise provider coverage from 76.1% to 95.5%
  * Improve session/internal coverage from 61.6% to 90.1%
  * Add comprehensive tests for streaming components (STT, TTS, incremental)
  * Add tests for integration components (STT, TTS, transport, VAD, turn detection)
  * Add tests for session management (interruption, response cancellation, strategy)
  * Add tests for adaptive noise profile and FFT window functions

## [1.3.1](https://github.com/lookatitude/beluga-ai/compare/v1.3.0...v1.3.1) (2025-11-23)


### Bug Fixes

* Add package declarations to all corrupted mock files ([45e92c2](https://github.com/lookatitude/beluga-ai/commit/45e92c22bbc987699002da9c75c2d60dd712aa9c))

## [1.3.0](https://github.com/lookatitude/beluga-ai/compare/v1.2.1...v1.3.0) (2025-11-07)


### Features

* implement test-analyzer tool for identifying and fixing long-running tests ([67c92d9](https://github.com/lookatitude/beluga-ai/commit/67c92d978dc5da661f5e30e8885e9ec2ec572a30))

## [1.2.1](https://github.com/lookatitude/beluga-ai/compare/v1.2.0...v1.2.1) (2025-11-07)


### Bug Fixes

* resolve context timeout test failures and fix three critical bugs ([e991d61](https://github.com/lookatitude/beluga-ai/commit/e991d611e46b2c3e6f947023500d50bc41924e1c))

## [1.2.0](https://github.com/lookatitude/beluga-ai/compare/v1.1.0...v1.2.0) (2025-11-06)


### Features

* **ci:** add production-ready CI/CD infrastructure ([b6eaa34](https://github.com/lookatitude/beluga-ai/commit/b6eaa3472baae3759a645bac866de035994a0021))


### Bug Fixes

* change gosec output format from JSON to SARIF ([2cee8d7](https://github.com/lookatitude/beluga-ai/commit/2cee8d7f80f4a879948acac707f9f6a8cd2f511f))
* **ci:** display actual coverage data in PR comments ([4ffd82b](https://github.com/lookatitude/beluga-ai/commit/4ffd82b0b173bd19c74dcd9e27b5bcea30afb0b0))
* **ci:** exclude specs folder from security scans ([a4c462b](https://github.com/lookatitude/beluga-ai/commit/a4c462b252f35c1830405bb712fb1afbe6c0c540))
* **ci:** replace non-existent golang/vuln-action with direct govulncheck installation ([a77a563](https://github.com/lookatitude/beluga-ai/commit/a77a563ad180f06917dc9176093e1f8c52805dc2))
* remove gosec SARIF upload due to invalid format ([955474d](https://github.com/lookatitude/beluga-ai/commit/955474d809127f62436fb14cea57e6fb6b34ffb5))
* resolve golangci-lint configuration conflict ([ee0c78a](https://github.com/lookatitude/beluga-ai/commit/ee0c78a90b34056e90cd8d817c7a56971d333f40))
* resolve shell variable scope issue in check-go-version target ([e45be01](https://github.com/lookatitude/beluga-ai/commit/e45be01f7d0d2aafb1a2855c4aa705e39535521e))
* set builds to empty array in goreleaser config ([ab33c08](https://github.com/lookatitude/beluga-ai/commit/ab33c0840e42129ef74668b0ec63e4dda914db6b))

## [1.1.0](https://github.com/lookatitude/beluga-ai/compare/v1.0.4...v1.1.0) (2025-10-05)


### Features

* **agents:** complete package redesign per framework guidelines ([9d706a6](https://github.com/lookatitude/beluga-ai/commit/9d706a6859c0e7c7064e81a8c30066b4b95d0b10))
* align memory package with Beluga AI Framework design patterns ([d36356e](https://github.com/lookatitude/beluga-ai/commit/d36356e504d87c803947354dc28c21ee00153ff1))
* align memory package with Beluga AI Framework design patterns ([5e434cf](https://github.com/lookatitude/beluga-ai/commit/5e434cf06cb94caaace23df99df5f21abc36ba3c))
* Complete framework standardization and comprehensive testing infrastructure ([a2174d1](https://github.com/lookatitude/beluga-ai/commit/a2174d1072c03783076ca1f7454dd5cdd098c13b))
* complete orchestration package redesign and fixes ([a7773b9](https://github.com/lookatitude/beluga-ai/commit/a7773b95ccb3fd67d10315bd5edec77394b7b819))
* comprehensive monitoring package redesign and enhancement ([4cf109e](https://github.com/lookatitude/beluga-ai/commit/4cf109eb231fba4ed77891f7e6ac5d20d2f04360))
* comprehensive monitoring package redesign and enhancement ([437ab2a](https://github.com/lookatitude/beluga-ai/commit/437ab2a1096c14ed8cb0eb05428d5ee36a2a3694))
* comprehensively redesign config package with advanced features ([a7cd839](https://github.com/lookatitude/beluga-ai/commit/a7cd83971c866c1e9cc8044ae4767d643768cb05))
* **core:** complete core package redesign as framework glue layer ([81e20f8](https://github.com/lookatitude/beluga-ai/commit/81e20f8ffd66e9d0909cfe4fd40cf37c377cd92f))
* **embeddings:** complete package redesign and fixes ([0446346](https://github.com/lookatitude/beluga-ai/commit/04463469e626fcb4387974bf81fb918d490af519))
* **embeddings:** complete package redesign and fixes ([26a0de2](https://github.com/lookatitude/beluga-ai/commit/26a0de2ce25ddeb0b3d26db0d3ecf0ddf527475c))
* **embeddings:** Implement Embedder Factory with registry pattern and decoupled interface ([ba24fe0](https://github.com/lookatitude/beluga-ai/commit/ba24fe03258bb360af3171234da1cd229d256df3))
* **embeddings:** Implement Embedder interface and MockEmbedder provider ([581af31](https://github.com/lookatitude/beluga-ai/commit/581af3193bf97db7738aeaebd3929569e9aa46d3))
* **embeddings:** Implement OpenAIEmbedder provider and tests ([8cf8636](https://github.com/lookatitude/beluga-ai/commit/8cf8636a2463f365a9b95d71893515fc034ae4f8))
* enhance chatmodels package with comprehensive improvements ([a112f81](https://github.com/lookatitude/beluga-ai/commit/a112f81a7b45ecd3d2fb4e2966c64639969f997d))
* enhance retrievers package to align with Beluga AI Framework design patterns ([f6dd697](https://github.com/lookatitude/beluga-ai/commit/f6dd6973d6bc0f11dc36eb7d4123c869121e3c17))
* **llms:** complete package redesign and fix all tests ([235faea](https://github.com/lookatitude/beluga-ai/commit/235faeac77cdf3bcfba58b9db571a0840b30a576))


### Bug Fixes

* resolve linter errors in agents and vectorstores packages ([5059b8b](https://github.com/lookatitude/beluga-ai/commit/5059b8be982b4bc0cb5ab1816207a391efbb6896))

## [1.0.4](https://github.com/lookatitude/beluga-ai/compare/v1.0.3...v1.0.4) (2025-05-07)


### Bug Fixes

* updated libraries ([2679ca3](https://github.com/lookatitude/beluga-ai/commit/2679ca3d10fbf66ff082657b6bfae7236fa9fce6))

## [1.0.3](https://github.com/lookatitude/beluga-ai/compare/v1.0.2...v1.0.3) (2025-05-07)


### Bug Fixes

* **agents:** resolve undefined symbols in agent example ([4c685b5](https://github.com/lookatitude/beluga-ai/commit/4c685b5874ee9a6a468e0ae5dbc6c4db9bf6be4c))
* **agents:** resolve undefined symbols in agent example ([07cb007](https://github.com/lookatitude/beluga-ai/commit/07cb007103ff80679f721375c7eb6157bde6a06f))
* Corrected syntax error in switch/case block in Anthropic module ([8c2e97f](https://github.com/lookatitude/beluga-ai/commit/8c2e97f4c0c51dfa843baafb2b8b59c256e449e7))
* Corrected System prompt assignment in Anthropic Generate method ([9ac02d5](https://github.com/lookatitude/beluga-ai/commit/9ac02d5a1bd7f3535bf2f87eaee0bf1564abde7b))
* Corrected System prompt assignment in Anthropic StreamChat method ([ddbd4df](https://github.com/lookatitude/beluga-ai/commit/ddbd4dfe4084fcec493cd5f756b31c89599ffdf9))
* Corrected ToolChoiceAny and ToolChoiceTool usage in Anthropic module ([7848ae0](https://github.com/lookatitude/beluga-ai/commit/7848ae0bf797ad83626033a95250cabab3de40c6))
* Corrected ToolChoiceAuto usage in Anthropic Generate method (second instance) ([cc3163b](https://github.com/lookatitude/beluga-ai/commit/cc3163b942a85ea227a4bda8b40c27fc1af35a4a))
* Corrected ToolChoiceAuto usage in Anthropic module ([121f0e0](https://github.com/lookatitude/beluga-ai/commit/121f0e0309a782a27dc0e56807d767ead95e3de3))
* Modified StreamChat function signature in Anthropic module ([163da2e](https://github.com/lookatitude/beluga-ai/commit/163da2ed29b08e57358ebcaa5ed2169e30510145))
* Modify StreamChat signature (remove ctx) to isolate syntax error ([471f021](https://github.com/lookatitude/beluga-ai/commit/471f021b2fc766ef87cc010692db7166a583edab))
* Removed 'LATEST' literal from switch/case in Anthropic module ([8c2de7a](https://github.com/lookatitude/beluga-ai/commit/8c2de7aebc5b497fad3debb41e2b95169ba9a9ff))
* resolve compilation errors in multiple packages ([0522041](https://github.com/lookatitude/beluga-ai/commit/05220411405a280e4c824c0827ad74e2f13d65d8))
* Restore correct StreamChat function signature in Anthropic module ([75935ac](https://github.com/lookatitude/beluga-ai/commit/75935ac69afc76066787711eb4d93c3e4f5e2ab1))
* Restore full StreamChat function signature in Anthropic module ([5e8cbc8](https://github.com/lookatitude/beluga-ai/commit/5e8cbc8123fd3dd525dc0bd1010bf11d2483423b))
* Restore options parameter in StreamChat function signature in Anthropic module ([1a0e0b9](https://github.com/lookatitude/beluga-ai/commit/1a0e0b97905bde8e8ce5db1d790276bdac702da3))

## [1.0.2](https://github.com/lookatitude/beluga-ai/compare/v1.0.1...v1.0.2) (2025-05-06)


### Bug Fixes

* address multiple Go compilation errors ([cdb9bdc](https://github.com/lookatitude/beluga-ai/commit/cdb9bdc51c2e95071b866520a55576dc06e52df4))

## [1.0.1](https://github.com/lookatitude/beluga-ai/compare/v1.0.0...v1.0.1) (2025-05-06)


### Bug Fixes

* correct broken link in API index ([5dda4e8](https://github.com/lookatitude/beluga-ai/commit/5dda4e87e7d8b937c688ff5c0879bb5cc9b55e95))
* correct import path for ollama embedder in RAG example ([21ffe00](https://github.com/lookatitude/beluga-ai/commit/21ffe00168ec77b5a516536aeaad7a1f3c22d3b7))
* remove invalid link property from API docs category metadata ([21dbd2a](https://github.com/lookatitude/beluga-ai/commit/21dbd2a72cfc355dc15d089805856c33f1c62775))

## 1.0.0 (2025-05-06)


### Features

* add CI, release-please workflows and contributing guidelines ([0fa8a6f](https://github.com/lookatitude/beluga-ai/commit/0fa8a6f6834c117bc3f1032d8e277c84031a645d))
* add project logo and update website theme ([e1674e5](https://github.com/lookatitude/beluga-ai/commit/e1674e561772485b4053f24774ae15bba7d8e014))
* implement Bedrock Mistral and Meta, update docs and examples ([14de05d](https://github.com/lookatitude/beluga-ai/commit/14de05dffabd2eba5c8ce51d791b7285c619ab85))
* initial project structure and core functionalities ([dcbe9bd](https://github.com/lookatitude/beluga-ai/commit/dcbe9bda354eae1285990097f381ef3f96b9009e))
* setup Docusaurus website, API docs, and release process for alpha ([71c1f74](https://github.com/lookatitude/beluga-ai/commit/71c1f746a692ba998f8b953727cb0ba930470de3))


### Bug Fixes

* update release-please workflow permissions ([9b11b96](https://github.com/lookatitude/beluga-ai/commit/9b11b96fcc362823004dfee4f4828d6c07915624))
