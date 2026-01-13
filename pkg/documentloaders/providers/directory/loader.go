package directory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// DirectoryConfig is duplicated here to avoid import cycle.
// It matches the structure in the parent package.
type DirectoryConfig struct {
	MaxDepth       int
	Extensions     []string
	Concurrency    int
	MaxFileSize    int64
	FollowSymlinks bool
}

// RecursiveDirectoryLoader loads documents from a directory structure recursively.
type RecursiveDirectoryLoader struct {
	fsys    fs.FS
	config  *DirectoryConfig
	tracer  trace.Tracer
	metrics interface{} // Will be implemented when metrics are integrated
}

// NewRecursiveDirectoryLoader creates a new RecursiveDirectoryLoader.
func NewRecursiveDirectoryLoader(fsys fs.FS, config *DirectoryConfig) (iface.DocumentLoader, error) {
	// Basic validation
	if config.MaxDepth < 0 {
		return nil, newLoaderError("NewRecursiveDirectoryLoader", ErrCodeInvalidConfig, "", "MaxDepth must be >= 0", nil)
	}
	if config.Concurrency < 1 {
		return nil, newLoaderError("NewRecursiveDirectoryLoader", ErrCodeInvalidConfig, "", "Concurrency must be >= 1", nil)
	}
	if config.MaxFileSize < 1 {
		return nil, newLoaderError("NewRecursiveDirectoryLoader", ErrCodeInvalidConfig, "", "MaxFileSize must be >= 1", nil)
	}

	return &RecursiveDirectoryLoader{
		fsys:   fsys,
		config: config,
		tracer: otel.Tracer("github.com/lookatitude/beluga-ai/pkg/documentloaders/directory"),
	}, nil
}

