# smpc

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
2. Automatically trigger compilation (F12)
3. Handle any dialog prompts
4. Parse and display compilation results (errors, warnings, notices)
5. Close SIMPL Windows automatically

Exit codes:

- `0`: Compilation successful (warnings/notices are OK)
- `1`: Compilation failed with errors or runtime error

## Administrator Privileges

This tool requires elevated permissions to:

- Send keystrokes to SIMPL Windows
- Monitor and interact with system dialogs
- Automate the compilation process

### Interactive Use

When you run `smpc`, it will automatically:

- Check if it's running with administrator privileges
- If not, display a UAC (User Account Control) prompt to request elevation
- Relaunch itself with the required permissions

You may see a UAC prompt asking "Do you want to allow this app to make changes to your device?" - click **Yes** to continue.

### CI/CD Environments

For automated builds in CI/CD pipelines (GitHub Actions, Jenkins, etc.), UAC prompts will block execution. You must either:

- **Run the CI agent/runner with administrator privileges**, or
- **Disable UAC** on the build machine (not recommended for production systems)

To disable UAC temporarily for testing:

```pwsh
# Disable UAC (requires restart)
Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "EnableLUA" -Value 0

# Re-enable UAC (requires restart)
Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "EnableLUA" -Value 1
```

**Note**: Always ensure your CI/CD runner is executing with the necessary privileges before running `smpc`.

## LICENSE

[MIT](./LICENSE)
