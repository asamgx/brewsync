<div align="center">

# 🍺 BrewSync

### Sync Homebrew packages across macOS machines

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat)](LICENSE)
[![Made with Charm](https://img.shields.io/badge/Made%20with-Charm-F25D94.svg?style=flat)](https://charm.sh)

**Machine-centric** · **Git-based** · **Interactive** · **Non-destructive**

![BrewSync Demo](.github/demo/demo-interactive.gif)

</div>

---

## ✨ Overview

BrewSync keeps your macOS machines in sync. Install something on one machine, easily replicate it on your others — no manual tracking required.

### 🎯 Key Features

<table>
<tr>
<td width="50%">

**📦 Multi-Package Support**
- Homebrew taps, formulae & casks
- VSCode, Cursor & Antigravity extensions
- Go tools
- Mac App Store apps

**🎨 Interactive TUI**
- Beautiful Catppuccin Mocha theme
- Navigate with number keys (1-9)
- Live package selection
- Real-time progress tracking

</td>
<td width="50%">

**⚙️ Smart Management**
- Auto-captured package descriptions
- Profile system for package groups
- Global & per-machine ignore lists
- Diff view between machines
- Operation history logging

**🔒 Safe by Default**
- Import only adds, never removes
- Sync shows preview before applying
- Machine-specific package support
- Dry-run mode for all operations

</td>
</tr>
</table>

### 🏗️ Design Principles

- **Machine-centric**: Each machine maintains its own Brewfile as the source of truth
- **Git-based**: Designed for dotfiles workflows, not cloud storage
- **Non-destructive by default**: Import mode only adds packages, never removes
- **Interactive**: User picks what to install, with batch options available

---

## 📦 Installation

```bash
# Using Make (recommended)
make build        # Build to ./bin/brewsync
make install      # Install to $GOPATH/bin

# Or build directly with Go
go build -o brewsync ./cmd/brewsync
go install ./cmd/brewsync
```

> 💡 **Tip**: Run `make help` to see all available commands. See [MAKEFILE_GUIDE.md](MAKEFILE_GUIDE.md) for details.

---

## 🚀 Quick Start

<table>
<tr>
<td width="33%">

### 1️⃣ Initialize

```bash
brewsync config init
```

Creates `~/.config/brewsync/config.yaml` with machine settings based on hostname.

</td>
<td width="33%">

### 2️⃣ Capture State

```bash
brewsync dump
```

Updates your Brewfile with all installed packages, including descriptions.

</td>
<td width="33%">

### 3️⃣ Check Status

```bash
brewsync status
```

Shows machine info, package counts, and pending changes.

</td>
</tr>
</table>

---

## 💡 Usage Flow

### 🔄 Setting Up Multiple Machines

<table>
<tr>
<td width="50%">

**Machine A (main workstation)**

```bash
1. brewsync config init
2. brewsync dump
3. git commit & push
```

</td>
<td width="50%">

**Machine B (laptop)**

```bash
1. brewsync config init
2. brewsync dump
3. git pull
4. brewsync diff --from machineA
5. brewsync import --from machineA
```

</td>
</tr>
</table>

### 📅 Daily Workflow

```bash
# On your main machine - install new tools
brew install newtool
brew install --cask newapp

# Save state at end of day
brewsync dump
cd ~/dotfiles && git add -A && git commit -m "Update brewfile" && git push

# On another machine - sync up
cd ~/dotfiles && git pull
brewsync status           # See what's new
brewsync import           # Install missing packages
```

---

## 📚 Commands Reference

### 🎯 Core Commands

| Command | Description |
|---------|-------------|
| `dump` | Update Brewfile from installed packages |
| `list` | List packages in a Brewfile |
| `diff` | Show differences between machines |
| `import` | Install missing packages from another machine (interactive TUI) |
| `sync` | Make current machine match source exactly (preview + apply) |

### 🩺 Status & Diagnostics

| Command | Description |
|---------|-------------|
| `status` | Show current machine state overview |
| `doctor` | Validate setup and diagnose issues |
| `history` | View operation history |

### ⚙️ Configuration

| Command | Description |
|---------|-------------|
| `config show` | Display current configuration |
| `config edit` | Open config in $EDITOR |
| `config path` | Show config file path |
| `config init` | Initialize configuration |
| `config add-machine` | Add a new machine |

### 🚫 Ignore Management

| Command | Description |
|---------|-------------|
| `ignore list` | Show all ignored packages |
| `ignore add` | Add package to ignore list |
| `ignore remove` | Remove from ignore list |
| `ignore clear` | Clear all ignored packages |

### 📋 Profile Management

| Command | Description |
|---------|-------------|
| `profile list` | List available profiles |
| `profile show` | Display profile contents |
| `profile install` | Install packages from profile(s) |
| `profile create` | Create a new profile |
| `profile edit` | Edit profile in $EDITOR |
| `profile delete` | Delete a profile |

**Using Profiles:**

```bash
# Create a profile for core tools
brewsync profile create core --description "Essential tools"
brewsync profile edit core

# Install from profile on any machine
brewsync profile install core

# Install multiple profiles at once
brewsync profile install core,dev-go,k8s
```

### 🎛️ Global Flags

```bash
--config string   Config file (default ~/.config/brewsync/config.yaml)
--dry-run         Preview without executing
--verbose, -v     Detailed output
--quiet, -q       Minimal output
--no-color        Disable colored output
--yes, -y         Skip confirmations
```

---

## 📖 Command Examples

### dump

```bash
brewsync dump                    # Update Brewfile with descriptions
brewsync dump --commit           # Commit changes to git
brewsync dump --push             # Commit and push
brewsync dump --dry-run          # Preview changes
```

**Description Support**: By default, `brewsync dump` uses `brew bundle dump --describe` to capture package descriptions from Homebrew's database. Descriptions appear as comments above each package in your Brewfile, making it self-documenting.

To disable automatic descriptions (manual collection), edit your config:
```yaml
dump:
  use_brew_bundle: false
```

See [DUMP_DESCRIPTIONS.md](DUMP_DESCRIPTIONS.md) for more details.

### import

```bash
brewsync import                    # Interactive TUI selection
brewsync import --from air         # From specific machine
brewsync import --from mini,air    # Union of multiple machines
brewsync import --only brew,cask   # Filter categories
brewsync import --skip vscode      # Exclude categories
brewsync import --yes              # Install all without prompts
brewsync import --dry-run          # Preview only
brewsync import --include-machine-specific  # Include machine-specific packages
```

The interactive TUI lets you:
- Toggle packages with `space`
- Select all/none with `a`/`n`
- Filter by category with number keys `1-8`
- Search with `/`
- Mark as ignored with `i`
- Confirm with `enter`

### sync

```bash
brewsync sync                    # Preview mode (shows changes)
brewsync sync --apply            # Execute changes
brewsync sync --from air         # Sync from specific machine
brewsync sync --only brew        # Only sync specific types
brewsync sync --apply --yes      # Apply without confirmation
```

Sync differs from import:
- Import only **adds** missing packages
- Sync **adds AND removes** to match source exactly
- Protected packages (machine-specific, ignored) are never removed

### list

```bash
brewsync list                    # Current machine
brewsync list --from mini        # Another machine
brewsync list --only brew,cask   # Filter by type
brewsync list --format json      # JSON output
```

### diff

```bash
brewsync diff                    # Compare with default source
brewsync diff --from air         # Compare with specific machine
brewsync diff --only brew,cask   # Filter to specific types
brewsync diff --format json      # Output as JSON
```

**Note**: Packages marked with `(ignored)` are in your ignore list and won't be installed during import or sync operations.

### ignore

The ignore system has two layers stored in a separate `ignore.yaml` file:

**Category-level ignores** (ignore entire package types):
```bash
brewsync ignore category add mas                    # Ignore ALL Mac App Store apps
brewsync ignore category add go --machine mini      # Ignore ALL Go tools on mini
brewsync ignore category remove mas                 # Remove category ignore
brewsync ignore category list                       # List ignored categories
```

**Package-level ignores** (ignore specific packages):
```bash
brewsync ignore add cask:bluestacks                 # Ignore specific package
brewsync ignore add brew:postgresql --global        # Ignore globally
brewsync ignore add cask:steam --machine mini       # Ignore on specific machine
brewsync ignore remove cask:bluestacks              # Remove from ignore
brewsync ignore list                                # Show all ignores (categories + packages)
```

**Utility commands**:
```bash
brewsync ignore path                                # Show ignore file location
brewsync ignore init                                # Create default ignore.yaml
```

**Two-Layer System**:
- **Categories**: Ignore entire package types (e.g., all `mas`, all `go`)
- **Packages**: Ignore specific packages within non-ignored categories

**Scope**: Global (all machines) or per-machine

**Note**: Ignore lists apply to `import`, `sync`, and `diff` commands but **not** to `dump`. The dump command captures everything installed (source of truth).

### profile

```bash
brewsync profile list                           # List profiles
brewsync profile show core                      # Show profile contents
brewsync profile install core                   # Install from profile
brewsync profile install core,dev-go            # Install multiple
brewsync profile create web-dev                 # Create new profile
brewsync profile edit core                      # Edit in $EDITOR
brewsync profile delete old-profile             # Delete profile
```

## Configuration

Configuration is split into two files:
- **`config.yaml`** - Main settings (machines, defaults, output)
- **`ignore.yaml`** - Ignore rules (separate file for better organization)

Both are stored in `~/.config/brewsync/`.

### Example config.yaml

```yaml
machines:
  mini:
    hostname: "Andrews-Mac-mini"
    brewfile: "/Users/andrew/dotfiles/_brew_mini/Brewfile"
    description: "Mac Mini - primary workstation"
  air:
    hostname: "Andrews-MacBook-Air"
    brewfile: "/Users/andrew/dotfiles/_brew_air/Brewfile"
    description: "MacBook Air - portable"

current_machine: auto  # Auto-detect from hostname
default_source: mini   # Default machine for import/diff

default_categories:
  - tap
  - brew
  - cask
  - vscode
  - cursor
  - antigravity
  - go
  - mas

dump:
  use_brew_bundle: true  # Use 'brew bundle dump --describe' for descriptions

output:
  color: true
  verbose: false
```

### Example ignore.yaml

```yaml
# Global ignores (apply to all machines)
global:
  categories:
    - mas           # Ignore ALL Mac App Store apps
    - go            # Ignore ALL Go tools

  packages:
    cask:
      - "company-vpn"     # Specific cask to ignore
    brew:
      - "postgresql"      # Specific brew formula to ignore

# Machine-specific ignores
machines:
  mini:
    categories:
      - antigravity       # Don't use Antigravity on mini

    packages:
      cask:
        - "bluestacks"    # Don't need on workstation

  air:
    categories: []

    packages:
      brew:
        - "scrcpy"        # Laptop-specific exclusion
```

## Profiles

Profiles are YAML files stored in `~/.config/brewsync/profiles/`.

### Example Profile (`~/.config/brewsync/profiles/core.yaml`)

```yaml
name: core
description: "Essential tools for any machine"

packages:
  tap:
    - homebrew/bundle
  brew:
    - git
    - fzf
    - bat
    - eza
    - fd
    - ripgrep
    - lazygit
    - starship
  cask:
    - raycast
    - iterm2
  vscode:
    - vscodevim.vim
    - eamodio.gitlens
```

## Directory Structure

```
~/.config/brewsync/
├── config.yaml           # Main configuration
├── ignore.yaml           # Ignore rules (categories + packages)
├── history.log           # Operation history
└── profiles/             # Profile definitions
    ├── core.yaml
    ├── dev-go.yaml
    └── dev-python.yaml

~/dotfiles/               # Your dotfiles repo
├── _brew_mini/
│   └── Brewfile          # Mini's package list
├── _brew_air/
│   └── Brewfile          # Air's package list
└── ...
```

## Package Types

| Type | Source | Example |
|------|--------|---------|
| `tap` | Homebrew taps | `charmbracelet/tap` |
| `brew` | Homebrew formulae | `git`, `fzf`, `bat` |
| `cask` | Homebrew casks | `raycast`, `slack` |
| `vscode` | VSCode extensions | `golang.go` |
| `cursor` | Cursor extensions | `ms-python.python` |
| `antigravity` | Antigravity extensions | `python.lsp` |
| `go` | Go tools | `golang.org/x/tools/gopls` |
| `mas` | Mac App Store | `497799835` (Xcode) |

## Brewfile Format

BrewSync uses the standard Brewfile format with extensions:

```ruby
# Standard Homebrew entries
tap "homebrew/bundle"
# Distributed revision control system
brew "git"
brew "libpq", link: true
# Launcher and productivity tool
cask "raycast"
mas "Xcode", id: 497799835
vscode "golang.go"

# BrewSync extensions
cursor "golang.go"
antigravity "python.lsp"
go "golang.org/x/tools/gopls"
```

**Package Descriptions**: Comments above packages (e.g., `# Distributed revision control system`) are automatically captured by `brew bundle dump --describe`. This makes your Brewfile self-documenting and helps when reviewing packages across machines.

## Troubleshooting

### Run the doctor command

```bash
brewsync doctor
```

This checks:
- Config file exists and is valid
- Current machine is detected
- Brewfile paths exist
- Required CLI tools are available

### Common Issues

| Issue | Solution |
|-------|----------|
| "Machine not recognized" | Run `brewsync config init` or add machine manually |
| "Brewfile not found" | Run `brewsync dump` to create it |
| "brew command failed" | Check package name, verify network |
| CLI not available | Install missing tool (code, cursor, mas, go) |

## Requirements

- macOS
- Go 1.21+ (for building from source)
- Homebrew
- Optional: VSCode (`code` CLI), Cursor (`cursor` CLI), Antigravity (`agy` CLI), mas-cli, Go

---

## 🔧 Development

```bash
# Quick development cycle
make quick              # Build + test (fastest)
make dev                # Build + show version

# Testing
make test               # Run all tests
make test-coverage      # With coverage
make test-verbose       # Verbose output

# Code quality
make pre-commit         # Format + vet + test (run before committing)
make ci                 # CI checks

# Build
make build              # Build to ./bin/brewsync
make install            # Install to $GOPATH/bin
make release            # Optimized production build

# Demo generation
make demo               # Generate all demo GIFs
make demo-quick         # Generate quick demo
make demo-tui           # Generate TUI demo

# Manual testing
make test-setup         # Setup test environment
# ... follow MANUAL_TEST_GUIDE.md
make test-cleanup       # Cleanup test environment

# More commands
make help               # See all available commands
```

See [MAKEFILE_GUIDE.md](MAKEFILE_GUIDE.md) for complete documentation.

---

## 🗺️ Roadmap

The following TUI features are planned or need fixes:

| # | Item | Description |
|---|------|-------------|
| 1 | Fix TUI Sync page | Sync preview and apply functionality not fully working |
| 2 | Add global `i`/`X` hotkeys | Install (`i`) or uninstall (`X`) package under cursor in List/Diff screens |
| 3 | Fix Config TUI page | Config screen not displaying/editing properly |
| 4 | Fix Profiles TUI page | Profiles screen not fully functional |
| 5 | Support searching with `/` | Add fuzzy search in List, Diff, Import screens |
| 6 | Fix diff lists height bug | Column heights not calculated correctly in some cases |
| 7 | Enhance Doctor screen | Add recommended fixes for warnings + descriptions for each check |

---

## 📄 License

MIT

---

<div align="center">

**Built with [Charm](https://charm.sh)** 💖

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![Bubble Tea](https://img.shields.io/badge/Bubble%20Tea-F25D94?style=flat)](https://github.com/charmbracelet/bubbletea)
[![Lipgloss](https://img.shields.io/badge/Lipgloss-7D56F4?style=flat)](https://github.com/charmbracelet/lipgloss)

[Report Bug](https://github.com/asamgx/brewsync/issues) · [Request Feature](https://github.com/asamgx/brewsync/issues) · [Documentation](CLAUDE.md)

</div>
