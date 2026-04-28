package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	routerHost  = "192.168.1.1"
	defaultPort = "8090"
	userAgent   = "Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0"
)

type credential struct {
	user string
	pass string
}

var credentials = []credential{
	{"admin", "%0|F?H@f!berhO3e"},
	{"user", "user1234"},
	{"f~i!b@e#r$h%o^m*esuperadmin", "s(f)u_h+g|u"},
	{"admin", "lnadmin"},
	{"admin", "CUadmin"},
	{"admin", "admin"},
	{"telecomadmin", "nE7jA%5m"},
	{"adminpldt", "z6dUABtl270qRxt7a2uGTiw"},
	{"gestiontelebucaramanga", "t3l3buc4r4m4ng42013"},
	{"rootmet", "m3tr0r00t"},
	{"awnfibre", "fibre@dm!n"},
	{"trueadmin", "admintrue"},
	{"admin", "G0R2U1P2ag"},
	{"admin", "3UJUh2VemEfUtesEchEC2d2e"},
	{"admin", "888888"},
	{"L1vt1m4eng", "888888"},
	{"useradmin", "888888"},
	{"user", "888888"},
	{"admin", "1234"},
	{"user", "tattoo@home"},
	{"admin", "tele1234"},
	{"admin", "aisadmin"},
}

func b64Param(s string) string {
	return strings.ReplaceAll(base64.StdEncoding.EncodeToString([]byte(s)), "=", "%3D")
}

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

		if !ask("Deseja realizar um scan de todas as portas em " + routerHost + " para localizar o serviço web do roteador?") {
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
			finish(LevelError, "Nenhuma página de login encontrada nas portas abertas.")
			return
		}
		baseURL = found
	} else {
		logf(LevelSuccess, "Página de login acessível.")
	}

	// ── Step 2: Login ─────────────────────────────────────────────────────────
	loginURL := baseURL + "/goform/webLogin"
	loginReferer := "http://" + routerHost + "/login_inter.asp"

	logf(LevelInfo, "[2] Tentando %d combinações de credenciais em %s...", len(credentials), loginURL)

	var fhstamp string
	for i, cred := range credentials {
		if ctx.Err() != nil {
			return
		}
		logf(LevelInfo, "  [%d/%d] usuário: %s", i+1, len(credentials), cred.user)
		stamp, err := tryLogin(ctx, loginURL, loginReferer, cred)
		if err != nil {
			logf(LevelWarning, "    ✗ erro na requisição: %v", err)
			continue
		}
		if stamp != "" {
			logf(LevelSuccess, "    ✓ Login OK com %q / %q", cred.user, cred.pass)
			fhstamp = stamp
			break
		}
		logf(LevelInfo, "    ✗ credenciais inválidas")
	}

	if fhstamp == "" {
		finish(LevelError, fmt.Sprintf("Nenhuma das %d combinações funcionou.", len(credentials)))
		return
	}

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

func tryLogin(ctx context.Context, loginURL, referer string, cred credential) (string, error) {
	body := "username=" + b64Param(cred.user) + "&password=" + b64Param(cred.pass)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://"+routerHost)
	req.Header.Set("Referer", referer)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := newClient(10 * time.Second).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return extractFhstamp(resp.Cookies()), nil
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