// Load implements the DocumentLoader interface.
func (l *RecursiveDirectoryLoader) Load(ctx context.Context) ([]schema.Document, error) {
	ctx, span := l.tracer.Start(ctx, "documentloaders.directory.Load",
		trace.WithAttributes(
			attribute.String("loader.type", "directory"),
			attribute.Int("loader.max_depth", l.config.MaxDepth),
			attribute.Int("loader.concurrency", l.config.Concurrency),
		))
	defer span.End()

	start := time.Now()
	var documents []schema.Document
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	// Worker pool for concurrent file loading
	fileChan := make(chan string, l.config.Concurrency*2)
	workerCount := l.config.Concurrency
	if workerCount <= 0 {
		workerCount = runtime.GOMAXPROCS(0)
	}

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				// Check context cancellation
				select {
				case <-ctx.Done():
					errOnce.Do(func() {
						firstErr = newLoaderError("Load", ErrCodeCancelled, "", "context cancelled", ctx.Err())
					})
					return
				default:
				}

				doc, err := l.loadFile(ctx, filePath)
				if err != nil {
					errOnce.Do(func() {
						firstErr = err
					})
					continue
				}

				if doc != nil {
					mu.Lock()
					documents = append(documents, *doc)
					mu.Unlock()
				}
			}
		}()
	}

	// Track visited inodes for cycle detection (only if FollowSymlinks is enabled)
	var visitedInodes map[uint64]bool
	var visitedMu sync.Mutex
	if l.config.FollowSymlinks {
		visitedInodes = make(map[uint64]bool)
	}

	// Walk directory and send files to workers
	go func() {
		defer close(fileChan)
		err := fs.WalkDir(l.fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Handle symlink following with cycle detection
			if l.config.FollowSymlinks && (d.Type()&fs.ModeSymlink) != 0 {
				// Check for cycles using inode tracking
				resolvedPath, inode, shouldSkip, err := l.resolveSymlink(path, visitedInodes, &visitedMu)
				if err != nil {
					// Log error but continue (symlink might be broken)
					return nil
				}
				if shouldSkip {
					// Cycle detected or already visited
					if d.IsDir() {
						return fs.SkipDir
					}
					return nil
				}

				// If symlink points to a directory, we need to handle it specially
				// For now, we'll continue with the original path but track the inode
				// The resolved path might be outside our fs.FS, so we can't use it directly
				_ = resolvedPath
				_ = inode
			}

			if d.IsDir() {
				// Check MaxDepth for directories
				depth := l.getDepth(path)
				if l.config.MaxDepth > 0 && depth >= l.config.MaxDepth {
					return fs.SkipDir
				}
				return nil
			}

			// For files, check depth of the file's directory
			// Get directory depth by checking the parent path
			fileDepth := l.getDepth(path)
			if l.config.MaxDepth > 0 && fileDepth > l.config.MaxDepth {
				return nil // Skip this file
			}

			// Check extension filter
			if !l.matchesExtension(path) {
				return nil
			}

			// Send to worker pool
			select {
			case fileChan <- path:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			errOnce.Do(func() {
				firstErr = newLoaderError("Load", ErrCodeIOError, "", fmt.Sprintf("directory walk failed: %v", err), err)
			})
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	duration := time.Since(start)

	if firstErr != nil {
		span.RecordError(firstErr)
		span.SetStatus(codes.Error, firstErr.Error())
		return documents, firstErr
	}

	span.SetAttributes(
		attribute.Int("loader.documents_count", len(documents)),
		attribute.Int64("loader.duration_ms", duration.Milliseconds()),
	)
	span.SetStatus(codes.Ok, "")

	return documents, nil
}

// LazyLoad implements the DocumentLoader interface.
func (l *RecursiveDirectoryLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
	ctx, span := l.tracer.Start(ctx, "documentloaders.directory.LazyLoad",
		trace.WithAttributes(
			attribute.String("loader.type", "directory"),
		))
	defer span.End()

	ch := make(chan any, l.config.Concurrency*2)

	go func() {
		defer close(ch)

		fileChan := make(chan string, l.config.Concurrency*2)
		var wg sync.WaitGroup
		workerCount := l.config.Concurrency
		if workerCount <= 0 {
			workerCount = runtime.GOMAXPROCS(0)
		}

		// Start workers
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for filePath := range fileChan {
					select {
					case <-ctx.Done():
						ch <- ctx.Err()
						return
					default:
					}

					doc, err := l.loadFile(ctx, filePath)
					if err != nil {
						ch <- err
						continue
					}

					if doc != nil {
						select {
						case ch <- *doc:
						case <-ctx.Done():
							ch <- ctx.Err()
							return
						}
					}
				}
			}()
		}

		// Track visited inodes for cycle detection (only if FollowSymlinks is enabled)
		var visitedInodes map[uint64]bool
		var visitedMu sync.Mutex
		if l.config.FollowSymlinks {
			visitedInodes = make(map[uint64]bool)
		}

		// Walk directory
		go func() {
			defer close(fileChan)
			if err := fs.WalkDir(l.fsys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					ch <- err
					return nil
				}

				// Handle symlink following with cycle detection
				if l.config.FollowSymlinks && (d.Type()&fs.ModeSymlink) != 0 {
					_, _, shouldSkip, err := l.resolveSymlink(path, visitedInodes, &visitedMu)
					if err != nil {
						// Log error but continue (symlink might be broken)
						return nil
					}
					if shouldSkip {
						// Cycle detected or already visited
						if d.IsDir() {
							return fs.SkipDir
						}
						return nil
					}
				}

				if d.IsDir() {
					depth := l.getDepth(path)
					if depth > l.config.MaxDepth {
						return fs.SkipDir
					}
					return nil
				}

				if !l.matchesExtension(path) {
					return nil
				}

				select {
				case fileChan <- path:
				case <-ctx.Done():
					return ctx.Err()
				}

				return nil
			}); err != nil {
				// Send walk error to error channel
				select {
				case ch <- err:
				case <-ctx.Done():
				}
			}
		}()

		wg.Wait()
	}()

	return ch, nil
}

// loadFile loads a single file and returns a Document.
func (l *RecursiveDirectoryLoader) loadFile(ctx context.Context, filePath string) (*schema.Document, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, newLoaderError("loadFile", ErrCodeCancelled, filePath, "context cancelled", ctx.Err())
	default:
	}

	// Open file
	file, err := l.fsys.Open(filePath)
	if err != nil {
		return nil, newLoaderError("loadFile", ErrCodeIOError, filePath, fmt.Sprintf("failed to open file: %v", err), err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log but don't fail - file is already read
		}
	}()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return nil, newLoaderError("loadFile", ErrCodeIOError, filePath, fmt.Sprintf("failed to stat file: %v", err), err)
	}

	// Check MaxFileSize
	if info.Size() > l.config.MaxFileSize {
		return nil, newLoaderError("loadFile", ErrCodeFileTooLarge, filePath, fmt.Sprintf("file size %d exceeds max %d", info.Size(), l.config.MaxFileSize), nil)
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, newLoaderError("loadFile", ErrCodeIOError, filePath, fmt.Sprintf("failed to read file: %v", err), err)
	}

	// Detect binary content
	if l.isBinary(content) {
		return nil, newLoaderError("loadFile", ErrCodeBinaryFile, filePath, "binary content detected", nil)
	}

	// Create document
	doc := &schema.Document{
		PageContent: string(content),
		Metadata: map[string]string{
			"source":      filePath,
			"file_size":   fmt.Sprintf("%d", info.Size()),
			"modified_at": info.ModTime().Format(time.RFC3339),
			"loader_type": "directory",
		},
	}

	return doc, nil
}

