package screens

// NavigateMsg is sent when navigating to a different screen
type NavigateMsg struct {
	Target string // dashboard, import, sync, diff, dump, list, ignore, config, history, profile, doctor
	Data   any    // Optional data to pass to the target screen
}

// StatusMsg is sent to display a status message
type StatusMsg struct {
	Message string
	Type    string // info, success, error, warning
}

// SetupCompleteMsg is sent when initial setup is complete
type SetupCompleteMsg struct{}

// RefreshMsg is sent to refresh the current screen's data
type RefreshMsg struct{}

// Navigate creates a NavigateMsg to the target screen
func Navigate(target string) NavigateMsg {
	return NavigateMsg{Target: target}
}

// NavigateWithData creates a NavigateMsg with data
func NavigateWithData(target string, data any) NavigateMsg {
	return NavigateMsg{Target: target, Data: data}
}

// Status creates a StatusMsg
func Status(message, msgType string) StatusMsg {
	return StatusMsg{Message: message, Type: msgType}
}

// StatusSuccess creates a success status message
func StatusSuccess(message string) StatusMsg {
	return StatusMsg{Message: message, Type: "success"}
}

// StatusError creates an error status message
func StatusError(message string) StatusMsg {
	return StatusMsg{Message: message, Type: "error"}
}

// StatusInfo creates an info status message
func StatusInfo(message string) StatusMsg {
	return StatusMsg{Message: message, Type: "info"}
}

// StatusWarning creates a warning status message
func StatusWarning(message string) StatusMsg {
	return StatusMsg{Message: message, Type: "warning"}
}

// ShowIgnoredMsg is sent to toggle showing/hiding ignored items
type ShowIgnoredMsg struct {
	Show bool
}
