package process

// Pauses for user input, using Bash's built-in `read` command
func PauseForInput() {
	RunWithInheritedHandles([]string{"bash", "-c", `read -n 1 -rsp "Press any key to continue..."; echo ""`})
}
