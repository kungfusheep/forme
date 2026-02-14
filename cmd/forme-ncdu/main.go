// forme-ncdu: Disk usage viewer with reactive List and inline formatting
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/kungfusheep/riffkey"
	. "github.com/kungfusheep/forme"
)

type Entry struct {
	Name    string
	Size    int64
	IsDir   bool
	Display string
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

	currentPath := absPath
	selectedIdx := 0
	statusLine := "↑/↓ navigate, Enter=open dir, Backspace=parent, q=quit"
	var entries []Entry

	scan := func() {
		dirEntries, err := os.ReadDir(currentPath)
		if err != nil {
			statusLine = "Error: " + err.Error()
			entries = nil
			return
		}

		entries = entries[:0]
		var totalSize int64

		for _, e := range dirEntries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			entry := Entry{Name: e.Name(), IsDir: e.IsDir()}
			if e.IsDir() {
				entry.Size = getDirSize(filepath.Join(currentPath, e.Name()))
			} else {
				entry.Size = info.Size()
			}
			entries = append(entries, entry)
			totalSize += entry.Size
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Size > entries[j].Size
		})

		// format display strings
		for i := range entries {
			e := &entries[i]
			icon := "  "
			if e.IsDir {
				icon = "📁"
			}
			pct := ""
			if totalSize > 0 {
				pct = fmt.Sprintf("%3d%%", e.Size*100/totalSize)
			}
			e.Display = fmt.Sprintf("%s %s %s  %s", formatSize(e.Size), pct, icon, e.Name)
		}
	}
	scan()

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(VBox(
		Text("Disk Usage Analyzer").Bold(),
		HBox(Text("Path: "), Text(&currentPath)),
		Text(""),
		List(&entries).
			Selection(&selectedIdx).
			MaxVisible(20).
			Render(func(e *Entry) any {
				return Text(&e.Display)
			}).
			BindNav("<Down>", "<Up>").
			BindPageNav("<PageDown>", "<PageUp>").
			BindFirstLast("<Home>", "<End>").
			Handle("<Enter>", func(e *Entry) {
				if e.IsDir {
					currentPath = filepath.Join(currentPath, e.Name)
					selectedIdx = 0
					scan()
				}
			}),
		Text(""),
		Text(&statusLine),
	))

	app.Handle("<BS>", func(_ riffkey.Match) {
		parent := filepath.Dir(currentPath)
		if parent != currentPath {
			currentPath = parent
			selectedIdx = 0
			scan()
		}
	})

	app.Handle("q", func(_ riffkey.Match) { app.Stop() })
	app.Handle("<Esc>", func(_ riffkey.Match) { app.Stop() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
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
