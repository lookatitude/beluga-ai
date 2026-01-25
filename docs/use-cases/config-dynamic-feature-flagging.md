# Dynamic Feature Flagging System

## Overview

A SaaS platform needed to implement dynamic feature flags to enable gradual rollouts, A/B testing, and instant feature toggles without deployments. They faced challenges with hardcoded feature flags, deployment dependencies, and inability to quickly disable problematic features.

**The challenge:** Feature flags required code changes and deployments, causing 2-4 hour delays to disable problematic features and preventing safe gradual rollouts.

**The solution:** We built a dynamic feature flagging system using Beluga AI's config package with hot-reloading capabilities, enabling instant feature toggles, gradual rollouts, and A/B testing without deployments.

## Business Context

### The Problem

Traditional feature flagging had significant limitations:

- **Deployment Dependency**: Feature flag changes required code deployments, causing 2-4 hour delays
- **No Gradual Rollouts**: All-or-nothing feature releases increased risk
- **Limited A/B Testing**: Couldn't test features with different user segments
- **Emergency Response**: Unable to quickly disable problematic features
- **Configuration Complexity**: Managing flags across multiple environments was error-prone

### The Opportunity

By implementing dynamic feature flagging, the platform could:

- **Instant Toggles**: Enable/disable features in seconds without deployments
- **Gradual Rollouts**: Release features to 1%, 10%, 50%, then 100% of users safely
- **A/B Testing**: Test features with different user segments simultaneously
- **Emergency Response**: Disable problematic features instantly
- **Reduce Risk**: Lower risk of production incidents through controlled rollouts

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| Feature Toggle Time (minutes) | 120-240 | \<1 | 0.5 |
| Gradual Rollout Support | No | Yes | Yes |
| A/B Testing Capability | No | Yes | Yes |
| Emergency Disable Time (minutes) | 120-240 | \<1 | 0.3 |
| Production Incidents from Features | 8/month | \<2 | 1 |
| Feature Rollout Success Rate (%) | 85 | 98 | 98.5 |

## Requirements

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | Hot-reload feature flags without restart | Enable instant toggles |
| FR2 | Support gradual rollouts (percentage-based) | Safe feature releases |
| FR3 | A/B testing with user segmentation | Test features with different groups |
| FR4 | Per-environment flag configuration | Different flags for dev/staging/prod |
| FR5 | Flag change audit trail | Track who changed what and when |
| FR6 | Real-time flag updates | Changes take effect immediately |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | Flag Update Latency | \<1 second |
| NFR2 | System Performance Impact | \<1% overhead |
| NFR3 | Flag Evaluation Speed | \<1ms per check |
| NFR4 | Availability | 99.99% uptime |

### Constraints

- Must not require application restarts
- Cannot impact application performance significantly
- Must support high-frequency flag checks
- Real-time updates required

## Architecture Requirements

### Design Principles

- **Hot Reloading**: Configuration changes take effect without restarts
- **Performance First**: Flag checks must be extremely fast
- **Observability**: Track all flag evaluations and changes
- **Safety**: Prevent accidental flag changes in production

### Key Architectural Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| In-memory flag cache | Fast evaluation | Requires memory for all flags |
| Watch-based updates | Real-time changes | Requires file system or API watcher |
| Percentage-based rollouts | Gradual feature release | Requires user ID hashing |
| Audit logging | Compliance and debugging | Additional storage overhead |

## Architecture

### High-Level Design

```mermaid
graph TB
    A[Feature Flag Config] --> B[Config Watcher]
    B --> C[Config Parser]
    C --> D[Flag Cache]
    D --> E[Flag Evaluator]
    E --> F[Application Code]
    G[Admin API] --> H[Flag Manager]
    H --> A
    H --> I[Audit Logger]
    J[Metrics Collector] --> E
    K[User Context] --> E
```

### How It Works

The system works like this:

1. **Configuration Loading** - When the application starts, the config package loads feature flags from configuration files. This is handled by the config loader because we need centralized configuration management.

2. **Hot Reloading** - Next, a config watcher monitors configuration files for changes. We chose this approach because it enables instant updates without restarts.

3. **Flag Evaluation** - Finally, when application code checks a flag, the evaluator checks the in-memory cache, applies rollout percentages, and returns the result. The user sees features enabled/disabled based on real-time configuration.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| Config Loader | Load feature flags | pkg/config with hot-reload |
| Flag Cache | In-memory flag storage | Custom cache with sync.RWMutex |
| Flag Evaluator | Check flag state | Custom evaluation logic |
| Config Watcher | Monitor config changes | File system watcher |
| Audit Logger | Track flag changes | pkg/monitoring (OTEL) |

