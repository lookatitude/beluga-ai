# Tasks: Voice Agents

**Input**: Design documents from `/specs/004-feature-voice-agents/`
**Prerequisites**: plan.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → ✅ Loaded: Implementation plan with 7 packages, 65-85 tasks estimated
2. Load optional design documents:
   → ✅ data-model.md: 8 entities extracted → model tasks
   → ✅ contracts/: VoiceSession API contract → contract test tasks
   → ✅ research.md: Technical decisions → setup and integration tasks
   → ✅ quickstart.md: Test scenarios → integration test tasks
3. Generate tasks by category:
   → Setup: project structure, dependencies, linting
   → Tests: contract tests, integration tests (TDD)
   → Core: interfaces, configs, errors, metrics, registries
   → Providers: STT, TTS, VAD, Turn Detection, Transport, Noise
   → Session: lifecycle, state management, clarification features
   → Integration: cross-package integration, end-to-end
   → Polish: benchmarks, documentation, examples
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness:
   → All contracts have tests? ✅
   → All entities have models? ✅
   → All providers implemented? ✅
   → All clarification features implemented? ✅
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Package root**: `pkg/voice/`
- **Sub-packages**: `pkg/voice/{stt,tts,vad,turndetection,transport,session,noise}/`
- **Interfaces**: `pkg/voice/iface/` (shared) and `pkg/voice/{package}/iface/` (package-specific)
- **Providers**: `pkg/voice/{package}/providers/{provider_name}/`
- **Tests**: `pkg/voice/{package}/*_test.go`

---

## Phase 3.1: Setup & Project Structure

- [X] **T001** Create package directory structure for pkg/voice/ with all sub-packages (stt, tts, vad, turndetection, transport, session, noise)
- [X] **T002** [P] Create iface/ directory structure in pkg/voice/ for shared interfaces
- [X] **T003** [P] Create internal/ directory structure in pkg/voice/ for shared utilities (audio/, utils/)
- [X] **T004** [P] Create providers/ directory structure in pkg/voice/ for provider organization
- [X] **T005** Add voice package dependencies to go.mod (pion/webrtc, onnxruntime-go if needed)
- [X] **T006** [P] Configure linting rules for voice package in .golangci.yml
- [X] **T007** [P] Add voice package to CI/CD workflows for testing

---

## Phase 3.2: Core Infrastructure - Shared Interfaces & Types

- [X] **T008** [P] Create pkg/voice/iface/stt.go with STTProvider interface definition
- [X] **T009** [P] Create pkg/voice/iface/tts.go with TTSProvider interface definition
- [X] **T010** [P] Create pkg/voice/iface/vad.go with VADProvider interface definition
- [X] **T011** [P] Create pkg/voice/iface/turndetection.go with TurnDetector interface definition
- [X] **T012** [P] Create pkg/voice/iface/transport.go with Transport interface definition
- [X] **T013** [P] Create pkg/voice/iface/session.go with VoiceSession interface definition
- [X] **T014** [P] Create pkg/voice/iface/noise.go with NoiseCancellation interface definition
- [X] **T015** [P] Create pkg/voice/internal/audio/format.go with AudioFormat struct and validation
- [X] **T016** [P] Create pkg/voice/internal/audio/codec.go with audio codec utilities
- [X] **T017** [P] Create pkg/voice/internal/audio/converter.go with audio format conversion utilities
- [X] **T018** [P] Create pkg/voice/internal/utils/retry.go with retry logic utilities
- [X] **T019** [P] Create pkg/voice/internal/utils/circuitbreaker.go with circuit breaker utilities
- [X] **T020** [P] Create pkg/voice/internal/utils/ratelimit.go with rate limiting utilities

---

## Phase 3.3: STT Package - Foundation (TDD)

### STT Package Structure & Tests First
- [X] **T021** [P] Create pkg/voice/stt/iface/stt.go with STTProvider interface (package-specific)
- [X] **T022** [P] Create pkg/voice/stt/test_utils.go with AdvancedMockSTTProvider and testing utilities
- [X] **T023** [P] Create pkg/voice/stt/advanced_test.go with table-driven tests for STTProvider interface
- [X] **T024** [P] Create pkg/voice/stt/config_test.go with tests for STT configuration validation
- [X] **T025** [P] Create pkg/voice/stt/errors_test.go with tests for STT error handling
- [X] **T026** [P] Create pkg/voice/stt/metrics_test.go with tests for STT metrics
- [X] **T027** [P] Create pkg/voice/stt/registry_test.go with tests for STT provider registry

### STT Package Core Implementation
- [X] **T028** [P] Create pkg/voice/stt/config.go with STTConfig struct, validation, and functional options
- [X] **T029** [P] Create pkg/voice/stt/errors.go with STTError type and error codes (Op/Err/Code pattern)
- [X] **T030** [P] Create pkg/voice/stt/metrics.go with OTEL metrics implementation (NewMetrics pattern)
- [X] **T031** [P] Create pkg/voice/stt/registry.go with global provider registry pattern
- [X] **T032** [P] Create pkg/voice/stt/stt.go with factory function NewProvider and base types

### STT Package - Deepgram Provider
- [X] **T033** [P] Create pkg/voice/stt/providers/deepgram/config.go with DeepgramConfig struct
- [X] **T034** [P] Create pkg/voice/stt/providers/deepgram/provider.go with DeepgramProvider implementation
- [X] **T035** [P] Create pkg/voice/stt/providers/deepgram/websocket.go with WebSocket streaming implementation
- [X] **T036** [P] Create pkg/voice/stt/providers/deepgram/rest.go with REST API fallback implementation
- [X] **T037** [P] Create pkg/voice/stt/providers/deepgram/provider_test.go with Deepgram provider tests
- [X] **T038** [P] Create pkg/voice/stt/providers/deepgram/websocket_test.go with WebSocket streaming tests
- [X] **T039** Register Deepgram provider in pkg/voice/stt/registry.go

