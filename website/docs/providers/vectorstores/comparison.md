---
title: Vector Store Comparison
sidebar_position: 2
---

# Vector Store Provider Comparison

Compare all vector store providers to choose the right one.

## Feature Matrix

| Feature | InMemory | PgVector | Pinecone |
|---------|----------|----------|----------|
| Persistence | No | Yes | Yes |
| Scalability | Limited | High | Very High |
| ACID | No | Yes | No |
| Setup | Easy | Medium | Easy |
| Cost | Free | Low | Medium |

## Use Case Recommendations

### Development/Testing
**Recommended:** InMemory
- Fast setup
- No dependencies
- Good for testing

### Production (Self-hosted)
**Recommended:** PgVector
- ACID compliance
- Good performance
- PostgreSQL integration

### Production (Cloud)
**Recommended:** Pinecone
- Managed service
- High scalability
- Easy setup

## Decision Guide

1. **Development?** → InMemory
2. **Need ACID?** → PgVector
3. **Cloud-native?** → Pinecone
4. **Large scale?** → Pinecone

---

**Next:** Read provider-specific guides

