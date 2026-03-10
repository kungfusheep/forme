package glyph

import "testing"

func TestListBindNavCollected(t *testing.T) {
	items := []string{"a", "b", "c"}
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).BindNav("j", "k"),
	))
	if len(tmpl.pendingBindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[0].pattern != "j" {
		t.Errorf("expected pattern 'j', got %q", tmpl.pendingBindings[0].pattern)
	}
	if tmpl.pendingBindings[1].pattern != "k" {
		t.Errorf("expected pattern 'k', got %q", tmpl.pendingBindings[1].pattern)
	}
}

func TestListBindDeleteCollected(t *testing.T) {
	items := []string{"a", "b"}
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).BindNav("j", "k").BindDelete("d"),
	))
	if len(tmpl.pendingBindings) != 3 {
		t.Fatalf("expected 3 bindings, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[2].pattern != "d" {
		t.Errorf("expected pattern 'd', got %q", tmpl.pendingBindings[2].pattern)
	}
}

func TestCheckboxBindToggleCollected(t *testing.T) {
	checked := false
	tmpl := Build(VBox(
		Checkbox(&checked, "agree").BindToggle("x"),
	))
	if len(tmpl.pendingBindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[0].pattern != "x" {
		t.Errorf("expected pattern 'x', got %q", tmpl.pendingBindings[0].pattern)
	}
}

func TestRadioBindNavCollected(t *testing.T) {
	sel := 0
	tmpl := Build(VBox(
		Radio(&sel, "a", "b", "c").BindNav("n", "p"),
	))
	if len(tmpl.pendingBindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[0].pattern != "n" {
		t.Errorf("expected pattern 'n', got %q", tmpl.pendingBindings[0].pattern)
	}
	if tmpl.pendingBindings[1].pattern != "p" {
		t.Errorf("expected pattern 'p', got %q", tmpl.pendingBindings[1].pattern)
	}
}

func TestCheckListBindingsCollected(t *testing.T) {
	type Item struct {
		Name string
		Done bool
	}
	items := []Item{{Name: "a"}, {Name: "b"}}
	tmpl := Build(VBox(
		CheckList(&items).
			Checked(func(i *Item) *bool { return &i.Done }).
			Render(func(i *Item) any { return Text(&i.Name) }).
			BindNav("j", "k").
			BindToggle("x").
			BindDelete("d"),
	))
	// BindNav(2) + BindToggle(1) + BindDelete(1) = 4
	if len(tmpl.pendingBindings) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(tmpl.pendingBindings))
	}
	expected := []string{"j", "k", "x", "d"}
	for i, exp := range expected {
		if tmpl.pendingBindings[i].pattern != exp {
			t.Errorf("binding %d: expected %q, got %q", i, exp, tmpl.pendingBindings[i].pattern)
		}
	}
}

func TestInputBindCollected(t *testing.T) {
	tmpl := Build(VBox(
		Input().Placeholder("name").Bind(),
	))
	if tmpl.pendingTIB == nil {
		t.Fatal("expected textInputBinding to be set")
	}
	if tmpl.pendingTIB.value == nil {
		t.Error("expected value pointer to be set")
	}
	if tmpl.pendingTIB.cursor == nil {
		t.Error("expected cursor pointer to be set")
	}
}

func TestMultipleComponentBindingsCollected(t *testing.T) {
	checked := false
	sel := 0
	items := []string{"a", "b"}
	tmpl := Build(VBox(
		Checkbox(&checked, "agree").BindToggle("a"),
		Radio(&sel, "x", "y").BindNav("n", "p"),
		List(&items).Render(func(s *string) any { return Text(s) }).BindNav("j", "k"),
	))
	// 1 (checkbox) + 2 (radio) + 2 (list) = 5
	if len(tmpl.pendingBindings) != 5 {
		t.Fatalf("expected 5 bindings, got %d", len(tmpl.pendingBindings))
	}
}

func TestListBindPageNavCollected(t *testing.T) {
	items := []string{"a", "b", "c"}
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).BindPageNav("<C-d>", "<C-u>"),
	))
	if len(tmpl.pendingBindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[0].pattern != "<C-d>" {
		t.Errorf("expected pattern '<C-d>', got %q", tmpl.pendingBindings[0].pattern)
	}
	if tmpl.pendingBindings[1].pattern != "<C-u>" {
		t.Errorf("expected pattern '<C-u>', got %q", tmpl.pendingBindings[1].pattern)
	}
}

func TestListBindFirstLastCollected(t *testing.T) {
	items := []string{"a", "b", "c"}
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).BindFirstLast("g", "G"),
	))
	if len(tmpl.pendingBindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[0].pattern != "g" {
		t.Errorf("expected pattern 'g', got %q", tmpl.pendingBindings[0].pattern)
	}
	if tmpl.pendingBindings[1].pattern != "G" {
		t.Errorf("expected pattern 'G', got %q", tmpl.pendingBindings[1].pattern)
	}
}

func TestListBindVimNavCollected(t *testing.T) {
	items := []string{"a", "b", "c"}
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).BindVimNav(),
	))
	// j, k, <C-d>, <C-u>, g, G = 6
	if len(tmpl.pendingBindings) != 6 {
		t.Fatalf("expected 6 bindings, got %d", len(tmpl.pendingBindings))
	}
	expected := []string{"j", "k", "<C-d>", "<C-u>", "g", "G"}
	for i, exp := range expected {
		if tmpl.pendingBindings[i].pattern != exp {
			t.Errorf("binding %d: expected %q, got %q", i, exp, tmpl.pendingBindings[i].pattern)
		}
	}
}