// getDepth calculates the depth of a path.
func (l *RecursiveDirectoryLoader) getDepth(path string) int {
	if path == "." || path == "" {
		return 0
	}
	depth := 0
	for _, char := range path {
		if char == '/' || char == '\\' {
			depth++
		}
	}
	return depth
}

// matchesExtension checks if a file path matches the extension filter.
func (l *RecursiveDirectoryLoader) matchesExtension(path string) bool {
	if len(l.config.Extensions) == 0 {
		return true // No filter means all files
	}

	ext := filepath.Ext(path)
	for _, allowedExt := range l.config.Extensions {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

// isBinary detects if content is binary using HTTP content type detection.
func (l *RecursiveDirectoryLoader) isBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Read first 512 bytes for detection
	sample := content
	if len(sample) > 512 {
		sample = sample[:512]
	}

	contentType := http.DetectContentType(sample)

	// Accept anything that starts with "text/"
	if strings.HasPrefix(contentType, "text/") {
		return false
	}

	// Accept common text-like application types
	if contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/javascript" ||
		strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "application/xml") {
		return false
	}

	// Reject clearly binary types
	if strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "video/") ||
		strings.HasPrefix(contentType, "audio/") ||
		contentType == "application/octet-stream" {
		return true
	}

	// For unknown types, check if content is mostly printable
	// If >80% of bytes are printable ASCII/UTF-8, assume it's text
	printableCount := 0
	checkLen := len(sample)
	if checkLen > 100 {
		checkLen = 100
	}
	for i := 0; i < checkLen; i++ {
		b := sample[i]
		if (b >= 32 && b < 127) || b == 9 || b == 10 || b == 13 {
			printableCount++
		}
	}

	// If less than 80% printable, likely binary
	if checkLen > 0 && float64(printableCount)/float64(checkLen) < 0.8 {
		return true
	}

	// Default: assume text if we can't determine
	return false
}

// resolveSymlink resolves a symlink and checks for cycles using inode tracking.
// Returns the resolved path, inode number, whether to skip (cycle detected), and any error.
func (l *RecursiveDirectoryLoader) resolveSymlink(path string, visitedInodes map[uint64]bool, mu *sync.Mutex) (string, uint64, bool, error) {
	// Try to get the actual OS path if fs.FS is backed by os.DirFS
	// For in-memory filesystems (fstest.MapFS), we can't resolve symlinks
	// In that case, we'll use path-based tracking as a fallback

	// Attempt to resolve using filepath.EvalSymlinks (requires actual OS path)
	// This will only work if the fs.FS is backed by a real filesystem
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If we can't resolve (e.g., in-memory FS), use path-based tracking
		// For now, just track the path itself as a fallback
		mu.Lock()
		defer mu.Unlock()

		// Use path as key for in-memory filesystems
		// This is less robust but works for test scenarios
		if visitedInodes == nil {
			visitedInodes = make(map[uint64]bool)
		}

		// Hash the path to use as a pseudo-inode
		pathHash := l.hashPath(path)
		if visitedInodes[pathHash] {
			return path, 0, true, nil // Cycle detected
		}
		visitedInodes[pathHash] = true
		return path, pathHash, false, nil
	}

	// Get inode from resolved path
	stat, err := os.Stat(resolvedPath)
	if err != nil {
		return resolvedPath, 0, false, err
	}

	// Extract inode (platform-specific)
	var inode uint64
	if sysStat, ok := stat.Sys().(*syscall.Stat_t); ok {
		inode = sysStat.Ino
	} else {
		// Fallback for platforms without syscall.Stat_t (e.g., Windows)
		// Use a hash of the resolved path
		inode = l.hashPath(resolvedPath)
	}

	// Check for cycles
	mu.Lock()
	defer mu.Unlock()

	if visitedInodes == nil {
		visitedInodes = make(map[uint64]bool)
	}

	if visitedInodes[inode] {
		return resolvedPath, inode, true, nil // Cycle detected
	}

	visitedInodes[inode] = true
	return resolvedPath, inode, false, nil
}

// hashPath creates a simple hash from a path string for use as a pseudo-inode.
func (l *RecursiveDirectoryLoader) hashPath(path string) uint64 {
	// Simple FNV-1a hash
	var hash uint64 = 14695981039346656037 // FNV offset basis
	for _, b := range []byte(path) {
		hash ^= uint64(b)
		hash *= 1099511628211 // FNV prime
	}
	return hash
}

// Ensure RecursiveDirectoryLoader implements iface.DocumentLoader
var _ iface.DocumentLoader = (*RecursiveDirectoryLoader)(nil)
