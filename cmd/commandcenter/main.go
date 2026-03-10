// commandcenter: twitter demo — dense live dashboard with service drill-down
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
	p99Lat  := "24ms"
	errRate := "0.4%"
	clock   := time.Now().Format("15:04:05")

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

	var selectedPtr *service
	selectedSvc := services[0]
	showModal    := false
	restarting   := false
	spinnerFrame := 0

	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	app.JumpKey("g")

	app.SetView(
		VBox(
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
						func() {
							selectedPtr = svc
							selectedSvc = *svc
							showModal = true
							app.RequestRender()
						},
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
		),
	)

	app.Handle("q", app.Stop)

	app.Handle("<Escape>", func(_ riffkey.Match) {
		if showModal {
			showModal = false
			restarting = false
			app.RequestRender()
		}
	})

	app.Handle("r", func(_ riffkey.Match) {
		if !showModal || restarting {
			return
		}
		restarting = true
		go func() {
			for i := 0; i < 10; i++ {
				time.Sleep(200 * time.Millisecond)
				spinnerFrame++
				app.RequestRender()
			}
			restarting = false
			selectedSvc.Status = "live"
			if selectedPtr != nil {
				selectedPtr.Status = "live"
			}
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
			er  := rand.Float64() * 3

			copy(reqData, reqData[1:])
			reqData[len(reqData)-1] = rps
			copy(latData, latData[1:])
			latData[len(latData)-1] = lat
			copy(errData, errData[1:])
			errData[len(errData)-1] = er

			reqRate = fmt.Sprintf("%.0f req/s", rps)
			p99Lat  = fmt.Sprintf("%.0fms", lat)
			errRate = fmt.Sprintf("%.1f%%", er)
			clock   = time.Now().Format("15:04:05")

			for i := range services {
				services[i].CPU = math.Max(0.5, services[i].CPU+rand.Float64()*2-1)
				services[i].CPUStr = fmt.Sprintf("%5.1f%%", services[i].CPU)
				copy(services[i].CPUHistory, services[i].CPUHistory[1:])
				services[i].CPUHistory[len(services[i].CPUHistory)-1] = services[i].CPU
			}

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
