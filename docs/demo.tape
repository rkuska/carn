Output docs/demo.gif
Set Shell "zsh"
Set FontSize 14
Set Width 1000
Set Height 600
Set Framerate 12
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms

# --- Scene 1: Browser + Navigation ---
Type "carn"
Sleep 500ms
Enter
Sleep 1s
Enter

# Filter
Type "f"
Sleep 1s
Enter
Space
Sleep 500ms
Enter
Down
Type "/claude-search"
Enter
Sleep 500ms
Type "q"

# Navigate conversation list
Down 3
Sleep 200ms

# --- Scene 2: Deep Search ---
Ctrl+S
Sleep 500ms
Type "/readable"
Sleep 1s
Enter
Sleep 1s
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 1s
Type "O"
Sleep 200ms

# --- Scene 3: Viewer Toggle ---
# Scroll through transcript
PageDown
Sleep 1s

# Scroll through transcript
PageDown
Sleep 2s

# Toggle tool calls on/off
Type "T"
Sleep 2s
Type "T"
Sleep 2s
Type "?"
Sleep 4s
Type "q"
Sleep 1s


# --- Scene 4: Resume Session ---
Type "r"
Sleep 5s

# --- Scene 5: Exit ---
Type  "/exit"
Enter
Sleep 500ms
Type "q"
Sleep 500ms
Type "q"
Sleep 2s
