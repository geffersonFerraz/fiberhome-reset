package main

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ── Event types ───────────────────────────────────────────────────────────────

type Kind string
type Level string

const (
	KindLog      Kind  = "log"
	KindQuestion Kind  = "question"
	KindDone     Kind  = "done"

	LevelInfo    Level = "info"
	LevelSuccess Level = "success"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

type Event struct {
	Kind    Kind
	Level   Level
	Message string
}

// ── Session ───────────────────────────────────────────────────────────────────

type Session struct {
	mu       sync.Mutex
	running  bool
	events   chan Event
	answer   chan bool
	cancel   context.CancelFunc
	slowScan bool
}

var global = &Session{}

// ── UI ────────────────────────────────────────────────────────────────────────

func main() {
	loadConfig()

	a := app.New()
	w := a.NewWindow("Reset Fiber Home")
	w.Resize(fyne.NewSize(720, 520))

	logWidget := widget.NewRichText()
	logWidget.Wrapping = fyne.TextWrapWord

	scroll := container.NewScroll(logWidget)

	global.slowScan = cfg.SlowScan
	slowCheck := widget.NewCheck("Scan lento (HTTP GET, mais preciso porém mais lento)", func(checked bool) {
		global.mu.Lock()
		global.slowScan = checked
		global.mu.Unlock()
	})
	slowCheck.Checked = cfg.SlowScan

	var startBtn *widget.Button
	startBtn = widget.NewButton("Iniciar Reset", func() {
		startBtn.Disable()
		logWidget.Segments = nil
		logWidget.Refresh()

		global.mu.Lock()
		if global.running {
			global.mu.Unlock()
			startBtn.Enable()
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		global.running = true
		global.events = make(chan Event, 256)
		global.answer = make(chan bool, 1)
		global.cancel = cancel
		global.mu.Unlock()

		go runReset(ctx, global)
		go consumeEvents(w, logWidget, scroll, startBtn)
	})

	bottom := container.NewVBox(slowCheck, container.NewPadded(startBtn))
	w.SetContent(container.NewBorder(nil, bottom, nil, nil, scroll))
	w.ShowAndRun()
}

func levelColorName(level Level) fyne.ThemeColorName {
	switch level {
	case LevelSuccess:
		return theme.ColorNameSuccess
	case LevelError:
		return theme.ColorNameError
	case LevelWarning:
		return theme.ColorNameWarning
	default:
		return theme.ColorNameForeground
	}
}

func appendLog(richText *widget.RichText, scroll *container.Scroll, ev Event) {
	fyne.Do(func() {
		richText.Segments = append(richText.Segments, &widget.TextSegment{
			Text:  ev.Message + "\n",
			Style: widget.RichTextStyle{ColorName: levelColorName(ev.Level)},
		})
		richText.Refresh()
		scroll.ScrollToBottom()
	})
}

func consumeEvents(w fyne.Window, richText *widget.RichText, scroll *container.Scroll, btn *widget.Button) {
	for ev := range global.events {
		switch ev.Kind {
		case KindQuestion:
			done := make(chan struct{})
			fyne.Do(func() {
				dialog.NewConfirm("Confirmação", ev.Message, func(ok bool) {
					global.answer <- ok
					close(done)
				}, w).Show()
			})
			<-done
		case KindDone:
			appendLog(richText, scroll, ev)
			fyne.Do(func() { btn.Enable() })
		default:
			appendLog(richText, scroll, ev)
		}
	}
	fyne.Do(func() { btn.Enable() })
}