### STT Package - Google Cloud Provider
- [X] **T040** [P] Create pkg/voice/stt/providers/google/config.go with GoogleConfig struct
- [X] **T041** [P] Create pkg/voice/stt/providers/google/provider.go with GoogleProvider implementation
- [X] **T042** [P] Create pkg/voice/stt/providers/google/streaming.go with StreamingRecognize implementation
- [X] **T043** [P] Create pkg/voice/stt/providers/google/provider_test.go with Google provider tests
- [X] **T044** Register Google provider in pkg/voice/stt/registry.go

### STT Package - Azure Provider
- [X] **T045** [P] Create pkg/voice/stt/providers/azure/config.go with AzureConfig struct
- [X] **T046** [P] Create pkg/voice/stt/providers/azure/provider.go with AzureProvider implementation
- [X] **T047** [P] Create pkg/voice/stt/providers/azure/streaming.go with Azure Speech SDK integration
- [X] **T048** [P] Create pkg/voice/stt/providers/azure/provider_test.go with Azure provider tests
- [X] **T049** Register Azure provider in pkg/voice/stt/registry.go

### STT Package - OpenAI Whisper Provider
- [X] **T050** [P] Create pkg/voice/stt/providers/openai/config.go with OpenAIConfig struct
- [X] **T051** [P] Create pkg/voice/stt/providers/openai/provider.go with OpenAIProvider implementation
- [X] **T052** [P] Create pkg/voice/stt/providers/openai/provider_test.go with OpenAI provider tests
- [X] **T053** Register OpenAI provider in pkg/voice/stt/registry.go

### STT Package - Integration & Polish
- [X] **T054** Create pkg/voice/stt/README.md with STT package documentation
- [X] **T055** [P] Create pkg/voice/stt/benchmarks_test.go with performance benchmarks
- [X] **T056** [P] Create integration tests in tests/integration/voice/stt/ for STT package

---

## Phase 3.4: TTS Package - Foundation (TDD)

### TTS Package Structure & Tests First
- [X] **T057** [P] Create pkg/voice/tts/iface/tts.go with TTSProvider interface (package-specific)
- [X] **T058** [P] Create pkg/voice/tts/test_utils.go with AdvancedMockTTSProvider and testing utilities
- [X] **T059** [P] Create pkg/voice/tts/advanced_test.go with table-driven tests for TTSProvider interface
- [X] **T060** [P] Create pkg/voice/tts/config_test.go with tests for TTS configuration validation
- [X] **T061** [P] Create pkg/voice/tts/errors_test.go with tests for TTS error handling
- [X] **T062** [P] Create pkg/voice/tts/metrics_test.go with tests for TTS metrics
- [X] **T063** [P] Create pkg/voice/tts/registry_test.go with tests for TTS provider registry

### TTS Package Core Implementation
- [X] **T064** [P] Create pkg/voice/tts/config.go with TTSConfig struct, validation, and functional options
- [X] **T065** [P] Create pkg/voice/tts/errors.go with TTSError type and error codes (Op/Err/Code pattern)
- [X] **T066** [P] Create pkg/voice/tts/metrics.go with OTEL metrics implementation (NewMetrics pattern)
- [X] **T067** [P] Create pkg/voice/tts/registry.go with global provider registry pattern
- [X] **T068** [P] Create pkg/voice/tts/tts.go with factory function NewProvider and base types

### TTS Package - OpenAI Provider
- [X] **T069** [P] Create pkg/voice/tts/providers/openai/config.go with OpenAIConfig struct
- [X] **T070** [P] Create pkg/voice/tts/providers/openai/provider.go with OpenAIProvider implementation
- [X] **T071** [P] Create pkg/voice/tts/providers/openai/streaming.go with streaming TTS implementation
- [X] **T072** [P] Create pkg/voice/tts/providers/openai/provider_test.go with OpenAI provider tests
- [X] **T073** Register OpenAI provider in pkg/voice/tts/registry.go

### TTS Package - Google Cloud Provider
- [X] **T074** [P] Create pkg/voice/tts/providers/google/config.go with GoogleConfig struct
- [X] **T075** [P] Create pkg/voice/tts/providers/google/provider.go with GoogleProvider implementation
- [X] **T076** [P] Create pkg/voice/tts/providers/google/ssml.go with SSML support
- [X] **T077** [P] Create pkg/voice/tts/providers/google/provider_test.go with Google provider tests
- [X] **T078** Register Google provider in pkg/voice/tts/registry.go

### TTS Package - Azure Provider
- [X] **T079** [P] Create pkg/voice/tts/providers/azure/config.go with AzureConfig struct
- [X] **T080** [P] Create pkg/voice/tts/providers/azure/provider.go with AzureProvider implementation
- [X] **T081** [P] Create pkg/voice/tts/providers/azure/ssml.go with SSML support
- [X] **T082** [P] Create pkg/voice/tts/providers/azure/provider_test.go with Azure provider tests
- [X] **T083** Register Azure provider in pkg/voice/tts/registry.go

### TTS Package - ElevenLabs Provider
- [X] **T084** [P] Create pkg/voice/tts/providers/elevenlabs/config.go with ElevenLabsConfig struct
- [X] **T085** [P] Create pkg/voice/tts/providers/elevenlabs/provider.go with ElevenLabsProvider implementation
- [X] **T086** [P] Create pkg/voice/tts/providers/elevenlabs/streaming.go with streaming implementation
- [X] **T087** [P] Create pkg/voice/tts/providers/elevenlabs/voice_cloning.go with voice cloning support
- [X] **T088** [P] Create pkg/voice/tts/providers/elevenlabs/provider_test.go with ElevenLabs provider tests
- [X] **T089** Register ElevenLabs provider in pkg/voice/tts/registry.go

