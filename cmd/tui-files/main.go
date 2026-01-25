// tui-files: Simple file browser demo
// Tests: file operations, permissions display, scrolling, navigation
//
// GAPS FOUND:
// 1. No horizontal split layout (for dual-pane file manager)
// 2. No bordered Box component (for visual panel separation)
// 3. Same fixed slots workaround for file list
// 4. No way to have fixed-width columns in a row
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"riffkey"
	. "tui"
)

const maxVisible = 18

type FileEntry struct {
	Name    string
	IsDir   bool
	Size    int64
	Mode    string
	ModTime string
}

type State struct {
	CurrentPath  string
	Entries      []FileEntry
	SelectedIdx  int
	ScrollOffset int
	StatusLine   string
	PreviewLines [5]string
	DisplayLines [maxVisible]string
}

func main() {
	startPath := "."
	if len(os.Args) > 1 {
		startPath = os.Args[1]
	}

	absPath, err := filepath.Abs(startPath)
	if err != nil {
		log.Fatal(err)
	}

	state := &State{
		CurrentPath: absPath,
		StatusLine:  "↑/↓ navigate, Enter=open, Backspace=up, h=hidden, q=quit",
	}
	scanDir(state)

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(buildView(state))

	app.Handle("<Up>", func(_ riffkey.Match) {
		if state.SelectedIdx > 0 {
			state.SelectedIdx--
			adjustScroll(state)
			updateDisplay(state)
			updatePreview(state)
		}
		app.RequestRender()
	})

	app.Handle("<Down>", func(_ riffkey.Match) {
		if state.SelectedIdx < len(state.Entries)-1 {
			state.SelectedIdx++
			adjustScroll(state)
			updateDisplay(state)
			updatePreview(state)
		}
		app.RequestRender()
	})

	app.Handle("<Enter>", func(_ riffkey.Match) {
		if len(state.Entries) > 0 {
			entry := state.Entries[state.SelectedIdx]
			if entry.IsDir {
				state.CurrentPath = filepath.Join(state.CurrentPath, entry.Name)
				state.SelectedIdx = 0
				state.ScrollOffset = 0
				scanDir(state)
			}
		}
		app.RequestRender()
	})

	app.Handle("<BS>", func(_ riffkey.Match) {
		parent := filepath.Dir(state.CurrentPath)
		if parent != state.CurrentPath {
			oldDir := filepath.Base(state.CurrentPath)
			state.CurrentPath = parent
			state.SelectedIdx = 0
			state.ScrollOffset = 0
			scanDir(state)
			// Try to select the directory we came from
			for i, e := range state.Entries {
				if e.Name == oldDir {
					state.SelectedIdx = i
					adjustScroll(state)
					updateDisplay(state)
					break
				}
			}
		}
		app.RequestRender()
	})

	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	app.Handle("<Esc>", func(_ riffkey.Match) {
		app.Stop()
	})

	// Page up/down
	app.Handle("<PageUp>", func(_ riffkey.Match) {
		state.SelectedIdx -= maxVisible
		if state.SelectedIdx < 0 {
			state.SelectedIdx = 0
		}
		adjustScroll(state)
		updateDisplay(state)
		updatePreview(state)
		app.RequestRender()
	})

	app.Handle("<PageDown>", func(_ riffkey.Match) {
		state.SelectedIdx += maxVisible
		if state.SelectedIdx >= len(state.Entries) {
			state.SelectedIdx = len(state.Entries) - 1
		}
		if state.SelectedIdx < 0 {
			state.SelectedIdx = 0
		}
		adjustScroll(state)
		updateDisplay(state)
		updatePreview(state)
		app.RequestRender()
	})

	// Home/End
	app.Handle("<Home>", func(_ riffkey.Match) {
		state.SelectedIdx = 0
		state.ScrollOffset = 0
		updateDisplay(state)
		updatePreview(state)
		app.RequestRender()
	})

	app.Handle("<End>", func(_ riffkey.Match) {
		state.SelectedIdx = len(state.Entries) - 1
		if state.SelectedIdx < 0 {
			state.SelectedIdx = 0
		}
		adjustScroll(state)
		updateDisplay(state)
		updatePreview(state)
		app.RequestRender()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func scanDir(state *State) {
	entries, err := os.ReadDir(state.CurrentPath)
	if err != nil {
		state.StatusLine = "Error: " + err.Error()
		state.Entries = nil
		updateDisplay(state)
		return
	}

	state.Entries = nil

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		entry := FileEntry{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("Jan 02 15:04"),
		}
		state.Entries = append(state.Entries, entry)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(state.Entries, func(i, j int) bool {
		if state.Entries[i].IsDir != state.Entries[j].IsDir {
			return state.Entries[i].IsDir
		}
		return strings.ToLower(state.Entries[i].Name) < strings.ToLower(state.Entries[j].Name)
	})

	updateDisplay(state)
	updatePreview(state)
}

func adjustScroll(state *State) {
	// Keep selection visible
	if state.SelectedIdx < state.ScrollOffset {
		state.ScrollOffset = state.SelectedIdx
	}
	if state.SelectedIdx >= state.ScrollOffset+maxVisible {
		state.ScrollOffset = state.SelectedIdx - maxVisible + 1
	}
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%6.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%6.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%6.1fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%6dB", bytes)
	}
}

func updateDisplay(state *State) {
	for i := 0; i < maxVisible; i++ {
		idx := state.ScrollOffset + i
		if idx < len(state.Entries) {
			e := state.Entries[idx]
			prefix := "  "
			if idx == state.SelectedIdx {
				prefix = "> "
			}

			icon := "  "
			size := formatSize(e.Size)
			if e.IsDir {
				icon = "📁"
				size = "  <DIR>"
			}

			state.DisplayLines[i] = fmt.Sprintf("%s%s %-25s %s  %s",
				prefix, icon, truncate(e.Name, 25), size, e.ModTime)
		} else {
			state.DisplayLines[i] = ""
		}
	}
}

func updatePreview(state *State) {
	for i := range state.PreviewLines {
		state.PreviewLines[i] = ""
	}

	if len(state.Entries) == 0 || state.SelectedIdx >= len(state.Entries) {
		return
	}

	entry := state.Entries[state.SelectedIdx]
	if entry.IsDir {
		// Preview directory contents
		path := filepath.Join(state.CurrentPath, entry.Name)
		entries, err := os.ReadDir(path)
		if err != nil {
			state.PreviewLines[0] = "(cannot read)"
			return
		}
		state.PreviewLines[0] = fmt.Sprintf("Directory: %d items", len(entries))
		count := 0
		for i, e := range entries {
			if i >= 4 {
				state.PreviewLines[4] = "..."
				break
			}
			prefix := "  "
			if e.IsDir() {
				prefix = "📁"
			}
			state.PreviewLines[i+1] = fmt.Sprintf("  %s %s", prefix, e.Name())
			count++
		}
	} else {
		// Preview file contents (first few lines)
		path := filepath.Join(state.CurrentPath, entry.Name)
		content, err := os.ReadFile(path)
		if err != nil {
			state.PreviewLines[0] = "(cannot read)"
			return
		}

		// Check if binary
		if isBinary(content) {
			state.PreviewLines[0] = fmt.Sprintf("Binary file (%s)", formatSize(entry.Size))
			return
		}

		lines := strings.Split(string(content), "\n")
		for i := 0; i < 5 && i < len(lines); i++ {
			line := lines[i]
			if len(line) > 50 {
				line = line[:47] + "..."
			}
			state.PreviewLines[i] = line
		}
	}
}

func isBinary(data []byte) bool {
	// Check first 512 bytes for null bytes
	limit := 512
	if len(data) < limit {
		limit = len(data)
	}
	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "~"
}

func buildView(state *State) any {
	return VBoxNode{Children: []any{
		// Header
		TextNode{Content: "File Browser", Style: Style{Attr: AttrBold}},
		HBoxNode{Children: []any{
			TextNode{Content: "Path: "},
			TextNode{Content: &state.CurrentPath},
		}},
		TextNode{Content: ""},

		// File list
		TextNode{Content: &state.DisplayLines[0]},
		TextNode{Content: &state.DisplayLines[1]},
		TextNode{Content: &state.DisplayLines[2]},
		TextNode{Content: &state.DisplayLines[3]},
		TextNode{Content: &state.DisplayLines[4]},
		TextNode{Content: &state.DisplayLines[5]},
		TextNode{Content: &state.DisplayLines[6]},
		TextNode{Content: &state.DisplayLines[7]},
		TextNode{Content: &state.DisplayLines[8]},
		TextNode{Content: &state.DisplayLines[9]},
		TextNode{Content: &state.DisplayLines[10]},
		TextNode{Content: &state.DisplayLines[11]},
		TextNode{Content: &state.DisplayLines[12]},
		TextNode{Content: &state.DisplayLines[13]},
		TextNode{Content: &state.DisplayLines[14]},
		TextNode{Content: &state.DisplayLines[15]},
		TextNode{Content: &state.DisplayLines[16]},
		TextNode{Content: &state.DisplayLines[17]},

		TextNode{Content: ""},
		TextNode{Content: "Preview:"},
		TextNode{Content: &state.PreviewLines[0]},
		TextNode{Content: &state.PreviewLines[1]},
		TextNode{Content: &state.PreviewLines[2]},
		TextNode{Content: &state.PreviewLines[3]},
		TextNode{Content: &state.PreviewLines[4]},

		// Status
		TextNode{Content: ""},
		TextNode{Content: &state.StatusLine},
	}}
}
