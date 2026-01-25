// tui-fzf: Fuzzy finder demo
// Tests: text input, filtering, selection, scrolling
//
// GAPS FOUND:
// 1. No TextInput component - must handle each char key individually
// 2. No dynamic list with selection styling - workaround: fixed slots with pointers
// 3. ForEach can't do index-based conditional styling (e.g., highlight selected row)
// 4. No cursor rendering for text input fields
package main

import (
	"fmt"
	"log"
	"strings"

	"riffkey"
	. "tui"
)

const maxVisible = 15 // Max items to show

// Sample data - programming languages
var allItems = []string{
	"Go", "Rust", "Python", "JavaScript", "TypeScript",
	"Ruby", "Java", "C", "C++", "C#",
	"Swift", "Kotlin", "Scala", "Haskell", "Erlang",
	"Elixir", "Clojure", "F#", "OCaml", "Lua",
	"Perl", "PHP", "R", "Julia", "Dart",
	"Zig", "Nim", "Crystal", "V", "Odin",
}

type State struct {
	Query         string
	FilteredItems []string
	SelectedIdx   int
	StatusLine    string

	// Display slots - workaround for dynamic list
	// GAP: Can't have truly dynamic lists with selection
	DisplayLines [maxVisible]string
}

func main() {
	state := &State{
		Query:         "",
		FilteredItems: allItems,
		StatusLine:    "Type to filter, ↑/↓ select, Enter confirm, Esc quit",
	}
	updateDisplayLines(state)

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(buildView(state))

	// Key handlers
	app.Handle("<Up>", func(_ riffkey.Match) {
		if state.SelectedIdx > 0 {
			state.SelectedIdx--
			updateDisplayLines(state)
		}
		app.RequestRender()
	})

	app.Handle("<Down>", func(_ riffkey.Match) {
		if state.SelectedIdx < len(state.FilteredItems)-1 {
			state.SelectedIdx++
			updateDisplayLines(state)
		}
		app.RequestRender()
	})

	app.Handle("<Esc>", func(_ riffkey.Match) {
		app.Stop()
	})

	app.Handle("<Enter>", func(_ riffkey.Match) {
		if len(state.FilteredItems) > 0 {
			state.StatusLine = "Selected: " + state.FilteredItems[state.SelectedIdx]
		}
		app.RequestRender()
	})

	app.Handle("<BS>", func(_ riffkey.Match) {
		if len(state.Query) > 0 {
			state.Query = state.Query[:len(state.Query)-1]
			filterItems(state)
			updateDisplayLines(state)
		}
		app.RequestRender()
	})

	// Handle printable characters for text input
	// GAP: No built-in text input component!
	for _, char := range "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_" {
		c := string(char)
		app.Handle(c, func(_ riffkey.Match) {
			state.Query += c
			filterItems(state)
			updateDisplayLines(state)
			app.RequestRender()
		})
	}

	// Space is a special key in riffkey
	app.Handle("<Space>", func(_ riffkey.Match) {
		state.Query += " "
		filterItems(state)
		updateDisplayLines(state)
		app.RequestRender()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func updateDisplayLines(state *State) {
	for i := 0; i < maxVisible; i++ {
		if i < len(state.FilteredItems) {
			prefix := "  "
			if i == state.SelectedIdx {
				prefix = "> "
			}
			state.DisplayLines[i] = fmt.Sprintf("%s%-20s", prefix, state.FilteredItems[i])
		} else {
			state.DisplayLines[i] = ""
		}
	}
}

func filterItems(state *State) {
	if state.Query == "" {
		state.FilteredItems = allItems
	} else {
		state.FilteredItems = nil
		q := strings.ToLower(state.Query)
		for _, item := range allItems {
			if strings.Contains(strings.ToLower(item), q) {
				state.FilteredItems = append(state.FilteredItems, item)
			}
		}
	}
	// Reset selection if out of bounds
	if state.SelectedIdx >= len(state.FilteredItems) {
		state.SelectedIdx = len(state.FilteredItems) - 1
	}
	if state.SelectedIdx < 0 {
		state.SelectedIdx = 0
	}
}

func buildView(state *State) any {
	// Use fixed slots with pointer bindings
	// This works because DisplayLines is updated when selection changes
	return VBoxNode{Children: []any{
		// Header
		TextNode{Content: "Fuzzy Finder", Style: Style{Attr: AttrBold}},
		TextNode{Content: ""},

		// Query input - GAP: No text input component!
		HBoxNode{Children: []any{
			TextNode{Content: "> "},
			TextNode{Content: &state.Query},
			TextNode{Content: "_"}, // Cursor
		}},
		TextNode{Content: ""},

		// Results - fixed slots bound to pointers
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

		// Status
		TextNode{Content: ""},
		TextNode{Content: &state.StatusLine},
	}}
}