### TTS Package - Integration & Polish
- [X] **T090** Create pkg/voice/tts/README.md with TTS package documentation
- [X] **T091** [P] Create pkg/voice/tts/benchmarks_test.go with performance benchmarks
- [X] **T092** [P] Create integration tests in tests/integration/voice/tts/ for TTS package

---

## Phase 3.5: VAD Package - Foundation (TDD)

### VAD Package Structure & Tests First
- [X] **T093** [P] Create pkg/voice/vad/iface/vad.go with VADProvider interface (package-specific)
- [X] **T094** [P] Create pkg/voice/vad/test_utils.go with AdvancedMockVADProvider and testing utilities
- [X] **T095** [P] Create pkg/voice/vad/advanced_test.go with table-driven tests for VADProvider interface
- [X] **T096** [P] Create pkg/voice/vad/config_test.go with tests for VAD configuration validation
- [X] **T097** [P] Create pkg/voice/vad/errors_test.go with tests for VAD error handling
- [X] **T098** [P] Create pkg/voice/vad/metrics_test.go with tests for VAD metrics
- [X] **T099** [P] Create pkg/voice/vad/registry_test.go with tests for VAD provider registry

### VAD Package Core Implementation
- [X] **T100** [P] Create pkg/voice/vad/config.go with VADConfig struct, validation, and functional options
- [X] **T101** [P] Create pkg/voice/vad/errors.go with VADError type and error codes (Op/Err/Code pattern)
- [X] **T102** [P] Create pkg/voice/vad/metrics.go with OTEL metrics implementation (NewMetrics pattern)
- [X] **T103** [P] Create pkg/voice/vad/registry.go with global provider registry pattern
- [X] **T104** [P] Create pkg/voice/vad/vad.go with factory function NewProvider and base types

### VAD Package - Silero Provider
- [X] **T105** [P] Create pkg/voice/vad/providers/silero/config.go with SileroVADConfig struct
- [X] **T106** [P] Create pkg/voice/vad/providers/silero/provider.go with SileroProvider implementation
- [X] **T107** [P] Create pkg/voice/vad/providers/silero/onnx.go with ONNX model loading and inference
- [X] **T108** [P] Create pkg/voice/vad/providers/silero/provider_test.go with Silero provider tests
- [X] **T109** Register Silero provider in pkg/voice/vad/registry.go

### VAD Package - Energy-Based Provider
- [X] **T110** [P] Create pkg/voice/vad/providers/energy/config.go with EnergyVADConfig struct
- [X] **T111** [P] Create pkg/voice/vad/providers/energy/provider.go with EnergyProvider implementation
- [X] **T112** [P] Create pkg/voice/vad/providers/energy/energy.go with energy threshold calculation
- [X] **T113** [P] Create pkg/voice/vad/providers/energy/provider_test.go with Energy provider tests
- [X] **T114** Register Energy provider in pkg/voice/vad/registry.go

### VAD Package - WebRTC Provider
- [X] **T115** [P] Create pkg/voice/vad/providers/webrtc/config.go with WebRTCVADConfig struct
- [X] **T116** [P] Create pkg/voice/vad/providers/webrtc/provider.go with WebRTCProvider implementation
- [X] **T117** [P] Create pkg/voice/vad/providers/webrtc/provider_test.go with WebRTC provider tests
- [X] **T118** Register WebRTC provider in pkg/voice/vad/registry.go

### VAD Package - RNNoise Provider
- [X] **T119** [P] Create pkg/voice/vad/providers/rnnoise/config.go with RNNoiseVADConfig struct
- [X] **T120** [P] Create pkg/voice/vad/providers/rnnoise/provider.go with RNNoiseProvider implementation
- [X] **T121** [P] Create pkg/voice/vad/providers/rnnoise/provider_test.go with RNNoise provider tests
- [X] **T122** Register RNNoise provider in pkg/voice/vad/registry.go

### VAD Package - Integration & Polish
- [X] **T123** Create pkg/voice/vad/README.md with VAD package documentation
- [X] **T124** [P] Create pkg/voice/vad/benchmarks_test.go with performance benchmarks
- [X] **T125** [P] Create integration tests in tests/integration/voice/vad/ for VAD package

---

## Phase 3.6: Turn Detection Package - Foundation (TDD)

### Turn Detection Package Structure & Tests First
- [X] **T122** [P] Create pkg/voice/turndetection/iface/turndetection.go with TurnDetector interface
- [X] **T123** [P] Create pkg/voice/turndetection/test_utils.go with AdvancedMockTurnDetector and testing utilities
- [X] **T124** [P] Create pkg/voice/turndetection/advanced_test.go with table-driven tests for TurnDetector interface
- [X] **T125** [P] Create pkg/voice/turndetection/config_test.go with tests for turn detection configuration
- [X] **T126** [P] Create pkg/voice/turndetection/errors_test.go with tests for error handling
- [X] **T127** [P] Create pkg/voice/turndetection/metrics_test.go with tests for metrics
- [X] **T128** [P] Create pkg/voice/turndetection/registry_test.go with tests for provider registry

