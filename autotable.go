package glyph

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// ColumnOption configures a single AutoTable column.
type ColumnOption func(*ColumnConfig)

// ColumnConfig holds rendering configuration for one column.
type ColumnConfig struct {
	align    Align
	hasAlign bool // true if explicitly set (vs type default)
	format   func(any) string
	style    func(any) Style
}

// Align sets the column alignment.
func (c *ColumnConfig) Align(a Align) { c.align = a; c.hasAlign = true }

// Format sets a function that converts the field value to display text.
func (c *ColumnConfig) Format(fn func(any) string) { c.format = fn }

// Style sets a function that returns a per-cell style based on the field value.
func (c *ColumnConfig) Style(fn func(any) Style) { c.style = fn }

// ----------------------------------------------------------------------------
// canned format presets
// ----------------------------------------------------------------------------

// Number formats numeric values with comma separators.
// decimals controls decimal places for floats (ignored for integers).
func Number(decimals int) ColumnOption {
	return func(c *ColumnConfig) {
		c.Align(AlignRight)
		c.Format(func(v any) string {
			return formatNumber(v, decimals)
		})
	}
}

// Currency formats numeric values with a symbol prefix and comma
// separators - it is by no means a full internationalization solution,
// but it's a quick default.
func Currency(symbol string, decimals int) ColumnOption {
	return func(c *ColumnConfig) {
		c.Align(AlignRight)
		c.Format(func(v any) string {
			return symbol + formatNumber(v, decimals)
		})
	}
}

// Percent formats numeric values as percentages.
func Percent(decimals int) ColumnOption {
	return func(c *ColumnConfig) {
		c.Align(AlignRight)
		c.Format(func(v any) string {
			f := toFloat64(v)
			return strconv.FormatFloat(f, 'f', decimals, 64) + "%"
		})
	}
}

// PercentChange formats numeric values as signed percentages with green/red coloring.
func PercentChange(decimals int) ColumnOption {
	return func(c *ColumnConfig) {
		c.Align(AlignRight)
		c.Format(func(v any) string {
			f := toFloat64(v)
			sign := "+"
			if f < 0 {
				sign = ""
			}
			return sign + strconv.FormatFloat(f, 'f', decimals, 64) + "%"
		})
		c.Style(func(v any) Style {
			if toFloat64(v) >= 0 {
				return Style{FG: Green}
			}
			return Style{FG: Red}
		})
	}
}

// Bytes formats numeric values as human-readable byte sizes.
func Bytes() ColumnOption {
	return func(c *ColumnConfig) {
		c.Align(AlignRight)
		c.Format(func(v any) string {
			return formatBytes(toFloat64(v))
		})
	}
}

// Bool formats boolean values with custom labels.
func Bool(yes, no string) ColumnOption {
	return func(c *ColumnConfig) {
		c.Align(AlignCenter)
		c.Format(func(v any) string {
			if b, ok := v.(bool); ok && b {
				return yes
			}
			return no
		})
	}
}

// ----------------------------------------------------------------------------
// canned style presets
// ----------------------------------------------------------------------------

// StyleSign colors cells based on the numeric sign of the value.
func StyleSign(positive, negative Style) ColumnOption {
	return func(c *ColumnConfig) {
		c.Style(func(v any) Style {
			if toFloat64(v) >= 0 {
				return positive
			}
			return negative
		})
	}
}

// StyleBool colors cells based on a boolean value.
func StyleBool(trueStyle, falseStyle Style) ColumnOption {
	return func(c *ColumnConfig) {
		c.Style(func(v any) Style {
			if b, ok := v.(bool); ok && b {
				return trueStyle
			}
			return falseStyle
		})
	}
}

// StyleThreshold colors cells based on numeric value thresholds.
// Values < low get belowStyle, low..high get betweenStyle, > high get aboveStyle.
func StyleThreshold(low, high float64, belowStyle, betweenStyle, aboveStyle Style) ColumnOption {
	return func(c *ColumnConfig) {
		c.Style(func(v any) Style {
			f := toFloat64(v)
			if f < low {
				return belowStyle
			}
			if f > high {
				return aboveStyle
			}
			return betweenStyle
		})
	}
}

// ----------------------------------------------------------------------------
// internal helpers
// ----------------------------------------------------------------------------

// toFloat64 converts common numeric types to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	default:
		return 0
	}
}

// formatNumber formats a numeric value with comma separators.
func formatNumber(v any, decimals int) string {
	f := toFloat64(v)
	// format the number without commas first
	s := strconv.FormatFloat(f, 'f', decimals, 64)
	return insertCommas(s)
}

// insertCommas adds thousand separators to a numeric string.
func insertCommas(s string) string {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}

	// split on decimal point
	integer, decimal, hasDecimal := strings.Cut(s, ".")

	// insert commas into integer part from right to left
	n := len(integer)
	if n <= 3 {
		// no commas needed
	} else {
		var b strings.Builder
		b.Grow(n + n/3)
		start := n % 3
		if start == 0 {
			start = 3
		}
		b.WriteString(integer[:start])
		for i := start; i < n; i += 3 {
			b.WriteByte(',')
			b.WriteString(integer[i : i+3])
		}
		integer = b.String()
	}

	var result string
	if hasDecimal {
		result = integer + "." + decimal
	} else {
		result = integer
	}

	if neg {
		return "-" + result
	}
	return result
}

