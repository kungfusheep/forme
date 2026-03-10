// GLYPH DIAGNOSTIC REPORT — prints system + environment info for bug reports
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	. "github.com/kungfusheep/glyph"
	"golang.org/x/sys/unix"
)

func main() {
	goVersion := runtime.Version()
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	kernel := kernelVersion()

	termProgram := detectEmulator()
	termVer := firstOf(os.Getenv("TERM_PROGRAM_VERSION"), "-")
	termRaw := firstOf(os.Getenv("TERM"), "unknown")
	shell := filepath.Base(os.Getenv("SHELL"))
	mux := detectMux()
	ssh := detectSSH()

	colour := parseColour()
	locale := detectLocale()
	w := termWidth()
	h := termHeight()
	widthVal := fmt.Sprintf("%s %dc", bar(min(w*100/220, 100), 14), w)
	heightVal := fmt.Sprintf("%d rows", h)

	glyphVer := glyphVersion()

	app, err := NewInlineApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox.Border(BorderSingle).FitContent()(
			HRule().Char('·'),
			Text("GLYPH FRAMEWORK").Align(AlignCenter),
			Text("DIAGNOSTIC REPORT").Align(AlignCenter),
			HRule().Char('·'),
			HRule().Extend(),
			HBox.MarginVH(0, 1).Gap(1)(
				VBox.Width(9)(
					Text("GO"), Text("OS"), Text("ARCH"), Text("KERNEL"),
					HRule().Extend(),
					Text("EMULATOR"), Text("VERSION"), Text("TERM"), Text("SHELL"), Text("MUX"), Text("SSH"),
					HRule().Extend(),
					Text("COLOUR"), Text("LOCALE"), Text("WIDTH"), Text("HEIGHT"),
					HRule().Extend(),
					Text("GLYPH"), Text("RENDER"),
				),
				VRule().Extend(),
				VBox(
					Text(goVersion), Text(goos), Text(goarch), Text(kernel),
					HRule().Extend(),
					Text(termProgram), Text(termVer), Text(termRaw), Text(shell), Text(mux), Text(ssh),
					HRule().Extend(),
					Text(colour), Text(locale), Text(widthVal), Text(heightVal),
					HRule().Extend(),
					Text(glyphVer), Text("OK"),
				),
			),
		),
	)

	go func() {
		app.RequestRender()
		time.Sleep(80 * time.Millisecond)
		app.Stop()
	}()

	if err := app.RunNonInteractive(); err != nil {
		log.Fatal(err)
	}
}

func kernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func termWidth() int {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 80
	}
	return int(ws.Col)
}

func termHeight() int {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 24
	}
	return int(ws.Row)
}

func detectEmulator() string {
	// check terminal-unique env vars first — these survive multiplexers reliably
	// and are unambiguous, unlike TERM_PROGRAM which can be overridden
	switch {
	case os.Getenv("GHOSTTY_RESOURCES_DIR") != "" || os.Getenv("GHOSTTY_BIN_DIR") != "":
		return "Ghostty"
	case os.Getenv("KITTY_WINDOW_ID") != "":
		return "kitty"
	case os.Getenv("ALACRITTY_SOCKET") != "" || os.Getenv("ALACRITTY_LOG") != "":
		return "Alacritty"
	case os.Getenv("WEZTERM_EXECUTABLE") != "":
		return "WezTerm"
	case os.Getenv("KONSOLE_VERSION") != "" || os.Getenv("KONSOLE_DBUS_SERVICE") != "":
		return "Konsole"
	case os.Getenv("WT_SESSION") != "":
		return "Windows Terminal"
	}
	if os.Getenv("VTE_VERSION") != "" {
		switch {
		case os.Getenv("GNOME_TERMINAL_SERVICE") != "":
			return "GNOME Terminal"
		case os.Getenv("TERMINATOR_UUID") != "":
			return "Terminator"
		}
		return "VTE-based"
	}
	// TERM_PROGRAM is set by many terminals but can be overridden by mux configs
	if tp := os.Getenv("TERM_PROGRAM"); tp != "" {
		switch tp {
		case "iTerm.app":
			return "iTerm2"
		case "Apple_Terminal":
			return "Terminal.app"
		case "WezTerm":
			return "WezTerm"
		case "ghostty":
			return "Ghostty"
		case "Hyper":
			return "Hyper"
		case "vscode":
			return "VS Code"
		case "Tabby":
			return "Tabby"
		case "rio":
			return "Rio"
		}
	}
	// last resort: $TERM encodes the emulator for some terminals
	term := os.Getenv("TERM")
	switch {
	case term == "xterm-ghostty":
		return "Ghostty"
	case strings.Contains(term, "kitty"):
		return "kitty"
	case term == "alacritty":
		return "Alacritty"
	}
	return "unknown"
}

func detectMux() string {
	switch {
	case os.Getenv("TMUX") != "":
		return "tmux"
	case os.Getenv("STY") != "":
		return "screen"
	case os.Getenv("ZELLIJ") != "":
		return "zellij"
	case os.Getenv("WEZTERM_PANE") != "":
		return "wezterm"
	}
	return "none"
}

func detectSSH() string {
	if os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CLIENT") != "" {
		return "yes"
	}
	return "no"
}

func detectLocale() string {
	locale := firstOf(os.Getenv("LC_ALL"), os.Getenv("LC_CTYPE"), os.Getenv("LANG"), "")
	if locale == "" {
		return "unknown"
	}
	if i := strings.LastIndex(locale, "."); i >= 0 {
		return locale[i+1:]
	}
	return locale
}

func parseColour() string {
	switch strings.ToLower(os.Getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return "truecolor"
	case "256color":
		return "256"
	}
	if strings.Contains(os.Getenv("TERM"), "256color") {
		return "256"
	}
	if ct := os.Getenv("COLORTERM"); ct != "" {
		return ct
	}
	return "basic"
}

func bar(pct, width int) string {
	filled := pct * width / 100
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func firstOf(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func glyphVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	for _, dep := range bi.Deps {
		if strings.HasSuffix(dep.Path, "glyph") {
			return dep.Version
		}
	}
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}
