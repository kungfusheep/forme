package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	. "github.com/kungfusheep/glyph"
)

func TestCommandCenterLayout(t *testing.T) {
	reqData := make([]float64, 40)
	latData := make([]float64, 40)
	errData := make([]float64, 40)
	for i := range reqData {
		reqData[i] = 80 + rand.Float64()*60 + 30*math.Sin(float64(i)*0.2)
		latData[i] = 18 + rand.Float64()*12 + 6*math.Sin(float64(i)*0.15)
		errData[i] = rand.Float64() * 3
	}

	reqRate := "142 req/s"
	p99Lat  := "24ms"
	errRate := "0.4%"
	clock   := "12:00:00"

	services := []service{
		{Name: "api-gateway",      Status: "live", CPU: 12.0, CPUStr: " 12.0%", Mem: "240 MB"},
		{Name: "postgres-primary", Status: "live", CPU:  4.2, CPUStr: "  4.2%", Mem: "1.2 GB"},
		{Name: "redis-cluster",    Status: "warn", CPU: 28.1, CPUStr: " 28.1%", Mem: "380 MB"},
		{Name: "worker-pool",      Status: "live", CPU:  8.7, CPUStr: "  8.7%", Mem: "190 MB"},
		{Name: "cdn-edge",         Status: "live", CPU:  1.1, CPUStr: "  1.1%", Mem: " 42 MB"},
		{Name: "auth-service",     Status: "live", CPU:  6.3, CPUStr: "  6.3%", Mem: "128 MB"},
	}
	for i := range services {
		services[i].CPUHistory = make([]float64, 20)
		for j := range services[i].CPUHistory {
			services[i].CPUHistory[j] = services[i].CPU + rand.Float64()*5 - 2.5
		}
	}

	logLines := []string{
		fmt.Sprintf("%s  GET    /api/users         200   11ms", clock),
		fmt.Sprintf("%s  POST   /api/deploy        201  342ms", clock),
		fmt.Sprintf("%s  GET    /api/health        200    2ms", clock),
	}

	selectedSvc := services[0]
	showModal   := false
	restarting  := false
	spinnerFrame := 0

	view := VBox(
		HBox(
			Text("● ").FG(Cyan),
			Text("glyph control").FG(Cyan).Bold(),
			Space(),
			Text("prod-us-east-1  ").FG(BrightBlack),
			Text(&clock).FG(BrightBlack),
		),
		HRule().FG(BrightBlack),

		HBox.Gap(1)(
			VBox.Grow(1).Border(BorderSingle).BorderFG(BrightBlack).Title("requests/s")(
				Sparkline(&reqData).FG(Cyan),
				Text(&reqRate).FG(BrightBlack),
			),
			VBox.Grow(1).Border(BorderSingle).BorderFG(BrightBlack).Title("p99 latency")(
				Sparkline(&latData).FG(Green),
				Text(&p99Lat).FG(BrightBlack),
			),
			VBox.Grow(1).Border(BorderSingle).BorderFG(BrightBlack).Title("error rate")(
				Sparkline(&errData).FG(Yellow),
				Text(&errRate).FG(BrightBlack),
			),
		),

		VBox.Grow(1).Border(BorderSingle).BorderFG(BrightBlack).Title("services")(
			HBox.Gap(1)(
				Text("  ").FG(BrightBlack),
				Text(fmt.Sprintf("%-18s", "SERVICE")).FG(BrightBlack),
				Text("   CPU").FG(BrightBlack),
				Text("       MEM").FG(BrightBlack),
				Space(),
				Text("STATUS").FG(BrightBlack),
			),
			HRule().FG(BrightBlack),
			ForEach(&services, func(svc *service) any {
				return Jump(
					HBox.Gap(1)(
						Switch(&svc.Status).
							Case("warn", Text("○").FG(Yellow)).
							Default(Text("●").FG(Green)),
						Switch(&svc.Status).
							Case("warn", Text(&svc.Name).FG(Yellow)).
							Default(Text(&svc.Name).FG(Green)),
						IfOrd(&svc.CPU).Gt(20.0).
							Then(Text(&svc.CPUStr).FG(Yellow)).
							Else(Text(&svc.CPUStr).FG(BrightBlack)),
						Text(&svc.Mem).FG(BrightBlack),
						Space(),
						Switch(&svc.Status).
							Case("warn", Text("⚠ degraded").FG(Yellow)).
							Default(Text("  healthy").FG(BrightBlack)),
					),
					func() {},
				)
			}),
			Space(),
		),

		VBox.Border(BorderSingle).BorderFG(BrightBlack).Title("log")(
			ForEach(&logLines, func(l *string) any {
				return Text(l).FG(BrightBlack)
			}),
		),

		If(&showModal).Then(OverlayNode{
			Backdrop: true,
			Centered: true,
			Child: VBox.Width(46).Border(BorderRounded).BorderFG(BrightBlack)(
				HBox(
					Switch(&selectedSvc.Status).
						Case("warn", Text("○ ").FG(Yellow)).
						Default(Text("● ").FG(Green)),
					Switch(&selectedSvc.Status).
						Case("warn", Text(&selectedSvc.Name).FG(Yellow).Bold()).
						Default(Text(&selectedSvc.Name).FG(Green).Bold()),
					Space(),
					Text("esc  close").FG(BrightBlack),
				),
				HRule().FG(BrightBlack),
				Text("cpu history").FG(BrightBlack),
				Sparkline(&selectedSvc.CPUHistory).FG(Cyan),
				SpaceH(1),
				HBox.Gap(3)(
					VBox(
						Text("cpu").FG(BrightBlack),
						Text("mem").FG(BrightBlack),
					),
					VBox(
						IfOrd(&selectedSvc.CPU).Gt(20.0).
							Then(Text(&selectedSvc.CPUStr).FG(Yellow)).
							Else(Text(&selectedSvc.CPUStr).FG(Green)),
						Text(&selectedSvc.Mem).FG(White),
					),
				),
				HRule().FG(BrightBlack),
				If(&restarting).
					Then(HBox(Spinner(&spinnerFrame).FG(Cyan), Text("  restarting...").FG(BrightBlack))).
					Else(Text("[r] restart service").FG(BrightBlack)),
			),
		}),
	)

	tmpl := Build(view)
	buf := NewBuffer(120, 30)

	for i := 0; i < 5; i++ {
		reqData[len(reqData)-1] = float64(i+1) * 10
		buf.ClearDirty()
		tmpl.Execute(buf, 120, 30)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")

	t.Logf("output:\n%s", output)

	if len(lines) < 3 {
		t.Fatal("output too short")
	}

	panelBorderRow := lines[2]
	if !strings.ContainsAny(panelBorderRow, "┌╔+-") {
		t.Errorf("row 2 should be top border of sparkline panels; got: %q", panelBorderRow)
	}

	if len(lines) > 3 {
		sparklineRow := lines[3]
		if !strings.ContainsAny(sparklineRow, "▁▂▃▄▅▆▇█") {
			t.Errorf("row 3 should contain sparkline chars; got: %q", sparklineRow)
		}
		if strings.ContainsAny(sparklineRow, "┌╔") {
			t.Errorf("sparkline appears to be at border row; got: %q", sparklineRow)
		}
	}

	// service rows should contain both healthy and degraded status text
	if !strings.Contains(output, "healthy") {
		t.Error("expected healthy status text in service table")
	}
	if !strings.Contains(output, "degraded") {
		t.Error("expected degraded status text for redis-cluster")
	}

	// suppress unused variable warnings from the test scope
	_ = spinnerFrame
}
