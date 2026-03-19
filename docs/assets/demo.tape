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

# --- Navigate variables ---
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms
Down
Sleep 250ms

# --- Toggle secret masking ---
Ctrl+S
Sleep 1.2s
Ctrl+S
Sleep 800ms

# --- Edit a variable ---
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
Down
Sleep 250ms

# Jump to next diff
Type "n"
Sleep 400ms
Type "n"
Sleep 400ms

# Filter diffs only
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

# Quit
Type "q"
Sleep 400ms

# --- Theme preview ---
Type "lazyenv --themes"
Enter
Sleep 1.5s

Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Quit theme preview
Type "q"
Sleep 400ms
