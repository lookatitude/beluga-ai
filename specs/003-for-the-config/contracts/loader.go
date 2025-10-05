// Package contracts defines the API contracts for configuration loading operations
// These interfaces define the contract for configuration loader functionality
// implementing FR-005, FR-021, FR-024

package contracts

import (
	"context"
	"time"
)

// Loader defines the contract for configuration loading orchestration
type Loader interface {
	// LoadConfig loads configuration from all configured sources
	// Implements FR-021: System MUST respect context cancellation
	LoadConfig(ctx context.Context) (*Config, error)

	// MustLoadConfig loads configuration and panics on error (for initialization)
	MustLoadConfig(ctx context.Context) *Config

	// ReloadConfig reloads configuration from sources
	// Supports hot-reload functionality
	ReloadConfig(ctx context.Context) (*Config, error)

	// GetConfig returns the currently loaded configuration
	GetConfig() *Config

	// GetConfigVersion returns the version of currently loaded configuration
	// Implements FR-005: System MUST maintain backward compatibility
	GetConfigVersion() string
}

// EnhancedLoader extends Loader with advanced capabilities
type EnhancedLoader interface {
	Loader
	MigrationSupport
	ConfigWatcher
	LoaderMetrics
}

// MigrationSupport defines configuration migration capabilities
type MigrationSupport interface {
	// MigrateConfig migrates configuration from one version to another
	// Implements FR-024: System MUST include migration guides and utilities
	MigrateConfig(ctx context.Context, fromVersion, toVersion string) error

	// GetMigrationPlan returns the plan for migrating between versions
	GetMigrationPlan(fromVersion, toVersion string) (MigrationPlan, error)

	// ValidateMigration validates that a migration can be performed
	ValidateMigration(ctx context.Context, fromVersion, toVersion string) error

	// BackupConfig creates a backup of current configuration
	// Implements FR-024: Support for safe migrations with rollback
	BackupConfig(ctx context.Context, path string) error

	// RestoreConfig restores configuration from backup
	RestoreConfig(ctx context.Context, path string) error
}

// ConfigWatcher defines configuration watching capabilities
type ConfigWatcher interface {
	// StartWatching begins watching configuration sources for changes
	// Enables hot-reload functionality
	StartWatching(ctx context.Context, callback ChangeCallback) error

	// StopWatching stops watching configuration sources
	StopWatching(ctx context.Context) error

	// GetWatchStatus returns current watching status
	GetWatchStatus() WatchStatus

	// AddWatchPath adds a path to be watched for changes
	AddWatchPath(path string) error

	// RemoveWatchPath removes a path from watching
	RemoveWatchPath(path string) error
}

// LoaderMetrics defines metrics capabilities for loaders
type LoaderMetrics interface {
	// GetLoadMetrics returns metrics about loading operations
	GetLoadMetrics() LoadMetrics

	// ResetLoadMetrics resets all load metrics
	ResetLoadMetrics() error

	// GetPerformanceStats returns performance statistics
	GetPerformanceStats() PerformanceStats
}

// Config represents the loaded configuration structure
type Config interface {
	// GetVersion returns the configuration version
	GetVersion() string

	// GetTimestamp returns when the configuration was loaded
	GetTimestamp() time.Time

	// GetSources returns the sources used to load this configuration
	GetSources() []string

	// Validate validates the configuration structure
	Validate(ctx context.Context) error

	// Clone creates a deep copy of the configuration
	Clone() Config

	// Merge merges another configuration into this one
	Merge(other Config) Config

	// GetChecksum returns a checksum of the configuration for change detection
	GetChecksum() string
}

// ChangeCallback defines the signature for configuration change callbacks
type ChangeCallback func(oldConfig, newConfig Config, source string) error

// WatchStatus represents the current watching state
type WatchStatus struct {
	Watching     bool      `json:"watching"`
	WatchedPaths []string  `json:"watched_paths"`
	LastChange   time.Time `json:"last_change"`
	ChangeCount  int64     `json:"change_count"`
	ErrorCount   int64     `json:"error_count"`
	LastError    string    `json:"last_error,omitempty"`
}

