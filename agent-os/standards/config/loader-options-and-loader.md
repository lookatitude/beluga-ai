# LoaderOptions and Loader

**LoaderOptions:** `ConfigName`, `ConfigPaths`, `EnvPrefix`, `Validate`, `SetDefaults`.

**DefaultLoaderOptions()** — Returns a sensible default set (e.g. `ConfigName: "config"`, `ConfigPaths: ["./config", "."]`, `EnvPrefix: "BELUGA"`, `Validate: true`, `SetDefaults: true`). Different apps may override via their own factory or by editing the struct.

**Loader.LoadConfig()** — Uses a fixed provider (e.g. Viper). Flow: create provider from options → `Load` → `SetDefaults` (if enabled) → `Validate` (if enabled). Returns `*Config` or error.

**Helpers:** `LoadFromEnv(prefix)`, `LoadFromFile(path)` for env-only or single-file loading when needed.
