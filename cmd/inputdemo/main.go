package main

import (
	"fmt"
	"log"
	"strings"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

func main() {
	status := "Tab: next | Shift+Tab: prev | Enter: submit | Esc: quit"

	name := Input().Placeholder("Enter your name").Width(30)
	email := Input().Placeholder("you@example.com").Width(30)
	password := Input().Placeholder("Enter password").Width(30).Mask('*')

	form := Form.Gap(1).LabelFG(BrightWhite)(
		Field("Name", name),
		Field("Email", email),
		Field("Password", password),
	)

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	app.SetView(
		VBox(
			Text("Registration Form").FG(Cyan).Bold(),
			HRule().FG(BrightBlack),
			SpaceH(1),
			form,
			SpaceH(1),
			HRule().FG(BrightBlack),
			Text(&status).FG(BrightBlack),
		),
	)

	app.Handle("<Enter>", func(_ riffkey.Match) {
		n, e, p := name.Value(), email.Value(), password.Value()
		if n == "" || e == "" || p == "" {
			status = "Please fill in all fields"
		} else {
			status = fmt.Sprintf("Submitted! Name=%s, Email=%s, Password=%s",
				n, e, strings.Repeat("*", len(p)))
		}
	})
	app.Handle("<Escape>", func(_ riffkey.Match) {
		app.Stop()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final values: name=%q, email=%q, password=%q\n",
		name.Value(), email.Value(), password.Value())
}
