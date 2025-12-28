# AGENTS.md - BrewSync Development Guide for AI Agents

This document helps AI agents understand how to work effectively in the BrewSync codebase.

## Project Overview

**BrewSync** is a CLI tool for syncing Homebrew packages, VSCode/Cursor/Antigravity extensions, Go tools, and Mac App Store apps across multiple macOS machines using a git-based dotfiles workflow.

**Language**: Go 1.25.5  
**Architecture**: Cobra CLI with Bubble Tea TUI components  
**Key Libraries**: Cobra, Viper, Bubble Tea, Bubbles, Lipgloss, Huh

## Quick Reference Commands

### Development Workflow
```bash
make quick              # Fastest: build + test
make dev                # Build locally + show version
make build              # Build to ./bin/brewsync
make install            # Install to $GOPATH/bin
make run ARGS="status"  # Build and run with args
```

### Testing
```bash
make test                      # Run all tests
make test-coverage             # With coverage summary
make test-coverage-detail      # Generate coverage.html
make test-specific PKG=./internal/brewfile  # Test specific package
make test-race                 # Race detector
make test-verbose              # Verbose output
```

### Code Quality
```bash
make pre-commit         # Format + vet + test (run before commits)
make ci                 # CI checks (format + vet + test)
make fmt                # Format all code
make vet                # Run go vet
make check              # fmt + vet
```

### Building
```bash
go build -o brewsync ./cmd/brewsync      # Direct build
go run ./cmd/brewsync --help             # Run without building
make release                             # Optimized release build
```

### Dependencies
```bash
make deps               # Download dependencies
make deps-tidy          # Tidy go.mod
make deps-verify        # Verify checksums
make deps-update        # Update all dependencies
```

## Project Structure

```
brewsync/
├── cmd/brewsync/main.go           # Entry point - calls cli.Execute()
├── internal/
│   ├── cli/                       # Cobra commands (all register in init())
│   │   ├── root.go                # Root command, global flags
│   │   ├── import.go              # Import with TUI selection
│   │   ├── sync.go                # Sync with preview/apply modes
│   │   ├── diff.go                # Diff between machines
│   │   ├── dump.go                # Dump installed packages to Brewfile
│   │   ├── list.go                # List packages from Brewfile
│   │   ├── status.go              # Show machine status
│   │   ├── doctor.go              # Validate setup
│   │   ├── history.go             # View operation history
│   │   ├── profile.go             # Profile subcommands
│   │   ├── config.go              # Config subcommands (uses huh forms)
│   │   └── ignore.go              # Ignore list management
│   ├── config/                    # Configuration (Viper-based)
│   │   ├── config.go              # Load(), Init(), ConfigPath()
│   │   ├── types.go               # Config, Machine, IgnoreConfig structs
│   │   ├── machine.go             # DetectMachine(), GetLocalHostname()
│   │   ├── defaults.go            # Default values, DefaultCategories
│   │   └── ignore.go              # Ignore file operations
│   ├── brewfile/                  # Brewfile parsing
│   │   ├── types.go               # Package, Packages, PackageType, DiffResult
│   │   ├── parser.go              # Parse() - reads Brewfile
│   │   ├── writer.go              # Write() - writes Brewfile
│   │   └── diff.go                # Diff() - compares two Packages
│   ├── profile/
│   │   └── profile.go             # Profile struct, Load(), Save()
│   ├── installer/                 # Package installation
│   │   ├── installer.go           # Manager orchestrator
│   │   ├── brew.go                # BrewInstaller - taps, formulae, casks
│   │   ├── vscode.go              # VSCodeInstaller
│   │   ├── cursor.go              # CursorInstaller
│   │   ├── antigravity.go         # AntigravityInstaller
│   │   ├── mas.go                 # MasInstaller (Mac App Store)
│   │   └── gotools.go             # GoToolsInstaller
│   ├── tui/                       # Bubble Tea UI components
│   │   ├── styles/styles.go       # Shared lipgloss styles and colors
│   │   ├── app/                   # Full-screen TUI app
│   │   │   ├── layout.go          # Layout rendering
│   │   │   ├── keys.go            # KeyMap with all keybindings
│   │   │   └── screens/           # Individual screens (import, sync, etc.)
│   │   ├── selection/             # Package selection TUI
│   │   │   ├── keys.go            # KeyMap with all keybindings
│   │   │   ├── model.go           # Bubble Tea Model, Init/Update/View
│   │   │   └── view.go            # Rendering logic
│   │   └── progress/
│   │       └── model.go           # Installation progress UI
│   ├── history/
│   │   └── history.go             # Log(), Read() - append-only history
│   ├── exec/
│   │   └── runner.go              # Run() - executes shell commands
│   └── debug/
│       └── debug.go               # Debug utilities
└── pkg/version/
    └── version.go                 # Version, Commit, Date variables
```

