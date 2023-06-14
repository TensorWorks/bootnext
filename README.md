# bootnext: sets the UEFI `BootNext` variable and reboots

`bootnext` is a simple command-line tool for Linux and Windows that sets the UEFI [BootNext](https://uefi.org/specs/UEFI/2.10/03_Boot_Manager.html#globally-defined-variables) NVRAM variable and triggers a reboot into the target operating system. This facilitates quickly switching to another OS without modifying the default boot order. **Note that this tool only supports operating systems installed in UEFI mode rather than legacy BIOS mode, so be sure to read the [*Troubleshooting*](#troubleshooting) section for details on how to check which mode your OS is running under.**

Booting into another OS just requires specifying a regular expression that will be matched against the human-readable description of the target UEFI boot entry:

```bash
# Selects the Windows Boot Manager and boots into it
bootnext windows

# Selects the GRUB bootloader installed by Ubuntu Linux and boots into it
bootnext ubuntu

# Selects the first available bootable USB device and boots into it
# (Note that the regular expression is case insensitive, so `bootnext usb` works too)
bootnext USB
```


## Contents

- [Background and intended use](#background-and-intended-use)
- [Installation](#installation)
    - [Prerequisites](#prerequisites)
    - [System-wide installation](#system-wide-installation)
    - [Portable installation](#portable-installation)
- [Usage](#usage)
    - [Listing boot entries](#listing-boot-entries)
    - [Booting into a target OS](#booting-into-a-target-os)
    - [Performing a dry run](#performing-a-dry-run)
    - [Automatic privilege elevation](#automatic-privilege-elevation)
    - [Setting the `BootNext` variable without rebooting](#setting-the-bootnext-variable-without-rebooting)
- [Troubleshooting](#troubleshooting)
    - [Booting into Linux just loads its bootloader (e.g. GRUB) and boots the default menu entry](#booting-into-linux-just-loads-its-bootloader-eg-grub-and-boots-the-default-menu-entry)
    - [Determining whether an operating system is running under UEFI mode or legacy BIOS mode](#determining-whether-an-operating-system-is-running-under-uefi-mode-or-legacy-bios-mode)
    - [Running `bootnext` prints the error `unsupported system configuration: the operating system has not been booted in UEFI mode`](#running-bootnext-prints-the-error-unsupported-system-configuration-the-operating-system-has-not-been-booted-in-uefi-mode)
- [Building from source](#building-from-source)
- [Legal](#legal)


## Background and intended use

This tool is primarily designed for scenarios where a user is remotely accessing a bare metal machine that boots multiple operating systems, and they need to switch between OSes. Unless the remote machine supports [IPMI](https://en.wikipedia.org/wiki/Intelligent_Platform_Management_Interface) or is being accessed via a mechanism that supports BIOS/UEFI interaction such as a [KVM over IP switch](https://pikvm.org/), a lack of physical access typically precludes the ability to interact with any boot menus. The simplest fallback option is to manipulate the machine's UEFI NVRAM variables through OS-specific software mechanisms, and this is precisely what `bootnext` does:

- Under Linux, the [efibootmgr](https://github.com/rhboot/efibootmgr) command is used to manipulate UEFI variables

- Under Windows, the [bcdedit](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/bcdedit) command is used to manipulate UEFI variables

In addition to its intended use in remote access scenarios, `bootnext` can also simply act as a convenient cross-platform tool for quickly booting into a given target OS without needing to interact with the BIOS/UEFI. It is particularly handy when booting into installers and live environments stored on bootable USB devices, and can even be stored on the USB device itself when using multiboot systems such as [Ventoy](https://www.ventoy.net/). **However, take note of the limitations discussed in the [*Listing boot entries*](#listing-boot-entries) section, since USB devices will need to be present at system startup in order to be detected.**

When accessing boot entries that are already present in the UEFI NVRAM, running `bootnext` is typically faster and easier than rebooting and accessing the UEFI boot menu. Its ability to automatically select the first boot entry whose description matches a user-provided regular expression also makes it slightly faster than directly running `efibootmgr` or `bcdedit`.

When accessing boot entries that are not already present in the UEFI NVRAM and require a reboot in order to be detected (e.g USB devices that were plugged in after system startup), rebooting and running `bootnext` to trigger a second reboot will always be slower than just rebooting once and accessing the UEFI boot menu directly. That being said, using `bootnext` might still be preferable if you are unsure of the correct key to press to enter the boot menu at startup, or if you are using a machine with a buggy UEFI implementation that refuses to display a boot menu. (Note that if you do happen to have a machine with a buggy UEFI implementation, there are [other workarounds available](https://wiki.archlinux.org/title/Unified_Extensible_Firmware_Interface#Enter_firmware_setup_without_function_keys) that you can always use to access the UEFI setup or boot into another OS, so `bootnext` simply represents a convenience on these machines rather than a necessity.)


## Installation

### Prerequisites

Although the `bootnext` executable itself is fully self-contained, it does require that the appropriate OS-specific UEFI variable manipulation tool is available at runtime:

- Under Linux, the [efibootmgr](https://github.com/rhboot/efibootmgr) command needs to be installed. It is available in the system package repositories of most distributions. For example:
    
    - Arch / Manjaro: `sudo pacman -S efibootmgr`
    - CentOS / Fedora / RHEL: `sudo dnf install efibootmgr`
    - Debian / Ubuntu: `sudo apt-get install efibootmgr`
    - openSUSE: `sudo zypper install efibootmgr`
    
- Under Windows, the `bcdedit` command ships with the operating system by default, so nothing needs to be installed.

### System-wide installation

Pre-compiled binaries can be downloaded from the [releases page](https://github.com/TensorWorks/bootnext/releases), and just need to be placed in a directory that is included in the system's `PATH` environment variable. **Note that you will need to download the appropriate binary for your operating system and CPU architecture,** and each binary is named with the suffix `-OS-ARCH`, where `OS` is either `linux` or `windows` and `ARCH` is the name of the CPU architecture. The architecture names follow the naming conventions [used by the Go programming language](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63#goarch-values), and the following architectures are supported:

- `386`: 32-bit x86
- `amd64`: 64-bit x86, also known as x86_64
- `arm64`: 64-bit ARM

You can use the following commands to download and install the latest release binary under Linux:

```bash
# Note: replace the `amd64` suffix with the appropriate value under other CPU architectures
URL="https://github.com/TensorWorks/bootnext/releases/download/v0.0.1/bootnext-linux-amd64"
sudo curl -fSL "$URL" -o /usr/local/bin/bootnext
sudo chmod +x /usr/local/bin/bootnext
```

Under Windows, run the following command from an elevated command prompt or PowerShell window:

```powershell
# Note: replace the `amd64` suffix with the appropriate value under other CPU architectures
curl.exe -fSL "https://github.com/TensorWorks/bootnext/releases/download/v0.0.1/bootnext-windows-amd64.exe" -o "C:\Windows\System32\bootnext.exe"
```

### Portable installation

If you are using `bootnext` to boot from a multiboot USB device managed by [Ventoy](https://www.ventoy.net/) then it can be handy to store the binaries on the USB device itself. As mentioned in the [*Prerequisites*](#prerequisites) section, the binaries are fully portable and self-contained, so they can simply be downloaded from the [releases page](https://github.com/TensorWorks/bootnext/releases) and copied to the Ventoy filesystem partition on the USB device. Note that if you are running `bootnext` from a USB device on a Linux system then you will still need to ensure the `efibootmgr` command is installed on the system, just as you would when running a system-wide installation of `bootnext`.

With the binaries copied to the Ventoy partition, booting to the USB device is as simple as running `bootnext usb` from a terminal or command prompt in the directory containing the binaries, replacing `bootnext` with the appropriate binary for the system (e.g. `./bootnext-linux-amd64` under Linux or `.\bootnext-windows-amd64.exe` under Windows). As discussed in the [*Automatic privilege elevation*](#automatic-privilege-elevation) section, you will be prompted for elevated privileges (i.e. a `sudo` password request under Linux or a User Account Control dialog under Windows) if you are not running `bootnext` as a user with administrative privileges.

For an even simpler workflow on Windows machines, you can create a batch file and place it in the same directory as the `bootnext` binaries:

```bat
@rem Note: replace the `amd64` suffix with the appropriate value under other CPU architectures
%~dp0.\bootnext-windows-amd64.exe usb
```

With this batch file in place, booting to the USB device is as simple as double-clicking the `.bat` file in Windows Explorer, and accepting the UAC prompt that is displayed.


## Usage

### Listing boot entries

Before making any changes to the UEFI configuration, you can list the available UEFI boot entries by specifying the `--list` flag:

```bash
bootnext --list
```

Querying the available boot entries can be useful in scenarios where you are unsure of exactly how a given boot entry is labelled, and can help to inform the pattern that you specify when instructing `bootnext` to select a target boot entry.

There are a couple of important things to note regarding the UEFI boot entries that are listed:

- Boot entries are detected by the system UEFI/BIOS at startup, and the list that the currently running operating system sees will reflect the entries that were present when the system first booted. As a result, you will only see entries for USB devices if those devices were plugged in when the machine was powered on, and you will continue to see entries for USB devices that were present at startup even if you unplug the devices and the entries are invalid.

- Unlike `efibootmgr` under Linux, `bcdedit` under Windows does not seem to report boot entries for network booting options (e.g. PXE booting) on some machines. As a result, more UEFI boot entries may be listed under Linux than under Windows for the same machine.

### Booting into a target OS

When booting into a target OS, `bootnext` requires a single argument to specify which UEFI boot entry should be selected. This argument represents a case-insensitive regular expression (using the [syntax supported by Go](https://pkg.go.dev/regexp/syntax), which is consistent with other languages such as Perl or Python), but specifying a simple string will behave like a plain string match so long as no characters are included that have a special meaning in the regular expression syntax.

Each UEFI boot entry will be checked against the supplied pattern, and the first entry that matches will be selected as the target. **Note that only a subset of the boot entry's human-readable description needs to match the regular expression pattern, rather than the entire string.**

Consider this example list of boot entries:

```bash
$ bootnext --list

Detected the following UEFI boot entries:
- ID: "0000", Description: "ubuntu"
- ID: "0002", Description: "Linux"
- ID: "0003", Description: "Windows Boot Manager"
- ID: "0005", Description: "UEFI: HTTP IPv4 Intel(R) I211 Gigabit  Network Connection"
- ID: "0006", Description: "UEFI: PXE IPv4 Intel(R) I211 Gigabit  Network Connection"
- ID: "0007", Description: "UEFI: HTTP IPv6 Intel(R) I211 Gigabit  Network Connection"
- ID: "0008", Description: "UEFI: PXE IPv6 Intel(R) I211 Gigabit  Network Connection"
- ID: "0009", Description: "UEFI:  USB, Partition 1"
```

The following regular expressions demonstrate the matching behaviour used by `bootnext`:

- The pattern "`ubuntu`" will match `ubuntu` and that entry will be booted.

- The pattern "`windows`" will match `Windows Boot Manager` and that entry will be booted.

- The pattern "`usb`" will match `UEFI:  USB, Partition 1` and that entry will be booted.

- The pattern "`network`" will match all of the network booting entries, so the first one (in this case, `UEFI: HTTP IPv4 Intel(R) I211 Gigabit  Network Connection`) will be booted.

- The pattern "`uefi`" will match all of the network booting and USB boot entries, so the first one (in this case, `UEFI: HTTP IPv4 Intel(R) I211 Gigabit  Network Connection`) will be booted.

- The pattern "`u`" will match both `ubuntu` and all of the network booting and USB boot entries, so the first one (in this case, `ubuntu`) will be booted.

### Performing a dry run

If you would like to test a regular expression to determine which boot entry will be matched, without actually modifying the `BootNext` UEFI NVRAM variable or rebooting, you can specify the `--dry-run` flag:

```bash
# Tests which boot entry will be matched by the pattern "uefi" without modifying the system
bootnext uefi --dry-run
```

### Automatic privilege elevation

Writing to the system's UEFI NVRAM variables requires administrative privileges under both Linux and Windows, and reading the NVRAM variables also requires administrative privileges under Windows. To , `bootnext` will detect whether it is running with the required privileges for a given command, and automatically request elevated privileges when they are not present:

- Under Linux, `bootnext` will re-run itself using `sudo`, which will prompt the user for their password. This behaviour will be triggered when running a command that writes to the UEFI NVRAM variables and the user is not root. It will not be triggered when running as the root user or when the `--help`, `--list` or `--dry-run` flags are specified.

- Under Windows, `bootnext` will re-run itself as an elevated child process, which will trigger a [User Account Control (UAC)](https://learn.microsoft.com/en-us/windows/security/application-security/application-control/user-account-control/) dialog prompting the user to allow this. This behaviour will be triggered when running a command that reads or writes the UEFI NVRAM variables and the parent process is not already running with elevated privileges. It will not be triggered when the parent process is already running with elevated privileges or when the `--help` flag is specified.
    
    **It is important to note that when a Windows command-line application is run as an elevated child process, it will open a new console window rather than inheriting the console window from its non-elevated parent process.** To prevent the new console window from disappearing after `bootnext` finishes running, the privilege elevation logic will append the `--pause` flag when running the child process, which instructs `bootnext` to pause for the user to press a key before it exits.

If you would like `bootnext` to disable this automatic privilege elevation logic and instead exit with an error when it encounters a permissions error due to insufficient privileges, then you can specify the `--no-elevate` flag:

```bash
# Attempts to boot from a USB device, and fails if running with insufficient privileges
bootnext usb --no-elevate
```

This flag may be useful if you are invoking `bootnext` from an automated script that will run unattended, and an interactive prompt for privilege elevation would potentially hang indefinitely. In such a scenario, the script itself should be run with administrative privileges, and running it with insufficient privileges represents a genuine error that should be detected and reported.

### Setting the `BootNext` variable without rebooting

If you would like to modify the `BootNext` UEFI NVRAM variable without triggering an immediate system reboot, you can specify the `--no-reboot` flag:

```bash
# Sets the BootNext variable to boot from a USB device, and exits without rebooting
bootnext usb --no-reboot
```

The NVRAM variable will be set to the desired value, and will take effect the next time the machine is restarted.


## Troubleshooting

### Booting into Linux just loads its bootloader (e.g. GRUB) and boots the default menu entry

This is the expected behaviour for Linux distros that use a boot manager like GRUB as their UEFI bootloader. You will need to set your Linux distro as the default menu entry to ensure the bootloader boots into your distro when targeting the bootloader from `bootnext`.

### Determining whether an operating system is running under UEFI mode or legacy BIOS mode

The method of determining whether the system is booted in UEFI mode varies based on the operating system:

- Under Windows, there are a number of options available for checking the firmware type, including both command line tools and GUI tools. [This article](https://www.tenforums.com/tutorials/85195-check-if-windows-10-using-uefi-legacy-bios.html) lists various options that apply to Windows 10, Windows 11, Windows Server 2019 and Windows Server 2022.

- Under Linux, the simplest option is to check whether the directory `/sys/firmware/efi` exists. If it does exist then the system was booted in UEFI mode, and if it does not exist then the system was booted in either legacy BIOS mode or with non-UEFI firmware.

### Running `bootnext` prints the error `unsupported system configuration: the operating system has not been booted in UEFI mode`

This indicates that the operating system under which you are running `bootnext` was not itself booted in UEFI mode, and you can confirm this by following the instructions from the section above to check which mode it was booted in. This typically means the OS was installed in legacy BIOS mode, or that you're attempting to run `bootnext` on a device that does not ship with UEFI firmware, such as a single-board computer (e.g. a [Raspberry Pi](https://www.raspberrypi.org/)) or a [Google Chromebook](https://www.google.com/intl/en_au/chromebook/).

**Since `bootnext` specifically targets UEFI systems, it cannot run in these environments.** For machines that ship with UEFI firmware (e.g. most modern computers with x86 CPUs), the only way to run `bootnext` is to either reinstall the operating system in UEFI mode or attempt to convert the existing OS installation to UEFI:

- Existing Windows installations can be converted to UEFI using Microsoft's [MBR2GPT](https://learn.microsoft.com/en-us/windows/deployment/mbr-to-gpt) tool, which ships with Windows 10 version 1703 and newer. The tool facilitates in-place conversion without the need to reinstall anything, and is typically the easiest option for Windows. Correct use of MBR2GPT is outside the scope of this README.

- Existing Linux installations can be converted to UEFI by manually modifying filesystem partitions and installing bootloader files to the EFI System Partition (ESP). This may be more difficult than performing a full reinstall if you are unfamiliar with disk partitioning and with manual configuration of your distribution's preferred bootloader (e.g. GRUB). The details of this process will vary based on your chosen Linux distribution and are outside the scope of this README.

For devices that do not ship with UEFI firmware, it may be possible to flash custom firmware. Note however that this does not guarantee that a device will be able to run `bootnext`, and could also impact the functionality of your device:

- The [Pi Firmware Task Force](https://github.com/pftf) GitHub organisation provides UEFI firmware images for [Raspberry Pi 3](https://github.com/pftf/RPi3) and [Raspberry Pi 4](https://github.com/pftf/RPi4) single-board computers. **However, since [the Raspberry Pi does not have NVRAM](https://github.com/tianocore/edk2-platforms/tree/master/Platform/RaspberryPi/RPi4#nvram), tools such as `efibootmgr` (and by extension `bootnext`) will not function correctly.** Note that use of the UEFI firmware images may also impact the ability to access certain hardware features such as GPIO.

- The [MrChromebox](https://mrchromebox.tech/) website provides a script that can install UEFI firmware on some Google Chromebooks with x86 CPUs. Note that the custom UEFI firmware cannot boot ChromeOS, so a device flashed with this firmware will only be able to boot other operating systems such as Linux and Windows.


## Building from source

Building `bootnext` from source requires [Go](https://go.dev/) 1.18 or newer (1.20 or newer is recommended). To build the binaries for your host platform, run the following commands from the root of the source tree:

```bash
# Downloads the packages that bootnext depends upon
go mod download

# Builds the bootnext binary for the host OS and CPU architecture
go run build.go
```

To build binaries for all supported platforms, specify the `-release` flag:

```bash
# Builds bootnext binaries for all supported OSes and CPU architectures
go run build.go -release
```


## Legal

Copyright &copy; 2023 TensorWorks Pty Ltd. Licensed under the MIT License, see the file [LICENSE](./LICENSE) for details.
