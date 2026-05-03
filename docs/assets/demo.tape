# lazyenv — full demo
# Run: vhs docs/assets/demo.tape
# Optimize: gifsicle -O3 --lossy=80 docs/assets/demo.gif -o docs/assets/demo.gif

Output docs/assets/demo.gif
Set Shell "bash"
Set Width 1100
Set Height 500
Set FontSize 13
Set Framerate 12
Set Theme "Dracula"
Set TypingSpeed 50ms
Set Padding 8

# --- Launch ---
Type "lazyenv env/"
Enter
Sleep 1.5s

# --- Browse files ---
Down
Sleep 350ms
Down
Sleep 350ms

# Select file, focus moves to vars
Enter
Sleep 800ms

# --- Toggle prefix grouping (v0.7.0) ---
Type "g"
Sleep 1.5s

# Cursor is on the first DB var; jump to the DATABASE header and collapse it
Up
Sleep 300ms
Space
Sleep 1.2s

# Walk down to another header, collapse it too
Down
Down
Down
Down
Down
Down
Sleep 300ms
Space
Sleep 1.2s

# Re-expand both
Space
Sleep 600ms
Up
Up
Up
Up
Up
Up
Sleep 200ms
Space
Sleep 800ms

# Disable grouping, back to flat list
Type "g"
Sleep 800ms

# --- Toggle secret masking ---
Ctrl+S
Sleep 1.2s
Ctrl+S
Sleep 700ms

# --- Navigate variables ---
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms

# --- Edit a variable value ---
Type "e"
Sleep 400ms
Ctrl+U
Type "new-value-edited"
Sleep 400ms
Enter
Sleep 800ms

# --- Peek original value ---
Type "p"
Sleep 1.2s
Type "p"
Sleep 400ms

# --- Rename a variable key (E key) ---
Type "E"
Sleep 400ms
Ctrl+U
Type "RENAMED_KEY"
Sleep 400ms
Enter
Sleep 1s

# --- Back to file list ---
Left
Sleep 400ms

# --- Compare mode ---
Type "c"
Sleep 400ms
Down
Sleep 250ms
Enter
Sleep 1.5s

# Navigate diffs
Down
Sleep 250ms
Type "n"
Sleep 400ms
Type "n"
Sleep 400ms

# Filter to differences only
Type "f"
Sleep 1.2s
Type "f"
Sleep 400ms

# Exit compare
Escape
Sleep 600ms

# --- Completeness matrix ---
Type "m"
Sleep 1.5s

Down
Sleep 250ms
Down
Sleep 250ms
Right
Sleep 250ms
Right
Sleep 250ms

# Toggle sort
Type "o"
Sleep 800ms

# Exit matrix
Escape
Sleep 600ms

# Quit (session summary prints to stdout)
Type "q"
Sleep 1s

# --- Theme preview ---
Type "lazyenv --themes"
Enter
Sleep 1.5s

Down
Sleep 280ms
Down
Sleep 280ms
Down
Sleep 280ms
Down
Sleep 280ms
Down
Sleep 280ms

# Quit theme preview
Type "q"
Sleep 400ms
