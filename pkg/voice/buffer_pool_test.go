package voice

import (
	"sync"
	"testing"
)

func TestBufferPool_GetPut(t *testing.T) {
	pool := NewBufferPool()

	// Test getting a buffer
	buf := pool.Get(1024)
	if cap(buf) < 1024 {
		t.Errorf("Expected capacity >= 1024, got %d", cap(buf))
	}
	if len(buf) != 0 {
		t.Errorf("Expected length 0, got %d", len(buf))
	}

	// Test putting it back
	pool.Put(buf)

	// Test getting it again (should reuse)
	buf2 := pool.Get(1024)
	if cap(buf2) < 1024 {
		t.Errorf("Expected capacity >= 1024, got %d", cap(buf2))
	}
}

func TestBufferPool_GetExact(t *testing.T) {
	pool := NewBufferPool()

	buf := pool.GetExact(512)
	if len(buf) != 512 {
		t.Errorf("Expected length 512, got %d", len(buf))
	}
	if cap(buf) < 512 {
		t.Errorf("Expected capacity >= 512, got %d", cap(buf))
	}
}

func TestBufferPool_Concurrent(t *testing.T) {
	pool := NewBufferPool()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf := pool.Get(1024)
				pool.Put(buf)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkBufferPool_GetPut(b *testing.B) {
	pool := NewBufferPool()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := pool.Get(1024)
		pool.Put(buf)
	}
}

func BenchmarkBufferPool_GetPut_Concurrent(b *testing.B) {
	pool := NewBufferPool()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(1024)
			pool.Put(buf)
		}
	})
}

func BenchmarkMakeSlice(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := make([]byte, 1024)
		_ = buf
	}
}