## Code Organization & Patterns

### Package Types

BrewSync supports 8 package types defined in `internal/brewfile/types.go`:

```go
TypeTap         PackageType = "tap"          // Homebrew taps
TypeBrew        PackageType = "brew"         // Homebrew formulae
TypeCask        PackageType = "cask"         // Homebrew casks
TypeVSCode      PackageType = "vscode"       // VSCode extensions
TypeCursor      PackageType = "cursor"       // Cursor extensions
TypeAntigravity PackageType = "antigravity"  // Antigravity extensions
TypeGo          PackageType = "go"           // Go tools
TypeMas         PackageType = "mas"          // Mac App Store apps
```

### Core Types & Interfaces

#### brewfile.Package
```go
type Package struct {
    Type        PackageType       // tap, brew, cask, vscode, cursor, antigravity, go, mas
    Name        string            // Package identifier
    FullName    string            // For mas: app name
    Options     map[string]string // link: true, id: 123, etc.
    Description string            // From brew bundle dump --describe
}

// Key method
func (p Package) ID() string  // Returns "type:name"
```

#### brewfile.Packages
```go
type Packages []Package

// Important methods
func (ps Packages) ByType() map[PackageType][]Package
func (ps Packages) Filter(types ...PackageType) Packages
func (ps Packages) AddUnique(packages ...Package) Packages  // Prevents duplicates
func (ps Packages) MergeUnique(other Packages) Packages     // Merges with deduplication
```

#### brewfile.DiffResult
```go
type DiffResult struct {
    Additions Packages  // In source but not current
    Removals  Packages  // In current but not source
    Common    Packages  // In both
}
```

#### installer.Manager
```go
type Manager struct { /* contains type-specific installers */ }

func NewManager() *Manager
func (m *Manager) Install(pkg brewfile.Package) error
func (m *Manager) Uninstall(pkg brewfile.Package) error
func (m *Manager) InstallMany(packages Packages, onProgress func(...)) error
func (m *Manager) InstallWithProgress(pkg Package, onOutput func(line string)) error
```

### Common Patterns

#### 1. Config Loading (Singleton Pattern)
```go
cfg, err := config.Load()  // Singleton, auto-detects current machine
currentMachine := cfg.CurrentMachine
```

#### 2. Brewfile Operations
```go
// Parse Brewfile
pkgs, err := brewfile.Parse(path)

// Compare two package lists
diff := brewfile.Diff(sourcePkgs, currentPkgs)

// Filter by types
filtered := packages.Filter(brewfile.TypeBrew, brewfile.TypeCask)

// Write Brewfile
err = brewfile.Write(path, pkgs)
```

#### 3. Package Deduplication (Critical Pattern)
```go
// Add unique packages (prevents duplicates)
allPackages = allPackages.AddUnique(extensions...)

// Merge with preservation (keeps better descriptions)
merged := list1.MergeUnique(list2)

// Track additions for verbose output
beforeCount := len(allPackages)
allPackages = allPackages.AddUnique(vscodeExtensions...)
addedCount := len(allPackages) - beforeCount
```

**Why this matters**: When dumping, `brew bundle dump` may include packages that we also collect manually (e.g., VSCode extensions installed via Homebrew). Deduplication prevents duplicate entries.

#### 4. Command Execution
```go
// Using exec.Runner
runner := exec.NewRunner()
output, err := runner.Run("brew", "list", "--formula")
lines, err := runner.RunLines("brew", "tap")

// Check if command exists
if !runner.Exists("code") {
    return errors.New("VSCode not installed")
}
```

#### 5. Ignore System (Two-Layer)
```go
// Categories: Ignore entire package types
cfg.AddCategoryIgnore("mas", "")          // Global
cfg.AddCategoryIgnore("go", "mini")       // Machine-specific

// Packages: Ignore specific packages
cfg.AddPackageIgnore("cask:bluestacks", "")  // Global
cfg.AddPackageIgnore("brew:scrcpy", "air")   // Machine-specific
```

