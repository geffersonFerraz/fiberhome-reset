package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	scanWorkers = 2000
	scanTimeout = 400 * time.Millisecond
	totalPorts  = 65535
)

func scanPorts(ctx context.Context, host string, s *Session) ([]string, error) {
	s.events <- Event{
		Kind:    KindLog,
		Level:   LevelInfo,
		Message: fmt.Sprintf("Escaneando todas as %d portas em %s (%d workers simultâneos)...", totalPorts, host, scanWorkers),
	}

	jobs := make(chan uint16, scanWorkers*2)
	var mu sync.Mutex
	var openPorts []int
	var scanned atomic.Int32
	var wg sync.WaitGroup

	for range scanWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				if ctx.Err() != nil {
					continue
				}
				addr := net.JoinHostPort(host, strconv.Itoa(int(port)))
				conn, err := net.DialTimeout("tcp", addr, scanTimeout)
				if err == nil {
					conn.Close()
					mu.Lock()
					openPorts = append(openPorts, int(port))
					mu.Unlock()
					select {
					case s.events <- Event{Kind: KindLog, Level: LevelSuccess, Message: fmt.Sprintf("  %d/tcp aberta", port)}:
					case <-ctx.Done():
					}
				}
				if n := scanned.Add(1); n%5000 == 0 {
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
