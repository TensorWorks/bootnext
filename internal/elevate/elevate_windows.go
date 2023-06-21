package elevate

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32        = windows.NewLazyDLL("kernel32.dll")
	shell32         = windows.NewLazyDLL("shell32.dll")
	attachConsole   = kernel32.NewProc("AttachConsole")
	freeConsole     = kernel32.NewProc("FreeConsole")
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

// Constant from: <https://www.pinvoke.net/default.aspx/kernel32/AttachConsole.html>
const ATTACH_PARENT_PROCESS uint32 = 0x0ffffffff

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

// Detect when we are an elevated child process launched by `RunElevated()` and attach to the console of our
// non-elevated parent process
//
// Note: this workaround is necessary because the elevated executable will always open in a new console window,
// which will then close almost immediately, before the user has a chance to read any output.
//
// Although methods do exist for creating elevated processes that inherit the standard handles from a non-elevated
// parent process (<https://github.com/cubiclesoft/createprocess-windows>), these workflows are excessively complex
// and require the use of an additional executable that would need to be bundled with bootnext.
//
// Instead, we programmatically detect when we're the elevated child process and attach to the parent process console.
// This workaround is inspired by the implementation of the `sudo` command in Luke Sampson's "psutils" project:
// <https://github.com/lukesampson/psutils/blob/8af01127a949c64ea50b657989a4cd7744d4fffd/sudo.ps1#L27-L28>
func init() {

	// Don't bother checking our parent process details if we're running as a non-elevated process
	if !IsElevated() {
		return
	}

	// Retrieve the PID for the current process
	currentPID := windows.GetCurrentProcessId()

	// Retrieve the path to the executable for the current process
	executable, err := os.Executable()
	if err != nil {
		return
	}

	// Create a snapshot of the processes currently running on the system
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}

	// Ensure the snapshot handle is closed when this function completes
	defer windows.CloseHandle(snapshot)

	// Create a `PROCESSENTRY32` struct and populate its size field
	var processEntry windows.ProcessEntry32
	processEntry.Size = uint32(unsafe.Sizeof(processEntry))

	// Iterate over the processes in the snapshot until we find the details for the current process
	err = windows.Process32First(snapshot, &processEntry)
	for err == nil && processEntry.ProcessID != currentPID {
		err = windows.Process32Next(snapshot, &processEntry)
	}

	// Verify that we found the details for the current process
	if err != nil {
		return
	}

	// Retrieve the PID of our parent process and attempt to open a process handle
	// Attempt to open a handle to our parent process using its PID
	parentProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, true, processEntry.ParentProcessID)
	if err != nil {
		return
	}

	// Ensure the parent process handle is closed when this function completes
	defer windows.CloseHandle(parentProcess)

	// Retrieve the module for the parent process's executable
	var parentModule windows.Handle
	var bytesNeeded uint32
	if err := windows.EnumProcessModules(parentProcess, &parentModule, uint32(unsafe.Sizeof(parentModule)), &bytesNeeded); err != nil {
		return
	}

	// Retrieve the path to the executable for the parent process
	const bufSize = 4096
	var buffer [bufSize]uint16
	if err := windows.GetModuleFileNameEx(parentProcess, parentModule, &buffer[0], bufSize); err != nil {
		return
	}

	// If the executable paths match for the current process and the parent process then we're an elevated child process created by `RunElevated()`
	if windows.UTF16ToString(buffer[:]) == executable {

		// Detach from the console that was created for the elevated child process
		freeConsole.Call()

		// Attach to the console of the non-elevated parent process
		attachConsole.Call(uintptr(ATTACH_PARENT_PROCESS))

		// Update the standard handles in the `syscall` package
		// (See: <https://github.com/golang/go/blob/go1.20.5/src/syscall/syscall_windows.go#L493-L495>)
		syscall.Stdin, _ = syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
		syscall.Stdout, _ = syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
		syscall.Stderr, _ = syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)

		// Update the corresponding file objects in the `os` package
		// (See: <https://github.com/golang/go/blob/go1.20.5/src/os/file.go#L65-L67>)
		os.Stdin = os.NewFile(uintptr(syscall.Stdin), "/dev/stdin")
		os.Stdout = os.NewFile(uintptr(syscall.Stdout), "/dev/stdout")
		os.Stderr = os.NewFile(uintptr(syscall.Stderr), "/dev/stderr")
	}
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

	// Escape each of the arguments that were passed to the current process
	escapedArgs := []string{}
	for _, arg := range os.Args[1:] {

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
		fMask:                _SEE_MASK_NOCLOSEPROCESS,
		hwnd:                 0,
		lpVerb:               uintptr(unsafe.Pointer(verb)),
		lpFile:               uintptr(unsafe.Pointer(file)),
		lpParameters:         uintptr(unsafe.Pointer(args)),
		lpDirectory:          0,
		nShow:                windows.SW_HIDE,
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
