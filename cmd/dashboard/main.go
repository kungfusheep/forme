package main

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"time"

	. "github.com/kungfusheep/glyph"
)

func main() {
	var (
		reqData = make([]float64, 60)
		latData = make([]float64, 60)
		reqRate = "0 req/s"
		p99     = "0ms"
	)

	pr, pw := io.Pipe()

	app, _ := NewApp()
	app.SetView(
		VBox(
			HBox.Gap(1)(
				VBox.Grow(1).Border(BorderRounded).Title("requests/s")(
					Sparkline(&reqData).FG(Green),
					Text(&reqRate).FG(BrightBlack),
				),
				VBox.Grow(1).Border(BorderRounded).Title("p99 latency")(
					Sparkline(&latData).FG(Yellow),
					Text(&p99).FG(BrightBlack),
				),
			),
			FilterLog(pr).Grow(1).MaxLines(200),
		),
	).Router().NoCounts()

	endpoints := []string{
		"GET /api/users",
		"POST /api/orders",
		"GET /api/products",
		"PUT /api/inventory",
		"GET /api/health",
		"POST /api/auth/login",
		"DELETE /api/sessions",
	}
	statuses := []int{200, 200, 200, 200, 201, 204, 304, 400, 404, 500}

	go func() {
		i := 0
		for range time.NewTicker(500 * time.Millisecond).C {
			// shift data left, append new point
			rps := 80 + rand.Float64()*40 + 20*math.Sin(float64(i)*0.1)
			lat := 12 + rand.Float64()*8 + 5*math.Sin(float64(i)*0.15)

			copy(reqData, reqData[1:])
			reqData[len(reqData)-1] = rps
			copy(latData, latData[1:])
			latData[len(latData)-1] = lat

			reqRate = fmt.Sprintf("%.0f req/s", rps)
			p99 = fmt.Sprintf("%.1fms", lat)

			ep := endpoints[rand.Intn(len(endpoints))]
			status := statuses[rand.Intn(len(statuses))]
			dur := 2 + rand.Float64()*15
			fmt.Fprintf(pw, "%s  %s  %d  %.1fms\n",
				time.Now().Format("15:04:05"), ep, status, dur)

			i++
			app.RequestRender()
		}
	}()

	app.Handle("q", app.Stop)
	app.Run()
}
