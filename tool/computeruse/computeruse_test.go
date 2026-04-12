package computeruse

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/internal/testutil/mockllm"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock implementations ---

type mockAction struct {
	lastReq    ActionRequest
	screenshot []byte
	result     *ActionResult
	err        error
}

func (m *mockAction) Execute(_ context.Context, req ActionRequest) (*ActionResult, error) {
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &ActionResult{Success: true, Description: "action executed"}, nil
}

func (m *mockAction) Screenshot(_ context.Context) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.screenshot, nil
}

type mockBrowser struct {
	lastURL    string
	lastText   string
	lastX      int
	lastY      int
	screenshot []byte
	err        error
}

func (m *mockBrowser) Navigate(_ context.Context, url string) error {
	m.lastURL = url
	return m.err
}

func (m *mockBrowser) Screenshot(_ context.Context) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.screenshot, nil
}

func (m *mockBrowser) Click(_ context.Context, x, y int) error {
	m.lastX, m.lastY = x, y
	return m.err
}

func (m *mockBrowser) Type(_ context.Context, text string) error {
	m.lastText = text
	return m.err
}

// mockChatModel adapts mockllm to llm.ChatModel.
type mockChatModel struct {
	*mockllm.MockChatModel
}

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	return m.MockChatModel.Generate(ctx, msgs)
}

func (m *mockChatModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return m.MockChatModel.Stream(ctx, msgs)
}

func (m *mockChatModel) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	return &mockChatModel{m.MockChatModel.BindTools(tools)}
}

// --- SafetyGuard tests ---

func TestSafetyGuard_CheckURL(t *testing.T) {
	tests := []struct {
		name     string
		hosts    []string
		block    []string
		url      string
		wantErr  bool
		wantCode core.ErrorCode
	}{
		{
			name:  "allowed host",
			hosts: []string{"example.com"},
			url:   "https://example.com/page",
		},
		{
			name:     "blocked host",
			hosts:    []string{"example.com"},
			url:      "https://evil.com/page",
			wantErr:  true,
			wantCode: core.ErrGuardBlocked,
		},
		{
			name: "no allowlist permits all",
			url:  "https://anything.com",
		},
		{
			name:     "block pattern match",
			block:    []string{"/admin"},
			url:      "https://example.com/admin/settings",
			wantErr:  true,
			wantCode: core.ErrGuardBlocked,
		},
		{
			name:     "invalid URL",
			url:      "://bad",
			wantErr:  true,
			wantCode: core.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []GuardOption{}
			if len(tt.hosts) > 0 {
				opts = append(opts, WithAllowedHosts(tt.hosts...))
			}
			if len(tt.block) > 0 {
				opts = append(opts, WithBlockPatterns(tt.block...))
			}
			guard := NewSafetyGuard(opts...)

			err := guard.CheckURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				var coreErr *core.Error
				require.ErrorAs(t, err, &coreErr)
				assert.Equal(t, tt.wantCode, coreErr.Code)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSafetyGuard_CheckAction_RateLimit(t *testing.T) {
	guard := NewSafetyGuard(WithMaxActionsPerMinute(3))

	require.NoError(t, guard.CheckAction())
	require.NoError(t, guard.CheckAction())
	require.NoError(t, guard.CheckAction())

	err := guard.CheckAction()
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrRateLimit, coreErr.Code)
}

func TestSafetyGuard_ActionsInWindow(t *testing.T) {
	guard := NewSafetyGuard(WithMaxActionsPerMinute(10))
	_ = guard.CheckAction()
	_ = guard.CheckAction()
	assert.Equal(t, 2, guard.ActionsInWindow())
}

// --- ScreenAnalyzer tests ---

func TestScreenAnalyzer_Analyze(t *testing.T) {
	model := &mockChatModel{mockllm.New(mockllm.WithResponse(schema.NewAIMessage("A login page with username and password fields")))}
	analyzer, err := NewScreenAnalyzer(WithAnalyzerModel(model))
	require.NoError(t, err)

	desc, err := analyzer.Analyze(context.Background(), []byte("fake-png"))
	require.NoError(t, err)
	assert.Contains(t, desc, "login page")
}

func TestScreenAnalyzer_EmptyScreenshot(t *testing.T) {
	model := &mockChatModel{mockllm.New()}
	analyzer, err := NewScreenAnalyzer(WithAnalyzerModel(model))
	require.NoError(t, err)

	_, err = analyzer.Analyze(context.Background(), nil)
	require.Error(t, err)
}

func TestScreenAnalyzer_NoModel(t *testing.T) {
	_, err := NewScreenAnalyzer()
	require.Error(t, err)
}

// --- ComputerUseTool tests ---

func TestComputerUseTool_Name(t *testing.T) {
	action := &mockAction{screenshot: []byte("png")}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)
	assert.Equal(t, "computer_use", ct.Name())
}

func TestComputerUseTool_Description(t *testing.T) {
	action := &mockAction{screenshot: []byte("png")}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)
	assert.NotEmpty(t, ct.Description())
}

