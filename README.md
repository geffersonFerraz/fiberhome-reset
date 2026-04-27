# reset-fiber-home

Ferramenta web para resetar roteadores Fiber Home para as configurações de fábrica.

## Uso

```bash
./reset-fiber-home
```

Acesse `http://localhost:8080` no navegador, clique em **Iniciar Reset** e siga as instruções na tela.

### Fluxo

1. Verifica se a página de login do roteador (`192.168.1.1:8090`) está acessível
2. Se não estiver, oferece a opção de fazer um **scan de portas** com nmap para localizar o serviço
3. Realiza o login e envia o comando de reset de fábrica
4. Exibe o resultado em tempo real

## Dependências

- **nmap** — necessário apenas se o scan de portas for utilizado

```bash
# Debian/Ubuntu
sudo apt install nmap
```

## Release

Os binários para Linux e Windows estão disponíveis na [página de releases](../../releases).

| Plataforma | Arquivo |
|---|---|
| Linux x86-64 | `reset-fiber-home_*_linux_amd64.tar.gz` |
| Linux ARM64  | `reset-fiber-home_*_linux_arm64.tar.gz` |
| Windows x86-64 | `reset-fiber-home_*_windows_amd64.zip` |

## Build

```bash
go build -buildvcs=false -o reset-fiber-home .
```
