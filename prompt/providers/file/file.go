// Package file provides a filesystem-based PromptManager that loads prompt
// templates from a directory of JSON files. Each JSON file represents a single
// template version with name, version, content, and variables fields.
package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/prompt"
	"github.com/lookatitude/beluga-ai/schema"
)

// FileManager implements prompt.PromptManager by loading templates from
// JSON files in a directory. Files must have a .json extension and contain
// a valid prompt.Template structure.
type FileManager struct {
	dir string

	mu        sync.RWMutex
	templates map[string][]*prompt.Template // name → versions (sorted newest first)
}

// NewFileManager creates a FileManager that loads templates from the given
// directory. All .json files in the directory are parsed on creation.
// Returns an error if the directory cannot be read or any file is invalid.
func NewFileManager(dir string) (*FileManager, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("prompt/file: cannot access directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("prompt/file: %q is not a directory", dir)
	}

	fm := &FileManager{
		dir:       dir,
		templates: make(map[string][]*prompt.Template),
	}

	if err := fm.load(); err != nil {
		return nil, err
	}

	return fm, nil
}

// load reads all .json files from the directory and parses them as templates.
func (fm *FileManager) load() error {
	entries, err := os.ReadDir(fm.dir)
	if err != nil {
		return fmt.Errorf("prompt/file: reading directory %q: %w", fm.dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(fm.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("prompt/file: reading %q: %w", path, err)
		}

		var tmpl prompt.Template
		if err := json.Unmarshal(data, &tmpl); err != nil {
			return fmt.Errorf("prompt/file: parsing %q: %w", path, err)
		}

		if err := tmpl.Validate(); err != nil {
			return fmt.Errorf("prompt/file: validating %q: %w", path, err)
		}

		fm.templates[tmpl.Name] = append(fm.templates[tmpl.Name], &tmpl)
	}

	// Sort versions for each template name (lexicographic descending so latest is first).
	for _, versions := range fm.templates {
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].Version > versions[j].Version
		})
	}

	return nil
}

// Get retrieves a template by name and version. If version is empty,
// the latest version (lexicographically highest) is returned.
func (fm *FileManager) Get(name string, version string) (*prompt.Template, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	versions, ok := fm.templates[name]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("prompt/file: template %q not found", name)
	}

	if version == "" {
		return versions[0], nil
	}

	for _, t := range versions {
		if t.Version == version {
			return t, nil
		}
	}

	return nil, fmt.Errorf("prompt/file: template %q version %q not found", name, version)
}

// Render retrieves a template by name (latest version), renders it with the
// given variables, and returns the result as a single SystemMessage.
func (fm *FileManager) Render(name string, vars map[string]any) ([]schema.Message, error) {
	tmpl, err := fm.Get(name, "")
	if err != nil {
		return nil, err
	}

	rendered, err := tmpl.Render(vars)
	if err != nil {
		return nil, err
	}

	return []schema.Message{schema.NewSystemMessage(rendered)}, nil
}

// List returns summary information for all loaded templates. Each version
// of a template is returned as a separate entry.
func (fm *FileManager) List() []prompt.TemplateInfo {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	var infos []prompt.TemplateInfo
	for _, versions := range fm.templates {
		for _, t := range versions {
			infos = append(infos, prompt.TemplateInfo{
				Name:     t.Name,
				Version:  t.Version,
				Metadata: t.Metadata,
			})
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Name != infos[j].Name {
			return infos[i].Name < infos[j].Name
		}
		return infos[i].Version > infos[j].Version
	})

	return infos
}