func TestComputerUseTool_InputSchema(t *testing.T) {
	action := &mockAction{screenshot: []byte("png")}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)
	schema := ct.InputSchema()
	assert.Equal(t, "object", schema["type"])
}

func TestComputerUseTool_NoBackend(t *testing.T) {
	_, err := NewComputerUseTool()
	require.Error(t, err)
}

func TestComputerUseTool_Execute(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
		check   func(t *testing.T, action *mockAction, browser *mockBrowser)
	}{
		{
			name:  "screenshot via action",
			input: map[string]any{"action": "screenshot"},
		},
		{
			name:  "click via browser",
			input: map[string]any{"action": "click", "x": float64(100), "y": float64(200)},
			check: func(t *testing.T, _ *mockAction, browser *mockBrowser) {
				assert.Equal(t, 100, browser.lastX)
				assert.Equal(t, 200, browser.lastY)
			},
		},
		{
			name:  "type via browser",
			input: map[string]any{"action": "type", "text": "hello"},
			check: func(t *testing.T, _ *mockAction, browser *mockBrowser) {
				assert.Equal(t, "hello", browser.lastText)
			},
		},
		{
			name:  "navigate",
			input: map[string]any{"action": "navigate", "url": "https://example.com"},
			check: func(t *testing.T, _ *mockAction, browser *mockBrowser) {
				assert.Equal(t, "https://example.com", browser.lastURL)
			},
		},
		{
			name:  "key_press via action",
			input: map[string]any{"action": "key_press", "text": "Enter"},
			check: func(t *testing.T, action *mockAction, _ *mockBrowser) {
				assert.Equal(t, ActionKeyPress, action.lastReq.Type)
				assert.Equal(t, "Enter", action.lastReq.Text)
			},
		},
		{
			name:  "missing action",
			input: map[string]any{},
		},
		{
			name:  "unknown action",
			input: map[string]any{"action": "fly"},
		},
		{
			name:  "type missing text",
			input: map[string]any{"action": "type"},
		},
		{
			name:  "navigate missing url",
			input: map[string]any{"action": "navigate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &mockAction{
				screenshot: []byte("fake-png"),
				result:     &ActionResult{Success: true, Description: "done"},
			}
			browser := &mockBrowser{screenshot: []byte("fake-png")}
			guard := NewSafetyGuard(WithAllowedHosts("example.com"))

			ct, err := NewComputerUseTool(
				WithAction(action),
				WithBrowser(browser),
				WithSafetyGuard(guard),
			)
			require.NoError(t, err)

			result, err := ct.Execute(context.Background(), tt.input)
			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.check != nil {
				tt.check(t, action, browser)
			}
		})
	}
}

