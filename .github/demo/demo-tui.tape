# BrewSync TUI Demo - Full Navigation
# Generate with: vhs demo-tui.tape
# Shows the full TUI with screen navigation

Output .github/demo/demo-tui.gif

# Terminal settings
Set FontSize 22
Set Width 1600
Set Height 1000
Set TypingSpeed 35ms
Set Theme "Catppuccin Mocha"
Set Padding 20

# Title
Type "# BrewSync - Interactive Terminal UI"
Sleep 1.5s
Enter
Enter
Sleep 500ms

# Launch TUI
Type "brewsync"
Sleep 500ms
Enter
Sleep 2.5s

# Show navigation
Type "2"
Sleep 2s

Type "3"
Sleep 2s

Type "4"
Sleep 2s

Type "5"
Sleep 2s

Type "6"
Sleep 2s

Type "7"
Sleep 2s

Type "1"
Sleep 2s

# Quit
Type "q"
Sleep 500ms

# Show CLI commands
Ctrl+L
Sleep 300ms
Type "# BrewSync CLI Commands"
Enter
Enter
Sleep 500ms

Type "brewsync status"
Sleep 300ms
Enter
Sleep 2.5s

Ctrl+L
Sleep 300ms
Type "brewsync list --only brew | head -12"
Sleep 300ms
Enter
Sleep 2.5s

Ctrl+L
Sleep 300ms
Type "brewsync diff --dry-run 2>/dev/null || echo 'Compares packages with another machine'"
Sleep 300ms
Enter
Sleep 2.5s

# Final screen
Ctrl+L
Sleep 300ms
Type "# BrewSync ✨"
Enter
Type "# • Interactive TUI (press 1-9 to navigate)"
Enter
Type "# • Powerful CLI for automation"
Enter
Type "# • Git-based dotfiles workflow"
Enter
Enter
Type "# github.com/asamgx/brewsync"
Sleep 3s
