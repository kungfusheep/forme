package main

import (
	. "github.com/kungfusheep/forme"
	"github.com/kungfusheep/riffkey"
)

type Task struct {
	Name string
	Done bool
}

func main() {
	app, _ := NewApp()

	// Simple state
	agreed := false
	selectedPlan := 0
	tasks := []Task{
		{Name: "Read documentation", Done: true},
		{Name: "Try the examples", Done: false},
		{Name: "Build something cool", Done: false},
	}
	var taskList *CheckListC[Task]

	app.SetView(VBox.Border(BorderRounded).Title("Form Components Demo").Gap(1)(
		// Checkbox - toggle with 'a'
		VBox.Gap(0)(
			Text("Checkbox (press 'a' to toggle):").Bold(),
			Checkbox(&agreed, "I agree to the terms"),
			HBox.Gap(1)(
				Text("Status:"),
				If(&agreed).Then(Text("Agreed!").FG(Green)).Else(Text("Not agreed").FG(Red)),
			),
		),

		HRule(),

		// Radio - use count prefix: 1r, 2r, 3r
		VBox.Gap(0)(
			Text("Radio (1r/2r/3r to select):").Bold(),
			Radio(&selectedPlan, "Free", "Pro", "Enterprise"),
		),

		HRule(),

		// CheckList - j/k to navigate, x to toggle
		VBox.Gap(0)(
			Text("CheckList (j/k to nav, x to toggle):").Bold(),
			CheckList(&tasks).
				Checked(func(t *Task) *bool { return &t.Done }).
				Render(func(t *Task) any { return Text(&t.Name) }).
				BindNav(app, "j", "k").
				BindToggle(app, "x").
				Ref(&taskList),
		),

		HRule(),
		Text("Press 'q' to quit").Dim(),
	))

	// Non-conflicting bindings
	app.Handle("a", func() { agreed = !agreed })
	app.Handle("r", func(m riffkey.Match) {
		// Use count-1 as index (1r = index 0, 2r = index 1, etc)
		if m.Count > 0 && m.Count <= 3 {
			selectedPlan = m.Count - 1
		}
	})
	app.Handle("q", app.Stop)

	app.Run()
}
