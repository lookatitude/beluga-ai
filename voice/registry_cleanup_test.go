package voice

// unregisterForTest removes a test-only registration so the package's tests
// can run repeatedly in one process (go test -count=N).
func unregisterForTest(name string) {
	vadRegistryMu.Lock()
	defer vadRegistryMu.Unlock()
	delete(vadRegistry, name)
}
