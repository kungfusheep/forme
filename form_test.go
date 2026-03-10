package glyph_test

import (
	"testing"

	. "github.com/kungfusheep/glyph"
)

func TestInputBindsToString(t *testing.T) {
	name := ""
	input := Input(&name).Placeholder("name")

	// simulate typing by setting value directly
	input.SetValue("pete")
	// bound value should not update from SetValue (only from keystroke onChange)
	// but Value() should work
	if input.Value() != "pete" {
		t.Errorf("expected Value() = 'pete', got %q", input.Value())
	}
}

func TestInputValidateRequired(t *testing.T) {
	name := ""
	input := Input(&name).Validate(VRequired, VOnChange)

	// trigger validation manually
	input.SetValue("")
	// runValidation is unexported, but we can check via Err after construction
	// since VOnChange would fire from handleChange, let's test the validator directly
	if err := VRequired(""); err == nil {
		t.Error("VRequired should reject empty string")
	}
	if err := VRequired("hello"); err != nil {
		t.Errorf("VRequired should accept 'hello', got %v", err)
	}
}

func TestVEmailValidator(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"", false},
		{"user@example.com", true},
		{"user@sub.example.com", true},
		{"bad", false},
		{"@example.com", false},
		{"user@", false},
		{"user@example.", false},
	}
	for _, tt := range tests {
		err := VEmail(tt.input)
		if tt.valid && err != nil {
			t.Errorf("VEmail(%q) should be valid, got %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("VEmail(%q) should be invalid", tt.input)
		}
	}
}

func TestVMinMaxLen(t *testing.T) {
	min3 := VMinLen(3)
	if err := min3("ab"); err == nil {
		t.Error("VMinLen(3) should reject 'ab'")
	}
	if err := min3("abc"); err != nil {
		t.Errorf("VMinLen(3) should accept 'abc', got %v", err)
	}

	max5 := VMaxLen(5)
	if err := max5("hello"); err != nil {
		t.Errorf("VMaxLen(5) should accept 'hello', got %v", err)
	}
	if err := max5("toolong"); err == nil {
		t.Error("VMaxLen(5) should reject 'toolong'")
	}
}

func TestVTrue(t *testing.T) {
	if err := VTrue(true); err != nil {
		t.Errorf("VTrue(true) should pass, got %v", err)
	}
	if err := VTrue(false); err == nil {
		t.Error("VTrue(false) should fail")
	}
}

func TestCheckboxValidate(t *testing.T) {
	agreed := false
	cb := Checkbox(&agreed, "I agree").Validate(VTrue, VOnChange)

	// toggle to true
	cb.Toggle()
	if cb.Err() != "" {
		t.Errorf("expected no error after toggling to true, got %q", cb.Err())
	}

	// toggle back to false
	cb.Toggle()
	if cb.Err() == "" {
		t.Error("expected error after toggling to false")
	}
}

func TestFormCompilesWithValidation(t *testing.T) {
	name := ""
	email := ""
	role := 0
	agree := false

	_ = Form.LabelBold()(
		Field("Name", Input(&name).Placeholder("name").Validate(VRequired, VOnBlur)),
		Field("Email", Input(&email).Placeholder("email").Validate(VEmail, VOnBlur)),
		Field("Role", Radio(&role, "Admin", "User", "Guest")),
		Field("Terms", Checkbox(&agree, "I accept").Validate(VTrue, VOnSubmit)),
	)
}
