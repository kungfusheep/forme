// forme-fzf: fuzzy finder demo using FilterList
package main

import (
	"fmt"
	"log"

	. "github.com/kungfusheep/forme"
)

var languages = []string{
	"Go", "Rust", "Python", "JavaScript", "TypeScript",
	"Ruby", "Java", "C", "C++", "C#",
	"Swift", "Kotlin", "Scala", "Haskell", "Erlang",
	"Elixir", "Clojure", "F#", "OCaml", "Lua",
	"Perl", "PHP", "R", "Julia", "Dart",
	"Zig", "Nim", "Crystal", "V", "Odin",
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	status := "type to filter, ctrl-n/p nav, enter select, esc quit"

	app.View("main",
		VBox.Border(BorderRounded).Title("fuzzy finder")(
			FilterList(&languages, func(s *string) string { return *s }).
				Placeholder("type to filter...").
				Render(func(s *string) any { return Text(s) }).
				Handle("<Enter>", func(s *string) {
					status = fmt.Sprintf("selected: %s", *s)
				}).
				HandleClear("<Esc>", app.Stop),
			Space(),
			HRule(),
			Text(&status).Dim(),
		),
	).Handle("q", app.Stop)

	if err := app.RunFrom("main"); err != nil {
		log.Fatal(err)
	}
}
