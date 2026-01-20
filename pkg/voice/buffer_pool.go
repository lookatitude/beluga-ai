// Package voice provides buffer pooling for efficient audio processing.
// This package implements sync.Pool-based buffer management to reduce GC pressure
// in high-throughput audio processing pipelines.
package voice

import (
	"sync"
)

// BufferPool provides pooled byte buffers for audio processing.
// It maintains separate pools for different buffer sizes to optimize memory usage.
type BufferPool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
}

// NewBufferPool creates a new buffer pool with predefined size pools.
func NewBufferPool() *BufferPool {
	bp := &BufferPool{
		pools: make(map[int]*sync.Pool),
	}

	// Pre-create pools for common audio buffer sizes
	commonSizes := []int{
		512,   // 32ms at 8kHz (256 samples * 2 bytes)
		1024,  // 64ms at 8kHz (512 samples * 2 bytes)
		2048,  // 128ms at 8kHz (1024 samples * 2 bytes)
		4096,  // 256ms at 8kHz (2048 samples * 2 bytes)
		8192,  // 512ms at 8kHz (4096 samples * 2 bytes)
		16384, // 1s at 8kHz (8192 samples * 2 bytes)
		32768, // 2s at 8kHz (16384 samples * 2 bytes)
	}

	for _, size := range commonSizes {
		size := size // Capture loop variable
		bp.pools[size] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, size)
			},
		}
	}

	return bp
}

// Get retrieves a buffer from the pool with at least the specified capacity.
// If no suitable pool exists, a new buffer is allocated.
func (bp *BufferPool) Get(size int) []byte {
	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()

	if exists {
		if buf := pool.Get(); buf != nil {
			return buf.([]byte)[:0] // Reset length but keep capacity
		}
	}

	// No pool for this size, allocate new
	return make([]byte, 0, size)
}

// Put returns a buffer to the appropriate pool.
// The buffer is reset to zero length but retains its capacity.
func (bp *BufferPool) Put(buf []byte) {
	if buf == nil || cap(buf) == 0 {
		return
	}

	capacity := cap(buf)

	bp.mu.RLock()
	pool, exists := bp.pools[capacity]
	bp.mu.RUnlock()

	if exists {
		// Reset buffer length and return to pool
		buf = buf[:0]
		pool.Put(buf)
	}
	// If no pool for this capacity, buffer is discarded (GC will reclaim it)
}

// GetExact retrieves a buffer with exactly the specified size from the pool.
func (bp *BufferPool) GetExact(size int) []byte {
	buf := bp.Get(size)
	return buf[:size] // Set length to requested size
}

// Global buffer pool instance
var globalBufferPool *BufferPool
var bufferPoolOnce sync.Once

// GetGlobalBufferPool returns the global buffer pool instance.
// It initializes the pool on first access.
func GetGlobalBufferPool() *BufferPool {
	bufferPoolOnce.Do(func() {
		globalBufferPool = NewBufferPool()
	})
	return globalBufferPool
}
