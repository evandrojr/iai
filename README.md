# iai — Install by AI

`iai` (Install by AI) ajuda a instalar pacotes de forma segura no Arch Linux usando um LLM para descoberta/validação, mas com **verificação determinística de ameaças antes de qualquer instalação**. 

**Regra:** O LLM nunca decide segurança sozinho — ele sugere, o código verifica.

## Funcionalidades

- **Busca unificada**: `pacman` (repositórios oficiais) + AUR RPC
- **Auditoria de pacotes AUR**: Análise determinística do PKGBUILD por regras + análise via LLM
- **Pacotes oficiais confiáveis**: Repositórios oficiais são considerados seguros (PGP assinado)
- **Verificação de fontes**: `verify` valida magic bytes de arquivos `.ttf`, `.otf`, `.woff`, `.woff2`
- **Listagem de fontes**: `list` busca pacotes por nome/descrição

## Comandos

```bash
iai list <termo>        # Busca pacotes no pacman + AUR
iai install <termo>     # Busca, audita e instala com verificação
iai audit <pacote>      # Analisa sem instalar (mostra ameaças + veredito LLM)
iai verify <arquivo>    # Verifica e instala arquivo de fonte local
```

## Instalação

```bash
go build -o iai .
sudo cp iai /usr/local/bin/
```

## Configuração LLM

Crie `~/.config/iai/config.json`:

```json
{
  "url": "http://127.0.0.1:2211/v1",
  "model": "opencode/big-pickle",
  "token": ""
}
```

Compatível com OpenAI Chat Completions API.

## Fluxo de segurança

1. Busca pacotes (oficial + AUR)
2. Se pacote está em repositório oficial → seguro, instala direto
3. Se for AUR → baixa PKGBUILD, analisa com regras determinísticas
4. Se houver ameaças críticas ou score alto → bloqueia
5. Se LLM disponível e aprovar (sem ameaças críticas) → permite instalação
6. Caso contrário → bloqueia com explicação detalhada

## Regras determinísticas

Detecta: `curl|sh`, `wget|sh`, `eval`, `base64 -d`, pastebin/gist, IP direto, `sudo` no build, `/etc/`, `chmod 777`, `rm -rf /`, TLDs suspeitos, etc.

## Desenvolvimento

```bash
go build ./...
go run . --help
```
