package main

import (
	"fmt"
	"log"
	"strings"

	"riffkey"
	. "forme"
)

func main() {
	name := Field{}
	email := Field{}
	password := Field{}
	focus := FocusGroup{}
	status := "Tab: next | Shift+Tab: prev | Enter: submit | Esc: quit"

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox(
			Text("Registration Form").FG(Cyan).Bold(),
			HRule().Style(Style{FG: BrightBlack}),
			SpaceH(1),

			HBox(
				Text("    Name: "),
				TextInput{
					Field:       &name,
					FocusGroup:  &focus,
					FocusIndex:  0,
					Placeholder: "Enter your name",
					Width:       30,
				},
			),
			SpaceH(1),

			HBox(
				Text("   Email: "),
				TextInput{
					Field:       &email,
					FocusGroup:  &focus,
					FocusIndex:  1,
					Placeholder: "you@example.com",
					Width:       30,
				},
			),
			SpaceH(1),

			HBox(
				Text("Password: "),
				TextInput{
					Field:       &password,
					FocusGroup:  &focus,
					FocusIndex:  2,
					Placeholder: "Enter password",
					Width:       30,
					Mask:        '*',
				},
			),

			SpaceH(2),
			HRule().Style(Style{FG: BrightBlack}),
			Text(&status).FG(BrightBlack),
		),
	)

	fields := []*Field{&name, &email, &password}
	handlers := make([]*riffkey.TextHandler, len(fields))
	for i, f := range fields {
		handlers[i] = riffkey.NewTextHandler(&f.Value, &f.Cursor)
	}

	app.Router().HandleUnmatched(func(k riffkey.Key) bool {
		return handlers[focus.Current].HandleKey(k)
	})

	app.Handle("<Tab>", func(_ riffkey.Match) {
		focus.Current = (focus.Current + 1) % len(fields)
	})
	app.Handle("<S-Tab>", func(_ riffkey.Match) {
		focus.Current = (focus.Current + len(fields) - 1) % len(fields)
	})
	app.Handle("<Enter>", func(_ riffkey.Match) {
		if name.Value == "" || email.Value == "" || password.Value == "" {
			status = "Please fill in all fields"
		} else {
			status = fmt.Sprintf("Submitted! Name=%s, Email=%s, Password=%s",
				name.Value, email.Value, strings.Repeat("*", len(password.Value)))
		}
	})
	app.Handle("<Escape>", func(_ riffkey.Match) {
		app.Stop()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final values: name=%q, email=%q, password=%q\n",
		name.Value, email.Value, password.Value)
}
