// forme-files: File browser with reactive List and file preview
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kungfusheep/riffkey"
	. "github.com/kungfusheep/forme"
)

type FileEntry struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime string
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
	statusLine := "↑/↓ navigate, Enter=open, Backspace=up, q=quit"
	previewLines := [5]string{}
	var entries []FileEntry

	updatePreview := func() {
		for i := range previewLines {
			previewLines[i] = ""
		}
		if len(entries) == 0 || selectedIdx >= len(entries) {
			return
		}
		entry := entries[selectedIdx]
		if entry.IsDir {
			path := filepath.Join(currentPath, entry.Name)
			dirEntries, err := os.ReadDir(path)
			if err != nil {
				previewLines[0] = "(cannot read)"
				return
			}
			previewLines[0] = fmt.Sprintf("Directory: %d items", len(dirEntries))
			for i, e := range dirEntries {
				if i >= 4 {
					previewLines[4] = "..."
					break
				}
				prefix := "  "
				if e.IsDir() {
					prefix = "📁"
				}
				previewLines[i+1] = fmt.Sprintf("  %s %s", prefix, e.Name())
			}
		} else {
			path := filepath.Join(currentPath, entry.Name)
			content, err := os.ReadFile(path)
			if err != nil {
				previewLines[0] = "(cannot read)"
				return
			}
			if isBinary(content) {
				previewLines[0] = fmt.Sprintf("Binary file (%s)", formatSize(entry.Size))
				return
			}
			lines := strings.Split(string(content), "\n")
			for i := 0; i < 5 && i < len(lines); i++ {
				line := lines[i]
				if len(line) > 50 {
					line = line[:47] + "..."
				}
				previewLines[i] = line
			}
		}
	}

	scan := func() {
		dirEntries, err := os.ReadDir(currentPath)
		if err != nil {
			statusLine = "Error: " + err.Error()
			entries = nil
			return
		}

		entries = entries[:0]
		for _, e := range dirEntries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			entry := FileEntry{
				Name:    e.Name(),
				IsDir:   e.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime().Format("Jan 02 15:04"),
			}
			icon := "  "
			size := formatSize(entry.Size)
			if entry.IsDir {
				icon = "📁"
				size = "  <DIR>"
			}
			entry.Display = fmt.Sprintf("%s %-25s %s  %s",
				icon, truncate(entry.Name, 25), size, entry.ModTime)
			entries = append(entries, entry)
		}

		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir != entries[j].IsDir {
				return entries[i].IsDir
			}
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})

		updatePreview()
	}
	scan()

	fileList := List(&entries).
		Selection(&selectedIdx).
		MaxVisible(18).
		Render(func(e *FileEntry) any {
			return Text(&e.Display)
		}).
		Handle("<Enter>", func(e *FileEntry) {
			if e.IsDir {
				currentPath = filepath.Join(currentPath, e.Name)
				selectedIdx = 0
				scan()
			}
		})

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(VBox(
		Text("File Browser").Bold(),
		HBox(Text("Path: "), Text(&currentPath)),
		Text(""),
		fileList,
		Text(""),
		Text("Preview:"),
		Text(&previewLines[0]),
		Text(&previewLines[1]),
		Text(&previewLines[2]),
		Text(&previewLines[3]),
		Text(&previewLines[4]),
		Text(""),
		Text(&statusLine),
	))

	onNav := func() { updatePreview() }
	app.Handle("<Up>", func(_ riffkey.Match) { fileList.Up(nil); onNav() })
	app.Handle("<Down>", func(_ riffkey.Match) { fileList.Down(nil); onNav() })
	app.Handle("<PageUp>", func(_ riffkey.Match) { fileList.PageUp(nil); onNav() })
	app.Handle("<PageDown>", func(_ riffkey.Match) { fileList.PageDown(nil); onNav() })
	app.Handle("<Home>", func(_ riffkey.Match) { fileList.First(nil); onNav() })
	app.Handle("<End>", func(_ riffkey.Match) { fileList.Last(nil); onNav() })

	app.Handle("<BS>", func(_ riffkey.Match) {
		parent := filepath.Dir(currentPath)
		if parent != currentPath {
			oldDir := filepath.Base(currentPath)
			currentPath = parent
			selectedIdx = 0
			scan()
			for i, e := range entries {
				if e.Name == oldDir {
					fileList.SetIndex(i)
					break
				}
			}
		}
	})

	app.Handle("q", func(_ riffkey.Match) { app.Stop() })
	app.Handle("<Esc>", func(_ riffkey.Match) { app.Stop() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
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

func isBinary(data []byte) bool {
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
