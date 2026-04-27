#!/bin/bash

set -euo pipefail

ROUTER_HOST="192.168.1.1"
ROUTER_PORT="8090"
BASE_URL="http://${ROUTER_HOST}:${ROUTER_PORT}"
LOGIN_URL="${BASE_URL}/goform/webLogin"
RESET_URL="${BASE_URL}/goform/adminRestore"
LOGIN_PAGE="http://${ROUTER_HOST}/login_inter.asp"

USERNAME_B64="YWRtaW4="
PASSWORD_B64="JTB8Rj9IQGYhYmVyaE8zZQ=="

COOKIE_FILE=$(mktemp)
trap 'rm -f "$COOKIE_FILE"' EXIT

echo "[1] Verificando se a página de login está acessível..."

if ! curl -sf --max-time 10 "${BASE_URL}" -o /dev/null; then
    echo "ERRO: Não foi possível acessar ${BASE_URL}. Verifique se o roteador está ligado e acessível."
    exit 1
fi

echo "    OK - Página carregada com sucesso."

echo "[2] Realizando login..."

LOGIN_RESPONSE=$(curl -si --max-time 15 \
    -X POST \
    -H "User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0" \
    -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" \
    -H "Accept-Language: pt-BR" \
    -H "Accept-Encoding: identity" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "Origin: http://${ROUTER_HOST}" \
    -H "Connection: keep-alive" \
    -H "Referer: ${LOGIN_PAGE}" \
    -H "Upgrade-Insecure-Requests: 1" \
    -H "Pragma: no-cache" \
    -H "Cache-Control: no-cache" \
    -c "$COOKIE_FILE" \
    --data-raw "username=${USERNAME_B64}%3D&password=${PASSWORD_B64}%3D" \
    "${LOGIN_URL}" 2>&1)

SESSION_COOKIE=$(grep -i 'set-cookie' <<< "$LOGIN_RESPONSE" | grep -o 'fhstamp=[^;]*' | head -1 || true)

if [[ -z "$SESSION_COOKIE" ]]; then
    echo "ERRO: Login falhou. Cookie de sessão não encontrado na resposta."
    echo "--- Resposta do servidor ---"
    echo "$LOGIN_RESPONSE"
    exit 1
fi

echo "    OK - Login realizado. Cookie: ${SESSION_COOKIE}"

echo "[3] Enviando comando de reset de fábrica..."

RESET_RESPONSE=$(curl -si --max-time 30 \
    -X POST \
    -H "User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0" \
    -H "Accept: */*" \
    -H "Accept-Language: pt-BR" \
    -H "Accept-Encoding: identity" \
    -H "Content-type: application/x-www-form-urlencoded" \
    -H "Origin: ${BASE_URL}" \
    -H "Connection: keep-alive" \
    -H "Referer: ${BASE_URL}/management/adminRestore.asp" \
    -H "Cookie: ${SESSION_COOKIE}" \
    -H "Pragma: no-cache" \
    -H "Cache-Control: no-cache" \
    --data-raw "n/a&x-csrftoken=${SESSION_COOKIE#fhstamp=}" \
    "${RESET_URL}" 2>&1)

HTTP_STATUS=$(grep -o 'HTTP/[^ ]* [0-9]*' <<< "$RESET_RESPONSE" | tail -1 | grep -o '[0-9]*$' || echo "0")

if [[ "$HTTP_STATUS" == "200" ]]; then
    echo "    OK - Reset enviado com sucesso (HTTP ${HTTP_STATUS})."
    echo ""
    echo "O roteador está sendo resetado para as configurações de fábrica."
    echo "Aguarde alguns minutos até que ele reinicie completamente."
else
    echo "AVISO: Resposta inesperada (HTTP ${HTTP_STATUS}). Verifique manualmente."
    echo "--- Resposta do servidor ---"
    echo "$RESET_RESPONSE"
    exit 1
fi