### Turn Detection Package Core Implementation
- [X] **T129** [P] Create pkg/voice/turndetection/config.go with TurnDetectionConfig struct, validation, and functional options
- [X] **T130** [P] Create pkg/voice/turndetection/errors.go with TurnDetectionError type and error codes
- [X] **T131** [P] Create pkg/voice/turndetection/metrics.go with OTEL metrics implementation
- [X] **T132** [P] Create pkg/voice/turndetection/registry.go with global provider registry pattern
- [X] **T133** [P] Create pkg/voice/turndetection/turndetection.go with factory function NewDetector and base types

### Turn Detection Package - ONNX Provider
- [X] **T134** [P] Create pkg/voice/turndetection/providers/onnx/config.go with ONNXConfig struct
- [X] **T135** [P] Create pkg/voice/turndetection/providers/onnx/provider.go with ONNXProvider implementation
- [X] **T136** [P] Create pkg/voice/turndetection/providers/onnx/model.go with ONNX model loading and inference
- [X] **T137** [P] Create pkg/voice/turndetection/providers/onnx/provider_test.go with ONNX provider tests
- [X] **T138** Register ONNX provider in pkg/voice/turndetection/registry.go

### Turn Detection Package - Heuristic Provider
- [X] **T139** [P] Create pkg/voice/turndetection/providers/heuristic/config.go with HeuristicConfig struct
- [X] **T140** [P] Create pkg/voice/turndetection/providers/heuristic/provider.go with HeuristicProvider implementation
- [X] **T141** [P] Create pkg/voice/turndetection/providers/heuristic/silence.go with silence detection logic
- [X] **T142** [P] Create pkg/voice/turndetection/providers/heuristic/provider_test.go with Heuristic provider tests
- [X] **T143** Register Heuristic provider in pkg/voice/turndetection/registry.go

### Turn Detection Package - Integration & Polish
- [X] **T144** Create pkg/voice/turndetection/README.md with turn detection package documentation
- [X] **T145** [P] Create pkg/voice/turndetection/benchmarks_test.go with performance benchmarks
- [X] **T146** [P] Create integration tests in tests/integration/voice/turndetection/ for turn detection package

---

## Phase 3.7: Transport Package - Foundation (TDD)

### Transport Package Structure & Tests First
- [X] **T147** [P] Create pkg/voice/transport/iface/transport.go with Transport interface
- [X] **T148** [P] Create pkg/voice/transport/test_utils.go with AdvancedMockTransport and testing utilities
- [X] **T149** [P] Create pkg/voice/transport/advanced_test.go with table-driven tests for Transport interface
- [X] **T150** [P] Create pkg/voice/transport/config_test.go with tests for transport configuration
- [X] **T151** [P] Create pkg/voice/transport/errors_test.go with tests for error handling
- [X] **T152** [P] Create pkg/voice/transport/metrics_test.go with tests for metrics
- [X] **T153** [P] Create pkg/voice/transport/registry_test.go with tests for Transport provider registry

### Transport Package Core Implementation
- [X] **T154** [P] Create pkg/voice/transport/config.go with TransportConfig struct, validation, and functional options
- [X] **T155** [P] Create pkg/voice/transport/errors.go with TransportError type and error codes
- [X] **T156** [P] Create pkg/voice/transport/metrics.go with OTEL metrics implementation
- [X] **T157** [P] Create pkg/voice/transport/registry.go with global provider registry pattern
- [X] **T158** [P] Create pkg/voice/transport/transport.go with factory function NewProvider and base types

### Transport Package - WebRTC Implementation
- [X] **T157** [P] Create pkg/voice/transport/webrtc/config.go with WebRTCConfig struct
- [X] **T158** [P] Create pkg/voice/transport/webrtc/connection.go with WebRTC connection management
- [X] **T159** [P] Create pkg/voice/transport/webrtc/signaling.go with signaling implementation
- [X] **T160** [P] Create pkg/voice/transport/webrtc/codec.go with audio codec support (Opus, PCMU, PCMA)
- [X] **T161** [P] Create pkg/voice/transport/webrtc/rtp.go with RTP packetization/depacketization
- [X] **T162** [P] Create pkg/voice/transport/webrtc/reconnection.go with automatic reconnection logic
- [X] **T163** [P] Create pkg/voice/transport/webrtc/provider.go with WebRTCTransport implementation
- [X] **T164** [P] Create pkg/voice/transport/webrtc/provider_test.go with WebRTC transport tests
- [X] **T165** [P] Create pkg/voice/transport/webrtc/connection_test.go with connection management tests
- [X] **T166** [P] Create pkg/voice/transport/webrtc/signaling_test.go with signaling tests

### Transport Package - WebSocket Implementation
- [X] **T167** [P] Create pkg/voice/transport/websocket/config.go with WebSocketConfig struct
- [X] **T168** [P] Create pkg/voice/transport/websocket/provider.go with WebSocketTransport implementation
- [X] **T169** [P] Create pkg/voice/transport/websocket/provider_test.go with WebSocket transport tests

### Transport Package - Integration & Polish
- [X] **T170** Create pkg/voice/transport/README.md with transport package documentation
- [X] **T171** [P] Create pkg/voice/transport/benchmarks_test.go with performance benchmarks
- [X] **T172** [P] Create integration tests in tests/integration/voice/transport/ for transport package

---

## Phase 3.8: Noise Cancellation Package - Foundation (TDD)

### Noise Cancellation Package Structure & Tests First
- [X] **T170** [P] Create pkg/voice/noise/iface/noise.go with NoiseCancellation interface
- [X] **T171** [P] Create pkg/voice/noise/test_utils.go with AdvancedMockNoiseCancellation and testing utilities
- [X] **T172** [P] Create pkg/voice/noise/advanced_test.go with table-driven tests for NoiseCancellation interface
- [X] **T173** [P] Create pkg/voice/noise/config_test.go with tests for noise cancellation configuration
- [X] **T174** [P] Create pkg/voice/noise/errors_test.go with tests for error handling
- [X] **T175** [P] Create pkg/voice/noise/metrics_test.go with tests for metrics
- [X] **T176** [P] Create pkg/voice/noise/registry_test.go with tests for provider registry

