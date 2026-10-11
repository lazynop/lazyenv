# lazyenv: still for the lazynop.dev Home card, the compare view
# Render with: just page-media (the card is this clip's last frame)

Output dist/page-media/card.gif
Set Shell "bash"
Set Width 1080
Set Height 608
Set FontSize 24
Set Padding 16
Set Theme "Dracula"

Hide
Type `cd "$PAGE_DEMO_DIR" && export HOME=/home/demo && clear && lazyenv`
Enter
Sleep 1.5s
Type "c"
Down
Down
Enter
Sleep 500ms
Show
Sleep 300ms
