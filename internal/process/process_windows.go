package process

// Pauses for user input, using the command prompt's built-in `pause` command
func PauseForInput() {
	RunWithInheritedHandles([]string{"cmd.exe", "/C", "pause"})
}
