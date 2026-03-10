package glyph

import "testing"

func TestInsertCommas(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "0"},
		{"100", "100"},
		{"1000", "1,000"},
		{"12345", "12,345"},
		{"1234567", "1,234,567"},
		{"1234567890", "1,234,567,890"},
		{"-1234", "-1,234"},
		{"-1234567.89", "-1,234,567.89"},
		{"1000.50", "1,000.50"},
		{"999", "999"},
	}

	for _, tt := range tests {
		got := insertCommas(tt.input)
		if got != tt.want {
			t.Errorf("insertCommas(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		val      any
		decimals int
		want     string
	}{
		{1234.5, 2, "1,234.50"},
		{1234, 0, "1,234"},
		{0.5, 1, "0.5"},
		{-9876.543, 2, "-9,876.54"},
		{int64(1000000), 0, "1,000,000"},
	}

	for _, tt := range tests {
		cfg := &ColumnConfig{}
		Number(tt.decimals)(cfg)
		got := cfg.format(tt.val)
		if got != tt.want {
			t.Errorf("Number(%d)(%v) = %q, want %q", tt.decimals, tt.val, got, tt.want)
		}
	}
}

func TestCurrency(t *testing.T) {
	tests := []struct {
		symbol   string
		decimals int
		val      any
		want     string
	}{
		{"$", 2, 1234.5, "$1,234.50"},
		{"€", 2, 99.9, "€99.90"},
		{"£", 0, 1000, "£1,000"},
	}

	for _, tt := range tests {
		cfg := &ColumnConfig{}
		Currency(tt.symbol, tt.decimals)(cfg)
		got := cfg.format(tt.val)
		if got != tt.want {
			t.Errorf("Currency(%q, %d)(%v) = %q, want %q", tt.symbol, tt.decimals, tt.val, got, tt.want)
		}
		if cfg.align != AlignRight {
			t.Error("Currency should set AlignRight")
		}
	}
}

func TestPercent(t *testing.T) {
	cfg := &ColumnConfig{}
	Percent(1)(cfg)

	got := cfg.format(12.34)
	if got != "12.3%" {
		t.Errorf("Percent(1)(12.34) = %q, want %q", got, "12.3%")
	}
}

func TestPercentChange(t *testing.T) {
	cfg := &ColumnConfig{}
	PercentChange(1)(cfg)

	pos := cfg.format(5.67)
	if pos != "+5.7%" {
		t.Errorf("PercentChange positive = %q, want %q", pos, "+5.7%")
	}

	neg := cfg.format(-3.21)
	if neg != "-3.2%" {
		t.Errorf("PercentChange negative = %q, want %q", neg, "-3.2%")
	}

	// style should return green for positive
	posStyle := cfg.style(5.67)
	if posStyle.FG != Green {
		t.Error("PercentChange positive should be Green")
	}

	negStyle := cfg.style(-3.21)
	if negStyle.FG != Red {
		t.Error("PercentChange negative should be Red")
	}
}

func TestBytes(t *testing.T) {
	tests := []struct {
		val  any
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{float64(1099511627776), "1.0 TB"},
	}

	cfg := &ColumnConfig{}
	Bytes()(cfg)

	for _, tt := range tests {
		got := cfg.format(tt.val)
		if got != tt.want {
			t.Errorf("Bytes()(%v) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestBool(t *testing.T) {
	cfg := &ColumnConfig{}
	Bool("✓", "✗")(cfg)

	if cfg.format(true) != "✓" {
		t.Errorf("Bool true = %q, want ✓", cfg.format(true))
	}
	if cfg.format(false) != "✗" {
		t.Errorf("Bool false = %q, want ✗", cfg.format(false))
	}
	if cfg.align != AlignCenter {
		t.Error("Bool should set AlignCenter")
	}
}

func TestStyleSign(t *testing.T) {
	posStyle := Style{FG: Green}
	negStyle := Style{FG: Red}

	cfg := &ColumnConfig{}
	StyleSign(posStyle, negStyle)(cfg)

	if cfg.style(5.0) != posStyle {
		t.Error("StyleSign(5.0) should return positive style")
	}
	if cfg.style(-3.0) != negStyle {
		t.Error("StyleSign(-3.0) should return negative style")
	}
	if cfg.style(0.0) != posStyle {
		t.Error("StyleSign(0.0) should return positive style")
	}
}

func TestStyleBool(t *testing.T) {
	trueStyle := Style{FG: Green}
	falseStyle := Style{FG: Red}

	cfg := &ColumnConfig{}
	StyleBool(trueStyle, falseStyle)(cfg)

	if cfg.style(true) != trueStyle {
		t.Error("StyleBool(true) should return true style")
	}
	if cfg.style(false) != falseStyle {
		t.Error("StyleBool(false) should return false style")
	}
}

func TestStyleThreshold(t *testing.T) {
	low := Style{FG: Red}
	mid := Style{FG: Yellow}
	high := Style{FG: Green}

	cfg := &ColumnConfig{}
	StyleThreshold(25, 75, low, mid, high)(cfg)

	if cfg.style(10.0) != low {
		t.Error("StyleThreshold(10) should return low style")
	}
	if cfg.style(50.0) != mid {
		t.Error("StyleThreshold(50) should return mid style")
	}
	if cfg.style(90.0) != high {
		t.Error("StyleThreshold(90) should return high style")
	}
	// boundary values
	if cfg.style(25.0) != mid {
		t.Error("StyleThreshold(25) should return mid style (inclusive)")
	}
	if cfg.style(75.0) != mid {
		t.Error("StyleThreshold(75) should return mid style (inclusive)")
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		val  any
		want float64
	}{
		{int(42), 42},
		{int8(8), 8},
		{int16(16), 16},
		{int32(32), 32},
		{int64(64), 64},
		{uint(42), 42},
		{uint8(8), 8},
		{uint16(16), 16},
		{uint32(32), 32},
		{uint64(64), 64},
		{float32(3.14), 3.140000104904175}, // float32 precision
		{float64(3.14), 3.14},
		{"not a number", 0},
	}

	for _, tt := range tests {
		got := toFloat64(tt.val)
		if got != tt.want {
			t.Errorf("toFloat64(%v) = %v, want %v", tt.val, got, tt.want)
		}
	}
}