func TestComputerUseTool_NavigateBlocked(t *testing.T) {
	browser := &mockBrowser{}
	guard := NewSafetyGuard(WithAllowedHosts("safe.com"))

	ct, err := NewComputerUseTool(WithBrowser(browser), WithSafetyGuard(guard))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "navigate",
		"url":    "https://evil.com",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_RateLimited(t *testing.T) {
	action := &mockAction{screenshot: []byte("png")}
	guard := NewSafetyGuard(WithMaxActionsPerMinute(1))

	ct, err := NewComputerUseTool(WithAction(action), WithSafetyGuard(guard))
	require.NoError(t, err)

	// First action succeeds.
	result, err := ct.Execute(context.Background(), map[string]any{"action": "screenshot"})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	// Second action rate limited.
	result, err = ct.Execute(context.Background(), map[string]any{"action": "screenshot"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_BrowserError(t *testing.T) {
	browser := &mockBrowser{err: errors.New("browser crashed")}
	ct, err := NewComputerUseTool(WithBrowser(browser))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "navigate",
		"url":    "https://example.com",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_ScreenshotWithAnalyzer(t *testing.T) {
	model := &mockChatModel{mockllm.New(mockllm.WithResponse(schema.NewAIMessage("A web page")))}
	analyzer, err := NewScreenAnalyzer(WithAnalyzerModel(model))
	require.NoError(t, err)

	browser := &mockBrowser{screenshot: []byte("fake-png")}
	ct, err := NewComputerUseTool(WithBrowser(browser), WithScreenAnalyzer(analyzer))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{"action": "screenshot"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Should have image part + text description.
	assert.Len(t, result.Content, 2)
}

func TestToInt(t *testing.T) {
	assert.Equal(t, 42, toInt(float64(42)))
	assert.Equal(t, 42, toInt(42))
	assert.Equal(t, 0, toInt("not a number"))
	assert.Equal(t, 0, toInt(nil))
	assert.Equal(t, 99, toInt(json.Number("99")))
}

// mockExtendedBrowser is a BrowserBackend that also implements ExtendedBrowser.
type mockExtendedBrowser struct {
	mockBrowser
	lastScrollX     int
	lastScrollY     int
	lastScrollDelta int
	scrollErr       error
}

func (m *mockExtendedBrowser) Scroll(_ context.Context, x, y, delta int) error {
	m.lastScrollX = x
	m.lastScrollY = y
	m.lastScrollDelta = delta
	return m.scrollErr
}

func (m *mockExtendedBrowser) ExecuteJS(_ context.Context, _ string) (string, error) {
	return "", nil
}

func TestComputerUseTool_Scroll(t *testing.T) {
	tests := []struct {
		name       string
		useAction  bool
		useBrowser bool
		extBrowser bool
		scrollErr  bool
		input      map[string]any
		wantError  bool
	}{
		{
			name:       "scroll via extended browser",
			useBrowser: true,
			extBrowser: true,
			input:      map[string]any{"action": "scroll", "x": float64(10), "y": float64(20), "scroll_delta": float64(3)},
		},
		{
			name:       "scroll via extended browser error",
			useBrowser: true,
			extBrowser: true,
			scrollErr:  true,
			input:      map[string]any{"action": "scroll", "x": float64(10), "y": float64(20), "scroll_delta": float64(3)},
			wantError:  true,
		},
		{
			name:      "scroll via action backend",
			useAction: true,
			input:     map[string]any{"action": "scroll", "x": float64(5), "y": float64(5), "scroll_delta": float64(-1)},
		},
		{
			name:       "scroll no action backend (plain browser)",
			useBrowser: true,
			input:      map[string]any{"action": "scroll", "x": float64(0), "y": float64(0), "scroll_delta": float64(1)},
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []ToolOption
			var extBr *mockExtendedBrowser
			var act *mockAction

			if tt.extBrowser {
				scrollErr := error(nil)
				if tt.scrollErr {
					scrollErr = errors.New("scroll failed")
				}
				extBr = &mockExtendedBrowser{scrollErr: scrollErr}
				opts = append(opts, WithBrowser(extBr))
			} else if tt.useBrowser {
				opts = append(opts, WithBrowser(&mockBrowser{}))
			}

			if tt.useAction {
				act = &mockAction{}
				opts = append(opts, WithAction(act))
			}

			ct, err := NewComputerUseTool(opts...)
			require.NoError(t, err)

			result, err := ct.Execute(context.Background(), tt.input)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantError, result.IsError)

			if extBr != nil && !tt.scrollErr {
				assert.Equal(t, 10, extBr.lastScrollX)
				assert.Equal(t, 20, extBr.lastScrollY)
				assert.Equal(t, 3, extBr.lastScrollDelta)
			}
		})
	}
}

func TestComputerUseTool_ClickViaAction(t *testing.T) {
	action := &mockAction{result: &ActionResult{Success: true, Description: "clicked"}}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "click", "x": float64(50), "y": float64(60),
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, ActionClick, action.lastReq.Type)
}

func TestComputerUseTool_ClickViaActionError(t *testing.T) {
	action := &mockAction{err: errors.New("action failed")}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "click", "x": float64(50), "y": float64(60),
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_ClickViaActionNoBrowser(t *testing.T) {
	action := &mockAction{}
	browser := &mockBrowser{err: errors.New("click failed")}
	ct, err := NewComputerUseTool(WithAction(action), WithBrowser(browser))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "click", "x": float64(50), "y": float64(60),
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_TypeViaAction(t *testing.T) {
	action := &mockAction{result: &ActionResult{Success: true, Description: "typed"}}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "type", "text": "hello world",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, ActionTypeText, action.lastReq.Type)
	assert.Equal(t, "hello world", action.lastReq.Text)
}

func TestComputerUseTool_TypeViaActionError(t *testing.T) {
	action := &mockAction{err: errors.New("type failed")}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "type", "text": "hello",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_TypeViaActionNoBrowserText(t *testing.T) {
	action := &mockAction{}
	browser := &mockBrowser{err: errors.New("type failed")}
	ct, err := NewComputerUseTool(WithAction(action), WithBrowser(browser))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "type", "text": "hello",
	})
	require.NoError(t, err)
	// browser Type error returns an error result, but browser path is tried first
	assert.True(t, result.IsError)
}

func TestComputerUseTool_KeyPressMissingText(t *testing.T) {
	action := &mockAction{}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{"action": "key_press"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_KeyPressActionError(t *testing.T) {
	action := &mockAction{err: errors.New("key press failed")}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{
		"action": "key_press", "text": "Escape",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_ScreenshotActionError(t *testing.T) {
	action := &mockAction{err: errors.New("screenshot failed")}
	ct, err := NewComputerUseTool(WithAction(action))
	require.NoError(t, err)

	result, err := ct.Execute(context.Background(), map[string]any{"action": "screenshot"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestComputerUseTool_InputSchema_BrowserOnly(t *testing.T) {
	browser := &mockBrowser{}
	ct, err := NewComputerUseTool(WithBrowser(browser))
	require.NoError(t, err)

	schema := ct.InputSchema()
	props := schema["properties"].(map[string]any)
	actionSchema := props["action"].(map[string]any)
	actions := actionSchema["enum"].([]string)

	// key_press should not be in actions list when no ComputerAction backend
	for _, a := range actions {
		assert.NotEqual(t, "key_press", a)
	}
	// navigate should be present for browser backend
	found := false
	for _, a := range actions {
		if a == "navigate" {
			found = true
		}
	}
	assert.True(t, found, "navigate should be advertised when browser backend is set")
}

func TestSafetyGuard_CheckURL_SchemeBlocked(t *testing.T) {
	guard := NewSafetyGuard()
	tests := []struct {
		name string
		url  string
	}{
		{"file scheme", "file:///etc/passwd"},
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/html,<h1>test</h1>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := guard.CheckURL(tt.url)
			require.Error(t, err)
			var coreErr *core.Error
			require.ErrorAs(t, err, &coreErr)
			assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
		})
	}
}

func TestSafetyGuard_WithAllowedHosts_IPv6(t *testing.T) {
	// IPv6 addresses: url.Hostname() strips brackets, returning "::1".
	// WithAllowedHosts must normalize "[::1]" to "::1" so that the comparison
	// is consistent (the colon-check detects the colons inside the brackets
	// and leaves the entry unchanged because strings.Contains(normalized[:i], ":")
	// is true for an IPv6 address).
	guard := NewSafetyGuard(WithAllowedHosts("::1"))
	err := guard.CheckURL("https://[::1]/page")
	require.NoError(t, err)
}

func TestWithAnalyzerPrompt(t *testing.T) {
	model := &mockChatModel{mockllm.New(mockllm.WithResponse(schema.NewAIMessage("described")))}
	analyzer, err := NewScreenAnalyzer(
		WithAnalyzerModel(model),
		WithAnalyzerPrompt("custom prompt"),
	)
	require.NoError(t, err)
	assert.Equal(t, "custom prompt", analyzer.opts.prompt)

	desc, err := analyzer.Analyze(context.Background(), []byte("fake-png"))
	require.NoError(t, err)
	assert.Equal(t, "described", desc)
}
