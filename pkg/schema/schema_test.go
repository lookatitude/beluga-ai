package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageType(t *testing.T) {
	assert.Equal(t, "human", string(RoleHuman))
	assert.Equal(t, "ai", string(RoleAssistant))
	assert.Equal(t, "system", string(RoleSystem))
	assert.Equal(t, "tool", string(RoleTool))
	assert.Equal(t, "function", string(RoleFunction)) // Assuming RoleFunction is an alias or similar to RoleTool
}

func TestBaseMessage(t *testing.T) {
	bm := BaseMessage{Content: "hello"}
	assert.Equal(t, "hello", bm.GetContent())
	// assert.Equal(t, RoleHuman, bm.GetType()) // Default type; BaseMessage does not have GetType()

	bm.Content = "world" // Direct assignment
	assert.Equal(t, "world", bm.GetContent())
}

func TestChatMessage(t *testing.T) {
	cm := ChatMessage{
		BaseMessage: BaseMessage{Content: "test chat"},
		Role:        RoleAssistant,
	}
	assert.Equal(t, "test chat", cm.GetContent())
	assert.Equal(t, RoleAssistant, cm.GetType())
	assert.Equal(t, RoleAssistant, cm.Role)
}

func TestToolMessage(t *testing.T) {	tm := ToolMessage{
		BaseMessage: BaseMessage{Content: "tool output here"},
		ToolCallID:  "tool_123",
	}
	assert.Equal(t, "tool output here", tm.GetContent())
	assert.Equal(t, RoleTool, tm.GetType())
	assert.Equal(t, "tool_123", tm.ToolCallID)
}

func TestFunctionMessage(t *testing.T) {
	// Assuming FunctionMessage is similar to ToolMessage or uses RoleFunction
	fm := FunctionMessage{
		BaseMessage: BaseMessage{Content: "function output here"},
		Name:        "get_weather",
	}
	assert.Equal(t, "function output here", fm.GetContent())
	assert.Equal(t, RoleFunction, fm.GetType())
	assert.Equal(t, "get_weather", fm.Name)
}

func TestNewHumanMessage(t *testing.T) {
	msg := NewHumanMessage("hello human")
	assert.Equal(t, "hello human", msg.GetContent())
	assert.Equal(t, RoleHuman, msg.GetType())
	// Ensure it's a ChatMessage underneath with the correct role
	cm, ok := msg.(*ChatMessage)
	assert.True(t, ok)
	assert.Equal(t, RoleHuman, cm.Role)
}

func TestNewAIMessage(t *testing.T) {
	msg := NewAIMessage("hello ai")
	assert.Equal(t, "hello ai", msg.GetContent())
	assert.Equal(t, RoleAssistant, msg.GetType())
	// Ensure it's an AIMessage underneath
	_, ok := msg.(*AIMessage) // MODIFIED HERE
	assert.True(t, ok)        // MODIFIED HERE
	// Role is implicitly tested by msg.GetType() == RoleAssistant
}

func TestNewSystemMessage(t *testing.T) {
	msg := NewSystemMessage("hello system")
	assert.Equal(t, "hello system", msg.GetContent())
	assert.Equal(t, RoleSystem, msg.GetType())
	cm, ok := msg.(*ChatMessage)
	assert.True(t, ok)
	assert.Equal(t, RoleSystem, cm.Role)
}

func TestNewToolMessage(t *testing.T) {
	msg := NewToolMessage("tool_output", "call_abc")
	assert.Equal(t, "tool_output", msg.GetContent())
	assert.Equal(t, RoleTool, msg.GetType())
	tm, ok := msg.(*ToolMessage)
	assert.True(t, ok)
	assert.Equal(t, "call_abc", tm.ToolCallID)
}

func TestNewChatMessage(t *testing.T) {
	msg := NewChatMessage(RoleAssistant, "generic chat")
	assert.Equal(t, "generic chat", msg.GetContent())
	assert.Equal(t, RoleAssistant, msg.GetType())
	cm, ok := msg.(*ChatMessage)
	assert.True(t, ok)
	assert.Equal(t, RoleAssistant, cm.Role)
}

