---
title: "Jitter Buffer Management"
package: "voice/stt"
category: "audio"
complexity: "advanced"
---

# Jitter Buffer Management

## Problem

You need to handle network jitter and packet reordering in real-time speech-to-text streams, buffering audio packets to smooth out delivery irregularities and ensure continuous transcription quality.

## Solution

Implement a jitter buffer that receives out-of-order audio packets, reorders them, buffers a small amount to compensate for jitter, and delivers them in order to the STT processor. This works because you can timestamp packets, buffer them briefly, and deliver them in sequence order.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "sort"
    "sync"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.stt.jitter_buffer")

// AudioPacket represents an audio packet with sequence info
type AudioPacket struct {
    SequenceNumber int
    Timestamp      time.Time
    Data           []byte
    ReceivedAt     time.Time
}

// JitterBuffer manages audio packet buffering and reordering
type JitterBuffer struct {
    packets      map[int]*AudioPacket
    nextSequence int
    bufferSize   int
    maxWait      time.Duration
    mu           sync.Mutex
    outputCh     chan *AudioPacket
}

// NewJitterBuffer creates a new jitter buffer
func NewJitterBuffer(bufferSize int, maxWait time.Duration) *JitterBuffer {
    return &JitterBuffer{
        packets:    make(map[int]*AudioPacket),
        bufferSize: bufferSize,
        maxWait:    maxWait,
        outputCh:   make(chan *AudioPacket, bufferSize),
    }
}

// AddPacket adds a packet to the buffer
func (jb *JitterBuffer) AddPacket(ctx context.Context, packet *AudioPacket) error {
    ctx, span := tracer.Start(ctx, "jitter_buffer.add_packet")
    defer span.End()
    
    jb.mu.Lock()
    defer jb.mu.Unlock()
    
    span.SetAttributes(
        attribute.Int("sequence", packet.SequenceNumber),
        attribute.Int("buffer_size", len(jb.packets)),
    )
    
    // Store packet
    jb.packets[packet.SequenceNumber] = packet
    
    // Deliver packets in order
    jb.deliverOrdered(ctx)
    
    span.SetStatus(trace.StatusOK, "packet added")
    return nil
}

// deliverOrdered delivers packets in sequence order
func (jb *JitterBuffer) deliverOrdered(ctx context.Context) {
    for {
        packet, exists := jb.packets[jb.nextSequence]
        if !exists {
            break // Next packet not arrived yet
        }

        // Check if we should wait for more packets
        if jb.shouldWait(packet) {
            break
        }
        
        // Deliver packet
        select {
        case jb.outputCh <- packet:
            delete(jb.packets, jb.nextSequence)
            jb.nextSequence++
        case <-ctx.Done():
            return
        default:
            // Output channel full, will retry
            return
        }
    }
}

// shouldWait determines if we should wait for more packets
func (jb *JitterBuffer) shouldWait(packet *AudioPacket) bool {
    // If we have few packets buffered, wait a bit
    if len(jb.packets) < jb.bufferSize/2 {
        return time.Since(packet.ReceivedAt) < jb.maxWait
    }
    
    // If we have enough packets, don't wait
    return false
}

// GetOutputChannel returns the output channel
func (jb *JitterBuffer) GetOutputChannel() <-chan *AudioPacket {
    return jb.outputCh
}

// Flush flushes remaining packets
func (jb *JitterBuffer) Flush(ctx context.Context) {
    jb.mu.Lock()
    defer jb.mu.Unlock()
    
    // Sort remaining packets by sequence
    sequences := make([]int, 0, len(jb.packets))
    for seq := range jb.packets {
        sequences = append(sequences, seq)
    }
    sort.Ints(sequences)
    
    // Deliver all remaining packets
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

    // Create jitter buffer
    jitterBuffer := NewJitterBuffer(10, 100*time.Millisecond)
    
    // Process output
    go func() {
        for packet := range jitterBuffer.GetOutputChannel() {
            fmt.Printf("Received packet %d\n", packet.SequenceNumber)
            // Process with STT
        }
    }()
    
    // Simulate receiving packets out of order
    packets := []*AudioPacket{
        {SequenceNumber: 2, ReceivedAt: time.Now(), Data: []byte("packet2")},
        {SequenceNumber: 0, ReceivedAt: time.Now(), Data: []byte("packet0")},
        {SequenceNumber: 1, ReceivedAt: time.Now(), Data: []byte("packet1")},
    }
    
    for _, packet := range packets {
        jitterBuffer.AddPacket(ctx, packet)
    }
    // Flush remaining
    jitterBuffer.Flush(ctx)
```
    
    fmt.Println("Jitter buffer processed packets")
}

## Explanation

Let's break down what's happening:

1. **Sequence tracking** - Notice how we track sequence numbers and only deliver packets in order. This ensures the STT processor receives audio in the correct sequence.

2. **Smart buffering** - We buffer packets briefly to allow late-arriving packets to catch up. If we have few packets, we wait longer; if we have many, we deliver quickly.

3. **Ordered delivery** - Packets are delivered in strict sequence order, even if they arrive out of order. This is critical for maintaining audio quality.

```go
**Key insight:** Balance buffer size with latency. Too small a buffer causes gaps, too large causes delay. Adjust based on network conditions.

## Testing

```
Here's how to test this solution:
```go
func TestJitterBuffer_ReordersPackets(t *testing.T) {
    buffer := NewJitterBuffer(10, 50*time.Millisecond)
    
    // Add out of order
    buffer.AddPacket(context.Background(), &AudioPacket{SequenceNumber: 2})
    buffer.AddPacket(context.Background(), &AudioPacket{SequenceNumber: 0})
    buffer.AddPacket(context.Background(), &AudioPacket{SequenceNumber: 1})
    
    // Should receive in order
    p0 := <-buffer.GetOutputChannel()
    require.Equal(t, 0, p0.SequenceNumber)
}

## Variations

### Adaptive Buffer Size

Adjust buffer size based on network conditions:
func (jb *JitterBuffer) AdaptBufferSize(packetLossRate float64) {
    // Increase buffer if loss is high
}
```

### Timestamp-based Reordering

Use timestamps instead of sequence numbers:
```go
type TimestampedPacket struct {
    Timestamp time.Time
    Data      []byte
}
```
## Related Recipes

- **[Voice STT Overcoming Background Noise](./voice-stt-overcoming-background-noise.md)** - Improve STT quality
- **[Voice S2S Minimizing Glass-to-Glass Latency](./voice-s2s-minimizing-glass-to-glass-latency.md)** - Reduce latency
- **[Voice STT Guide](../guides/voice-providers.md)** - For a deeper understanding of STT
