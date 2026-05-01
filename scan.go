package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

const totalPorts = 65535

func scanPorts(ctx context.Context, host string, s *Session) ([]string, error) {
	slow := s.slowScan
	workers := cfg.ScanWorkers
	if slow {
		workers = cfg.ScanSlowWorkers
	}

	mode := "TCP dial"
	if slow {
		mode = "HTTP GET"
	}
	s.events <- Event{
		Kind:    KindLog,
		Level:   LevelInfo,
		Message: fmt.Sprintf("Escaneando todas as %d portas em %s (%d workers, modo: %s)...", totalPorts, host, workers, mode),
	}

	jobs := make(chan uint16, workers*2)
	var mu sync.Mutex
	var openPorts []int
	var scanned atomic.Int32
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				if ctx.Err() != nil {
					continue
				}
				open := false
				if slow {
					u := fmt.Sprintf("http://%s:%d", host, port)
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
					if err == nil {
						req.Header.Set("User-Agent", userAgent)
						client := &http.Client{
							Timeout: cfg.ScanSlowTimeout,
							CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
								return http.ErrUseLastResponse
							},
						}
						resp, err := client.Do(req)
						if err == nil {
							io.Copy(io.Discard, resp.Body)
							resp.Body.Close()
							open = true
						}
					}
				} else {
					addr := net.JoinHostPort(host, strconv.Itoa(int(port)))
					conn, err := net.DialTimeout("tcp", addr, cfg.ScanTimeout)
					if err == nil {
						conn.Close()
						open = true
					}
				}
				if open {
					mu.Lock()
					openPorts = append(openPorts, int(port))
					mu.Unlock()
					select {
					case s.events <- Event{Kind: KindLog, Level: LevelSuccess, Message: fmt.Sprintf("  %d/tcp aberta", port)}:
					case <-ctx.Done():
					}
				}
				interval := int32(5000)
				if slow {
					interval = 1000
				}
				if n := scanned.Add(1); n%interval == 0 {
					pct := int(n) * 100 / totalPorts
					select {
					case s.events <- Event{Kind: KindLog, Level: LevelInfo, Message: fmt.Sprintf("  progresso: %d/%d portas (%d%%)...", n, totalPorts, pct)}:
					default:
					}
				}
			}
		}()
	}

	for port := uint16(1); port < totalPorts; port++ {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case jobs <- port:
		}
	}
	jobs <- totalPorts
	close(jobs)
	wg.Wait()

	sort.Ints(openPorts)
	result := make([]string, len(openPorts))
	for i, p := range openPorts {
		result[i] = strconv.Itoa(p)
	}
	return result, nil
}
