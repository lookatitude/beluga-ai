// Package config provides configuration loading, validation, environment
// variable merging, provider configuration, and file watching for the
// Beluga AI framework.
//
// Configuration is loaded from JSON files, environment variables, or both,
// with struct-tag-based defaults and validation. The package also provides
// a file-watching mechanism for hot-reloading configuration at runtime.
//
// # Loading Configuration
//
// [Load] reads a JSON file and unmarshals it into a typed struct. Defaults
// from struct tags are applied to zero-valued fields, and the result is
// validated:
//
//	type AppConfig struct {
//	    Port    int    `json:"port" default:"8080" min:"1" max:"65535"`
//	    Host    string `json:"host" default:"localhost" required:"true"`
//	    Debug   bool   `json:"debug" default:"false"`
//	}
//
//	cfg, err := config.Load[AppConfig]("config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Environment Variables
//
// [LoadFromEnv] populates a config struct entirely from environment
// variables. Each exported field maps to PREFIX_FIELDNAME (uppercase):
//
//	cfg, err := config.LoadFromEnv[AppConfig]("BELUGA")
//	// reads BELUGA_PORT, BELUGA_HOST, BELUGA_DEBUG
//
// [MergeEnv] overlays environment variable values onto an existing config,
// only overriding fields with corresponding set variables:
//
//	config.MergeEnv(&cfg, "BELUGA")
//
// # Validation
//
// [Validate] checks a struct against its field tags:
//
//   - required:"true" — field must not be zero-valued
//   - min:"N" — numeric fields must be >= N
//   - max:"N" — numeric fields must be <= N
//
// Validation errors are returned as [*ValidationError] with the field name
// and descriptive message.
//
// # Provider Configuration
//
// [ProviderConfig] holds common configuration for any external provider
// (LLM, embedding, vector store, etc.), including provider name, API key,
// model identifier, base URL, timeout, and a flexible Options map for
// provider-specific settings. [GetOption] retrieves typed values from the
// Options map:
//
//	temp, ok := config.GetOption[float64](cfg, "temperature")
//
// # File Watching
//
// The [Watcher] interface abstracts configuration change detection.
// [FileWatcher] polls a file at regular intervals using SHA-256 content
// hashing, invoking a callback when changes are detected:
//
//	watcher := config.NewFileWatcher("config.json", 5*time.Second)
//	err := watcher.Watch(ctx, func(newConfig any) {
//	    data := newConfig.([]byte)
//	    // re-parse and apply configuration
//	})
package config
