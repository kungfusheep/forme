package forme

// ThemeEx provides a set of styles for consistent UI appearance.
// Use CascadeStyle on containers to apply theme styles to children.
type ThemeEx struct {
	Base   Style // default text style
	Muted  Style // de-emphasized text
	Accent Style // highlighted/important text
	Error  Style // error messages
	Border Style // border/divider style
}

// Pre-defined themes

// ThemeDark is a dark theme with light text on dark background.
var ThemeDark = ThemeEx{
	Base:   Style{FG: White},
	Muted:  Style{FG: BrightBlack},
	Accent: Style{FG: BrightCyan},
	Error:  Style{FG: BrightRed},
	Border: Style{FG: BrightBlack},
}

// ThemeLight is a light theme with dark text on light background.
var ThemeLight = ThemeEx{
	Base:   Style{FG: Black},
	Muted:  Style{FG: BrightBlack},
	Accent: Style{FG: Blue},
	Error:  Style{FG: Red},
	Border: Style{FG: White},
}

// ThemeMonochrome is a minimal theme using only attributes.
var ThemeMonochrome = ThemeEx{
	Base:   Style{},
	Muted:  Style{Attr: AttrDim},
	Accent: Style{Attr: AttrBold},
	Error:  Style{Attr: AttrBold | AttrUnderline},
	Border: Style{Attr: AttrDim},
}
