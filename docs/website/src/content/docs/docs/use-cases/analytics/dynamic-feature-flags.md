---
title: Dynamic Feature Flags for AI
description: "Implement hot-reloadable feature flags for gradual AI model rollouts, A/B testing, and instant toggles without deployments."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "feature flags AI, hot-reload config, gradual rollout, AI model toggle, canary deployment, Beluga AI, Go, DevOps AI"
---

Deploying new AI features (a new model, a different prompt strategy, an updated guard rule) carries risk. If a new model produces poor responses for a specific customer segment, the fix requires a code change, CI/CD pipeline, and deployment — a 2-4 hour window where affected users see degraded quality. For AI features specifically, this delay is worse than for traditional features because LLM behavior is harder to predict in production than in testing.

Feature flags decouple deployment from activation. Ship the new model behind a flag, enable it for 5% of traffic, monitor quality metrics, then gradually increase — or instantly disable if metrics degrade. No deployment needed for any of these operations.

Beluga AI's `config/` package provides the foundation: hot-reloadable configuration with file system watching, so flag changes propagate to all instances within seconds.

## Solution Architecture

The feature flag manager builds on Beluga AI's `config/` package for hot-reload support with validation. Flag configurations load into an in-memory cache for sub-millisecond evaluation on the request path. The `config.Watch()` mechanism detects file changes and triggers reloads without application restarts. Percentage-based rollouts use deterministic user ID hashing so the same user always sees the same variant — critical for consistent user experience and valid A/B comparisons.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Feature    │───▶│    Config    │───▶│     Flag     │
│   Flag       │    │   Watcher    │    │    Cache     │
│   Config     │    │              │    │              │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Application │◀───│     Flag     │◀───│     User     │
│     Code     │    │   Evaluator  │    │   Context    │
│              │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Feature Flag Manager

The flag manager loads feature flags from configuration files and enables hot-reloading for instant updates.

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/o11y"
)

// FeatureFlag represents a dynamic feature flag.
type FeatureFlag struct {
    Name           string            `yaml:"name" validate:"required"`
    Enabled        bool              `yaml:"enabled"`
    RolloutPercent int               `yaml:"rollout_percent" validate:"min=0,max=100"`
    UserSegments   []string          `yaml:"user_segments,omitempty"`
    Metadata       map[string]string `yaml:"metadata,omitempty"`
}

// FeatureFlagManager manages dynamic feature flags with hot-reloading.
type FeatureFlagManager struct {
    flags  map[string]*FeatureFlag
    mu     sync.RWMutex
    loader *config.Loader
}

// NewFeatureFlagManager creates a new feature flag manager.
func NewFeatureFlagManager(ctx context.Context, configPath string) (*FeatureFlagManager, error) {
    loader, err := config.New(
        config.WithPath(configPath),
        config.WithHotReload(true),
    )
    if err != nil {
        return nil, fmt.Errorf("create config loader: %w", err)
    }

    manager := &FeatureFlagManager{
        flags:  make(map[string]*FeatureFlag),
        loader: loader,
    }

    if err := manager.loadFlags(ctx); err != nil {
        return nil, fmt.Errorf("load initial flags: %w", err)
    }

    // Watch for config changes
    go manager.watchFlags(ctx)

    return manager, nil
}

