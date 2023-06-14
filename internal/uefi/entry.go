package uefi

// Represents an individual UEFI boot entry
type BootEntry struct {

	// The system-specific identifier for the boot entry
	// (Under Linux this is a hexadecimal number, under Windows it is a GUID)
	ID string

	// The human-readable description for the boot entry
	Description string
}
