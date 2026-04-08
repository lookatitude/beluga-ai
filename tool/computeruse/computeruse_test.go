package computeruse

import (
	"context"
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
}
