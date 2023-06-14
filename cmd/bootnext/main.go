package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tensorworks/bootnext/internal/constants"
	"github.com/tensorworks/bootnext/internal/elevate"
	"github.com/tensorworks/bootnext/internal/process"
	"github.com/tensorworks/bootnext/internal/reboot"
	"github.com/tensorworks/bootnext/internal/uefi"
)

func run(pattern string, dryRun bool, listOnly bool, noElevate bool, noReboot bool) error {

	// Verify that the operating system has been booted in UEFI mode
	enabled, err := uefi.IsUEFIEnabled()
	if err != nil {
		return fmt.Errorf("failed to query system UEFI status: %v", err)
	} else if !enabled {
		return fmt.Errorf("unsupported system configuration: the operating system has not been booted in UEFI mode")
	}

	// Verify that all of the system tools we require for interacting with UEFI NVRAM variables are available
	requiredTools := uefi.RequiredTools()
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("a required application was not found in the system PATH: %v", tool)
		}
	}

	// Determine whether we require elevated privileges
	// (We need them for writing to NVRAM variables under Linux, and for both reading and writing under Windows)
	requireElevation := (!dryRun && !listOnly) || runtime.GOOS == "windows"

	// Determine whether the process is running with insufficient privileges
	if requireElevation && !elevate.IsElevated() {

		// Determine whether we should automatically request elevated privileges
		if !noElevate {

			// Re-run the process with elevated privileges and propagate the exit code
			exitCode, err := elevate.RunElevated()
			if err != nil {
				return fmt.Errorf("failed to re-launch the process with elevated privileges: %v", err)
			} else {
				os.Exit(exitCode)
			}

		} else {
			fmt.Print("Warning: running without elevated privileges, access to UEFI NVRAM variables may be denied.\n\n")
		}
	}

	// Retrieve the list of UEFI boot entries
	entries, err := uefi.ListBootEntries()
	if err != nil {
		return fmt.Errorf("failed to list UEFI boot entries: %v", err)
	}

	// Print the list of boot entries
	fmt.Println("Detected the following UEFI boot entries:")
	for _, entry := range entries {
		fmt.Print("- ID: \"", entry.ID, "\", Description: \"", entry.Description, "\"\n")
	}

	// If we are just listing the boot entries then stop here
	if listOnly {
		return nil
	}

	// Compile the regular expression pattern supplied by the user, enabling case-insensitive matching
	regex, err := regexp.Compile(fmt.Sprintf("(?i)%s", pattern))
	if err != nil {
		return fmt.Errorf("failed to compile regular expression \"%s\": %v", pattern, err)
	}

	// Identify the first boot entry that matches the pattern
	fmt.Printf("\nMatching boot entries against regular expression \"%s\"\n", pattern)
	for _, entry := range entries {
		if regex.MatchString(entry.Description) {

			// Print the matching boot entry
			fmt.Printf("Found matching boot entry: \"%s\"\n", entry.Description)

			// Don't modify the BootNext variable or reboot if we are performing a dry run
			if !dryRun {

				// Set the value of the BootNext variable to the entry's identifier
				fmt.Println("Setting the BootNext variable...")
				if err := uefi.SetBootNext(entry); err != nil {
					return fmt.Errorf("failed to set BootNext variable value: %v", err)
				}

				// Determine whether we are triggering a reboot
				if !noReboot {
					fmt.Println("Rebooting now...")
					if err := reboot.Reboot(); err != nil {
						return fmt.Errorf("failed to reboot: %v", err)
					}
				}
			}

			return nil
		}
	}

	// If we reach this point then none of the boot entries matched the pattern
	return fmt.Errorf("could not find any UEFI boot entries matching the pattern \"%s\"", pattern)
}

func main() {

	// Define our Cobra command
	command := &cobra.Command{

		Long: strings.Join([]string{
			fmt.Sprintf("bootnext v%s", constants.VERSION),
			"Copyright (c) 2023, TensorWorks Pty Ltd",
			"",
			"Sets the UEFI \"BootNext\" variable and triggers a reboot into the target operating system.",
			"This facilitates quickly switching to another OS without modifying the default boot order.",
		}, "\n"),

		Use: "bootnext pattern",

		SilenceUsage: true,

		Example: strings.Join([]string{
			"  bootnext windows   Selects the Windows Boot Manager and boots into it",
			"  bootnext ubuntu    Selects the GRUB bootloader installed by Ubuntu Linux and boots into it",
			"  bootnext USB       Selects the first available bootable USB device and boots into it",
		}, "\n"),
	}

	// Inject the usage information for our command's positional arguments
	patternUsage := strings.Join([]string{
		"  pattern            A regular expression that will be used to select the target boot entry",
		"                     (case insensitive)",
	}, "\n")
	template := command.UsageTemplate()
	template = strings.Replace(template, "\nFlags:\n", fmt.Sprintf("\nPositional Arguments:\n%s\n\nFlags:\n", patternUsage), 1)
	command.SetUsageTemplate(template)

	// Define the command-line flags for our command
	dryRun := command.Flags().Bool("dry-run", false, "Describe the actions that would be performed but do not make any changes to the system")
	listOnly := command.Flags().Bool("list", false, "Print the list of UEFI boot entries but do not set the BootNext variable")
	noElevate := command.Flags().Bool("no-elevate", false, "Do not automatically prompt for elevated privileges when required")
	noReboot := command.Flags().Bool("no-reboot", false, "Do not automatically reboot after setting the BootNext variable")
	pause := command.Flags().Bool("pause", false, "Pause for input when the application is finished running")

	// Wire up the validation logic for our command-line flags and positional arguments
	command.RunE = func(cmd *cobra.Command, args []string) error {

		// If no flags or arguments were specified then print the usage message
		if len(os.Args) < 2 {
			cmd.Help()
			return nil
		}

		// Verify that a pattern was provided if `--list` was not specified
		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		} else if !*listOnly {
			return fmt.Errorf("a pattern must be specified for selecting the target UEFI boot entry")
		}

		// Process the provided input values and propagate any errors
		return run(pattern, *dryRun, *listOnly, *noElevate, *noReboot)
	}

	// Execute the command
	err := command.Execute()
	if err != nil {
		process.ExitWithPause(1, *pause)
	}

	process.ExitWithPause(0, *pause)
}
