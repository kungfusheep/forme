// tui-ncdu: Disk usage viewer demo
// Tests: filesystem data, tree navigation, progress bars, scrolling
//
// GAPS FOUND:
// 1. No Progress bar component - can't show visual size bars inline
// 2. Can't combine Text + Progress on same Row easily
// 3. Same fixed slots workaround as fzf for dynamic lists
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"riffkey"
	"tui"
)

const maxVisible = 20

type Entry struct {
	Name  string
	Size  int64
	IsDir bool
}

type State struct {
	CurrentPath   string
	Entries       []Entry
	SelectedIdx   int
	TotalSize     int64
	StatusLine    string
	DisplayLines  [maxVisible]string
	DisplayBars   [maxVisible]int // percentage for progress bars
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
		StatusLine:  "↑/↓ navigate, Enter=open dir, Backspace=parent, q=quit",
	}
	scanDirectory(state)

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(buildView(state))

	app.Handle("<Up>", func(_ riffkey.Match) {
		if state.SelectedIdx > 0 {
			state.SelectedIdx--
			updateDisplay(state)
		}
		app.RequestRender()
	})

	app.Handle("<Down>", func(_ riffkey.Match) {
		if state.SelectedIdx < len(state.Entries)-1 {
			state.SelectedIdx++
			updateDisplay(state)
		}
		app.RequestRender()
	})

	app.Handle("<Enter>", func(_ riffkey.Match) {
		if len(state.Entries) > 0 && state.Entries[state.SelectedIdx].IsDir {
			state.CurrentPath = filepath.Join(state.CurrentPath, state.Entries[state.SelectedIdx].Name)
			state.SelectedIdx = 0
			scanDirectory(state)
		}
		app.RequestRender()
	})

	app.Handle("<BS>", func(_ riffkey.Match) {
		parent := filepath.Dir(state.CurrentPath)
		if parent != state.CurrentPath {
			state.CurrentPath = parent
			state.SelectedIdx = 0
			scanDirectory(state)
		}
		app.RequestRender()
	})

	app.Handle("q", func(_ riffkey.Match) {
		app.Stop()
	})

	app.Handle("<Esc>", func(_ riffkey.Match) {
		app.Stop()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func scanDirectory(state *State) {
	entries, err := os.ReadDir(state.CurrentPath)
	if err != nil {
		state.StatusLine = "Error: " + err.Error()
		state.Entries = nil
		updateDisplay(state)
		return
	}

	state.Entries = nil
	state.TotalSize = 0

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		entry := Entry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
		}

		if e.IsDir() {
			// Quick size estimate - just count immediate children
			// Full recursive would be slow
			entry.Size = getDirSize(filepath.Join(state.CurrentPath, e.Name()))
		} else {
			entry.Size = info.Size()
		}

		state.Entries = append(state.Entries, entry)
		state.TotalSize += entry.Size
	}

	// Sort by size descending
	sort.Slice(state.Entries, func(i, j int) bool {
		return state.Entries[i].Size > state.Entries[j].Size
	})

	updateDisplay(state)
}

func getDirSize(path string) int64 {
	var size int64
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !e.IsDir() {
			size += info.Size()
		}
	}
	return size
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%6.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%6.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%6.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%6d B ", bytes)
	}
}

func updateDisplay(state *State) {
	for i := 0; i < maxVisible; i++ {
		if i < len(state.Entries) {
			e := state.Entries[i]
			prefix := "  "
			if i == state.SelectedIdx {
				prefix = "> "
			}
			icon := "  "
			if e.IsDir {
				icon = "📁"
			}
			state.DisplayLines[i] = fmt.Sprintf("%s%s %s  %-30s", prefix, formatSize(e.Size), icon, e.Name)

			// Calculate bar percentage
			if state.TotalSize > 0 {
				state.DisplayBars[i] = int(e.Size * 100 / state.TotalSize)
			} else {
				state.DisplayBars[i] = 0
			}
		} else {
			state.DisplayLines[i] = ""
			state.DisplayBars[i] = 0
		}
	}
}

func buildView(state *State) any {
	return tui.Col{Children: []any{
		// Header
		tui.Text{Content: "Disk Usage Analyzer", Style: tui.Style{Attr: tui.AttrBold}},
		tui.Row{Children: []any{
			tui.Text{Content: "Path: "},
			tui.Text{Content: &state.CurrentPath},
		}},
		tui.Text{Content: ""},

		// Entries - using fixed slots
		tui.Text{Content: &state.DisplayLines[0]},
		tui.Text{Content: &state.DisplayLines[1]},
		tui.Text{Content: &state.DisplayLines[2]},
		tui.Text{Content: &state.DisplayLines[3]},
		tui.Text{Content: &state.DisplayLines[4]},
		tui.Text{Content: &state.DisplayLines[5]},
		tui.Text{Content: &state.DisplayLines[6]},
		tui.Text{Content: &state.DisplayLines[7]},
		tui.Text{Content: &state.DisplayLines[8]},
		tui.Text{Content: &state.DisplayLines[9]},
		tui.Text{Content: &state.DisplayLines[10]},
		tui.Text{Content: &state.DisplayLines[11]},
		tui.Text{Content: &state.DisplayLines[12]},
		tui.Text{Content: &state.DisplayLines[13]},
		tui.Text{Content: &state.DisplayLines[14]},
		tui.Text{Content: &state.DisplayLines[15]},
		tui.Text{Content: &state.DisplayLines[16]},
		tui.Text{Content: &state.DisplayLines[17]},
		tui.Text{Content: &state.DisplayLines[18]},
		tui.Text{Content: &state.DisplayLines[19]},

		// Status
		tui.Text{Content: ""},
		tui.Text{Content: &state.StatusLine},
	}}
}
