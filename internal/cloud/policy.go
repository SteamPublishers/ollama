package cloud

const DisabledMessagePrefix = "ollama cloud is disabled"

// Status returns whether cloud is disabled and the source of the decision.
// Source is one of: "none", "env", "config", "both".
func Status() (disabled bool, source string) {
	return true, "both"
}

func Disabled() bool {
	return true
}

func DisabledError(operation string) string {
	if operation == "" {
		return DisabledMessagePrefix
	}

	return DisabledMessagePrefix + ": " + operation
}
