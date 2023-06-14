package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Executes the specified command and captures its output, providing pretty error messages for non-zero exit codes
func CaptureOutput(command []string) (string, error) {

	// Run the command and retrieve the combined stdout and stderr
	cmd := exec.Command(command[0], command[1:]...)
	output, err := cmd.CombinedOutput()

	// If an error occurred, determine whether it was a non-zero exit code or a failure to run the command
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return "", fmt.Errorf(
				"command %v failed with exit code %v and output:\n%s",
				command,
				exitError.ProcessState.ExitCode(),
				string(output),
			)
		} else {
			return "", fmt.Errorf("failed to run command %v: %v", command, err)
		}
	}

	// Treat the output as a UTF-8 string
	return string(output), nil
}

// Executes the specified command, ensuring it inherits the standard streams from the current process
func RunWithInheritedHandles(command []string) (int, error) {

	// Retrieve the path to the executable
	executable, err := exec.LookPath(command[0])
	if err != nil {
		return -1, err
	}

	// Attempt to run the executable, ensuring it inherits the standard streams from the current process
	process, err := os.StartProcess(executable, command, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})

	// Verify that the child process started successfully
	if err != nil {
		return -1, err
	}

	// Wait for the child process to complete
	status, err := process.Wait()
	if err != nil {
		return -1, err
	}

	// Return the exit code from the child process
	return status.ExitCode(), nil
}

// Exit the current process with the specified exit code, optionally pausing beforehand
func ExitWithPause(exitCode int, pause bool) {

	// Pause if requested
	if pause {
		PauseForInput()
	}

	// Exit with the specified exit code
	os.Exit(exitCode)
}