// formatBytes converts a byte count to a human-readable string.
func formatBytes(b float64) string {
	if b < 0 {
		return "-" + formatBytes(-b)
	}

	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	if b < 1 {
		return "0 B"
	}

	exp := int(math.Log(b) / math.Log(1024))
	if exp >= len(units) {
		exp = len(units) - 1
	}

	val := b / math.Pow(1024, float64(exp))

	if exp == 0 {
		return fmt.Sprintf("%.0f %s", val, units[exp])
	}
	return fmt.Sprintf("%.1f %s", val, units[exp])
}

// ============================================================================
// AutoTable component
// ============================================================================

// autoTableSortState tracks the current sort column and direction.
// allocated once by Sortable, shared via pointer through value copies.
type autoTableSortState struct {
	col         int    // -1 = unsorted, 0..n-1 = column index
	asc         bool   // true = ascending
	initialCol  string // field name for default sort (resolved at compile time)
	initialAsc  bool
	initialDone bool // true once initialCol has been resolved
}

// autoTableScroll manages viewport scrolling for AutoTable.
// renders all rows to an internal buffer, blits the visible window to screen.
type autoTableScroll struct {
	offset     int     // first visible data row
	maxVisible int     // viewport height in data rows (excludes header)
	buf        *Buffer // internal buffer for all data rows (nil until first render)
	bufW       int     // width of internal buffer (for resize detection)
}

func (s *autoTableScroll) scrollDown(n int, total int) {
	s.offset += n
	if max := total - s.maxVisible; max > 0 {
		if s.offset > max {
			s.offset = max
		}
	} else {
		s.offset = 0
	}
}

func (s *autoTableScroll) scrollUp(n int) {
	s.offset -= n
	if s.offset < 0 {
		s.offset = 0
	}
}

func (s *autoTableScroll) pageDown(total int) { s.scrollDown(s.maxVisible, total) }
func (s *autoTableScroll) pageUp()            { s.scrollUp(s.maxVisible) }

func (s *autoTableScroll) clamp(total int) {
	if max := total - s.maxVisible; max > 0 {
		if s.offset > max {
			s.offset = max
		}
	} else {
		s.offset = 0
	}
	if s.offset < 0 {
		s.offset = 0
	}
}

type AutoTableC struct {
	data        any      // slice of structs
	columns     []string // field names to display (nil = all exported)
	headers     []string // custom header names (parallel to columns)
	headerStyle Style
	rowStyle    Style
	altRowStyle *Style
	gap         int8
	border      BorderStyle
	margin      [4]int16

	columnConfigs map[string]ColumnOption // per-column config keyed by field name

	sortState        *autoTableSortState // nil unless Sortable called
	scroll           *autoTableScroll    // nil unless Scrollable called
	declaredBindings []binding
}

// AutoTable creates a table directly from a slice of structs.
// Pass a plain slice for a static snapshot, or a pointer (&items) for reactive updates.
// Columns are derived from exported struct fields; use .Columns() to select and order them.
func AutoTable(data any) AutoTableC {
	return AutoTableC{
		data:        data,
		headerStyle: Style{Attr: AttrBold},
		gap:         1,
	}
}

// Columns selects which struct fields to display and in what order.
// Field names are case-sensitive and must match exported struct fields.
func (t AutoTableC) Columns(names ...string) AutoTableC {
	t.columns = names
	return t
}

// Headers sets custom header labels for the columns.
// Must be called after Columns() and have the same number of entries.
func (t AutoTableC) Headers(names ...string) AutoTableC {
	t.headers = names
	return t
}

// Column configures rendering for a specific column by struct field name.
// The option can be a canned preset or a custom function:
//
//	AutoTable(&data).
//	    Column("Price", Currency("$", 2)).
//	    Column("Change", PercentChange(1)).
//	    Column("Active", func(c *ColumnConfig) {
//	        c.Align(AlignCenter)
//	        c.Format(func(v any) string { ... })
//	    })
func (t AutoTableC) Column(name string, opt ColumnOption) AutoTableC {
	if t.columnConfigs == nil {
		t.columnConfigs = make(map[string]ColumnOption)
	}
	t.columnConfigs[name] = opt
	return t
}

// HeaderStyle sets the header row style.
func (t AutoTableC) HeaderStyle(s Style) AutoTableC {
	t.headerStyle = s
	return t
}

// RowStyle sets the default row style.
func (t AutoTableC) RowStyle(s Style) AutoTableC {
	t.rowStyle = s
	return t
}

// AltRowStyle sets the alternating row style.
func (t AutoTableC) AltRowStyle(s Style) AutoTableC {
	t.altRowStyle = &s
	return t
}

// Gap sets the spacing between columns.
func (t AutoTableC) Gap(g int8) AutoTableC {
	t.gap = g
	return t
}

// Border sets the border style.
func (t AutoTableC) Border(b BorderStyle) AutoTableC {
	t.border = b
	return t
}