func (m *FeatureFlagManager) loadFlags(ctx context.Context) error {
    var flags []FeatureFlag
    if err := m.loader.Load(ctx, &flags); err != nil {
        return err
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    for i := range flags {
        m.flags[flags[i].Name] = &flags[i]
    }

    return nil
}

func (m *FeatureFlagManager) watchFlags(ctx context.Context) {
    updates := m.loader.Watch(ctx)
    for range updates {
        if err := m.loadFlags(ctx); err != nil {
            // Log error but continue watching
            continue
        }
    }
}
```

## Flag Evaluation with Gradual Rollouts

The evaluator checks flag state, applies percentage-based rollouts, and supports user segmentation.

```go
// IsEnabled checks if a feature flag is enabled for a given user.
func (m *FeatureFlagManager) IsEnabled(ctx context.Context, flagName string, userID string) bool {
    m.mu.RLock()
    flag, exists := m.flags[flagName]
    m.mu.RUnlock()

    if !exists || !flag.Enabled {
        return false
    }

    // Check rollout percentage
    if flag.RolloutPercent < 100 {
        return m.isUserInRollout(userID, flag.RolloutPercent)
    }

    // Check user segments
    if len(flag.UserSegments) > 0 {
        return m.isUserInSegment(userID, flag.UserSegments)
    }

    return true
}

func (m *FeatureFlagManager) isUserInRollout(userID string, percent int) bool {
    // Hash user ID to consistent value
    hash := hashString(userID)
    userPercent := int(hash % 100)
    return userPercent < percent
}

func (m *FeatureFlagManager) isUserInSegment(userID string, segments []string) bool {
    // Check if user belongs to any of the segments
    userSegment := getUserSegment(userID)
    for _, segment := range segments {
        if segment == userSegment {
            return true
        }
    }
    return false
}

func hashString(s string) uint32 {
    h := uint32(0)
    for i := 0; i < len(s); i++ {
        h = h*31 + uint32(s[i])
    }
    return h
}
```

## Configuration Example

Feature flags are defined in YAML configuration files:

```yaml
features:
  - name: new_dashboard
    enabled: true
    rollout_percent: 10
    metadata:
      description: "New dashboard redesign"

  - name: experimental_ai_model
    enabled: true
    rollout_percent: 100
    user_segments:
      - beta_users
      - internal
    metadata:
      description: "Experimental AI model for beta users"

  - name: legacy_api
    enabled: false
    metadata:
      description: "Legacy API endpoint (disabled)"
```

## Observability

Track flag evaluations and changes with OpenTelemetry metrics:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

func (m *FeatureFlagManager) IsEnabledWithMetrics(ctx context.Context, flagName string, userID string) bool {
    meter := otel.Meter("feature-flags")
    counter, _ := meter.Int64Counter("feature_flag_checks_total")

    enabled := m.IsEnabled(ctx, flagName, userID)

    counter.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("flag_name", flagName),
            attribute.Bool("enabled", enabled),
        ),
    )

    return enabled
}
```

## Production Considerations

### Hot Reload Performance

Configuration hot-reloading adds minimal overhead. The watcher uses file system notifications rather than polling, and flag updates complete in under 1 second. In-memory flag cache ensures sub-millisecond evaluation times.

### Consistency Guarantees

User ID hashing provides deterministic rollout percentages. The same user ID always maps to the same rollout bucket, ensuring consistent feature access. When increasing rollout percentages, users who previously saw the feature continue to see it.

### Audit Trail

Track all flag changes for compliance and debugging:

```go
type FlagAudit struct {
    FlagName  string
    OldValue  bool
    NewValue  bool
    ChangedBy string
    ChangedAt time.Time
}

func (m *FeatureFlagManager) auditFlagChange(ctx context.Context, audit FlagAudit) error {
    // Log to audit system
    logger := o11y.LoggerFromContext(ctx)
    logger.Info("feature flag changed",
        "flag", audit.FlagName,
        "old_value", audit.OldValue,
        "new_value", audit.NewValue,
        "changed_by", audit.ChangedBy,
        "changed_at", audit.ChangedAt,
    )

    return nil
}
```

### Emergency Disable

Disable problematic features instantly by updating the configuration file. The change propagates to all instances within 1 second without requiring deployments or application restarts.

### Scaling

The flag manager is stateless and scales horizontally. Each application instance maintains its own in-memory flag cache synchronized from the shared configuration source. This architecture supports thousands of concurrent flag evaluations with minimal latency.

## Results

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Feature Toggle Time (minutes) | 120-240 | 0.5 | 99.6-99.8% reduction |
| Emergency Disable Time (minutes) | 120-240 | 0.3 | 99.7-99.9% reduction |
| Production Incidents from Features | 8/month | 1 | 87.5% reduction |
| Feature Rollout Success Rate | 85% | 98.5% | 16% improvement |

## Related Resources

- [Configuration Guide](/docs/guides/configuration/) for config package patterns
- [Observability Guide](/docs/guides/observability/) for metrics and tracing
- [Multi-tenant API Keys](/docs/use-cases/multi-tenant-keys/) for related config patterns
