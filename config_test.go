package main

import (
	"strings"
	"testing"
	"time"
)

func defaultCfg() Config {
	return Config{
		ScanWorkers:     2000,
		ScanTimeout:     400 * time.Millisecond,
		ScanSlowWorkers: 2000,
		ScanSlowTimeout: 600 * time.Millisecond,
		SlowScan:        false,
	}
}

func TestParseConfig_AllFields(t *testing.T) {
	input := `
# comentario ignorado

scanWorkers=500
scanTimeout=200
scanSlowWorkers=100
scanSlowTimeout=1500
slowScan=true
`
	c := defaultCfg()
	parseConfig(strings.NewReader(input), &c)

	if c.ScanWorkers != 500 {
		t.Errorf("ScanWorkers: got %d, want 500", c.ScanWorkers)
	}
	if c.ScanTimeout != 200*time.Millisecond {
		t.Errorf("ScanTimeout: got %v, want 200ms", c.ScanTimeout)
	}
	if c.ScanSlowWorkers != 100 {
		t.Errorf("ScanSlowWorkers: got %d, want 100", c.ScanSlowWorkers)
	}
	if c.ScanSlowTimeout != 1500*time.Millisecond {
		t.Errorf("ScanSlowTimeout: got %v, want 1500ms", c.ScanSlowTimeout)
	}
	if !c.SlowScan {
		t.Error("SlowScan: got false, want true")
	}
}

func TestParseConfig_SlowScanFalse(t *testing.T) {
	c := defaultCfg()
	c.SlowScan = true
	parseConfig(strings.NewReader("slowScan=false"), &c)
	if c.SlowScan {
		t.Error("SlowScan: got true, want false")
	}
}

func TestParseConfig_EmptyFile(t *testing.T) {
	c := defaultCfg()
	parseConfig(strings.NewReader(""), &c)
	if c.ScanWorkers != 2000 {
		t.Errorf("ScanWorkers deve manter padrão: got %d", c.ScanWorkers)
	}
}

func TestParseConfig_CommentsAndBlanks(t *testing.T) {
	input := `
# isto é um comentário
   # outro comentário com espaço

`
	c := defaultCfg()
	parseConfig(strings.NewReader(input), &c)
	if c.ScanWorkers != 2000 {
		t.Errorf("ScanWorkers deve manter padrão: got %d", c.ScanWorkers)
	}
}

func TestParseConfig_InvalidValues(t *testing.T) {
	input := `
scanWorkers=abc
scanTimeout=-10
scanSlowWorkers=0
scanSlowTimeout=
`
	c := defaultCfg()
	parseConfig(strings.NewReader(input), &c)

	if c.ScanWorkers != 2000 {
		t.Errorf("ScanWorkers deve manter padrão com valor inválido: got %d", c.ScanWorkers)
	}
	if c.ScanTimeout != 400*time.Millisecond {
		t.Errorf("ScanTimeout deve manter padrão com valor negativo: got %v", c.ScanTimeout)
	}
	if c.ScanSlowWorkers != 2000 {
		t.Errorf("ScanSlowWorkers deve manter padrão com valor zero: got %d", c.ScanSlowWorkers)
	}
}

func TestParseConfig_PartialOverride(t *testing.T) {
	c := defaultCfg()
	parseConfig(strings.NewReader("scanWorkers=300"), &c)

	if c.ScanWorkers != 300 {
		t.Errorf("ScanWorkers: got %d, want 300", c.ScanWorkers)
	}
	if c.ScanTimeout != 400*time.Millisecond {
		t.Errorf("ScanTimeout deve manter padrão: got %v", c.ScanTimeout)
	}
}

func TestParseConfig_UnknownKey(t *testing.T) {
	c := defaultCfg()
	parseConfig(strings.NewReader("chaveDesconhecida=999"), &c)
	if c.ScanWorkers != 2000 {
		t.Errorf("ScanWorkers deve manter padrão: got %d", c.ScanWorkers)
	}
}
