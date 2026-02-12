---
title: Voice & Audio AI Use Cases
description: "Build voice-enabled AI applications with STT, TTS, VAD, and frame-based pipelines. Production-ready Go examples for IVR, tutoring, and more."
sidebar:
  order: 0
head:
  - tag: meta
    attrs:
      name: keywords
      content: "voice AI use cases, speech-to-text, text-to-speech, voice assistant, IVR AI, Beluga AI, Go, voice pipeline examples"
---

Build voice-enabled applications with STT, TTS, S2S, VAD, and frame-based pipelines using Beluga AI's voice system. These use cases demonstrate the frame-based `FrameProcessor` architecture where each voice component (VAD, STT, TTS, turn detection) is a composable processor connected via `voice.Chain()`. S2S is used when latency is critical and text is not needed as an intermediate representation; separate STT+TTS is used when the application needs to inspect or validate transcribed text.

| Use Case | Description |
|----------|-------------|
| [Voice AI Applications](./voice-applications/) | Build voice-enabled applications with STT, TTS, S2S, and frame-based pipelines. |
| [Voice-Enabled IVR](./voice-ivr/) | Replace touch-tone IVR with voice-enabled interactive voice response. |
| [Automated Outbound Calling](./outbound-calling/) | Automate outbound calls for appointment reminders, consent verification, and surveys. |
| [Bilingual Conversation Tutor](./bilingual-tutor/) | Build an AI language tutor with real-time voice conversations and pronunciation feedback. |
| [AI Hotel Concierge](./hotel-concierge/) | Build a 24/7 AI concierge service with natural voice conversations. |
| [Multi-Turn Voice Forms](./voice-forms/) | Collect structured data through natural voice conversations with turn-by-turn validation. |
| [Voice Sessions](./voice-sessions-overview/) | Build production-ready voice agents with real-time audio transport and session management. |
| [Voice-Activated Industrial Control](./industrial-control/) | Implement hands-free voice commands for industrial equipment with noise-resistant STT. |
| [Live Meeting Minutes](./meeting-minutes/) | Generate structured meeting minutes from live audio with real-time transcription. |
| [E-Learning Voiceovers](./elearning-voiceovers/) | Generate multi-language voiceovers for educational content at scale. |
| [Interactive Audiobooks](./interactive-audiobooks/) | Create dynamic audiobook experiences with character voices and branching storylines. |
| [Barge-In Detection](./barge-in-detection/) | Enable users to interrupt voice agents mid-speech with low-latency detection. |
| [Low-Latency Turn Prediction](./low-latency-prediction/) | Reduce voice agent response delay with tuned turn-end detection. |
| [Multi-Speaker Segmentation](./multi-speaker-segmentation/) | Segment meeting audio by speaker using VAD and diarization. |
| [Noise-Resistant VAD](./noise-resistant-vad/) | Implement reliable voice activity detection in high-noise environments. |
