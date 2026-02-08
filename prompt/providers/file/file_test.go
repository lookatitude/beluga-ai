package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lookatitude/beluga-ai/prompt"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTemplate is a test helper that writes a template to a JSON file.
func writeTemplate(t *testing.T, dir, filename string, tmpl prompt.Template) {
	t.Helper()
	data, err := json.Marshal(tmpl)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, filename), data, 0644)
	require.NoError(t, err)
}

// TestInterfaceCompliance verifies FileManager implements PromptManager.
func TestInterfaceCompliance(t *testing.T) {
	var _ prompt.PromptManager = (*FileManager)(nil)
}

func TestNewFileManager_ValidDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create valid templates
	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello {{.name}}!",
	})
	writeTemplate(t, dir, "farewell.json", prompt.Template{
		Name:    "farewell",
		Version: "1.0.0",
		Content: "Goodbye {{.name}}!",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)
	assert.NotNil(t, fm)
	assert.Equal(t, dir, fm.dir)
	assert.Len(t, fm.templates, 2)
}

func TestNewFileManager_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	fm, err := NewFileManager(dir)
	require.NoError(t, err)
	assert.NotNil(t, fm)
	assert.Len(t, fm.templates, 0)
}

func TestNewFileManager_NonExistentDirectory(t *testing.T) {
	fm, err := NewFileManager("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
	assert.Nil(t, fm)
	assert.Contains(t, err.Error(), "cannot access directory")
}

func TestNewFileManager_PathIsFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notadir.txt")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	fm, err := NewFileManager(filePath)
	assert.Error(t, err)
	assert.Nil(t, fm)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestNewFileManager_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	// Write invalid JSON
	err := os.WriteFile(filepath.Join(dir, "invalid.json"), []byte("{invalid json"), 0644)
	require.NoError(t, err)

	fm, err := NewFileManager(dir)
	assert.Error(t, err)
	assert.Nil(t, fm)
	assert.Contains(t, err.Error(), "parsing")
}

func TestNewFileManager_InvalidTemplate_EmptyName(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "badtemplate.json", prompt.Template{
		Name:    "", // invalid: empty name
		Version: "1.0.0",
		Content: "Hello world",
	})

	fm, err := NewFileManager(dir)
	assert.Error(t, err)
	assert.Nil(t, fm)
	assert.Contains(t, err.Error(), "validating")
}

func TestNewFileManager_InvalidTemplate_EmptyContent(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "badtemplate.json", prompt.Template{
		Name:    "test",
		Version: "1.0.0",
		Content: "", // invalid: empty content
	})

	fm, err := NewFileManager(dir)
	assert.Error(t, err)
	assert.Nil(t, fm)
	assert.Contains(t, err.Error(), "validating")
}

func TestNewFileManager_InvalidTemplate_BadTemplateSyntax(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "badtemplate.json", prompt.Template{
		Name:    "test",
		Version: "1.0.0",
		Content: "Hello {{.name", // invalid: unclosed template
	})

	fm, err := NewFileManager(dir)
	assert.Error(t, err)
	assert.Nil(t, fm)
	assert.Contains(t, err.Error(), "validating")
}

func TestNewFileManager_NonJSONFilesSkipped(t *testing.T) {
	dir := t.TempDir()

	// Create a valid JSON template
	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello!",
	})

	// Create a .txt file that should be ignored
	err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("This is not a template"), 0644)
	require.NoError(t, err)

	fm, err := NewFileManager(dir)
	require.NoError(t, err)
	assert.Len(t, fm.templates, 1)
	assert.Contains(t, fm.templates, "greeting")
}

func TestNewFileManager_SubdirectoriesSkipped(t *testing.T) {
	dir := t.TempDir()

	// Create a valid template
	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello!",
	})

	// Create a subdirectory with a template (should be ignored)
	subdir := filepath.Join(dir, "subdir")
	err := os.Mkdir(subdir, 0755)
	require.NoError(t, err)
	writeTemplate(t, subdir, "ignored.json", prompt.Template{
		Name:    "ignored",
		Version: "1.0.0",
		Content: "Should not load",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)
	assert.Len(t, fm.templates, 1)
	assert.Contains(t, fm.templates, "greeting")
	assert.NotContains(t, fm.templates, "ignored")
}

