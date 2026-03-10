package glyph

import (
	"fmt"
	"regexp"
	"strings"
)

// validatable is implemented by controls that support validation.
type validatable interface {
	Err() string
	runValidation()
}

// FormField pairs a label with an input control.
type FormField struct {
	label   string
	control any
	err     string // validation error for this field
	focused bool
}

// Field creates a form field pairing a label with any control component.
func Field(label string, control any) FormField {
	return FormField{label: label, control: control}
}

type FormC struct {
	fields     []FormField
	fm         *FocusManager
	gap        int8
	labelWidth int16
	labelStyle Style
	grow       float32
	margin     [4]int16
	onSubmit   func()
}

type FormFn func(fields ...FormField) *FormC

// Form creates a form from labeled fields with aligned labels
// and automatic focus management. Configure with methods, then call with fields.
//
//	Form.LabelBold().OnSubmit(register)(
//	    Field("Name", Input().Placeholder("Enter your name")),
//	    Field("Email", Input().Placeholder("you@example.com")),
//	    Field("Password", Input().Placeholder("password").Mask('*')),
//	)
var Form FormFn = func(fields ...FormField) *FormC {
	f := &FormC{
		fields: fields,
		fm:     NewFocusManager(),
	}

	// auto-calculate label width from longest label + colon
	for _, ff := range fields {
		w := int16(len(ff.label) + 1) // +1 for ":"
		if w > f.labelWidth {
			f.labelWidth = w
		}
	}

	// auto-wire focusable controls and blur validation
	var focusableFields []*FormField // maps FM index → FormField
	for idx := range fields {
		ff := &f.fields[idx]
		if fc, ok := ff.control.(focusable); ok {
			fieldRef := ff
			focusableFields = append(focusableFields, fieldRef)
			switch ctrl := ff.control.(type) {
			case *InputC:
				ctrl.ManagedBy(f.fm)
				ctrl.onBlur = func() {
					fieldRef.err = ctrl.Err()
				}
			case *CheckboxC:
				f.fm.Register(fc)
				ctrl.onBlur = func() {
					fieldRef.err = ctrl.Err()
				}
				f.fm.ItemBindings(
					binding{pattern: "<Space>", handler: func() { ctrl.Toggle() }},
				)
			case *RadioC:
				f.fm.Register(fc)
				f.fm.ItemBindings(
					binding{pattern: "j", handler: func() { ctrl.Next() }},
					binding{pattern: "k", handler: func() { ctrl.Prev() }},
				)
			default:
				f.fm.Register(fc)
			}
		}
	}

	// first focusable field starts focused
	if len(focusableFields) > 0 {
		focusableFields[0].focused = true
	}

	// track focus changes to update visual indicator
	f.fm.OnChange(func(idx int) {
		for i, ff := range focusableFields {
			ff.focused = (i == idx)
		}
	})
	f.fm.OnBlur(func() {
		for _, ff := range focusableFields {
			ff.focused = false
		}
	})

	return f
}

// Gap sets the vertical gap between fields.
func (f FormFn) Gap(g int8) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.gap = g
		return form
	}
}

// LabelStyle sets the full style for all labels.
func (f FormFn) LabelStyle(s Style) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.labelStyle = s
		return form
	}
}

// LabelFG sets the foreground color for all labels.
func (f FormFn) LabelFG(c Color) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.labelStyle.FG = c
		return form
	}
}

// LabelBold sets labels to bold.
func (f FormFn) LabelBold() FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.labelStyle = form.labelStyle.Bold()
		return form
	}
}

// NextKey sets the key for advancing focus (default: Tab).
func (f FormFn) NextKey(key string) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.fm.NextKey(key)
		return form
	}
}

// PrevKey sets the key for reversing focus (default: Shift-Tab).
func (f FormFn) PrevKey(key string) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.fm.PrevKey(key)
		return form
	}
}

// OnFocusChange sets a callback that fires when focus changes.
func (f FormFn) OnFocusChange(fn func(index int)) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.fm.OnChange(fn)
		return form
	}
}

// OnSubmit sets a callback that fires when Enter is pressed.
func (f FormFn) OnSubmit(fn func()) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.onSubmit = fn
		return form
	}
}

// Grow sets the flex grow factor.
func (f FormFn) Grow(g float32) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.grow = g
		return form
	}
}

// Margin sets equal margin on all sides.
func (f FormFn) Margin(m int16) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.margin = [4]int16{m, m, m, m}
		return form
	}
}

// MarginVH sets vertical and horizontal margin.
func (f FormFn) MarginVH(v, h int16) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.margin = [4]int16{v, h, v, h}
		return form
	}
}

