package learning

// Hooks provides lifecycle callbacks for tool learning operations.
// All fields are optional — nil hooks are skipped.
type Hooks struct {
	// OnToolCreated is called when a new tool is generated and validated.
	// Receives the tool name and the generated code.
	OnToolCreated func(toolName string, code string)

	// OnToolTested is called after a tool has been tested.
	// Receives the tool name and whether all tests passed.
	OnToolTested func(toolName string, allPassed bool)

	// OnVersionActivated is called when a tool version is activated.
	// Receives the tool name and the activated version number.
	OnVersionActivated func(toolName string, version int)
}

// ComposeHooks merges multiple Hooks into a single Hooks struct.
// OnToolCreated hooks run in order. OnToolTested hooks run in order.
// OnVersionActivated hooks run in order.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnToolCreated: func(toolName string, code string) {
			for _, h := range hooks {
				if h.OnToolCreated != nil {
					h.OnToolCreated(toolName, code)
				}
			}
		},
		OnToolTested: func(toolName string, allPassed bool) {
			for _, h := range hooks {
				if h.OnToolTested != nil {
					h.OnToolTested(toolName, allPassed)
				}
			}
		},
		OnVersionActivated: func(toolName string, version int) {
			for _, h := range hooks {
				if h.OnVersionActivated != nil {
					h.OnVersionActivated(toolName, version)
				}
			}
		},
	}
}
