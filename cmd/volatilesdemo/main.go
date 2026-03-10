// volatilesdemo: High-frequency updating financial data grid
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

type Market struct {
	Symbol    string
	Name      string
	Price     string
	Change    string
	ChangePct string
	Bid       string
	Ask       string
	Spread    string
	Volume    string
	High      string
	Low       string
	LastTrade string
	Trend     string // ▲ ▼ ●

	// internal state
	priceVal   float64
	changeVal  float64
	bidVal     float64
	askVal     float64
	volumeVal  int
	highVal    float64
	lowVal     float64
	history    []float64
	historyIdx int
}

func (m *Market) update() {
	// Random walk the price
	delta := (rand.Float64() - 0.5) * m.priceVal * 0.002 // 0.2% max move
	m.priceVal += delta
	if m.priceVal < 0.01 {
		m.priceVal = 0.01
	}

	// Update change from open
	m.changeVal += delta
	changePct := (m.changeVal / (m.priceVal - m.changeVal)) * 100

	// Bid/ask spread
	spreadPct := 0.0005 + rand.Float64()*0.001 // 0.05-0.15% spread
	m.bidVal = m.priceVal * (1 - spreadPct/2)
	m.askVal = m.priceVal * (1 + spreadPct/2)

	// Volume tick
	m.volumeVal += rand.Intn(1000)

	// Track high/low
	if m.priceVal > m.highVal {
		m.highVal = m.priceVal
	}
	if m.priceVal < m.lowVal {
		m.lowVal = m.priceVal
	}

	// History for sparkline
	m.history[m.historyIdx] = m.priceVal
	m.historyIdx = (m.historyIdx + 1) % len(m.history)

	// Update display strings
	m.Price = formatPrice(m.priceVal)
	m.Bid = formatPrice(m.bidVal)
	m.Ask = formatPrice(m.askVal)
	m.Spread = fmt.Sprintf("%.2f", (m.askVal-m.bidVal)*100)
	m.Volume = formatVolume(m.volumeVal)
	m.High = formatPrice(m.highVal)
	m.Low = formatPrice(m.lowVal)
	m.LastTrade = time.Now().Format("15:04:05")

	if m.changeVal > 0 {
		m.Change = fmt.Sprintf("+%.2f", m.changeVal)
		m.ChangePct = fmt.Sprintf("+%.2f%%", changePct)
		m.Trend = "▲"
	} else if m.changeVal < 0 {
		m.Change = fmt.Sprintf("%.2f", m.changeVal)
		m.ChangePct = fmt.Sprintf("%.2f%%", changePct)
		m.Trend = "▼"
	} else {
		m.Change = "0.00"
		m.ChangePct = "0.00%"
		m.Trend = "●"
	}
}

func (m *Market) sparkline() []float64 {
	// Return history in order (oldest to newest)
	result := make([]float64, len(m.history))
	for i := 0; i < len(m.history); i++ {
		idx := (m.historyIdx + i) % len(m.history)
		result[i] = m.history[idx]
	}
	return result
}

func formatPrice(p float64) string {
	if p >= 1000 {
		return fmt.Sprintf("%.2f", p)
	} else if p >= 1 {
		return fmt.Sprintf("%.4f", p)
	}
	return fmt.Sprintf("%.6f", p)
}

