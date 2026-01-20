package main

import (
	"fmt"
	"log"
	"strings"

	"riffkey"
	"tui"
)

func main() {
	// Form fields - using Field struct for cleaner state management
	name := tui.Field{}
	email := tui.Field{}
	password := tui.Field{}

	// Single focus tracker for all fields
	focus := tui.FocusGroup{}

	// Status line
	status := "Tab: next | Shift+Tab: prev | Enter: submit | Esc: quit"

	app, err := tui.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Build view once - all updates via pointers
	app.SetView(
		tui.VBox{
			Children: []any{
				tui.Text{Content: "Registration Form", Style: tui.Style{FG: tui.Cyan, Attr: tui.AttrBold}},
				tui.HRule{Style: tui.Style{FG: tui.BrightBlack}},
				tui.Spacer{Height: 1},

				// Name field
				tui.HBox{
					Children: []any{
						tui.Text{Content: "    Name: "},
						tui.TextInput{
							Field:       &name,
							FocusGroup:  &focus,
							FocusIndex:  0,
							Placeholder: "Enter your name",
							Width:       30,
						},
					},
				},
				tui.Spacer{Height: 1},

				// Email field
				tui.HBox{
					Children: []any{
						tui.Text{Content: "   Email: "},
						tui.TextInput{
							Field:       &email,
							FocusGroup:  &focus,
							FocusIndex:  1,
							Placeholder: "you@example.com",
							Width:       30,
						},
					},
				},
				tui.Spacer{Height: 1},

				// Password field (masked)
				tui.HBox{
					Children: []any{
						tui.Text{Content: "Password: "},
						tui.TextInput{
							Field:       &password,
							FocusGroup:  &focus,
							FocusIndex:  2,
							Placeholder: "Enter password",
							Width:       30,
							Mask:        '*',
						},
					},
				},

				tui.Spacer{Height: 2},
				tui.HRule{Style: tui.Style{FG: tui.BrightBlack}},
				tui.Text{Content: &status, Style: tui.Style{FG: tui.BrightBlack}},
			},
		},
	)

	// Field handlers - riffkey handles all editing (backspace, delete, arrows, etc.)
	fields := []*tui.Field{&name, &email, &password}
	handlers := make([]*riffkey.TextHandler, len(fields))
	for i, f := range fields {
		handlers[i] = riffkey.NewTextHandler(&f.Value, &f.Cursor)
	}

	// Route text input to focused field
	app.Router().HandleUnmatched(func(k riffkey.Key) bool {
		return handlers[focus.Current].HandleKey(k)
	})

	// Tab to next field
	app.Handle("<Tab>", func(_ riffkey.Match) {
		focus.Current = (focus.Current + 1) % len(fields)
	})

	// Shift+Tab to previous field
	app.Handle("<S-Tab>", func(_ riffkey.Match) {
		focus.Current = (focus.Current + len(fields) - 1) % len(fields)
	})

	// Enter to submit
	app.Handle("<Enter>", func(_ riffkey.Match) {
		if name.Value == "" || email.Value == "" || password.Value == "" {
			status = "Please fill in all fields"
		} else {
			status = fmt.Sprintf("Submitted! Name=%s, Email=%s, Password=%s",
				name.Value, email.Value, strings.Repeat("*", len(password.Value)))
		}
	})

	// Escape to quit
	app.Handle("<Escape>", func(_ riffkey.Match) {
		app.Stop()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final values: name=%q, email=%q, password=%q\n",
		name.Value, email.Value, password.Value)
}
