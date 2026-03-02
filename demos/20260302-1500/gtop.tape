Output gtop.gif
Output gtop.mp4

Set Shell "bash"
Set FontFamily "Berkeley Mono"
Set FontSize 13
Set LineHeight 1.2
Set Width 1000
Set Height 580
Set Padding 24
Set TypingSpeed 80ms
Set Framerate 24

Set Theme {
  "name": "glyph-dark",
  "background":          "#0f0f0d",
  "foreground":          "#c8c4b8",
  "cursor":              "#c44040",
  "selectionBackground": "#2a2a26",
  "black":               "#1a1a18",
  "red":                 "#c44040",
  "green":               "#5a9060",
  "yellow":              "#b89040",
  "blue":                "#4878a8",
  "magenta":             "#983848",
  "cyan":                "#408888",
  "white":               "#c8c4b8",
  "brightBlack":         "#5a5a54",
  "brightRed":           "#e85050",
  "brightGreen":         "#70b078",
  "brightYellow":        "#d4a858",
  "brightBlue":          "#6898c8",
  "brightMagenta":       "#c84860",
  "brightCyan":          "#58a8a8",
  "brightWhite":         "#e8e4d8"
}

Hide
Type "cd /Users/petegriffiths/code/go/src/tui && go run ./cmd/gtop"
Enter
Sleep 3s
Show
Sleep 2s
Type "go"
Sleep 1.5s
Type "pher"
Sleep 1.5s
Ctrl+N
Sleep 400ms
Ctrl+N
Sleep 400ms
Ctrl+N
Sleep 800ms
Ctrl+A
Ctrl+K
Sleep 500ms
Type "node"
Sleep 2s
Ctrl+A
Ctrl+K
Sleep 500ms
Sleep 2s
Hide
Escape