func formatVolume(v int) string {
	if v >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(v)/1_000_000)
	} else if v >= 1000 {
		return fmt.Sprintf("%.1fK", float64(v)/1000)
	}
	return fmt.Sprintf("%d", v)
}

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize markets
	markets := []Market{
		{Symbol: "BTC/USD", Name: "Bitcoin", priceVal: 67234.50},
		{Symbol: "ETH/USD", Name: "Ethereum", priceVal: 3456.78},
		{Symbol: "AAPL", Name: "Apple Inc", priceVal: 178.92},
		{Symbol: "GOOGL", Name: "Alphabet", priceVal: 141.23},
		{Symbol: "MSFT", Name: "Microsoft", priceVal: 378.45},
		{Symbol: "TSLA", Name: "Tesla", priceVal: 248.67},
		{Symbol: "NVDA", Name: "NVIDIA", priceVal: 721.34},
		{Symbol: "AMD", Name: "AMD", priceVal: 156.78},
		{Symbol: "META", Name: "Meta", priceVal: 485.23},
		{Symbol: "AMZN", Name: "Amazon", priceVal: 178.56},
		{Symbol: "EUR/USD", Name: "Euro", priceVal: 1.0845},
		{Symbol: "GBP/USD", Name: "Pound", priceVal: 1.2634},
		{Symbol: "XAU/USD", Name: "Gold", priceVal: 2034.50},
		{Symbol: "XAG/USD", Name: "Silver", priceVal: 22.87},
		{Symbol: "SOL/USD", Name: "Solana", priceVal: 143.56},
	}

	// Initialize history and state
	for i := range markets {
		markets[i].history = make([]float64, 20)
		markets[i].highVal = markets[i].priceVal
		markets[i].lowVal = markets[i].priceVal
		for j := range markets[i].history {
			markets[i].history[j] = markets[i].priceVal
		}
		markets[i].update()
	}

	// Stats
	var updateCount int64
	var ups string = "0"
	var lastSecond = time.Now().Unix()
	var updatesThisSecond int64
	var mu sync.Mutex

	// Colors
	headerBG := PaletteColor(236)
	rowBG := PaletteColor(234)
	altRowBG := PaletteColor(235)

	// Column widths
	const (
		colSymbol = 10
		colPrice  = 12
		colChange = 9
		colPct    = 8
		colBid    = 11
		colAsk    = 11
		colSpread = 6
		colVolume = 8
		colHigh   = 11
		colLow    = 11
		colTime   = 9
		colSpark  = 20
	)

	// Header row
	header := HBox.CascadeStyle(&Style{BG: headerBG, FG: Cyan, Attr: AttrBold})(
		Text("Symbol").Width(colSymbol),
		Text("Price").Width(colPrice),
		Text("Chg").Width(colChange),
		Text("Chg%").Width(colPct),
		Text("Bid").Width(colBid),
		Text("Ask").Width(colAsk),
		Text("Sprd").Width(colSpread),
		Text("Vol").Width(colVolume),
		Text("High").Width(colHigh),
		Text("Low").Width(colLow),
		Text("Time").Width(colTime),
		Text("Trend").Width(colSpark),
	)

	// Market row with sparkline
	marketRow := func(m *Market, isAlt bool) any {
		bg := rowBG
		if isAlt {
			bg = altRowBG
		}

		// Determine colors based on change
		var changeColor Color
		if m.changeVal > 0 {
			changeColor = RGB(0, 255, 128)
		} else if m.changeVal < 0 {
			changeColor = RGB(255, 80, 80)
		} else {
			changeColor = PaletteColor(245)
		}

		// Sparkline widget
		spark := Widget(
			func(availW int16) (w, h int16) { return colSpark, 1 },
			func(buf *Buffer, x, y, w, h int16) {
				data := m.sparkline()
				if len(data) == 0 {
					return
				}

				// Find min/max for normalization
				minV, maxV := data[0], data[0]
				for _, v := range data {
					if v < minV {
						minV = v
					}
					if v > maxV {
						maxV = v
					}
				}

				chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
				rng := maxV - minV
				if rng < 0.0001 {
					rng = 0.0001
				}

				for i, v := range data {
					if i >= int(w) {
						break
					}
					norm := (v - minV) / rng
					idx := int(norm * float64(len(chars)-1))
					if idx >= len(chars) {
						idx = len(chars) - 1
					}
					if idx < 0 {
						idx = 0
					}

					// Color gradient based on position in range
					color := LerpColor(RGB(255, 100, 100), RGB(100, 255, 100), norm)
					buf.Set(int(x)+i, int(y), Cell{Rune: chars[idx], Style: Style{FG: color, BG: bg}})
				}
			},
		)

		return HBox.CascadeStyle(&Style{BG: bg})(
			Text(&m.Symbol).Width(colSymbol).FG(White).Bold(),
			Text(&m.Price).Width(colPrice).FG(BrightWhite),
			Text(&m.Change).Width(colChange).FG(changeColor),
			Text(&m.ChangePct).Width(colPct).FG(changeColor),
			Text(&m.Bid).Width(colBid).FG(PaletteColor(250)),
			Text(&m.Ask).Width(colAsk).FG(PaletteColor(250)),
			Text(&m.Spread).Width(colSpread).FG(PaletteColor(245)),
			Text(&m.Volume).Width(colVolume).FG(PaletteColor(245)),
			Text(&m.High).Width(colHigh).FG(RGB(100, 255, 100)),
			Text(&m.Low).Width(colLow).FG(RGB(255, 100, 100)),
			Text(&m.LastTrade).Width(colTime).FG(PaletteColor(240)),
			spark,
		)
	}

	// Ticker tape at bottom
	tickerPos := 0
	tickerText := ""
	updateTicker := func() {
		var parts []string
		for _, m := range markets {
			color := "+"
			if m.changeVal < 0 {
				color = "-"
			}
			parts = append(parts, fmt.Sprintf("%s %s (%s%s)", m.Symbol, m.Price, color, m.ChangePct))
		}
		tickerText = "    " + joinWith(parts, "  ●  ") + "    "
	}
	updateTicker()

	tickerWidget := Widget(
		func(availW int16) (w, h int16) { return availW, 1 },
		func(buf *Buffer, x, y, w, h int16) {
			text := tickerText
			textLen := len([]rune(text))

			for i := int16(0); i < w; i++ {
				idx := (tickerPos + int(i)) % textLen
				ch := []rune(text)[idx]

				// Color based on character context
				color := PaletteColor(245)
				if ch == '+' {
					color = RGB(0, 255, 128)
				} else if ch == '-' {
					color = RGB(255, 80, 80)
				} else if ch == '●' {
					color = PaletteColor(240)
				}

				buf.Set(int(x+i), int(y), Cell{Rune: ch, Style: Style{FG: color, BG: PaletteColor(232)}})
			}
		},
	)

	// Build view
	app.SetView(VBox(
		HBox(
			Text(" LIVE MARKETS ").BG(Cyan).FG(Black).Bold(),
			Space(),
			Text(&ups).FG(Green),
			Text(" updates/sec").FG(PaletteColor(245)),
		),
		HRule().Style(Style{FG: PaletteColor(238)}),
		header,
		HRule().Style(Style{FG: PaletteColor(238)}),

		// Market rows - manually laid out to avoid ForEach for this perf demo
		marketRow(&markets[0], false),
		marketRow(&markets[1], true),
		marketRow(&markets[2], false),
		marketRow(&markets[3], true),
		marketRow(&markets[4], false),
		marketRow(&markets[5], true),
		marketRow(&markets[6], false),
		marketRow(&markets[7], true),
		marketRow(&markets[8], false),
		marketRow(&markets[9], true),
		marketRow(&markets[10], false),
		marketRow(&markets[11], true),
		marketRow(&markets[12], false),
		marketRow(&markets[13], true),
		marketRow(&markets[14], false),

		Space(),
		HRule().Style(Style{FG: PaletteColor(238)}),
		tickerWidget,
		HRule().Style(Style{FG: PaletteColor(238)}),
		Text(" q: quit | Data updates ~60Hz | All values simulated").FG(PaletteColor(240)),
	))

	// High-frequency update goroutine
	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60 updates/sec
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()

			// Update 3-5 random markets per tick for realistic feel
			numUpdates := 3 + rand.Intn(3)
			for i := 0; i < numUpdates; i++ {
				idx := rand.Intn(len(markets))
				markets[idx].update()
			}

			// Scroll ticker
			tickerPos++
			if tickerPos >= len([]rune(tickerText)) {
				tickerPos = 0
				updateTicker() // Refresh ticker content
			}

			// Track updates/sec
			updateCount++
			updatesThisSecond++
			now := time.Now().Unix()
			if now != lastSecond {
				ups = fmt.Sprintf("%d", updatesThisSecond)
				updatesThisSecond = 0
				lastSecond = now
			}

			mu.Unlock()
			app.RequestRender()
		}
	}()

	app.Handle("q", func(_ riffkey.Match) { app.Stop() })
	app.Handle("<Escape>", func(_ riffkey.Match) { app.Stop() })

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total updates: %d\n", updateCount)
}

func joinWith(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += sep + p
	}
	return result
}

// hueToRGB converts hue (0-1) to RGB
func hueToRGB(h float64) Color {
	h = math.Mod(h, 1.0)
	if h < 0 {
		h += 1
	}
	var r, g, b float64
	switch {
	case h < 1.0/6.0:
		r, g, b = 1, h*6, 0
	case h < 2.0/6.0:
		r, g, b = 1-(h-1.0/6.0)*6, 1, 0
	case h < 3.0/6.0:
		r, g, b = 0, 1, (h-2.0/6.0)*6
	case h < 4.0/6.0:
		r, g, b = 0, 1-(h-3.0/6.0)*6, 1
	case h < 5.0/6.0:
		r, g, b = (h-4.0/6.0)*6, 0, 1
	default:
		r, g, b = 1, 0, 1-(h-5.0/6.0)*6
	}
	return RGB(uint8(r*255), uint8(g*255), uint8(b*255))
}