// MarginTRBL sets top, right, bottom, left margin individually.
func (f FormFn) MarginTRBL(t, r, b, l int16) FormFn {
	return func(fields ...FormField) *FormC {
		form := f(fields...)
		form.margin = [4]int16{t, r, b, l}
		return form
	}
}

// FocusManager returns the internal focus manager for external wiring.
func (f *FormC) FocusManager() *FocusManager {
	return f.fm
}

// ValidateAll runs validation on all fields that have VOnSubmit set.
// Returns true if all fields are valid.
func (f *FormC) ValidateAll() bool {
	valid := true
	for i := range f.fields {
		ff := &f.fields[i]
		if v, ok := ff.control.(validatable); ok {
			v.runValidation()
			ff.err = v.Err()
			if ff.err != "" {
				valid = false
			}
		}
	}
	return valid
}

// toTemplate builds the VBox of HBox rows with optional error display.
func (f *FormC) toTemplate() any {
	rows := make([]any, 0, len(f.fields)*2)
	for i := range f.fields {
		ff := &f.fields[i]
		ls := f.labelStyle
		ls.Align = AlignRight
		ls = ls.MarginTRBL(0, 1, 0, 0)

		label := Text(ff.label + ":").Width(f.labelWidth).Style(ls)
		indicator := If(&ff.focused).
			Then(Text("▸").Width(1)).
			Else(Text("").Width(1))
		rows = append(rows, HBox(indicator, label, ff.control))

		// add error display if the control supports validation
		if _, ok := ff.control.(validatable); ok {
			spacer := Text("").Width(f.labelWidth+2).MarginTRBL(0, 1, 0, 0)
			rows = append(rows, If(&ff.err).Then(
				HBox(spacer, Text(&ff.err).FG(Red)),
			))
		}
	}

	box := VBox.Gap(f.gap)
	if f.grow > 0 {
		box = box.Grow(f.grow)
	}
	if f.margin != [4]int16{} {
		box = box.MarginTRBL(f.margin[0], f.margin[1], f.margin[2], f.margin[3])
	}
	return box(rows...)
}

// bindings returns Form-specific bindings only.
// Tab/Shift-Tab are handled by the FocusManager in wireBindings.
func (f *FormC) bindings() []binding {
	if f.onSubmit != nil {
		enterBinding := binding{pattern: "<Enter>", handler: f.onSubmit}
		f.fm.subBindings = append(f.fm.subBindings, enterBinding)
		return []binding{enterBinding}
	}
	return nil
}

// ============================================================================
// Validators
// ============================================================================

// ValidateOn controls when validation runs. Combine with bitwise OR.
type ValidateOn uint8

const (
	VOnChange ValidateOn = 1 << iota // validate on every keystroke
	VOnBlur                          // validate when field loses focus
	VOnSubmit                        // validate on form submit
)

// StringValidator validates a string value. Pass to Input().Validate().
// Return nil for valid, non-nil error for the message to display.
type StringValidator func(string) error

// BoolValidator validates a boolean value. Pass to Checkbox().Validate().
// Return nil for valid, non-nil error for the message to display.
type BoolValidator func(bool) error

// VRequired rejects empty strings.
func VRequired(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("required")
	}
	return nil
}

// VEmail rejects strings that don't look like email addresses.
func VEmail(s string) error {
	if !strings.Contains(s, "@") || !strings.Contains(s, ".") {
		return fmt.Errorf("invalid email")
	}
	at := strings.LastIndex(s, "@")
	if at == 0 || at == len(s)-1 {
		return fmt.Errorf("invalid email")
	}
	domain := s[at+1:]
	if !strings.Contains(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("invalid email")
	}
	return nil
}

// VMinLen rejects strings shorter than n.
func VMinLen(n int) StringValidator {
	return func(s string) error {
		if len(s) < n {
			return fmt.Errorf("min %d characters", n)
		}
		return nil
	}
}

// VMaxLen rejects strings longer than n.
func VMaxLen(n int) StringValidator {
	return func(s string) error {
		if len(s) > n {
			return fmt.Errorf("max %d characters", n)
		}
		return nil
	}
}

// VMatch rejects strings that don't match the given regex pattern.
func VMatch(pattern string) StringValidator {
	re := regexp.MustCompile(pattern)
	return func(s string) error {
		if s == "" {
			return nil
		}
		if !re.MatchString(s) {
			return fmt.Errorf("invalid format")
		}
		return nil
	}
}

// VTrue rejects false values.
func VTrue(b bool) error {
	if !b {
		return fmt.Errorf("required")
	}
	return nil
}