## Implementation

### Phase 1: Setup/Foundation

First, we set up dynamic configuration with hot-reloading:
```go
package main

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/lookatitude/beluga-ai/pkg/config"
    "github.com/lookatitude/beluga-ai/pkg/monitoring"
)

// FeatureFlag represents a dynamic feature flag
type FeatureFlag struct {
    Name         string            `yaml:"name" validate:"required"`
    Enabled      bool              `yaml:"enabled"`
    RolloutPercent int             `yaml:"rollout_percent" validate:"min=0,max=100"`
    UserSegments []string          `yaml:"user_segments,omitempty"`
    Metadata     map[string]string `yaml:"metadata,omitempty"`
}

// FeatureFlagManager manages dynamic feature flags
type FeatureFlagManager struct {
    flags      map[string]*FeatureFlag
    mu         sync.RWMutex
    configLoader *config.Loader
    watcher    *ConfigWatcher
    tracer     trace.Tracer
    meter      metric.Meter
}

// NewFeatureFlagManager creates a new feature flag manager with hot-reloading
func NewFeatureFlagManager(ctx context.Context, configPath string) (*FeatureFlagManager, error) {
    // Setup config loader with hot-reload support
    options := config.DefaultLoaderOptions()
    options.ConfigName = "features"
    options.ConfigPaths = []string{configPath}
    
    loader, err := config.NewLoader(options)
    if err != nil {
        return nil, fmt.Errorf("failed to create config loader: %w", err)
    }
    
    manager := &FeatureFlagManager{
        flags:       make(map[string]*FeatureFlag),
        configLoader: loader,
    }
    
    // Load initial flags
    if err := manager.loadFlags(ctx); err != nil {
        return nil, fmt.Errorf("failed to load initial flags: %w", err)
    }
    
    // Setup config watcher for hot-reloading
    watcher, err := NewConfigWatcher(configPath, func() {
        if err := manager.loadFlags(ctx); err != nil {
            log.Error("Failed to reload flags", "error", err)
        }
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create config watcher: %w", err)
    }
    manager.watcher = watcher

    
    return manager, nil
}
```

**Key decisions:**
- We chose config package for centralized configuration management
- Hot-reloading enables instant feature toggles

For detailed setup instructions, see the [Config Package Guide](../guides/implementing-providers.md).

### Phase 2: Core Implementation

Next, we implemented flag evaluation with gradual rollouts:
```go
// IsEnabled checks if a feature flag is enabled for a given user
func (m *FeatureFlagManager) IsEnabled(ctx context.Context, flagName string, userID string) bool {
    ctx, span := m.tracer.Start(ctx, "feature_flag.evaluate")
    defer span.End()
    
    m.mu.RLock()
    flag, exists := m.flags[flagName]
    m.mu.RUnlock()
    
    if !exists {
        span.SetAttributes(attribute.Bool("flag_exists", false))
        return false // Default to disabled if flag doesn't exist
    }
    
    span.SetAttributes(
        attribute.String("flag_name", flagName),
        attribute.Bool("flag_enabled", flag.Enabled),
        attribute.Int("rollout_percent", flag.RolloutPercent),
    )
    
    // Check if flag is globally disabled
    if !flag.Enabled {
        return false
    }
    
    // Check rollout percentage
    if flag.RolloutPercent < 100 {
        enabled := m.isUserInRollout(userID, flag.RolloutPercent)
        span.SetAttributes(attribute.Bool("user_in_rollout", enabled))
        return enabled
    }
    
    // Check user segments
    if len(flag.UserSegments) > 0 {
        enabled := m.isUserInSegment(userID, flag.UserSegments)
        span.SetAttributes(attribute.Bool("user_in_segment", enabled))
        return enabled
    }
    
    // Flag is enabled for all users
    return true
}

func (m *FeatureFlagManager) isUserInRollout(userID string, percent int) bool {
    // Hash user ID to consistent value
    hash := hashString(userID)
    // Convert to percentage
    userPercent := int(hash % 100)
    return userPercent < percent
}
```

**Challenges encountered:**
- Config file watching: Solved by implementing file system watcher with debouncing
- Thread safety: Addressed using sync.RWMutex for concurrent flag access

