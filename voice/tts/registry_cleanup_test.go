package tts

// unregisterForTest removes a test-only registration so the package's tests
// can run repeatedly in one process (go test -count=N).
func unregisterForTest(name string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(registry, name)
}