// Margin sets uniform margin on all sides.
func (t AutoTableC) Margin(all int16) AutoTableC { t.margin = [4]int16{all, all, all, all}; return t }

// MarginVH sets vertical and horizontal margin.
func (t AutoTableC) MarginVH(v, h int16) AutoTableC { t.margin = [4]int16{v, h, v, h}; return t }

// MarginTRBL sets individual margins for top, right, bottom, left.
func (t AutoTableC) MarginTRBL(a, b, c, d int16) AutoTableC {
	t.margin = [4]int16{a, b, c, d}
	return t
}

// Sortable enables column sorting via jump labels.
// when the app's jump key is pressed, each column header becomes a jump target.
// selecting a column sorts ascending; selecting the same column again toggles direction.
func (t AutoTableC) Sortable() AutoTableC {
	if t.sortState == nil {
		t.sortState = &autoTableSortState{col: -1}
	}
	return t
}

// SortBy sets the default sort column and direction. implies Sortable().
// field is the struct field name (e.g. "CPU"). asc true = ascending.
func (t AutoTableC) SortBy(field string, asc bool) AutoTableC {
	t = t.Sortable()
	t.sortState.initialCol = field
	t.sortState.initialAsc = asc
	return t
}

// Scrollable enables viewport scrolling with the given maximum visible rows.
// renders all data rows to an internal buffer, blits only the visible window.
func (t AutoTableC) Scrollable(maxVisible int) AutoTableC {
	if t.scroll == nil {
		t.scroll = &autoTableScroll{maxVisible: maxVisible}
	} else {
		t.scroll.maxVisible = maxVisible
	}
	return t
}

// BindNav registers key bindings for scrolling down/up by one row.
// the closures capture the scroll pointer and data pointer, reading the
// current slice length at invocation time for correct clamping.
func (t AutoTableC) BindNav(down, up string) AutoTableC {
	sc := t.scroll
	data := t.data
	t.declaredBindings = append(t.declaredBindings,
		binding{pattern: down, handler: func() {
			if sc == nil {
				return
			}
			total := reflect.ValueOf(data).Elem().Len()
			sc.scrollDown(1, total)
		}},
		binding{pattern: up, handler: func() {
			if sc == nil {
				return
			}
			sc.scrollUp(1)
		}},
	)
	return t
}

// BindPageNav registers key bindings for page-sized scrolling.
func (t AutoTableC) BindPageNav(pageDown, pageUp string) AutoTableC {
	sc := t.scroll
	data := t.data
	t.declaredBindings = append(t.declaredBindings,
		binding{pattern: pageDown, handler: func() {
			if sc == nil {
				return
			}
			total := reflect.ValueOf(data).Elem().Len()
			sc.pageDown(total)
		}},
		binding{pattern: pageUp, handler: func() {
			if sc == nil {
				return
			}
			sc.pageUp()
		}},
	)
	return t
}

// BindVimNav wires standard vim-style scroll keys:
// j/k for line, Ctrl-d/Ctrl-u for page.
func (t AutoTableC) BindVimNav() AutoTableC {
	return t.BindNav("j", "k").BindPageNav("<C-d>", "<C-u>")
}

func (t AutoTableC) bindings() []binding { return t.declaredBindings }

// autoTableSort sorts a *[]T slice in-place by the given struct field index.
func autoTableSort(data any, fieldIdx int, asc bool) {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Pointer {
		return
	}
	slice := rv.Elem()
	if slice.Kind() != reflect.Slice {
		return
	}

	n := slice.Len()
	if n < 2 {
		return
	}

	// copy to avoid aliasing during write-back
	tmp := make([]reflect.Value, n)
	for i := 0; i < n; i++ {
		tmp[i] = reflect.New(slice.Type().Elem()).Elem()
		tmp[i].Set(slice.Index(i))
	}

	sortSliceReflect(tmp, fieldIdx, asc)

	for i, v := range tmp {
		slice.Index(i).Set(v)
	}
}

// sortSliceReflect sorts reflected values by a struct field.
func sortSliceReflect(items []reflect.Value, fieldIdx int, asc bool) {
	n := len(items)
	// simple insertion sort -- tables are typically small
	for i := 1; i < n; i++ {
		for j := i; j > 0; j-- {
			a := derefValue(items[j-1]).Field(fieldIdx)
			b := derefValue(items[j]).Field(fieldIdx)
			cmp := compareValues(a, b)
			if !asc {
				cmp = -cmp
			}
			if cmp <= 0 {
				break
			}
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}

func derefValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
}

// compareValues compares two reflected values, handling numeric types natively
// and falling back to string comparison.
func compareValues(a, b reflect.Value) int {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ai, bi := a.Int(), b.Int()
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ai, bi := a.Uint(), b.Uint()
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0
	case reflect.Float32, reflect.Float64:
		ai, bi := a.Float(), b.Float()
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
		return 0
	case reflect.String:
		as, bs := a.String(), b.String()
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	default:
		// fallback: compare string representations
		as := fmt.Sprintf("%v", a.Interface())
		bs := fmt.Sprintf("%v", b.Interface())
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	}
}
