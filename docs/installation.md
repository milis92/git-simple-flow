# Installation

[README](../README.md) · [Workflow Guide](simple-flow.md)

This guide covers every way to install `git-sf` on Linux, macOS, and Windows.

## Prerequisites

- **git** 2.x or later
- **gh** ([GitHub CLI](https://cli.github.com/)) — required for PR operations

## Quick Install

**Any platform with Go:**
```sh
go install github.com/milis92/git-simple-flow@latest
```

To install a specific version:
```sh
go install github.com/milis92/git-simple-flow@v1.2.3
```

## Command-Line Download

Download the latest stable release binary for your platform.

<details>
<summary><strong>Linux (amd64)</strong></summary>

```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_linux_amd64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```
</details>

<details>
<summary><strong>Linux (arm64)</strong></summary>

```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_linux_arm64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```
</details>

<details>
<summary><strong>macOS (Apple Silicon)</strong></summary>

```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_darwin_arm64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```
</details>

<details>
<summary><strong>macOS (Intel)</strong></summary>

```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_darwin_amd64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```
</details>

<details>
<summary><strong>Windows (amd64)</strong></summary>

Download [git-sf_windows_amd64.zip](https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_windows_amd64.zip), extract to a directory such as `C:\tools\`, and add that directory to your system `PATH` via Settings > System > About > Advanced system settings > Environment Variables.
</details>

<details>
<summary><strong>Windows (arm64)</strong></summary>

Download [git-sf_windows_arm64.zip](https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_windows_arm64.zip), extract to a directory such as `C:\tools\`, and add that directory to your system `PATH` via Settings > System > About > Advanced system settings > Environment Variables.
</details>

Alternatively, download a pre-built binary for any platform from the [Releases page](https://github.com/milis92/git-simple-flow/releases).

## Package Managers

### APT (Debian / Ubuntu)

> [!NOTE]
> Package filenames include the version number, so these commands use `gh` to download the latest release automatically.
> Since `gh` is already required for `git-sf`'s PR operations, it should already be installed.

**amd64:**
```sh
gh release download --repo milis92/git-simple-flow --pattern '*_linux_amd64.deb' --output git-sf.deb
sudo dpkg -i git-sf.deb
```

**arm64:**
```sh
gh release download --repo milis92/git-simple-flow --pattern '*_linux_arm64.deb' --output git-sf.deb
sudo dpkg -i git-sf.deb
```

### RPM (Fedora / RHEL)

**amd64:**
```sh
gh release download --repo milis92/git-simple-flow --pattern '*_linux_amd64.rpm' --output git-sf.rpm
sudo rpm -i git-sf.rpm
```

**arm64:**
```sh
gh release download --repo milis92/git-simple-flow --pattern '*_linux_arm64.rpm' --output git-sf.rpm
sudo rpm -i git-sf.rpm
```

## Upgrading

To upgrade, re-run the same install command you used originally. For `go install`, run `go install github.com/milis92/git-simple-flow@latest` again. For binary downloads, download the new version and replace the existing binary.

## Verify Installation

```sh
git sf -h
```

You should see the list of available commands.

## How It Works with Git

Git automatically discovers any executable named `git-<name>` on your `PATH` and exposes it as `git <name>`. Because the binary is called `git-sf`, you can run:

```sh
git sf feature start my-feature
```

No aliases or configuration needed.

## Shell Completions

Generate and source a completion script for tab completion of commands and flags.

**Bash:**
```sh
# Add to ~/.bashrc
eval "$(git sf completion bash)"
```

**Zsh:**
```sh
# Add to ~/.zshrc
eval "$(git sf completion zsh)"
```

**Fish:**
```sh
git sf completion fish | source
```

To persist across sessions:
```sh
git sf completion fish > ~/.config/fish/completions/git-sf.fish
```

**PowerShell:**
```powershell
# Add to your PowerShell profile
git sf completion powershell | Out-String | Invoke-Expression
```

After sourcing completions, open a new terminal and type `git sf ` followed by Tab to verify suggestions appear.

## Next Steps

- Read the [workflow guide](simple-flow.md) to understand the Simple Flow branching model
- Check the [README](../README.md) for the full command reference and configuration options