### Noise Cancellation Package Core Implementation
- [X] **T177** [P] Create pkg/voice/noise/config.go with NoiseConfig struct, validation, and functional options
- [X] **T178** [P] Create pkg/voice/noise/errors.go with NoiseError type and error codes
- [X] **T179** [P] Create pkg/voice/noise/metrics.go with OTEL metrics implementation
- [X] **T180** [P] Create pkg/voice/noise/registry.go with global provider registry pattern
- [X] **T181** [P] Create pkg/voice/noise/noise.go with factory function NewProvider and base types

### Noise Cancellation Package - Spectral Subtraction Provider
- [X] **T182** [P] Create pkg/voice/noise/providers/spectral/config.go with SpectralConfig struct
- [X] **T183** [P] Create pkg/voice/noise/providers/spectral/provider.go with SpectralProvider implementation
- [X] **T184** [P] Create pkg/voice/noise/providers/spectral/fft.go with FFT-based noise reduction
- [X] **T185** [P] Create pkg/voice/noise/providers/spectral/adaptive.go with adaptive noise profile
- [X] **T186** [P] Create pkg/voice/noise/providers/spectral/provider_test.go with Spectral provider tests
- [X] **T187** Register Spectral provider in pkg/voice/noise/registry.go

### Noise Cancellation Package - RNNoise Provider
- [X] **T188** [P] Create pkg/voice/noise/providers/rnnoise/config.go with RNNoiseConfig struct
- [X] **T189** [P] Create pkg/voice/noise/providers/rnnoise/provider.go with RNNoiseProvider implementation
- [X] **T190** [P] Create pkg/voice/noise/providers/rnnoise/model.go with RNNoise model integration
- [X] **T191** [P] Create pkg/voice/noise/providers/rnnoise/provider_test.go with RNNoise provider tests
- [X] **T192** Register RNNoise provider in pkg/voice/noise/registry.go

### Noise Cancellation Package - WebRTC Provider
- [X] **T193** [P] Create pkg/voice/noise/providers/webrtc/config.go with WebRTCNoiseConfig struct
- [X] **T194** [P] Create pkg/voice/noise/providers/webrtc/provider.go with WebRTCNoiseProvider implementation
- [X] **T195** [P] Create pkg/voice/noise/providers/webrtc/provider_test.go with WebRTC noise provider tests
- [X] **T196** Register WebRTC provider in pkg/voice/noise/registry.go

### Noise Cancellation Package - Integration & Polish
- [X] **T197** Create pkg/voice/noise/README.md with noise cancellation package documentation
- [X] **T198** [P] Create pkg/voice/noise/benchmarks_test.go with performance benchmarks
- [X] **T199** [P] Create integration tests in tests/integration/voice/noise/ for noise cancellation package

---

## Phase 3.9: Session Package - Foundation (TDD)

### Session Package Structure & Tests First
- [X] **T200** [P] Create pkg/voice/session/iface/session.go with VoiceSession interface (package-specific)
- [X] **T201** [P] Create pkg/voice/session/test_utils.go with AdvancedMockVoiceSession and testing utilities
- [X] **T202** [P] Create pkg/voice/session/advanced_test.go with table-driven tests for VoiceSession interface
- [X] **T203** [P] Create pkg/voice/session/config_test.go with tests for VoiceSessionConfig validation
- [X] **T204** [P] Create pkg/voice/session/errors_test.go with tests for session error handling
- [X] **T205** [P] Create pkg/voice/session/metrics_test.go with tests for session metrics
- [X] **T206** [P] Create pkg/voice/session/state_test.go with tests for state machine transitions (covered in tests/contract/voice/session/state_test.go)
- [X] **T207** [P] Create pkg/voice/session/contract_test.go with contract tests from voice-session-api.md (covered in tests/contract/voice/session/)

### Session Package Core Implementation
- [X] **T208** [P] Create pkg/voice/session/config.go with VoiceSessionConfig and VoiceOptions structs, validation, and functional options
- [X] **T209** [P] Create pkg/voice/session/errors.go with SessionError type and error codes (Op/Err/Code pattern)
- [X] **T210** [P] Create pkg/voice/session/metrics.go with OTEL metrics implementation (NewMetrics pattern)
- [X] **T211** [P] Create pkg/voice/session/types.go with SessionState, SayOptions, SayHandle, and other types
- [X] **T212** [P] Create pkg/voice/session/session.go with NewVoiceSession factory function

### Session Package - Core Session Implementation
- [X] **T213** Create pkg/voice/session/internal/session_impl.go with VoiceSession implementation struct
- [X] **T214** Create pkg/voice/session/internal/state.go with state machine implementation
- [X] **T215** Create pkg/voice/session/internal/lifecycle.go with Start() and Stop() methods
- [X] **T216** Create pkg/voice/session/internal/say.go with Say() and SayWithOptions() methods
- [X] **T217** Create pkg/voice/session/internal/audio_processing.go with ProcessAudio() method
- [X] **T218** Create pkg/voice/session/internal/state_callbacks.go with OnStateChanged() callback management

### Session Package - Error Handling (Clarification Feature)
- [X] **T219** Create pkg/voice/session/internal/error_recovery.go with silent retry and automatic recovery logic
- [X] **T220** Create pkg/voice/session/internal/circuit_breaker.go with circuit breaker for provider failures
- [X] **T221** Create pkg/voice/session/internal/fallback.go with provider fallback switching logic
- [X] **T222** [P] Create pkg/voice/session/error_recovery_test.go with tests for error recovery scenarios

