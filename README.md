# Lever

Lever is a collection of Unix-style command-line tools for Windows. It runs as
`lever <command>` and can expose individual commands in PowerShell and CMD.

## Commands

`ls`, `cat`, `grep`, `touch`, `mkdir`, `cp`, `mv`, `rm`, `pwd`,
`clear`, `nc`, `du`, `awk`, `tail`, `curl`, `hash`, `ping`,
`sed`, `uname`, and `wc`.

Run `lever --help` for the command list or `lever <command> --help` for
options. For example:

```powershell
lever ls
lever grep -h
lever curl -I https://example.com
```

## Install on Windows

Download `lever-installer-x64.exe` for x64-compatible Windows or
`lever-installer-x86.exe` for 32-bit Windows from the
[GitHub Releases](https://github.com/oninescom/lever/releases) page when a
release is available. Each installer contains only its matching binary and
sets up PowerShell and CMD commands automatically. Administrator privileges
are not required. Open a new terminal after installation.

To add PowerShell functions and CMD integration when using a standalone
`lever.exe`, place the executable in a permanent directory and run:

```powershell
.\lever.exe install
```

Run `lever uninstall` to remove that shell integration. To remove an
installer installation, use Windows **Installed apps**.

## Build from source

Requires Windows and Go 1.26 or newer:

```powershell
go build -o lever.exe .
go test ./...
```

To build both installers, install Inno Setup 7 or newer, then build each
architecture and compile `lever.iss` twice:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags="-s -w" -o dist/x64/lever.exe .
$env:GOARCH = "386"
go build -trimpath -ldflags="-s -w" -o dist/x86/lever.exe .
ISCC.exe /DTargetArch=x64 lever.iss
ISCC.exe /DTargetArch=x86 lever.iss
```

If `ISCC.exe` is not on `PATH`, invoke it using its full installation path.

The installers are written to `lever-installer-x64.exe` and
`lever-installer-x86.exe`. Prebuilt binaries and generated files are excluded
from Git.

## License

Lever is licensed under the [MIT License](LICENSE). Third-party notices for
distributed binaries are in [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES).
