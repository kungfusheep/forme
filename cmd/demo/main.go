package main

import (
	"fmt"
	"log"
	"time"

	"tui"
)

func main() {
	// Create screen
	screen, err := tui.NewScreen(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create loop
	loop := tui.NewLoop(screen)

	// App state
	var (
		cursorX   = 5
		cursorY   = 5
		counter   = 0
		animValue = 0.0
		animating = false
		messages  []string
	)

	addMessage := func(msg string) {
		messages = append(messages, msg)
		if len(messages) > 10 {
			messages = messages[1:]
		}
	}

	addMessage("Press arrow keys or hjkl to move")
	addMessage("Press 'a' to start animation")
	addMessage("Press 'q' to quit")

	// Animation using CallLater
	var animate func()
	animate = func() {
		if !animating {
			return
		}
		animValue += 0.05
		if animValue > 1.0 {
			animValue = 0.0
		}
		counter++
		loop.RequestRender()

		// Schedule next frame (~60fps)
		time.AfterFunc(16*time.Millisecond, func() {
			loop.CallLater(animate)
		})
	}

	// Input handler
	loop.OnInput(func(input []byte) {
		for _, b := range input {
			switch b {
			case 'q':
				loop.Stop()
				return
			case 'h':
				cursorX--
				addMessage("Left (h)")
			case 'l':
				cursorX++
				addMessage("Right (l)")
			case 'j':
				cursorY++
				addMessage("Down (j)")
			case 'k':
				cursorY--
				addMessage("Up (k)")
			case 'a':
				animating = !animating
				if animating {
					addMessage("Animation started")
					loop.CallLater(animate)
				} else {
					addMessage("Animation stopped")
				}
			}
		}

		// Handle escape sequences for arrow keys
		if len(input) == 3 && input[0] == 27 && input[1] == '[' {
			switch input[2] {
			case 'A':
				cursorY--
				addMessage("Up arrow")
			case 'B':
				cursorY++
				addMessage("Down arrow")
			case 'C':
				cursorX++
				addMessage("Right arrow")
			case 'D':
				cursorX--
				addMessage("Left arrow")
			}
		}

		// Clamp cursor
		size := screen.Size()
		if cursorX < 0 {
			cursorX = 0
		}
		if cursorX >= size.Width {
			cursorX = size.Width - 1
		}
		if cursorY < 0 {
			cursorY = 0
		}
		if cursorY >= size.Height {
			cursorY = size.Height - 1
		}
	})

	// Resize handler
	loop.OnResize(func(size tui.Size) {
		addMessage(fmt.Sprintf("Resized to %dx%d", size.Width, size.Height))
	})

	// Render callback
	loop.OnRender(func(buf *tui.Buffer) {
		size := screen.Size()

		// Draw border around the screen
		buf.DrawBorder(0, 0, size.Width, size.Height, tui.BorderRounded, tui.DefaultStyle().Foreground(tui.Cyan))

		// Title
		title := " TUI Demo "
		titleX := (size.Width - len(title)) / 2
		buf.WriteString(titleX, 0, title, tui.DefaultStyle().Foreground(tui.Yellow).Bold())

		// Draw a colorful box
		boxStyle := tui.DefaultStyle().Background(tui.Blue).Foreground(tui.White)
		buf.FillRect(3, 3, 20, 5, tui.NewCell(' ', boxStyle))
		buf.WriteString(5, 4, "Colorful Box!", boxStyle.Bold())
		buf.WriteString(5, 6, fmt.Sprintf("Counter: %d", counter), boxStyle)

		// Draw cursor position indicator
		buf.Set(cursorX, cursorY, tui.NewCell('@', tui.DefaultStyle().Foreground(tui.Green).Bold()))

		// Animation indicator
		if animating {
			// Progress bar
			barWidth := 20
			filled := int(animValue * float64(barWidth))
			barY := 10
			buf.WriteString(3, barY, "[", tui.DefaultStyle().Foreground(tui.White))
			for i := 0; i < barWidth; i++ {
				style := tui.DefaultStyle().Background(tui.Green)
				if i >= filled {
					style = tui.DefaultStyle().Background(tui.BrightBlack)
				}
				buf.Set(4+i, barY, tui.NewCell(' ', style))
			}
			buf.WriteString(4+barWidth, barY, "]", tui.DefaultStyle().Foreground(tui.White))
			buf.WriteString(4+barWidth+2, barY, fmt.Sprintf("%.0f%%", animValue*100), tui.DefaultStyle().Foreground(tui.Yellow))
		}

		// Message log
		logX := size.Width - 35
		if logX < 30 {
			logX = 30
		}
		logY := 3
		buf.WriteString(logX, logY-1, "Messages:", tui.DefaultStyle().Foreground(tui.Magenta).Underline())
		for i, msg := range messages {
			style := tui.DefaultStyle().Foreground(tui.White).Dim()
			if i == len(messages)-1 {
				style = tui.DefaultStyle().Foreground(tui.White)
			}
			buf.WriteStringClipped(logX, logY+i, msg, style, 30)
		}

		// Color palette demo
		paletteY := size.Height - 5
		buf.WriteString(3, paletteY-1, "Colors:", tui.DefaultStyle().Foreground(tui.White))
		colors := []tui.Color{
			tui.Red, tui.Green, tui.Yellow, tui.Blue,
			tui.Magenta, tui.Cyan, tui.White,
			tui.BrightRed, tui.BrightGreen, tui.BrightYellow, tui.BrightBlue,
		}
		for i, c := range colors {
			buf.Set(3+i*2, paletteY, tui.NewCell(' ', tui.DefaultStyle().Background(c)))
			buf.Set(4+i*2, paletteY, tui.NewCell(' ', tui.DefaultStyle().Background(c)))
		}

		// RGB gradient
		buf.WriteString(3, paletteY+2, "RGB Gradient:", tui.DefaultStyle().Foreground(tui.White))
		for i := 0; i < 24; i++ {
			r := uint8(255 - i*10)
			g := uint8(i * 10)
			b := uint8(128)
			buf.Set(3+i, paletteY+3, tui.NewCell(' ', tui.DefaultStyle().Background(tui.RGB(r, g, b))))
		}

		// Status bar
		statusY := size.Height - 1
		status := fmt.Sprintf(" Pos: (%d,%d) | Size: %dx%d | Press 'q' to quit ", cursorX, cursorY, size.Width, size.Height)
		buf.WriteString(1, statusY, status, tui.DefaultStyle().Foreground(tui.Black).Background(tui.White))
	})

	// Run the loop
	addMessage("Starting...")
	if err := loop.Run(); err != nil {
		log.Fatal(err)
	}
}
