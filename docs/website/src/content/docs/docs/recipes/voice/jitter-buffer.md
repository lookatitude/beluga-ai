---
title: "Jitter Buffer Management"
description: "Recipe for handling network jitter and packet reordering in Go real-time STT streams with buffered, ordered delivery for smooth audio processing."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, jitter buffer, Go real-time audio, packet reordering, STT streaming, network resilience, audio buffer recipe"
---

## Problem

Real-time audio streaming over networks encounters two fundamental challenges: jitter and packet reordering. Jitter occurs when network latency varies unpredictably, causing audio packets to arrive at irregular intervals even if they were sent at a steady rate. Some packets arrive early, others late, creating gaps in the audio stream that manifest as pops, clicks, or stuttering in playback. Packet reordering happens when the network delivers packets out of order due to different routing paths or queuing delays. For speech-to-text processing, disordered or irregularly spaced audio frames produce garbled transcriptions because the STT model expects continuous, ordered input.

Without buffering, the STT processor must handle these irregularities directly: wait for late packets (increasing latency), skip missing packets (degrading quality), or maintain complex state machines to reorder packets on the fly. Real-time constraints make this challenging because you cannot wait indefinitely for late packets without introducing unacceptable delay.

## Solution

A jitter buffer decouples packet arrival timing from packet processing timing by introducing a small controlled delay that absorbs timing variations. Incoming packets are stored in a map indexed by sequence number, enabling constant-time lookup regardless of arrival order. The buffer maintains a next expected sequence number and delivers packets in order once they arrive and the buffer conditions are met. Smart buffering logic adapts delivery timing: when the buffer is nearly empty (few packets waiting), the system waits up to a maximum delay for late packets to arrive; when the buffer is filling (many packets waiting), the system delivers packets immediately to prevent unbounded growth.

This strategy trades a small constant latency (typically 50-200ms) for smooth, continuous audio delivery. The STT processor receives packets in the correct order at a steady rate, producing clean transcriptions even when network conditions are poor. Sequence number tracking ensures correctness: packets are never delivered twice or out of order. The maximum wait time caps worst-case latency, preventing indefinite stalls when packets are truly lost rather than merely late.

The buffer also provides a natural place to detect packet loss: if the next expected sequence number never arrives within the maximum wait time, the buffer can signal loss to upstream components or insert silence frames as placeholders.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.stt.jitter_buffer")

// AudioPacket represents an audio packet with sequence info.
type AudioPacket struct {
	SequenceNumber int
	Timestamp      time.Time
	Data           []byte
	ReceivedAt     time.Time
}

// JitterBuffer manages audio packet buffering and reordering.
type JitterBuffer struct {
	packets      map[int]*AudioPacket
	nextSequence int
	bufferSize   int
	maxWait      time.Duration
	mu           sync.Mutex
	outputCh     chan *AudioPacket
}

func NewJitterBuffer(bufferSize int, maxWait time.Duration) *JitterBuffer {
	return &JitterBuffer{
		packets:    make(map[int]*AudioPacket),
		bufferSize: bufferSize,
		maxWait:    maxWait,
		outputCh:   make(chan *AudioPacket, bufferSize),
	}
}

// AddPacket adds a packet to the buffer.
func (jb *JitterBuffer) AddPacket(ctx context.Context, packet *AudioPacket) error {
	ctx, span := tracer.Start(ctx, "jitter_buffer.add_packet")
	defer span.End()

	jb.mu.Lock()
	defer jb.mu.Unlock()

	span.SetAttributes(
		attribute.Int("sequence", packet.SequenceNumber),
		attribute.Int("buffer_size", len(jb.packets)),
	)

	jb.packets[packet.SequenceNumber] = packet
	jb.deliverOrdered(ctx)

	span.SetStatus(trace.StatusOK, "packet added")
	return nil
}

func (jb *JitterBuffer) deliverOrdered(ctx context.Context) {
	for {
		packet, exists := jb.packets[jb.nextSequence]
		if !exists {
			break
		}

		if jb.shouldWait(packet) {
			break
		}

		select {
		case jb.outputCh <- packet:
			delete(jb.packets, jb.nextSequence)
			jb.nextSequence++
		case <-ctx.Done():
			return
		default:
			return
		}
	}
}

func (jb *JitterBuffer) shouldWait(packet *AudioPacket) bool {
	if len(jb.packets) < jb.bufferSize/2 {
		return time.Since(packet.ReceivedAt) < jb.maxWait
	}
	return false
}

// GetOutputChannel returns the ordered output channel.
func (jb *JitterBuffer) GetOutputChannel() <-chan *AudioPacket {
	return jb.outputCh
}

