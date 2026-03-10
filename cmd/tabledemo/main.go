// tabledemo: Demonstrates AutoTable - automatic table from struct slice
package main

import (
	"log"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

type Stock struct {
	Symbol string
	Name   string
	Price  float64
	Change float64
	Volume int
	Buy    bool
}

type Person struct {
	Name    string
	Age     int
	City    string
	Country string
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	stocks := []Stock{
		{"AAPL", "Apple Inc", 178.92, 2.34, 52_000_000, true},
		{"GOOGL", "Alphabet", 141.23, -1.56, 28_000_000, true},
		{"MSFT", "Microsoft", 378.45, 5.12, 31_000_000, false},
		{"TSLA", "Tesla", 248.67, -8.90, 95_000_000, false},
		{"NVDA", "NVIDIA", 721.34, 12.45, 45_000_000, false},
		{"AMD", "AMD", 156.78, 3.21, 62_000_000, false},
	}

	people := []Person{
		{"Alice", 30, "New York", "USA"},
		{"Bob", 25, "London", "UK"},
		{"Charlie", 35, "Tokyo", "Japan"},
		{"Diana", 28, "Paris", "France"},
		{"Eve", 32, "Sydney", "Australia"},
	}

	title := Style{FG: Yellow, Attr: AttrBold}.MarginTRBL(2, 0, 0, 0)

	app.SetView(VBox.Gap(0)(
		Text("AutoTable Demo").FG(Cyan).Bold(),
		HRule().Style(Style{FG: PaletteColor(238)}),

		Text("Stocks (with column formatting):").Style(title),
		AutoTable(stocks).
			Column("Price", Currency("$", 2)).
			Column("Change", PercentChange(1)).
			Column("Volume", Number(0)).
			Column("Buy", Bool("✓", "✗")).
			HeaderStyle(Style{FG: Cyan, Attr: AttrBold}).
			AltRowStyle(Style{BG: PaletteColor(235)}),

		Text("People (selected columns with custom headers):").Style(title),
		AutoTable(people).
			Columns("Name", "Age", "City").
			Headers("Person", "Years", "Location").
			HeaderStyle(Style{FG: Green, Attr: AttrBold}),

		Text("Stocks (just Symbol and Price):").FG(Yellow).Style(title),
		AutoTable(stocks).
			Columns("Symbol", "Price").
			Column("Price", Currency("$", 2)).
			Gap(3),

		HRule().Style(Style{FG: PaletteColor(238)}),
		Text("q: quit").FG(PaletteColor(245)),
	))

	app.Handle("q", func(_ riffkey.Match) { app.Stop() })
	app.Handle("<Escape>", func(_ riffkey.Match) { app.Stop() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
