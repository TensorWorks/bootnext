package elevate

import (
	"os"

	"github.com/tensorworks/bootnext/internal/process"
	"golang.org/x/sys/unix"
)

// Determines whether the current process is running with elevated privileges
func IsElevated() bool {
	return unix.Geteuid() == 0
}

// Re-launches the current process with elevated privileges
func RunElevated() (int, error) {

	// Retrieve the path to the executable for the current process
	executable, err := os.Executable()
	if err != nil {
		return -1, err
	}

	// Attempt to re-run the executable using `sudo`, ensuring it inherits the standard streams from the parent
	command := append([]string{"sudo", executable}, os.Args[1:]...)
	return process.RunWithInheritedHandles(command)
}
