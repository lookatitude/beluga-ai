package simulation

import (
	"context"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/testutil/mockllm"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatModel adapts mockllm.MockChatModel to llm.ChatModel.
type mockChatModel struct {
	*mockllm.MockChatModel
}

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	return m.MockChatModel.Generate(ctx, msgs)
}

func (m *mockChatModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return m.MockChatModel.Stream(ctx, msgs)
}

func newMock(opts ...mockllm.Option) *mockChatModel {
	return &mockChatModel{mockllm.New(opts...)}
}

// --- WebSimulator tests ---

func testPages() []*Page {
	return []*Page{
		{
			Path:    "/",
			Title:   "Home",
			Content: "Welcome to the site",
			Links:   []string{"/login"},
		},
		{
			Path:    "/login",
			Title:   "Login",
			Content: "Please log in",
			Forms: map[string][]FormField{
				"login": {
					{Name: "username", Type: "text", Required: true},
					{Name: "password", Type: "password", Required: true},
				},
			},
			Links: []string{"/"},
		},
	}
}

func TestWebSimulator_Navigate(t *testing.T) {
	ws := NewWebSimulator(WithPages(testPages()...), WithStartPage("/"))
	ctx := context.Background()

	obs, err := ws.Reset(ctx)
	require.NoError(t, err)
	assert.Contains(t, obs.Text, "Home")

	obs, err = ws.Step(ctx, "navigate /login")
	require.NoError(t, err)
	assert.Contains(t, obs.Text, "Login")

	// Navigate to nonexistent page.
	obs, err = ws.Step(ctx, "navigate /missing")
	require.NoError(t, err)
	assert.Contains(t, obs.Text, "not found")
}

func TestWebSimulator_FormSubmission(t *testing.T) {
	ws := NewWebSimulator(WithPages(testPages()...), WithStartPage("/login"))
	ctx := context.Background()

	_, err := ws.Reset(ctx)
	require.NoError(t, err)

	// Fill and submit.
	_, err = ws.Step(ctx, "fill username admin")
	require.NoError(t, err)
	_, err = ws.Step(ctx, "fill password secret123")
	require.NoError(t, err)

	obs, err := ws.Step(ctx, "submit login")
	require.NoError(t, err)
	assert.NotNil(t, obs)

	subs := ws.Submissions()
	require.Len(t, subs, 1)
	assert.Equal(t, "admin", subs[0]["username"])
	assert.Equal(t, "secret123", subs[0]["password"])
}

func TestWebSimulator_RequiredFieldValidation(t *testing.T) {
	ws := NewWebSimulator(WithPages(testPages()...), WithStartPage("/login"))
	ctx := context.Background()
	_, err := ws.Reset(ctx)
	require.NoError(t, err)

	// Submit without filling required fields.
	obs, err := ws.Step(ctx, "submit login")
	require.NoError(t, err)
	assert.Contains(t, obs.Text, "Required field missing")
}

func TestWebSimulator_InvalidActions(t *testing.T) {
	ws := NewWebSimulator(WithPages(testPages()...), WithStartPage("/"))
	ctx := context.Background()
	_, _ = ws.Reset(ctx)

	tests := []struct {
		name    string
		action  string
		wantErr bool
	}{
		{"empty action", "", true},
		{"unknown action", "click button", true},
		{"navigate no path", "navigate", true},
		{"fill no value", "fill", true},
		{"submit no form", "submit", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ws.Step(ctx, tt.action)
			if tt.wantErr {
				require.Error(t, err)
			}
		})
	}
}

func TestWebSimulator_Close(t *testing.T) {
	ws := NewWebSimulator(WithPages(testPages()...), WithStartPage("/"))
	ctx := context.Background()
	_, _ = ws.Reset(ctx)

	require.NoError(t, ws.Close())

	_, err := ws.Step(ctx, "navigate /login")
	require.Error(t, err)
}

func TestWebSimulator_Observe(t *testing.T) {
	ws := NewWebSimulator(WithPages(testPages()...), WithStartPage("/"))
	ctx := context.Background()
	_, _ = ws.Reset(ctx)

	obs, err := ws.Observe(ctx)
	require.NoError(t, err)
	assert.Contains(t, obs.Text, "Home")
}

// --- SimulatedUser tests ---

func TestSimulatedUser_Respond(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("I'd like to book a flight")))
	user, err := NewSimulatedUser(
		WithUserModel(model),
		WithPersona("A business traveler"),
		WithGoal("Book a flight to NYC"),
	)
	require.NoError(t, err)

	resp, err := user.Respond(context.Background(), "How can I help you?")
	require.NoError(t, err)
	assert.Equal(t, "I'd like to book a flight", resp.Message)
	assert.False(t, resp.GoalComplete)
	assert.False(t, resp.GoalFailed)
}