func TestGet_LatestVersion_EmptyVersionString(t *testing.T) {
	dir := t.TempDir()

	// Create multiple versions of the same template
	writeTemplate(t, dir, "greeting-1.0.0.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello v1.0.0",
	})
	writeTemplate(t, dir, "greeting-1.1.0.json", prompt.Template{
		Name:    "greeting",
		Version: "1.1.0",
		Content: "Hello v1.1.0",
	})
	writeTemplate(t, dir, "greeting-2.0.0.json", prompt.Template{
		Name:    "greeting",
		Version: "2.0.0",
		Content: "Hello v2.0.0",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Get latest version (empty version string)
	tmpl, err := fm.Get("greeting", "")
	require.NoError(t, err)
	assert.Equal(t, "greeting", tmpl.Name)
	assert.Equal(t, "2.0.0", tmpl.Version)
	assert.Equal(t, "Hello v2.0.0", tmpl.Content)
}

func TestGet_SpecificVersion(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting-1.0.0.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello v1.0.0",
	})
	writeTemplate(t, dir, "greeting-2.0.0.json", prompt.Template{
		Name:    "greeting",
		Version: "2.0.0",
		Content: "Hello v2.0.0",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Get specific version
	tmpl, err := fm.Get("greeting", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "greeting", tmpl.Name)
	assert.Equal(t, "1.0.0", tmpl.Version)
	assert.Equal(t, "Hello v1.0.0", tmpl.Content)
}

func TestGet_UnknownName(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello!",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	tmpl, err := fm.Get("nonexistent", "")
	assert.Error(t, err)
	assert.Nil(t, tmpl)
	assert.Contains(t, err.Error(), "not found")
}

func TestGet_UnknownVersion(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello!",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	tmpl, err := fm.Get("greeting", "99.0.0")
	assert.Error(t, err)
	assert.Nil(t, tmpl)
	assert.Contains(t, err.Error(), "version")
	assert.Contains(t, err.Error(), "not found")
}

func TestRender_WithVariables(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello {{.name}}, welcome to {{.service}}!",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	msgs, err := fm.Render("greeting", map[string]any{
		"name":    "Alice",
		"service": "Beluga AI",
	})
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	systemMsg, ok := msgs[0].(*schema.SystemMessage)
	require.True(t, ok)
	assert.Equal(t, "Hello Alice, welcome to Beluga AI!", systemMsg.Text())
}

func TestRender_WithDefaultVariables(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello {{.name}}, welcome to {{.service}}!",
		Variables: map[string]string{
			"name":    "Guest",
			"service": "Beluga AI",
		},
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Render without providing variables (should use defaults)
	msgs, err := fm.Render("greeting", nil)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	systemMsg, ok := msgs[0].(*schema.SystemMessage)
	require.True(t, ok)
	assert.Equal(t, "Hello Guest, welcome to Beluga AI!", systemMsg.Text())
}

func TestRender_OverrideDefaultVariables(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello {{.name}}, welcome to {{.service}}!",
		Variables: map[string]string{
			"name":    "Guest",
			"service": "Beluga AI",
		},
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Override one default variable
	msgs, err := fm.Render("greeting", map[string]any{
		"name": "Bob",
	})
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	systemMsg, ok := msgs[0].(*schema.SystemMessage)
	require.True(t, ok)
	assert.Equal(t, "Hello Bob, welcome to Beluga AI!", systemMsg.Text())
}

func TestRender_UnknownTemplate(t *testing.T) {
	dir := t.TempDir()

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	msgs, err := fm.Render("nonexistent", nil)
	assert.Error(t, err)
	assert.Nil(t, msgs)
	assert.Contains(t, err.Error(), "not found")
}

