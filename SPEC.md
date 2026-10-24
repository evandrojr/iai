# Especificação Técnica - iai

## Visão Geral

Sistema de instalação segura de pacotes Arch Linux com verificação em duas camadas: regras determinísticas + validação por LLM.

## Arquitetura

```
main.go
cmd/           # CLI (cobra)
  ├── root.go
  ├── install.go
  ├── list.go
  ├── audit.go
  └── verify.go
internal/
  ├── search/   # Busca pacman + AUR RPC
  ├── aur/      # Consulta/info/fetch PKGBUILD AUR
  ├── scan/     # Regras determinísticas + prompt LLM
  ├── install/  # Instalação (pacman/paru/yay/makepkg, fontes)
  ├── llm/      # Cliente OpenAI-compatible
  └── notify/   # Notificações desktop
```

## Componentes

### search
- `Official(query)`: `pacman -Ss`, parseia repositório/pacote/versão/descrição/instalado
- `AUR(query)`: `GET https://aur.archlinux.org/rpc/v5/search/%s?by=name-desc`
- `Search(query, limit)`: merge, dedup (prioriza oficial), ordena por repo oficial > popularidade

### aur
- `Check(pkg)`: RPC v5 info, trata `OutOfDate` como bool/number
- `FetchPKGBUILD(pkg)`: baixa de `aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=...`

### scan
**Regras (score):**
- critical (+40): curl|sh, wget|sh, eval com string, base64 -d
- high (+25): pastebin, gist raw, IP direto, sudo build, .service, /etc/
- medium (+10): chmod 777, rm -rf /absoluto
- low (+5): TLD suspeito

**LLM:** Retorna JSON `{verdict, reasons[], upstream_check}` com valores `safe|suspicious|malicious`

### install
- `OfficialRepo(pkg)`: `pacman -Si`, extrai repositório
- `InstallOfficial(pkg)`: `sudo pacman -S --noconfirm`
- `DetectAURHelper()`: busca paru/yay
- `InstallAUR(pkg)`: usa helper ou erro (fallback makepkg)
- `InstallFontFile(path)`: copia para `~/.local/share/fonts/`, `fc-cache -f`

### llm
Cliente HTTP OpenAI-compatible. Suporta stream (não usado). Timeout 300s.

## Lógica de Decisão (install/audit)

```go
// Oficial
if em repo oficial -> Safe=true, Reason="repo oficial (PGP assinado)"

// AUR
score = soma regras
if score >= 40 -> bloqueia
if LLM ok e verdict=="safe" e sem critical -> Safe=true
else if score==0 e sem threats e LLM safe -> Safe=true
else bloqueia com motivos (threats + LLM reasons)
```

## Segurança

- LLM apenas aconselha, decisão é determinística
- Repositórios oficiais sempre aceitos
- AUR sempre auditado
- Críticos (+40) nunca permitidos sem revisão manual? Bloqueia sempre
- `notify-send` alerta em caso de bloqueio
- Sem execução de PKGBUILD local, apenas análise estática

## Dependências

- `github.com/spf13/cobra` v1.10.2
- Go 1.26.5
- Ferramentas externas: `pacman`, `fc-list`, `notify-send`, `paru`/`yay` (opcional)

## Limitações

- Análise estática apenas (não executa PKGBUILD)
- Arquivo `.install` não analisado
- LLM precisa estar disponível para AUR (comportamento atual)
- Sem verificação de checksums PGP do próprio PKGBUILD