#### 6. History Logging
```go
history.LogImport(machine, source, []string{"brew:git"})
history.LogSync(machine, source, addedCount, removedCount)
history.LogDump(machine, map[string]int{"brew": 10}, committed)
```

### Cobra Command Structure

All CLI commands in `internal/cli/` follow this pattern:

```go
var someCmd = &cobra.Command{
    Use:   "command",
    Short: "Short description",
    Long:  `Detailed description with examples`,
    RunE:  runCommand,
}

func init() {
    someCmd.Flags().StringVar(&someFlag, "flag", "", "flag description")
    rootCmd.AddCommand(someCmd)
}

func runCommand(cmd *cobra.Command, args []string) error {
    // 1. Load config
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    // 2. Parse flags and validate
    
    // 3. Perform operation
    
    // 4. Log to history
    
    return nil
}
```

**Global flags** (defined in `root.go`):
- `--dry-run`: Preview without executing
- `--verbose, -v`: Detailed output
- `--quiet, -q`: Minimal output
- `--no-color`: Disable colored output
- `--yes, -y`: Skip confirmations
- `--config <path>`: Use alternate config file

### Bubble Tea TUI Patterns

The TUI uses the Elm architecture (Model-View-Update):

#### Selection TUI (`internal/tui/selection/`)
```go
// Create model
model := selection.New(title, packages)
model.SetIgnored(ignoredMap)
model.SetSelected(selectedMap)

// Run TUI
p := tea.NewProgram(model)
finalModel, err := p.Run()

// Get results
selected := finalModel.(selection.Model).Selected()
ignored := finalModel.(selection.Model).Ignored()
```

#### Progress TUI (`internal/tui/progress/`)
```go
model := progress.New(title, packages, func(pkg Package) error {
    return installer.Install(pkg)
})

p := tea.NewProgram(model)
finalModel, err := p.Run()

installed := finalModel.(progress.Model).Installed()
failed := finalModel.(progress.Model).Failed()
```

#### Full-Screen TUI (`internal/tui/app/`)
The main TUI (launched with `brewsync` with no args) uses screens:
- Import screen
- Sync screen
- Diff screen
- List screen
- Profile screen
- Config screen
- Setup wizard

### Styling & Colors

All TUI components use **Catppuccin Mocha** color palette defined in `internal/cli/root.go` and `internal/tui/styles/styles.go`:

```go
// Common styles
styleSuccess = lipgloss.NewStyle().Foreground(catGreen).Bold(true)
styleError   = lipgloss.NewStyle().Foreground(catRed).Bold(true)
styleWarning = lipgloss.NewStyle().Foreground(catPeach).Bold(true)
styleBold    = lipgloss.NewStyle().Foreground(catLavender).Bold(true)
styleDim     = lipgloss.NewStyle().Foreground(catOverlay0)
```

**Color consistency**: Always use the Catppuccin palette for new UI elements. Don't introduce arbitrary colors.

## Testing Conventions

### Test File Naming
- Test files: `*_test.go`
- Co-located with implementation files
- 17 test files covering core functionality

### Test Patterns
```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := setupTestData()
    
    // Act
    result, err := FunctionToTest(input)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Test Coverage
- `brewfile`: 95% - parser, diff logic (excellent coverage)
- `config`: 60% - loading, defaults
- `exec`: 100% - command execution
- `history`: 26% - log parsing (needs improvement)

### Running Tests
```bash
go test ./...                              # All tests
go test ./internal/brewfile -v             # Specific package
go test ./internal/config -cover           # With coverage
make test                                  # Via Makefile
make test-specific PKG=./internal/brewfile # Specific with Make
```

## Configuration Files

### Main Config (`~/.config/brewsync/config.yaml`)
```yaml
machines:
  mini:
    hostname: "Andrews-Mac-mini"
    brewfile: "/Users/andrew/dotfiles/_brew_mini/Brewfile"
    description: "Mac Mini - primary workstation"

current_machine: auto  # or explicit: "mini"
default_source: mini
default_categories: [tap, brew, cask, vscode, cursor, antigravity, go, mas]

dump:
  use_brew_bundle: true  # Use 'brew bundle dump --describe' for descriptions

