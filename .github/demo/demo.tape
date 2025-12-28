# BrewSync Demo
# Generate with: vhs demo.tape

Output .github/demo/demo.gif

# Terminal settings
Set FontSize 28
Set Width 1400
Set Height 900
Set TypingSpeed 50ms
Set Theme "Catppuccin Mocha"
Set Padding 20

# Title screen
Type "# BrewSync - Sync packages across macOS machines"
Sleep 1s
Enter
Sleep 500ms

Type "# Built with ❤️  using Charm libraries"
Sleep 1s
Enter
Enter
Sleep 1s

# Show version
Type "brewsync --version"
Sleep 500ms
Enter
Sleep 2s

# Clear and show status
Ctrl+L
Sleep 500ms

Type "# Check current machine status"
Sleep 500ms
Enter
Type "brewsync status"
Sleep 500ms
Enter
Sleep 3s

# Clear and show list
Ctrl+L
Sleep 500ms

Type "# List packages from Brewfile"
Sleep 500ms
Enter
Type "brewsync list --only brew | head -15"
Sleep 500ms
Enter
Sleep 3s

# Clear and show diff
Ctrl+L
Sleep 500ms

Type "# Compare with another machine (dry-run)"
Sleep 500ms
Enter
Type "brewsync diff --from air 2>/dev/null || echo 'Demo: shows package differences'"
Sleep 500ms
Enter
Sleep 3s

# Clear and show doctor
Ctrl+L
Sleep 500ms

Type "# Validate setup"
Sleep 500ms
Enter
Type "brewsync doctor"
Sleep 500ms
Enter
Sleep 3s

# Clear and show help
Ctrl+L
Sleep 500ms

Type "# Available commands"
Sleep 500ms
Enter
Type "brewsync --help"
Sleep 500ms
Enter
Sleep 4s

# Final message
Ctrl+L
Sleep 500ms
Type "# BrewSync - Machine-centric dotfiles sync"
Enter
Type "# Learn more: github.com/asamgx/brewsync"
Sleep 3s
