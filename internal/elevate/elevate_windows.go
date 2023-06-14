package elevate

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32         = windows.NewLazyDLL("shell32.dll")
	shellExecuteExW = shell32.NewProc("ShellExecuteExW")
)

// Constants from: <https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow#members>
const (

	// Bitmask flags
	_SEE_MASK_NOCLOSEPROCESS uint32 = 0x00000040
	_SEE_MASK_NO_CONSOLE     uint32 = 0x00008000

	// Error codes
	_SE_ERR_FNF             uint32 = 2
	_SE_ERR_PNF             uint32 = 3
	_SE_ERR_ACCESSDENIED    uint32 = 5
	_SE_ERR_OOM             uint32 = 8
	_SE_ERR_SHARE           uint32 = 26
	_SE_ERR_ASSOCINCOMPLETE uint32 = 27
	_SE_ERR_DDETIMEOUT      uint32 = 28
	_SE_ERR_DDEFAIL         uint32 = 29
	_SE_ERR_DDEBUSY         uint32 = 30
	_SE_ERR_NOASSOC         uint32 = 31
	_SE_ERR_DLLNOTFOUND     uint32 = 32
)

// SHELLEXECUTEINFOW structure, from: <https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow>
type _SHELLEXECUTEINFOW struct {
	cbSize               uint32
	fMask                uint32
	hwnd                 uintptr
	lpVerb               uintptr
	lpFile               uintptr
	lpParameters         uintptr
	lpDirectory          uintptr
	nShow                int32
	hInstApp             uintptr
	lpIDList             uintptr
	lpClass              uintptr
	hkeyClass            uintptr
	dwHotKey             uint32
	union_hIcon_hMonitor uintptr
	hProcess             windows.Handle
}

// Determines whether the current process is running with elevated privileges
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// Re-launches the current process with elevated privileges
func RunElevated() (int, error) {

	// Retrieve the path to the executable for the current process
	executable, err := os.Executable()
	if err != nil {
		return -1, err
	}

	// Append `--pause` to our list of arguments
	//
	// Note: this workaround is necessary because the elevated executable will always open in a new
	// console window, which will then close almost immediately, before the user has a chance to read
	// any output.
	//
	// Although methods do exist for creating elevated processes that inherit the standard handles
	// from a non-elevated parent process (<https://github.com/cubiclesoft/createprocess-windows>),
	// these workflows are excessively complex and require the use of an additional executable that
	// would need to be bundled with bootnext.
	//
	// Instead, we use this much simpler workaround.
	//
	cmdArgs := append(os.Args, "--pause")

	// Escape each of the arguments
	escapedArgs := []string{}
	for _, arg := range cmdArgs {

		// Iterate over each character in the argument
		var builder strings.Builder
		for index, char := range arg {

			// If the last character in the argument is a backslash then ignore it
			// (This ensures our trailing double quote isn't inadvertently escaped)
			if char == '\\' && index == len(arg)-1 {
				continue
			}

			// Prepend backslashes to any double quotes or existing backslashes
			if char == '"' || char == '\\' {
				builder.WriteRune('\\')
			}

			// Add the character to our escaped string
			builder.WriteRune(char)
		}

		// Wrap the escaped argument in double quotes, regardless of whether it contains any spaces
		escapedArgs = append(escapedArgs, fmt.Sprintf("\"%s\"", builder.String()))
	}

	// Convert each of the text parameters to UTF-16 strings
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return -1, err
	}
	file, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return -1, err
	}
	args, err := windows.UTF16PtrFromString(strings.Join(escapedArgs, " "))
	if err != nil {
		return -1, err
	}

	// Prepare our input struct for `ShellExecuteExW()`
	execInfo := &_SHELLEXECUTEINFOW{
		cbSize:               uint32(unsafe.Sizeof(_SHELLEXECUTEINFOW{})),
		fMask:                _SEE_MASK_NOCLOSEPROCESS | _SEE_MASK_NO_CONSOLE,
		hwnd:                 0,
		lpVerb:               uintptr(unsafe.Pointer(verb)),
		lpFile:               uintptr(unsafe.Pointer(file)),
		lpParameters:         uintptr(unsafe.Pointer(args)),
		lpDirectory:          0,
		nShow:                windows.SW_SHOW,
		hInstApp:             0, // Will be set by `ShellExecuteExW()`
		lpIDList:             0,
		lpClass:              0,
		hkeyClass:            0,
		dwHotKey:             0,
		union_hIcon_hMonitor: 0,
		hProcess:             0, // Will be set by `ShellExecuteExW()`
	}

	// Attempt to launch the process with elevated privileges and verify that the child process started successfully
	result, _, lastError := shellExecuteExW.Call(uintptr(unsafe.Pointer(execInfo)))
	if uint32(result) == 0 {
		errorMessages := map[uint32]string{
			_SE_ERR_FNF:             "File not found.",
			_SE_ERR_PNF:             "Path not found.",
			_SE_ERR_ACCESSDENIED:    "Access denied.",
			_SE_ERR_OOM:             "Out of memory.",
			_SE_ERR_SHARE:           "Cannot share an open file.",
			_SE_ERR_ASSOCINCOMPLETE: "File association information not complete.",
			_SE_ERR_DDETIMEOUT:      "DDE operation timed out.",
			_SE_ERR_DDEFAIL:         "DDE operation failed.",
			_SE_ERR_DDEBUSY:         "DDE operation is busy.",
			_SE_ERR_NOASSOC:         "File association not available.",
			_SE_ERR_DLLNOTFOUND:     "Dynamic-link library not found.",
		}
		if message, found := errorMessages[uint32(execInfo.hInstApp)]; found {
			return -1, fmt.Errorf(message)
		} else {
			return -1, lastError
		}
	}

	// Ensure we close the handle to the child process when we're done
	defer windows.CloseHandle(execInfo.hProcess)

	// Wait for the child process to complete
	if _, err := windows.WaitForSingleObject(execInfo.hProcess, windows.INFINITE); err != nil {
		return -1, err
	}

	// Retrieve the exit code from the child process
	var exitCode uint32
	if err := windows.GetExitCodeProcess(execInfo.hProcess, &exitCode); err != nil {
		return -1, err
	}

	return int(exitCode), nil
}
