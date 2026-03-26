# Installation

## Prerequisites

- **git** 2.x or later
- **gh** ([GitHub CLI](https://cli.github.com/)) — required for PR operations

## Quick Install

**Any platform with Go:**
```sh
go install github.com/milis92/git-simple-flow@latest
```

## Download

Download the latest stable release binary for your platform.

**Linux (amd64):**
```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_linux_amd64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```

**Linux (arm64):**
```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_linux_arm64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```

**macOS (Apple Silicon):**
```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_darwin_arm64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```

**macOS (Intel):**
```sh
curl -fsSL https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_darwin_amd64.tar.gz | tar -xz
sudo mv git-sf /usr/local/bin/
```

**Windows (amd64):**

Download [git-sf_windows_amd64.zip](https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_windows_amd64.zip), extract, and add `git-sf.exe` to your `PATH`.

**Windows (arm64):**

Download [git-sf_windows_arm64.zip](https://github.com/milis92/git-simple-flow/releases/latest/download/git-sf_windows_arm64.zip), extract, and add `git-sf.exe` to your `PATH`.

## Package Managers

### APT (Debian / Ubuntu)

> [!NOTE]
> Package filenames include the version number, so these commands use `gh` to download the latest release automatically.

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

## Manual Download

Download a pre-built binary for your platform from the [Releases page](https://github.com/milis92/git-simple-flow/releases).

Extract the archive and place the `git-sf` binary somewhere on your `PATH`.

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
# To persist: git sf completion fish > ~/.config/fish/completions/git-sf.fish
```

**PowerShell:**
```powershell
# Add to your PowerShell profile
git sf completion powershell | Out-String | Invoke-Expression
```
