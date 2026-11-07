package sway

import "fmt"

// ErrInvalidSwayInput is returned when a user-supplied value cannot be safely
// embedded in a sway IPC command.
var ErrInvalidSwayInput = fmt.Errorf("invalid sway command input")

// validSwayWord reports whether s is safe to embed in a sway IPC command,
// either unquoted (workspace names, output names) or inside double quotes
// (app_id criteria such as [app_id="..."]).
//
// Only letters, digits, '.', '-' and '_' are permitted, and the string must
// be non-empty. This blocks command injection: newlines separate sway IPC
// commands, and '"' would break out of a quoted criterion. Anything outside
// this whitelist is rejected rather than escaped.
func validSwayWord(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch {
		case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		case b == '.', b == '-', b == '_':
		default:
			return false
		}
	}
	return true
}

// ValidateWorkspaceName validates a workspace name for use in a sway command.
func ValidateWorkspaceName(name string) error {
	if !validSwayWord(name) {
		return fmt.Errorf("%w: workspace name %q contains unsafe characters", ErrInvalidSwayInput, name)
	}
	return nil
}

// ValidateAppID validates an app_id for use in a sway command criterion.
func ValidateAppID(appID string) error {
	if !validSwayWord(appID) {
		return fmt.Errorf("%w: app_id %q contains unsafe characters", ErrInvalidSwayInput, appID)
	}
	return nil
}

// ValidateOutputName validates a sway output name for use in a command.
func ValidateOutputName(output string) error {
	if !validSwayWord(output) {
		return fmt.Errorf("%w: output name %q contains unsafe characters", ErrInvalidSwayInput, output)
	}
	return nil
}
