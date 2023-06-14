package uefi

import (
	"regexp"
	"strings"

	"github.com/tensorworks/bootnext/internal/process"
)

// Determines whether the operating system has been booted in UEFI mode
func IsUEFIEnabled() (bool, error) {

	// Use PowerShell to query the system firmware type
	output, err := process.CaptureOutput([]string{
		"powershell.exe",
		"-ExecutionPolicy", "Bypass",
		"-Command", "Write-Host $env:firmware_type",
	})
	if err != nil {
		return false, err
	}

	// Determine whether the reported firmware type is legacy BIOS or UEFI
	return strings.TrimSpace(strings.ToUpper(output)) == "UEFI", nil
}

// Returns the list of system tools that we require in order to interact with UEFI NVRAM variables
func RequiredTools() []string {
	return []string{"bcdedit"}
}

// Lists the UEFI boot entries for the host machine
func ListBootEntries() ([]BootEntry, error) {

	// Run `bcdedit` to print the list of boot entries
	output, err := process.CaptureOutput([]string{"bcdedit", "/enum", "firmware"})
	if err != nil {
		return nil, err
	}

	// Compile our regular expressions for parsing the output
	separatorRegex, err := regexp.Compile(`^-+$`)
	if err != nil {
		return nil, err
	}
	identifierRegex, err := regexp.Compile(`identifier +(.+)`)
	if err != nil {
		return nil, err
	}
	descriptionRegex, err := regexp.Compile(`description +(.+)`)
	if err != nil {
		return nil, err
	}

	// Examine each line of the output and parse the boot entries
	entries := []BootEntry{}
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	for _, line := range lines {

		// Determine whether the line is a separator that marks the start of a new boot entry
		if separatorRegex.FindStringIndex(line) != nil {
			entries = append(entries, BootEntry{})

		} else if len(entries) > 0 {

			// Determine whether the line provides the GUID or the description for the boot entry
			if match := identifierRegex.FindStringSubmatch(line); match != nil {
				entries[len(entries)-1].ID = match[1]

			} else if match := descriptionRegex.FindStringSubmatch(line); match != nil {
				entries[len(entries)-1].Description = match[1]

			}
		}
	}

	// Filter out any malformed boot entries
	filtered := []BootEntry{}
	for _, entry := range entries {
		if entry.ID != "" && entry.Description != "" {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// Sets the value of the BootNext UEFI NVRAM variable
func SetBootNext(entry BootEntry) error {
	_, err := process.CaptureOutput([]string{"bcdedit", "/set", "{fwbootmgr}", "bootsequence", entry.ID})
	return err
}