func TestList_AllTemplatesSorted(t *testing.T) {
	dir := t.TempDir()

	// Create multiple templates with multiple versions
	writeTemplate(t, dir, "alpha-1.0.0.json", prompt.Template{
		Name:     "alpha",
		Version:  "1.0.0",
		Content:  "Alpha v1",
		Metadata: map[string]any{"author": "test"},
	})
	writeTemplate(t, dir, "alpha-2.0.0.json", prompt.Template{
		Name:     "alpha",
		Version:  "2.0.0",
		Content:  "Alpha v2",
		Metadata: map[string]any{"author": "test"},
	})
	writeTemplate(t, dir, "beta-1.0.0.json", prompt.Template{
		Name:     "beta",
		Version:  "1.0.0",
		Content:  "Beta v1",
		Metadata: map[string]any{"author": "test"},
	})
	writeTemplate(t, dir, "charlie-1.5.0.json", prompt.Template{
		Name:     "charlie",
		Version:  "1.5.0",
		Content:  "Charlie v1.5",
		Metadata: map[string]any{"author": "test"},
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	infos := fm.List()
	require.Len(t, infos, 4)

	// Verify sorting: name ascending, then version descending
	expected := []struct {
		name    string
		version string
	}{
		{"alpha", "2.0.0"},
		{"alpha", "1.0.0"},
		{"beta", "1.0.0"},
		{"charlie", "1.5.0"},
	}

	for i, exp := range expected {
		assert.Equal(t, exp.name, infos[i].Name)
		assert.Equal(t, exp.version, infos[i].Version)
		assert.NotNil(t, infos[i].Metadata)
	}
}

func TestList_EmptyManager(t *testing.T) {
	dir := t.TempDir()

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	infos := fm.List()
	assert.Empty(t, infos)
}

func TestVersionSorting_Lexicographic(t *testing.T) {
	dir := t.TempDir()

	// Create versions in non-sorted order
	writeTemplate(t, dir, "test-1.10.0.json", prompt.Template{
		Name:    "test",
		Version: "1.10.0",
		Content: "v1.10.0",
	})
	writeTemplate(t, dir, "test-1.2.0.json", prompt.Template{
		Name:    "test",
		Version: "1.2.0",
		Content: "v1.2.0",
	})
	writeTemplate(t, dir, "test-1.20.0.json", prompt.Template{
		Name:    "test",
		Version: "1.20.0",
		Content: "v1.20.0",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Get latest (should be lexicographically highest)
	tmpl, err := fm.Get("test", "")
	require.NoError(t, err)
	// "1.20.0" < "1.3.0" lexicographically, but "1.20.0" > "1.10.0" > "1.2.0"
	assert.Equal(t, "1.20.0", tmpl.Version)
}

func TestConcurrentAccess(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello {{.name}}!",
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_, err := fm.Get("greeting", "")
			assert.NoError(t, err)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent renders
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_, err := fm.Render("greeting", map[string]any{"name": "Test"})
			assert.NoError(t, err)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent lists
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			infos := fm.List()
			assert.Len(t, infos, 1)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTemplateWithMetadata(t *testing.T) {
	dir := t.TempDir()

	writeTemplate(t, dir, "greeting.json", prompt.Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello!",
		Metadata: map[string]any{
			"author":      "test-author",
			"description": "A simple greeting",
			"tags":        []string{"greeting", "simple"},
		},
	})

	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	infos := fm.List()
	require.Len(t, infos, 1)
	assert.Equal(t, "test-author", infos[0].Metadata["author"])
	assert.Equal(t, "A simple greeting", infos[0].Metadata["description"])

	tmpl, err := fm.Get("greeting", "1.0.0")
	require.NoError(t, err)
	assert.NotNil(t, tmpl.Metadata)
	assert.Equal(t, "test-author", tmpl.Metadata["author"])
}
