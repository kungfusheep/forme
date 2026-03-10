// commandcenter: dense live dashboard demo with service drill-down
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	. "github.com/kungfusheep/glyph"
	"github.com/kungfusheep/riffkey"
)

type service struct {
	Name       string
	Status     string
	CPU        float64
	CPUStr     string
	Mem        string
	CPUHistory []float64
}

func main() {
	reqData := make([]float64, 40)
	latData := make([]float64, 40)
	errData := make([]float64, 40)
	for i := range reqData {
		reqData[i] = 80 + rand.Float64()*60 + 30*math.Sin(float64(i)*0.2)
		latData[i] = 18 + rand.Float64()*12 + 6*math.Sin(float64(i)*0.15)
		errData[i] = rand.Float64() * 3
	}

	reqRate := "142 req/s"
	p99Lat := "24ms"
	errRate := "0.4%"
	clock := time.Now().Format("15:04:05")

	services := []service{
		{Name: "api-gateway", Status: "live", CPU: 7.2, CPUStr: "  7.2%", Mem: "240 MB"},
		{Name: "postgres-primary", Status: "live", CPU: 3.8, CPUStr: "  3.8%", Mem: "1.2 GB"},
		{Name: "redis-cluster", Status: "warn", CPU: 6.1, CPUStr: "  6.1%", Mem: "380 MB"},
		{Name: "worker-pool", Status: "live", CPU: 4.9, CPUStr: "  4.9%", Mem: "190 MB"},
		{Name: "cdn-edge", Status: "live", CPU: 1.1, CPUStr: "  1.1%", Mem: " 42 MB"},
		{Name: "auth-service", Status: "live", CPU: 5.4, CPUStr: "  5.4%", Mem: "128 MB"},
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

	var selectedPtr *service
	selectedSvc := services[0]
	showModal := false
	restarting := false
	restartPct := 0

	pulseStyle := Style{}
	var modalRouter *riffkey.Router
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	app.JumpKey("g")

	// helper: metric panel with sparkline + label, grows equally in an HBox
	metricPanel := func(title string, data *[]float64, label *string, col Color) any {
		return VBox.Grow(1).Border(BorderRounded).BorderFG(BrightBlack).Title(title)(
			Sparkline(data).FG(col),
			Text(label).FG(BrightBlack),
		)
	}

	svcList := FilterList(&services, func(s *service) string { return s.Name }).
		Placeholder("filter...").
		Render(func(svc *service) any {
			return HBox.Gap(2)(
				VBox.Width(1)(Switch(&svc.Status).
					Case("warn", Text("○").FG(BrightBlack)).
					Default(Text("●").FG(BrightBlack))),
				Text(&svc.Name),
				Space(),
				Text(&svc.CPUStr).FG(BrightBlack).Width(6).Align(AlignRight),
				Text(&svc.Mem).FG(BrightBlack).Width(8).Align(AlignRight),
				VBox.Width(11)(Switch(&svc.Status).
					Case("warn", Text("degraded").FG(Yellow)).
					Default(Text("healthy").FG(BrightBlack))),
			)
		}).
		Handle("<Enter>", func(svc *service) {
			selectedPtr = svc
			selectedSvc = *svc
			showModal = true
			app.Push(modalRouter)
			app.RequestRender()
		}).
		HandleClear("<Esc>", nil)

	app.SetView(
		VBox.MarginVH(1, 2)(
			HBox.CascadeStyle(&Style{FG: BrightBlack})(
				Text("● glyph control"),
				Space(),
				Text("prod-us-east-1  "),
				Text(&clock),
			),
			HRule().Char(BorderDouble.Horizontal).FG(BrightBlack),

			HBox.Gap(1)(
				metricPanel("requests/s", &reqData, &reqRate, Cyan),
				metricPanel("p99 latency", &latData, &p99Lat, Green),
				metricPanel("error rate", &errData, &errRate, Yellow),
			),

			HBox.Grow(1).Gap(1)(
				VBox.Grow(1).Border(BorderRounded).BorderFG(BrightBlack).Title("services")(
					HBox.Gap(2)(
						Text("●").FG(BrightBlack),
						Text("SERVICE").FG(BrightBlack),
						Space(),
						Text("CPU").FG(BrightBlack).Width(6).Align(AlignRight),
						Text("MEM").FG(BrightBlack).Width(8).Align(AlignRight),
						Text("STATUS").FG(BrightBlack).Width(11),
					),
					HRule().FG(BrightBlack).Extend(),
					svcList,
				),
				VBox.Border(BorderRounded).BorderFG(BrightBlack).Title("log")(
					ForEach(&logLines, func(l *string) any {
						return Text(l).FG(BrightBlack)
					}),
				),
			),

			HRule().Char(BorderDouble.Horizontal).FG(BrightBlack),
			Text("press [ctrl+c] to quit  [enter] to inspect  [r] restart (modal)").FG(BrightBlack),

			If(&showModal).Then(OverlayNode{
				Backdrop: true,
				BG:       Black,
				Child: VBox.Width(46).MarginVH(1, 2).Fill(Black)(
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
								Else(Text(&selectedSvc.CPUStr).FG(White)),
							Text(&selectedSvc.Mem).FG(White),
						),
					),
					HRule().FG(BrightBlack),
					If(&restarting).
						Then(HBox.Gap(1)(Text("restarting").FG(BrightBlack), HBox.Grow(1).CascadeStyle(&pulseStyle)(Progress(&restartPct)))).
						Else(Text("[r] restart service").FG(BrightBlack)),
				),
			}),
		),
	)


	app.Handle("<C-c>", func(_ riffkey.Match) { app.Stop() })

	app.Handle("<Escape>", func(_ riffkey.Match) {
		if showModal {
			showModal = false
			restarting = false
			app.Pop()
			app.RequestRender()
		}
	})

	modalRouter = riffkey.NewRouter()
	modalRouter.Handle("r", func(_ riffkey.Match) {
		if restarting {
			return
		}
		restarting = true
		restartPct = 0
		go func() {
			for i := 1; i <= 50; i++ {
				time.Sleep(40 * time.Millisecond)
				restartPct = i * 2
				t := float64(restartPct) / 100.0
				b := math.Pow(t, 1.5)
				pulseStyle.FG = RGB(uint8(30+b*80), uint8(120+b*100), uint8(110+b*90))
				app.RequestRender()
			}
			restarting = false
			restartPct = 0
			selectedSvc.Status = "live"
			if selectedPtr != nil {
				selectedPtr.Status = "live"
			}
			showModal = false
			app.Pop()
			app.RequestRender()
		}()
	})

	tick := 0
	go func() {
		for range time.NewTicker(400 * time.Millisecond).C {
			tick++
			t := float64(tick)

			rps := 80 + rand.Float64()*60 + 30*math.Sin(t*0.2)
			lat := 18 + rand.Float64()*12 + 6*math.Sin(t*0.15)
			er := rand.Float64() * 3

			copy(reqData, reqData[1:])
			reqData[len(reqData)-1] = rps
			copy(latData, latData[1:])
			latData[len(latData)-1] = lat
			copy(errData, errData[1:])
			errData[len(errData)-1] = er

			reqRate = fmt.Sprintf("%.0f req/s", rps)
			p99Lat = fmt.Sprintf("%.0fms", lat)
			errRate = fmt.Sprintf("%.1f%%", er)
			clock = time.Now().Format("15:04:05")

			for i := range services {
				services[i].CPU = math.Max(0.5, services[i].CPU+rand.Float64()*2-1)
				services[i].CPUStr = fmt.Sprintf("%5.1f%%", services[i].CPU)
				copy(services[i].CPUHistory, services[i].CPUHistory[1:])
				services[i].CPUHistory[len(services[i].CPUHistory)-1] = services[i].CPU
			}
			svcList.Refresh()

			line := fmt.Sprintf("%s  %-6s %-22s %d  %dms",
				clock,
				[]string{"GET", "POST", "GET", "PUT", "DELETE"}[rand.Intn(5)],
				[]string{"/api/users", "/api/deploy", "/api/health", "/api/orders", "/api/metrics"}[rand.Intn(5)],
				[]int{200, 200, 200, 201, 204, 304, 400, 404}[rand.Intn(8)],
				2+rand.Intn(340),
			)
			logLines = append(logLines[1:], line)

			app.RequestRender()
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