### Session Package - Session Timeout (Clarification Feature)
- [X] **T223** Create pkg/voice/session/internal/timeout.go with automatic session timeout on inactivity
- [X] **T224** Create pkg/voice/session/internal/away_detection.go with user away state detection
- [X] **T225** [P] Create pkg/voice/session/timeout_test.go with tests for session timeout behavior

### Session Package - Interruption Handling (Clarification Feature)
- [X] **T226** Create pkg/voice/session/internal/interruption.go with configurable interruption threshold logic
- [X] **T227** Create pkg/voice/session/internal/interruption_detector.go with word count and duration threshold detection
- [X] **T228** Create pkg/voice/session/internal/response_cancellation.go with response cancellation on interruption
- [X] **T229** [P] Create pkg/voice/session/interruption_test.go with tests for interruption handling

### Session Package - Preemptive Generation (Clarification Feature)
- [X] **T230** Create pkg/voice/session/internal/preemptive.go with preemptive generation logic
- [X] **T231** Create pkg/voice/session/internal/interim_handler.go with interim transcript handling
- [X] **T232** Create pkg/voice/session/internal/final_handler.go with final transcript handling and configurable behavior
- [X] **T233** Create pkg/voice/session/internal/response_strategy.go with configurable response strategy (discard, use if similar, always use)
- [X] **T234** [P] Create pkg/voice/session/preemptive_test.go with tests for preemptive generation

### Session Package - Long Utterance Handling (Clarification Feature)
- [X] **T235** Create pkg/voice/session/internal/chunking.go with configurable chunk size and processing strategy
- [X] **T236** Create pkg/voice/session/internal/buffering.go with buffering strategy for long utterances
- [X] **T237** Create pkg/voice/session/internal/streaming_incremental.go with streaming incremental processing
- [X] **T238** [P] Create pkg/voice/session/chunking_test.go with tests for long utterance handling

### Session Package - Integration with Providers
- [X] **T239** Create pkg/voice/session/internal/stt_integration.go with STT provider integration
- [X] **T240** Create pkg/voice/session/internal/tts_integration.go with TTS provider integration
- [X] **T241** Create pkg/voice/session/internal/vad_integration.go with VAD provider integration
- [X] **T242** Create pkg/voice/session/internal/turn_detection_integration.go with turn detector integration
- [X] **T243** Create pkg/voice/session/internal/transport_integration.go with transport integration
- [X] **T244** Create pkg/voice/session/internal/memory_integration.go with memory package integration
- [X] **T245** Create pkg/voice/session/internal/agent_integration.go with agent package integration

### Session Package - Streaming Support
- [X] **T246** Create pkg/voice/session/internal/streaming_stt.go with streaming STT integration
- [X] **T247** Create pkg/voice/session/internal/streaming_tts.go with streaming TTS integration
- [X] **T248** Create pkg/voice/session/internal/streaming_agent.go with streaming agent response integration
- [X] **T249** [P] Create pkg/voice/session/streaming_test.go with tests for streaming functionality

### Session Package - Integration & Polish
- [X] **T250** Create pkg/voice/session/README.md with session package documentation
- [X] **T251** [P] Create pkg/voice/session/benchmarks_test.go with performance benchmarks
- [X] **T252** [P] Create integration tests in tests/integration/voice/session/ for session package

---

## Phase 3.10: Integration Tests & Cross-Package Integration

### Contract Tests from API Contracts
- [X] **T253** [P] Create tests/contract/voice/session/start_test.go with contract tests for Start() method
- [X] **T254** [P] Create tests/contract/voice/session/stop_test.go with contract tests for Stop() method
- [X] **T255** [P] Create tests/contract/voice/session/say_test.go with contract tests for Say() and SayWithOptions() methods
- [X] **T256** [P] Create tests/contract/voice/session/process_audio_test.go with contract tests for ProcessAudio() method
- [X] **T257** [P] Create tests/contract/voice/session/state_test.go with contract tests for state machine
- [X] **T258** [P] Create tests/contract/voice/session/performance_test.go with performance contract tests

### Integration with Existing Beluga AI Packages
- [X] **T259** [P] Create tests/integration/voice/agents/agent_integration_test.go with agent package integration tests
- [X] **T260** [P] Create tests/integration/voice/memory/memory_integration_test.go with memory package integration tests
- [X] **T261** [P] Create tests/integration/voice/config/config_integration_test.go with config package integration tests
- [X] **T262** [P] Create tests/integration/voice/llms/llms_integration_test.go with LLM package integration tests
- [X] **T263** [P] Create tests/integration/voice/prompts/prompts_integration_test.go with prompts package integration tests
- [X] **T264** [P] Create tests/integration/voice/monitoring/monitoring_integration_test.go with monitoring package integration tests

### End-to-End Integration Tests
- [X] **T265** [P] Create tests/integration/voice/e2e/simple_session_test.go with simple voice session end-to-end test
- [X] **T266** [P] Create tests/integration/voice/e2e/streaming_session_test.go with streaming voice session test
- [X] **T267** [P] Create tests/integration/voice/e2e/interruption_test.go with interruption handling end-to-end test
- [X] **T268** [P] Create tests/integration/voice/e2e/timeout_test.go with session timeout end-to-end test
- [X] **T269** [P] Create tests/integration/voice/e2e/error_recovery_test.go with error recovery end-to-end test
- [X] **T270** [P] Create tests/integration/voice/e2e/preemptive_generation_test.go with preemptive generation end-to-end test
- [X] **T271** [P] Create tests/integration/voice/e2e/long_utterance_test.go with long utterance handling end-to-end test
- [X] **T272** [P] Create tests/integration/voice/e2e/multi_provider_test.go with multi-provider fallback test
- [X] **T273** [P] Create tests/integration/voice/e2e/concurrent_sessions_test.go with concurrent sessions test (100+ sessions)