// Flush delivers all remaining packets in sequence order.
func (jb *JitterBuffer) Flush(ctx context.Context) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	sequences := make([]int, 0, len(jb.packets))
	for seq := range jb.packets {
		sequences = append(sequences, seq)
	}
	sort.Ints(sequences)

	for _, seq := range sequences {
		if seq >= jb.nextSequence {
			select {
			case jb.outputCh <- jb.packets[seq]:
				delete(jb.packets, seq)
			case <-ctx.Done():
				return
			}
		}
	}

	close(jb.outputCh)
}

func main() {
	ctx := context.Background()

	jitterBuffer := NewJitterBuffer(10, 100*time.Millisecond)

	go func() {
		for packet := range jitterBuffer.GetOutputChannel() {
			fmt.Printf("Received packet %d\n", packet.SequenceNumber)
		}
	}()

	// Simulate receiving packets out of order
	packets := []*AudioPacket{
		{SequenceNumber: 2, ReceivedAt: time.Now(), Data: []byte("packet2")},
		{SequenceNumber: 0, ReceivedAt: time.Now(), Data: []byte("packet0")},
		{SequenceNumber: 1, ReceivedAt: time.Now(), Data: []byte("packet1")},
	}

	for _, packet := range packets {
		if err := jitterBuffer.AddPacket(ctx, packet); err != nil {
			fmt.Printf("Error adding packet: %v\n", err)
		}
	}

	jitterBuffer.Flush(ctx)
	fmt.Println("Jitter buffer processed packets")
}
```

The code implements a map-based buffer with sequence tracking and adaptive delivery logic. The `shouldWait` method encapsulates the buffer's decision logic: wait when under-full, deliver immediately when filling. This prevents both premature delivery (causing gaps) and unbounded buffering (causing latency growth).

## Explanation

1. **Sequence tracking** -- Every packet carries a monotonically increasing sequence number assigned by the sender. The buffer tracks `nextSequence`, the sequence number it expects to deliver next. When a packet with `nextSequence` arrives, the buffer delivers it immediately and increments `nextSequence`. When a later packet arrives (sequence > `nextSequence`), the buffer stores it in the map and waits for the missing packets to fill the gap. This strict ordering ensures the STT processor receives audio frames in the correct temporal order, maintaining the integrity of the audio stream. Sequence numbers also enable gap detection: if `nextSequence` remains unchanged for longer than `maxWait`, the buffer knows packets were lost.

2. **Smart buffering** -- The `shouldWait` function implements adaptive buffering that balances latency and smoothness. When the buffer contains fewer than half its capacity, the system assumes it is under-filled due to network delays and waits up to `maxWait` for late packets to arrive, tolerating jitter at the cost of increased latency. When the buffer contains more than half its capacity, the system assumes packets are arriving faster than being consumed and delivers them immediately to prevent unbounded memory growth and latency accumulation. This self-regulating behavior adapts to varying network conditions without manual tuning: the buffer automatically adds latency when needed to smooth jitter and removes latency when conditions are good.

3. **Ordered delivery** -- The `deliverOrdered` method ensures packets are delivered in strict sequence order regardless of arrival order. It attempts to deliver `nextSequence`, then `nextSequence+1`, and so on, stopping when it encounters a missing sequence or the buffer decides to wait. This loop runs after every packet insertion, ensuring immediate delivery when possible. The output channel is buffered to prevent blocking when the consumer is temporarily slow, allowing the buffer to continue accepting new packets. This pattern prevents head-of-line blocking where a single missing packet stalls delivery of all subsequent packets indefinitely.

**Key insight:** Balance buffer size with latency tolerance. A buffer that is too small cannot absorb jitter and produces gaps when late packets arrive after the buffer has already delivered subsequent packets. A buffer that is too large adds unnecessary latency, making the system feel sluggish even when network conditions are good. Start with 100ms buffering (10 packets at 10ms per packet) and adjust based on observed packet loss and latency metrics. Monitor buffer occupancy: consistently full buffers indicate the system cannot keep up with packet arrival rates; consistently empty buffers indicate over-buffering that adds latency unnecessarily.

## Variations

### Adaptive Buffer Size

Adjust buffer size dynamically based on observed packet loss rate or jitter magnitude. Increase the buffer when packet reordering is severe (high gap counts) or decrease it when packets arrive consistently in order, optimizing the latency-smoothness tradeoff per current network conditions.

### Timestamp-Based Reordering

Use packet timestamps instead of sequence numbers when sequence information is unavailable or untrusted. Sort packets by timestamp and deliver them in temporal order, handling clock skew by comparing timestamp deltas rather than absolute values.

## Related Recipes

- **[Background Noise Reduction](./background-noise)** -- Improve STT quality with preprocessing
- **[Glass-to-Glass Latency](./glass-to-glass-latency)** -- End-to-end latency optimization
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