// MigrationPlan describes how to migrate between configuration versions
type MigrationPlan struct {
	FromVersion    string          `json:"from_version"`
	ToVersion      string          `json:"to_version"`
	Steps          []MigrationStep `json:"steps"`
	Reversible     bool            `json:"reversible"`
	BackupRequired bool            `json:"backup_required"`
	EstimatedTime  time.Duration   `json:"estimated_time"`
	Warnings       []string        `json:"warnings"`
}

// MigrationStep represents a single step in a migration plan
type MigrationStep struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Type        MigrationStepType         `json:"type"`
	Operation   func(config Config) error `json:"-"`
	Reversible  bool                      `json:"reversible"`
	Required    bool                      `json:"required"`
}

// MigrationStepType defines different types of migration steps
type MigrationStepType string

const (
	MigrationStepTypeFieldRename   MigrationStepType = "field_rename"
	MigrationStepTypeFieldRemove   MigrationStepType = "field_remove"
	MigrationStepTypeFieldAdd      MigrationStepType = "field_add"
	MigrationStepTypeValidationAdd MigrationStepType = "validation_add"
	MigrationStepTypeTransform     MigrationStepType = "transform"
	MigrationStepTypeCustom        MigrationStepType = "custom"
)

// LoadMetrics contains metrics about loading operations
type LoadMetrics struct {
	LoadCount       int64         `json:"load_count"`
	ReloadCount     int64         `json:"reload_count"`
	FailureCount    int64         `json:"failure_count"`
	AverageLoadTime time.Duration `json:"average_load_time"`
	LastLoadTime    time.Time     `json:"last_load_time"`
	LastReloadTime  time.Time     `json:"last_reload_time"`
	ConfigSize      int64         `json:"config_size"`
	SourceCount     int           `json:"source_count"`
}

// PerformanceStats contains performance statistics for optimization
type PerformanceStats struct {
	FastestLoad    time.Duration `json:"fastest_load"`
	SlowestLoad    time.Duration `json:"slowest_load"`
	P95LoadTime    time.Duration `json:"p95_load_time"`
	P99LoadTime    time.Duration `json:"p99_load_time"`
	CacheHitRate   float64       `json:"cache_hit_rate"`
	MemoryUsage    int64         `json:"memory_usage"`
	GoroutineCount int           `json:"goroutine_count"`
}

// LoaderOptions defines configuration for the loader
type LoaderOptions struct {
	ConfigName        string        `mapstructure:"config_name" default:"config"`
	ConfigPaths       []string      `mapstructure:"config_paths"`
	EnvPrefix         string        `mapstructure:"env_prefix" default:""`
	Validate          bool          `mapstructure:"validate" default:"true"`
	SetDefaults       bool          `mapstructure:"set_defaults" default:"true"`
	EnableHotReload   bool          `mapstructure:"enable_hot_reload" default:"false"`
	LoadTimeout       time.Duration `mapstructure:"load_timeout" default:"30s"`
	ValidationTimeout time.Duration `mapstructure:"validation_timeout" default:"10s"`
	EnableMigration   bool          `mapstructure:"enable_migration" default:"false"`
	MigrationPath     string        `mapstructure:"migration_path"`
	BackupPath        string        `mapstructure:"backup_path"`
	EnableCaching     bool          `mapstructure:"enable_caching" default:"true"`
	CacheTTL          time.Duration `mapstructure:"cache_ttl" default:"5m"`
}

// LoaderError defines structured errors for loader operations
type LoaderError struct {
	Op     string // operation that failed
	Source string // source involved
	Path   string // file path involved
	Err    error  // underlying error
	Code   string // error code
}

const (
	ErrCodeConfigNotFound   = "CONFIG_NOT_FOUND"
	ErrCodeLoadTimeout      = "LOAD_TIMEOUT"
	ErrCodeMigrationFailed  = "MIGRATION_FAILED"
	ErrCodeWatchingFailed   = "WATCHING_FAILED"
	ErrCodeBackupFailed     = "BACKUP_FAILED"
	ErrCodeRestoreFailed    = "RESTORE_FAILED"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
)
