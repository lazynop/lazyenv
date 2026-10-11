# lazyenv: demo clip for the project page at https://lazynop.dev/lazyenv/
# Render with: just page-media (its last frame becomes the poster)
#
# The page plays the clip muted, looped and about 360 CSS px wide on phones,
# so the terminal stays at 68 columns: the narrowest width where the default
# layout shows the completeness matrix for all three files.

Output dist/page-media/demo.gif
Set Shell "bash"
Set Width 1080
Set Height 608
Set FontSize 24
Set Padding 16
Set Framerate 30
Set Theme "Dracula"
Set TypingSpeed 90ms

# Off camera: the recipe prepares PAGE_DEMO_DIR; HOME points there so the
# recorder's own config stays out of the shot.
Hide
Type `cd "$PAGE_DEMO_DIR" && clear && HOME="$PWD" lazyenv`
Enter
Sleep 1.5s
Show

# --- Every .env file in the project, secrets masked ---
Sleep 1.8s

# --- Browse: the variables follow the file cursor ---
Down
Sleep 1.1s
Down
Sleep 1.1s
Up
Sleep 150ms
Up
Sleep 600ms

# --- Edit a value ---
Right
Sleep 300ms
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 400ms
Type "e"
Sleep 400ms
Ctrl+U
Type "info"
Sleep 300ms
Enter
Sleep 1.2s

# --- Compare .env with .env.staging ---
Left
Sleep 300ms
Type "c"
Sleep 500ms
Down
Sleep 200ms
Down
Sleep 200ms
Enter
Sleep 1.5s
Type "n"
Sleep 400ms
Type "n"
Sleep 800ms
Escape
Sleep 400ms

# --- Completeness matrix, then the least complete keys first ---
Type "m"
Sleep 1.6s
Type "o"
Sleep 2s

# Back to the start screen, plus the edited value: the poster and the loop
# point both land close to the first frame
Escape
Sleep 800ms