func TestGeneration(t *testing.T) {
	gen := Generation{
		Text: "generated text",
		Message: &ChatMessage{BaseMessage: BaseMessage{Content: "ai message for generation"}, Role: RoleAssistant},
		GenerationInfo: map[string]interface{}{"finish_reason": "stop"},
	}
	assert.Equal(t, "generated text", gen.Text)
	assert.NotNil(t, gen.Message)
	assert.Equal(t, "ai message for generation", gen.Message.GetContent())
	assert.Equal(t, RoleAssistant, gen.Message.GetType())
	assert.Equal(t, "stop", gen.GenerationInfo["finish_reason"])
}

func TestLLMResponse(t *testing.T) {
	resp := LLMResponse{
		Generations: [][]*Generation{
			{
				{Text: "gen1"},
				{Text: "gen2"},
			},
		},
		LLMOutput: map[string]interface{}{"token_usage": 100},
	}
	assert.Len(t, resp.Generations, 1)
	assert.Len(t, resp.Generations[0], 2)
	assert.Equal(t, "gen1", resp.Generations[0][0].Text)
	assert.Equal(t, 100, resp.LLMOutput["token_usage"])
}

func TestDocument(t *testing.T) {
	doc := Document{
		PageContent: "This is a test document.",
		Metadata:    map[string]string{"source": "test.txt", "page_str": "1"}, // Changed to string value for page
		Score:       0.95,
	}
	assert.Equal(t, "This is a test document.", doc.PageContent)
	assert.Equal(t, "test.txt", doc.Metadata["source"])
	assert.Equal(t, "1", doc.Metadata["page_str"])
	assert.Equal(t, float32(0.95), doc.Score)
}

func TestHistory(t *testing.T) {
	h := NewBaseChatHistory()
	messages, err := h.Messages()
	assert.NoError(t, err)
	assert.Empty(t, messages)
	h.AddUserMessage("Hello AI")
	messages, err = h.Messages()
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, RoleHuman, messages[0].GetType())
	assert.Equal(t, "Hello AI", messages[0].GetContent())

	h.AddAIMessage("Hello Human")
	messages, err = h.Messages()
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, RoleAssistant, messages[1].GetType())
	assert.Equal(t, "Hello Human", messages[1].GetContent())

	h.AddMessage(NewSystemMessage("System init"))
	messages, err = h.Messages()
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
	assert.Equal(t, RoleSystem, messages[2].GetType())
	assert.Equal(t, "System init", messages[2].GetContent())

	h.Clear()
	messages, err = h.Messages()
	assert.NoError(t, err)
	assert.Empty(t, messages)
} // ADDED MISSING BRACE

func TestAgentAction(t *testing.T) {
	action := AgentAction{
		Tool:      "calculator",
		ToolInput: map[string]interface{}{"expression": "2+2"},
		Log:       "User asked to calculate 2+2",
	}
	assert.Equal(t, "calculator", action.Tool)
	assert.Equal(t, "2+2", action.ToolInput.(map[string]interface{})["expression"])
	assert.Equal(t, "User asked to calculate 2+2", action.Log)
}

func TestAgentFinish(t *testing.T) {
	finish := AgentFinish{
		ReturnValues: map[string]interface{}{"result": 4},
		Log:          "Calculation finished.",
	}
	assert.Equal(t, 4, finish.ReturnValues["result"])
	assert.Equal(t, "Calculation finished.", finish.Log)
}

