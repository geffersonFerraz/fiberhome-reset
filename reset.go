package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	routerHost  = "192.168.1.1"
	defaultPort = "8090"
	usernameB64 = "YWRtaW4="
	passwordB64 = "JTB8Rj9IQGYhYmVyaE8zZQ=="
	userAgent   = "Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0"
)

func newClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func runReset(ctx context.Context, s *Session) {
	defer func() {
		s.mu.Lock()
		s.running = false
		close(s.events)
		s.mu.Unlock()
	}()

	emit := func(kind Kind, level Level, msg string) {
		select {
		case s.events <- Event{Kind: kind, Level: level, Message: msg}:
		case <-ctx.Done():
		}
	}
	logf := func(level Level, format string, args ...any) {
		emit(KindLog, level, fmt.Sprintf(format, args...))
	}
	ask := func(msg string) bool {
		emit(KindQuestion, LevelWarning, msg)
		select {
		case ans := <-s.answer:
			return ans
		case <-time.After(5 * time.Minute):
			logf(LevelWarning, "Tempo de resposta esgotado. Operação cancelada.")
			return false
		case <-ctx.Done():
			return false
		}
	}
	finish := func(level Level, msg string) {
		emit(KindDone, level, msg)
	}

	// ── Step 1: Check login page ──────────────────────────────────────────────
	baseURL := "http://" + routerHost + ":" + defaultPort
	logf(LevelInfo, "[1] Verificando se a página de login está acessível em %s...", baseURL)

	if err := checkURL(ctx, baseURL); err != nil {
		logf(LevelError, "Falha ao acessar %s: %v", baseURL, err)

		if !ask("Deseja realizar um scan de portas em " + routerHost + " para localizar o serviço web do roteador?") {
			finish(LevelError, "Operação cancelada pelo usuário.")
			return
		}

		logf(LevelInfo, "[1.1] Iniciando scan de portas em %s...", routerHost)
		ports, err := scanPorts(ctx, routerHost, s)
		if err != nil {
			finish(LevelError, "Falha no scan de portas: "+err.Error())
			return
		}

		if len(ports) == 0 {
			finish(LevelError, "Nenhuma porta aberta encontrada em "+routerHost+". Verifique se o roteador está ligado e conectado à rede.")
			return
		}

		logf(LevelInfo, "Portas abertas: %s — testando páginas de login...", strings.Join(ports, ", "))

		found := ""
		for _, p := range ports {
			u := fmt.Sprintf("http://%s:%s", routerHost, p)
			logf(LevelInfo, "  → Testando %s...", u)
			if err := checkURL(ctx, u); err == nil {
				logf(LevelSuccess, "  ✓ Página encontrada em %s", u)
				found = u
				break
			} else {
				logf(LevelInfo, "  ✗ %s: sem resposta", u)
			}
		}

		if found == "" {
			finish(LevelError, "Nenhuma página de login encontrada nas portas abertas. O roteador pode estar inacessível.")
			return
		}
		baseURL = found
	} else {
		logf(LevelSuccess, "Página de login acessível.")
	}

	// ── Step 2: Login ─────────────────────────────────────────────────────────
	loginURL := baseURL + "/goform/webLogin"
	loginReferer := "http://" + routerHost + "/login_inter.asp"

	logf(LevelInfo, "[2] Realizando login em %s...", loginURL)

	body := "username=" + usernameB64 + "%3D&password=" + passwordB64 + "%3D"
	loginReq, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(body))
	if err != nil {
		finish(LevelError, "Falha ao montar requisição de login: "+err.Error())
		return
	}
	loginReq.Header.Set("User-Agent", userAgent)
	loginReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	loginReq.Header.Set("Accept-Language", "pt-BR")
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("Origin", "http://"+routerHost)
	loginReq.Header.Set("Referer", loginReferer)
	loginReq.Header.Set("Upgrade-Insecure-Requests", "1")
	loginReq.Header.Set("Pragma", "no-cache")
	loginReq.Header.Set("Cache-Control", "no-cache")

	loginResp, err := newClient(15 * time.Second).Do(loginReq)
	if err != nil {
		finish(LevelError, "Falha na requisição de login: "+err.Error())
		return
	}
	defer loginResp.Body.Close()
	io.Copy(io.Discard, loginResp.Body)

	fhstamp := extractFhstamp(loginResp.Cookies())
	if fhstamp == "" {
		finish(LevelError, fmt.Sprintf("Login falhou — cookie 'fhstamp' ausente na resposta (HTTP %d).", loginResp.StatusCode))
		return
	}
	logf(LevelSuccess, "Login realizado. Cookie: fhstamp=%s", fhstamp)

	// ── Step 3: Factory reset ─────────────────────────────────────────────────
	resetURL := baseURL + "/goform/adminRestore"
	logf(LevelInfo, "[3] Enviando comando de reset de fábrica para %s...", resetURL)

	resetReq, err := http.NewRequestWithContext(ctx, http.MethodPost, resetURL, strings.NewReader("n/a&x-csrftoken="+fhstamp))
	if err != nil {
		finish(LevelError, "Falha ao montar requisição de reset: "+err.Error())
		return
	}
	resetReq.Header.Set("User-Agent", userAgent)
	resetReq.Header.Set("Accept", "*/*")
	resetReq.Header.Set("Accept-Language", "pt-BR")
	resetReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resetReq.Header.Set("Origin", baseURL)
	resetReq.Header.Set("Referer", baseURL+"/management/adminRestore.asp")
	resetReq.Header.Set("Cookie", "fhstamp="+fhstamp)
	resetReq.Header.Set("Pragma", "no-cache")
	resetReq.Header.Set("Cache-Control", "no-cache")

	resetResp, err := newClient(30 * time.Second).Do(resetReq)
	if err != nil {
		if isConnectionReset(err) {
			logf(LevelSuccess, "Conexão encerrada pelo roteador — comportamento esperado ao iniciar o reset.")
			finish(LevelSuccess, "Reset iniciado! O roteador está restaurando as configurações de fábrica. Aguarde alguns minutos.")
			return
		}
		finish(LevelError, "Falha na requisição de reset: "+err.Error())
		return
	}
	defer resetResp.Body.Close()
	io.Copy(io.Discard, resetResp.Body)

	if resetResp.StatusCode == http.StatusOK {
		logf(LevelSuccess, "Comando aceito (HTTP %d).", resetResp.StatusCode)
		finish(LevelSuccess, "Reset iniciado! O roteador está restaurando as configurações de fábrica. Aguarde alguns minutos.")
	} else {
		finish(LevelError, fmt.Sprintf("Resposta inesperada do roteador (HTTP %d).", resetResp.StatusCode))
	}
}

func checkURL(ctx context.Context, u string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := newClient(8 * time.Second).Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

func extractFhstamp(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c.Name == "fhstamp" {
			return c.Value
		}
	}
	return ""
}

func isConnectionReset(err error) bool {
	s := err.Error()
	return strings.Contains(s, "EOF") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "forcibly closed")
}