func TestCheckListBindPageNavCollected(t *testing.T) {
	type Item struct {
		Name string
		Done bool
	}
	items := []Item{{Name: "a"}, {Name: "b"}}
	tmpl := Build(VBox(
		CheckList(&items).
			Checked(func(i *Item) *bool { return &i.Done }).
			Render(func(i *Item) any { return Text(&i.Name) }).
			BindPageNav("<C-d>", "<C-u>").
			BindFirstLast("g", "G"),
	))
	// 2 (page) + 2 (first/last) = 4
	if len(tmpl.pendingBindings) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(tmpl.pendingBindings))
	}
	expected := []string{"<C-d>", "<C-u>", "g", "G"}
	for i, exp := range expected {
		if tmpl.pendingBindings[i].pattern != exp {
			t.Errorf("binding %d: expected %q, got %q", i, exp, tmpl.pendingBindings[i].pattern)
		}
	}
}

func TestCheckListBindVimNavCollected(t *testing.T) {
	type Item struct {
		Name string
		Done bool
	}
	items := []Item{{Name: "a"}, {Name: "b"}}
	tmpl := Build(VBox(
		CheckList(&items).
			Checked(func(i *Item) *bool { return &i.Done }).
			Render(func(i *Item) any { return Text(&i.Name) }).
			BindVimNav(),
	))
	// j, k, <C-d>, <C-u>, g, G = 6
	if len(tmpl.pendingBindings) != 6 {
		t.Fatalf("expected 6 bindings, got %d", len(tmpl.pendingBindings))
	}
	expected := []string{"j", "k", "<C-d>", "<C-u>", "g", "G"}
	for i, exp := range expected {
		if tmpl.pendingBindings[i].pattern != exp {
			t.Errorf("binding %d: expected %q, got %q", i, exp, tmpl.pendingBindings[i].pattern)
		}
	}
}

func TestListHandleCollected(t *testing.T) {
	items := []string{"a", "b", "c"}
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).
			Handle("<Enter>", func(s *string) {}).
			Handle("w", func(s *string) {}),
	))
	if len(tmpl.pendingBindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[0].pattern != "<Enter>" {
		t.Errorf("expected pattern '<Enter>', got %q", tmpl.pendingBindings[0].pattern)
	}
	if tmpl.pendingBindings[1].pattern != "w" {
		t.Errorf("expected pattern 'w', got %q", tmpl.pendingBindings[1].pattern)
	}
}

func TestListHandleCallsWithSelected(t *testing.T) {
	items := []string{"a", "b", "c"}
	var got string
	var list *ListC[string]
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).
			Ref(func(l *ListC[string]) { list = l }).
			Handle("<Enter>", func(s *string) { got = *s }),
	))
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	list.Down(nil) // select "b"
	// extract and call the handler
	for _, b := range list.bindings() {
		if b.pattern == "<Enter>" {
			b.handler.(func())()
		}
	}
	if got != "b" {
		t.Errorf("expected 'b', got %q", got)
	}
}

func TestListHandleSkipsNilSelection(t *testing.T) {
	items := []string{}
	called := false
	var list *ListC[string]
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }).
			Ref(func(l *ListC[string]) { list = l }).
			Handle("<Enter>", func(s *string) { called = true }),
	))
	buf := NewBuffer(40, 10)
	tmpl.Execute(buf, 40, 10)

	for _, b := range list.bindings() {
		if b.pattern == "<Enter>" {
			b.handler.(func())()
		}
	}
	if called {
		t.Error("expected handler not to be called with empty list")
	}
}

func TestCheckListHandleCollected(t *testing.T) {
	type Item struct {
		Name string
		Done bool
	}
	items := []Item{{Name: "a"}, {Name: "b"}}
	tmpl := Build(VBox(
		CheckList(&items).
			Checked(func(i *Item) *bool { return &i.Done }).
			Render(func(i *Item) any { return Text(&i.Name) }).
			Handle("<Enter>", func(i *Item) {}),
	))
	if len(tmpl.pendingBindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(tmpl.pendingBindings))
	}
	if tmpl.pendingBindings[0].pattern != "<Enter>" {
		t.Errorf("expected pattern '<Enter>', got %q", tmpl.pendingBindings[0].pattern)
	}
}

func TestBindingsBubbleUpFromIfElse(t *testing.T) {
	items := []string{"a", "b"}
	show := true
	tmpl := Build(VBox(
		If(&show).Then(
			List(&items).Render(func(s *string) any { return Text(s) }).BindNav("j", "k"),
		).Else(
			Text("empty"),
		),
	))
	if len(tmpl.pendingBindings) != 2 {
		t.Fatalf("expected 2 bindings from Then branch, got %d", len(tmpl.pendingBindings))
	}
}

func TestBindingsBubbleUpFromElseBranch(t *testing.T) {
	items := []string{"a", "b"}
	show := false
	tmpl := Build(VBox(
		If(&show).Then(
			Text("loading"),
		).Else(
			List(&items).Render(func(s *string) any { return Text(s) }).BindVimNav(),
		),
	))
	// j, k, <C-d>, <C-u>, g, G = 6
	if len(tmpl.pendingBindings) != 6 {
		t.Fatalf("expected 6 bindings from Else branch, got %d", len(tmpl.pendingBindings))
	}
}

func TestNoBindingsWhenNotUsed(t *testing.T) {
	items := []string{"a", "b"}
	tmpl := Build(VBox(
		List(&items).Render(func(s *string) any { return Text(s) }),
	))
	if len(tmpl.pendingBindings) != 0 {
		t.Errorf("expected 0 bindings, got %d", len(tmpl.pendingBindings))
	}
}
