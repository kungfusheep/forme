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
		{Name: "api-gateway",      Status: "live", CPU:  7.2, CPUStr: "  7.2%", Mem: "240 MB"},
		{Name: "postgres-primary", Status: "live", CPU:  3.8, CPUStr: "  3.8%", Mem: "1.2 GB"},
		{Name: "redis-cluster",    Status: "warn", CPU:  6.1, CPUStr: "  6.1%", Mem: "380 MB"},
		{Name: "worker-pool",      Status: "live", CPU:  4.9, CPUStr: "  4.9%", Mem: "190 MB"},
		{Name: "cdn-edge",         Status: "live", CPU:  1.1, CPUStr: "  1.1%", Mem: " 42 MB"},
		{Name: "auth-service",     Status: "live", CPU:  5.4, CPUStr: "  5.4%", Mem: "128 MB"},
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

	metricPanel := func(title string, data *[]float64, label *string, col Color) any {
		return VBox.Grow(1).Border(BorderSingle).BorderFG(BrightBlack).Title(title)(
			Sparkline(data).FG(col),
			Text(label).FG(BrightBlack),
		)
	}

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
			metricPanel("requests/s", &reqData, &reqRate, Cyan),
			metricPanel("p99 latency", &latData, &p99Lat, Green),
			metricPanel("error rate", &errData, &errRate, Yellow),
		),

		VBox.Grow(1).Border(BorderSingle).BorderFG(BrightBlack).Title("services")(
			HBox.Gap(2)(
				Text("●").FG(BrightBlack),
				Text("SERVICE").FG(BrightBlack),
				Space(),
				Text("CPU").FG(BrightBlack).Width(6).Align(AlignRight),
				Text("MEM").FG(BrightBlack).Width(8).Align(AlignRight),
				Text("STATUS").FG(BrightBlack).Width(11),
			),
			HRule().FG(BrightBlack),
			ForEach(&services, func(svc *service) any {
				return Jump(
					HBox.Gap(2)(
						VBox.Width(1)(Switch(&svc.Status).
							Case("warn", Text("○").FG(Yellow)).
							Default(Text("●").FG(Green))),
						Text(&svc.Name),
						Space(),
						IfOrd(&svc.CPU).Gt(20.0).
							Then(Text(&svc.CPUStr).FG(Yellow).Width(6).Align(AlignRight)).
							Else(Text(&svc.CPUStr).FG(BrightBlack).Width(6).Align(AlignRight)),
						Text(&svc.Mem).FG(BrightBlack).Width(8).Align(AlignRight),
						VBox.Width(11)(Switch(&svc.Status).
							Case("warn", Text("⚠ degraded").FG(Yellow)).
							Default(Text("healthy").FG(BrightBlack))),
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
					If(&selectedSvc.Status).Eq("warn").
						Then(Text("○ ").FG(Yellow)).
						Else(Text("● ").FG(Green)),
					If(&selectedSvc.Status).Eq("warn").
						Then(Text(&selectedSvc.Name).FG(Yellow).Bold()).
						Else(Text(&selectedSvc.Name).FG(Green).Bold()),
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
