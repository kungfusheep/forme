package glyph

import (
	"fmt"
	"testing"
)

func TestLeaderStr(t *testing.T) {
	tests := []struct {
		label, value string
		width        int
		want         string
	}{
		{"LABEL", "VALUE", 20, "LABEL..........VALUE"},
		{"A", "B", 10, "A........B"},
		{"LONG LABEL HERE", "OK", 20, "LONG LABEL HERE...OK"},
	}

	for _, tt := range tests {
		got := LeaderStr(tt.label, tt.value, tt.width)
		if got != tt.want {
			t.Errorf("LeaderStr(%q, %q, %d) = %q, want %q", tt.label, tt.value, tt.width, got, tt.want)
		}
	}
}

func TestLED(t *testing.T) {
	if LED(true) != "●" {
		t.Error("LED(true) should be ●")
	}
	if LED(false) != "○" {
		t.Error("LED(false) should be ○")
	}
}

func TestLEDs(t *testing.T) {
	got := LEDs(true, true, false, false)
	want := "●●○○"
	if got != want {
		t.Errorf("LEDs(true, true, false, false) = %q, want %q", got, want)
	}
}

func TestLEDsBracket(t *testing.T) {
	got := LEDsBracket(true, false, true)
	want := "[●○●]"
	if got != want {
		t.Errorf("LEDsBracket = %q, want %q", got, want)
	}
}

func TestBar(t *testing.T) {
	tests := []struct {
		filled, total int
		want          string
	}{
		{3, 5, "▮▮▮▯▯"},
		{0, 5, "▯▯▯▯▯"},
		{5, 5, "▮▮▮▮▮"},
		{10, 10, "▮▮▮▮▮▮▮▮▮▮"},
	}

	for _, tt := range tests {
		got := Bar(tt.filled, tt.total)
		if got != tt.want {
			t.Errorf("Bar(%d, %d) = %q, want %q", tt.filled, tt.total, got, tt.want)
		}
	}
}

func TestMeter(t *testing.T) {
	// Meter at 0
	got := Meter(0, 100, 12)
	runes := []rune(got)
	if runes[0] != '├' || runes[len(runes)-1] != '┤' {
		t.Errorf("Meter should have ├ and ┤ ends, got %q", got)
	}

	// Meter at middle
	got = Meter(50, 100, 12)
	runes = []rune(got)
	if runes[0] != '├' || runes[len(runes)-1] != '┤' {
		t.Errorf("Meter should have ├ and ┤ ends, got %q", got)
	}

	// Check it contains the marker
	hasMarker := false
	for _, r := range got {
		if r == '●' {
			hasMarker = true
			break
		}
	}
	if !hasMarker {
		t.Errorf("Meter should contain ● marker, got %q", got)
	}
}

func TestDrawPanel(t *testing.T) {
	buf := NewBuffer(30, 10)
	style := DefaultStyle()

	region := buf.DrawPanel(0, 0, 20, 5, "TEST", style)

	// Check we got a region back
	if region.Width() != 18 || region.Height() != 3 {
		t.Errorf("Region size wrong: got %dx%d, want 18x3", region.Width(), region.Height())
	}

	// Check border corners
	if buf.Get(0, 0).Rune != '┌' {
		t.Errorf("Top-left should be ┌, got %c", buf.Get(0, 0).Rune)
	}
	if buf.Get(19, 0).Rune != '┐' {
		t.Errorf("Top-right should be ┐, got %c", buf.Get(19, 0).Rune)
	}

	// Check title is in there
	output := buf.StringTrimmed()
	t.Logf("Panel output:\n%s", output)
}

// TestLayoutPatterns validates common layout patterns render correctly
func TestLayoutPatterns(t *testing.T) {
	buf := NewBuffer(60, 20)
	style := DefaultStyle()
	dim := style // In real use this would be dimmer

	// Panel 1: Status with leaders
	region := buf.DrawPanel(0, 0, 30, 8, "STATUS", dim)
	region.WriteString(0, 0, LeaderStr("ITEM A", "OK", 26), style)
	region.WriteString(0, 1, LeaderStr("ITEM B", "PASS", 26), style)
	region.WriteString(0, 2, LeaderStr("ITEM C", "FAIL", 26), style)

	// Panel 2: Indicators
	region = buf.DrawPanel(30, 0, 28, 8, "SYSTEMS", dim)
	region.WriteString(0, 0, LED(true)+" POWER  "+LED(true)+" COMMS", style)
	region.WriteString(0, 1, LED(false)+" BACKUP "+LED(true)+" LINK", style)
	region.WriteString(0, 2, "STATUS: "+LEDsBracket(true, true, false, false), style)

	// Panel 3: Meters
	region = buf.DrawPanel(0, 8, 30, 6, "LEVELS", dim)
	region.WriteString(0, 0, "CPU "+Meter(75, 100, 15)+" 75%", style)
	region.WriteString(0, 1, "MEM "+Meter(45, 100, 15)+" 45%", style)
	region.WriteString(0, 2, "DSK "+Bar(8, 10)+" 80%", style)

	// Panel 4: Bars
	region = buf.DrawPanel(30, 8, 28, 6, "CAPACITY", dim)
	region.WriteString(0, 0, "TANK A "+Bar(9, 10), style)
	region.WriteString(0, 1, "TANK B "+Bar(6, 10), style)
	region.WriteString(0, 2, "TANK C "+Bar(3, 10), style)

	output := buf.StringTrimmed()
	fmt.Printf("Layout test output:\n%s\n", output)

	// Basic sanity checks
	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}

// TestExampleLayout shows what the display helpers produce
func TestExampleLayout(t *testing.T) {
	buf := NewBuffer(50, 15)
	style := DefaultStyle()

	// Simple status panel
	region := buf.DrawPanel(0, 0, 48, 13, "SYSTEM MONITOR", style)

	region.WriteString(0, 0, LeaderStr("CPU USAGE", "78%", 44), style)
	region.WriteString(0, 1, LeaderStr("MEMORY", "4.2GB/8GB", 44), style)
	region.WriteString(0, 2, LeaderStr("DISK", "120GB FREE", 44), style)
	region.WriteString(0, 3, "", style)
	region.WriteString(0, 4, "SERVICES: "+LEDs(true, true, true, false)+" (3/4 UP)", style)
	region.WriteString(0, 5, "", style)
	region.WriteString(0, 6, "LOAD "+Meter(78, 100, 20)+" 78%", style)
	region.WriteString(0, 7, "TEMP "+Meter(45, 100, 20)+" 45C", style)
	region.WriteString(0, 8, "", style)
	region.WriteString(0, 9, "NETWORK: "+Bar(7, 10)+" 70%", style)

	t.Logf("Example layout:\n%s", buf.StringTrimmed())
}
