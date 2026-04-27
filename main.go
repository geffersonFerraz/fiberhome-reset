package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed web/index.html
var webFS embed.FS

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
	Kind    Kind   `json:"kind"`
	Level   Level  `json:"level"`
	Message string `json:"message"`
}

// ── Session ───────────────────────────────────────────────────────────────────

type Session struct {
	mu      sync.Mutex
	running bool
	events  chan Event
	answer  chan bool
	cancel  context.CancelFunc
}

var global = &Session{}

// ── Server ────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/api/start", handleStart)
	mux.HandleFunc("/api/events", handleEvents)
	mux.HandleFunc("/api/answer", handleAnswer)

	addr := ":8080"
	fmt.Printf("Interface web disponível em http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	data, _ := webFS.ReadFile("web/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	global.mu.Lock()
	if global.running {
		global.mu.Unlock()
		http.Error(w, "already running", http.StatusConflict)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	global.running = true
	global.events = make(chan Event, 256)
	global.answer = make(chan bool, 1)
	global.cancel = cancel
	global.mu.Unlock()

	go runReset(ctx, global)
	w.WriteHeader(http.StatusOK)
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	global.mu.Lock()
	events := global.events
	global.mu.Unlock()

	if events == nil {
		return
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, open := <-events:
			if !open {
				return
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func handleAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	accepted := strings.TrimSpace(string(body)) == "true"

	global.mu.Lock()
	ch := global.answer
	global.mu.Unlock()

	if ch == nil {
		http.Error(w, "no pending question", http.StatusBadRequest)
		return
	}

	select {
	case ch <- accepted:
	case <-time.After(5 * time.Second):
		http.Error(w, "timeout", http.StatusGatewayTimeout)
		return
	}

	w.WriteHeader(http.StatusOK)
}