### Quickstart Validation Tests
- [X] **T274** [P] Create tests/integration/voice/quickstart/quickstart_test.go with quickstart.md validation tests
- [X] **T275** [P] Create tests/integration/voice/quickstart/provider_setup_test.go with provider setup validation
- [X] **T276** [P] Create tests/integration/voice/quickstart/session_lifecycle_test.go with session lifecycle validation

---

## Phase 3.11: Documentation & Examples

### Package Documentation
- [X] **T277** [P] Create pkg/voice/README.md with voice package overview and architecture
- [X] **T278** [P] Update pkg/voice/stt/README.md with STT package usage examples
- [X] **T279** [P] Update pkg/voice/tts/README.md with TTS package usage examples
- [X] **T280** [P] Update pkg/voice/vad/README.md with VAD package usage examples
- [X] **T281** [P] Update pkg/voice/turndetection/README.md with turn detection usage examples
- [X] **T282** [P] Update pkg/voice/transport/README.md with transport package usage examples
- [X] **T283** [P] Update pkg/voice/session/README.md with session package usage examples
- [X] **T284** [P] Update pkg/voice/noise/README.md with noise cancellation usage examples

### Example Applications
- [X] **T285** [P] Create examples/voice/simple/main.go with simple voice agent example
- [X] **T286** [P] Create examples/voice/multi_provider/main.go with multi-provider example
- [X] **T287** [P] Create examples/voice/streaming/main.go with streaming example
- [X] **T288** [P] Create examples/voice/custom_provider/main.go with custom provider example
- [X] **T289** [P] Create examples/voice/interruption/main.go with interruption handling example
- [X] **T290** [P] Create examples/voice/preemptive/main.go with preemptive generation example

### API Documentation
- [X] **T291** [P] Generate API documentation using gomarkdoc for all voice packages
- [X] **T292** [P] Create docs/guides/voice-agents.md with voice agents usage guide
- [X] **T293** [P] Create docs/guides/voice-providers.md with provider configuration guide
- [X] **T294** [P] Create docs/guides/voice-performance.md with performance tuning guide
- [X] **T295** [P] Create docs/guides/voice-troubleshooting.md with troubleshooting guide

---

## Phase 3.12: Performance & Benchmarks

- [X] **T296** [P] Create pkg/voice/stt/benchmarks_test.go with STT provider benchmarks
- [X] **T297** [P] Create pkg/voice/tts/benchmarks_test.go with TTS provider benchmarks
- [X] **T298** [P] Create pkg/voice/vad/benchmarks_test.go with VAD provider benchmarks
- [X] **T299** [P] Create pkg/voice/turndetection/benchmarks_test.go with turn detection benchmarks
- [X] **T300** [P] Create pkg/voice/transport/benchmarks_test.go with transport benchmarks
- [X] **T301** [P] Create pkg/voice/session/benchmarks_test.go with session benchmarks
- [X] **T302** [P] Create pkg/voice/noise/benchmarks_test.go with noise cancellation benchmarks
- [X] **T303** Create pkg/voice/benchmarks/latency_test.go with latency benchmarks (target <200ms)
- [X] **T304** Create pkg/voice/benchmarks/concurrent_test.go with concurrent sessions benchmarks (target 100+)
- [X] **T305** Create pkg/voice/benchmarks/throughput_test.go with throughput benchmarks (target 1000+ chunks/sec)

---

## Phase 3.13: Final Polish & Validation

- [X] **T306** Run all tests and ensure 100% coverage for all voice packages
- [X] **T307** [P] Run all benchmarks and verify performance targets are met
- [X] **T308** [P] Run all integration tests and verify end-to-end functionality
- [X] **T309** [P] Verify constitutional compliance for all packages (structure, interfaces, observability)
- [X] **T310** [P] Run linter and fix all linting issues
- [X] **T311** [P] Run go mod verify and ensure all dependencies are valid
- [X] **T312** [P] Update go.mod with all voice package dependencies
- [X] **T313** Create CHANGELOG.md entry for voice agents feature
- [X] **T314** Update main README.md with voice agents feature documentation

---

## Dependencies

### Critical Path Dependencies
- T001-T007 (Setup) → All other tasks
- T008-T020 (Core Infrastructure) → All package tasks
- T021-T027, T057-T063, T093-T099, T122-T128, T147-T152, T170-T176, T200-T207 (Test utilities) → All implementation tasks
- T028-T032, T064-T068, T100-T104, T129-T133, T153-T156, T177-T181, T208-T212 (Core package files) → Provider tasks
- T213-T218 (Session core) → T219-T248 (Session features)
- All provider tasks → T239-T248 (Session integration)
- All package tasks → T253-T276 (Integration tests)
- All implementation tasks → T296-T305 (Benchmarks)
- All tasks → T306-T314 (Final polish)

### Parallel Execution Opportunities
- **T008-T020**: All can run in parallel (different interface/utility files)
- **T021-T027, T057-T063, T093-T099, T122-T128, T147-T152, T170-T176, T200-T207**: Test utilities for different packages can run in parallel
- **T033-T053**: STT providers can be implemented in parallel (different provider directories)
- **T069-T089**: TTS providers can be implemented in parallel
- **T105-T118**: VAD providers can be implemented in parallel
- **T134-T143**: Turn detection providers can be implemented in parallel
- **T182-T196**: Noise cancellation providers can be implemented in parallel
- **T253-T276**: Integration tests can run in parallel (different test files)
- **T277-T295**: Documentation tasks can run in parallel

