package documentloaders

import (
	"io/fs"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/directory"
	_ "github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/text"
)

// init registers built-in loaders with the global registry.
func init() {
	registry := GetRegistry()

	// Register directory loader
	registry.Register("directory", func(config map[string]any) (iface.DocumentLoader, error) {
		var fsys fs.FS
		var opts []DirectoryOption

		// Get file system
		if fsysVal, ok := config["fsys"]; ok {
			if fsysImpl, ok := fsysVal.(fs.FS); ok {
				fsys = fsysImpl
			} else if path, ok := fsysVal.(string); ok {
				fsys = os.DirFS(path)
			}
		} else if path, ok := config["path"].(string); ok {
			fsys = os.DirFS(path)
		}

		if fsys == nil {
			fsys = os.DirFS(".")
		}

		// Apply options
		if maxDepth, ok := config["max_depth"].(int); ok {
			opts = append(opts, WithMaxDepth(maxDepth))
		}
		if extensions, ok := config["extensions"].([]string); ok {
			opts = append(opts, WithExtensions(extensions...))
		}
		if concurrency, ok := config["concurrency"].(int); ok {
			opts = append(opts, WithConcurrency(concurrency))
		}
		if maxFileSize, ok := config["max_file_size"].(int64); ok {
			opts = append(opts, func(cfg *DirectoryConfig) {
				cfg.MaxFileSize = maxFileSize
			})
		}
		if followSymlinks, ok := config["follow_symlinks"].(bool); ok {
			opts = append(opts, WithFollowSymlinks(followSymlinks))
		}

		return NewDirectoryLoader(fsys, opts...)
	})

	// Register text loader
	registry.Register("text", func(config map[string]any) (iface.DocumentLoader, error) {
		path, ok := config["path"].(string)
		if !ok {
			return nil, NewLoaderError("registry", ErrCodeInvalidConfig, "", "path is required", nil)
		}

		var opts []Option
		if maxFileSize, ok := config["max_file_size"].(int64); ok {
			opts = append(opts, func(cfg *LoaderConfig) {
				cfg.MaxFileSize = maxFileSize
			})
		}

		return NewTextLoader(path, opts...)
	})
}
