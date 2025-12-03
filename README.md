# smpc

[![CI](https://github.com/Norgate-AV/smpc/workflows/CI/badge.svg)](https://github.com/Norgate-AV/smpc/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/Norgate-AV/smpc)](https://goreportcard.com/report/github.com/Norgate-AV/smpc)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Release](https://img.shields.io/github/v/release/Norgate-AV/smpc)](https://github.com/Norgate-AV/smpc/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A tool to automate compiling [Crestron][crestron] [SIMPL Windows][smpwin] programs.

[crestron]: https://www.crestron.com/
[smpwin]: https://www.crestron.com/Products/Catalog/Control-and-Management/Software/Programming-Commissioning/SW-SIMPL

## Installation

### Using Scoop

```bash
scoop bucket add norgateav-crestron https://github.com/Norgate-AV/scoop-norgateav-crestron.git
scoop install smpc
```

### Using Go Install

```bash
go install github.com/Norgate-AV/smpc@latest
```

### Manual Installation

1. Clone the repository:

    ```bash
    git clone https://github.com/Norgate-AV/smpc.git && cd smpc
    ```

2. Build and install the binary:

    ```bash
    make install
    ```

    This will compile the `smpc` binary and place it in your `$GOBIN` directory.

## Usage

**Note**: This tool requires administrator privileges. See [Administrator Privileges](#administrator-privileges) for details.

Compile a SIMPL Windows program:

```bash
smpc path/to/your/program.smw
```

The tool will:

1. Launch SIMPL Windows with the specified file
2. Automatically trigger compilation
3. Handle any dialog prompts
4. Parse and display compilation results (errors, warnings, notices)
5. Close SIMPL Windows automatically

Exit codes:

- `0`: Compilation successful (warnings/notices are OK)
- `1`: Compilation failed with errors or runtime error

## Configuration

### Custom SIMPL Windows Path

By default, `smpc` looks for SIMPL Windows at:

```text
C:\Program Files (x86)\Crestron\Simpl\smpwin.exe
```

If SIMPL Windows is installed in a different location, set the `SIMPL_WINDOWS_PATH` environment variable:

```powershell
# PowerShell - Current session only
$env:SIMPL_WINDOWS_PATH = "D:\Custom\Path\To\smpwin.exe"

# Or set it permanently (Windows User environment variable)
[System.Environment]::SetEnvironmentVariable('SIMPL_WINDOWS_PATH', 'D:\Custom\Path\To\smpwin.exe', 'User')

# Or add to your PowerShell profile for automatic loading
Add-Content $PROFILE "`n`$env:SIMPL_WINDOWS_PATH = 'D:\Custom\Path\To\smpwin.exe'"
```

```cmd
:: Command Prompt
set SIMPL_WINDOWS_PATH=D:\Custom\Path\To\smpwin.exe

:: Or set it permanently
setx SIMPL_WINDOWS_PATH "D:\Custom\Path\To\smpwin.exe"
```

## Administrator Privileges

This tool requires elevated permissions to:

- Send keystrokes to SIMPL Windows
- Monitor and interact with system dialogs
- Automate the compilation process

### Interactive Use

For the best experience, run `smpc` from an administrator terminal. This allows you to see the
compilation output and logs directly in your terminal.

If you run `smpc` from a non-elevated terminal, it will automatically:

- Check if it's running with administrator privileges
- If not, display a UAC (User Account Control) prompt to request elevation
- Relaunch itself with the required permissions in a new elevated terminal window

You may see a UAC prompt asking "Do you want to allow this app to make changes to your device?" -
click **Yes** to continue.

**Note**: When auto-elevation occurs, the new terminal window will close immediately after
compilation completes. You can view the compilation logs afterward using `smpc --logs`.

### CI/CD Environments

For automated builds in CI/CD pipelines (GitHub Actions, Jenkins, etc.), UAC prompts will block
execution. There are two approaches to handle this:

#### Option 1: Run CI Agent with Administrator Privileges

Configure your CI agent or runner to execute with administrator privileges. This allows UAC prompts
to be automatically approved.

#### Option 2: Disable UAC

Disable UAC on the build machine to prevent interactive prompts. Refer to your Windows
documentation or system administrator for the appropriate method for your environment.

Both approaches are necessary workarounds for automating SIMPL Windows compilation, which requires
elevated privileges to interact with the application's UI.

## LICENSE

[MIT](./LICENSE)