machine_specific:
  mini:
    brew: ["postgresql@16", "redis"]
    cask: ["orbstack"]

output:
  color: true
  verbose: false
  show_descriptions: true
```

### Ignore File (`~/.config/brewsync/ignore.yaml`)
Two-layer system:
1. **Categories**: Ignore ALL packages of a type
2. **Packages**: Ignore specific packages within non-ignored types

```yaml
global:
  categories:
    - mas           # Ignore ALL Mac App Store apps
  packages:
    cask: ["company-vpn"]

machines:
  mini:
    categories:
      - antigravity
    packages:
      cask: ["bluestacks"]
```

## Important Gotchas & Non-Obvious Patterns

### 1. Ignore Lists vs. Dump Command
**Critical**: Ignore lists apply to `import`, `sync`, and `diff` but **NOT** to `dump`. The dump command captures everything installed (source of truth), regardless of ignore lists.

### 2. Package Descriptions
When `dump.use_brew_bundle: true` (default):
- Uses `brew bundle dump --describe` for Homebrew packages (includes descriptions)
- Performance: ~1-2 seconds for 100+ packages
- Descriptions stored as comments above packages in Brewfile
- VSCode/Cursor/Antigravity/Go/mas extensions added afterward with deduplication

### 3. Machine-Specific Packages
Packages in `machine_specific` config:
- Won't be suggested for other machines during import
- Won't be removed during sync on their designated machine
- Require explicit `--include-machine-specific` flag to import

### 4. Current Machine Detection
Machine detection uses hostname matching:
```go
hostname, _ := os.Hostname()
for name, machine := range cfg.Machines {
    if machine.Hostname == hostname {
        cfg.CurrentMachine = name
        break
    }
}
```

Override with `MACHINE` env var: `MACHINE=air brewsync status`

### 5. Sync vs. Import
- **Import**: Additive only (installs missing packages, never removes)
- **Sync**: Makes machines identical (adds AND removes packages)
- **Sync defaults to dry-run**: Must use `--apply` to execute changes

### 6. Brewfile Parser Behavior
- Comments immediately before a package line become its description
- Empty lines reset the "last comment" tracker
- Unknown lines are silently skipped (not an error)

### 7. Deduplication is Critical
When collecting extensions, always use `AddUnique()`:
```go
// WRONG - creates duplicates
allPackages = append(allPackages, vscodeExtensions...)

// RIGHT - prevents duplicates
allPackages = allPackages.AddUnique(vscodeExtensions...)
```

### 8. Command Registration
Commands self-register via `init()` functions. When adding a new command:
```go
func init() {
    rootCmd.AddCommand(newCmd)  // Auto-runs at package import
}
```

### 9. Error Formatting
Use `fmt.Errorf()` with `%w` for error wrapping:
```go
return fmt.Errorf("failed to load config: %w", err)
```

### 10. TUI State Management
Bubble Tea models are immutable. Always return updated model:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Update m fields
    return m, cmd  // Return modified copy
}
```

## Version Management

Version info in `pkg/version/version.go` is set via ldflags:
```bash
-ldflags "-X github.com/asamgx/brewsync/pkg/version.Version=$(VERSION) \
          -X github.com/asamgx/brewsync/pkg/version.Commit=$(COMMIT) \
          -X github.com/asamgx/brewsync/pkg/version.Date=$(BUILD_DATE)"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Nothing to do |
| 3 | User cancelled |
| 4 | Config error |
| 5 | Brewfile not found |
| 6 | Machine not recognized |

## Development Best Practices

### 1. Adding New Package Types
To add a new package type:
1. Add constant to `internal/brewfile/types.go` (TypeXxx)
2. Update `AllTypes()` and `ParsePackageType()`
3. Create installer in `internal/installer/xxx.go` implementing `Installer` interface
4. Register installer in `installer.Manager`
5. Update parser patterns in `internal/brewfile/parser.go`
6. Update writer in `internal/brewfile/writer.go`
7. Add to default categories in `internal/config/defaults.go`
8. Update config types in `internal/config/types.go` (PackageIgnoreList, etc.)

### 2. Adding New Commands
1. Create `internal/cli/command.go`
2. Define command with `cobra.Command`
3. Add `init()` function that registers with `rootCmd.AddCommand()`
4. Follow error handling pattern (return errors, don't log)
5. Use global flags (dryRun, verbose, etc.) from root.go
6. Log operations to history

### 3. Modifying TUI Components
- Use Catppuccin Mocha colors only
- Keep layout constants in `internal/tui/app/layout.go`
- Key bindings in separate `keys.go` files
- Separate view rendering logic in `view.go` when complex
- Test on different terminal sizes (minimum 80x24)

### 4. Error Handling
- Return errors, don't log them in library code
- Only log in command handlers (internal/cli/)
- Use `fmt.Errorf` with `%w` for wrapping
- Provide context in error messages

### 5. Config Changes
- Update defaults in `internal/config/defaults.go`
- Add validation in `internal/config/config.go`
- Document in CLAUDE.md
- Maintain backward compatibility when possible

### 6. Testing New Features
```bash
# Unit tests
go test ./internal/newfeature -v

