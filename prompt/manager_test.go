package prompt

import (
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

// inMemoryManager is a test implementation of PromptManager backed by an
// in-memory map of templates keyed by "name:version".
type inMemoryManager struct {
	templates map[string]*Template // key: "name" or "name:version"
}

func newInMemoryManager() *inMemoryManager {
	return &inMemoryManager{templates: make(map[string]*Template)}
}

func (m *inMemoryManager) add(t *Template) {
	// Store under name:version for versioned lookup.
	if t.Version != "" {
		m.templates[t.Name+":"+t.Version] = t
	}
	// Store under name for latest (overwrites previous latest).
	m.templates[t.Name] = t
}

func (m *inMemoryManager) addVersionOnly(t *Template) {
	// Store under name:version only (no latest alias).
	if t.Version != "" {
		m.templates[t.Name+":"+t.Version] = t
	}
}

func (m *inMemoryManager) Get(name string, version string) (*Template, error) {
	if version != "" {
		key := name + ":" + version
		if t, ok := m.templates[key]; ok {
			return t, nil
		}
		return nil, &templateNotFoundError{name: name, version: version}
	}
	if t, ok := m.templates[name]; ok {
		return t, nil
	}
	return nil, &templateNotFoundError{name: name, version: version}
}

func (m *inMemoryManager) Render(name string, vars map[string]any) ([]schema.Message, error) {
	t, err := m.Get(name, "")
	if err != nil {
		return nil, err
	}
	rendered, err := t.Render(vars)
	if err != nil {
		return nil, err
	}
	return []schema.Message{schema.NewSystemMessage(rendered)}, nil
}

func (m *inMemoryManager) List() []TemplateInfo {
	seen := make(map[string]bool)
	var infos []TemplateInfo
	for _, t := range m.templates {
		if seen[t.Name] {
			continue
		}
		seen[t.Name] = true
		infos = append(infos, TemplateInfo{
			Name:     t.Name,
			Version:  t.Version,
			Metadata: t.Metadata,
		})
	}
	return infos
}

type templateNotFoundError struct {
	name    string
	version string
}

func (e *templateNotFoundError) Error() string {
	if e.version != "" {
		return "template not found: " + e.name + ":" + e.version
	}
	return "template not found: " + e.name
}

// Verify inMemoryManager implements PromptManager.
var _ PromptManager = (*inMemoryManager)(nil)

func TestPromptManager_Get_Found(t *testing.T) {
	mgr := newInMemoryManager()
	mgr.add(&Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello, {{.name}}!",
	})

	tmpl, err := mgr.Get("greeting", "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if tmpl.Name != "greeting" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "greeting")
	}
	if tmpl.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", tmpl.Version, "1.0.0")
	}
}

func TestPromptManager_Get_ByVersion(t *testing.T) {
	mgr := newInMemoryManager()
	mgr.add(&Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello v1!",
	})
	mgr.add(&Template{
		Name:    "greeting",
		Version: "2.0.0",
		Content: "Hello v2!",
	})

	tmpl, err := mgr.Get("greeting", "2.0.0")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if tmpl.Content != "Hello v2!" {
		t.Errorf("Content = %q, want %q", tmpl.Content, "Hello v2!")
	}
}

func TestPromptManager_Get_NotFound(t *testing.T) {
	mgr := newInMemoryManager()
	_, err := mgr.Get("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestPromptManager_Get_VersionNotFound(t *testing.T) {
	mgr := newInMemoryManager()
	mgr.add(&Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello!",
	})

	_, err := mgr.Get("greeting", "9.9.9")
	if err == nil {
		t.Fatal("expected error for nonexistent version")
	}
}

func TestPromptManager_Render_Success(t *testing.T) {
	mgr := newInMemoryManager()
	mgr.add(&Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello, {{.name}}!",
	})

	msgs, err := mgr.Render("greeting", map[string]any{"name": "Alice"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].GetRole() != schema.RoleSystem {
		t.Errorf("role = %s, want system", msgs[0].GetRole())
	}
	text := extractMessageText(msgs[0])
	if text != "Hello, Alice!" {
		t.Errorf("rendered text = %q, want %q", text, "Hello, Alice!")
	}
}

func TestPromptManager_Render_NotFound(t *testing.T) {
	mgr := newInMemoryManager()
	_, err := mgr.Render("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestPromptManager_Render_WithDefaults(t *testing.T) {
	mgr := newInMemoryManager()
	mgr.add(&Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello, {{.name}}!",
		Variables: map[string]string{
			"name": "World",
		},
	})

	msgs, err := mgr.Render("greeting", nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	text := extractMessageText(msgs[0])
	if text != "Hello, World!" {
		t.Errorf("rendered text = %q, want %q", text, "Hello, World!")
	}
}

func TestPromptManager_List_Empty(t *testing.T) {
	mgr := newInMemoryManager()
	infos := mgr.List()
	if len(infos) != 0 {
		t.Errorf("expected 0 infos, got %d", len(infos))
	}
}

func TestPromptManager_List_WithTemplates(t *testing.T) {
	mgr := newInMemoryManager()
	mgr.add(&Template{
		Name:    "greeting",
		Version: "1.0.0",
		Content: "Hello!",
		Metadata: map[string]any{
			"author": "test",
		},
	})
	mgr.add(&Template{
		Name:    "farewell",
		Version: "1.0.0",
		Content: "Goodbye!",
	})

	infos := mgr.List()
	if len(infos) != 2 {
		t.Fatalf("expected 2 infos, got %d", len(infos))
	}

	// Check that both templates are listed.
	names := map[string]bool{}
	for _, info := range infos {
		names[info.Name] = true
	}
	if !names["greeting"] {
		t.Error("expected 'greeting' in list")
	}
	if !names["farewell"] {
		t.Error("expected 'farewell' in list")
	}
}

func TestTemplateInfo_Fields(t *testing.T) {
	info := TemplateInfo{
		Name:    "test",
		Version: "2.0.0",
		Metadata: map[string]any{
			"author": "bot",
			"tags":   []string{"system"},
		},
	}
	if info.Name != "test" {
		t.Errorf("Name = %q, want %q", info.Name, "test")
	}
	if info.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", info.Version, "2.0.0")
	}
	if info.Metadata["author"] != "bot" {
		t.Errorf("Metadata[author] = %v, want %q", info.Metadata["author"], "bot")
	}
}

func TestPromptManager_Render_InvalidTemplate(t *testing.T) {
	mgr := newInMemoryManager()
	mgr.add(&Template{
		Name:    "",
		Content: "invalid",
	})

	// The template name is empty, which should fail validation.
	_, err := mgr.Render("", nil)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

// extractMessageText extracts text from a message's content parts.
func extractMessageText(msg schema.Message) string {
	for _, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}