func TestSimulatedUser_GoalComplete(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("[GOAL_COMPLETE]")))
	user, err := NewSimulatedUser(
		WithUserModel(model),
		WithGoal("Test goal"),
	)
	require.NoError(t, err)

	resp, err := user.Respond(context.Background(), "Done!")
	require.NoError(t, err)
	assert.True(t, resp.GoalComplete)
}

func TestSimulatedUser_GoalFailed(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("[GOAL_FAILED]")))
	user, err := NewSimulatedUser(
		WithUserModel(model),
		WithGoal("Impossible goal"),
	)
	require.NoError(t, err)

	resp, err := user.Respond(context.Background(), "I can't do that")
	require.NoError(t, err)
	assert.True(t, resp.GoalFailed)
}

func TestSimulatedUser_Validation(t *testing.T) {
	t.Run("no model", func(t *testing.T) {
		_, err := NewSimulatedUser(WithGoal("test"))
		require.Error(t, err)
		var coreErr *core.Error
		require.ErrorAs(t, err, &coreErr)
		assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
	})

	t.Run("no goal", func(t *testing.T) {
		_, err := NewSimulatedUser(WithUserModel(newMock()))
		require.Error(t, err)
	})
}

func TestSimulatedUser_Reset(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("hello")))
	user, err := NewSimulatedUser(
		WithUserModel(model),
		WithGoal("test"),
	)
	require.NoError(t, err)

	_, _ = user.Respond(context.Background(), "hi")
	assert.Len(t, user.history, 2)

	user.Reset()
	assert.Empty(t, user.history)
}

// --- SimRunner tests ---

func TestSimRunner_RunEpisode(t *testing.T) {
	// User completes goal on second turn.
	callCount := 0
	mock := mockllm.New()
	model := &mockChatModel{mock}

	agent := func(_ context.Context, msg string) (string, error) {
		if msg == "" {
			return "Hello! How can I help?", nil
		}
		return "Your flight is booked!", nil
	}

	// First call: normal response, second call: goal complete.
	mock.SetResponse(schema.NewAIMessage("Book a flight please"))

	user, err := NewSimulatedUser(
		WithUserModel(model),
		WithGoal("Book a flight"),
	)
	require.NoError(t, err)

	// Override Respond to control flow: first turn normal, second complete.
	runner, err := NewSimRunner(
		WithAgent(agent),
		WithSimUser(user),
		WithMaxTurns(5),
		WithOnTurn(func(idx int, _, _ string) { callCount++ }),
	)
	require.NoError(t, err)

	// Swap mock response mid-test to simulate goal completion.
	go func() {
		// After first generate call, switch to GOAL_COMPLETE.
		for {
			if mock.GenerateCalls() >= 1 {
				mock.SetResponse(schema.NewAIMessage("[GOAL_COMPLETE]"))
				return
			}
		}
	}()

	result, err := runner.RunEpisode(context.Background())
	require.NoError(t, err)
	assert.True(t, result.TurnCount >= 1)
	assert.True(t, result.Duration > 0)
}

func TestSimRunner_Validation(t *testing.T) {
	model := newMock(mockllm.WithResponse(schema.NewAIMessage("hi")))
	user, _ := NewSimulatedUser(WithUserModel(model), WithGoal("test"))

	t.Run("no agent", func(t *testing.T) {
		_, err := NewSimRunner(WithSimUser(user))
		require.Error(t, err)
	})

	t.Run("no user", func(t *testing.T) {
		_, err := NewSimRunner(WithAgent(func(_ context.Context, _ string) (string, error) { return "", nil }))
		require.Error(t, err)
	})
}

func TestSimRunner_Run(t *testing.T) {
	mock := mockllm.New(mockllm.WithResponse(schema.NewAIMessage("[GOAL_COMPLETE]")))
	model := &mockChatModel{mock}
	user, err := NewSimulatedUser(WithUserModel(model), WithGoal("test"))
	require.NoError(t, err)

	agent := func(_ context.Context, _ string) (string, error) {
		return "Done!", nil
	}

	runner, err := NewSimRunner(WithAgent(agent), WithSimUser(user), WithMaxTurns(3))
	require.NoError(t, err)

	report, err := runner.Run(context.Background(), 3)
	require.NoError(t, err)
	assert.Len(t, report.Episodes, 3)
	assert.Equal(t, 1.0, report.SuccessRate)
	assert.True(t, report.Duration > 0)
}

func TestSimRunner_RunInvalidEpisodes(t *testing.T) {
	mock := mockllm.New(mockllm.WithResponse(schema.NewAIMessage("[GOAL_COMPLETE]")))
	model := &mockChatModel{mock}
	user, _ := NewSimulatedUser(WithUserModel(model), WithGoal("test"))
	runner, _ := NewSimRunner(
		WithAgent(func(_ context.Context, _ string) (string, error) { return "", nil }),
		WithSimUser(user),
	)

	_, err := runner.Run(context.Background(), 0)
	require.Error(t, err)
}