# Integration test manually
make build
./bin/brewsync newcommand --dry-run

# Test TUI rendering
./bin/brewsync newcommand  # Resize terminal, test edge cases
```

## Useful Development Commands

```bash
# Quick iteration loop
make quick && ./bin/brewsync status

# Debug with verbose output
go run ./cmd/brewsync --verbose status

# Test specific functionality
go run ./cmd/brewsync import --from air --dry-run

# Check code quality before commit
make pre-commit

# Update dependencies
make deps-tidy
make deps-verify

# View coverage
make test-coverage-detail
open coverage.html
```

## Common Tasks

### Task: Add a new CLI flag to import command
1. Edit `internal/cli/import.go`
2. Add variable: `var importNewFlag string`
3. Register in `init()`: `importCmd.Flags().StringVar(&importNewFlag, ...)`
4. Use in `runImport()`
5. Update command's `Long` description with example

### Task: Fix a bug in Brewfile parsing
1. Write a failing test in `internal/brewfile/parser_test.go`
2. Fix in `internal/brewfile/parser.go`
3. Verify test passes: `make test-specific PKG=./internal/brewfile`
4. Update CLAUDE.md if behavior changes

### Task: Add a new TUI screen
1. Create `internal/tui/app/screens/newscreen.go`
2. Implement Model with Init, Update, View methods
3. Add to screen enum in `internal/tui/app/model.go`
4. Add navigation in `internal/tui/app/keys.go`
5. Wire up in main app Update method

### Task: Improve test coverage
1. Check current coverage: `make test-coverage-detail`
2. Open `coverage.html`, find red sections
3. Add tests in corresponding `_test.go` file
4. Run specific tests: `make test-specific PKG=./internal/package`

## Documentation Files

- **CLAUDE.md**: Comprehensive design document (architecture, types, commands)
- **README.md**: User-facing documentation (installation, quick start, usage)
- **MAKEFILE_GUIDE.md**: Makefile command reference
- **AGENTS.md**: This file (development guide for AI agents)

When making significant changes, update CLAUDE.md to reflect design decisions.

## Debugging Tips

### Enable Verbose Output
```bash
brewsync --verbose command
```

### Check Config Loading
```bash
brewsync config show
```

### Validate Setup
```bash
brewsync doctor
```

### View Recent Operations
```bash
brewsync history
```

### Test Brewfile Parsing
```go
packages, err := brewfile.Parse("/path/to/Brewfile")
for _, pkg := range packages {
    fmt.Printf("%s: %s\n", pkg.Type, pkg.Name)
}
```

### Debug TUI Issues
Add debug logging in Update method:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    fmt.Fprintf(os.Stderr, "msg: %#v\n", msg)  // Logs to stderr
    // ... rest of update
}
```

## Performance Considerations

- **Brewfile dump**: ~1-2 seconds for 100+ packages (with descriptions)
- **Package installation**: Varies by package size and network speed
- **Config loading**: Cached after first load (singleton pattern)
- **TUI rendering**: Optimized for 60 FPS, handles 1000+ packages

## Future Improvement Areas (Known Technical Debt)

1. **History package test coverage**: Currently 26%, should be >80%
2. **Progress streaming**: Only Brew installer supports output streaming
3. **Error recovery**: Some installers don't handle partial failures gracefully
4. **Config migration**: No automatic migration for config schema changes
5. **Parallel installs**: Currently sequential, could parallelize where safe

---

**For detailed architecture and design philosophy, see CLAUDE.md.**  
**For user documentation, see README.md.**  
**For Makefile commands, see MAKEFILE_GUIDE.md.**