func TestLLMProviderConfig(t *testing.T) {
	config := LLMProviderConfig{
		Name:      "test_config_openai", // Added Name
		Provider:  "openai",             // Added Provider
		ModelName: "gpt-3.5-turbo",
		APIKey:    "sk-xxxx",
		DefaultCallOptions: map[string]interface{}{
			"temperature":       0.7,
			"max_tokens":        256,
			"top_p":             0.9,
			"frequency_penalty": 0.1,
			"presence_penalty":  0.1,
			"stop":              []string{"\nHuman:"},
			"streaming":         true,
		},
		ProviderSpecific: map[string]interface{}{"custom_param": "custom_value"}, // Example of ProviderSpecific
	}
	assert.Equal(t, "test_config_openai", config.Name)
	assert.Equal(t, "openai", config.Provider)
	assert.Equal(t, "gpt-3.5-turbo", config.ModelName)
	assert.Equal(t, "sk-xxxx", config.APIKey)

	// Assertions for DefaultCallOptions with type assertions
	if temp, ok := config.DefaultCallOptions["temperature"].(float64); ok {
		assert.Equal(t, 0.7, temp)
	} else {
		t.Errorf("temperature not found or not a float64 in DefaultCallOptions")
	}

	if maxTokens, ok := config.DefaultCallOptions["max_tokens"].(int); ok {
		assert.Equal(t, 256, maxTokens)
	} else {
		if maxTokensF, okF := config.DefaultCallOptions["max_tokens"].(float64); okF { // Handle if it was unmarshalled as float64
			assert.Equal(t, 256, int(maxTokensF))
		} else {
			t.Errorf("max_tokens not found or not an int/float64 in DefaultCallOptions")
		}
	}

	if topP, ok := config.DefaultCallOptions["top_p"].(float64); ok {
		assert.Equal(t, 0.9, topP)
	} else {
		t.Errorf("top_p not found or not a float64 in DefaultCallOptions")
	}

	if freqPenalty, ok := config.DefaultCallOptions["frequency_penalty"].(float64); ok {
		assert.Equal(t, 0.1, freqPenalty)
	} else {
		t.Errorf("frequency_penalty not found or not a float64 in DefaultCallOptions")
	}

	if presPenalty, ok := config.DefaultCallOptions["presence_penalty"].(float64); ok {
		assert.Equal(t, 0.1, presPenalty)
	} else {
		t.Errorf("presence_penalty not found or not a float64 in DefaultCallOptions")
	}

	if stop, ok := config.DefaultCallOptions["stop"].([]string); ok {
		assert.Contains(t, stop, "\nHuman:")
	} else {
		t.Errorf("stop not found or not a []string in DefaultCallOptions")
	}

	if streaming, ok := config.DefaultCallOptions["streaming"].(bool); ok {
		assert.True(t, streaming)
	} else {
		t.Errorf("streaming not found or not a bool in DefaultCallOptions")
	}
    
    if customParam, ok := config.ProviderSpecific["custom_param"].(string); ok {
        assert.Equal(t, "custom_value", customParam)
    } else {
        t.Errorf("custom_param not found or not a string in ProviderSpecific")
    }
}

func TestAgentScratchPadEntry(t *testing.T) {
	entry := AgentScratchPadEntry{
		Action:      AgentAction{Tool: "search", ToolInput: map[string]interface{}{"query": "AI"}},
		Observation: "AI is artificial intelligence.",
	}
	assert.Equal(t, "search", entry.Action.Tool)
	assert.Equal(t, "AI is artificial intelligence.", entry.Observation)
}

func TestToolCall(t *testing.T) {
	argsMap := map[string]interface{}{"location": "Boston", "unit": "celsius"}
	argsJSON, _ := json.Marshal(argsMap)

	tc := ToolCall{
		ID:       "call_123",
		Type:     "function",
		Function: FunctionCall{Name: "get_current_weather", Arguments: string(argsJSON)},
	}

	assert.Equal(t, "call_123", tc.ID)
	assert.Equal(t, "function", tc.Type)
	assert.Equal(t, "get_current_weather", tc.Function.Name)

	var parsedArgs map[string]interface{}
	err := json.Unmarshal([]byte(tc.Function.Arguments), &parsedArgs)
	assert.NoError(t, err)
	assert.Equal(t, "Boston", parsedArgs["location"])
	assert.Equal(t, "celsius", parsedArgs["unit"])
}

func TestAIMessageWithToolCalls(t *testing.T) {
	argsMap := map[string]interface{}{"location": "London"}
	argsJSON, _ := json.Marshal(argsMap)

	aim := AIMessage{
		BaseMessage: BaseMessage{Content: "I can get the weather for you."},
		ToolCalls: []ToolCall{
			{
				ID:       "call_abc",
				Type:     "function",
				Function: FunctionCall{Name: "fetch_weather", Arguments: string(argsJSON)},
			},
		},
	}

	assert.Equal(t, "I can get the weather for you.", aim.GetContent())
	assert.Equal(t, RoleAssistant, aim.GetType())
	assert.Len(t, aim.ToolCalls, 1)
	assert.Equal(t, "call_abc", aim.ToolCalls[0].ID)
	assert.Equal(t, "fetch_weather", aim.ToolCalls[0].Function.Name)

	var parsedArgs map[string]interface{}
	err := json.Unmarshal([]byte(aim.ToolCalls[0].Function.Arguments), &parsedArgs)
	assert.NoError(t, err)
	assert.Equal(t, "London", parsedArgs["location"])
}



