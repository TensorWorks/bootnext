package uefi

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/tensorworks/bootnext/internal/process"
)

// Determines whether the operating system has been booted in UEFI mode
func IsUEFIEnabled() (bool, error) {

	// Determine whether `/sys/firmware/efi` exists
	_, err := os.Stat("/sys/firmware/efi")
	notExist := errors.Is(err, os.ErrNotExist)

	// If the query failed then propagate the error
	if err != nil && !notExist {
		return false, err
	} else {
		return !notExist, nil
	}
}

// Returns the list of system tools that we require in order to interact with UEFI NVRAM variables
func RequiredTools() []string {
	return []string{"efibootmgr"}
}

// Lists the UEFI boot entries for the host machine
func ListBootEntries() ([]BootEntry, error) {

	// Run `efibootmgr` with no flags to print the list of boot entries
	output, err := process.CaptureOutput([]string{"efibootmgr"})
	if err != nil {
		return nil, err
	}

	// Compile our regular expression for parsing the output
	regex, err := regexp.Compile(`Boot([0-9]+)\*?\s+(.+)`)
	if err != nil {
		return nil, err
	}

	// Parse the list of entries
	entries := []BootEntry{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if groups := regex.FindStringSubmatch(line); groups != nil {
			entries = append(entries, BootEntry{
				ID:          groups[1],
				Description: groups[2],
			})
		}
	}

	return entries, nil
}

// Sets the value of the BootNext UEFI NVRAM variable
func SetBootNext(entry BootEntry) error {
	_, err := process.CaptureOutput([]string{"efibootmgr", "--bootnext", entry.ID})
	return err
}