### Sequential Requirements
- Test utilities (T021-T027, etc.) must complete before implementation tests
- Core package files (config.go, errors.go, metrics.go, registry.go) must complete before providers
- Session core (T213-T218) must complete before session features (T219-T248)
- All implementation must complete before integration tests
- All tests must pass before benchmarks
- All benchmarks must pass before final polish

---

## Parallel Execution Examples

### Example 1: Core Infrastructure (T008-T020)
```bash
# All can run in parallel - different files
Task: "Create pkg/voice/iface/stt.go with STTProvider interface definition"
Task: "Create pkg/voice/iface/tts.go with TTSProvider interface definition"
Task: "Create pkg/voice/iface/vad.go with VADProvider interface definition"
Task: "Create pkg/voice/iface/turndetection.go with TurnDetector interface definition"
Task: "Create pkg/voice/iface/transport.go with Transport interface definition"
Task: "Create pkg/voice/iface/session.go with VoiceSession interface definition"
Task: "Create pkg/voice/iface/noise.go with NoiseCancellation interface definition"
```

### Example 2: STT Providers (T033-T053)
```bash
# All STT providers can be implemented in parallel - different directories
Task: "Create pkg/voice/stt/providers/deepgram/provider.go with DeepgramProvider implementation"
Task: "Create pkg/voice/stt/providers/google/provider.go with GoogleProvider implementation"
Task: "Create pkg/voice/stt/providers/azure/provider.go with AzureProvider implementation"
Task: "Create pkg/voice/stt/providers/openai/provider.go with OpenAIProvider implementation"
```

### Example 3: Integration Tests (T253-T276)
```bash
# All integration tests can run in parallel - different test files
Task: "Create tests/contract/voice/session/start_test.go with contract tests for Start() method"
Task: "Create tests/integration/voice/agents/agent_integration_test.go with agent package integration tests"
Task: "Create tests/integration/voice/e2e/simple_session_test.go with simple voice session end-to-end test"
```

---

## Notes

- **[P] tasks** = different files, no dependencies
- **Verify tests fail** before implementing (TDD approach)
- **Commit after each task** or logical group of tasks
- **Constitutional compliance** must be verified for each package
- **Performance targets** must be validated: <200ms latency, 100+ concurrent sessions, 1000+ chunks/sec
- **Clarification features** must be fully implemented and tested:
  - Silent retry with automatic recovery
  - Automatic session timeout on inactivity
  - Configurable interruption thresholds
  - Configurable preemptive generation behavior
  - Configurable long utterance chunking strategy

---

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts**:
   - VoiceSession interface → 6 contract test tasks (T253-T258)
   - Each method → implementation task in session package

2. **From Data Model**:
   - 8 entities → model/struct creation tasks (integrated into package tasks)
   - Relationships → integration tasks (T239-T248)

3. **From User Stories**:
   - 8 acceptance scenarios → integration test tasks (T265-T273)
   - Quickstart scenarios → validation tasks (T274-T276)

4. **From Clarifications**:
   - Error handling → T219-T222
   - Session timeout → T223-T225
   - Interruptions → T226-T229
   - Preemptive generation → T230-T234
   - Long utterances → T235-T238

5. **Ordering**:
   - Setup → Tests → Interfaces → Config/Errors/Metrics → Providers → Session → Integration → Polish
   - Dependencies block parallel execution
   - TDD: Tests before implementation

---

## Validation Checklist
*GATE: Checked by main() before returning*

### Constitutional Compliance
- [x] Package structure tasks follow standard layout (config.go, metrics.go, errors.go, etc.)
- [x] OTEL metrics implementation tasks included for all packages
- [x] Test utilities (test_utils.go, advanced_test.go) tasks present for all packages
- [x] Registry pattern tasks for multi-provider packages (stt, tts, vad, turndetection, noise)

### Task Quality
- [x] All contracts have corresponding tests (T253-T258)
- [x] All entities have model tasks (integrated into package tasks)
- [x] All tests come before implementation (TDD approach)
- [x] Parallel tasks truly independent (different files/directories)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] All clarification features have implementation tasks
- [x] All providers have implementation tasks
- [x] Integration tests cover all user stories
- [x] Performance benchmarks included

### Completeness
- [x] All 7 packages have complete task breakdown
- [x] All 4+ STT providers have tasks
- [x] All 4+ TTS providers have tasks
- [x] All 3+ VAD providers have tasks
- [x] All clarification features have tasks
- [x] Integration with all 6+ existing packages has tasks
- [x] Documentation and examples have tasks

---

## Summary

**Total Tasks**: 314 tasks
- **Setup**: 7 tasks (T001-T007)
- **Core Infrastructure**: 13 tasks (T008-T020)
- **STT Package**: 36 tasks (T021-T056)
- **TTS Package**: 36 tasks (T057-T092)
- **VAD Package**: 29 tasks (T093-T121)
- **Turn Detection Package**: 25 tasks (T122-T146)
- **Transport Package**: 23 tasks (T147-T169)
- **Noise Cancellation Package**: 30 tasks (T170-T199)
- **Session Package**: 53 tasks (T200-T252)
- **Integration Tests**: 24 tasks (T253-T276)
- **Documentation**: 19 tasks (T277-T295)
- **Performance**: 10 tasks (T296-T305)
- **Final Polish**: 9 tasks (T306-T314)

**Estimated Parallel Execution**: ~150 tasks can run in parallel across different packages/providers

**Critical Path**: Setup → Core Infrastructure → Package Foundations → Providers → Session → Integration → Polish

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*

