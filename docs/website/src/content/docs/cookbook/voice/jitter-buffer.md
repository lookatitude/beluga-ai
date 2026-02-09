---
title: "Jitter Buffer Management"
description: "Handle network jitter and packet reordering in real-time STT streams with buffered ordered delivery."
---

## Problem

You need to handle network jitter and packet reordering in real-time speech-to-text streams, buffering audio packets to smooth out delivery irregularities and ensure continuous transcription quality.

## Solution

Implement a jitter buffer that receives out-of-order audio packets, reorders them by sequence number, buffers a small amount to compensate for jitter, and delivers them in order to the STT processor.

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

## Explanation

1. **Sequence tracking** -- Packets are tracked by sequence number and only delivered in order. This ensures the STT processor receives audio in the correct sequence.

2. **Smart buffering** -- Packets are buffered briefly to allow late-arriving packets to catch up. When few packets are buffered, the system waits longer; when many are available, it delivers quickly.

3. **Ordered delivery** -- Packets are delivered in strict sequence order even if they arrive out of order. This is critical for maintaining audio quality in real-time transcription.

**Key insight:** Balance buffer size with latency. Too small a buffer causes gaps from missing packets; too large a buffer increases delay. Adjust based on network conditions.

## Variations

### Adaptive Buffer Size

Adjust buffer size dynamically based on observed packet loss rate. Increase the buffer when loss is high.

### Timestamp-Based Reordering

Use timestamps instead of sequence numbers when sequence information is unavailable.

## Related Recipes

- **[Background Noise Reduction](./background-noise)** -- Improve STT quality with preprocessing
- **[Glass-to-Glass Latency](./glass-to-glass-latency)** -- End-to-end latency optimization
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