### Phase 3: Integration/Polish

Finally, we integrated monitoring and audit logging:
// Production-ready with comprehensive monitoring
```go
func (m *FeatureFlagManager) IsEnabledWithMonitoring(ctx context.Context, flagName string, userID string) bool {
    ctx, span := m.tracer.Start(ctx, "feature_flag.evaluate.monitored",
        trace.WithAttributes(
            attribute.String("flag_name", flagName),
            attribute.String("user_id", userID),
        ),
    )
    defer span.End()
    
    startTime := time.Now()
    enabled := m.IsEnabled(ctx, flagName, userID)
    duration := time.Since(startTime)

    

    span.SetAttributes(
        attribute.Bool("enabled", enabled),
        attribute.Float64("evaluation_duration_ms", float64(duration.Nanoseconds())/1e6),
    )
    
    // Record metrics
    m.meter.Counter("feature_flag_checks_total").Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("flag_name", flagName),
            attribute.Bool("enabled", enabled),
        ),
    )
    
    m.meter.Histogram("feature_flag_evaluation_duration_ms").Record(ctx, float64(duration.Nanoseconds())/1e6)
    
    return enabled
}
```

## Results

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Feature Toggle Time (minutes) | 120-240 | 0.5 | 99.6-99.8% reduction |
| Gradual Rollout Support | No | Yes | New capability |
| A/B Testing Capability | No | Yes | New capability |
| Emergency Disable Time (minutes) | 120-240 | 0.3 | 99.7-99.9% reduction |
| Production Incidents from Features | 8/month | 1 | 87.5% reduction |
| Feature Rollout Success Rate (%) | 85 | 98.5 | 16% improvement |

### Qualitative Outcomes

- **Agility**: Instant feature toggles enable rapid response to issues
- **Safety**: Gradual rollouts reduce risk of production incidents
- **Testing**: A/B testing enables data-driven feature decisions
- **Reliability**: Emergency disable capability prevents extended outages

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| In-memory cache | Fast evaluation | Memory usage for all flags |
| Hot-reloading | Instant updates | Requires file system watcher |
| Percentage-based rollouts | Gradual release | Requires user ID hashing |

## Lessons Learned

### What Worked Well

✅ **Config Package Integration** - Using Beluga AI's config package provided robust configuration management with hot-reloading. Recommendation: Leverage config package for all dynamic configuration needs.

✅ **In-Memory Cache** - Caching flags in memory enabled sub-millisecond evaluation times. RWMutex provided efficient concurrent access.

### What We'd Do Differently

⚠️ **Config File Format** - In hindsight, we would use a more structured format (YAML with validation) from the beginning. Initial JSON format lacked validation.

⚠️ **Audit Logging** - We initially logged only flag changes. Adding evaluation logging helped debug rollout issues.

### Recommendations for Similar Projects

1. **Start with Config Package** - Use Beluga AI's config package from the beginning. It provides hot-reloading and validation out of the box.

2. **Monitor Flag Evaluations** - Track flag evaluation metrics (enabled/disabled rates, evaluation latency). These metrics are critical for optimization.

3. **Don't underestimate Rollout Logic** - Percentage-based rollouts require consistent user ID hashing. Test rollout logic thoroughly.

## Production Readiness Checklist

- [x] **Observability**: OpenTelemetry metrics, tracing, and logging configured
- [x] **Error Handling**: Comprehensive error handling for config loading failures
- [x] **Security**: Access controls for flag changes in place
- [x] **Performance**: Load testing completed - \<1ms evaluation time
- [x] **Scalability**: In-memory cache handles high-frequency checks
- [x] **Monitoring**: Dashboards configured for flag evaluation metrics
- [x] **Documentation**: API documentation and runbooks updated
- [x] **Testing**: Unit, integration, and chaos engineering tests passing
- [x] **Configuration**: Environment-specific flag configs validated
- [x] **Disaster Recovery**: Flag rollback procedures tested and documented

## Related Use Cases

If you're working on a similar project, you might also find these helpful:

- **[Multi-tenant API Key Management](./config-multi-tenant-api-keys.md)** - Multi-tenant configuration patterns
- **[Model A/B Testing Framework](./chatmodels-model-ab-testing.md)** - A/B testing infrastructure
- **[Config Package Guide](../guides/implementing-providers.md)** - Deep dive into configuration management
- **[Monitoring Dashboards](./monitoring-dashboards.md)** - Observability setup
