package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ScanWorkers     int
	ScanTimeout     time.Duration
	ScanSlowWorkers int
	ScanSlowTimeout time.Duration
	SlowScan        bool
}

var cfg = Config{
	ScanWorkers:     2000,
	ScanTimeout:     400 * time.Millisecond,
	ScanSlowWorkers: 2000,
	ScanSlowTimeout: 600 * time.Millisecond,
	SlowScan:        false,
}

func loadConfig() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	f, err := os.Open(filepath.Join(filepath.Dir(exe), "conf.g"))
	if err != nil {
		return
	}
	defer f.Close()
	parseConfig(f, &cfg)
}

func parseConfig(r io.Reader, c *Config) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "scanWorkers":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				c.ScanWorkers = n
			}
		case "scanTimeout":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				c.ScanTimeout = time.Duration(n) * time.Millisecond
			}
		case "scanSlowWorkers":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				c.ScanSlowWorkers = n
			}
		case "scanSlowTimeout":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				c.ScanSlowTimeout = time.Duration(n) * time.Millisecond
			}
		case "slowScan":
			c.SlowScan = val == "true"
		}
	}
}
