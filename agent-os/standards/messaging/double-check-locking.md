# Double-Check Locking for Sessions

Thread-safe session creation without blocking readers.

```go
// 1. Check with read lock (allows parallel readers)
m.provider.mu.RLock()
session, exists := m.provider.sessions[conversationSID]
m.provider.mu.RUnlock()

if !exists {
    // 2. Upgrade to write lock
    m.provider.mu.Lock()
    // 3. Double-check: another goroutine may have created it
    session, exists = m.provider.sessions[conversationSID]
    if !exists {
        session, err = NewMessagingSession(...)
        m.provider.sessions[conversationSID] = session
    }
    m.provider.mu.Unlock()
}
```

## Why This Pattern?
- **Read-heavy workload**: Most requests find session exists; RLock allows parallel reads
- **Expensive creation**: Sessions involve network calls; duplicates waste resources

## When to Use
- Get-or-create with expensive creation
- Map access where reads far outnumber writes
- NOT for simple caches (use `sync.Map` instead)
