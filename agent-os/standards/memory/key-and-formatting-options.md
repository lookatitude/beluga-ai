# Key and Formatting Options

Standardize **preferred key names** and expose **GetInputOutputKeys** and **GetBufferString** at package level so internal and provider code can reuse them and adjust for provider needs. HumanPrefix, AIPrefix, and ReturnMessages are **optional** with documented defaults.

- **Keys:** MemoryKey, InputKey, OutputKey. Preferred input names: input, query, question, human_input, user_input. Preferred output names: output, result, answer, ai_output, response. GetInputOutputKeys(inputs, outputs) at package level — accessible to internal/ and providers; providers may use different key conventions.
- **Formatting:** HumanPrefix, AIPrefix, ReturnMessages (messages vs string). Optional; documented defaults (e.g. Human, AI, false). GetBufferString(messages, humanPrefix, aiPrefix) at package level for reuse by internal/ and providers.
- **Other options:** WindowSize, MaxTokenLimit, TopK for window/summary/vector memory. Optional with defaults.
