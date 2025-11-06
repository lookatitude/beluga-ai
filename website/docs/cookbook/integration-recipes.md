---
title: Integration Recipes
sidebar_position: 1
---

# Integration Recipes

Common integration patterns with external services.

## Database Integration

```go
// Save to database
db.SaveMemory(userID, memoryVars)

// Load from database
memoryVars := db.LoadMemory(userID)
```

## API Integration

```go
// Make HTTP request
resp, err := http.Get("https://api.example.com/data")
// Process response
```

## File System Integration

```go
// Read file
data, _ := os.ReadFile("document.txt")
doc := schema.NewDocument(string(data), nil)
```

---

**More Recipes:** [Quick Solutions](./quick-solutions)

