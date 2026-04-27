package main

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var portRe = regexp.MustCompile(`^(\d+)/tcp\s+open`)

func scanPorts(ctx context.Context, host string, s *Session) ([]string, error) {
	if _, err := exec.LookPath("nmap"); err != nil {
		return nil, fmt.Errorf("nmap não encontrado — instale com: sudo apt install nmap")
	}

	args := []string{"--top-ports", "1000", "--open", "-T4", host}
	s.events <- Event{Kind: KindLog, Level: LevelInfo, Message: "$ nmap " + strings.Join(args, " ")}

	cmd := exec.CommandContext(ctx, "nmap", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var ports []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case s.events <- Event{Kind: KindLog, Level: LevelInfo, Message: "  " + line}:
		case <-ctx.Done():
			cmd.Process.Kill()
			cmd.Wait()
			return nil, ctx.Err()
		}
		if m := portRe.FindStringSubmatch(line); len(m) > 1 {
			ports = append(ports, m[1])
		}
	}

	cmd.Wait()
	return ports, nil
}
