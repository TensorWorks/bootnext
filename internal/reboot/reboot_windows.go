package reboot

import "github.com/tensorworks/bootnext/internal/process"

// Attempts to reboot the system
func Reboot() error {
	_, err := process.CaptureOutput([]string{"shutdown", "/r", "/t", "0"})
	return err
}
